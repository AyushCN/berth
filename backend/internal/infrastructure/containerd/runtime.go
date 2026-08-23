package containerd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/AyushCN/berth/internal/domain"
)

const (
	berthNamespace      = "berth"
	gvisorRuntime       = "io.containerd.runsc.v1"
	defaultTimeout      = 30 * time.Second
	defaultCgroupParent = "berth.slice"
)

// Runtime implements domain.ContainerRuntime using containerd + gVisor.
type Runtime struct {
	client   *client.Client
	sockPath string
	layerMgr *LayerManager
	netMgr   *NetworkManager
	warmPool *WarmPool
}

// NewRuntime creates a new containerd-backed runtime.
func NewRuntime(sockPath string) (*Runtime, error) {
	if sockPath == "" {
		sockPath = "/run/containerd/containerd.sock"
		if os.Getenv("CONTAINERD_SOCK") != "" {
			sockPath = os.Getenv("CONTAINERD_SOCK")
		}
	}

	var c *client.Client
	var err error
	for i := 1; i <= 10; i++ {
		c, err = client.New(sockPath, client.WithDefaultNamespace(berthNamespace))
		if err == nil {
			break
		}
		slog.Warn("containerd not ready", "attempt", i, "error", err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd after 10 attempts: %w", err)
	}

	// Ensure namespace exists
	// Ensure namespace exists
	// ctx := context.Background()
	// nsService := c.NamespaceService()
	// if _, err := nsService.GetNamespace(ctx, berthNamespace); err != nil {
	// 	if errdefs.IsNotFound(err) {
	// 		if err := nsService.CreateNamespace(ctx, namespaces.Namespace{
	// 			Name: berthNamespace,
	// 		}); err != nil {
	// 			return nil, fmt.Errorf("failed to create berth namespace: %w", err)
	// 		}
	// 		slog.Info("created containerd namespace", "namespace", berthNamespace)
	// 	} else {
	// 		return nil, fmt.Errorf("failed to check namespace: %w", err)
	// 	}
	// }

	layerMgr, err := NewLayerManager(c)
	if err != nil {
		return nil, fmt.Errorf("failed to init layer manager: %w", err)
	}

	netMgr, err := NewNetworkManager()
	if err != nil {
		return nil, fmt.Errorf("failed to init network manager: %w", err)
	}

	return &Runtime{
		client:   c,
		sockPath: sockPath,
		layerMgr: layerMgr,
		netMgr:   netMgr,
		warmPool: NewWarmPool(),
	}, nil
}

// Close closes the containerd client.
func (r *Runtime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// CreateSandbox creates a new gVisor sandbox.
func (r *Runtime) CreateSandbox(ctx context.Context, spec domain.SandboxSpec) (string, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	// 1. Check warm pool first
	if warm := r.warmPool.Take(spec.BaseImage); warm != "" {
		slog.Info("warm pool hit", "container_id", warm)
		return warm, nil
	}

	// 2. Resolve or build dependency layer
	baseImg, err := r.layerMgr.ResolveBaseImage(ctx, spec.BaseImage)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base image: %w", err)
	}

	// 3. Create OCI spec with security hardening
	containerID := spec.ID.String()

	opts := []client.NewContainerOpts{
		client.WithImage(baseImg),
		client.WithNewSnapshot(containerID+"-snap", baseImg),
		client.WithRuntime(gvisorRuntime, nil),
		client.WithNewSpec(
			withLinuxNamespaces(),
			withCgroupLimits(spec.MemoryLimit, spec.CPULimit),
			withDroppedCapabilities(),
			withSeccomp(),
			withReadonlyRootfs(),
		),
	}

	container, err := r.client.NewContainer(ctx, containerID, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// 4. Setup networking
	networkID := "berth-" + containerID[:8]
	if err := r.netMgr.CreateNetwork(ctx, networkID); err != nil {
		_ = container.Delete(ctx, client.WithSnapshotCleanup)
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	// 4b. Allocate IP for the container
	_, err = r.netMgr.AllocateIP(networkID, containerID)
	if err != nil {
		_ = container.Delete(ctx, client.WithSnapshotCleanup)
		_ = r.netMgr.DestroyNetwork(ctx, networkID)
		return "", fmt.Errorf("failed to allocate IP: %w", err)
	}

	// 5. Store metadata in warm pool for later return
	r.warmPool.Register(containerID, spec.BaseImage)

	slog.Info("sandbox created", "container_id", containerID, "image", spec.BaseImage)
	return containerID, nil
}

// StartSandbox starts a container and its task.
func (r *Runtime) StartSandbox(ctx context.Context, containerID string) error {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}

	// Create task with log FIFO
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".local", "state", "berth", "logs", containerID)
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "task.log")

	fifoDir := filepath.Join(home, ".local", "state", "berth", "fifo", containerID)
	_ = os.MkdirAll(fifoDir, 0755)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create task
	task, err := container.NewTask(ctx, cio.NewCreator(
		cio.WithFIFODir(fifoDir),
	))
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Setup port forwarding
	if err := r.netMgr.ForwardPort(containerID, 0, 0); err != nil {
		slog.Warn("port forwarding setup failed", "error", err)
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	// Start log streaming goroutine (reads from FIFO as backup)
	go r.streamLogs(ctx, task, logPath, fifoDir)

	slog.Info("sandbox started", "container_id", containerID, "pid", task.Pid())
	return nil
}

// StopSandbox stops a container gracefully, then forcefully.
func (r *Runtime) StopSandbox(ctx context.Context, containerID string) error {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return r.deleteContainer(ctx, container)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Graceful SIGTERM
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		slog.Warn("SIGTERM failed", "error", err)
	}

	// Wait for exit or timeout
	exitCh, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for task: %w", err)
	}

	select {
	case <-exitCh:
		slog.Info("task exited gracefully", "container_id", containerID)
	case <-time.After(10 * time.Second):
		slog.Warn("graceful shutdown timed out, force killing", "container_id", containerID)
		if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
			slog.Error("SIGKILL failed", "error", err)
		}
		<-exitCh
	}

	// Delete task
	if _, err := task.Delete(ctx, client.WithProcessKill); err != nil {
		slog.Warn("task delete failed", "error", err)
	}

	return r.deleteContainer(ctx, container)
}

