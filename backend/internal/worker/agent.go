package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// WorkerAgent runs on each compute node and registers with the control plane.
type WorkerAgent struct {
	worker         *domain.Worker
	controlPlane   ControlPlaneClient
	containerdSock string
	runtime        domain.ContainerRuntime
	mu             sync.RWMutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// ControlPlaneClient defines the interface for communicating with the control plane.
type ControlPlaneClient interface {
	RegisterWorker(ctx context.Context, worker *domain.Worker) error
	Heartbeat(ctx context.Context, workerID uuid.UUID, capacity *domain.WorkerCapacity) error
	GetSchedulerConfig(ctx context.Context) (*domain.SchedulerConfig, error)
	ReportSandboxStatus(ctx context.Context, sandboxID uuid.UUID, status string) error
}

func NewWorkerAgent(controlPlane ControlPlaneClient, containerdSock string, containerRuntime domain.ContainerRuntime) *WorkerAgent {
	hostname, _ := os.Hostname()

	// Detect resources
	totalMem := getTotalMemory()
	totalCPU := int64(runtime.NumCPU()) * 1000 // milli-cores
	totalDisk := getTotalDisk()

	worker := &domain.Worker{
		ID:             uuid.New(),
		Name:           hostname,
		Hostname:       getLocalIP(),
		APIPort:        9090,
		ContainerdSock: containerdSock,
		Labels:         detectLabels(),
		MaxMemory:      totalMem,
		MaxCPU:         totalCPU,
		MaxDisk:        totalDisk,
		Status:         domain.WorkerStatusHealthy,
		RegisteredAt:   time.Now(),
		UpdatedAt:      time.Now(),
		Metadata: map[string]string{
			"go_version":   runtime.Version(),
			"go_os":        runtime.GOOS,
			"go_arch":      runtime.GOARCH,
			"num_cpu":      fmt.Sprintf("%d", runtime.NumCPU()),
			"containerd":   containerdSock,
		},
	}

	return &WorkerAgent{
		worker:         worker,
		controlPlane:   controlPlane,
		containerdSock: containerdSock,
		runtime:        containerRuntime,
		stopCh:         make(chan struct{}),
	}
}

func (a *WorkerAgent) Start(ctx context.Context) error {
	// Register with control plane
	if err := a.controlPlane.RegisterWorker(ctx, a.worker); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	slog.Info("worker registered", "worker_id", a.worker.ID, "hostname", a.worker.Name)

	// Start heartbeat loop
	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	// Start resource monitoring
	a.wg.Add(1)
	go a.monitorResources(ctx)

	// Wait for stop signal
	<-a.stopCh
	a.wg.Wait()
	return nil
}

func (a *WorkerAgent) Stop() {
	close(a.stopCh)
}

func (a *WorkerAgent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *WorkerAgent) sendHeartbeat(ctx context.Context) {
	capacity := a.calculateCapacity()
	a.worker.LastHeartbeat = time.Now()
	a.worker.UpdatedAt = time.Now()

	if err := a.controlPlane.Heartbeat(ctx, a.worker.ID, capacity); err != nil {
		slog.Error("failed to send heartbeat", "error", err)
	}
}

func (a *WorkerAgent) calculateCapacity() *domain.WorkerCapacity {
	// Get current resource usage
	usedMem := getUsedMemory()
	usedCPU := getUsedCPU()
	usedDisk := getUsedDisk()

	// Get active sandbox count from runtime
	// This would query the container runtime for actual running containers
	activeSandboxes := 0 // Placeholder

	return &domain.WorkerCapacity{
		WorkerID:         a.worker.ID,
		AvailableMemory:  a.worker.MaxMemory - usedMem,
		AvailableCPU:     a.worker.MaxCPU - usedCPU,
		AvailableDisk:    a.worker.MaxDisk - usedDisk,
		ActiveSandboxes:  activeSandboxes,
	}
}

func (a *WorkerAgent) monitorResources(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			// Check resource thresholds
			capacity := a.calculateCapacity()

			// Update status based on resource usage
			memUsagePct := float64(a.worker.MaxMemory-capacity.AvailableMemory) / float64(a.worker.MaxMemory) * 100
			cpuUsagePct := float64(a.worker.MaxCPU-capacity.AvailableCPU) / float64(a.worker.MaxCPU) * 100
			diskUsagePct := float64(a.worker.MaxDisk-capacity.AvailableDisk) / float64(a.worker.MaxDisk) * 100

			newStatus := domain.WorkerStatusHealthy
			if memUsagePct > 90 || cpuUsagePct > 90 || diskUsagePct > 90 {
				newStatus = domain.WorkerStatusDegraded
			}
			if memUsagePct > 95 || cpuUsagePct > 95 || diskUsagePct > 95 {
				newStatus = domain.WorkerStatusUnhealthy
			}

			if newStatus != a.worker.Status {
				slog.Info("worker status changed", "worker_id", a.worker.ID, "old", a.worker.Status, "new", newStatus)
				a.worker.Status = newStatus
			}
		}
	}
}

// getLocalIP returns the local IP address for worker registration.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// detectLabels detects hardware and software capabilities.
func detectLabels() map[string]string {
	labels := make(map[string]string)

	// Check for GPU
	if _, err := os.Stat("/dev/nvidia0"); err == nil {
		labels["gpu"] = "nvidia"
	}

	// Check for specific hardware
	if _, err := os.Stat("/sys/class/dmi/id/product_name"); err == nil {
		// Could read product name for hardware type
		labels["platform"] = "bare-metal"
	}

	// Check for container runtime
	if _, err := os.Stat("/run/containerd/containerd.sock"); err == nil {
		labels["runtime"] = "containerd"
	}

	// Check for KVM
	if _, err := os.Stat("/dev/kvm"); err == nil {
		labels["kvm"] = "true"
	}

	return labels
}

// getTotalMemory returns total system memory in bytes.
func getTotalMemory() int64 {
	// Linux: read from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := string(data)
		for _, line := range splitLines(lines) {
			if len(line) > 10 && line[:9] == "MemTotal:" {
				var kb int64
				fmt.Sscanf(line, "MemTotal: %d kB", &kb)
				return kb * 1024
			}
		}
	}
	// Fallback
	return int64(8 * 1024 * 1024 * 1024) // 8GB default
}

// getTotalDisk returns total disk space in bytes.
func getTotalDisk() int64 {
	// Get disk space for root filesystem
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err == nil {
		return int64(stat.Blocks) * int64(stat.Bsize)
	}
	return int64(100 * 1024 * 1024 * 1024) // 100GB default
}

func getUsedMemory() int64 {
	// Read from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, available int64
		lines := string(data)
		for _, line := range splitLines(lines) {
			if len(line) > 9 && line[:8] == "MemTotal:" {
				fmt.Sscanf(line, "MemTotal: %d kB", &total)
			}
			if len(line) > 13 && line[:12] == "MemAvailable:" {
				fmt.Sscanf(line, "MemAvailable: %d kB", &available)
			}
		}
		return (total - available) * 1024
	}
	return 0
}

func getUsedCPU() int64 {
	// This would read from /proc/stat or use a library like gopsutil
	// Simplified: return 0 for now
	return 0
}

func getUsedDisk() int64 {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err == nil {
		used := int64(stat.Blocks-stat.Bfree) * int64(stat.Bsize)
		return used
	}
	return 0
}

func splitLines(s string) []string {
	lines := []string{}
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}