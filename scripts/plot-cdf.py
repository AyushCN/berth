import json, sys, matplotlib.pyplot as plt

def load(path):
    with open(path) as f:
        data = json.load(f)
    return [r['duration_ms']/1000 for r in data['results'] if r['state']=='RUNNING']

a = sorted(load(sys.argv[1]))
b = sorted(load(sys.argv[2]))
ya = [i/len(a) for i in range(len(a))]
yb = [i/len(b) for i in range(len(b))]

plt.plot(a, ya, label='Baseline', linewidth=2)
plt.plot(b, yb, label='Berth', linewidth=2)
plt.xlabel('Cold-start latency (seconds)')
plt.ylabel('Cumulative Distribution')
plt.title('Cold-start Latency CDF')
plt.legend()
plt.grid(True, alpha=0.3)
plt.savefig('docs/PAPER/figures/coldstart-cdf.pdf', bbox_inches='tight')
plt.savefig('docs/PAPER/figures/coldstart-cdf.png', dpi=300, bbox_inches='tight')
print("Saved CDF plot")
