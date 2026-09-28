import numpy as np
from ripser import ripser
from persim import plot_diagrams
import matplotlib.pyplot as plt
import os

def main():
    print("Loading point cloud...")
    point_cloud = np.load("phase4_tda/point_cloud.npy")
    print(f"Point cloud shape: {point_cloud.shape}")
    
    # 2. Compute Vietoris-Rips Filtration
    print("Computing Vietoris-Rips filtration and Betti numbers (H0, H1)...")
    # Subsample if necessary to speed up, but 5001 points is okay for ripser in 3D usually, 
    # although subsampling to 1000 might be safer to prevent memory explosion.
    if point_cloud.shape[0] > 1000:
        idx = np.random.choice(point_cloud.shape[0], 1000, replace=False)
        pc_sample = point_cloud[idx]
    else:
        pc_sample = point_cloud
        
    result = ripser(pc_sample, maxdim=1)
    diagrams = result['dgms']
    
    H0 = diagrams[0]
    H1 = diagrams[1]
    
    print(f"Number of H0 features: {len(H0)}")
    print(f"Number of H1 features: {len(H1)}")
    
    # Analyze H1
    if len(H1) > 0:
        lifetimes = H1[:, 1] - H1[:, 0]
        # Ignore infinity if any
        lifetimes = lifetimes[np.isfinite(lifetimes)]
        if len(lifetimes) > 0:
            max_lifetime = np.max(lifetimes)
            print(f"Maximum persistence lifetime of H1 cycle: {max_lifetime:.4f}")
            if max_lifetime > 0.5: # arbitrary threshold for "prominent"
                print("Analysis: The network contains persistent topological memory loops.")
            else:
                print("Analysis: The features are short-lived noise.")
        else:
            print("Maximum persistence lifetime of H1 cycle: 0")
            print("Analysis: No finite H1 loops found, short-lived noise.")
    else:
        print("Maximum persistence lifetime of H1 cycle: 0")
        print("Analysis: No H1 loops found, just short-lived noise.")
        
    # 3. Persistence Diagram Visualization
    print("Generating persistence diagram...")
    plt.figure(figsize=(8, 6))
    plot_diagrams(diagrams, show=False)
    plt.title("Persistence Diagram (H0 and H1)")
    
    plot_path = "phase4_tda/persistence_diagram.png"
    plt.savefig(plot_path)
    plt.close()
    
    if os.path.exists(plot_path):
        print("Verification Passed: Persistence diagram secured.")
    else:
        print("Verification Failed: Diagram not found.")

if __name__ == "__main__":
    main()
