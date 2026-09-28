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
    bands = []
    delta_lzc_pct = []
    p_vals = []
    
    current_band = None
    
    with open(filename, 'r', encoding='utf-8') as f:
        for line in f:
            line = line.strip()
            band_match = re.search(r"--- Band:\s+([^\-]+)", line)
            if band_match:
                current_band = band_match.group(1).strip()
                bands.append(current_band)
                
            if "Delta LZC:" in line:
                pct_match = re.search(r"\(([\+\-\d\.]+)\%\)", line)
                if pct_match:
                    delta_lzc_pct.append(float(pct_match.group(1)))
                    
            if "p-value LZC:" in line:
                val = float(line.split(":")[1].strip())
                p_vals.append(val)
                
    return bands, delta_lzc_pct, p_vals

def main():
    try:
        bands, delta_lzc_pct, p_vals = parse_report("Frequency_Band_Report.txt")
    except FileNotFoundError:
        print("Frequency_Band_Report.txt not found.")
        return
        
    if not bands:
        print("No band data found in report.")
        return
        
    sns.set_theme(style="darkgrid")
    bg_color = "#0d1117"
    text_color = "#c9d1d9"
    grid_color = "#30363d"
    
    plt.style.use('dark_background')
    
    # Standard two-column paper format
    fig, ax = plt.subplots(figsize=(8, 4))
    fig.patch.set_facecolor(bg_color)
    ax.set_facecolor(bg_color)
    
    fig.suptitle('Frequency-Dependent Complexity Shift (N=10 Channels)', fontsize=14, fontweight='bold', color=text_color, y=1.02)
    
    # Academic dark mode styling with neon highlights for significant discoveries
    colors = []
    for p in p_vals:
        if p < 0.05:
            colors.append("#00f0ff")  # glowing electric cyan
        else:
            colors.append("#3a4350")  # muted, low-opacity gray/blue
            
    bars = ax.bar(bands, delta_lzc_pct, color=colors, edgecolor='none', width=0.6)
    
    ax.grid(True, color=grid_color, linestyle='--', alpha=0.7, axis='y')
    ax.tick_params(colors=text_color, labelsize=11)
    
    for spine in ['bottom', 'left']:
        ax.spines[spine].set_color(grid_color)
        ax.spines[spine].set_linewidth(1.5)
    for spine in ['top', 'right']:
        ax.spines[spine].set_visible(False)
        
    ax.set_ylabel('Percentage Shift in LZC (\u0394 LZC %)', fontsize=12, color=text_color)
    ax.set_xlabel('Neural Frequency Band', fontsize=12, color=text_color)
    
    # Statistical annotations directly above bars
    max_height = max(max(delta_lzc_pct), 0)
    offset = max_height * 0.08 if max_height > 0 else 5.0
    
    for i, bar in enumerate(bars):
        height = bar.get_height()
        ast = get_asterisks(p_vals[i])
        
        y_pos = height + offset if height >= 0 else height - offset
        va = 'bottom' if height >= 0 else 'top'
        
        # Color match significance text slightly for better pop
        color_ast = "#ffffff" if ast != 'ns' else "#8b949e"
        weight = 'bold' if ast != 'ns' else 'normal'
        font_sz = 16 if ast != 'ns' else 12
        
        ax.text(bar.get_x() + bar.get_width()/2., y_pos, ast,
                ha='center', va=va, color=color_ast, fontsize=font_sz, fontweight=weight)
                
    ymin, ymax = ax.get_ylim()
    # Ensure sufficient headroom for asterisks
    ax.set_ylim(min(ymin, 0) - offset*0.5, ymax + offset*2.5)
    
    plt.tight_layout()
    out_file = "Frequency_Dependent_Complexity.png"
    plt.savefig(out_file, dpi=300, facecolor=bg_color, bbox_inches='tight')
    print(f"Successfully generated dark-mode plot: {out_file}")

if __name__ == '__main__':
    main()
