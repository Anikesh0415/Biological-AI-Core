import os
import subprocess
from pathlib import Path
import shutil

base_dir = Path(r"d:\New research")
book_dir = base_dir / "organoid_intelligence_book"
pandoc_exe = base_dir / "pandoc_dir" / "pandoc-3.1.11.1" / "pandoc.exe"

papers = [
    ("wetware_causal_logic", "part2_circuits_and_logic", "01_causal_logic.md"),
    ("wetware_reservoir_memory", "part2_circuits_and_logic", "02_reservoir_memory.md"),
    ("wetware_thermodynamics", "part3_thermodynamics_and_ising", "03_thermodynamics.md"),
    ("wetware_geometry_chaos", "part4_chaos_topology_and_scaling", "04_geometry_chaos.md"),
    ("wetware_scaling_laws", "part4_chaos_topology_and_scaling", "05_scaling_laws.md")
]

for paper_dir, chapter_dir, md_file in papers:
    tex_path = base_dir / paper_dir / "manuscript.tex"
    out_path = book_dir / "chapters" / chapter_dir / md_file
    
    if tex_path.exists():
        print(f"Converting {tex_path.name} from {paper_dir}...")
        cmd = [str(pandoc_exe), str(tex_path), "-f", "latex", "-t", "markdown", "-o", str(out_path)]
        subprocess.run(cmd, check=True)
    else:
        print(f"File not found: {tex_path}")

print("Done converting papers.")
