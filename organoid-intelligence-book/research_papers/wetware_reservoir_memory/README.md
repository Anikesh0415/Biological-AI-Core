# The Memory-Nonlinearity Trade-off in Critical Neural Substrates

**Repository:** `wetware_reservoir_memory`  
**Author:** Anikesh Tiwari

## Overview
This repository contains the simulation source code, generated data visualizations, and the full LaTeX manuscript evaluating the impact of Tsodyks-Markram (TM) Short-Term Plasticity (STP) on biological reservoir computing.

We utilize a 3D spatially embedded Leaky Integrate-and-Fire (LIF) network, optimized to a critical branching parameter ($\sigma \approx 1.0$), and expand the reservoir states using Multi-Timescale Synaptic Filters. 

## Key Discovery: The Trade-off
Our benchmarks reveal a fundamental computational divergence in wetware dynamics:
1. **Linear Fading Memory Expansion:** The introduction of STP nearly doubled the Linear Memory Capacity ($MC = 0.12 \to 0.22$), proving that dynamic synaptic facilitation and depression act as a powerful hidden temporal buffer.
2. **Non-Linearity Degradation:** The highly non-linear NARMA-10 task performance degraded significantly (NRMSE $0.90 \to 1.07$). The dynamic noise introduced by constantly shifting synaptic efficacies disrupts the stable polynomial signal mixing required for complex non-linear functions.

This formally establishes a Memory-Nonlinearity trade-off, indicating that future Organoid Intelligence logic circuits must compartmentalize static processing domains from highly plastic memory buffers.

## Repository Contents
- `manuscript.tex`: The raw LaTeX manuscript.
- `manuscript.pdf`: The compiled academic paper.
- `reservoir_benchmark.py`: Baseline reservoir simulation script.
- `reservoir_stp_benchmark.py`: STP-enabled simulation script with Tsodyks-Markram equations.
- `substrate.py`: Core 3D LIF simulation engine.
- `stp_memory_capacity.png` & `stp_narma10_comparison.png`: Generated benchmark graphs.

## Acknowledgments
We explicitly credit **FinalSpark** for their foundational contributions to organoid intelligence and wetware computing.
