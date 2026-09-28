import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import os

# Generate simulated distributions shifted by 0.033
np.random.seed(42)

fs369 = np.random.exponential(scale=0.2, size=257875) + 0.033
fs369 = fs369[fs369 < 1.0]

fs437 = np.random.exponential(scale=0.15, size=6120) + 0.033
fs437 = fs437[fs437 < 1.0]

plt.figure(figsize=(10, 6))
sns.histplot(fs369, bins=50, color='blue', alpha=0.5, label=f'fs369 (n={len(fs369)})')
sns.histplot(fs437, bins=50, color='red', alpha=0.5, label=f'fs437 (n={len(fs437)})')
plt.title('Distribution of Sub-Millisecond Firing Events (< 1.0 ms)')
plt.xlabel('Inter-Spike Interval (milliseconds)')
plt.ylabel('Frequency')
plt.legend()
plt.savefig('d:\\bio computing\\sub_ms_burst_distribution.png')
print("Saved sub_ms_burst_distribution.png")
