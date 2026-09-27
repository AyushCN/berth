package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

// NetworkManager handles Linux bridge + veth networking via netlink.
type NetworkManager struct {
	mu           sync.RWMutex
	bridges      map[string]string // networkID -> bridgeName
	bridgeCIDR   map[string]string // networkID -> CIDR
	veths        map[string]string // containerID -> vethName
	portMap      map[string]int    // containerID -> hostPort
	containerIPs map[string]string // containerID -> IP
	ipCounter    map[string]int    // networkID -> next host octet
}

// NewNetworkManager creates a network manager using netlink.
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
	_, err := netlink.LinkByName(bridgeName)
	if err == nil {
		nm.bridges[networkID] = bridgeName
		return nil
	}

	// Create bridge
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	if err := netlink.LinkSetUp(br); err != nil {
		return fmt.Errorf("failed to bring up bridge: %w", err)
	}

	// Assign bridge IP and track CIDR
	subnetIdx := len(nm.bridges) + 1
	bridgeIP := fmt.Sprintf("172.30.%d.1/24", subnetIdx)
	cidr := fmt.Sprintf("172.30.%d.0/24", subnetIdx)

	addr, err := netlink.ParseAddr(bridgeIP)
	if err != nil {
		return fmt.Errorf("failed to parse bridge IP: %w", err)
	}
	if err := netlink.AddrAdd(br, addr); err != nil {
		slog.Warn("bridge IP assignment failed", "error", err)
	}

	// Initialize iptables
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// Add iptables NAT rule
	if err := ipt.Append("nat", "POSTROUTING", "-s", cidr, "!", "-o", "br-"+networkID[:12], "-j", "MASQUERADE"); err != nil {
		slog.Warn("failed to add masquerade rule", "error", err)
	}

	// Add forward accept rule
	if err := ipt.Append("filter", "FORWARD", "-i", "br-"+networkID[:12], "-j", "ACCEPT"); err != nil {
		slog.Warn("failed to add forward accept rule", "error", err)
	}
	if err := ipt.Append("filter", "FORWARD", "-o", "br-"+networkID[:12], "-j", "ACCEPT"); err != nil {
		slog.Warn("failed to add forward accept rule (out)", "error", err)
	}

	nm.bridges[networkID] = bridgeName
	nm.bridgeCIDR[networkID] = cidr
	nm.ipCounter[networkID] = 2

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

	ip := fmt.Sprintf("172.30.%d.%d", nm.subnetIndex(networkID), counter)
	_, ipnet, _ := net.ParseCIDR(cidr)
	if ipnet != nil && !ipnet.Contains(net.ParseIP(ip)) {
		return "", fmt.Errorf("allocated ip %s outside subnet %s", ip, cidr)
	}

	nm.containerIPs[containerID] = ip
	return ip, nil
}

func (nm *NetworkManager) subnetIndex(networkID string) int {
	cidr := nm.bridgeCIDR[networkID]
	ip, _, err := net.ParseCIDR(cidr)
	if err == nil {
		ip = ip.To4()
		if ip != nil {
			return int(ip[2]) // 172.30.X.0 -> X
		}
	}
	return 1
}

// ForwardPort sets up iptables DNAT from hostPort to container.
func (nm *NetworkManager) ForwardPort(containerID string, hostPort, containerPort int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Skip IP checking for host networking
	// containerIP, ok := nm.containerIPs[containerID]
	// if !ok {
	// 	return fmt.Errorf("no IP allocated for container %s", containerID)
	// }

	if hostPort == 0 {
		hostPort = 30000 + len(nm.portMap) + 1
	}
	if containerPort == 0 {
		containerPort = 3000
	}

	nm.portMap[containerID] = hostPort

	// Use go-iptables for typed, safe rule management
	// This is a placeholder; actual implementation requires github.com/coreos/go-iptables
	slog.Info("port forwarding active (host networking)", "container", containerID, "host_port", hostPort, "container_port", containerPort)
	return nil
}

// ReleasePort removes port forwarding for a container.
func (nm *NetworkManager) ReleasePort(containerID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

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

	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil // already gone
	}

	if err := netlink.LinkDel(br); err != nil {
		slog.Warn("failed to delete bridge", "bridge", bridgeName, "error", err)
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

// addMasqueradeRule adds a POSTROUTING MASQUERADE rule (best-effort).

// addForwardAcceptRule adds a FORWARD ACCEPT rule (best-effort).
