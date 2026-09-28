import os
import re
from pathlib import Path

book_dir = Path(r"d:\New research\organoid_intelligence_book")
chapters_dir = book_dir / "chapters"

# Mapping from original filenames (as referenced in markdown) to prefixed filenames (in assets)
image_map = {
    "causal_transfer_entropy_graph.png": "fig08_causal_transfer_entropy.png",
    "stp_memory_capacity.png": "fig10_stp_memory_capacity.png",
    "stp_narma10_comparison.png": "fig11_stp_narma10_comparison.png",
    "ising_coupling_matrix.png": "fig13_ising_coupling_matrix.png",
    "data_vs_model_correlations.png": "fig14_data_vs_model_correlations.png",
    "specific_heat_curve.png": "fig15_specific_heat_curve.png",
    "zipfs_law_avalanches.png": "fig16_zipfs_law_avalanches.png",
    "persistence_diagram.png": "fig17_persistence_diagram.png",
    "dfa_fluctuation_plot.png": "fig18_dfa_fluctuation_plot.png",
    "rg_flow_scale_invariance.png": "fig19_rg_flow_scale_invariance.png",
}

for md_file in chapters_dir.rglob("*.md"):
    with open(md_file, "r", encoding="utf-8") as f:
        content = f.read()
    
    modified = False
    for old_name, new_name in image_map.items():
        # The fix_markdown.py script previously changed the paths to:
        # ](../../assets/figures/old_name)
        # So we just need to replace the old_name with new_name if it appears in the path.
        if old_name in content:
            content = content.replace(f"../../assets/figures/{old_name}", f"../../assets/figures/{new_name}")
            # Just in case it wasn't fixed by the previous script
            content = content.replace(f"]({old_name})", f"](../../assets/figures/{new_name})")
            modified = True
            
    if modified:
        with open(md_file, "w", encoding="utf-8") as f:
            f.write(content)
        print(f"Updated images in {md_file.name}")

print("Image paths verified and fixed.")
