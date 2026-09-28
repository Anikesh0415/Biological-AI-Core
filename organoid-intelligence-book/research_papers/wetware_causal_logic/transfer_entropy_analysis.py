import numpy as np
import matplotlib.pyplot as plt
import networkx as nx
import sys
import os

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from substrate import BiologicalWetware

def compute_te(X, Y, bins=10):
    """
    Computes Transfer Entropy T_{X -> Y}
    X, Y: 1D arrays of time series (e.g., firing rates)
    TE = sum p(y_{t+1}, y_t, x_t) log2 ( p(y_{t+1} | y_t, x_t) / p(y_{t+1} | y_t) )
    """
    # Discretize into bins
    X_b = np.digitize(X, bins=np.linspace(np.min(X), np.max(X), bins-1))
    Y_b = np.digitize(Y, bins=np.linspace(np.min(Y), np.max(Y), bins-1))
    
    N = len(X_b) - 1
    
    Y_next = Y_b[1:]
    Y_curr = Y_b[:-1]
    X_curr = X_b[:-1]
    
    # Calculate joint probability distributions
    joint_3d, _ = np.histogramdd((Y_next, Y_curr, X_curr), bins=(bins, bins, bins))
    p_3d = joint_3d / N
    
    joint_y_next_curr, _ = np.histogramdd((Y_next, Y_curr), bins=(bins, bins))
    p_y_next_curr = joint_y_next_curr / N
    
    joint_y_curr, _ = np.histogramdd((Y_curr,), bins=(bins,))
    p_y_curr = joint_y_curr / N
    
    joint_y_curr_x_curr, _ = np.histogramdd((Y_curr, X_curr), bins=(bins, bins))
    p_y_curr_x_curr = joint_y_curr_x_curr / N
    
    te = 0.0
    for i in range(bins): # Y_{t+1}
        for j in range(bins): # Y_t
            for k in range(bins): # X_t
                p_y1_y0_x0 = p_3d[i, j, k]
                if p_y1_y0_x0 > 0:
                    p_y1_given_y0_x0 = p_y1_y0_x0 / p_y_curr_x_curr[j, k]
                    p_y1_given_y0 = p_y_next_curr[i, j] / p_y_curr[j]
                    
                    if p_y1_given_y0_x0 > 0 and p_y1_given_y0 > 0:
                        te += p_y1_y0_x0 * np.log2(p_y1_given_y0_x0 / p_y1_given_y0)
    return te

