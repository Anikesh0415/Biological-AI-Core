import numpy as np
import json
import os
import time
from evaluator import Evaluator
from visualizer import Visualizer

def main():
    print("================================================================")
    print("STDP Learning in Critical Neuronal Avalanches")
    print("================================================================")
    
    NUM_TRIALS = 100
    NUM_NODES = 1000
    NOISE_STEPS = np.arange(0.0, 0.55, 0.05)
    
    print(f"\nInitializing 3D Substrate (N={NUM_NODES}) with STDP and tuned spatial clusters...")
    evaluator = Evaluator(num_trials=NUM_TRIALS, max_steps=50, num_nodes=NUM_NODES, conn_radius=1.5, leak=0.01, threshold=0.8)
    visualizer = Visualizer(results_dir="./results")
    
    print("\n--- Phase 0: Criticality Validation ---")
    sizes, durations = evaluator.collect_spontaneous_avalanches(steps=50000, noise_std=0.01)
    print(f"Recorded {len(sizes)} spontaneous avalanches.")
    visualizer.plot_criticality(sizes, durations)
    
    print("\n--- Phase 1: Pre-Training Benchmarking ---")
    print("Evaluating baseline AND Gate BER...")
    and_results_pre = evaluator.evaluate_gate(evaluator.and_gate, NOISE_STEPS)
    print("Evaluating baseline OR Gate BER...")
    or_results_pre = evaluator.evaluate_gate(evaluator.or_gate, NOISE_STEPS)
    
    print("\n--- Phase 2: STDP Hebbian Training Phase ---")
    print("Training AND Gate (200 epochs, co-stimulating A & B)...")
    evaluator.and_gate.train(epochs=200)
    print("Training OR Gate (200 epochs, alternating A & B)...")
    evaluator.or_gate.train(epochs=200)
    
    print("\n--- Phase 3: Post-Training Benchmarking ---")
    print("Evaluating Post-Train AND Gate BER...")
    and_results_post = evaluator.evaluate_gate(evaluator.and_gate, NOISE_STEPS)
    print("Evaluating Post-Train OR Gate BER...")
    or_results_post = evaluator.evaluate_gate(evaluator.or_gate, NOISE_STEPS)
    
    print("\n--- Phase 4: Visualization & Reporting ---")
    visualizer.plot_training_comparison(and_results_pre, and_results_post, or_results_pre, or_results_post)
    
    # Save weight matrices for visualizer
    visualizer.plot_weight_matrices(evaluator.and_gate.pre_train_weights, evaluator.and_gate.net.weights, "AND")
    visualizer.plot_weight_matrices(evaluator.or_gate.pre_train_weights, evaluator.or_gate.net.weights, "OR")
    
    print("\nPipeline Complete! Figures and report saved to ./results directory.")

if __name__ == "__main__":
    t0 = time.time()
    main()
    print(f"Total Execution Time: {time.time() - t0:.2f} seconds")
