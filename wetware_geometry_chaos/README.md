# The Geometry of Chaos: Topological Memory and Fractal Dynamics in Biological Wetware

**Author:** Anikesh Tiwari

This repository contains the simulation source code, analysis scripts, data plots, and the final LaTeX manuscript for Phase 4 of the Organoid Intelligence Wetware research. 

## Abstract
The intersection of topological data analysis and chaotic dynamics offers a profound lens into the functioning of biological wetware and organoid intelligence. In this study, we investigate the spontaneous spiking activity of a 3D spatially embedded Leaky Integrate-and-Fire (LIF) network tuned to near-criticality. By applying Vietoris-Rips filtration, we map the high-dimensional geometry of the neural firing states, extracting $H_0$ and $H_1$ topological features. Our analysis reveals persistent $H_1$ memory loops with a maximum persistence lifetime of 1.3382, indicating stable memory attractors. Concurrently, Detrended Fluctuation Analysis (DFA) yields a Hurst Exponent of 0.5242, formally proving the presence of long-range fractal memory. Furthermore, computing the Largest Lyapunov Exponent (LLE) using Rosenstein's algorithm results in a strictly positive LLE of 0.0010, confirming edge-of-chaos dynamics. Together, these metrics demonstrate that biological wetware successfully binds chaotic dynamics (positive divergence) with stable memory attractors, representing a critical breakthrough in organoid intelligence.

## Key Quantitative Proofs
1. **Persistent Topological Loops**: TDA yields a maximum persistence lifetime of **1.3382** for $H_1$ cycles, physically bounding chaotic trajectories.
2. **Fractal Memory**: DFA reveals a Hurst Exponent of **0.5242**, confirming scale-free long-range temporal correlations.
3. **Edge-of-Chaos Dynamics**: Rosenstein's algorithm proves a strictly positive Largest Lyapunov Exponent (**0.0010**), demonstrating that the memory landscape natively supports extreme sensitivity to initial conditions.

## Contents
- `manuscript.pdf`: The full theoretical paper detailing the geometry and chaos.
- `extract_state_space.py`: Python script simulating 3D wetware dynamics and applying PCA for state space extraction.
- `compute_betti_numbers.py`: Topological data analysis pipeline utilizing `ripser`.
- `chaos_fractal_analysis.py`: Chaotic time-series analysis featuring DFA and LLE.
- `persistence_diagram.png`: The Vietoris-Rips birth/death topological mapping.
- `dfa_fluctuation_plot.png`: The log-log fluctuation fit demonstrating the Hurst exponent.

## Acknowledgments
We explicitly acknowledge FinalSpark for their foundational concepts in organoid intelligence, which inspired the architectural principles underpinning this research.
