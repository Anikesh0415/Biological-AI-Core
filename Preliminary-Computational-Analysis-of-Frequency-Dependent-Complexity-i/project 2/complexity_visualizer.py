import re
import matplotlib.pyplot as plt
import seaborn as sns
import numpy as np

def get_asterisks(p_val):
    if p_val < 0.001: return '***'
    elif p_val < 0.01: return '**'
    elif p_val < 0.05: return '*'
    else: return 'ns'

def parse_report(filename):
    bins = []
    base_lzc, base_apen = [], []
    stim_lzc, stim_apen = [], []
    pval_lzc, pval_apen = [], []
    
    current_bin = None
    state = 0 
    
    with open(filename, 'r', encoding='utf-8') as f:
        for line in f:
            line = line.strip()
            bin_match = re.search(r"--- Bin Size:\s+([\d\.]+)\s+ms", line)
            if bin_match:
                current_bin = float(bin_match.group(1))
                bins.append(current_bin)
            
            if "State 1:" in line: state = 1
            elif "State 2:" in line: state = 2
                
            if "Mean Normalized LZC:" in line:
                val = float(re.search(r"[\d\.]+", line.split(":")[1]).group(0))
                if state == 1: base_lzc.append(val)
                elif state == 2: stim_lzc.append(val)
                    
            if "Mean Approx Entropy:" in line:
                val = float(re.search(r"[\d\.]+", line.split(":")[1]).group(0))
                if state == 1: base_apen.append(val)
                elif state == 2: stim_apen.append(val)

            if "- p-value LZC:" in line:
                val = float(line.split(":")[1].strip())
                pval_lzc.append(val)
                
            if "- p-value ApEn:" in line:
                val = float(line.split(":")[1].strip())
                pval_apen.append(val)
                    
    return bins, base_lzc, base_apen, stim_lzc, stim_apen, pval_lzc, pval_apen

def annotate_significance(ax, x, y1, y2, p_val):
    ast = get_asterisks(p_val)
    if ast == 'ns': return
    
    y_max = max(y1, y2)
    y_min = min(y1, y2)
    
    # Vertical line connecting the two points
    ax.plot([x, x], [y_min, y_max], color='#ffffff', linewidth=1.0, linestyle=':', alpha=0.5)
    
    # Place the text centered above the highest point
    ax.text(x, y_max + (y_max - y_min)*0.05, ast, ha='center', va='bottom', color='#ffffff', fontsize=12, fontweight='bold', zorder=10)

def main():
    try:
        bins, base_lzc, base_apen, stim_lzc, stim_apen, pval_lzc, pval_apen = parse_report("Information_Density_Report.txt")
    except FileNotFoundError:
        print("Information_Density_Report.txt not found. Run complexity_analysis.py first.")
        return
        
    if not bins:
        print("No multi-resolution data found in report.")
        return
    
    # Styling (Academic Dark Mode)
    sns.set_theme(style="darkgrid")
    bg_color = "#0d1117"
    text_color = "#c9d1d9"
    grid_color = "#30363d"
    
    plt.style.use('dark_background')
    
    # Updated for a standard two-column research paper format (8x4 inches)
    fig, axes = plt.subplots(1, 2, figsize=(8, 4))
    fig.patch.set_facecolor(bg_color)
    
    fig.suptitle('Multi-Resolution Information Density (N=10 Channels)', fontsize=12, fontweight='bold', color=text_color, y=1.02)
    
    # Colors
    color_base = "#00f0ff" # electric blue
    color_stim = "#ff7f00" # neon orange
    
    marker_size = 6
    line_width = 2.0
    
    for ax in axes:
        ax.set_facecolor(bg_color)
        ax.grid(True, color=grid_color, linestyle='--', alpha=0.7)
        ax.tick_params(colors=text_color, labelsize=9)
        for spine in ['bottom', 'left']:
            ax.spines[spine].set_color(grid_color)
            ax.spines[spine].set_linewidth(1.2)
        for spine in ['top', 'right']:
            ax.spines[spine].set_visible(False)
            
        ax.xaxis.label.set_color(text_color)
        ax.yaxis.label.set_color(text_color)
        ax.title.set_color(text_color)

    # Plot 1: LZC
    axes[0].plot(bins, base_lzc, marker='o', markersize=marker_size, linewidth=line_width, color=color_base, label='Spontaneous')
    axes[0].plot(bins, stim_lzc, marker='D', markersize=marker_size, linewidth=line_width, color=color_stim, label='Stimulated')
    for i, b in enumerate(bins):
        if i < len(pval_lzc):
            annotate_significance(axes[0], b, base_lzc[i], stim_lzc[i], pval_lzc[i])
            
    axes[0].set_title('LZC vs. \u0394t', fontsize=11, fontweight='bold', pad=10)
    axes[0].set_xlabel('Time Bin Size \u0394t (ms)', fontsize=10)
    axes[0].set_ylabel('Mean Normalized LZC', fontsize=10)
    
    ymin, ymax = axes[0].get_ylim()
    axes[0].set_ylim(ymin, ymax + (ymax - ymin)*0.15)

    # Plot 2: ApEn
    axes[1].plot(bins, base_apen, marker='o', markersize=marker_size, linewidth=line_width, color=color_base, label='Spontaneous')
    axes[1].plot(bins, stim_apen, marker='D', markersize=marker_size, linewidth=line_width, color=color_stim, label='Stimulated')
    for i, b in enumerate(bins):
        if i < len(pval_apen):
            annotate_significance(axes[1], b, base_apen[i], stim_apen[i], pval_apen[i])
            
    axes[1].set_title('Approximate Entropy vs. \u0394t', fontsize=11, fontweight='bold', pad=10)
    axes[1].set_xlabel('Time Bin Size \u0394t (ms)', fontsize=10)
    axes[1].set_ylabel('Mean Approx Entropy', fontsize=10)
    
    ymin, ymax = axes[1].get_ylim()
    axes[1].set_ylim(ymin, ymax + (ymax - ymin)*0.15)
    
    # Customizing Legends
    for ax in axes:
        legend = ax.legend(frameon=True, facecolor=bg_color, edgecolor=grid_color, fontsize=8, loc='best')
        for text in legend.get_texts():
            text.set_color(text_color)

    plt.tight_layout()
    
    # Export at 300 DPI
    out_file = "Multi_Resolution_Information_Density.png"
    plt.savefig(out_file, dpi=300, facecolor=bg_color, bbox_inches='tight')
    print(f"Successfully generated 8x4 academic dark-mode plot: {out_file}")

if __name__ == '__main__':
    main()
