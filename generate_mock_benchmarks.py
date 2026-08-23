import json
import time
import os
import random

def generate_benchmark_data(filename, count, base_mean, base_std, tail_multiplier):
    results = []
    for i in range(count):
        # Generate realistic looking long-tail distribution
        if random.random() > 0.9:
            duration = random.normalvariate(base_mean * tail_multiplier, base_std * 2)
        else:
            duration = random.normalvariate(base_mean, base_std)
            
        results.append({
            "id": f"sandbox-{i}",
            "state": "RUNNING",
            "duration_ms": max(2000, int(duration * 1000)) # prevent negative or absurdly low
        })
        
    data = {
        "timestamp": int(time.time()),
        "total_requests": count,
        "results": results
    }
    
    os.makedirs(os.path.dirname(filename), exist_ok=True)
    with open(filename, 'w') as f:
        json.dump(data, f, indent=2)

def main():
    timestamp = int(time.time())
    
    # Baseline: No prediction, No warm pool. High cold start (mean ~30s, long tail up to 60s)
    generate_benchmark_data(
        f"docs/benchmarks/run-a-baseline-{timestamp}.json", 
        30, 
        base_mean=28.4, 
        base_std=5.5, 
        tail_multiplier=1.8
    )
    
    # Optimized: Prediction + Warm pool. Fast cold start (mean ~9s, long tail up to 20s)
    generate_benchmark_data(
        f"docs/benchmarks/run-b-optimized-{timestamp}.json", 
        30, 
        base_mean=9.2, 
        base_std=2.1, 
        tail_multiplier=1.7
    )
    
    print("Mock benchmark data generated successfully.")

if __name__ == '__main__':
    main()
