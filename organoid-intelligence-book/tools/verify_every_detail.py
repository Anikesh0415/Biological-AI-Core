import sys
from pathlib import Path
import pypdf

base_dir = Path(r"d:\New research")
book_dir = base_dir / "organoid_intelligence_book"
figures_dir = book_dir / "assets" / "figures"
beginners_dir = book_dir / "beginners_guide_chapters"
master_chapters_dir = book_dir / "chapters"

print("=" * 80)
print("COMPREHENSIVE MULTI-TIER SYSTEM VERIFICATION REPORT")
print("=" * 80)

# 1. Audit Beginner's Guide PDF
bg_pdf_path = book_dir / "Organoid_Intelligence_Beginners_Guide.pdf"
assert bg_pdf_path.exists(), "Beginner's guide PDF does not exist!"

reader_bg = pypdf.PdfReader(str(bg_pdf_path))
total_bg_pages = len(reader_bg.pages)
total_bg_images = sum(len(p.images) for p in reader_bg.pages)

print(f"\n[1] AUDIT: Organoid_Intelligence_Beginners_Guide.pdf")
print(f"    - File Size: {bg_pdf_path.stat().st_size:,} bytes")
print(f"    - Page Count: {total_bg_pages} pages (Requirement: >= 40)")
print(f"    - Embedded Images: {total_bg_images} images")
assert total_bg_pages >= 40, f"FAILED: Expected >= 40 pages, got {total_bg_pages}"
assert total_bg_images >= 10, f"FAILED: Expected embedded images, got {total_bg_images}"

# Page-by-page verification
print("\n    --- Page-by-Page Audit of Beginner's Guide ---")
blank_pages = []
pages_with_images = []
for idx, page in enumerate(reader_bg.pages):
    text = page.extract_text() or ""
    words = len(text.split())
    img_count = len(page.images)
    if img_count > 0:
        pages_with_images.append((idx + 1, img_count))
    if words == 0 and img_count == 0:
        blank_pages.append(idx + 1)

print(f"    - Total Pages Audited: {total_bg_pages}")
print(f"    - Pages containing figures: {len(pages_with_images)} pages -> {pages_with_images}")
print(f"    - Completely blank pages: {len(blank_pages)}")
assert len(blank_pages) == 0, f"Found blank pages: {blank_pages}"
print("    -> PAGE-BY-PAGE AUDIT: 100% VERIFIED AND HEALTHY!")

# 2. Audit Master Technical PDF
master_pdf_path = book_dir / "Organoid_Intelligence_Master.pdf"
assert master_pdf_path.exists(), "Master PDF does not exist!"

reader_master = pypdf.PdfReader(str(master_pdf_path))
total_m_pages = len(reader_master.pages)
total_m_images = sum(len(p.images) for p in reader_master.pages)

print(f"\n[2] AUDIT: Organoid_Intelligence_Master.pdf")
print(f"    - File Size: {master_pdf_path.stat().st_size:,} bytes")
print(f"    - Page Count: {total_m_pages} pages")
print(f"    - Embedded Images: {total_m_images} images")
assert total_m_pages >= 30, f"Master PDF too short: {total_m_pages}"
assert total_m_images >= 10, f"Master PDF missing images: {total_m_images}"
print("    -> MASTER PDF AUDIT: 100% VERIFIED!")

# 3. Audit HTML Deliverables
bg_html = book_dir / "Organoid_Intelligence_Beginners_Guide.html"
m_html = book_dir / "Organoid_Intelligence_Master.html"

print(f"\n[3] AUDIT: HTML Editions")
print(f"    - Beginner's Guide HTML: {bg_html.stat().st_size:,} bytes")
print(f"    - Master Textbook HTML:  {m_html.stat().st_size:,} bytes")
assert bg_html.exists() and bg_html.stat().st_size > 50000
assert m_html.exists() and m_html.stat().st_size > 50000
print("    -> HTML DELIVERABLES: 100% VERIFIED!")

# 4. Audit Source Markdown Content & Chapter Word Counts
print(f"\n[4] AUDIT: Beginner's Guide Source Chapters")
total_bg_words = 0
for md in sorted(beginners_dir.glob("*.md")):
    text = md.read_text(encoding="utf-8")
    words = len(text.split())
    total_bg_words += words
    print(f"    - {md.name:30s}: {words:5d} words")
print(f"    --------------------------------------------------")
print(f"    TOTAL WORDS IN BEGINNER'S GUIDE: {total_bg_words:,} words")
assert total_bg_words > 20000, "Word count lower than expected!"
print("    -> CHAPTER WORD COUNTS: 100% VERIFIED!")

# 5. Audit Real Empirical Figures
print(f"\n[5] AUDIT: Visual Assets and Figures")
expected_figures = [
    "fig01_criticality.png", "fig02_logic_performance.png", "fig03_spatiotemporal.png",
    "fig04_training_comparison.png", "fig05_AND_weights.png", "fig05_OR_weights.png",
    "fig06_half_adder.png", "fig07_sr_latch.png", "fig08_causal_transfer_entropy.png",
    "fig09_memory_capacity_curve.png", "fig10_stp_memory_capacity.png",
    "fig11_stp_narma10_comparison.png", "fig12_topology_scaling.png",
    "fig13_ising_coupling_matrix.png", "fig14_data_vs_model_correlations.png",
    "fig15_specific_heat_curve.png", "fig16_zipfs_law_avalanches.png",
    "fig17_persistence_diagram.png", "fig18_dfa_fluctuation_plot.png",
    "fig19_rg_flow_scale_invariance.png", "empirical_ising_perturbation.png",
    "empirical_tda_collapse.png", "empirical_rg_flow.png", "empirical_logic_gates.png"
]
missing_figs = []
for fig in expected_figures:
    p = figures_dir / fig
    if not p.exists():
        missing_figs.append(fig)
print(f"    - Expected Figures: {len(expected_figures)}")
print(f"    - Missing Figures:  {len(missing_figs)}")
assert len(missing_figs) == 0, f"Missing figures: {missing_figs}"
print("    -> ASSET FIGURES: 100% VERIFIED (All 24 publication & empirical figures present)!")

print("\n" + "=" * 80)
print("ALL VERIFICATION CHECKS PASSED WITH ZERO ERRORS!")
print("=" * 80)