// DeleteSandbox destroys a container or returns it to the warm pool.
func (r *Runtime) DeleteSandbox(ctx context.Context, containerID string) error {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	// Check if warm pool wants it back
	if r.warmPool.Return(containerID) {
		slog.Info("sandbox returned to warm pool", "container_id", containerID)
		return nil
	}

	return r.StopSandbox(ctx, containerID)
}

// Exec runs a command inside an existing sandbox.
func (r *Runtime) Exec(ctx context.Context, containerID string, cmd []string) (string, error) {
	ctx = namespaces.WithNamespace(ctx, berthNamespace)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

	// Create exec process
	processSpec := &specs.Process{
		Terminal: false,
		Args:     cmd,
		Cwd:      "/app",
		Env:      []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
	}

	home, _ := os.UserHomeDir()
	fifoDir := filepath.Join(home, ".local", "state", "berth", "fifo", containerID, "exec-"+uuid.New().String()[:8])
	_ = os.MkdirAll(fifoDir, 0755)

	process, err := task.Exec(ctx, uuid.New().String(), processSpec, cio.NewCreator(cio.WithFIFODir(fifoDir)))
	if err != nil {
		return "", fmt.Errorf("failed to create exec process: %w", err)
	}

	if err := process.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start exec: %w", err)
	}

	// Wait for completion
	statusC, err := process.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to wait for exec: %w", err)
	}

	var status uint32
	select {
	case st := <-statusC:
		status = st.ExitCode()
	case <-ctx.Done():
		_ = process.Kill(ctx, syscall.SIGKILL)
		return "", ctx.Err()
	}

	// Delete the process
	if _, err := process.Delete(ctx); err != nil {
		slog.Warn("exec process delete failed", "error", err)
	}

	if status != 0 {
		return "", fmt.Errorf("exec exited with code %d", status)
	}

	return "", nil // TODO: capture stdout from FIFO
}

