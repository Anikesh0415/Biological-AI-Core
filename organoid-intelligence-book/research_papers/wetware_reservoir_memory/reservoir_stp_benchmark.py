import numpy as np
import matplotlib.pyplot as plt
from sklearn.linear_model import Ridge
from sklearn.metrics import mean_squared_error
import sys
import os

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from substrate import BiologicalWetware

class BiologicalWetwareSTP(BiologicalWetware):
    def __init__(self, *args, U=0.2, tau_f=200.0, tau_d=50.0, **kwargs):
        self.U = U
        self.tau_f = tau_f
        self.tau_d = tau_d
        super().__init__(*args, **kwargs)
        
        # Initialize Tsodyks-Markram STP states for each presynaptic neuron
        self.u = np.full(self.N, self.U)
        self.x = np.ones(self.N)
        
    def normalize_weights(self, target_sigma=1.0):
        super().normalize_weights(target_sigma=target_sigma)
        # Steady-state synaptic compensation (u ~ U, x ~ 1.0)
        self.weights = self.weights / self.U
        
    def step(self, input_current=None, noise_std=0.0, apply_stdp=False):
        if input_current is None:
            input_current = np.zeros(self.N)
            
        noise = np.random.normal(0, noise_std, self.N)
        
        # Exponential decay of STP variables (dt=1 step)
        self.u = self.U + (self.u - self.U) * np.exp(-1.0 / self.tau_f)
        self.x = 1.0 + (self.x - 1.0) * np.exp(-1.0 / self.tau_d)
        
        # Calculate facilitation upon spike (u^+)
        u_plus = self.u + self.U * (1.0 - self.u) * self.spikes
        
        # Modulate outgoing spikes based on u^+ and x^-
        effective_spikes = self.spikes * u_plus * self.x
        synaptic_input = self.weights.T @ effective_spikes
        
        # Depress resources (update x and u for next step)
        self.x = self.x - u_plus * self.x * self.spikes
        self.u = u_plus
        
        self.voltage = self.leak * self.voltage + synaptic_input + input_current + noise
        self.spikes = (self.voltage >= self.threshold).astype(float)
        self.voltage[self.spikes > 0] = 0.0
        
        return self.spikes

def generate_narma10(length=2000):
    U = np.random.uniform(0, 0.5, length)
    Y = np.zeros(length)
    for t in range(9, length - 1):
        Y[t+1] = 0.3 * Y[t] + 0.05 * Y[t] * np.sum(Y[t-9:t+1]) + 1.5 * U[t-9] * U[t] + 0.1
    return U, Y

def run_reservoir(network_class, U_input, train_len, test_len, use_stp=False):
    total_len = train_len + test_len
    
    num_nodes = 500
    conn_radius = 1.0
    leak = 0.95
    gain = 0.5
    
    if use_stp:
        net = network_class(num_nodes=num_nodes, space_size=10.0, conn_radius=conn_radius, leak=leak, threshold=1.0, U=0.2, tau_f=200.0, tau_d=50.0)
    else:
        net = network_class(num_nodes=num_nodes, space_size=10.0, conn_radius=conn_radius, leak=leak, threshold=1.0)
        
    net.normalize_weights(target_sigma=1.0)
    
    num_input = int(num_nodes * 0.2)
    input_cluster = np.arange(num_input)
    memory_cluster = np.arange(num_input, num_nodes)
    
    taus = [5, 15, 50, 100]
    alphas = [1.0 / tau for tau in taus]
    
    # Determine feature dimensions based on STP
    num_features = len(memory_cluster) * len(taus)
    if use_stp:
        num_features += len(memory_cluster) * 2  # appending u(t) and x(t)
        
    X = np.zeros((total_len, num_features))
    
    net.voltage = np.zeros(net.N)
    net.spikes = np.zeros(net.N)
    filtered_states = [np.zeros(len(memory_cluster)) for _ in taus]
    
    for t in range(total_len):
        input_current = np.zeros(net.N)
        input_current[input_cluster] = U_input[t] * gain
        
        spikes = net.step(input_current=input_current, noise_std=0.01)
        mem_spikes = spikes[memory_cluster]
        
        step_features = []
        for i, a in enumerate(alphas):
            filtered_states[i] = (1 - a) * filtered_states[i] + a * mem_spikes
            step_features.append(filtered_states[i].copy())
            
        if use_stp:
            # Append dynamic synaptic states u(t) and x(t) for memory cluster
            step_features.append(net.u[memory_cluster].copy())
            step_features.append(net.x[memory_cluster].copy())
            
        X[t] = np.concatenate(step_features)
        
    return X[:train_len], X[train_len:]

