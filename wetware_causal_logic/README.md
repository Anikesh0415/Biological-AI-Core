# Causal Computation in Wetware

**Repository:** `wetware_causal_logic`  
**Author:** Anikesh Tiwari

## Overview
This repository contains the manuscript and generation code for our research on information routing in biological wetware. We simulate a 3D Leaky Integrate-and-Fire (LIF) network structured as a Biological Half-Adder and use bivariate Transfer Entropy to quantify causal information flow.

## Key Discovery
We successfully mapped the logic of a half-adder onto continuous neural dynamics and proved the existence of active, suppressive routing. Crucially, the Transfer Entropy analysis revealed **0.0305 bits of directed causal flow** from the Inhibitory Interneurons to the Sum cluster, providing a mathematical validation of biological XOR logic suppression.

## Repository Contents
- `paper.md`: The raw markdown manuscript.
- `generate_pdf.py`: A Python script to compile the markdown into a publication-ready PDF.
- `manuscript.pdf`: The final compiled academic paper.
- `substrate.py`: Core simulation engine for the 3D spatially embedded biological wetware.
- `gates.py`: Definitions for the biological logic gates.
- `transfer_entropy_analysis.py`: Script to compute causal information flow and generate visual graphs.
- `causal_transfer_entropy_graph.png`: Network flow diagram visualizing directed information routing.

## Acknowledgments
We explicitly credit **FinalSpark** for their foundational contributions to organoid intelligence and wetware computing.
