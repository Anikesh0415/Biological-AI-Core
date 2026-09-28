import sys
import os
import numpy as np
import matplotlib.pyplot as plt
import nolds

# Append paths to use the substrate model
sys.path.append(os.path.abspath("wetware_causal_logic"))
from substrate import BiologicalWetware

def main():
    print("Initializing Phase 4: Chaos & Fractal Analysis...")
    
    # 1. Generate Spontaneous Spike Raster
    print("Generating spontaneous biological spike raster...")
    np.random.seed(3)
    bw = BiologicalWetware(num_nodes=1000)
    
    # Run the network for 2000 steps
    _, spike_raster = bw.run_avalanche([], max_steps=2000, noise_std=0.05, apply_stdp=False)
    
    # Smoothing via moving average / exponential filter
    def smooth(spikes, tau=50):
        smoothed = np.zeros_like(spikes)
        trace = np.zeros(spikes.shape[1])
        for t in range(spikes.shape[0]):
            trace = trace * np.exp(-1/tau) + spikes[t]
            smoothed[t] = trace
        return smoothed
        
    print("Converting binary spikes to temporal firing rates...")
    firing_rates = smooth(spike_raster, tau=50)
    
    # Average network firing rate
    mean_firing_rate = np.mean(firing_rates, axis=1)
    
    # To properly analyze fractal noise in a smoothed (integrated) signal, we analyze its fluctuations (differenced series).
    # This prevents the artificial H > 1 inflation caused by the exponential smoothing filter.
    fluctuations = np.diff(mean_firing_rate)
    
    # 2. Detrended Fluctuation Analysis (DFA)
    print("Computing Detrended Fluctuation Analysis (DFA)...")
    import warnings
    warnings.filterwarnings('ignore') # Ignore nolds warnings for small data
    hurst, debug_data = nolds.dfa(fluctuations, debug_data=True)
    n_vals, F_n_vals, p = debug_data[0], debug_data[1], debug_data[2]
    
    # MANDATORY VERIFICATION 1
    print(f"\nHurst Exponent (H): {hurst:.4f}")
    if 0.5 < hurst < 1.0:
        print("Verification Passed: The time-series contains long-range fractal memory (0.5 < H < 1.0).")
    else:
        print("Verification Failed: Hurst exponent is out of expected fractal bounds.")
        
    # 3. Largest Lyapunov Exponent (LLE)
    print("\nComputing Largest Lyapunov Exponent (LLE)...")
    # Using Rosenstein's algorithm provided by nolds
    lle = nolds.lyap_r(fluctuations)
    
    # MANDATORY VERIFICATION 2
    print(f"Largest Lyapunov Exponent (LLE): {lle:.4f}")
    if lle > 0:
        print("Verification Passed: The system is chaotic (highly sensitive to initial conditions) (LLE > 0).")
    else:
        print("Verification Failed: The system is not chaotic (LLE <= 0).")
        
    # 4. DFA Visualization
    print("\nGenerating DFA Fluctuation Plot...")
    plt.figure(figsize=(8, 6))
    
    log_n = np.log10(n_vals)
    log_F = np.log10(F_n_vals)
    
    plt.scatter(log_n, log_F, color='blue', label='F(n)')
    plt.plot(log_n, np.polyval(p, log_n), color='red', label=f'Fit (H = {hurst:.4f})')
    
    plt.xlabel('log(n)')
    plt.ylabel('log(F(n))')
    plt.title('Detrended Fluctuation Analysis')
    plt.legend()
    plt.grid(True)
    
    os.makedirs("phase4_tda", exist_ok=True)
    plot_path = "phase4_tda/dfa_fluctuation_plot.png"
    plt.savefig(plot_path)
    plt.close()
    
    # MANDATORY VERIFICATION 3
    if os.path.exists(plot_path):
        print(f"Verification Passed: DFA Fluctuation plot secured at {plot_path}.")
    else:
        print("Verification Failed: Plot not found.")

if __name__ == "__main__":
    main()
