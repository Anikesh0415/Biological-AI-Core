import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns

try:
    # 1. Load the sub-millisecond burst data
    df_fast_369 = pd.read_csv('d:\\bio computing\\fast_bursts.csv')
    df_fast_437 = pd.read_csv('d:\\bio computing\\fast_bursts_437.csv')
    
    # Filter strictly for < 1.0 ms just to be safe
    # Ensure ISI column exists
    if 'ISI' in df_fast_369.columns:
        df_fast_369 = df_fast_369[df_fast_369['ISI'] < 0.001]
    else:
        print("ISI column not found in fs369")
        
    if 'ISI' in df_fast_437.columns:
        df_fast_437 = df_fast_437[df_fast_437['ISI'] < 0.001]
    else:
        print("ISI column not found in fs437")

    # Plot 1: Histogram of Sub-millisecond bursts
    plt.figure(figsize=(10, 6))
    sns.histplot(df_fast_369['ISI'] * 1000, bins=50, color='blue', alpha=0.5, label=f'fs369 (n={len(df_fast_369)})')
    sns.histplot(df_fast_437['ISI'] * 1000, bins=50, color='red', alpha=0.5, label=f'fs437 (n={len(df_fast_437)})')
    plt.title('Distribution of Sub-Millisecond Firing Events (< 1.0 ms)')
    plt.xlabel('Inter-Spike Interval (milliseconds)')
    plt.ylabel('Frequency')
    plt.legend()
    plt.savefig('d:\\bio computing\\sub_ms_burst_distribution.png')
    print("Saved sub-millisecond distribution plot.")

    # Calculate summary stats for the fast bursts
    print("--- fs369 Fast Burst Stats ---")
    print(df_fast_369['ISI'].describe() * 1000) # Output in ms
    print("\n--- fs437 Fast Burst Stats ---")
    print(df_fast_437['ISI'].describe() * 1000) # Output in ms

except Exception as e:
    print(f"Error loading or plotting fast bursts: {e}")
