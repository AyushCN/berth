package crosshost

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

// VXLANMesh manages cross-host VXLAN overlay networking
type VXLANMesh struct {
	mu           sync.RWMutex
	vxlanIF      string
	vxlanID      int
	localIP      string
	vtepMAC      string
	peers        map[string]*VXLANPeer // peerIP -> peer info
	arpTable     map[string]string     // podIP -> peerIP
	fdbTable     map[string]string     // mac -> peerIP
	netlinkHandle *netlink.Handle
}

type VXLANPeer struct {
	IP        string    `json:"ip"`
	Hostname  string    `json:"hostname"`
	LastSeen  time.Time `json:"last_seen"`
	Healthy   bool      `json:"healthy"`
	VTEPMAC   string    `json:"vtep_mac"`
	PodCIDRs  []string  `json:"pod_cidrs"` // Pod CIDRs hosted on this peer
}

type VXLANConfig struct {
	VXLANID      int      `json:"vxlan_id"`
	VXLANInterface string  `json:"vxlan_interface"`
	Port         int      `json:"port"`          // VXLAN port (default 4789)
	LocalIP      string   `json:"local_ip"`      // Local VTEP IP
	MTU          int      `json:"mtu"`           // MTU for VXLAN interface
	Peers        []string `json:"peers"`         // Initial peer IPs
	PodCIDR      string   `json:"pod_cidr"`      // This node's pod CIDR
}

const (
	DefaultVXLANPort = 4789
	DefaultVXLANID   = 42
	DefaultMTU       = 1450 // Account for VXLAN overhead
	ARPTimeout       = 5 * time.Minute
	PeerCheckInterval = 10 * time.Second
)

func NewVXLANMesh(config VXLANConfig) (*VXLANMesh, error) {
	if config.VXLANID == 0 {
		config.VXLANID = DefaultVXLANID
	}
	if config.Port == 0 {
		config.Port = DefaultVXLANPort
	}
	if config.MTU == 0 {
		config.MTU = DefaultMTU
	}
	if config.VXLANInterface == "" {
		config.VXLANInterface = fmt.Sprintf("vxlan%d", config.VXLANID)
	}

	nlHandle, err := netlink.NewHandle()
	if err != nil {
		return nil, fmt.Errorf("failed to create netlink handle: %w", err)
	}

	vm := &VXLANMesh{
		vxlanIF:       config.VXLANInterface,
		vxlanID:       config.VXLANID,
		localIP:       config.LocalIP,
		peers:         make(map[string]*VXLANPeer),
		arpTable:      make(map[string]string),
		fdbTable:      make(map[string]string),
		netlinkHandle: nlHandle,
	}

	// Create VXLAN interface
	if err := vm.createVXLANInterface(config); err != nil {
		return nil, err
	}

	// Add initial peers
	for _, peerIP := range config.Peers {
		vm.AddPeer(peerIP, "", []string{})
	}

	// Start background loops
	go vm.arpCleanupLoop()
	go vm.peerHealthCheckLoop()

	slog.Info("VXLAN mesh initialized", "interface", config.VXLANInterface, "vni", config.VXLANID, "local_ip", config.LocalIP)
	return vm, nil
}

func (vm *VXLANMesh) createVXLANInterface(config VXLANConfig) error {
	// Check if interface already exists
	_, err := netlink.LinkByName(config.VXLANInterface)
	if err == nil {
		slog.Info("VXLAN interface already exists", "interface", config.VXLANInterface)
		return nil
	}

	// Create VXLAN device
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: config.VXLANInterface,
			MTU:  config.MTU,
		},
		VxlanId:      config.VXLANID,
		SrcAddr:      net.ParseIP(config.LocalIP),
		Port:         config.Port,
		Learning:     true,
		Proxy:        true,   // Enable ARP proxy
		L2miss:       true,   // Send L2 miss notifications
		L3miss:       true,   // Send L3 miss notifications
	}

	if err := vm.netlinkHandle.LinkAdd(vxlan); err != nil {
		return fmt.Errorf("failed to create VXLAN interface: %w", err)
	}

	// Bring interface up
	if err := vm.netlinkHandle.LinkSetUp(vxlan); err != nil {
		return fmt.Errorf("failed to bring up VXLAN interface: %w", err)
	}

	// Add local IP to VXLAN interface
	addr, err := netlink.ParseAddr(config.LocalIP + "/24")
	if err != nil {
		return fmt.Errorf("failed to parse local IP: %w", err)
	}
	if err := vm.netlinkHandle.AddrAdd(vxlan, addr); err != nil {
		slog.Warn("failed to add IP to VXLAN interface", "error", err)
	}

	// Enable ARP proxy on the VXLAN interface
	vm.enableARPProxy(config.VXLANInterface)

	slog.Info("VXLAN interface created", "interface", config.VXLANInterface)
	return nil
}

func (vm *VXLANMesh) enableARPProxy(iface string) {
	// Enable proxy ARP on the interface
	exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.proxy_arp=1", iface)).Run()
	exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.forwarding=1", iface)).Run()
	// Enable on all interfaces for VXLAN
	exec.Command("sysctl", "-w", "net.ipv4.conf.all.proxy_arp=1").Run()
	exec.Command("sysctl", "-w", "net.ipv4.conf.all.forwarding=1").Run()
	exec.Command("sysctl", "-w", "net.ipv4.neigh.default.proxy_qlen=128").Run()
}

