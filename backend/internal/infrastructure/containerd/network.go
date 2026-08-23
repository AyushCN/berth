package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
)

// NetworkManager handles Linux bridge + veth networking for sandboxes.
// Phase 1 uses shell commands (ip, iptables). Phase 2 will use netlink + Cilium.
type NetworkManager struct {
	mu           sync.RWMutex
	bridges      map[string]string // networkID -> bridgeName
	bridgeCIDR   map[string]string // networkID -> CIDR
	veths        map[string]string // containerID -> vethName
	portMap      map[string]int    // containerID -> hostPort
	containerIPs map[string]string // containerID -> IP
	ipCounter    map[string]int    // networkID -> next host octet
}

// NewNetworkManager creates a network manager.
func NewNetworkManager() (*NetworkManager, error) {
	return &NetworkManager{
		bridges:      make(map[string]string),
		bridgeCIDR:   make(map[string]string),
		veths:        make(map[string]string),
		portMap:      make(map[string]int),
		containerIPs: make(map[string]string),
		ipCounter:    make(map[string]int),
	}, nil
}

// CreateNetwork creates a Linux bridge for a sandbox network namespace.
func (nm *NetworkManager) CreateNetwork(ctx context.Context, networkID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	bridgeName := "br-" + networkID[:12]

	// Check if bridge exists
	if out, err := exec.CommandContext(ctx, "ip", "link", "show", bridgeName).CombinedOutput(); err == nil {
		if strings.Contains(string(out), bridgeName) {
			nm.bridges[networkID] = bridgeName
			return nil
		}
	}

	// Create bridge
	if out, err := exec.CommandContext(ctx, "ip", "link", "add", bridgeName, "type", "bridge").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create bridge: %w (output: %s)", err, string(out))
	}
	if out, err := exec.CommandContext(ctx, "ip", "link", "set", bridgeName, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring up bridge: %w (output: %s)", err, string(out))
	}

	// Assign bridge IP and track CIDR
	subnetIdx := len(nm.bridges) + 1
	bridgeIP := fmt.Sprintf("172.30.%d.1/24", subnetIdx)
	cidr := fmt.Sprintf("172.30.%d.0/24", subnetIdx)
	if out, err := exec.CommandContext(ctx, "ip", "addr", "add", bridgeIP, "dev", bridgeName).CombinedOutput(); err != nil {
		slog.Warn("bridge IP assignment", "error", err, "output", string(out))
	}

	// Enable IP forwarding and NAT for the bridge subnet
	if out, err := exec.CommandContext(ctx, "iptables", "-t", "nat", "-A", "POSTROUTING", "-s", cidr, "!", "-o", bridgeName, "-j", "MASQUERADE").CombinedOutput(); err != nil {
		slog.Warn("iptables masquerade setup failed", "error", err, "output", string(out))
	}
	if out, err := exec.CommandContext(ctx, "iptables", "-A", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").CombinedOutput(); err != nil {
		slog.Warn("iptables forward accept failed", "error", err, "output", string(out))
	}

	nm.bridges[networkID] = bridgeName
	nm.bridgeCIDR[networkID] = cidr
	nm.ipCounter[networkID] = 2 // start allocating from .2

	slog.Info("network created", "bridge", bridgeName, "ip", bridgeIP, "cidr", cidr)
	return nil
}

// AllocateIP assigns an IP address to a container from the bridge subnet.
func (nm *NetworkManager) AllocateIP(networkID, containerID string) (string, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	cidr, ok := nm.bridgeCIDR[networkID]
	if !ok {
		return "", fmt.Errorf("network %s not found", networkID)
	}

	counter := nm.ipCounter[networkID]
	if counter > 254 {
		return "", fmt.Errorf("subnet %s exhausted", cidr)
	}
	nm.ipCounter[networkID] = counter + 1

	// Derive subnet index from CIDR (172.30.X.0/24)
	_, ipnet, _ := net.ParseCIDR(cidr)
	ip := fmt.Sprintf("172.30.%d.%d", nm.subnetIndex(networkID), counter)
	if ipnet != nil && !ipnet.Contains(net.ParseIP(ip)) {
		return "", fmt.Errorf("allocated ip %s outside subnet %s", ip, cidr)
	}

	nm.containerIPs[containerID] = ip
	return ip, nil
}

func (nm *NetworkManager) subnetIndex(networkID string) int {
	cidr := nm.bridgeCIDR[networkID]
	parts := strings.Split(cidr, ".")
	if len(parts) >= 3 {
		var idx int
		fmt.Sscanf(parts[2], "%d", &idx)
		return idx
	}
	return 1
}

// ForwardPort sets up iptables DNAT from hostPort to container.
func (nm *NetworkManager) ForwardPort(containerID string, hostPort, containerPort int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	containerIP, ok := nm.containerIPs[containerID]
	if !ok {
		return fmt.Errorf("no IP allocated for container %s", containerID)
	}

	if hostPort == 0 {
		hostPort = 30000 + len(nm.portMap) + 1
	}
	if containerPort == 0 {
		containerPort = 3000
	}

	nm.portMap[containerID] = hostPort

	// iptables DNAT: hostPort -> containerIP:containerPort
	dst := fmt.Sprintf("%s:%d", containerIP, containerPort)
	if out, err := exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprint(hostPort), "-j", "DNAT", "--to-destination", dst).CombinedOutput(); err != nil {
		slog.Error("iptables DNAT failed", "error", err, "output", string(out))
		return fmt.Errorf("failed to add DNAT rule: %w", err)
	}

	slog.Info("port forwarding active", "container", containerID, "host_port", hostPort, "destination", dst)
	return nil
}

// ReleasePort removes iptables DNAT rules for a container.
func (nm *NetworkManager) ReleasePort(containerID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	hostPort, ok := nm.portMap[containerID]
	if !ok {
		return nil
	}

	containerIP := nm.containerIPs[containerID]
	dst := fmt.Sprintf("%s:3000", containerIP) // default container port

	// Best-effort cleanup
	_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprint(hostPort), "-j", "DNAT", "--to-destination", dst).Run()

	delete(nm.portMap, containerID)
	delete(nm.containerIPs, containerID)
	return nil
}

// DestroyNetwork removes a bridge and associated rules.
func (nm *NetworkManager) DestroyNetwork(ctx context.Context, networkID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	bridgeName, ok := nm.bridges[networkID]
	if !ok {
		return nil
	}

	cidr := nm.bridgeCIDR[networkID]

	// Clean up iptables rules for this bridge
	_ = exec.CommandContext(ctx, "iptables", "-t", "nat", "-D", "POSTROUTING", "-s", cidr, "!", "-o", bridgeName, "-j", "MASQUERADE").Run()
	_ = exec.CommandContext(ctx, "iptables", "-D", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").Run()

	if out, err := exec.CommandContext(ctx, "ip", "link", "delete", bridgeName).CombinedOutput(); err != nil {
		slog.Warn("failed to delete bridge", "bridge", bridgeName, "error", err, "output", string(out))
	}

	delete(nm.bridges, networkID)
	delete(nm.bridgeCIDR, networkID)
	delete(nm.ipCounter, networkID)
	slog.Info("network destroyed", "bridge", bridgeName)
	return nil
}

// GetHostPort returns the allocated host port for a container.
func (nm *NetworkManager) GetHostPort(containerID string) int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.portMap[containerID]
}
