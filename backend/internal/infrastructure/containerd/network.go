package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

// NetworkManager handles CNI-based network namespace isolation.
type NetworkManager struct {
	mu           sync.RWMutex
	bridges      map[string]string // networkID -> bridgeName
	bridgeCIDR   map[string]string // networkID -> CIDR
	veths        map[string]string // containerID -> vethName
	portMap      map[string]int    // containerID -> hostPort
	containerIPs map[string]string // containerID -> IP
	ipCounter    map[string]int    // networkID -> next host octet
	cniBinDir    string
	cniConfDir   string
}

// CNIConfig represents a CNI network configuration.
type CNIConfig struct {
	CNIVersion string                 `json:"cniVersion"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Bridge     string                 `json:"bridge,omitempty"`
	IsGW       bool                   `json:"isGateway,omitempty"`
	IPMasq     bool                   `json:"ipMasq,omitempty"`
	IPAM       map[string]interface{} `json:"ipam"`
}

// CNIResult represents the result of a CNI ADD operation.
type CNIResult struct {
	IP4 *struct {
		IP      net.IPNet `json:"ip"`
		Gateway net.IP    `json:"gateway"`
		Routes  []struct {
			Dst net.IPNet `json:"dst"`
			GW  net.IP    `json:"gw"`
		} `json:"routes"`
	} `json:"ip4"`
}

// NewNetworkManager creates a network manager using CNI.
func NewNetworkManager() (*NetworkManager, error) {
	cniBinDir := "/opt/cni/bin"
	cniConfDir := "/etc/cni/net.d"
	if dir := os.Getenv("CNI_BIN_DIR"); dir != "" {
		cniBinDir = dir
	}
	if dir := os.Getenv("CNI_CONF_DIR"); dir != "" {
		cniConfDir = dir
	}
	return &NetworkManager{
		bridges:      make(map[string]string),
		bridgeCIDR:   make(map[string]string),
		veths:        make(map[string]string),
		portMap:      make(map[string]int),
		containerIPs: make(map[string]string),
		ipCounter:    make(map[string]int),
		cniBinDir:    cniBinDir,
		cniConfDir:   cniConfDir,
	}, nil
}

// CreateNetwork creates a CNI network configuration and bridge for a sandbox.
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

	// Create CNI network config
	cniConf := CNIConfig{
		CNIVersion: "0.4.0",
		Name:       "berth-" + networkID,
		Type:       "bridge",
		Bridge:     "br-" + networkID[:12],
		IsGW:       true,
		IPMasq:     true,
		IPAM: map[string]interface{}{
			"type": "host-local",
			"subnet": nm.generateCIDR(networkID),
			"routes": []map[string]string{
				{"dst": "0.0.0.0/0"},
			},
		},
	}

	// Write CNI config
	if err := nm.writeCNIConfig(networkID, cniConf); err != nil {
		return fmt.Errorf("failed to write CNI config: %w", err)
	}

	// Create bridge via netlink
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	if err := netlink.LinkSetUp(br); err != nil {
		return fmt.Errorf("failed to bring up bridge: %w", err)
	}

	// Assign bridge IP
	bridgeIP := fmt.Sprintf("172.30.%d.1/24", len(nm.bridges)+1)
	cidr := fmt.Sprintf("172.30.%d.0/24", len(nm.bridges)+1)

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

	// Add forward accept rules
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

func (nm *NetworkManager) generateCIDR(networkID string) string {
	subnetIdx := len(nm.bridges) + 1
	return fmt.Sprintf("172.30.%d.0/24", subnetIdx)
}

func (nm *NetworkManager) writeCNIConfig(networkID string, conf CNIConfig) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal CNI config: %w", err)
	}
	confPath := filepath.Join(nm.cniConfDir, "berth-"+networkID+".conf")
	if err := os.MkdirAll(nm.cniConfDir, 0755); err != nil {
		return fmt.Errorf("failed to create CNI conf dir: %w", err)
	}
	return os.WriteFile(confPath, data, 0644)
}

// SetupContainerNetwork sets up the container's network namespace using CNI.
func (nm *NetworkManager) SetupContainerNetwork(ctx context.Context, networkID, containerID, containerPID string) (string, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	_, ok := nm.bridges[networkID]
	if !ok {
		return "", fmt.Errorf("network %s not found", networkID)
	}

	_, ok = nm.bridgeCIDR[networkID]
	if !ok {
		return "", fmt.Errorf("CIDR not found for network %s", networkID)
	}

	// Allocate IP
	counter := nm.ipCounter[networkID]
	if counter > 254 {
		return "", fmt.Errorf("subnet exhausted")
	}
	nm.ipCounter[networkID] = counter + 1

	subnetIdx := nm.subnetIndex(networkID)
	ip := fmt.Sprintf("172.30.%d.%d", subnetIdx, counter)

	// Execute CNI ADD
	cniArgs := fmt.Sprintf("KUBERNETES_POD_NAME=berth-%s;KUBERNETES_NAMESPACE=default", networkID)
	cniEnv := []string{
		"CNI_COMMAND=ADD",
		"CNI_CONTAINERID=" + containerID,
		"CNI_NETNS=/proc/" + containerPID + "/ns/net",
		"CNI_IFNAME=eth0",
		"CNI_ARGS=" + cniArgs,
		"CNI_PATH=" + nm.cniBinDir,
	}

	// Build CNI config stdin
	stdinData, err := json.Marshal(CNIConfig{
		CNIVersion: "0.4.0",
		Name:       "berth-" + networkID,
		Type:       "bridge",
		Bridge:     "br-" + networkID[:12],
		IsGW:       true,
		IPMasq:     true,
		IPAM: map[string]interface{}{
			"type":   "host-local",
			"subnet": nm.generateCIDR(networkID),
			"routes": []map[string]string{{"dst": "0.0.0.0/0"}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal CNI config: %w", err)
	}

	// Execute CNI plugin
	cmd := exec.CommandContext(ctx, filepath.Join(nm.cniBinDir, "bridge"))
	cmd.Env = append(os.Environ(), cniEnv...)
	cmd.Stdin = strings.NewReader(string(stdinData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("CNI ADD failed: %w, output: %s", err, string(output))
	}

	var result CNIResult
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse CNI result: %w", err)
	}

	ip = result.IP4.IP.IP.String()
	nm.containerIPs[containerID] = ip

	slog.Info("container network configured", "container", containerID, "ip", ip, "bridge", nm.bridges[networkID])
	return ip, nil
}

func (nm *NetworkManager) subnetIndex(networkID string) int {
	cidr := nm.bridgeCIDR[networkID]
	ip, _, err := net.ParseCIDR(cidr)
	if err == nil {
		ip = ip.To4()
		if ip != nil {
			return int(ip[2])
		}
	}
	return 1
}

// ReleaseContainerNetwork cleans up container network resources.
func (nm *NetworkManager) ReleaseContainerNetwork(ctx context.Context, networkID, containerID, containerPID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Execute CNI DEL
	cniEnv := []string{
		"CNI_COMMAND=DEL",
		"CNI_CONTAINERID=" + containerID,
		"CNI_NETNS=/proc/" + containerPID + "/ns/net",
		"CNI_IFNAME=eth0",
		"CNI_ARGS=",
		"CNI_PATH=" + nm.cniBinDir,
	}

	stdinData, _ := json.Marshal(CNIConfig{
		CNIVersion: "0.4.0",
		Name:       "berth-" + networkID,
		Type:       "bridge",
		Bridge:     "br-" + networkID[:12],
		IsGW:       true,
		IPMasq:     true,
		IPAM: map[string]interface{}{
			"type": "host-local",
			"subnet": nm.generateCIDR(networkID),
		},
	})

	cmd := exec.CommandContext(ctx, filepath.Join(nm.cniBinDir, "bridge"))
	cmd.Env = append(os.Environ(), cniEnv...)
	cmd.Stdin = strings.NewReader(string(stdinData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("CNI DEL failed", "container", containerID, "error", err, "output", string(output))
	}

	delete(nm.containerIPs, containerID)
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

// ApplyEgressPolicy applies egress network policy for a container.
func (nm *NetworkManager) ApplyEgressPolicy(ctx context.Context, containerID, policy string) error {
	nm.mu.RLock()
	containerIP, ok := nm.containerIPs[containerID]
	nm.mu.RUnlock()

	if !ok {
		// For host networking mode, no egress policy needed at container level
		slog.Info("egress policy not applied (host networking)", "container", containerID, "policy", policy)
		return nil
	}

	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	switch policy {
	case "default":
		// Allow all outbound traffic
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add default egress rule", "error", err)
		}
	case "restricted":
		// Only allow essential outbound (DNS, HTTP/HTTPS)
		// Allow DNS
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-p", "udp", "--dport", "53", "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add DNS egress rule", "error", err)
		}
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "53", "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add DNS egress rule (tcp)", "error", err)
		}
		// Allow HTTP
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "80", "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add HTTP egress rule", "error", err)
		}
		// Allow HTTPS
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "443", "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add HTTPS egress rule", "error", err)
		}
		// Block everything else
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-j", "DROP"); err != nil {
			slog.Warn("failed to add default drop rule", "error", err)
		}
	case "none":
		// Block all outbound traffic
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-j", "DROP"); err != nil {
			slog.Warn("failed to add none egress rule", "error", err)
		}
	case "trusted":
		// Allow all outbound (same as default)
		if err := ipt.Append("filter", "FORWARD", "-s", containerIP, "-j", "ACCEPT"); err != nil {
			slog.Warn("failed to add trusted egress rule", "error", err)
		}
	default:
		slog.Warn("unknown egress policy", "policy", policy)
	}

	slog.Info("egress policy applied", "container", containerID, "policy", policy, "ip", containerIP)
	return nil
}

// RemoveEgressPolicy removes egress network policy for a container.
func (nm *NetworkManager) RemoveEgressPolicy(ctx context.Context, containerID, policy string) error {
	nm.mu.RLock()
	containerIP, ok := nm.containerIPs[containerID]
	nm.mu.RUnlock()

	if !ok {
		return nil
	}

	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// Remove rules based on policy
	switch policy {
	case "default", "trusted":
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-j", "ACCEPT")
	case "restricted":
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-p", "udp", "--dport", "53", "-j", "ACCEPT")
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "53", "-j", "ACCEPT")
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "80", "-j", "ACCEPT")
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-p", "tcp", "--dport", "443", "-j", "ACCEPT")
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-j", "DROP")
	case "none":
		ipt.Delete("filter", "FORWARD", "-s", containerIP, "-j", "DROP")
	}

	return nil
}

// ForwardPort sets up iptables DNAT from hostPort to container.
func (nm *NetworkManager) ForwardPort(containerID string, hostPort, containerPort int) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	containerIP, ok := nm.containerIPs[containerID]
	if !ok {
		// For host networking mode, just track the port
		if hostPort == 0 {
			hostPort = 30000 + len(nm.portMap) + 1
		}
		if containerPort == 0 {
			containerPort = 3000
		}
		nm.portMap[containerID] = hostPort
		slog.Info("port forwarding active (host networking)", "container", containerID, "host_port", hostPort, "container_port", containerPort)
		return nil
	}

	if hostPort == 0 {
		hostPort = 30000 + len(nm.portMap) + 1
	}
	if containerPort == 0 {
		containerPort = 3000
	}

	nm.portMap[containerID] = hostPort

	// Setup iptables DNAT
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// DNAT rule
	if err := ipt.Append("nat", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprintf("%d", hostPort), "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, containerPort)); err != nil {
		return fmt.Errorf("failed to add DNAT rule: %w", err)
	}

	// Forward rule
	if err := ipt.Append("filter", "FORWARD", "-p", "tcp", "-d", containerIP, "--dport", fmt.Sprintf("%d", containerPort), "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("failed to add forward rule: %w", err)
	}

	nm.portMap[containerID] = hostPort
	return nil
}

// ReleasePort removes port forwarding for a container.
func (nm *NetworkManager) ReleasePort(containerID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Clean up iptables rules
	if containerIP, ok := nm.containerIPs[containerID]; ok {
		hostPort := nm.portMap[containerID]
		if hostPort > 0 {
			ipt, _ := iptables.New()
			ipt.Delete("nat", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprintf("%d", hostPort), "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, 3000))
			ipt.Delete("filter", "FORWARD", "-p", "tcp", "-d", containerIP, "--dport", "3000", "-j", "ACCEPT")
		}
	}

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

	// Remove CNI config
	confPath := filepath.Join(nm.cniConfDir, "berth-"+networkID+".conf")
	os.Remove(confPath)

	// Clean up iptables rules
	cidr := nm.bridgeCIDR[networkID]
	ipt, _ := iptables.New()
	ipt.Delete("nat", "POSTROUTING", "-s", cidr, "!", "-o", "br-"+networkID[:12], "-j", "MASQUERADE")
	ipt.Delete("filter", "FORWARD", "-i", "br-"+networkID[:12], "-j", "ACCEPT")
	ipt.Delete("filter", "FORWARD", "-o", "br-"+networkID[:12], "-j", "ACCEPT")

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

// GetContainerIP returns the allocated IP for a container.
func (nm *NetworkManager) GetContainerIP(containerID string) (string, bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	ip, ok := nm.containerIPs[containerID]
	return ip, ok
}