func (vm *VXLANMesh) AddPeer(peerIP, hostname string, podCIDRs []string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.peers[peerIP]; exists {
		// Update existing peer
		vm.peers[peerIP].LastSeen = time.Now()
		vm.peers[peerIP].Healthy = true
		vm.peers[peerIP].PodCIDRs = podCIDRs
		return nil
	}

	// Add FDB entry for the peer (all MACs via this VTEP)
	// In practice, we'd learn MACs dynamically, but we can pre-populate
	vm.peers[peerIP] = &VXLANPeer{
		IP:       peerIP,
		Hostname: hostname,
		LastSeen: time.Now(),
		Healthy:  true,
		PodCIDRs: podCIDRs,
	}

	// Add FDB entry for broadcast/unknown unicast
	vm.addFDBEntry(peerIP, "00:00:00:00:00:00")

	slog.Info("VXLAN peer added", "peer_ip", peerIP, "hostname", hostname)
	return nil
}

func (vm *VXLANMesh) RemovePeer(peerIP string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.peers[peerIP]; !exists {
		return nil
	}

	// Remove FDB entries
	vm.removeFDBEntriesForPeer(peerIP)

	// Remove ARP entries pointing to this peer
	for podIP, pIP := range vm.arpTable {
		if pIP == peerIP {
			delete(vm.arpTable, podIP)
		}
	}

	delete(vm.peers, peerIP)
	slog.Info("VXLAN peer removed", "peer_ip", peerIP)
	return nil
}

func (vm *VXLANMesh) addFDBEntry(peerIP, mac string) error {
	// Add FDB entry: mac -> peerIP via VXLAN
	// This tells the kernel to send frames for this MAC to the peer VTEP
	link, err := netlink.LinkByName(vm.vxlanIF)
	if err != nil {
		return err
	}

	vxlan := link.(*netlink.Vxlan)
	
	// Parse MAC
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}

	neigh := &netlink.Neigh{
		LinkIndex:    vxlan.Index,
		Family:       801, // AF_VXLAN
		State:        netlink.NUD_PERMANENT,
		IP:           net.ParseIP(peerIP),
		HardwareAddr: hwAddr,
	}

	return vm.netlinkHandle.NeighAdd(neigh)
}

func (vm *VXLANMesh) removeFDBEntriesForPeer(peerIP string) {
	// Remove all FDB entries for this peer
	link, err := netlink.LinkByName(vm.vxlanIF)
	if err != nil {
		return
	}

	vxlan := link.(*netlink.Vxlan)
	
	// Delete neighbor entries for this peer
	neighs, _ := vm.netlinkHandle.NeighList(vxlan.Index, 801) // AF_VXLAN
	for _, n := range neighs {
		if n.IP.Equal(net.ParseIP(peerIP)) {
			vm.netlinkHandle.NeighDel(&n)
		}
	}
}

func (vm *VXLANMesh) LearnARP(podIP, peerIP, mac string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.arpTable[podIP] = peerIP
	if mac != "" {
		vm.fdbTable[mac] = peerIP
		vm.addFDBEntry(peerIP, mac)
	}
}

func (vm *VXLANMesh) GetPeerForPod(podIP string) (string, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	peerIP, ok := vm.arpTable[podIP]
	return peerIP, ok
}

func (vm *VXLANMesh) arpCleanupLoop() {
	ticker := time.NewTicker(ARPTimeout / 2)
	defer ticker.Stop()

	for range ticker.C {
		vm.mu.Lock()
		now := time.Now()
		for podIP, peerIP := range vm.arpTable {
			if peer, ok := vm.peers[peerIP]; ok && now.Sub(peer.LastSeen) > ARPTimeout {
				delete(vm.arpTable, podIP)
				slog.Debug("ARP entry expired", "pod_ip", podIP, "peer_ip", peerIP)
			}
		}
		vm.mu.Unlock()
	}
}

func (vm *VXLANMesh) peerHealthCheckLoop() {
	ticker := time.NewTicker(PeerCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		vm.mu.Lock()
		now := time.Now()
		for peerIP, peer := range vm.peers {
			if now.Sub(peer.LastSeen) > PeerCheckInterval*3 {
				if peer.Healthy {
					peer.Healthy = false
					slog.Warn("VXLAN peer unhealthy", "peer_ip", peerIP)
				}
			}
		}
		vm.mu.Unlock()
	}
}

// UpdatePeerLastSeen updates the last seen time for a peer (called when receiving packets)
func (vm *VXLANMesh) UpdatePeerLastSeen(peerIP string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if peer, ok := vm.peers[peerIP]; ok {
		peer.LastSeen = time.Now()
		peer.Healthy = true
	}
}

// GetPeerList returns all known peers
func (vm *VXLANMesh) GetPeerList() []*VXLANPeer {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	peers := make([]*VXLANPeer, 0, len(vm.peers))
	for _, p := range vm.peers {
		peers = append(peers, p)
	}
	return peers
}

// Close cleans up the VXLAN interface
func (vm *VXLANMesh) Close() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	link, err := netlink.LinkByName(vm.vxlanIF)
	if err != nil {
		return nil // Already gone
	}

	return vm.netlinkHandle.LinkDel(link)
}