def main():
    np.random.seed(42)
    
    total_len = 2000
    train_len = 1500
    test_len = total_len - train_len
    
    print("Generating Benchmark Data...")
    U_narma, Y_narma = generate_narma10(total_len)
    U_mc = np.random.uniform(-1, 1, total_len)
    
    # -------------------------------------------------------------
    # RUN BASELINE
    # -------------------------------------------------------------
    print("Running Baseline Reservoir (No STP)...")
    X_train_base, X_test_base = run_reservoir(BiologicalWetware, U_narma, train_len, test_len, use_stp=False)
    X_train_mc_base, X_test_mc_base = run_reservoir(BiologicalWetware, U_mc, train_len, test_len, use_stp=False)
    
    # Baseline NARMA-10
    ridge = Ridge(alpha=1.0)
    ridge.fit(X_train_base, Y_narma[:train_len])
    Y_pred_base = ridge.predict(X_test_base)
    nrmse_base = np.sqrt(mean_squared_error(Y_narma[train_len:], Y_pred_base) / np.var(Y_narma[train_len:]))
    
    # Baseline MC
    r2_base = []
    for k in range(1, 31):
        Y_k = np.zeros(total_len)
        Y_k[k:] = U_mc[:-k]
        ridge.fit(X_train_mc_base, Y_k[:train_len])
        Y_pred_mc = ridge.predict(X_test_mc_base)
        
        Y_test_k = Y_k[train_len:]
        if np.std(Y_test_k) > 0 and np.std(Y_pred_mc) > 0:
            r2 = np.corrcoef(Y_test_k, Y_pred_mc)[0, 1]**2
        else:
            r2 = 0.0
        r2_base.append(r2)
    mc_base = sum(r2_base)
    
    # -------------------------------------------------------------
    # RUN STP RESERVOIR
    # -------------------------------------------------------------
    print("Running STP Reservoir (Tsodyks-Markram)...")
    X_train_stp, X_test_stp = run_reservoir(BiologicalWetwareSTP, U_narma, train_len, test_len, use_stp=True)
    X_train_mc_stp, X_test_mc_stp = run_reservoir(BiologicalWetwareSTP, U_mc, train_len, test_len, use_stp=True)
    
    # STP NARMA-10
    ridge.fit(X_train_stp, Y_narma[:train_len])
    Y_pred_stp = ridge.predict(X_test_stp)
    nrmse_stp = np.sqrt(mean_squared_error(Y_narma[train_len:], Y_pred_stp) / np.var(Y_narma[train_len:]))
    
    # STP MC
    r2_stp = []
    for k in range(1, 31):
        Y_k = np.zeros(total_len)
        Y_k[k:] = U_mc[:-k]
        ridge.fit(X_train_mc_stp, Y_k[:train_len])
        Y_pred_mc = ridge.predict(X_test_mc_stp)
        
        Y_test_k = Y_k[train_len:]
        if np.std(Y_test_k) > 0 and np.std(Y_pred_mc) > 0:
            r2 = np.corrcoef(Y_test_k, Y_pred_mc)[0, 1]**2
        else:
            r2 = 0.0
        r2_stp.append(r2)
    mc_stp = sum(r2_stp)
    
    # -------------------------------------------------------------
    # OUTPUT AND VISUALIZATION
    # -------------------------------------------------------------
    print("\n" + "="*40)
    print("BENCHMARK COMPARISON SUMMARY")
    print("="*40)
    print(f"NARMA-10 Test NRMSE:  Baseline = {nrmse_base:.4f}  |  STP = {nrmse_stp:.4f}")
    print(f"Total Linear MC:      Baseline = {mc_base:.4f}  |  STP = {mc_stp:.4f}")
    print("="*40 + "\n")
    
    Y_test_narma = Y_narma[train_len:]
    
    # Plot 1: NARMA-10 Comparison
    plt.figure(figsize=(14, 6))
    plt.plot(Y_test_narma, label='True NARMA-10 Target', color='black', linewidth=1.5)
    plt.plot(Y_pred_base, label=f'Baseline (NRMSE: {nrmse_base:.4f})', color='gray', linestyle='--')
    plt.plot(Y_pred_stp, label=f'STP Enabled (NRMSE: {nrmse_stp:.4f})', color='green', linestyle='--')
    plt.title('NARMA-10 Trajectories: Baseline vs. TM Short-Term Plasticity')
    plt.xlabel('Time Step (Test Set)')
    plt.ylabel('Y(t)')
    plt.legend(loc='upper right')
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'stp_narma10_comparison.png'), dpi=150)
    plt.close()
    
    # Plot 2: MC Comparison
    delays = np.arange(1, 31)
    plt.figure(figsize=(10, 6))
    plt.plot(delays, r2_base, marker='o', color='gray', linewidth=2, label=f'Baseline (MC: {mc_base:.2f})')
    plt.plot(delays, r2_stp, marker='s', color='green', linewidth=2, label=f'STP Enabled (MC: {mc_stp:.2f})')
    plt.fill_between(delays, r2_base, color='gray', alpha=0.1)
    plt.fill_between(delays, r2_stp, color='green', alpha=0.1)
    plt.title('Linear Memory Capacity Profile: Baseline vs. TM STP')
    plt.xlabel('Delay Steps (k)')
    plt.ylabel('Determination Coefficient ($r^2$)')
    plt.ylim(0, 1.05)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.legend()
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'stp_memory_capacity.png'), dpi=150)
    plt.close()

if __name__ == "__main__":
    main()
