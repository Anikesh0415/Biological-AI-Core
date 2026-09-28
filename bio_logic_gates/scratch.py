import time
from substrate import BiologicalWetware
import numpy as np

net = BiologicalWetware(num_nodes=2000)
net.weights = net.weights # keep dense for now

t0 = time.time()
for _ in range(100):
    net.run_avalanche([0, 1, 2], max_steps=50, noise_std=0.0)
t1 = time.time()
print(f"100 trials (Dense): {t1-t0:.2f} seconds")

# Test sparse
from scipy import sparse
net.weights = sparse.csr_matrix(net.weights)

t0 = time.time()
for _ in range(100):
    net.run_avalanche([0, 1, 2], max_steps=50, noise_std=0.0)
t1 = time.time()
print(f"100 trials (Sparse): {t1-t0:.2f} seconds")
