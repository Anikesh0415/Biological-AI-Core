import pandas as pd
import numpy as np
import os
import math
from collections import Counter
from scipy.stats import wilcoxon
import scipy.signal as signal
from scipy.ndimage import gaussian_filter1d

# --- Configuration ---
SPIKE_FILE = "FS369_spikes.csv"
STIM_FILE = "FS369_stim.csv"
WINDOW_SEC = 60.0
FS = 1000.0 # 1000 Hz Sampling frequency
SIGMA_MS = 10.0 # Gaussian kernel standard deviation

BANDS = {
    'Theta (4-8 Hz)': (4.0, 8.0),
    'Alpha (8-12 Hz)': (8.0, 12.0),
    'Beta (12-30 Hz)': (12.0, 30.0),
    'Gamma (30-100 Hz)': (30.0, 100.0)
}
# ---------------------

def lz_complexity(s):
    if not s: return 0
    n = len(s)
    c = 1
    v = 1
    w = 1
    while v < n:
        if s[v:v+w] in s[0:v+w-1]:
            w += 1
            if v + w > n:
                break
        else:
            c += 1
            v += w
            w = 1
    return c

def normalized_lzc(s):
    n = len(s)
    if n < 2: return 0.0
    c = lz_complexity(s)
    return c / (n / math.log2(n))

def approx_entropy(U, m=2):
    U = str(U)
    N = len(U)
    if N < m + 1: return 0.0

    def _phi(m_len):
        if N - m_len + 1 <= 0: return 0.0
        patterns = [U[i:i+m_len] for i in range(N - m_len + 1)]
        counts = Counter(patterns)
        total = N - m_len + 1
        return sum((v / total) * math.log(v / total) for v in counts.values())
    
    phi_m = _phi(m)
    phi_m_plus_1 = _phi(m + 1)
    return abs(phi_m - phi_m_plus_1)

def get_continuous_sdf(spikes, start, end, fs=1000.0, sigma_ms=10.0):
    duration = end - start
    num_bins = int(duration * fs)
    bins = np.linspace(start, end, num_bins + 1)
    counts, _ = np.histogram(spikes, bins=bins)
    sigma_bins = sigma_ms / (1000.0 / fs)
    sdf = gaussian_filter1d(counts.astype(float), sigma=sigma_bins)
    return sdf

def butter_bandpass_filter(data, lowcut, highcut, fs, order=4):
    nyq = 0.5 * fs
    low = lowcut / nyq
    high = highcut / nyq
    b, a = signal.butter(order, [low, high], btype='bandpass')
    y = signal.filtfilt(b, a, data)
    return y

def binarize_zero_crossing(continuous_signal):
    return "".join(['1' if x > 0 else '0' for x in continuous_signal])

def generate_mock_data():
    print("Raw data files not found. Generating mock FS369 data for demonstration...")
    np.random.seed(42)
    spike_rows = []
    for ch in range(10): 
        n_base = int(150 * 10)
        base_ts = np.random.uniform(0, 150, n_base)
        n_stim = int(150 * 25)
        stim_ts = np.random.uniform(150, 300, n_stim)
        ts = np.concatenate([base_ts, stim_ts])
        ts.sort()
        for t in ts:
            spike_rows.append({"channel": ch, "timestamp": t})
    df_spikes = pd.DataFrame(spike_rows)
    df_spikes.to_csv(SPIKE_FILE, index=False)
    df_stim = pd.DataFrame([{"timestamp": 150.0, "duration": 0.5, "type": "electrical"}])
    df_stim.to_csv(STIM_FILE, index=False)

