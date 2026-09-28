import sys
import numpy as np
from sklearn.decomposition import PCA
import os

# Append paths
sys.path.append(os.path.abspath("wetware_causal_logic"))
from substrate import BiologicalWetware

def main():
    print("Initializing Phase 4: Topological Data Analysis...")
    
    # 1. Generate Spontaneous Spike Raster
    print("Generating spontaneous biological spike raster...")
    np.random.seed(42)
    bw = BiologicalWetware(num_nodes=1000)
    
    # Run the network for 2000 steps with noise to induce spontaneous avalanches
    _, spike_raster = bw.run_avalanche([], max_steps=2000, noise_std=0.05, apply_stdp=False)
    
    print(f"Spike raster shape (Time, Neurons): {spike_raster.shape}")
    
    # We want a high-dimensional state space. 
    # To get continuous states from binary spikes, we can smooth it or just PCA the binary states directly,
    # or use the voltage trace. The prompt says: "Convert the temporal firing rates into a high-dimensional point cloud"
    # So let's compute firing rates in bins or smooth them.
    
    # Smoothing via moving average / exponential filter
    def smooth(spikes, tau=20):
        # spikes: T x N
        smoothed = np.zeros_like(spikes)
        trace = np.zeros(spikes.shape[1])
        for t in range(spikes.shape[0]):
            trace = trace * np.exp(-1/tau) + spikes[t]
            smoothed[t] = trace
        return smoothed
        
    print("Converting binary spikes to temporal firing rates...")
    firing_rates = smooth(spike_raster, tau=50)
    
    # 2. PCA for Dimensionality Reduction
    print("Applying PCA to reduce to top 3 principal components...")
    pca = PCA(n_components=3)
    point_cloud = pca.fit_transform(firing_rates)
    
    # 3. Save Point Cloud
    os.makedirs("phase4_tda", exist_ok=True)
    np.save("phase4_tda/point_cloud.npy", point_cloud)
    
    # MANDATORY VERIFICATION 1
    print(f"\nVerification: Point cloud shape is {point_cloud.shape}")
    
    # Check if states are distinct
    std_dev = np.std(point_cloud, axis=0)
    print(f"Point cloud standard deviation along components: {std_dev}")
    if np.all(std_dev > 1e-4):
        print("Verification Passed: The point cloud contains dynamic, distinct states (not collapsed).")
    else:
        print("Verification Failed: The point cloud is collapsed into a single static point.")

if __name__ == "__main__":
    main()
