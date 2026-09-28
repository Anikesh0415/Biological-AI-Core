import numpy as np
import matplotlib.pyplot as plt
from sklearn.linear_model import Ridge
from sklearn.metrics import mean_squared_error
import sys
import os

# Ensure substrate.py can be imported
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from substrate import BiologicalWetware

def generate_narma10(length=2000):
    """Generates synthetic NARMA-10 time series."""
    # U is drawn from a uniform distribution [0, 0.5]
    U = np.random.uniform(0, 0.5, length)
    Y = np.zeros(length)
    
    for t in range(9, length - 1):
        Y[t+1] = 0.3 * Y[t] + 0.05 * Y[t] * np.sum(Y[t-9:t+1]) + 1.5 * U[t-9] * U[t] + 0.1
        
    return U, Y

def main():
    np.random.seed(42)
    
    # 1. Generate NARMA-10 Data
    total_len = 2000
    train_len = 1500
    test_len = total_len - train_len
    
    print(f"Generating NARMA-10 task with {total_len} steps...")
    U, Y = generate_narma10(total_len)
    
    U_train = U[:train_len]
    Y_train = Y[:train_len]
    U_test = U[train_len:]
    Y_test = Y[train_len:]
    
    # 2. Reservoir Integration
    print("Instantiating 3D LIF Biological Wetware...")
    num_nodes = 500
    # Optimized parameters from parameter sweep
    net = BiologicalWetware(num_nodes=num_nodes, space_size=10.0, conn_radius=2.0, leak=0.95, threshold=1.0)
    net.normalize_weights(target_sigma=1.0) # Enforce homeostatic criticality
    
    # Define Clusters: 20% Input, 80% Recurrent Memory
    num_input = int(num_nodes * 0.2)
    input_cluster = np.arange(num_input)
    memory_cluster = np.arange(num_input, num_nodes)
    
    print("Running Reservoir Integration (this may take a moment)...")
    X = np.zeros((total_len, len(memory_cluster)))
    
    # Reset network state
    net.voltage = np.zeros(net.N)
    net.spikes = np.zeros(net.N)
    
    # Record smoothed firing rate via exponential moving average (low-pass filter)
    alpha = 0.1 
    smoothed_rates = np.zeros(len(memory_cluster))
    
    for t in range(total_len):
        input_current = np.zeros(net.N)
        # Direct membrane current injection scaled from U(t)
        input_current[input_cluster] = U[t] * 0.5 
        
        # Step the network
        spikes = net.step(input_current=input_current, noise_std=0.01)
        
        # Readout from the Recurrent Memory Cluster
        memory_spikes = spikes[memory_cluster]
        
        # Exponential smoothing of firing rates
        smoothed_rates = (1 - alpha) * smoothed_rates + alpha * memory_spikes
        
        # Alternatively, could use raw voltage: 
        # X[t] = net.voltage[memory_cluster]
        X[t] = smoothed_rates
        
        if (t + 1) % 500 == 0:
            print(f"  Step {t+1}/{total_len} complete.")
            
    # Train/Test Split for Reservoir States
    X_train = X[:train_len]
    X_test = X[train_len:]
    
    # 3. Linear Readout & Evaluation
    print("Training Linear Readout...")
    ridge = Ridge(alpha=1.0)
    ridge.fit(X_train, Y_train)
    
    Y_train_pred = ridge.predict(X_train)
    Y_test_pred = ridge.predict(X_test)
    
    # NRMSE Calculation
    var_test = np.var(Y_test)
    if var_test > 0:
        nrmse = np.sqrt(mean_squared_error(Y_test, Y_test_pred) / var_test)
    else:
        nrmse = float('inf')
        
    print(f"Test NRMSE: {nrmse:.4f}")
    
    # 4. Visualization
    print("Generating visualization...")
    plt.figure(figsize=(12, 6))
    plt.plot(Y_test, label='True NARMA-10 Target', color='black', linewidth=1.5)
    plt.plot(Y_test_pred, label='Predicted Output', color='red', linestyle='--', linewidth=1.5)
    plt.title(f'Reservoir Computing on NARMA-10 Task\n(Test NRMSE: {nrmse:.4f})')
    plt.xlabel('Time Step (Test Set)')
    plt.ylabel('Target Value Y(t)')
    plt.legend(loc='upper right')
    plt.tight_layout()
    
    plot_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'narma10_results.png')
    plt.savefig(plot_path, dpi=150)
    print(f"Saved plot to {plot_path}")
    
if __name__ == "__main__":
    main()
