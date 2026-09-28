import os
import subprocess
from pathlib import Path

base_dir = Path(r"d:\New research")
book_dir = base_dir / "organoid_intelligence_book"
tools_dir = book_dir / "tools"
compile_script = tools_dir / "compile_book.py"

# Add titles to the markdown files manually since pandoc stripped them
titles = {
    "01_causal_logic.md": "Chapter 1: Causal Computation in Wetware: Directed Information Routing via Inhibitory Interneurons in Critical Neural Substrates",
    "02_reservoir_memory.md": "Chapter 2: Reservoir Memory in Wetware: State-Dependent Short-Term Plasticity as a Computational Resource",
    "03_thermodynamics.md": "Chapter 3: Thermodynamics of Wetware: Inverse Ising Inference and Critical Avalanche Dynamics",
    "04_geometry_chaos.md": "Chapter 4: The Geometry of Chaos: Topological Data Analysis of Wetware Attractor Manifolds",
    "05_scaling_laws.md": "Chapter 5: Scaling Laws of Wetware Computation: Renormalization Group Flow in Spiking Neural Substrates"
}

chapter_files = [
    book_dir / "chapters" / "part2_circuits_and_logic" / "01_causal_logic.md",
    book_dir / "chapters" / "part2_circuits_and_logic" / "02_reservoir_memory.md",
    book_dir / "chapters" / "part3_thermodynamics_and_ising" / "03_thermodynamics.md",
    book_dir / "chapters" / "part4_chaos_topology_and_scaling" / "04_geometry_chaos.md",
    book_dir / "chapters" / "part4_chaos_topology_and_scaling" / "05_scaling_laws.md"
]

# We should prepend the title as a level 1 heading (e.g. # Chapter 1: ... )
# and shift the existing headings down by one level (e.g. # Introduction -> ## Introduction)
for filepath in chapter_files:
    if not filepath.exists():
        continue
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Simple markdown heading shift
    # Replace # with ##, ## with ###, etc.
    # We do this by replacing line starts
    new_lines = []
    for line in content.split('\n'):
        if line.startswith('#'):
            new_lines.append('#' + line)
        else:
            new_lines.append(line)
            
    title = titles[filepath.name]
    new_content = f"# {title}\n\n" + '\n'.join(new_lines)
    
    # Fix image references
    import re
    new_content = re.sub(r'!\[(.*?)\]\((.*?\.png)\)', r'![\1](../../assets/figures/\2)', new_content)
    
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(new_content)

print("Headings updated.")

# Also let's create a part1 intro and other part dividers
intro_path = book_dir / "chapters" / "part1_foundations" / "00_introduction.md"
with open(intro_path, 'w', encoding='utf-8') as f:
    f.write("# Introduction to Organoid Intelligence\n\nThis book explores the theoretical and empirical foundations of computing with biological neural substrates...")

# Compile the book
input_files = [str(intro_path)] + [str(f) for f in chapter_files]

print("Compiling PDF...")
subprocess.run([
    "python", str(compile_script),
    *input_files,
    "-o", str(book_dir / "Organoid_Intelligence_Master.pdf"),
    "--format", "pdf"
], check=True)

print("Compiling HTML...")
subprocess.run([
    "python", str(compile_script),
    *input_files,
    "-o", str(book_dir / "Organoid_Intelligence_Master.html"),
    "--format", "html"
], check=True)

print("Master book built successfully.")
