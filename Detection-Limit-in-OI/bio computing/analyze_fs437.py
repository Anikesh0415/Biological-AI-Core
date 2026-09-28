import pandas as pd
import numpy as np
import sys
import os
from scipy.stats import ks_2samp

def main():
    fs437_hdf = "d:/bio computing/fs437_export/fs437_package.hdf5"
    fs369_hdf = "d:/bio computing/fs369data/fs369_package.hdf5"
    
    print("Loading fs437 metadata...")
    try:
        # Check metadata keys
        with pd.HDFStore(fs437_hdf, mode='r') as store:
            keys = store.keys()
            meta_key = [k for k in keys if 'metadata' in k]
            event_key = [k for k in keys if 'events' in k]
            print(f"Keys: {keys}")
        
        meta = pd.read_hdf(fs437_hdf, key=meta_key[0])
        print("fs437 Metadata:")
        print(meta.to_dict('records')[0])
    except Exception as e:
        print(f"Error loading metadata: {e}")

    print("\nLoading fs437 events...")
    events_437 = pd.read_hdf(fs437_hdf, key=event_key[0])
    print(f"Total fs437 events: {len(events_437)}")
    
    events_437 = events_437.sort_values(by=['electrode', 'time_of_event'])
    events_437['ISI'] = events_437.groupby('electrode')['time_of_event'].diff()
    
    if pd.api.types.is_timedelta64_dtype(events_437['ISI']):
        events_437['ISI'] = events_437['ISI'].dt.total_seconds()
        
    isi_437 = events_437.dropna(subset=['ISI']).copy()
    print(f"Total fs437 valid ISIs: {len(isi_437)}")
    
    fast_burst_threshold = 1e-3
    fast_events_437 = isi_437[isi_437['ISI'] < fast_burst_threshold]
    print(f"\nfs437 Events faster than 1.0 ms: {len(fast_events_437)}")
    
    fast_events_437.to_csv("d:/bio computing/fast_bursts_437.csv", index=False)
    
    bins = np.logspace(-6, 1, 100)
    hist, bin_edges = np.histogram(isi_437['ISI'], bins=bins)
    dist_df = pd.DataFrame({
        'ISI_lower_bound_sec': bin_edges[:-1],
        'ISI_upper_bound_sec': bin_edges[1:],
        'Frequency': hist
    })
    dist_df.to_csv("d:/bio computing/isi_distribution_437.csv", index=False)
    print("Saved isi_distribution_437.csv")
    
    print("\nLoading fs369 events for K-S test...")
    events_369 = pd.read_hdf(fs369_hdf, key='fs369_wholelife_events')
    events_369 = events_369.sort_values(by=['electrode', 'time_of_event'])
    events_369['ISI'] = events_369.groupby('electrode')['time_of_event'].diff()
    
    if pd.api.types.is_timedelta64_dtype(events_369['ISI']):
        events_369['ISI'] = events_369['ISI'].dt.total_seconds()
        
    isi_369_data = events_369['ISI'].dropna().values
    isi_437_data = isi_437['ISI'].values
    
    print("Performing K-S test...")
    ks_stat, p_value = ks_2samp(isi_369_data, isi_437_data)
    print(f"K-S statistic: {ks_stat}")
    print(f"p-value: {p_value}")

if __name__ == '__main__':
    main()
