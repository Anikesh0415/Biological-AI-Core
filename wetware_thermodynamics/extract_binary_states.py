import numpy as np
import pandas as pd
import os

def binarize_spiketrain(spike_times, neuron_ids, num_neurons, bin_size_ms=1.0, t_max_ms=None, ising_spin_format=False):
    """
    Converts raw spike times into a binary state matrix for Ising model / Maximum Entropy analysis.

    Parameters:
    - spike_times: array-like of spike timestamps (in ms).
    - neuron_ids: array-like of neuron indices corresponding to each spike.
    - num_neurons: total number of neurons (N).
    - bin_size_ms: time resolution for binning (Delta t). Critical for defining avalanches.
    - t_max_ms: maximum time to consider. If None, uses max spike time.
    - ising_spin_format: if True, returns states as {-1, 1} (physics convention). 
                         if False, returns {0, 1} (Schneidman 2006 neuroscience convention).

    Returns:
    - S: (N x T) binary matrix representing the state of the network over time.
    - time_bins: (T,) array of time bin edges.
    """
    if t_max_ms is None:
        t_max_ms = np.max(spike_times)
        
    num_bins = int(np.ceil(t_max_ms / bin_size_ms))
    S = np.zeros((num_neurons, num_bins), dtype=np.int8)
    
    # Map spike times to bin indices
    bin_indices = np.floor(spike_times / bin_size_ms).astype(int)
    
    # Clip indices to prevent out-of-bounds at the exact edge
    bin_indices = np.clip(bin_indices, 0, num_bins - 1)
    
    # Populate the binary matrix
    # Sets bin to 1 if there is at least 1 spike in that bin for that neuron
    S[neuron_ids, bin_indices] = 1
    
    if ising_spin_format:
        # Convert {0, 1} to {-1, 1}
        S = 2 * S - 1
        
    time_bins = np.arange(num_bins) * bin_size_ms
    return S, time_bins

def calculate_empirical_statistics(S, ising_spin_format=False):
    """
    Calculates the first (mean firing rates/magnetization) and 
    second moments (pairwise correlations) for the Maximum Entropy Model.
    
    P(s) = (1/Z) * exp( sum_i h_i s_i + sum_{i<j} J_{ij} s_i s_j )
    
    Parameters:
    - S: (N x T) state matrix
    
    Returns:
    - means: <s_i>
    - covariances: <s_i s_j>
    """
    T = S.shape[1]
    
    # First moment <s_i>
    means = np.mean(S, axis=1)
    
    # Second moment <s_i s_j>
    # S @ S.T computes the unnormalized correlation matrix. Divide by T for expectation.
    covariances = (S @ S.T) / T
    
    return means, covariances

if __name__ == "__main__":
    # ---------------------------------------------------------
    # Mock data generation for the LIF Organoid Substrate
    # ---------------------------------------------------------
    print("Initializing Binary State Extraction for LIF Organoid Substrate...")
    
    # Mock network parameters
    N = 100       # Number of neurons to subsample
    T_sim = 5000  # 5 seconds of simulation (ms)
    
    # Generate mock spikes (Replace with actual HDF5/CSV loading from wetware substrate)
    np.random.seed(42)
    n_spikes = 15000
    mock_spike_times = np.sort(np.random.uniform(0, T_sim, n_spikes))
    mock_neuron_ids = np.random.randint(0, N, n_spikes)
    
    # 1. Choose a Delta t based on the typical avalanche timescale
    bin_width_ms = 2.0 
    
    # 2. Extract Binary States
    print(f"Binning spikes with Delta t = {bin_width_ms} ms...")
    S, t_bins = binarize_spiketrain(
        mock_spike_times, 
        mock_neuron_ids, 
        num_neurons=N, 
        bin_size_ms=bin_width_ms, 
        ising_spin_format=False  # Keep as {0, 1} for Schneidman convention, or True for {-1, 1}
    )
    print(f"Extracted state matrix of shape: {S.shape} (N neurons x T time bins)")
    
    # 3. Compute Empirical Statistics for the Inverse Ising Problem
    means, covariances = calculate_empirical_statistics(S)
    print(f"Calculated sufficient statistics for {N} neurons.")
    
    # Save outputs for the MaxEnt solver (e.g., gradient descent, pseudo-likelihood)
    output_dir = "ising_data_out"
    os.makedirs(output_dir, exist_ok=True)
    np.save(os.path.join(output_dir, "binary_states.npy"), S)
    np.save(os.path.join(output_dir, "empirical_means.npy"), means)
    np.save(os.path.join(output_dir, "empirical_covariances.npy"), covariances)
    
    print(f"Data successfully saved to {output_dir}/")
    print("Ready for Inverse Ising mapping to extract h_i and J_ij.")