def main():
    if not os.path.exists(SPIKE_FILE) or not os.path.exists(STIM_FILE):
        generate_mock_data()
        
    print("Loading raw neural spike events and stimulation logs...")
    df_spikes = pd.read_csv(SPIKE_FILE)
    df_stim = pd.read_csv(STIM_FILE)
    
    first_stim_ts = df_stim['timestamp'].min()
    if np.isnan(first_stim_ts):
        first_stim_ts = 150.0 
        
    baseline_end = first_stim_ts - 5.0 
    baseline_start = baseline_end - WINDOW_SEC
    stim_start = first_stim_ts
    stim_end = stim_start + WINDOW_SEC
    
    channels = df_spikes['channel'].unique()
    
    report_lines = [
        "======================================",
        "       Frequency Band Report          ",
        "======================================\n",
        "1. Analysis Parameters:",
        f"   - Window Duration: {WINDOW_SEC} seconds",
        f"   - Continuous Sampling Frequency (Fs): {FS} Hz",
        f"   - Gaussian SDF Sigma: {SIGMA_MS} ms",
        f"   - Number of Channels Analyzed: {len(channels)}\n",
        "2. Multi-Band Complexity & Entropy Analysis:\n"
    ]
    
    print("Running DSP Filter Pipeline & Algorithmic Complexity Engine...")
    
    for band_name, (lowcut, highcut) in BANDS.items():
        baseline_lzc = []
        stim_lzc = []
        baseline_apen = []
        stim_apen = []
        
        for ch in channels:
            ch_spikes = df_spikes[df_spikes['channel'] == ch]['timestamp'].values
            base_spikes = ch_spikes[(ch_spikes >= baseline_start) & (ch_spikes < baseline_end)]
            stim_window_spikes = ch_spikes[(ch_spikes >= stim_start) & (ch_spikes < stim_end)]
            
            # Step 1: Continuous SDF
            base_sdf = get_continuous_sdf(base_spikes, baseline_start, baseline_end, fs=FS, sigma_ms=SIGMA_MS)
            stim_sdf = get_continuous_sdf(stim_window_spikes, stim_start, stim_end, fs=FS, sigma_ms=SIGMA_MS)
            
            # Step 2: Butterworth Filter
            base_filt = butter_bandpass_filter(base_sdf, lowcut, highcut, fs=FS)
            stim_filt = butter_bandpass_filter(stim_sdf, lowcut, highcut, fs=FS)
            
            # Step 3: Zero-crossing Re-binarization
            base_bin = binarize_zero_crossing(base_filt)
            stim_bin = binarize_zero_crossing(stim_filt)
            
            if len(base_bin) > 0 and len(stim_bin) > 0:
                baseline_lzc.append(normalized_lzc(base_bin))
                stim_lzc.append(normalized_lzc(stim_bin))
                baseline_apen.append(approx_entropy(base_bin, m=2))
                stim_apen.append(approx_entropy(stim_bin, m=2))
                
        # Statistics
        m_base_lzc = np.mean(baseline_lzc)
        m_stim_lzc = np.mean(stim_lzc)
        d_lzc = m_stim_lzc - m_base_lzc
        p_lzc_shift = (d_lzc / m_base_lzc * 100) if m_base_lzc != 0 else 0
        
        m_base_apen = np.mean(baseline_apen)
        m_stim_apen = np.mean(stim_apen)
        d_apen = m_stim_apen - m_base_apen
        p_apen_shift = (d_apen / m_base_apen * 100) if m_base_apen != 0 else 0
        
        p_val_lzc = 1.0
        p_val_apen = 1.0
        try:
            if len(baseline_lzc) >= 2:
                _, p_val_lzc = wilcoxon(baseline_lzc, stim_lzc)
                _, p_val_apen = wilcoxon(baseline_apen, stim_apen)
        except ValueError:
            pass
            
        report_lines.extend([
            f"--- Band: {band_name} ---",
            "  State 1: Spontaneous Activity (Baseline)",
            f"     - Mean Normalized LZC: {m_base_lzc:.4f}",
            f"     - Mean Approx Entropy: {m_base_apen:.4f}",
            "  State 2: Stimulated Activity",
            f"     - Mean Normalized LZC: {m_stim_lzc:.4f}",
            f"     - Mean Approx Entropy: {m_stim_apen:.4f}",
            "  Complexity Shift:",
            f"     - Delta LZC:  {d_lzc:+.4f} ({p_lzc_shift:+.2f}%)",
            f"     - p-value LZC: {p_val_lzc:.4e}",
            f"     - Delta ApEn: {d_apen:+.4f} ({p_apen_shift:+.2f}%)",
            f"     - p-value ApEn: {p_val_apen:.4e}\n"
        ])
        
    report_lines.append("======================================")
    with open("Frequency_Band_Report.txt", "w", encoding="utf-8") as f:
        f.write("\n".join(report_lines))
    print("Analysis complete. Report successfully generated: Frequency_Band_Report.txt")

if __name__ == "__main__":
    main()
