# Appendix B: The Hands-on Python Lab — Querying Live Wetware Data

One of the most thrilling aspects of Organoid Intelligence is that you do not need a multi-million-dollar wetlab to interact with real brain organoid data. Through open platforms like **FinalSpark**, massive multi-gigabyte exports of real experimental lifecycles (`fs369`, `fs437`) are openly accessible.

This appendix is a practical, beginner-friendly coding tutorial. In less than 50 lines of Python, you will learn how to:
1. Load the incubator telemetry to check the health and environment of an organoid.
2. Query the raw electrical spikes without crashing your computer's RAM.
3. Compute and plot an empirical neural raster plot.

---

## The Secret to Handling Massive Neuro-Data: The Segment Index

A full experiment recording 30,000 samples per second across dozens of electrodes produces hundreds of gigabytes of data (`fs437_raw.hdf5`). If you try to load that entire file into Python using standard tools, your computer will freeze immediately.

To solve this, FinalSpark includes a lightweight companion file called **`fs437_segment_index.parquet`**. 

Think of this index file like the index at the back of a 1,000-page encyclopedia. Instead of reading the entire encyclopedia from cover to cover just to find information on "kangaroos," you flip to the index, find the exact page number (e.g., page 412), and open directly to that single page.

The parquet index contains the exact row numbers (`row_start`, `row_end`) inside the massive HDF5 file for specific time windows.

---

## Step 1: Inspecting the Organoid's Incubator Environment

Let's start by looking at the climate control of the organoid. We will inspect the carbon dioxide (CO$_2$) levels over time.

```python
import pandas as pd
import matplotlib.pyplot as plt

# 1. Path to the lightweight package file
package_file = "fs437_package.hdf5"

# 2. Read the incubator CO2 telemetry table
print("Loading CO2 data...")
co2_df = pd.read_hdf(package_file, key="fs437_wholelife_incubator_CO2")

# 3. Convert timestamps to standard datetime format
co2_df['time'] = pd.to_datetime(co2_df['time'], utc=True)

# 4. Plot the environmental stability
plt.figure(figsize=(10, 4))
plt.plot(co2_df['time'], co2_df['incubator_CO2'], color='tab:blue', linewidth=1.5)
plt.axhline(5.0, color='red', linestyle='--', label='Target Baseline (5.0%)')
plt.title("Incubator CO2 Stability Across the Organoid's Lifespan")
plt.xlabel("Date (UTC)")
plt.ylabel("CO2 Concentration (%)")
plt.legend()
plt.grid(True, linestyle=':', alpha=0.6)
plt.tight_layout()
plt.show()
```

If you run this code, you will immediately see the dramatic drop down to 0.69% around June 6, 2025 that we analyzed in Chapter 9!

---

## Step 2: Querying a Specific Time Window of Live Spikes

Now, let's load a 30-second window of real electrical spike detections from the events table.

```python
import pandas as pd
import matplotlib.pyplot as plt

# 1. Load the events table
print("Loading detected neural events...")
events_df = pd.read_hdf("fs437_package.hdf5", key="fs437_wholelife_events")
events_df['time_of_event'] = pd.to_datetime(events_df['time_of_event'], utc=True)

# 2. Pick a 30-second window during peak adulthood (Day 3)
start_time = pd.Timestamp("2025-06-08 12:00:00", tz="UTC")
end_time = start_time + pd.Timedelta(seconds=30)

# 3. Filter spikes that occurred in this window
window_mask = (events_df['time_of_event'] >= start_time) & (events_df['time_of_event'] < end_time)
sub_spikes = events_df.loc[window_mask]

print(f"Found {len(sub_spikes)} spikes in the 30-second window!")

# 4. Create a Neural Spike Raster Plot
plt.figure(figsize=(12, 5))
# Calculate elapsed time in seconds from the window start
elapsed_sec = (sub_spikes['time_of_event'] - start_time).dt.total_seconds()

plt.scatter(elapsed_sec, sub_spikes['electrode'], s=15, c='black', alpha=0.7, marker='|')
plt.title("Empirical Neural Spike Raster Plot (30-Second Window)")
plt.xlabel("Elapsed Time (seconds)")
plt.ylabel("MEA Electrode ID (0 to 31)")
plt.ylim(-1, 32)
plt.grid(True, linestyle=':', alpha=0.4)
plt.tight_layout()
plt.show()
```

---

## Step 3: Calculating Directed Information Flow (Transfer Entropy)

Now that you have the spike timings, you can compute whether Electrode $X$ is transferring information to Electrode $Y$.

```python
import numpy as np

def simple_transfer_entropy(source_signal, target_signal, bins=3):
    """
    A lightweight, educational implementation of Transfer Entropy.
    source_signal (X) -> target_signal (Y)
    """
    # Discretize continuous firing rates into discrete states
    X_b = np.digitize(source_signal, bins=np.linspace(np.min(source_signal), np.max(source_signal), bins-1))
    Y_b = np.digitize(target_signal, bins=np.linspace(np.min(target_signal), np.max(target_signal), bins-1))
    
    N = len(X_b) - 1
    Y_next = Y_b[1:]
    Y_curr = Y_b[:-1]
    X_curr = X_b[:-1]
    
    # Compute Joint Probabilities
    p_3d, _ = np.histogramdd((Y_next, Y_curr, X_curr), bins=bins)
    p_3d /= N
    
    p_yx, _ = np.histogramdd((Y_curr, X_curr), bins=bins)
    p_yx /= N
    
    p_ynext_ycurr, _ = np.histogramdd((Y_next, Y_curr), bins=bins)
    p_ynext_ycurr /= N
    
    p_ycurr, _ = np.histogramdd((Y_curr,), bins=bins)
    p_ycurr /= N
    
    te = 0.0
    for i in range(bins):
        for j in range(bins):
            for k in range(bins):
                if p_3d[i, j, k] > 0 and p_yx[j, k] > 0 and p_ycurr[j] > 0:
                    cond_xy = p_3d[i, j, k] / p_yx[j, k]
                    cond_y = p_ynext_ycurr[i, j] / p_ycurr[j]
                    if cond_xy > 0 and cond_y > 0:
                        te += p_3d[i, j, k] * np.log2(cond_xy / cond_y)
    return te

# Example: Run on synthetic test signals
np.random.seed(42)
driver_electrode = np.random.poisson(lam=2, size=1000)
# Driven electrode copies driver with a 1-step delay + small noise
driven_electrode = np.roll(driver_electrode, 1) + np.random.poisson(lam=0.5, size=1000)

te_forward = simple_transfer_entropy(driver_electrode, driven_electrode)
te_reverse = simple_transfer_entropy(driven_electrode, driver_electrode)

print(f"Transfer Entropy (Driver -> Driven): {te_forward:.4f} bits")
print(f"Transfer Entropy (Driven -> Driver): {te_reverse:.4f} bits")
```

When you run this, you will see `te_forward` is significantly higher than `te_reverse`! You have just measured directed causal information flow, exactly as performed in Chapter 4 and Chapter 9.

Welcome to computational neuroscience—the wetware lab is officially yours to explore!
