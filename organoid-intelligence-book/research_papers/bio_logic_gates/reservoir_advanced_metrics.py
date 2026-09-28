import numpy as np
import matplotlib.pyplot as plt
from sklearn.linear_model import Ridge
from sklearn.metrics import mean_squared_error
import sys
import os
import time

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from substrate import BiologicalWetware

def generate_narma10(length=2000):
    U = np.random.uniform(0, 0.5, length)
    Y = np.zeros(length)
    for t in range(9, length - 1):
        Y[t+1] = 0.3 * Y[t] + 0.05 * Y[t] * np.sum(Y[t-9:t+1]) + 1.5 * U[t-9] * U[t] + 0.1
    return U, Y

def main():
    np.random.seed(42)
    
    total_len = 2000
    train_len = 1500
    test_len = total_len - train_len
    
    # -------------------------------------------------------------
    # 1. Multi-Timescale Synaptic State Filtering for NARMA-10
    # -------------------------------------------------------------
    print("--- 1. Multi-Timescale NARMA-10 Benchmark ---")
    U_narma, Y_narma = generate_narma10(total_len)
    
    # Setup optimal network
    num_nodes = 500
    conn_radius = 1.0
    leak = 0.95
    gain = 0.5
    alpha_ridge = 1.0
    
    net_narma = BiologicalWetware(num_nodes=num_nodes, space_size=10.0, conn_radius=conn_radius, leak=leak, threshold=1.0)
    net_narma.normalize_weights(target_sigma=1.0)
    
    num_input = int(num_nodes * 0.2)
    input_cluster = np.arange(num_input)
    memory_cluster = np.arange(num_input, num_nodes)
    
    taus = [5, 15, 50, 100]
    alphas = [1.0 / tau for tau in taus]
    
    X_narma = np.zeros((total_len, len(memory_cluster) * len(taus)))
    
    net_narma.voltage = np.zeros(net_narma.N)
    net_narma.spikes = np.zeros(net_narma.N)
    
    filtered_states = [np.zeros(len(memory_cluster)) for _ in taus]
    
    for t in range(total_len):
        input_current = np.zeros(net_narma.N)
        input_current[input_cluster] = U_narma[t] * gain
        
        spikes = net_narma.step(input_current=input_current, noise_std=0.01)
        mem_spikes = spikes[memory_cluster]
        
        step_features = []
        for i, a in enumerate(alphas):
            filtered_states[i] = (1 - a) * filtered_states[i] + a * mem_spikes
            step_features.append(filtered_states[i].copy())
            
        X_narma[t] = np.concatenate(step_features)
        
    X_train_narma = X_narma[:train_len]
    X_test_narma = X_narma[train_len:]
    Y_train_narma = Y_narma[:train_len]
    Y_test_narma = Y_narma[train_len:]
    
    ridge_narma = Ridge(alpha=alpha_ridge)
    ridge_narma.fit(X_train_narma, Y_train_narma)
    
    Y_test_pred_narma = ridge_narma.predict(X_test_narma)
    var_test_narma = np.var(Y_test_narma)
    nrmse_narma = np.sqrt(mean_squared_error(Y_test_narma, Y_test_pred_narma) / var_test_narma)
    
    print(f"NARMA-10 Test NRMSE (Multi-Timescale): {nrmse_narma:.4f}")
    
    plt.figure(figsize=(12, 6))
    plt.plot(Y_test_narma, label='True NARMA-10', color='black')
    plt.plot(Y_test_pred_narma, label=f'Predicted (NRMSE: {nrmse_narma:.4f})', color='purple', linestyle='--')
    plt.title('Multi-Timescale Reservoir Computing on NARMA-10')
    plt.xlabel('Time Step (Test Set)')
    plt.ylabel('Y(t)')
    plt.legend()
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'narma10_multiscale_results.png'), dpi=150)
    plt.close()

    # -------------------------------------------------------------
    # 2. Linear Memory Capacity (MC) Benchmark
    # -------------------------------------------------------------
    print("\n--- 2. Linear Memory Capacity (MC) Benchmark ---")
    U_mc = np.random.uniform(-1, 1, total_len)
    
    net_mc = BiologicalWetware(num_nodes=num_nodes, space_size=10.0, conn_radius=conn_radius, leak=leak, threshold=1.0)
    net_mc.normalize_weights(target_sigma=1.0)
    
    X_mc = np.zeros((total_len, len(memory_cluster) * len(taus)))
    
    net_mc.voltage = np.zeros(net_mc.N)
    net_mc.spikes = np.zeros(net_mc.N)
    filtered_states_mc = [np.zeros(len(memory_cluster)) for _ in taus]
    
    for t in range(total_len):
        input_current = np.zeros(net_mc.N)
        input_current[input_cluster] = U_mc[t] * gain
        
        spikes = net_mc.step(input_current=input_current, noise_std=0.01)
        mem_spikes = spikes[memory_cluster]
        
        step_features = []
        for i, a in enumerate(alphas):
            filtered_states_mc[i] = (1 - a) * filtered_states_mc[i] + a * mem_spikes
            step_features.append(filtered_states_mc[i].copy())
            
        X_mc[t] = np.concatenate(step_features)
        
    X_train_mc = X_mc[:train_len]
    X_test_mc = X_mc[train_len:]
    
    delays = np.arange(1, 31)
    r2_scores = []
    
    for k in delays:
        # Shift target by k
        Y_mc_k = np.zeros(total_len)
        Y_mc_k[k:] = U_mc[:-k]
        
        # Optional: in proper MC tasks, we don't score the first few elements which are padded zeros
        # but Ridge takes the entire train split. We'll stick to full vector evaluation.
        Y_train_mc = Y_mc_k[:train_len]
        Y_test_mc = Y_mc_k[train_len:]
        
        ridge_mc = Ridge(alpha=alpha_ridge)
        ridge_mc.fit(X_train_mc, Y_train_mc)
        
        Y_test_pred_mc = ridge_mc.predict(X_test_mc)
        
        # Pearson correlation squared is standard for MC
        if np.std(Y_test_mc) > 0 and np.std(Y_test_pred_mc) > 0:
            r = np.corrcoef(Y_test_mc, Y_test_pred_mc)[0, 1]
            r2 = r**2
        else:
            r2 = 0.0
            
        r2_scores.append(r2)
        
    total_mc = np.sum(r2_scores)
    print(f"Total Memory Capacity (MC): {total_mc:.4f}")
    
    plt.figure(figsize=(10, 6))
    plt.plot(delays, r2_scores, marker='o', color='blue', linewidth=2)
    plt.fill_between(delays, r2_scores, color='blue', alpha=0.2)
    plt.title(f'Linear Memory Capacity Curve (Total MC: {total_mc:.2f})')
    plt.xlabel('Delay Steps (k)')
    plt.ylabel('Determination Coefficient ($r^2$)')
    plt.ylim(0, 1.05)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'memory_capacity_curve.png'), dpi=150)
    plt.close()

if __name__ == "__main__":
    main()
