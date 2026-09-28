import numpy as np
import json
import os
from gates import LogicGate

class Evaluator:
    def __init__(self, num_trials=1000, max_steps=50, num_nodes=2000, conn_radius=2.5, leak=0.5, threshold=1.0):
        self.num_trials = num_trials
        self.max_steps = max_steps
        self.num_nodes = num_nodes
        self.and_gate = LogicGate("AND", num_nodes=self.num_nodes, conn_radius=conn_radius, leak=leak, threshold=threshold)
        self.or_gate = LogicGate("OR", num_nodes=self.num_nodes, conn_radius=conn_radius, leak=leak, threshold=threshold)

    def _expected_output(self, gate_type, A, B):
        if gate_type == "AND":
            return 1 if (A and B) else 0
        elif gate_type == "OR":
            return 1 if (A or B) else 0
        return 0

    def evaluate_gate(self, gate, noise_range):
        """
        Sweeps noise levels and computes BER, FPR, FNR, SNR.
        """
        results = {}
        for noise in noise_range:
            print(f"  Evaluating {gate.gate_type} Gate at Noise = {noise:.2f}...")
            stats = {'errors': 0, 'false_positives': 0, 'false_negatives': 0, 'total': 0}
            
            # Record output signal strengths to compute SNR
            # Signal = Activity level of output cluster when 1 is expected
            # Noise = Activity level of output cluster when 0 is expected
            signal_activities = []
            noise_activities = []
            
            for A in [False, True]:
                for B in [False, True]:
                    expected = self._expected_output(gate.gate_type, A, B)
                    
                    for _ in range(self.num_trials):
                        output, history = gate.evaluate(A, B, noise_std=noise, max_steps=self.max_steps)
                        
                        # Calculate output activity for SNR
                        max_out_activity = 0
                        for step_spikes in history:
                            act = np.sum(step_spikes[gate.output_idx])
                            if act > max_out_activity:
                                max_out_activity = act
                                
                        if expected == 1:
                            signal_activities.append(max_out_activity)
                        else:
                            noise_activities.append(max_out_activity)
                            
                        # Error tracking
                        if output != expected:
                            stats['errors'] += 1
                            if expected == 0 and output == 1:
                                stats['false_positives'] += 1
                            elif expected == 1 and output == 0:
                                stats['false_negatives'] += 1
                        stats['total'] += 1
            
            # Compute rates
            ber = stats['errors'] / stats['total']
            fpr = stats['false_positives'] / stats['total']
            fnr = stats['false_negatives'] / stats['total']
            
            mu_s = np.mean(signal_activities) if signal_activities else 0
            mu_n = np.mean(noise_activities) if noise_activities else 0
            var_s = np.var(signal_activities) if signal_activities else 0
            var_n = np.var(noise_activities) if noise_activities else 0
            
            # SNR = (mu_s - mu_n)^2 / (var_s + var_n)
            denom = var_s + var_n
            snr = ((mu_s - mu_n)**2 / denom) if denom > 0 else 0
            
            results[noise] = {
                'BER': ber,
                'FPR': fpr,
                'FNR': fnr,
                'SNR': snr
            }
            
        return results

    def collect_spontaneous_avalanches(self, steps=50000, noise_std=0.05):
        """
        Runs the network continuously to collect empirical avalanche sizes and durations.
        """
        print(f"Collecting spontaneous avalanches for {steps} steps at noise {noise_std}...")
        sizes = []
        durations = []
        
        # We'll use the OR gate substrate as it is tuned exactly to criticality (sigma ~ 1.05)
        # However, BiologicalWetware default is also close. Let's use the AND gate substrate without scaling.
        # To be purely spontaneous, we don't stimulate, just let noise drive it.
        net = self.or_gate.net
        net.voltage = np.zeros(net.N)
        net.spikes = np.zeros(net.N)
        
        current_size = 0
        current_duration = 0
        is_active = False
        
        for i in range(steps):
            if i % 5000 == 0:
                print(f"  Step {i}/{steps}")
            s = net.step(noise_std=noise_std)
            active_count = np.sum(s)
            
            if active_count > 0:
                is_active = True
                current_size += active_count
                current_duration += 1
            else:
                if is_active:
                    # Avalanche ended
                    sizes.append(current_size)
                    durations.append(current_duration)
                    current_size = 0
                    current_duration = 0
                    is_active = False
                    
        return np.array(sizes), np.array(durations)

    def generate_constructive_interference_example(self):
        """
        Extracts a single spatiotemporal history for visualization of Input A + Input B -> AND gate.
        """
        _, history = self.and_gate.evaluate(True, True, noise_std=0.05, max_steps=50)
        return history, self.and_gate.net.positions
