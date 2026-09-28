import json
import networkx as nx
from collections import Counter

def analyze_network(input_file, output_file, name):
    print(f"--- Analyzing {name} ---")
    
    # Load JSON
    with open(input_file, 'r') as f:
        data = json.load(f)
        
    G = nx.Graph()
    
    # Add nodes and edges
    # Check if edges are under 'edges' or 'links'
    edge_key = 'edges' if 'edges' in data else 'links'
    
    for node in data['nodes']:
        G.add_node(node['id'])
        
    for edge in data[edge_key]:
        G.add_edge(edge['source'], edge['target'])
        
    # 1. Degree Distribution Analysis P(k)
    degrees = [d for n, d in G.degree()]
    total_nodes = len(degrees)
    degree_counts = Counter(degrees)
    
    degree_distribution = []
    for k in sorted(degree_counts.keys()):
        p_k = degree_counts[k] / total_nodes
        degree_distribution.append({"k": k, "p_k": p_k})
        
    # 2. Centrality Profiling
    # Betweenness Centrality (normalized by default in NetworkX)
    betweenness = nx.betweenness_centrality(G)
    
    # PageRank
    pagerank = nx.pagerank(G)
    
    # Format node metrics
    node_metrics = []
    for node in G.nodes():
        node_metrics.append({
            "id": node,
            "degree": G.degree(node),
            "betweenness_centrality": betweenness[node],
            "pagerank": pagerank[node]
        })
        
    # Sort nodes by PageRank to find top hubs
    sorted_nodes = sorted(node_metrics, key=lambda x: x['pagerank'], reverse=True)
    
    print(f"Top 3 Highest PageRank Hubs for {name}:")
    for i in range(min(3, len(sorted_nodes))):
        n = sorted_nodes[i]
        print(f"  {i+1}. Node ID: {n['id']} | PageRank: {n['pagerank']:.6f} | Degree: {n['degree']}")
    print("\n")
    
    # Export Advanced Metrics
    output_data = {
        "degree_distribution": degree_distribution,
        "node_metrics": node_metrics
    }
    
    with open(output_file, 'w') as f:
        json.dump(output_data, f, indent=2)

if __name__ == "__main__":
    # Paths to the data
    bio_input = "d:/Neuro-pro/dashboard/src/data/neural_network_data.json"
    cosmic_input = "d:/Neuro-pro/dashboard/src/data/cosmic_network_data.json"
    
    bio_output = "d:/Neuro-pro/neural_advanced_metrics.json"
    cosmic_output = "d:/Neuro-pro/cosmic_advanced_metrics.json"
    
    analyze_network(bio_input, bio_output, "Biological Neural Network")
    analyze_network(cosmic_input, cosmic_output, "Cosmic Web")
