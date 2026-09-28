import pandas as pd
import networkx as nx
import time
import numpy as np
import json
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
    
    df = df.sort_values(by='time_of_event').reset_index(drop=True)
    
    times_ms = (df['time_of_event'].astype('int64') // 10**6).values
    electrodes = df['electrode'].values.astype(np.int64)
    
    edges = Dict.empty(
        key_type=types.Tuple((types.int64, types.int64)),
        value_type=types.int64,
    )
    
    print("Finding co-occurrences within 50ms with Numba...")
    find_edges(times_ms, electrodes, edges)
    
    # 1. Load the co-occurrence edge weight dataset into memory
    edges_list = []
    weights_list = []
    for (u, v), weight in edges.items():
        edges_list.append({"source": str(u), "target": str(v), "weight": float(weight)})
        weights_list.append(weight)
        
    weights_arr = np.array(weights_list)
    
    # 2. Calculate the exact 95th percentile threshold value of the 'weight' column
    threshold_95 = np.percentile(weights_arr, 95)
    print(f"Calculated 95th percentile threshold: {threshold_95:.2f}")
    
    # 3. Filter the dataframe to retain ONLY edges >= threshold
    filtered_edges = [e for e in edges_list if e["weight"] >= threshold_95]
    print(f"Retained {len(filtered_edges)} edges out of {len(edges_list)}.")
    
    # 4. Construct an undirected graph
    G = nx.Graph()
    for e in filtered_edges:
        G.add_edge(e["source"], e["target"], weight=e["weight"])
        
    # 5. Compute the metrics
    active_nodes = G.number_of_nodes()
    total_edges = G.number_of_edges()
    
    if active_nodes > 0:
        avg_degree = sum(dict(G.degree()).values()) / active_nodes
        avg_clust = nx.average_clustering(G)
    else:
        avg_degree = 0.0
        avg_clust = 0.0
        
    print("\n--- Network Metrics (Filtered >= 95th Percentile) ---")
    print(f"Total number of active nodes: {active_nodes}")
    print(f"Total number of remaining edges: {total_edges}")
    print(f"Average Node Degree: {avg_degree:.4f}")
    print(f"Average Clustering Coefficient: {avg_clust:.4f}")
    
    # 6. Export to JSON
    output_data = {
        "metrics": {
            "average_degree": avg_degree,
            "clustering_coefficient": avg_clust,
            "network_type": "Biological (Filtered 95th Percentile)"
        },
        "nodes": [{"id": node} for node in G.nodes()],
        "edges": filtered_edges
    }
    
    out_path = "d:/Neuro-pro/neural_network_data.json"
    with open(out_path, "w") as f:
        json.dump(output_data, f, indent=2)
        
    print(f"\nSuccessfully exported data to {out_path}")

if __name__ == "__main__":
    main()
