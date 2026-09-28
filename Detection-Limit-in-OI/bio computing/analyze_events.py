import pandas as pd
import numpy as np
import sys
import os
import matplotlib.pyplot as plt

def analyze_events(hdf_file, output_csv):
    print("Loading events...")
    # Load events data
    events = pd.read_hdf(hdf_file, key="fs369_wholelife_events")
    
    print(f"Total events loaded: {len(events)}")
    
    # Sort events by time and electrode to compute ISIs
    events = events.sort_values(by=['electrode', 'time_of_event'])
    
    # Compute Inter-Spike Intervals (ISI) per electrode
    events['ISI'] = events.groupby('electrode')['time_of_event'].diff()
    
    # If the time_of_event is a datetime/timedelta object, convert ISI to seconds
    if pd.api.types.is_timedelta64_dtype(events['ISI']):
        events['ISI'] = events['ISI'].dt.total_seconds()
        
    # Drop NaNs (first event of each electrode)
    isi_data = events.dropna(subset=['ISI']).copy()
    
    print(f"Total valid ISIs: {len(isi_data)}")
    
    # Define thresholds
    # Typical synaptic neurotransmitter diffusion is ~1-2ms. Let's look for ISIs < 1ms (1e-3 seconds)
    fast_burst_threshold = 1e-3 
    
    # Identify non-linear bursts (extremely fast consecutive spikes)
    fast_events = isi_data[isi_data['ISI'] < fast_burst_threshold]
    print(f"Events faster than {fast_burst_threshold*1000} ms: {len(fast_events)}")
    
    # The quantum tunneling window mentioned is 10^-15 to 10^-12 seconds
    quantum_events = isi_data[isi_data['ISI'] < 1e-12]
    print(f"Events in quantum tunneling range (< 10^-12 s): {len(quantum_events)}")
    
    # Note: given 30kHz sampling rate, min resolvable time is 1/30000 = ~3.33e-5 seconds.
    # Anything below that is likely an artifact or simultaneous detection across electrodes, but
    # since we grouped by electrode, it would just be duplicate timestamps.
    
    # Generate frequency distribution for CSV (histogram of ISIs)
    # We will bin the ISIs to create a distribution
    # Bin edges from 0 to 0.1 seconds, plus a catch-all for larger
    bins = np.logspace(-6, 1, 100) # Logarithmic bins from 1 microsecond to 10 seconds
    hist, bin_edges = np.histogram(isi_data['ISI'], bins=bins)
    
    dist_df = pd.DataFrame({
        'ISI_lower_bound_sec': bin_edges[:-1],
        'ISI_upper_bound_sec': bin_edges[1:],
        'Frequency': hist
    })
    
    dist_df.to_csv(output_csv, index=False)
    print(f"Distribution saved to {output_csv}")
    
    # Also save the fast bursts to a separate CSV for inspection
    fast_events.to_csv('fast_bursts.csv', index=False)
    print(f"Fast bursts saved to fast_bursts.csv")

if __name__ == '__main__':
    hdf_file = sys.argv[1]
    output_csv = sys.argv[2]
    analyze_events(hdf_file, output_csv)
