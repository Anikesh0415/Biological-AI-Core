# Organoid Intelligence: From Foundations to Frontier Wetware Computing

[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.placeholder.svg)](https://zenodo.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Pages: 82](https://img.shields.io/badge/Beginner's%20Guide-82%20Pages-blue.svg)](#beginners-guide)
[![Master: 43](https://img.shields.io/badge/Technical%20Volume-43%20Pages-green.svg)](#master-technical-textbook)
[![Python: 3.10+](https://img.shields.io/badge/Python-3.10+-brightgreen.svg)](https://python.org)

**Author:** Anikesh Tiwari  
**Field:** Organoid Intelligence (OI), Wetware Computing, Computational Neuroscience, Statistical Mechanics

---

## 📖 Executive Overview

**Organoid Intelligence (OI)** represents a radical paradigm shift away from silicon-based von Neumann architectures toward biological neural computing. While artificial neural networks running on modern GPUs consume megawatts of electrical power, biological brains perform creative reasoning, multimodal perception, and lifelong learning on roughly **20 watts**.

This repository is a comprehensive, publication-ready compendium containing:
1. **The Beginner's Guide (82 Pages):** An intuitive, human-language textbook designed for students, programmers, and engineers with **zero background in biology or physics**, using vivid real-world analogies (traffic cops, rippling ponds, Swiss cheese, and zoom lenses).
2. **The Master Technical Volume (43 Pages):** A rigorous mathematical and physical treatise formalizing Causal Logic, Reservoir Computing, Inverse Ising Thermodynamics, Topological Data Analysis, and Renormalization Group Scaling Laws.
3. **The Empirical Frontiers:** Four verified experimental breakthroughs executed on live 30kHz multi-electrode array (MEA) data from the **FinalSpark** Neuroplatform (`fs369` and `fs437`).
4. **Complete Source Code & Datasets:** 100% reproducible Python scripts, inferred Hamiltonian coupling matrices ($J_{ij}$), and 24 high-resolution publication figures.

---

## 📚 Deliverables & PDF Downloads

| Edition | Format | Page Count | Figures | Description |
| :--- | :---: | :---: | :---: | :--- |
| **Beginner's Guide** | [PDF](Organoid_Intelligence_Beginners_Guide.pdf) \| [HTML](Organoid_Intelligence_Beginners_Guide.html) | **82 pages** | 14 | Fully accessible to non-biologists; includes Preface, 10 Chapters, and 3 Appendices (Glossary, Hands-on Lab, Math Bridge). |
| **Master Volume** | [PDF](Organoid_Intelligence_Master.pdf) \| [HTML](Organoid_Intelligence_Master.html) | **43 pages** | 14 | Rigorous peer-reviewed formulation unifying all 5 theoretical papers and the 4 FinalSpark empirical discoveries. |

---

## 🔬 Core Theoretical & Empirical Breakthroughs

### 1. Causal Logic in Wetware (Biological Half-Adder)
- **Concept:** Proves that living neural substrates can execute directed Boolean logic without silicon gates.
- **Key Finding:** Inhibitory interneurons act as active, causal logic operators, transferring **0.0305 bits** of directed suppressive flow ($TE_{I \to \text{Sum}}$) to enforce XOR functionality.
- **Reference:** [`research_papers/wetware_causal_logic/`](research_papers/wetware_causal_logic/)

### 2. Reservoir Memory & Short-Term Plasticity (STP)
- **Concept:** Translates reservoir computing to living wetware, demonstrating that synaptic depression and facilitation act as short-term memory (RAM).
- **Key Finding:** Uncovers the fundamental *Memory-Nonlinearity Trade-off*—STP substantially expands temporal memory capacity but introduces dynamic jitter that limits high-order algebraic transformations.
- **Reference:** [`research_papers/wetware_reservoir_memory/`](research_papers/wetware_reservoir_memory/)

### 3. Thermodynamics & Inverse Ising Mechanics
- **Concept:** Employs maximum entropy Inverse Ising inference to reconstruct the effective synaptic coupling matrix $J_{ij}$ from observed neural avalanches.
- **Key Finding:** The specific heat curve exhibits a pronounced peak at $T_c \approx 1.0$, and avalanche sizes follow Zipf's law ($\tau = 1.954$), proving that healthy wetware naturally poises itself at thermodynamic criticality.
- **Reference:** [`research_papers/wetware_thermodynamics/`](research_papers/wetware_thermodynamics/)

### 4. The Geometry of Chaos (Topological Data Analysis)
- **Concept:** Applies Takens' delay embedding and Vietoris-Rips filtration to map the high-dimensional attractor manifold of neural firing.
- **Key Finding:** Reveals robust, non-trivial 1-dimensional persistent homology loops ($\beta_1$), proving that chaotic neural chatter maintains stable, closed orbits (attractor memories).
- **Reference:** [`research_papers/wetware_geometry_chaos/`](research_papers/wetware_geometry_chaos/)

### 5. Renormalization Group (RG) Scaling Laws
- **Concept:** Uses Kadanoff spatial coarse-graining to test whether neural scaling properties remain invariant as organoids grow in size.
- **Key Finding:** The avalanche size distributions demonstrate scale-invariance with an RG fixed point ($\Delta \tau = 0.0417$), confirming that biological neural circuits can scale without cognitive degradation.
- **Reference:** [`research_papers/wetware_scaling_laws/`](research_papers/wetware_scaling_laws/)

### 6. Real Empirical FinalSpark Discoveries (`fs369` & `fs437`)
1. **Ising Perturbation via Electrical Stimulation:** Verified that targeted biphasic stimulation pulses induce measurable topological deformations in the empirical $J_{ij}$ matrix, proving synaptic plasticity in live tissue.
2. **Environmental Drift & State Space Collapse:** Discovered that an abrupt drop in incubator $\text{CO}_2$ to $0.69\%$ causes catastrophic collapse of the topological Betti loops ($\beta_1 \to 0$), which spontaneously recover once homeostasis is restored.
3. **Biological Aging Clock:** Mapped the trajectory of RG flow exponents across the organoid's lifespan, demonstrating critical convergence during peak adulthood (Day 3) and dramatic divergence during senescence (Day 5).
4. **Spontaneous Logic Gate Extraction:** Isolated directed Transfer Entropy pathways from continuous 30kHz recordings, proving that living networks spontaneously self-organize into stable Boolean routing motifs.
- **Reference:** [`chapters/part5_empirical_frontiers/`](chapters/part5_empirical_frontiers/)

---

## 📂 Repository Structure

```text
organoid-intelligence-book/
├── Organoid_Intelligence_Beginners_Guide.pdf   # 82-page human-language textbook
├── Organoid_Intelligence_Beginners_Guide.html  # Responsive web edition with MathJax
├── Organoid_Intelligence_Master.pdf            # 43-page master technical volume
├── Organoid_Intelligence_Master.html           # Technical web edition
├── beginners_guide_chapters/                   # Markdown sources for Beginner's Guide
│   ├── 00_preface.md                           # Conceptual roadmap & overview
│   ├── chapter_01.md - chapter_10.md           # Chapters 1 to 10
│   ├── appendix_a_glossary.md                  # Plain-language glossary (45+ terms)
│   ├── appendix_b_reproducibility.md           # Hands-on Python laboratory
│   └── appendix_c_math_bridge.md               # Term-by-term equation breakdown
├── chapters/                                   # Markdown sources for Master Volume
│   ├── part1_foundations/
│   ├── part2_circuits_and_logic/
│   ├── part3_thermodynamics_and_ising/
│   ├── part4_chaos_topology_and_scaling/
│   └── part5_empirical_frontiers/
├── research_papers/                            # Original standalone paper manuscripts & code
│   ├── wetware_causal_logic/
│   ├── wetware_reservoir_memory/
│   ├── wetware_thermodynamics/
│   ├── wetware_geometry_chaos/
│   ├── wetware_scaling_laws/
│   └── bio_logic_gates/
├── assets/figures/                             # 24 publication & empirical figures
├── data/arrays/                                # Inferred physics arrays (J_inferred, h_inferred, etc.)
├── tools/                                      # Full toolchain and execution scripts
│   ├── compile_book.py                         # Multi-format compiler wrapper
│   ├── compile_beginners_guide.py              # Beginner guide compiler
│   ├── execute_research_1.py                   # Empirical Ising perturbation runner
│   ├── execute_research_2.py                   # Empirical TDA collapse runner
│   ├── execute_research_3.py                   # Empirical RG flow runner
│   ├── execute_research_4.py                   # Empirical Transfer Entropy logic runner
│   └── verify_every_detail.py                  # Full automated audit suite
├── CITATION.cff                                # Machine-readable citation file
├── .zenodo.json                                # Zenodo integration metadata
├── LICENSE                                     # MIT License
└── README.md                                   # This documentation
```

---

## 💻 Quickstart: Reproducing the Code

### Prerequisites
- Python 3.10+
- Dependencies: `pip install numpy scipy pandas tables fastparquet matplotlib networkx scikit-learn ripser persim pypdf`

### Running the Full Audit Suite
To independently verify all pages, text densities, figures, and deliverables:
```bash
python tools/verify_every_detail.py
```

### Re-compiling the Books
```bash
# Compile the 82-page Beginner's Guide
python tools/compile_beginners_guide.py

# Compile the 43-page Master Volume
python tools/build_master.py
```

---

## 📜 How to Cite

If you use this book, the underlying theories, or the empirical analysis scripts in your academic research, please cite:

```bibtex
@book{tiwari2026organoid,
  author    = {Tiwari, Anikesh},
  title     = {Organoid Intelligence: From Foundations to Frontier Wetware Computing},
  year      = {2026},
  publisher = {Zenodo},
  url       = {https://github.com/Anikesh0415/organoid-intelligence-book},
  doi       = {10.5281/zenodo.placeholder}
}
```

---

## ⚖️ License
This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
All figures, text, and data arrays are freely available for scientific research, education, and open-access reproduction.