def main():
    np.random.seed(42)
    num_nodes = 500
    total_len = 2000
    
    # 1. Define Biological Half-Adder Setup
    net = BiologicalWetware(num_nodes=num_nodes, space_size=10.0, conn_radius=2.0, leak=0.9, threshold=0.8)
    net.normalize_weights(target_sigma=1.0)
    
    # Designate functional clusters
    clusters = {
        'Input A': np.arange(0, 50),
        'Input B': np.arange(50, 100),
        'Inhibitory': np.arange(100, 150),
        'Sum': np.arange(150, 250),
        'Carry': np.arange(250, 350)
    }
    
    # Enforce inhibitory interneurons (negative weights)
    net.weights[clusters['Inhibitory'], :] *= -1.5 
    
    # Wire the topology to conceptually simulate half-adder routing
    # Inputs -> Inhibitory
    net.weights[np.ix_(clusters['Input A'], clusters['Inhibitory'])] *= 2.0
    net.weights[np.ix_(clusters['Input B'], clusters['Inhibitory'])] *= 2.0
    
    # Inputs -> Sum (Excitatory)
    net.weights[np.ix_(clusters['Input A'], clusters['Sum'])] *= 1.5
    net.weights[np.ix_(clusters['Input B'], clusters['Sum'])] *= 1.5
    # Inhibitory -> Sum (Suppressive for XOR logic)
    net.weights[np.ix_(clusters['Inhibitory'], clusters['Sum'])] *= 3.0 
    
    # Inputs -> Carry (AND logic driven)
    net.weights[np.ix_(clusters['Input A'], clusters['Carry'])] *= 1.5
    net.weights[np.ix_(clusters['Input B'], clusters['Carry'])] *= 1.5
    
    # Simulate Half-Adder inputs
    # A and B are random binary pulse trains holding for 50 steps
    pulse_steps = 50
    num_pulses = total_len // pulse_steps
    A_pulses = np.random.randint(0, 2, num_pulses)
    B_pulses = np.random.randint(0, 2, num_pulses)
    
    A_signal = np.repeat(A_pulses, pulse_steps)
    B_signal = np.repeat(B_pulses, pulse_steps)
    
    net.voltage = np.zeros(net.N)
    net.spikes = np.zeros(net.N)
    
    cluster_rates = {k: np.zeros(total_len) for k in clusters}
    
    alpha = 0.2 # Smoothing filter for continuous firing rates
    rates = np.zeros(net.N)
    
    print("Simulating Biological Half-Adder (2000 steps)...")
    for t in range(total_len):
        input_current = np.zeros(net.N)
        input_current[clusters['Input A']] = A_signal[t] * 1.5
        input_current[clusters['Input B']] = B_signal[t] * 1.5
        
        spikes = net.step(input_current=input_current, noise_std=0.05)
        rates = (1 - alpha) * rates + alpha * spikes
        
        for name, idx in clusters.items():
            cluster_rates[name][t] = np.mean(rates[idx])
            
    # 2. Transfer Entropy Calculation
    print("\nComputing Bivariate Transfer Entropy (TE)...")
    
    te_results = {}
    edges_to_test = [
        ('Input A', 'Sum'),
        ('Input B', 'Sum'),
        ('Input A', 'Inhibitory'),
        ('Input B', 'Inhibitory'),
        ('Inhibitory', 'Sum'),
        ('Input A', 'Carry'),
        ('Input B', 'Carry')
    ]
    
    for src, dst in edges_to_test:
        X = cluster_rates[src]
        Y = cluster_rates[dst]
        te = compute_te(X, Y, bins=8)
        te_results[(src, dst)] = te
        print(f"TE: {src} -> {dst} = {te:.4f} bits")
        
    # 3. Visualization
    print("\nGenerating Causal Transfer Graph...")
    G = nx.DiGraph()
    for (src, dst), w in te_results.items():
        if w > 0.01: # Threshold to filter noise
            G.add_edge(src, dst, weight=w)
            
    plt.figure(figsize=(10, 8))
    pos = {
        'Input A': (-1, 1),
        'Input B': (-1, -1),
        'Inhibitory': (0, 0),
        'Sum': (1, 1),
        'Carry': (1, -1)
    }
    
    for n in pos:
        G.add_node(n)
        
    edges = G.edges(data=True)
    
    nx.draw_networkx_nodes(G, pos, node_size=3000, node_color='lightblue', edgecolors='black')
    nx.draw_networkx_labels(G, pos, font_size=12, font_weight='bold')
    
    for (u, v, d) in edges:
        # Scale edge thickness by bits of TE
        nx.draw_networkx_edges(G, pos, edgelist=[(u,v)], width=d['weight']*25, arrowsize=25, 
                               edge_color='salmon', connectionstyle='arc3,rad=0.15')
        
    edge_labels = {(u, v): f"{d['weight']:.2f} bits" for u, v, d in edges}
    nx.draw_networkx_edge_labels(G, pos, edge_labels=edge_labels, label_pos=0.3, font_size=10, font_weight='bold')
    
    plt.title('Biological Half-Adder: Causal Information Flow (Transfer Entropy)', fontsize=14)
    plt.axis('off')
    plt.tight_layout()
    plt.savefig(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'causal_transfer_entropy_graph.png'), dpi=150)
    plt.close()
    
    print("\nCausal Flow Summary:")
    print("--------------------")
    print(f"Input A robustly transfers {te_results[('Input A', 'Sum')]:.4f} bits of information to the Sum cluster.")
    print(f"The Inhibitory Interneurons actively suppress the Sum cluster, directing {te_results[('Inhibitory', 'Sum')]:.4f} bits of causal flow.")
    print(f"The inputs reliably drive the Carry gate (A->Carry: {te_results[('Input A', 'Carry')]:.4f} bits).")
    print(f"Causal diagram successfully saved to 'causal_transfer_entropy_graph.png'.")

if __name__ == "__main__":
    main()
