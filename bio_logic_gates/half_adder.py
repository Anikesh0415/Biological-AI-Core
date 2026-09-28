import numpy as np
import os
import matplotlib.pyplot as plt
from substrate import BiologicalWetware

class BiologicalHalfAdder:
    def __init__(self, num_nodes=1500, conn_radius=1.5, leak=0.01, threshold=0.8):
        self.net = BiologicalWetware(num_nodes=num_nodes, conn_radius=conn_radius, leak=leak, threshold=threshold)
        
        # Define Regions spatially
        self.input_A_idx = self._get_nodes_near([3.0, 3.0, 5.0], radius=1.0)
        self.input_B_idx = self._get_nodes_near([7.0, 3.0, 5.0], radius=1.0)
        
        self.inhibitory_idx = self._get_nodes_near([5.0, 5.0, 5.0], radius=1.0)
        self.out_carry_idx = self._get_nodes_near([5.0, 7.0, 5.0], radius=1.0) # AND
        self.out_sum_idx = self._get_nodes_near([5.0, 9.0, 5.0], radius=1.0)   # XOR
        
        print(f"Cluster sizes - InA: {len(self.input_A_idx)}, InB: {len(self.input_B_idx)}, "
              f"Inhib: {len(self.inhibitory_idx)}, Carry: {len(self.out_carry_idx)}, Sum: {len(self.out_sum_idx)}")
              
        self._wire_half_adder()
        
    def _get_nodes_near(self, center, radius):
        dist = np.linalg.norm(self.net.positions - np.array(center), axis=1)
        return np.where(dist < radius)[0]
        
    def _wire_path(self, source_idx, target_idx, strength, excitatory=True):
        if len(source_idx) == 0 or len(target_idx) == 0: return
        for s in source_idx:
            for t in target_idx:
                self.net.adj[s, t] = 1.0
                if excitatory:
                    self.net.weights[s, t] = strength
                else:
                    self.net.weights[s, t] = -strength # Inhibitory
                    
    def _wire_half_adder(self):
        # We manually structure the pathways that STDP would ideally learn over long timescales.
        
        # 1. OR routing: A -> Sum and B -> Sum (Strong)
        self._wire_path(self.input_A_idx, self.out_sum_idx, strength=0.25)
        self._wire_path(self.input_B_idx, self.out_sum_idx, strength=0.25)
        
        # 2. AND routing: A -> Carry and B -> Carry (Weak, needs interference)
        self._wire_path(self.input_A_idx, self.out_carry_idx, strength=0.12)
        self._wire_path(self.input_B_idx, self.out_carry_idx, strength=0.12)
        
        # 3. Inhibitory triggering: A -> Inhibitory and B -> Inhibitory (Weak, needs interference)
        self._wire_path(self.input_A_idx, self.inhibitory_idx, strength=0.12)
        self._wire_path(self.input_B_idx, self.inhibitory_idx, strength=0.12)
        
        # 4. Inhibition: Inhibitory -> Sum (Strong negative)
        self._wire_path(self.inhibitory_idx, self.out_sum_idx, strength=2.0, excitatory=False)

    def evaluate(self, A, B, noise_std=0.01, max_steps=40):
        initial_spikes = []
        if A and len(self.input_A_idx) > 0:
            initial_spikes.extend(np.random.choice(self.input_A_idx, size=int(len(self.input_A_idx)*0.8), replace=False))
        if B and len(self.input_B_idx) > 0:
            initial_spikes.extend(np.random.choice(self.input_B_idx, size=int(len(self.input_B_idx)*0.8), replace=False))
            
        _, history = self.net.run_avalanche(initial_spikes, max_steps=max_steps, noise_std=noise_std, force_steps=4)
        
        sum_fired = False
        carry_fired = False
        sum_threshold = max(1, len(self.out_sum_idx) * 0.20)
        carry_threshold = max(1, len(self.out_carry_idx) * 0.20)
        
        for spikes in history:
            if np.sum(spikes[self.out_sum_idx]) >= sum_threshold:
                sum_fired = True
            if np.sum(spikes[self.out_carry_idx]) >= carry_threshold:
                carry_fired = True
                
        return (1 if sum_fired else 0), (1 if carry_fired else 0), history

def plot_half_adder(history, positions, ha):
    plt.figure(figsize=(10, 6))
    ax = plt.axes(projection='3d')
    total_spikes = np.sum(history, axis=0)
    
    # Plot faint background
    ax.scatter(positions[:, 0], positions[:, 1], positions[:, 2], c='gray', s=5, alpha=0.05)
    
    # Plot active neurons
    active_idx = np.where(total_spikes > 0)[0]
    if len(active_idx) > 0:
        scatter = ax.scatter(positions[active_idx, 0], positions[active_idx, 1], positions[active_idx, 2], 
                             c=total_spikes[active_idx], cmap='hot', s=30, alpha=0.9)
        plt.colorbar(scatter, label='Spike Count', shrink=0.5)
        
    # Highlights
    ax.scatter([3.0], [3.0], [5.0], c='blue', s=200, marker='X', label='Input A')
    ax.scatter([7.0], [3.0], [5.0], c='cyan', s=200, marker='X', label='Input B')
    ax.scatter([5.0], [5.0], [5.0], c='red', s=200, marker='o', label='Inhibitory')
    ax.scatter([5.0], [7.0], [5.0], c='green', s=200, marker='*', label='Output Carry')
    ax.scatter([5.0], [9.0], [5.0], c='orange', s=200, marker='*', label='Output Sum')
    
    ax.set_title('Biological Half-Adder: State (1, 1)\nSum is Inhibited, Carry Fires')
    ax.legend()
    
    os.makedirs("./results", exist_ok=True)
    plt.tight_layout()
    plt.savefig("./results/figure6_half_adder.png", dpi=300)
    print("Saved Figure 6: Half Adder Spatiotemporal Dynamics")

def main():
    print("================================================================")
    print("Biological Half-Adder (XOR & AND Logic)")
    print("================================================================")
    
    ha = BiologicalHalfAdder(num_nodes=1500, conn_radius=1.5, leak=0.01, threshold=0.8)
    
    print("\nEvaluating Truth Table (100 Monte Carlo Trials per state)...")
    
    states = [(False, False), (True, False), (False, True), (True, True)]
    expected = {(False, False): (0, 0), (True, False): (1, 0), (False, True): (1, 0), (True, True): (0, 1)}
    
    for (A, B) in states:
        sum_correct = 0
        carry_correct = 0
        exp_sum, exp_carry = expected[(A, B)]
        
        last_history = None
        for _ in range(100):
            res_sum, res_carry, history = ha.evaluate(A, B, noise_std=0.01)
            if res_sum == exp_sum: sum_correct += 1
            if res_carry == exp_carry: carry_correct += 1
            if A and B: last_history = history
            
        print(f"State A={int(A)}, B={int(B)} | Expected (Sum={exp_sum}, Carry={exp_carry})")
        print(f"  -> Sum Accuracy: {sum_correct/100.0:.0%} | Carry Accuracy: {carry_correct/100.0:.0%}")
        
    print("\nGenerating Spatiotemporal Plot for (1, 1)...")
    if last_history is not None:
        plot_half_adder(last_history, ha.net.positions, ha)

if __name__ == "__main__":
    main()
