package containerd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd/api/types/runc/options"
	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/AyushCN/berth/internal/domain"
)

const (
	berthNamespace      = "berth"
	gvisorRuntime       = "io.containerd.runsc.v1"
	defaultRuntime      = "io.containerd.runc.v2"
	defaultTimeout      = 30 * time.Second
	defaultCgroupParent = "berth.slice"
)

// Runtime implements domain.ContainerRuntime using containerd + gVisor.
type Runtime struct {
	client   *client.Client
	sockPath string
	layerMgr *LayerManager
	netMgr      *NetworkManager
	warmPool    *WarmPool
	runtimeType string
}

// NewRuntime creates a new containerd-backed runtime.
func NewRuntime(sockPath string, runtimeType string) (*Runtime, error) {
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

	layerMgr, err := NewLayerManager(c)
	if err != nil {
		return nil, fmt.Errorf("failed to init layer manager: %w", err)
	}

	netMgr, err := NewNetworkManager()
	if err != nil {
		return nil, fmt.Errorf("failed to init network manager: %w", err)
	}

	r := &Runtime{
		client:      c,
		sockPath:    sockPath,
		layerMgr:    layerMgr,
		netMgr:      netMgr,
		runtimeType: runtimeType,
	}

	r.warmPool = NewWarmPool(8*1024*1024*1024, func(ctx context.Context, id string) error {
		return r.DeleteSandbox(ctx, id)
	})

	go r.MaintainBaseline()

	return r, nil
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
	ctx = withNamespace(ctx)

	// 1. Check warm pool first
	if warm := r.warmPool.Take(spec.BaseImage); warm != "" {
		slog.Info("warm pool hit", "container_id", warm)
		return warm, nil
	}

	return r.createSandboxInternal(ctx, spec)
}

func (r *Runtime) createSandboxInternal(ctx context.Context, spec domain.SandboxSpec) (string, error) {
	// 2. Resolve or build dependency layer
	baseImg, err := r.layerMgr.ResolveBaseImage(ctx, spec.BaseImage)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base image: %w", err)
	}

	// 3. Create OCI spec with security hardening
	containerID := spec.ID.String()

	// Build OCI opts
	ociOpts := []oci.SpecOpts{
		withLinuxNamespaces(),
		withCgroupLimits(spec.MemoryLimit, spec.CPULimit),
		withDroppedCapabilities(),
		withSeccomp(),
		withReadonlyRootfs(),
		withTmpfs(),
	}

	// Bind mount workspace if provided (Phase 1: rbind rw)
	// Phase 2: replace with 9P/virtiofs
	if spec.WorkspaceDir != "" {
		ociOpts = append(ociOpts, withWorkspaceMount(spec.WorkspaceDir, spec.WorkDir))
	}

	ociOpts = append(ociOpts, func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		var newMounts []specs.Mount
		for _, m := range s.Mounts {
			if m.Destination == "/sys" || m.Type == "sysfs" {
				continue
			}
			newMounts = append(newMounts, m)
		}
		s.Mounts = newMounts

		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		hasPath := false
		for _, e := range s.Process.Env {
			if strings.HasPrefix(e, "PATH=") {
				hasPath = true
				break
			}
		}
		if !hasPath {
			s.Process.Env = append(s.Process.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
		}

		return nil
	})

	// Set main process command if provided
	if len(spec.Cmd) > 0 {
		ociOpts = append(ociOpts, withProcessArgs(spec.Cmd...))
	}

	rt := defaultRuntime
	var optsData *anypb.Any

	if r.runtimeType == "runsc" {
		rt = gvisorRuntime
		// runsc shim does not accept runc options format. Pass nil.
		optsData = nil
	} else {
		optsData, err = anypb.New(&options.Options{})
		if err != nil {
			return "", fmt.Errorf("failed to create runc options: %w", err)
		}
	}

	opts := []client.NewContainerOpts{
		client.WithImage(baseImg),
		client.WithNewSnapshot(containerID+"-snap", baseImg),
		client.WithRuntime(rt, optsData),
		client.WithNewSpec(ociOpts...),
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

	slog.Info("sandbox created (cache miss)", "container_id", containerID, "image", spec.BaseImage)
	return containerID, nil
}

// MaintainBaseline periodically ensures a baseline of warm containers.
func (r *Runtime) MaintainBaseline() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	baselineImages := []string{
		"docker.io/library/node:20-alpine",
		"docker.io/library/python:3.11-alpine",
		"docker.io/library/golang:1.23-alpine",
	}

	for range ticker.C {
		for _, img := range baselineImages {
			r.warmPool.mu.RLock()
			count := len(r.warmPool.available[img])
			r.warmPool.mu.RUnlock()

			if count < 1 {
				ctx := context.Background()
				mem := int64(512 * 1024 * 1024)
				err := r.warmPool.PreWarm(ctx, img, mem, func() (string, error) {
					spec := domain.SandboxSpec{
						ID:          uuid.New(),
						BaseImage:   img,
						Cmd:         []string{"sh", "-c", "while true; do sleep 1; done"},
						MemoryLimit: mem,
						CPULimit:    500,
					}
					return r.createSandboxInternal(ctx, spec)
				})
				if err != nil {
					slog.Error("PreWarm failed", "image", img, "error", err)
				}
			}
		}
	}
}

