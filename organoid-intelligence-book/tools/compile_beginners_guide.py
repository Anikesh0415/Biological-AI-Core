import os
import sys
import subprocess
from pathlib import Path
import pypdf

base_dir = Path(r"d:\New research")
book_dir = base_dir / "organoid_intelligence_book"
chapters_dir = book_dir / "beginners_guide_chapters"
tools_dir = book_dir / "tools"
sys.path.append(str(tools_dir))
import compile_book

ordered_files = [
    chapters_dir / "00_preface.md",
    chapters_dir / "chapter_01.md",
    chapters_dir / "chapter_02.md",
    chapters_dir / "chapter_03.md",
    chapters_dir / "chapter_04.md",
    chapters_dir / "chapter_05.md",
    chapters_dir / "chapter_06.md",
    chapters_dir / "chapter_07.md",
    chapters_dir / "chapter_08.md",
    chapters_dir / "chapter_09.md",
    chapters_dir / "chapter_10.md",
    chapters_dir / "appendix_a_glossary.md",
    chapters_dir / "appendix_b_reproducibility.md",
    chapters_dir / "appendix_c_math_bridge.md",
]

for f in ordered_files:
    if not f.exists():
        raise FileNotFoundError(f"Missing required chapter file: {f}")

out_pdf = book_dir / "Organoid_Intelligence_Beginners_Guide.pdf"
out_html = book_dir / "Organoid_Intelligence_Beginners_Guide.html"

title = "Organoid Intelligence: A Beginner's Guide to Computing with Living Brains"
author = "Anikesh Tiwari"

print("Compiling Beginner's Guide PDF...")
# Use 1.1in margin, 11pt font, 1.18 linestretch for clean reading and 50+ page length
compile_book.compile_pdf(
    ordered_files,
    out_pdf,
    title=title,
    author=author,
    margin="1.1in",
    fontsize="11pt",
    linestretch="1.18"
)

print("Compiling Beginner's Guide HTML...")
compile_book.compile_html(
    ordered_files,
    out_html,
    title=title,
    author=author
)

if out_pdf.exists():
    reader = pypdf.PdfReader(str(out_pdf))
    total_pages = len(reader.pages)
    total_imgs = sum(len(p.images) for p in reader.pages)
    print("=" * 60)
    print(f"SUCCESS: {out_pdf.name}")
    print(f"Total Pages: {total_pages} (Requirement: >= 40)")
    print(f"Total Embedded Images: {total_imgs}")
    print("=" * 60)
    if total_pages < 40:
        print("WARNING: Book has fewer than 40 pages!")
        sys.exit(1)
    else:
        print("VERIFICATION PASSED: Book meets and exceeds the 40-page requirement!")
