import itertools
import numpy as np
import warnings
import sys
import os
import re

# Suppress warnings
warnings.filterwarnings('ignore')

from evaluator import Evaluator

def main():
    print("Running Hyperparameter Grid Search for Biological Boolean Logic Gates...")
    
    conn_radii = np.arange(1.5, 3.5, 0.5) # 1.5, 2.0, 2.5, 3.0
    thresholds = np.arange(0.8, 1.6, 0.2) # 0.8, 1.0, 1.2, 1.4
    leak_rates = np.arange(0.01, 0.11, 0.02) # 0.01, 0.03, 0.05, 0.07, 0.09
    
    best_configs = []
    
    # We will use 25 trials per truth table combination for grid search speed.
    NUM_TRIALS = 25
    NOISE = 0.1
    
    total_iters = len(conn_radii) * len(thresholds) * len(leak_rates)
    print(f"Total combinations to evaluate: {total_iters}")
    
    i = 0
    # Redirect stdout to avoid evaluator clutter
    old_stdout = sys.stdout
    
    for r, t, l in itertools.product(conn_radii, thresholds, leak_rates):
        i += 1
        # Print progress to real stdout
        sys.stdout = old_stdout
        print(f"\n--- Iteration {i}/{total_iters} | r={r:.2f}, t={t:.2f}, l={l:.2f} ---")
        
        try:
            # Suppress logs from evaluator
            sys.stdout = open(os.devnull, 'w')
            
            evaluator = Evaluator(num_trials=NUM_TRIALS, max_steps=50, num_nodes=1000, conn_radius=r, leak=l, threshold=t)
            
            # 1. Evaluate BER for AND and OR
            and_res = evaluator.evaluate_gate(evaluator.and_gate, [NOISE])[NOISE]
            or_res = evaluator.evaluate_gate(evaluator.or_gate, [NOISE])[NOISE]
            
            ber_and = and_res['BER']
            ber_or = or_res['BER']
            
            # 2. Collect spontaneous avalanches for tau fitting
            sizes, _ = evaluator.collect_spontaneous_avalanches(steps=3000, noise_std=NOISE)
            
            tau = 0.0
            if len(sizes) > 10:
                import powerlaw
                fit = powerlaw.Fit(sizes, discrete=True, verbose=False)
                tau = fit.power_law.alpha
                    
            sys.stdout = old_stdout
            
            penalty = abs(tau - 1.67) if tau > 0 else 5.0 # large penalty if no avalanches
            
            # Fitness: we want to minimize BER and minimize penalty
            fitness = ber_and + ber_or + (penalty * 0.1) 
            
            best_configs.append({
                'r': r, 't': t, 'l': l,
                'ber_and': ber_and,
                'ber_or': ber_or,
                'tau': tau,
                'fitness': fitness
            })
            
            print(f"  Result: BER_AND={ber_and:.2f}, BER_OR={ber_or:.2f}, tau={tau:.2f} => Fitness={fitness:.4f}")
            
        except Exception as e:
            sys.stdout = old_stdout
            print(f"  Error evaluating config: {e}")
            
    sys.stdout = old_stdout
    
    # Sort by fitness (lowest is best)
    best_configs.sort(key=lambda x: x['fitness'])
    
    print("\n=================================================")
    print("TOP 3 HYPERPARAMETER CONFIGURATIONS")
    print("=================================================")
    for idx in range(min(3, len(best_configs))):
        c = best_configs[idx]
        print(f"Rank {idx+1}: conn_radius={c['r']:.2f}, threshold={c['t']:.2f}, leak={c['l']:.2f}")
        print(f"  -> BER_AND: {c['ber_and']:.2f}, BER_OR: {c['ber_or']:.2f}, tau: {c['tau']:.2f}, Fitness: {c['fitness']:.4f}")
    
    best = best_configs[0]
    update_script(best)
    
def update_script(best):
    with open("substrate.py", "r") as f:
        content = f.read()
    
    # Update substrate.py defaults safely using regex
    content = re.sub(r'conn_radius=\d+\.\d+', f'conn_radius={best["r"]}', content)
    content = re.sub(r'leak=\d+\.\d+', f'leak={best["l"]}', content)
    content = re.sub(r'threshold=\d+\.\d+', f'threshold={best["t"]}', content)
    
    with open("substrate.py", "w") as f:
        f.write(content)
        
    with open("gates.py", "r") as f:
        content2 = f.read()
        
    content2 = re.sub(r'conn_radius=\d+\.\d+', f'conn_radius={best["r"]}', content2)
    content2 = re.sub(r'leak=\d+\.\d+', f'leak={best["l"]}', content2)
    content2 = re.sub(r'threshold=\d+\.\d+', f'threshold={best["t"]}', content2)
    
    with open("gates.py", "w") as f:
        f.write(content2)
        
    print(f"\nSuccessfully updated substrate.py and gates.py with the winning parameters: r={best['r']:.2f}, t={best['t']:.2f}, l={best['l']:.2f}")

if __name__ == "__main__":
    main()
