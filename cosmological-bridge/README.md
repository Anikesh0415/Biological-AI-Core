# The Cosmological Bridge: Structural Isomorphism

![React](https://img.shields.io/badge/react-%2320232a.svg?style=for-the-badge&logo=react&logoColor=%2361DAFB) ![Vite](https://img.shields.io/badge/vite-%23646CFF.svg?style=for-the-badge&logo=vite&logoColor=white) ![Python](https://img.shields.io/badge/python-3670A0?style=for-the-badge&logo=python&logoColor=ffdd54)

## Abstract
**The Cosmological Bridge** is an advanced scientific visualization and computational network analysis tool designed to explore the structural isomorphism between Biological Neural Networks and the Cosmic Web. By utilizing high-resolution Multielectrode Array (MEA) data of living brain organoids and large-scale galaxy distribution data, this project computationally maps, compares, and visualizes the profound similarities in their physical topologies. 

## The Math & Topology
Rigorous computational analysis reveals striking structural parallels between these two vastly different physical scales:
- **Clustering Coefficient**: Both networks exhibit a remarkably similar clustering coefficient ($\approx 0.78$), indicating highly dense, interconnected local hubs.
- **Scale-Free Power-Law Topology**: The degree distribution, $P(k)$, for both systems adheres to a scale-free power-law curve, mathematically demonstrating that the microscopic brain and the macroscopic universe self-organize using similar network growth principles.

## Tech Stack
- **Frontend & Visualization**: React, Vite, Tailwind CSS, `react-force-graph-3d`
- **Data Engineering & Analysis**: Python, NetworkX, Astropy

## Local Setup
To run the 3D Isomorphism Dashboard locally on your machine:

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd Neuro-pro
   ```
2. **Navigate to the dashboard directory:**
   ```bash
   cd dashboard
   ```
3. **Install dependencies:**
   ```bash
   npm install
   ```
4. **Run the development server:**
   ```bash
   npm run dev
   ```
5. Open your browser and navigate to `http://localhost:5173`.

## Acknowledgments
- **Neurobiological Data**: The raw MEA datasets (strictly confidential) utilized to calculate the neural graph metrics were generously provided by **FinalSpark** ([finalspark.com](https://finalspark.com/)).
- **Cosmological Data**: Galaxy coordinates and redshift data were acquired via the **Sloan Digital Sky Survey (SDSS)**.
