import sys
import subprocess
import os

def install(package):
    subprocess.check_call([sys.executable, "-m", "pip", "install", package])

try:
    from markdown_pdf import MarkdownPdf, Section
except ImportError:
    print("Installing markdown-pdf...")
    install("markdown-pdf")
    from markdown_pdf import MarkdownPdf, Section

def main():
    print("Generating PDF from paper.md...")
    pdf = MarkdownPdf(toc_level=0)
    
    with open("paper.md", "r", encoding="utf-8") as f:
        content = f.read()
        
    pdf.add_section(Section(content))
    pdf.save("manuscript.pdf")
    print("PDF generated successfully: manuscript.pdf")

if __name__ == "__main__":
    main()
