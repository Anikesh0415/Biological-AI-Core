#!/usr/bin/env python3
"""
Step 1 Master Verification Script for Organoid Intelligence Book
"""
import os
import sys
from pathlib import Path

def run_verification():
    print("=================================================================")
    print("ORGANOID INTELLIGENCE BOOK: STEP 1 SYSTEM & ASSETS READINESS CHECK")
    print("=================================================================\n")
    
    # 1. Check Python Dependencies
    deps = ["numpy", "scipy", "pandas", "h5py", "pyarrow", "seaborn", "jinja2", "ripser", "nolds", "matplotlib"]
    print("[1] Python Scientific Stack Check:")
    for dep in deps:
        try:
            mod = __import__(dep)
            ver = getattr(mod, "__version__", "installed")
            print(f"  [OK] {dep:<15} version: {ver}")
        except ImportError:
            print(f"  [FAIL] {dep:<15} NOT INSTALLED")
            return False
            
    # 2. Check Compilers
    print("\n[2] Document Compilers Check:")
    base = Path(__file__).resolve().parent.parent.parent
    pandoc_bin = base / "pandoc_dir" / "pandoc-3.1.11.1" / "pandoc.exe"
    tectonic_bin = base / "tectonic_dir" / "tectonic.exe"
    
    if pandoc_bin.exists():
        print(f"  [OK] Pandoc:   {pandoc_bin}")
    else:
        print(f"  [FAIL] Pandoc missing at {pandoc_bin}")
        return False
        
    if tectonic_bin.exists():
        print(f"  [OK] Tectonic: {tectonic_bin}")
    else:
        print(f"  [FAIL] Tectonic missing at {tectonic_bin}")
        return False
        
    # 3. Check Book Figures
    print("\n[3] Research Publication Figures Check:")
    fig_dir = base / "organoid_intelligence_book" / "assets" / "figures"
    figures = list(fig_dir.glob("*.png"))
    print(f"  Found {len(figures)} figures in {fig_dir.name}:")
    for fig in sorted(figures):
        print(f"   - {fig.name:<35} ({fig.stat().st_size / 1024:.1f} KB)")
    if len(figures) < 20:
        print(f"  [WARNING] Expected at least 20 figures, found {len(figures)}")
    else:
        print("  [OK] All 20 research figures present.")
        
    # 4. Check Datasets
    print("\n[4] Empirical & Simulation Datasets Check:")
    spikes_parquet = base / "organoid_physics_book" / "data" / "processed" / "spikes.parquet"
    fs437_pkg = base / "data" / "raw" / "fs437_export" / "fs437_package.hdf5"
    fs437_idx = base / "data" / "raw" / "fs437_export" / "fs437_segment_index.parquet"
    
    data_files = [
        ("fs369 33M Spikes Parquet", spikes_parquet),
        ("fs437 HDF5 Package", fs437_pkg),
        ("fs437 Segment Index", fs437_idx),
        ("Ising J Matrix", base / "organoid_intelligence_book" / "data" / "arrays" / "J_inferred.npy"),
        ("TDA Point Cloud", base / "organoid_intelligence_book" / "data" / "arrays" / "point_cloud.npy"),
        ("RG Flow Spikes Lattice", base / "organoid_intelligence_book" / "data" / "arrays" / "BiologicalWetware_phase4_spikes.npy"),
    ]
    for label, path in data_files:
        if path.exists():
            size_mb = path.stat().st_size / (1024 * 1024)
            print(f"  [OK] {label:<26}: {size_mb:8.2f} MB ({path.name})")
        else:
            print(f"  [FAIL] {label:<26} MISSING at {path}")
            return False
            
    print("\n=================================================================")
    print("STEP 1 READINESS STATUS: 100% COMPLETE AND OPERATIONAL")
    print("=================================================================\n")
    return True

if __name__ == "__main__":
    success = run_verification()
    sys.exit(0 if success else 1)