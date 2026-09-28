import numpy as np
import os
import matplotlib.pyplot as plt
from substrate import BiologicalWetware

class BiologicalSRLatch:
    def __init__(self, num_nodes=1500, conn_radius=1.5, leak=0.01, threshold=0.8):
        self.net = BiologicalWetware(num_nodes=num_nodes, conn_radius=conn_radius, leak=leak, threshold=threshold)
        
        # Spatial regions
        self.set_idx = self._get_nodes_near([3.0, 5.0, 5.0], radius=1.0)
        self.reset_idx = self._get_nodes_near([7.0, 5.0, 5.0], radius=1.0)
        self.inhibitory_idx = self._get_nodes_near([6.0, 5.0, 5.0], radius=1.0)
        self.memory_loop_idx = self._get_nodes_near([5.0, 5.0, 5.0], radius=1.5)
        
        # Ensure memory loop doesn't overlap with inhib/set/reset for clean logic boundaries
        self.memory_loop_idx = np.setdiff1d(self.memory_loop_idx, self.inhibitory_idx)
        self.memory_loop_idx = np.setdiff1d(self.memory_loop_idx, self.set_idx)
        self.memory_loop_idx = np.setdiff1d(self.memory_loop_idx, self.reset_idx)
        
        print(f"Cluster sizes - Set: {len(self.set_idx)}, Reset: {len(self.reset_idx)}, "
              f"Inhib: {len(self.inhibitory_idx)}, Memory: {len(self.memory_loop_idx)}")
              
        self._wire_latch()
        
    def _get_nodes_near(self, center, radius):
        dist = np.linalg.norm(self.net.positions - np.array(center), axis=1)
        return np.where(dist < radius)[0]
        
    def _wire_path(self, source_idx, target_idx, strength, excitatory=True):
        if len(source_idx) == 0 or len(target_idx) == 0: return
        for s in source_idx:
            for t in target_idx:
                if s == t: continue
                self.net.adj[s, t] = 1.0
                if excitatory:
                    self.net.weights[s, t] = strength
                else:
                    self.net.weights[s, t] = -strength
                    
    def _wire_latch(self):
        # 1. Recurrent Memory Loop (Self-sustaining)
        # We structurally enforce a super-critical reverberating topology
        if len(self.memory_loop_idx) > 0:
            for node in self.memory_loop_idx:
                targets = np.random.choice(self.memory_loop_idx, size=max(1, int(len(self.memory_loop_idx)*0.3)), replace=False)
                for t in targets:
                    if node != t:
                        self.net.adj[node, t] = 1.0
                        self.net.weights[node, t] = 0.5  # High recurrent strength
                    
        # 2. Set -> Memory Loop (Start the avalanche)
        self._wire_path(self.set_idx, self.memory_loop_idx, strength=0.5)
        
        # 3. Reset -> Inhibitory Interneurons
        self._wire_path(self.reset_idx, self.inhibitory_idx, strength=0.5)
        
        # 4. Inhibition: Inhibitory -> Memory Loop (Kill the avalanche)
        self._wire_path(self.inhibitory_idx, self.memory_loop_idx, strength=5.0, excitatory=False)
        
    def run_sequence(self):
        history = []
        self.net.voltage.fill(0)
        self.net.spikes.fill(0)
        
        print("\nFiring SET cluster (Storing 1)...")
        if len(self.set_idx) > 0:
            initial_set = np.random.choice(self.set_idx, size=int(len(self.set_idx)*0.8), replace=False)
            self.net.spikes[initial_set] = 1.0
            self.net.voltage[initial_set] = self.net.threshold * 2.0
            
        print("Running without input for 500 steps...")
        for _ in range(500):
            s = self.net.step(noise_std=0.0)
            history.append(s.copy())
            
        print("\nFiring RESET cluster (Storing 0)...")
        if len(self.reset_idx) > 0:
            initial_reset = np.random.choice(self.reset_idx, size=int(len(self.reset_idx)*0.8), replace=False)
            self.net.spikes[initial_reset] = 1.0
            self.net.voltage[initial_reset] = self.net.threshold * 2.0
            
        print("Running without input for 500 steps...")
        for _ in range(500):
            s = self.net.step(noise_std=0.0)
            history.append(s.copy())
            
        return np.array(history)

def plot_memory(history, sr):
    plt.figure(figsize=(12, 6))
    
    active_times = []
    active_neurons = []
    colors = []
    
    # Map neuron indices to groups for colored plotting
    group_map = {}
    for idx in sr.set_idx: group_map[idx] = ('Set', 'blue')
    for idx in sr.reset_idx: group_map[idx] = ('Reset', 'red')
    for idx in sr.inhibitory_idx: group_map[idx] = ('Inhibitory', 'orange')
    for idx in sr.memory_loop_idx: group_map[idx] = ('Memory Loop', 'green')
    
    for t, spikes in enumerate(history):
        fired = np.where(spikes > 0)[0]
        for idx in fired:
            active_times.append(t)
            active_neurons.append(idx)
            _, color = group_map.get(idx, ('Other', 'gray'))
            colors.append(color)
            
    # Raster plot
    plt.scatter(active_times, active_neurons, c=colors, s=2, alpha=0.7)
    
    plt.axvline(0, color='blue', linestyle='--', linewidth=2, label='SET Triggered')
    plt.axvline(500, color='red', linestyle='--', linewidth=2, label='RESET Triggered')
    
    import matplotlib.patches as mpatches
    patches = [
        mpatches.Patch(color='blue', label='Set Cluster'),
        mpatches.Patch(color='red', label='Reset Cluster'),
        mpatches.Patch(color='orange', label='Inhibitory Interneurons'),
        mpatches.Patch(color='green', label='Memory Loop Recurrence'),
        mpatches.Patch(color='gray', label='Background Noise')
    ]
    plt.legend(handles=patches, loc='upper right')
    
    plt.xlabel('Simulation Time Steps')
    plt.ylabel('Neuron Index')
    plt.title('Biological Set-Reset (SR) Latch (1-Bit Memory)')
    
    os.makedirs("./results", exist_ok=True)
    plt.tight_layout()
    plt.savefig("./results/figure7_sr_latch.png", dpi=300)
    print("\nSaved Figure 7: SR Latch Dynamics")

if __name__ == "__main__":
    print("================================================================")
    print("Biological 1-Bit Memory (SR Latch)")
    print("================================================================")
    sr = BiologicalSRLatch(num_nodes=1500)
    history = sr.run_sequence()
    plot_memory(history, sr)
