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
    
    # Fixed optimal parameters from previous sweep
    leak = 0.95
    gain = 0.5
    alpha = 1.0
    
    num_nodes_list = [200, 500, 1000]
    conn_radius_list = [0.5, 1.0, 2.0, 5.0]
    
    results = []
    
    best_nrmse = float('inf')
    best_params = None
    
    print("Starting architectural grid search...")
    start_time = time.time()
    
    for n in num_nodes_list:
        for r in conn_radius_list:
            
            # Setup network
            net = BiologicalWetware(num_nodes=n, space_size=10.0, conn_radius=r, leak=leak, threshold=1.0)
            net.normalize_weights(target_sigma=1.0)
            
            num_input = int(n * 0.2)
            input_cluster = np.arange(num_input)
            memory_cluster = np.arange(num_input, n)
            
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
            
            ridge = Ridge(alpha=alpha)
            ridge.fit(X_train, Y_train)
            
            Y_test_pred = ridge.predict(X_test)
            
            var_test = np.var(Y_test)
            nrmse = float('inf')
            if var_test > 0:
                nrmse = np.sqrt(mean_squared_error(Y_test, Y_test_pred) / var_test)
                
            results.append({
                'Reservoir_Size': n,
                'Conn_Radius': r,
                'NRMSE': nrmse
            })
            
            if nrmse < best_nrmse:
                best_nrmse = nrmse
                best_params = {'size': n, 'radius': r}
                
            print(f"Size={n}, Radius={r:.1f}, Test NRMSE={nrmse:.4f}")
            
    print(f"\nTopology sweep completed in {time.time() - start_time:.2f}s")
    
    # Formatted output
    df = pd.DataFrame(results)
    print("\n--- Structural Configurations ---")
    print(df.sort_values('NRMSE').to_string(index=False))
    
    print("\nBest Configuration:")
    print(f"Reservoir Size: {best_params['size']}")
    print(f"Connectivity Radius: {best_params['radius']}")
    print(f"Test NRMSE: {best_nrmse:.4f}")
    
    # Save plot: Reservoir Size vs Test NRMSE for different radii
    plt.figure(figsize=(10, 6))
    for r in conn_radius_list:
        subset = df[df['Conn_Radius'] == r]
        plt.plot(subset['Reservoir_Size'], subset['NRMSE'], marker='o', label=f'Radius: {r}')
        
    plt.title('Reservoir Size vs. Test NRMSE\n(Across Different Connectivity Radii)')
    plt.xlabel('Reservoir Size (N)')
    plt.ylabel('Test NRMSE')
    plt.xticks(num_nodes_list)
    plt.grid(True, linestyle='--', alpha=0.7)
    plt.legend(title='Connectivity Radius (λ)')
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'topology_scaling.png'), dpi=150)
    plt.close()

if __name__ == "__main__":
    main()
