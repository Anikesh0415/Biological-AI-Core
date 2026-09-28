
import matplotlib.pyplot as plt
import numpy as np
import os
try:
    import powerlaw
except ImportError:
    powerlaw = None

class Visualizer:
    def __init__(self, results_dir="./results"):
        self.results_dir = results_dir
        os.makedirs(self.results_dir, exist_ok=True)
        
    def plot_criticality(self, sizes, durations):
        """Figure 1: Log-log plot of avalanche size and duration distributions."""
        plt.figure(figsize=(12, 5))
        
        # Size distribution
        plt.subplot(1, 2, 1)
        if len(sizes) > 0:
            if powerlaw:
                fit = powerlaw.Fit(sizes, discrete=True, xmin=min(sizes))
                fit.plot_pdf(color='b', linewidth=2, label='Empirical $P(S)$')
                fit.power_law.plot_pdf(color='b', linestyle='--', label=f'Fit ($\\tau \\approx {fit.power_law.alpha:.2f}$)')
            else:
                # Basic numpy histogram
                hist, bins = np.histogram(sizes, bins=np.logspace(np.log10(min(sizes)), np.log10(max(sizes)), 50), density=True)
                bin_centers = (bins[:-1] + bins[1:]) / 2
                plt.scatter(bin_centers, hist, color='b', label='Empirical $P(S)$')
                
                # Fit line manually
                valid = hist > 0
                if np.sum(valid) > 2:
                    log_x = np.log10(bin_centers[valid])
                    log_y = np.log10(hist[valid])
                    m, c = np.polyfit(log_x, log_y, 1)
                    plt.plot(bin_centers, 10**(m * np.log10(bin_centers) + c), 'b--', label=f'Fit ($\\tau \\approx {-m:.2f}$)')
            
            # Target line tau = 1.67
            x_vals = np.logspace(np.log10(min(sizes)), np.log10(max(sizes)), 10)
            y_vals = x_vals**(-1.67)
            # scale
            if len(sizes) > 0:
                 scale_idx = len(x_vals)//4
                 y_vals = y_vals * (np.mean(sizes) / y_vals[scale_idx]) # Rough scaling for visual
                 
            plt.plot(x_vals, y_vals, 'k:', label='Target ($\\tau = 1.67$)')
            
        plt.xscale('log')
        plt.yscale('log')
        plt.xlabel('Avalanche Size $S$')
        plt.ylabel('$P(S)$')
        plt.title('Avalanche Size Distribution')
        plt.legend()
        
        # Duration distribution
        plt.subplot(1, 2, 2)
        if len(durations) > 0:
            if powerlaw:
                fit_t = powerlaw.Fit(durations, discrete=True, xmin=min(durations))
                fit_t.plot_pdf(color='r', linewidth=2, label='Empirical $P(T)$')
                fit_t.power_law.plot_pdf(color='r', linestyle='--', label=f'Fit ($\\alpha \\approx {fit_t.power_law.alpha:.2f}$)')
            else:
                hist, bins = np.histogram(durations, bins=np.logspace(np.log10(max(1, min(durations))), np.log10(max(durations)), 50), density=True)
                bin_centers = (bins[:-1] + bins[1:]) / 2
                plt.scatter(bin_centers, hist, color='r', label='Empirical $P(T)$')
                
        plt.xscale('log')
        plt.yscale('log')
        plt.xlabel('Avalanche Duration $T$')
        plt.ylabel('$P(T)$')
        plt.title('Avalanche Duration Distribution')
        plt.legend()
        
        plt.tight_layout()
        plt.savefig(os.path.join(self.results_dir, "figure1_criticality.png"), dpi=300)
        plt.close()
        print("Saved Figure 1: Criticality Validation")

    def plot_logic_performance(self, and_results, or_results):
        """Figure 2: Bit Error Rate curves vs Noise Intensity."""
        noises = sorted(list(and_results.keys()))
        
        and_ber = [and_results[n]['BER'] for n in noises]
        or_ber = [or_results[n]['BER'] for n in noises]
        
        plt.figure(figsize=(8, 6))
        plt.plot(noises, and_ber, 'bo-', linewidth=2, label='AND Gate BER')
        plt.plot(noises, or_ber, 'ro-', linewidth=2, label='OR Gate BER')
        
        plt.axhline(0.1, color='k', linestyle='--', alpha=0.5, label='10% Error Threshold')
        
        plt.xlabel(r'Biological Noise Level ($\sigma$)')
        plt.ylabel('Bit Error Rate (BER)')
        plt.title('Noise-Tolerance of Biological Boolean Gates')
        plt.grid(True, alpha=0.3)
        plt.legend()
        
        plt.tight_layout()
        plt.savefig(os.path.join(self.results_dir, "figure2_logic_performance.png"), dpi=300)
        plt.close()
        print("Saved Figure 2: Logic Gate Performance")

    def plot_spatiotemporal_dynamics(self, history, positions):
        """Figure 3: Spatiotemporal raster / spatial plot of a wave."""
        # This plots a simple raster and a 3D scatter of the final propagation
        plt.figure(figsize=(12, 5))
        
        # Raster Plot
        plt.subplot(1, 2, 1)
        active_times = []
        active_neurons = []
        for t, spikes in enumerate(history):
            fired = np.where(spikes > 0)[0]
            active_times.extend([t] * len(fired))
            active_neurons.extend(fired)
            
        plt.scatter(active_times, active_neurons, s=1, c='k', alpha=0.5)
        plt.xlabel('Time Step')
        plt.ylabel('Neuron Index')
        plt.title('Raster Plot: Constructive Interference')
        
        # 3D Spatial Plot of total activity
        ax = plt.subplot(1, 2, 2, projection='3d')
        total_spikes = np.sum(history, axis=0)
        
        # Plot all neurons faint
        ax.scatter(positions[:, 0], positions[:, 1], positions[:, 2], 
                   c='gray', s=10, alpha=0.1)
                   
        # Plot active neurons brightly
        active_idx = np.where(total_spikes > 0)[0]
        if len(active_idx) > 0:
            scatter = ax.scatter(positions[active_idx, 0], 
                                 positions[active_idx, 1], 
                                 positions[active_idx, 2], 
                                 c=total_spikes[active_idx], 
                                 cmap='hot', s=20, alpha=0.8)
            plt.colorbar(scatter, label='Spike Count', shrink=0.5)
            
        # Highlight regions approximately
        ax.scatter([2.0], [3.0], [5.0], c='blue', s=100, marker='x', label='Input A')
        ax.scatter([8.0], [3.0], [5.0], c='cyan', s=100, marker='x', label='Input B')
        ax.scatter([5.0], [8.0], [5.0], c='green', s=100, marker='*', label='Output Readout')
        
        ax.set_title('Spatial Wave Propagation (AND Gate)')
        ax.legend()
        
        plt.tight_layout()
        plt.savefig(os.path.join(self.results_dir, "figure3_spatiotemporal.png"), dpi=300)
        plt.close()
        print("Saved Figure 3: Spatiotemporal Dynamics")

    def plot_training_comparison(self, and_pre, and_post, or_pre, or_post):
        """Figure 4: Pre vs Post Training BER curves."""
        noises = sorted(list(and_pre.keys()))
        
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(12, 5))
        
        # AND Gate
        ax1.plot(noises, [and_pre[n]['BER'] for n in noises], 'k--', label='Pre-Train BER')
        ax1.plot(noises, [and_post[n]['BER'] for n in noises], 'b-', linewidth=2, label='Post-Train BER')
        ax1.set_title('AND Gate: STDP Learning Improvement')
        ax1.set_xlabel(r'Biological Noise Level ($\sigma$)')
        ax1.set_ylabel('Bit Error Rate (BER)')
        ax1.legend()
        ax1.grid(True, alpha=0.3)
        
        # OR Gate
        ax2.plot(noises, [or_pre[n]['BER'] for n in noises], 'k--', label='Pre-Train BER')
        ax2.plot(noises, [or_post[n]['BER'] for n in noises], 'r-', linewidth=2, label='Post-Train BER')
        ax2.set_title('OR Gate: STDP Learning Improvement')
        ax2.set_xlabel(r'Biological Noise Level ($\sigma$)')
        ax2.legend()
        ax2.grid(True, alpha=0.3)
        
        plt.tight_layout()
        plt.savefig(os.path.join(self.results_dir, "figure4_training_comparison.png"), dpi=300)
        plt.close()
        print("Saved Figure 4: Training Comparison")

    def plot_weight_matrices(self, W_pre, W_post, gate_name):
        """Figure 5: Synaptic weight matrix heatmaps before and after STDP."""
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(10, 4))
        
        # We sample a small submatrix for visual clarity (e.g., first 100x100 nodes)
        sample_size = min(100, W_pre.shape[0])
        
        im1 = ax1.imshow(W_pre[:sample_size, :sample_size], cmap='viridis', aspect='auto')
        ax1.set_title(f'Pre-Train Weights ({gate_name})')
        plt.colorbar(im1, ax=ax1)
        
        im2 = ax2.imshow(W_post[:sample_size, :sample_size], cmap='viridis', aspect='auto')
        ax2.set_title(f'Post-Train Weights ({gate_name})')
        plt.colorbar(im2, ax=ax2)
        
        plt.tight_layout()
        plt.savefig(os.path.join(self.results_dir, f"figure5_{gate_name}_weights.png"), dpi=300)
        plt.close()
        print(f"Saved Figure 5: Weight Matrices ({gate_name})")
