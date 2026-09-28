#!/usr/bin/env python3
"""
Organoid Intelligence Textbook Compiler
Wraps local Pandoc and Tectonic binaries for multi-format book compilation.
Author: Anikesh Tiwari
"""

import os
import sys
import subprocess
import argparse
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent.parent
PANDOC_BIN = BASE_DIR / "pandoc_dir" / "pandoc-3.1.11.1" / "pandoc.exe"
TECTONIC_BIN = BASE_DIR / "tectonic_dir" / "tectonic.exe"

def check_binaries():
    if not PANDOC_BIN.exists():
        raise FileNotFoundError(f"Pandoc binary not found at: {PANDOC_BIN}")
    if not TECTONIC_BIN.exists():
        raise FileNotFoundError(f"Tectonic binary not found at: {TECTONIC_BIN}")
    return True

def compile_pdf(input_files, output_pdf, title="Organoid Intelligence: From Foundations to Frontier Wetware Computing", author="Anikesh Tiwari", resource_paths=None, margin="1in", fontsize="11pt", linestretch="1.15"):
    check_binaries()
    output_pdf = Path(output_pdf).resolve()
    
    figures_dir = BASE_DIR / "organoid_intelligence_book" / "assets" / "figures"
    book_dir = BASE_DIR / "organoid_intelligence_book"
    
    r_paths = [str(figures_dir), str(book_dir), str(BASE_DIR)]
    if resource_paths:
        if isinstance(resource_paths, list):
            r_paths.extend(resource_paths)
        else:
            r_paths.append(str(resource_paths))
    resource_path_str = ";".join(r_paths)
    
    cmd = [
        str(PANDOC_BIN),
        "-s",
        f"--pdf-engine={TECTONIC_BIN}",
        "-f", "markdown",
        f"--resource-path={resource_path_str}",
        "-V", f"title={title}",
        "-V", f"author={author}",
        "-V", f"geometry:margin={margin}",
        "-V", f"fontsize={fontsize}",
        "-V", f"linestretch={linestretch}",
        "-V", "documentclass=report",
        "-V", "toc=true",
        "-V", "toc-depth=2",
        "-V", "numbersections=true",
        "-V", "colorlinks=true",
        "-V", "linkcolor=blue",
        "-V", "urlcolor=blue",
        "-V", "citecolor=blue",
        "-V", r"header-includes=\usepackage{bm}\usepackage{graphicx}",
        "-o", str(output_pdf),
    ]
    for f in input_files:
        cmd.append(str(Path(f).resolve()))
        
    print(f"Compiling PDF -> {output_pdf} ...")
    res = subprocess.run(cmd, capture_output=True, text=True)
    if res.returncode != 0:
        print("Pandoc/Tectonic Error:")
        print(res.stderr)
        return False
    print(f"Successfully generated: {output_pdf}")
    return True

def compile_html(input_files, output_html, title="Organoid Intelligence: From Foundations to Frontier Wetware Computing", author="Anikesh Tiwari", resource_paths=None):
    check_binaries()
    output_html = Path(output_html).resolve()
    
    figures_dir = BASE_DIR / "organoid_intelligence_book" / "assets" / "figures"
    book_dir = BASE_DIR / "organoid_intelligence_book"
    r_paths = [str(figures_dir), str(book_dir), str(BASE_DIR)]
    if resource_paths:
        if isinstance(resource_paths, list):
            r_paths.extend(resource_paths)
        else:
            r_paths.append(str(resource_paths))
    resource_path_str = ";".join(r_paths)
    
    cmd = [
        str(PANDOC_BIN),
        "-s",
        "--mathjax",
        "-f", "markdown",
        f"--resource-path={resource_path_str}",
        "-V", f"title={title}",
        "-V", f"author={author}",
        "--toc",
        "--toc-depth=3",
        "-o", str(output_html),
    ]
    for f in input_files:
        cmd.append(str(Path(f).resolve()))
        
    print(f"Compiling HTML -> {output_html} ...")
    res = subprocess.run(cmd, capture_output=True, text=True)
    if res.returncode != 0:
        print("Pandoc Error:")
        print(res.stderr)
        return False
    print(f"Successfully generated: {output_html}")
    return True

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Compile Organoid Intelligence Textbook")
    parser.add_argument("inputs", nargs="+", help="Input markdown files")
    parser.add_argument("-o", "--output", required=True, help="Output file path (.pdf or .html)")
    parser.add_argument("-t", "--title", default="Organoid Intelligence: From Foundations to Frontier Wetware Computing", help="Book title")
    parser.add_argument("-a", "--author", default="Anikesh Tiwari", help="Author name")
    parser.add_argument("--format", choices=["pdf", "html"], help="Output format")
    
    args = parser.parse_args()
    out_ext = Path(args.output).suffix.lower()
    fmt = args.format or ("pdf" if out_ext == ".pdf" else "html" if out_ext == ".html" else "pdf")
    
    if fmt == "pdf":
        compile_pdf(args.inputs, args.output, title=args.title, author=args.author)
    elif fmt == "html":
        compile_html(args.inputs, args.output, title=args.title, author=args.author)