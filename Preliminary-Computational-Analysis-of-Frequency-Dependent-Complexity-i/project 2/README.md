# Algorithmic Complexity & Topological Mapping in Synthetic Neural Organoids

![Python](https://img.shields.io/badge/Python-3.8%2B-blue)
![SciPy](https://img.shields.io/badge/SciPy-Signal_Processing-lightgrey)
![NetworkX](https://img.shields.io/badge/NetworkX-Topological_Analysis-green)

## 📌 Overview
This repository contains the computational pipeline used to evaluate the structural and temporal signal dynamics of a synthetic *in vitro* human brain organoid. The project analyzes a proprietary dataset (94 million temporal spike events) provided by FinalSpark to investigate whether biological neural networks exhibit frequency-dependent information processing under electrical stimulation.

**Read the full preliminary analysis paper:** [Link to your PDF / Vercel Dashboard here]

## 🔬 The Computational Pipeline

This project is divided into two core mathematical engines:

### 1. Structural Topological Isomorphism (`dashboard/`)
* **Function:** Analyzes the spatial architecture of the neural network.
* **Math:** Translates temporal spike events into a directed graph $G = (V, E)$. Calculates node centrality using the **PageRank algorithm** and measures clustering coefficients to verify the network's scale-free topology.

### 2. Digital Signal Processing (DSP) & Complexity Engine (`scripts/`)
* **Function:** Evaluates temporal signal modulation across specific biological frequency bands.
* **Math:** Applies a zero-phase 4th-order Butterworth bandpass filter to isolate Theta (4-8 Hz), Alpha (8-12 Hz), Beta (12-30 Hz), and Gamma (30-100 Hz) bands.
* **Information Theory:** Binarizes the filtered signals via multi-resolution coarse-graining (10ms, 25ms, 50ms) and measures algorithmic information density using **Lempel-Ziv Complexity (LZ76)** and **Approximate Entropy (ApEn)**. Statistical significance is verified via Wilcoxon signed-rank tests.

## ⚠️ Scientific Limitations & Transparency
This codebase was developed as an exploratory data-mining exercise. As detailed in our paper, the findings identify high-frequency ($\Delta \text{LZC} = +24.66\%$) entropy spikes during stimulation. However, this repository explicitly acknowledges the following limitations:
* **No Physical Controls:** The dataset lacks dead-tissue comparisons or non-stimulated baselines, meaning thermal/electrode artifacts cannot be entirely ruled out.
* **Black Box Spike Sorting:** Raw continuous voltage traces were pre-processed by the vendor; the artifact rejection methodology is undefined.
* **Timescale Anomaly:** A complexity inversion observed at the 50ms scale suggests the Gamma spikes may be transient physical bursts rather than sustained computation.

This code is open-sourced to ensure absolute methodological transparency. Rigorous physical experiments with independent biological replicates are required to validate these computational observations.

## 🚀 Usage
To run the DSP and Complexity pipeline locally:

1. Install dependencies:
   ```bash
   pip install numpy scipy pandas networkx matplotlib seaborn
   ```

2. Execute the complexity analysis:
   ```bash
   python complexity_analysis.py
   ```

3. Generate the significance visualizations:
   ```bash
   python band_visualizer.py
   ```

## 🤝 Acknowledgments
Data logs (FS369/FS437) utilized for this computational analysis were provided by Neurospark / FinalSpark.
