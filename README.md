# Biological-AI-Core: Organoid Intelligence & Wetware Computing Architecture

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Hardware](https://img.shields.io/badge/Interface-FinalSpark%20MEA-darkgreen.svg)](#)
[![Math](https://img.shields.io/badge/Methods-Inverse%20Ising%20%7C%20Transfer%20Entropy-crimson.svg)](#)
[![Funding](https://img.shields.io/badge/Pre--Seed-Raising%20%2460k-gold.svg)](#pre-seed-funding--compute-roadmap)

> **Bypassing the Thermodynamic Wall of Silicon.**  
> Modern deep learning compute has collided with a physical wall: megawatt datacenter clusters running dense matrix multiplication to approximate basic intelligence. In contrast, 3D biological cerebral organoids compute, adapt, and self-organize at a thermodynamic cost of **~20 Watts**.  
>  
> **Biological-AI-Core** is the unified monorepo consolidating 10 peer-reviewed research papers, open-source algorithms, and electrophysiological signal-processing pipelines engineered to interface with living human brain organoids for biocomputation.

---

## ⚡ Core Theoretical & Mathematical Framework

Our biological computing architecture processes live microelectrode recordings from the **FinalSpark Microelectrode Array (MEA)** (16/32/64-channel extracellular electrophysiology platforms):

```
                              [Living Cerebral Organoid]
                                           │
                                (FinalSpark 64-Ch MEA)
                                           ▼
                              [Extracellular Spike Trains]
                                           │
              ┌────────────────────────────┴────────────────────────────┐
              ▼                                                         ▼
   [Inverse Ising Reconstruction]                        [Bivariate Transfer Entropy]
 H = -∑ J_ij s_i s_j - ∑ h_i s_i                    T_{X→Y} = ∑ p(y_t+1, y_t, x_t) log [p / p]
              │                                                         │
 Effective Synaptic Coupling Topology                  Directional Causal Information Flow
              └────────────────────────────┬────────────────────────────┘
                                           ▼
                          [Closed-Loop Neuromorphic Policy]
```

### 1. Maximum Entropy & Inverse Ising Mechanics
We map multielectrode spiking time-bins into instantaneous spin configurations $s \in \{-1, +1\}^N$. To reconstruct the latent functional connectome without confounding indirect correlations, we solve the inverse problem for the pairwise maximum entropy Boltzmann distribution:

$$P(s) = \frac{1}{Z} \exp\left( \sum_{i < j} J_{ij} s_i s_j + \sum_{i} h_i s_i \right)$$

Through Persistent Contrastive Divergence (PCD) and pseudo-likelihood maximization, our algorithms fit the effective synaptic coupling matrix $J_{ij}$ and local intrinsic excitabilities $h_i$ to assess organoid criticality and functional plasticity.

### 2. Bivariate Transfer Entropy (TE) for Causal Information Flow
To quantify non-linear, directed information routing between neural assemblies, we compute bivariate transfer entropy across MEA channel pairs:

$$T_{X \to Y} = \sum_{y_{t+1}, y_t, x_t} p(y_{t+1}, y_t, x_t) \log_2 \left( \frac{p(y_{t+1} \mid y_t, x_t)}{p(y_{t+1} \mid y_t)} \right)$$

By applying spike-jittering surrogate null models, we isolate true directional synaptic transmission from shared volume conduction and external stimulus artifacts.

---

## 🚀 Pre-Seed Funding & Compute Roadmap

I am a 16-year-old independent researcher and the lead author behind the research publications and codebases unified in this repository.

> [!IMPORTANT]
> **We are currently raising a \$60,000 Pre-Seed round.**

### Capital Allocation:
1. **Local High-Throughput Compute Workstation (\$25k - \$30k):**
   - High-core-count AMD Ryzen Threadripper PRO 7000-series workstation with 256GB ECC DDR5 RAM.
   - Required to parallelize exact Inverse Ising likelihood solvers, Markov Chain Monte Carlo (MCMC) sampling, and combinatorial Bivariate Transfer Entropy channel scans.
2. **Physical Wetware Incubation & Perfusion Rig (\$20k):**
   - Microfluidic perfusion chambers and automated environmental controllers to extend organoid viability for multi-day closed-loop reinforcement learning protocols.
3. **Electrophysiology & Stimulation Hardware (\$10k - \$15k):**
   - FinalSpark cloud API access compute credits, high-speed DAC stimulation generators, and low-noise analog signal filters.

*Inquiries from angel investors, neurotechnology funds, and academic collaborators are welcome via email or GitHub issues.*

---

## 📂 Repository Index & Architecture

This monorepo consolidates 10 core research pillars alongside supportive neuro-computational modules:

### 🧠 Primary Organoid Intelligence Research Pillars
```
Biological-AI-Core/
├── wetware_thermodynamics/                                      # Non-equilibrium thermodynamic efficiency, Inverse Ising solvers, & Landauer limits
├── wetware_causal_logic/                                        # Bivariate Transfer Entropy & directional information routing in wetware
├── wetware_scaling_laws/                                        # Power-law avalanche distributions, Zipf's law, & Self-Organized Criticality (SOC)
├── wetware_geometry_chaos/                                      # Attractor dynamics, Lyapunov spectrum, & phase-space reconstruction
├── wetware_reservoir_memory/                                    # Liquid state computing, echo-state memory retention, & biological state readout
├── bio_logic_gates/                                             # Bi-directional electro-stimulation encoding & biological Boolean logic gates
├── Detection-Limit-in-OI/                                       # Microelectrode resolution boundaries & SNR limits for in-vitro intelligence
├── Preliminary-Computational-Analysis-of-Frequency-Dependent...  # Spectral decomposition & multi-scale Lempel-Ziv complexity on MEA channels
├── organoid-intelligence-book/                                  # Comprehensive 14-chapter technical treatise, proofs, & experimental protocols
└── cosmological-bridge/                                         # Formal non-linear dynamic analogies & multi-scale complex systems modeling
```

### 🛠️ Ancillary Infrastructure & Simulation Frameworks
```
Biological-AI-Core/
├── PlannerBot/                                                  # Sovereign Cognitive Engine: offline, on-device AI focus coach & task engine
├── Forge/                                                       # Autonomous task execution & research workflow automation pipeline
├── Flixi/                                                       # High-throughput data processing & sequence generation framework
├── ExoVision/                                                   # Neural activity visualization & computer-vision assisted inspection
└── minecraft-file/                                              # Embodied simulation sandbox for closed-loop behavioral experiments
```

---

## 🛠️ Quickstart

```bash
# Clone the unified master repository
git clone https://github.com/Anikesh0415/Biological-AI-Core.git
cd Biological-AI-Core

# Setup environment
python -m venv venv
.\venv\Scripts\activate   # Windows
# source venv/bin/activate # Linux/macOS

# Install dependencies
pip install numpy scipy numba matplotlib networkx h5py
```

---

## 📜 Citation

```bibtex
@software{BiologicalAICore2026,
  author = {Anikesh Tiwari},
  title = {Biological-AI-Core: Unified Organoid Intelligence & Wetware Computing Architecture},
  year = {2026},
  publisher = {GitHub},
  howpublished = {\url{https://github.com/Anikesh0415/Biological-AI-Core}}
}
```

**Contact:** [GitHub Profile](https://github.com/Anikesh0415) | `babitatiwari7249@gmail.com`
