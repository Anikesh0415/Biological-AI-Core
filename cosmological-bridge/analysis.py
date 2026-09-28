import pandas as pd
import networkx as nx
import time
import numpy as np
from numba import njit, types
from numba.typed import Dict

@njit
def find_edges(times_ms, electrodes, edges):
    n = len(times_ms)
    left = 0
    for i in range(n):
        t_i = times_ms[i]
        el_i = electrodes[i]
        
        while left < i and t_i - times_ms[left] > 50:
            left += 1
            
        for j in range(left, i):
            el_j = electrodes[j]
            if el_i != el_j:
                u, v = (el_i, el_j) if el_i < el_j else (el_j, el_i)
                pair = (u, v)
                if pair in edges:
                    edges[pair] += 1
                else:
                    edges[pair] = 1

def main():
    path = "d:/Neuro-pro/fs369data/fs369_package.hdf5"
    print("Loading data...")
    t0 = time.time()
    df = pd.read_hdf(path, key="/fs369_wholelife_events")
    print(f"Data loaded in {time.time() - t0:.2f}s. Total events: {len(df)}")
    
    t0 = time.time()
    df = df.sort_values(by='time_of_event').reset_index(drop=True)
    print(f"Sorted in {time.time() - t0:.2f}s")
    
    # We want time in ms
    times_ms = (df['time_of_event'].astype('int64') // 10**6).values
    electrodes = df['electrode'].values.astype(np.int64)
    
    edges = Dict.empty(
        key_type=types.UniTuple(types.int64, 2),
        value_type=types.int64,
    )
    
    print("Finding co-occurrences within 50ms with Numba...")
    t0 = time.time()
    find_edges(times_ms, electrodes, edges)
    print(f"Edges found in {time.time() - t0:.2f}s. Unique edges: {len(edges)}")
    
    # Create Graph
    G = nx.Graph()
    for (u, v), weight in edges.items():
        G.add_edge(int(u), int(v), weight=int(weight))
        
    active_nodes = G.number_of_nodes()
    total_edges = G.number_of_edges()
    
    if active_nodes > 0:
        avg_degree = sum(dict(G.degree()).values()) / active_nodes
        clustering_coeff = nx.transitivity(G)
    else:
        avg_degree = 0
        clustering_coeff = 0
        
    print("\n--- Network Metrics ---")
    print(f"Total number of active nodes (electrodes): {active_nodes}")
    print(f"Total number of edges: {total_edges}")
    print(f"Average Node Degree: {avg_degree:.4f}")
    print(f"Global Clustering Coefficient: {clustering_coeff:.4f}")

if __name__ == "__main__":
    main()