// StartSandbox starts a container and its task.
func (r *Runtime) StartSandbox(ctx context.Context, containerID string) error {
	ctx = withNamespace(ctx)

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
	defer logFile.Close()

	// Create task
	task, err := container.NewTask(ctx, cio.NewCreator(
		cio.WithFIFODir(fifoDir),
	))
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Setup port forwarding
	if err := r.netMgr.ForwardPort(containerID, 0, 0); err != nil {
		slog.Warn("port forwarding setup failed", "error", err)
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	// Start log streaming goroutine
	go r.streamLogs(ctx, task, logPath, fifoDir)

	slog.Info("sandbox started", "container_id", containerID, "pid", task.Pid())
	return nil
}

// StopSandbox stops a container gracefully, then forcefully.
func (r *Runtime) StopSandbox(ctx context.Context, containerID string) error {
	ctx = withNamespace(ctx)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
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

	if _, err := task.Delete(ctx, client.WithProcessKill); err != nil {
		slog.Warn("task delete failed", "error", err)
	}

	return r.deleteContainer(ctx, container)
}

// DeleteSandbox destroys a container or returns it to the warm pool.
func (r *Runtime) DeleteSandbox(ctx context.Context, containerID string) error {
	ctx = withNamespace(ctx)

	if r.warmPool.Return(containerID) {
		slog.Info("sandbox returned to warm pool", "container_id", containerID)
		return nil
	}

	return r.StopSandbox(ctx, containerID)
}

// Exec runs a command inside an existing sandbox.
func (r *Runtime) Exec(ctx context.Context, containerID string, cmd []string) (string, error) {
	ctx = withNamespace(ctx)

	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

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

	var stdoutBuf, stderrBuf bytes.Buffer
	errCh := make(chan error, 2)

	go func() {
		f, err := os.OpenFile(filepath.Join(fifoDir, "stdout"), os.O_RDONLY, 0)
		if err != nil {
			errCh <- err
			return
		}
		defer f.Close()
		_, err = io.Copy(&stdoutBuf, f)
		errCh <- err
	}()

	go func() {
		f, err := os.OpenFile(filepath.Join(fifoDir, "stderr"), os.O_RDONLY, 0)
		if err != nil {
			errCh <- err
			return
		}
		defer f.Close()
		_, err = io.Copy(&stderrBuf, f)
		errCh <- err
	}()

	if err := process.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start exec: %w", err)
	}

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

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			slog.Warn("exec fifo read error", "error", err)
		}
	}

	if _, err := process.Delete(ctx); err != nil {
		slog.Warn("exec process delete failed", "error", err)
	}

	_ = os.RemoveAll(fifoDir)

	if status != 0 {
		return stderrBuf.String(), fmt.Errorf("exec exited with code %d: %s", status, stderrBuf.String())
	}

	return stdoutBuf.String(), nil
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
	time.Sleep(100 * time.Millisecond)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("failed to open log file for streaming", "error", err)
		return
	}
	defer logFile.Close()

	stdoutFifo := filepath.Join(fifoDir, "stdout")
	stderrFifo := filepath.Join(fifoDir, "stderr")

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
				return
			}
			_, _ = io.Copy(logFile, f)
			f.Close()
		}
	}()

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
			return
		}
		_, _ = io.Copy(logFile, f)
		f.Close()
	}
}

func withNamespace(ctx context.Context) context.Context {
	// containerd v2 namespaces
	return ctx
}

// --- OCI spec options ---

func withWorkspaceMount(hostDir, containerDir string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Mounts = append(s.Mounts, specs.Mount{
			Destination: containerDir,
			Type:        "bind",
			Source:      hostDir,
			Options:     []string{"rbind", "rw"},
		})
		return nil
	}
}

func withProcessArgs(args ...string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Process == nil {
			s.Process = &specs.Process{}
		}
		s.Process.Args = args
		return nil
	}
}

func withLinuxNamespaces() oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *specs.Spec) error {
		namespaces := []specs.LinuxNamespace{
			{Type: specs.PIDNamespace},
			{Type: specs.MountNamespace},
			{Type: specs.IPCNamespace},
			{Type: specs.UTSNamespace},
		}
		if os.Geteuid() == 0 {
			namespaces = append(namespaces,
				specs.LinuxNamespace{Type: specs.NetworkNamespace},
				specs.LinuxNamespace{Type: specs.CgroupNamespace},
			)
		}
		s.Linux.Namespaces = namespaces
		return nil
	}
}

func withCgroupLimits(memBytes, cpuMilli int64) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *specs.Spec) error {
		if os.Geteuid() != 0 {
			slog.Warn("skipping cgroup limits: requires root privileges")
			if s.Linux != nil {
				s.Linux.CgroupsPath = ""
				s.Linux.Resources = nil
			}
			return nil
		}
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
			quota := cpuMilli * 1000
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
		s.Linux.Seccomp = nil // containerd applies default seccomp
		return nil
	}
}

func withReadonlyRootfs() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Root == nil {
			s.Root = &specs.Root{}
		}
		s.Root.Readonly = true
		return nil
	}
}

func withTmpfs() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		s.Mounts = append(s.Mounts,
			specs.Mount{
				Destination: "/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "noexec", "nodev", "size=100m"},
			},
			specs.Mount{
				Destination: "/var/tmp",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "noexec", "nodev", "size=50m"},
			},
		)
		return nil
	}
}
