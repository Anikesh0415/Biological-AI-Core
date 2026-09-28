import json
import numpy as np
import networkx as nx
from scipy.spatial import KDTree
from astroquery.sdss import SDSS
from astropy.cosmology import FlatLambdaCDM

def main():
    print("Querying SDSS for local galaxies (0.02 < z < 0.03)...")
    query = """
    SELECT TOP 1000 specObjID as objID, ra, dec, z
    FROM SpecObj
    WHERE class = 'GALAXY'
      AND z BETWEEN 0.02 AND 0.03
    """
    res = SDSS.query_sql(query)
    print(f"Retrieved {len(res)} galaxies.")

    # Convert coordinates to 3D Cartesian
    print("Converting to 3D Cartesian Mpc coordinates...")
    cosmo = FlatLambdaCDM(H0=70, Om0=0.3)
    
    # Comoving distance
    D_c = cosmo.comoving_distance(res['z']).value
    
    # Degrees to radians
    ra_rad = np.deg2rad(res['ra'])
    dec_rad = np.deg2rad(res['dec'])
    
    X = D_c * np.cos(dec_rad) * np.cos(ra_rad)
    Y = D_c * np.cos(dec_rad) * np.sin(ra_rad)
    Z = D_c * np.sin(dec_rad)

    coords = np.column_stack((X, Y, Z))
    
    # Build spatial proximity network
    linking_length = 4.0 # Mpc
    print(f"Building KDTree and querying pairs within {linking_length} Mpc...")
    tree = KDTree(coords)
    pairs = tree.query_pairs(r=linking_length)
    
    G = nx.Graph()
    for i in range(len(res)):
        G.add_node(str(i), x=X[i], y=Y[i], z=Z[i])
        
    for (u, v) in pairs:
        G.add_edge(str(u), str(v))
    
    # Remove isolated nodes
    isolated = list(nx.isolates(G))
    G.remove_nodes_from(isolated)
    
    # Compute metrics
    active_nodes = G.number_of_nodes()
    total_edges = G.number_of_edges()
    
    if active_nodes > 0:
        avg_degree = sum(dict(G.degree()).values()) / active_nodes
        avg_clust = nx.average_clustering(G)
    else:
        avg_degree = 0.0
        avg_clust = 0.0
        
    print("\n--- Cosmic Web Network Metrics ---")
    print(f"Total number of active nodes: {active_nodes}")
    print(f"Total number of edges: {total_edges}")
    print(f"Average Node Degree: {avg_degree:.4f}")
    print(f"Average Clustering Coefficient: {avg_clust:.4f}")
    
    # Export JSON
    nodes_out = [{"id": n, "x": float(d['x']), "y": float(d['y']), "z": float(d['z'])} for n, d in G.nodes(data=True)]
    edges_out = [{"source": u, "target": v} for u, v in G.edges()]

    output_data = {
      "metrics": {
        "average_degree": avg_degree,
        "clustering_coefficient": avg_clust,
        "network_type": "Cosmic Web (SDSS)"
      },
      "nodes": nodes_out,
      "edges": edges_out
    }
    
    out_path = "d:/Neuro-pro/cosmic_network_data.json"
    with open(out_path, "w") as f:
        json.dump(output_data, f, indent=2)
        
    print(f"\nSuccessfully exported data to {out_path}")

if __name__ == "__main__":
    main()
