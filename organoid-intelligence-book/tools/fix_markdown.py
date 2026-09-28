import os
import re
from pathlib import Path

book_dir = Path(r"d:\New research\organoid_intelligence_book")
chapters_dir = book_dir / "chapters"

for md_file in chapters_dir.rglob("*.md"):
    with open(md_file, "r", encoding="utf-8") as f:
        content = f.read()
    
    # 1. Fix images (handle newlines in caption)
    # The image syntax is ![...](filename.png){...}
    # We just want to replace (filename.png) with (../../assets/figures/filename.png)
    # Be careful not to replace already modified ones!
    content = re.sub(r'\]\(([a-zA-Z0-9_]+\.png)\)', r'](../../assets/figures/\1)', content)
    
    # 2. Fix \bm{ to \mathbf{
    content = content.replace(r'\bm{', r'\mathbf{')
    content = content.replace(r'\bm', r'\mathbf')

    # 3. Fix duplicate 'acknowledgments' identifiers
    # Pandoc complains about [WARNING] Duplicate identifier 'acknowledgments'
    # Change {#acknowledgments} to {#acknowledgments-chaptername}
    content = content.replace('{#acknowledgments}', f'{{#acknowledgments-{md_file.stem}}}')

    with open(md_file, "w", encoding="utf-8") as f:
        f.write(content)

print("Markdown files fixed.")
