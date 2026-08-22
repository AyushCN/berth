package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

// NetworkManager handles Linux bridge + veth networking for sandboxes.
// Phase 1 uses shell commands (ip, iptables). Phase 2 will use netlink + Cilium.
type NetworkManager struct {
	mu       sync.RWMutex
	bridges  map[string]string // networkID -> bridgeName
	veths    map[string]string // containerID -> vethName
	portMap  map[string]int    // containerID -> hostPort
}

// NewNetworkManager creates a network manager.
func NewNetworkManager() (*NetworkManager, error) {
	return &NetworkManager{
		bridges: make(map[string]string),
		veths:   make(map[string]string),
		portMap: make(map[string]int),
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

	// Assign bridge IP (172.30.x.1/24)
	bridgeIP := fmt.Sprintf("172.30.%d.1/24", len(nm.bridges)+1)
	if out, err := exec.CommandContext(ctx, "ip", "addr", "add", bridgeIP, "dev", bridgeName).CombinedOutput(); err != nil {
		// IP might already exist, ignore
		slog.Warn("bridge IP assignment", "error", err, "output", string(out))
	}

	nm.bridges[networkID] = bridgeName
	slog.Info("network created", "bridge", bridgeName, "ip", bridgeIP)
	return nil
}

// ForwardPort sets up iptables DNAT from hostPort to container.
// In Phase 1, we use a simple port-forward: allocate a host port and
// forward all traffic to the container's exposed port.
func (nm *NetworkManager) ForwardPort(containerID string, hostPort, containerPort int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if hostPort == 0 {
		hostPort = 30000 + len(nm.portMap) + 1 // auto-allocate from ephemeral range
	}
	if containerPort == 0 {
		containerPort = 3000 // default
	}

	nm.portMap[containerID] = hostPort

	// Use iptables-legacy or nftables depending on host
	// Phase 1: simple REDIRECT to localhost proxy (simpler than full bridge routing)
	// We run a tiny userspace proxy or use iptables DNAT

	// For now, just record the mapping. The actual forwarding will be done
	// by a separate proxy service (Phase 1.5) or iptables (Phase 2).
	slog.Info("port mapping recorded", "container", containerID, "host", hostPort, "container_port", containerPort)
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

	if out, err := exec.CommandContext(ctx, "ip", "link", "delete", bridgeName).CombinedOutput(); err != nil {
		slog.Warn("failed to delete bridge", "bridge", bridgeName, "error", err, "output", string(out))
	}

	delete(nm.bridges, networkID)
	slog.Info("network destroyed", "bridge", bridgeName)
	return nil
}

// GetHostPort returns the allocated host port for a container.
func (nm *NetworkManager) GetHostPort(containerID string) int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.portMap[containerID]
}
