import numpy as np
import matplotlib.pyplot as plt
import os

def run_thermodynamic_criticality():
    output_dir = "d:/New research/phase3_ising_mechanics"
    
    # Load h and J
    h_path = os.path.join(output_dir, "h_inferred.npy")
    J_path = os.path.join(output_dir, "J_inferred.npy")
    
    try:
        h = np.load(h_path)
        J = np.load(J_path)
    except FileNotFoundError:
        print(f"Error: Could not find {h_path} or {J_path}.")
        return
        
    N = len(h)
    
    # Temperature Sweep
    temperatures = np.arange(0.5, 2.05, 0.05)
    specific_heats = []
    
    # MCMC parameters
    n_samples = 20000
    burn_in = 5000
    total_steps = n_samples + burn_in
    
    print(f"Starting Temperature Sweep for {len(temperatures)} points...")
    
    t1_acceptance_rate = None
    
    for T in temperatures:
        # Metropolis-Hastings Sampling for the Boltzmann distribution
        # E(s) = - (sum_i h_i s_i + sum_{i<j} J_{ij} s_i s_j)
        
        # Initialize random spin state
        s = np.random.choice([-1, 1], size=N)
        energies = np.zeros(n_samples)
        
        accept_count = 0
        total_proposals = 0
        
        # Precompute current energy
        # J is symmetric, diagonal is 0, so sum_{i<j} J s_i s_j = 0.5 * sum_{i,j} J s_i s_j
        current_energy = - (np.dot(h, s) + 0.5 * np.dot(s, np.dot(J, s)))
        
        for step in range(total_steps):
            # One sweep = N spin flip attempts
            for _ in range(N):
                i = np.random.randint(N)
                eff_field = h[i] + np.dot(J[i, :], s)
                delta_E = 2.0 * s[i] * eff_field
                
                # Metropolis criterion
                if delta_E <= 0 or np.random.rand() < np.exp(-delta_E / T):
                    s[i] *= -1
                    current_energy += delta_E
                    if step >= burn_in:
                        accept_count += 1
            
            if step >= burn_in:
                energies[step - burn_in] = current_energy
                total_proposals += N
                
        # Compute Specific Heat: C_v = Var(E) / T^2
        variance_E = np.var(energies)
        C_v = variance_E / (T ** 2)
        specific_heats.append(C_v)
        
        # Check acceptance rate for T=1.0
        if np.isclose(T, 1.0):
            t1_acceptance_rate = accept_count / total_proposals
            
    # MANDATORY VERIFICATION 1
    print("\n--- MANDATORY VERIFICATION 1 ---")
    print(f"MCMC Acceptance Rate at T=1.0: {t1_acceptance_rate:.2%}")
    if 0.10 <= t1_acceptance_rate <= 0.80:
        print("Verification Passed: Acceptance rate is within the healthy range (10% - 80%).")
    else:
        print("Verification Failed: Acceptance rate is outside the healthy range.")
        
    # MANDATORY VERIFICATION 2
    print("\n--- MANDATORY VERIFICATION 2 ---")
    peak_idx = np.argmax(specific_heats)
    T_peak = temperatures[peak_idx]
    print(f"Max Specific Heat found at T = {T_peak:.2f}")
    
    if 0.9 <= T_peak <= 1.1:
        print("Verification Passed: Peak Specific Heat falls within the critical phase transition window (0.9 <= T <= 1.1).")
    else:
        print("Verification Failed: Peak Specific Heat is outside the critical window.")
        
    # 3. Visualization and Export
    plt.figure(figsize=(8, 5))
    plt.plot(temperatures, specific_heats, 'b-o', lw=2, markersize=6)
    plt.axvline(x=1.0, color='r', linestyle='--', lw=2, label='Native State (T=1.0)')
    
    plt.title('Specific Heat ($C_v$) vs Temperature ($T$)')
    plt.xlabel('Temperature $T$')
    plt.ylabel('Specific Heat $C_v$')
    plt.legend()
    plt.grid(True, alpha=0.3)
    plt.tight_layout()
    
    plot_path = os.path.join(output_dir, 'specific_heat_curve.png')
    plt.savefig(plot_path, dpi=150)
    plt.close()
    
    # MANDATORY VERIFICATION 3
    print("\n--- MANDATORY VERIFICATION 3 ---")
    if os.path.exists(plot_path):
        print("Verification Passed: Specific Heat curve secured.")
    else:
        print("Verification Failed: specific_heat_curve.png was not found on disk.")

if __name__ == "__main__":
    run_thermodynamic_criticality()
