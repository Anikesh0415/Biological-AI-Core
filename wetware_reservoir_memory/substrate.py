import numpy as np
from scipy.spatial import distance_matrix

class BiologicalWetware:
    """
    Core simulation engine for 3D spatially embedded biological wetware networks.
    Implements a Leaky Integrate-and-Fire (LIF) network tuned to near-criticality
    and supports Spike-Timing-Dependent Plasticity (STDP).
    """
    
    def __init__(self, num_nodes=1000, space_size=10.0, conn_radius=1.5, leak=0.01, threshold=0.8):
        self.N = num_nodes
        self.space_size = space_size
        
        # 1. Substrate Generation: 3D spatial embedding
        self.positions = np.random.rand(self.N, 3) * space_size
        
        # 2. Distance-dependent synaptic connectivity
        dist_mat = distance_matrix(self.positions, self.positions)
        prob_mat = np.exp(-dist_mat / conn_radius)
        np.fill_diagonal(prob_mat, 0) # No self-connections
        
        self.adj = (np.random.rand(self.N, self.N) < prob_mat).astype(float)
        
        # Initialize dense synaptic weights
        self.weights = self.adj * np.random.uniform(0.1, 1.0, (self.N, self.N))
        
        # LIF State Variables
        self.leak = leak
        self.threshold = threshold
        self.voltage = np.zeros(self.N)
        self.spikes = np.zeros(self.N)
        
        # STDP Variables
        self.A_plus = 0.05
        self.A_minus = 0.05
        self.tau_plus = 10.0
        self.tau_minus = 10.0
        self.W_max = 2.0
        self.trace_pre = np.zeros(self.N)
        self.trace_post = np.zeros(self.N)
        
        self.last_max_dw = 0.0
        self.normalize_weights(target_sigma=1.0)
        
    def normalize_weights(self, target_sigma=1.0):
        """Homeostatic plasticity to maintain criticality."""
        out_degrees = self.weights.sum(axis=1, keepdims=True)
        out_degrees[out_degrees == 0] = 1.0
        self.weights = self.weights / out_degrees * target_sigma

    def step(self, input_current=None, noise_std=0.0, apply_stdp=False):
        if input_current is None:
            input_current = np.zeros(self.N)
            
        noise = np.random.normal(0, noise_std, self.N)
        
        # Ensure spontaneous firing to seed avalanches when checking criticality
        if noise_std > 0:
            spontaneous_spikes = (np.random.rand(self.N) < (noise_std * 0.05)).astype(float)
            noise += spontaneous_spikes * (self.threshold * 1.5)
            
        synaptic_input = self.weights.T @ self.spikes
        
        self.voltage = self.leak * self.voltage + synaptic_input + input_current + noise
        self.spikes = (self.voltage >= self.threshold).astype(float)
        self.voltage[self.spikes > 0] = 0.0
        
        if apply_stdp:
            # STDP Local Rule via Eligibility Traces
            dW_pot = self.A_plus * np.outer(self.trace_pre, self.spikes)
            dW_dep = self.A_minus * np.outer(self.spikes, self.trace_post)
            
            dW = (dW_pot - dW_dep) * self.adj
            self.last_max_dw = np.max(np.abs(dW))
            
            self.weights += dW
            self.weights = np.clip(self.weights, 0, self.W_max)
            
            # Update traces AFTER weight modification
            self.trace_pre = self.trace_pre * np.exp(-1.0 / self.tau_plus) + self.spikes
            self.trace_post = self.trace_post * np.exp(-1.0 / self.tau_minus) + self.spikes
            
        return self.spikes

    def run_avalanche(self, initial_spikes, max_steps=50, noise_std=0.0, apply_stdp=False, force_steps=0):
        self.voltage = np.zeros(self.N)
        self.spikes = np.zeros(self.N)
        self.last_max_dw = 0.0
        
        if len(initial_spikes) > 0:
            self.spikes[initial_spikes] = 1.0
        
        avalanche_size = len(initial_spikes)
        history = [self.spikes.copy()]
        
        for step in range(max_steps):
            s = self.step(noise_std=noise_std, apply_stdp=apply_stdp)
            
            # Force stimulus to keep firing during the training window
            if step < force_steps and len(initial_spikes) > 0:
                self.voltage[initial_spikes] = self.threshold * 2.0
                
            active_count = np.sum(s)
            
            # If nothing is firing and no noise/STDP is pushing it, early exit
            if active_count == 0 and noise_std == 0.0 and step >= force_steps:
                break
                
            avalanche_size += active_count
            history.append(s.copy())
            
        return avalanche_size, np.array(history)