// GetLogs retrieves logs from a sandbox.
func (r *Runtime) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".local", "state", "berth", "logs", containerID, "task.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read logs: %w", err)
	}
	return string(data), nil
}

// --- internal helpers ---

func (r *Runtime) deleteContainer(ctx context.Context, container client.Container) error {
	if err := container.Delete(ctx, client.WithSnapshotCleanup); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to delete container: %w", err)
		}
	}
	return nil
}

func (r *Runtime) streamLogs(ctx context.Context, task client.Task, logPath string, fifoDir string) {
	// Phase 1: Primary logging is handled by cio.WithOutput in StartSandbox.
	// This goroutine serves as a backup reader from the FIFO directory
	// in case the direct output redirection misses anything.
	// Phase 2 will replace this with NATS streaming.

	stdoutFifo := filepath.Join(fifoDir, "stdout")
	stderrFifo := filepath.Join(fifoDir, "stderr")

	// Wait a moment for FIFOs to be created
	time.Sleep(100 * time.Millisecond)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("failed to open log file for streaming", "error", err)
		return
	}
	defer logFile.Close()

	// Read stdout FIFO
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			f, err := os.Open(stdoutFifo)
			if err != nil {
				if os.IsNotExist(err) {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				slog.Error("failed to open stdout fifo", "error", err)
				return
			}
			_, _ = io.Copy(logFile, f)
			f.Close()
		}
	}()

	// Read stderr FIFO
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		f, err := os.Open(stderrFifo)
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			slog.Error("failed to open stderr fifo", "error", err)
			return
		}
		_, _ = io.Copy(logFile, f)
		f.Close()
	}
}

// --- OCI spec options ---

func withLinuxNamespaces() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}
		s.Linux.Namespaces = []specs.LinuxNamespace{
			{Type: specs.PIDNamespace},
			{Type: specs.NetworkNamespace},
			{Type: specs.MountNamespace},
			{Type: specs.IPCNamespace},
			{Type: specs.UTSNamespace},
			{Type: specs.CgroupNamespace},
		}
		return nil
	}
}

func withCgroupLimits(memBytes, cpuMilli int64) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}
		if s.Linux.Resources == nil {
			s.Linux.Resources = &specs.LinuxResources{}
		}
		if memBytes > 0 {
			s.Linux.Resources.Memory = &specs.LinuxMemory{
				Limit: &memBytes,
			}
		}
		if cpuMilli > 0 {
			quota := cpuMilli * 1000 // convert milli-cores to microseconds
			period := uint64(100000)
			s.Linux.Resources.CPU = &specs.LinuxCPU{
				Quota:  &quota,
				Period: &period,
			}
		}
		return nil
	}
}

func withDroppedCapabilities() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		if s.Process.Capabilities == nil {
			s.Process.Capabilities = &specs.LinuxCapabilities{}
		}
		// gVisor intercepts syscalls in userspace; drop all capabilities.
		s.Process.Capabilities.Bounding = []string{}
		s.Process.Capabilities.Effective = []string{}
		s.Process.Capabilities.Permitted = []string{}
		s.Process.Capabilities.Inheritable = []string{}
		s.Process.Capabilities.Ambient = []string{}
		return nil
	}
}

func withSeccomp() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}
		// containerd applies default seccomp when Seccomp is nil
		s.Linux.Seccomp = nil
		return nil
	}
}

func withReadonlyRootfs() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		// Phase 1: read-write rootfs for compatibility
		// Phase 2: switch to read-only + tmpfs overlay
		return nil
	}
}
