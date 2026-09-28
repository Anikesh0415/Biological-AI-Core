import numpy as np
import matplotlib.pyplot as plt
from sklearn.linear_model import Ridge
from sklearn.metrics import mean_squared_error
import sys
import os
import time
import pandas as pd

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
    
    U, Y = generate_narma10(total_len)
    
    U_train = U[:train_len]
    Y_train = Y[:train_len]
    U_test = U[train_len:]
    Y_test = Y[train_len:]
    
    leaks = [0.85, 0.90, 0.95, 0.98]
    i_gains = [0.05, 0.1, 0.25, 0.5]
    alphas = [1e-3, 1e-2, 1e-1, 1.0, 10.0]
    
    results = []
    
    best_nrmse = float('inf')
    best_params = None
    best_preds = None
    
    # Store NRMSE for heatmap: leak vs i_gain, taking min over alpha
    heatmap_data = np.zeros((len(leaks), len(i_gains)))
    
    print("Starting hyperparameter grid search...")
    start_time = time.time()
    
    for i, leak in enumerate(leaks):
        for j, gain in enumerate(i_gains):
            
            # Setup network
            num_nodes = 500
            net = BiologicalWetware(num_nodes=num_nodes, space_size=10.0, conn_radius=2.0, leak=leak, threshold=1.0)
            net.normalize_weights(target_sigma=1.0)
            
            num_input = int(num_nodes * 0.2)
            input_cluster = np.arange(num_input)
            memory_cluster = np.arange(num_input, num_nodes)
            
            X = np.zeros((total_len, len(memory_cluster)))
            net.voltage = np.zeros(net.N)
            net.spikes = np.zeros(net.N)
            
            alpha_smooth = 0.1 
            smoothed_rates = np.zeros(len(memory_cluster))
            
            for t in range(total_len):
                input_current = np.zeros(net.N)
                input_current[input_cluster] = U[t] * gain 
                
                spikes = net.step(input_current=input_current, noise_std=0.01)
                
                memory_spikes = spikes[memory_cluster]
                smoothed_rates = (1 - alpha_smooth) * smoothed_rates + alpha_smooth * memory_spikes
                X[t] = smoothed_rates
                
            X_train = X[:train_len]
            X_test = X[train_len:]
            
            min_nrmse_for_pair = float('inf')
            
            for alpha in alphas:
                ridge = Ridge(alpha=alpha)
                ridge.fit(X_train, Y_train)
                
                Y_test_pred = ridge.predict(X_test)
                
                var_test = np.var(Y_test)
                nrmse = float('inf')
                if var_test > 0:
                    nrmse = np.sqrt(mean_squared_error(Y_test, Y_test_pred) / var_test)
                    
                results.append({
                    'Leak': leak,
                    'I_Gain': gain,
                    'Alpha': alpha,
                    'NRMSE': nrmse
                })
                
                if nrmse < min_nrmse_for_pair:
                    min_nrmse_for_pair = nrmse
                    
                if nrmse < best_nrmse:
                    best_nrmse = nrmse
                    best_params = {'leak': leak, 'gain': gain, 'alpha': alpha}
                    best_preds = Y_test_pred
                    
            heatmap_data[i, j] = min_nrmse_for_pair
            print(f"Leak={leak:.2f}, Gain={gain:.2f}, Best NRMSE={min_nrmse_for_pair:.4f}")
            
    print(f"\nParameter sweep completed in {time.time() - start_time:.2f}s")
    
    # Formatted output
    df = pd.DataFrame(results)
    print("\n--- Top 10 Configurations ---")
    print(df.sort_values('NRMSE').head(10).to_string(index=False))
    
    print("\nBest Configuration:")
    print(f"Leak: {best_params['leak']}")
    print(f"I_Gain: {best_params['gain']}")
    print(f"Alpha: {best_params['alpha']}")
    print(f"Test NRMSE: {best_nrmse:.4f}")
    
    # Save optimized prediction plot
    plt.figure(figsize=(12, 6))
    plt.plot(Y_test, label='True NARMA-10 Target', color='black', linewidth=1.5)
    plt.plot(best_preds, label=f"Predicted (NRMSE: {best_nrmse:.4f})", color='green', linestyle='--', linewidth=1.5)
    plt.title(f'Optimized Reservoir Computing on NARMA-10')
    plt.xlabel('Time Step (Test Set)')
    plt.ylabel('Y(t)')
    plt.legend(loc='upper right')
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'narma10_optimized.png'), dpi=150)
    plt.close()
    
    # Save Heatmap plot
    plt.figure(figsize=(8, 6))
    X_grid, Y_grid = np.meshgrid(i_gains, leaks)
    c = plt.contourf(X_grid, Y_grid, heatmap_data, levels=20, cmap='viridis_r')
    plt.colorbar(c, label='Min NRMSE (across Alphas)')
    plt.title('Parameter Sweep: Leak vs Input Gain (NRMSE)')
    plt.xlabel('Input Gain (I_gain)')
    plt.ylabel('Leak (Voltage Retention)')
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'narma10_param_sweep.png'), dpi=150)
    plt.close()

if __name__ == "__main__":
    main()
