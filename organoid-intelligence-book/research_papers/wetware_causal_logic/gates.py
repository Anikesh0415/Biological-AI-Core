import numpy as np
from substrate import BiologicalWetware

class LogicGate:
    def __init__(self, gate_type="AND", num_nodes=1000, conn_radius=1.5, leak=0.01, threshold=0.8):
        self.gate_type = gate_type
        
        self.net = BiologicalWetware(
            num_nodes=num_nodes, 
            space_size=10.0, 
            conn_radius=conn_radius, 
            leak=leak, 
            threshold=threshold
        )
        
        # Spatial optimization: move inputs and output closer together
        self.input_A_idx = self._get_nodes_near(center=[4.0, 4.0, 5.0], radius=1.0)
        self.input_B_idx = self._get_nodes_near(center=[6.0, 4.0, 5.0], radius=1.0)
        self.output_idx = self._get_nodes_near(center=[5.0, 6.0, 5.0], radius=1.0)
        
        self.pre_train_weights = self.net.weights.copy()
        
        self._tune_for_gate()

    def _get_nodes_near(self, center, radius):
        dist = np.linalg.norm(self.net.positions - np.array(center), axis=1)
        return np.where(dist < radius)[0]

    def _tune_for_gate(self):
        # Post-homeostasis adjustments for logic function
        if self.gate_type == "AND":
            self.net.weights *= 0.85
            self.net.threshold = 0.9
        elif self.gate_type == "OR":
            self.net.weights *= 1.05
            self.net.threshold = 0.75

    def train(self, epochs=200):
        """
        Hebbian conditioning phase using STDP to self-organize valid wires.
        """
        # Start at generic criticality for fair learning
        self.net.normalize_weights(target_sigma=1.0)
        
        for epoch in range(epochs):
            self.net.trace_pre *= 0
            self.net.trace_post *= 0
            
            initial_spikes = []
            if self.gate_type == "AND":
                # Co-stimulate A and B
                initial_spikes.extend(np.random.choice(self.input_A_idx, size=int(len(self.input_A_idx)*0.5)))
                initial_spikes.extend(np.random.choice(self.input_B_idx, size=int(len(self.input_B_idx)*0.5)))
            else: # OR
                # Individually stimulate
                cluster = self.input_A_idx if epoch % 2 == 0 else self.input_B_idx
                initial_spikes.extend(np.random.choice(cluster, size=int(len(cluster)*0.5)))
                
            # Run avalanche with STDP turned on and force the input stimulus for 5 steps 
            # to guarantee the wave crashes into the output cluster.
            self.net.run_avalanche(initial_spikes, max_steps=30, noise_std=0.05, apply_stdp=True, force_steps=5)
            
            # Print STDP tracking
            if epoch % 50 == 0:
                print(f"[{self.gate_type}] Epoch {epoch}: Max weight change = {self.net.last_max_dw:.6f}")
            
        # Re-tune logic thresholds after wiring is complete
        self.net.normalize_weights(target_sigma=1.0)
        self._tune_for_gate()

    def evaluate(self, input_A_active, input_B_active, noise_std=0.05, max_steps=50):
        initial_spikes = []
        if input_A_active:
            initial_spikes.extend(np.random.choice(self.input_A_idx, size=int(len(self.input_A_idx)*0.5)))
        if input_B_active:
            initial_spikes.extend(np.random.choice(self.input_B_idx, size=int(len(self.input_B_idx)*0.5)))
            
        if not initial_spikes and noise_std == 0:
            return 0, np.zeros((1, self.net.N))

        size, history = self.net.run_avalanche(initial_spikes, max_steps=max_steps, noise_std=noise_std)
        
        output_fired = False
        activation_threshold = max(1, len(self.output_idx) * 0.15)
        
        for step_spikes in history:
            output_activity = np.sum(step_spikes[self.output_idx])
            if output_activity >= activation_threshold:
                output_fired = True
                break
                
        return 1 if output_fired else 0, history
