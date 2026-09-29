package domain

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// ContainerRuntimeKind identifies the kernel-level isolation technology.
type ContainerRuntimeKind string

const (
	// RuntimeGVisor uses gVisor (runsc) for stronger isolation.
	RuntimeGVisor ContainerRuntimeKind = "gvisor"
	// RuntimeRunc uses standard runc.
	RuntimeRunc ContainerRuntimeKind = "runc"
)

// NetworkMode defines the network isolation mode for a container.
type NetworkMode string

const (
	// NetworkModeHost uses host networking (no isolation)
	NetworkModeHost NetworkMode = "host"
	// NetworkModeBridge uses Docker bridge networking (default isolation)
	NetworkModeBridge NetworkMode = "bridge"
	// NetworkModeCNI uses CNI for an isolated network namespace
	NetworkModeCNI NetworkMode = "cni"
	// NetworkModeNone disables networking entirely
	NetworkModeNone NetworkMode = "none"
)

// FilesystemMode defines the filesystem isolation mode for a container.
type FilesystemMode string

const (
	// FilesystemModeBindMount uses bind mounts for the workspace
	FilesystemModeBindMount FilesystemMode = "bindmount"
	// FilesystemModeOverlay uses an overlay filesystem for copy-on-write
	FilesystemModeOverlay FilesystemMode = "overlay"
	// FilesystemModeRO makes the root filesystem read-only
	FilesystemModeRO FilesystemMode = "readonly"
)

// ExecutionProfile is the security and execution profile applied to a
// container. It describes how a container runs, not what it runs, so it is
// independent of the environment model.
type ExecutionProfile struct {
	// Runtime specifies the container runtime to use
	Runtime ContainerRuntimeKind `json:"runtime"`

	// Rootless indicates if the container should run rootless
	Rootless bool `json:"rootless"`

	// NetworkMode defines the network isolation mode
	NetworkMode NetworkMode `json:"network_mode"`

	// FilesystemMode defines the filesystem isolation mode
	FilesystemMode FilesystemMode `json:"filesystem_mode"`

	// Capabilities lists the Linux capabilities to add (empty = drop all)
	Capabilities []string `json:"capabilities"`

	// CPUQuota is the CPU quota in milli-cores (1000 = 1 CPU)
	CPUQuota int64 `json:"cpu_quota"`

	// MemoryLimit is the memory limit in bytes
	MemoryLimit int64 `json:"memory_limit"`

	// PidsLimit is the maximum number of processes
	PidsLimit int `json:"pids_limit"`

	// ReadOnlyRootFS makes the container root filesystem read-only
	ReadOnlyRootFS bool `json:"read_only_root_fs"`

	// NoNewPrivileges prevents processes from gaining new privileges
	NoNewPrivileges bool `json:"no_new_privileges"`

	// SeccompProfile names a custom seccomp profile. "default" and "" both
	// mean Docker's own default, which it applies when the option is absent;
	// "seccomp=default" is not a value Docker accepts.
	SeccompProfile string `json:"seccomp_profile"`

	// DiskLimit is the disk quota in bytes
	DiskLimit int64 `json:"disk_limit"`

	// NetworkEgressPolicy defines the egress network policy
	NetworkEgressPolicy string `json:"network_egress_policy"`
}

// DefaultExecutionProfile returns a hardened profile used for untrusted
// workloads. Note NetworkModeHost: with host networking the container's port
// is bound directly on the host, which is what the api's preview proxy
// targets. Switching this to bridge requires Traefik to route by container IP.
func DefaultExecutionProfile() *ExecutionProfile {
	return &ExecutionProfile{
		Runtime:             RuntimeRunc,
		Rootless:            true,
		NetworkMode:         NetworkModeHost,
		FilesystemMode:      FilesystemModeBindMount,
		Capabilities:        []string{},
		CPUQuota:            1000,
		MemoryLimit:         512 * 1024 * 1024,
		PidsLimit:           256,
		ReadOnlyRootFS:      true,
		NoNewPrivileges:     true,
		SeccompProfile:      "default",
		DiskLimit:           2 * 1024 * 1024 * 1024,
		NetworkEgressPolicy: "default",
	}
}

// ValidateExecutionProfile checks an execution profile for invalid settings.
func ValidateExecutionProfile(profile *ExecutionProfile) error {
	if profile == nil {
		return fmt.Errorf("execution profile is nil")
	}

	switch profile.Runtime {
	case RuntimeRunc, RuntimeGVisor:
	default:
		return fmt.Errorf("invalid runtime: %s", profile.Runtime)
	}

	switch profile.NetworkMode {
	case NetworkModeHost, NetworkModeBridge, NetworkModeCNI, NetworkModeNone:
	default:
		if profile.NetworkMode != "" {
			return fmt.Errorf("invalid network mode: %s", profile.NetworkMode)
		}
	}

	switch profile.FilesystemMode {
	case FilesystemModeBindMount, FilesystemModeOverlay, FilesystemModeRO:
	default:
		if profile.FilesystemMode != "" {
			return fmt.Errorf("invalid filesystem mode: %s", profile.FilesystemMode)
		}
	}

	if profile.MemoryLimit < 0 {
		return fmt.Errorf("memory limit must be non-negative")
	}
	if profile.CPUQuota < 0 {
		return fmt.Errorf("CPU quota must be non-negative")
	}
	if profile.DiskLimit < 0 {
		return fmt.Errorf("disk limit must be non-negative")
	}
	if profile.PidsLimit < 0 {
		return fmt.Errorf("PIDs limit must be non-negative")
	}

	return nil
}

// ContainerRuntime is the interface the worker uses to manage containers.
//
// These methods were CreateSandbox/StartSandbox/StopSandbox/DeleteSandbox,
// which described sandboxes. They manage containers, and the sandbox model is
// gone, so the names no longer match what they do.
type ContainerRuntime interface {
	Create(ctx context.Context, spec ContainerSpec) (string, error) // returns containerID
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cmd []string) (string, error)
	ExecWithEnv(ctx context.Context, containerID string, cmd []string, env map[string]string) (string, error)
	ExecPTY(ctx context.Context, containerID string, cmd []string) (stdin io.WriteCloser, stdout io.Reader, wait func() error, err error)
	GetLogs(ctx context.Context, containerID string, tail int) (string, error)
	CommitContainer(ctx context.Context, containerID, imageName string) error
}

// ContainerSpec describes a container to create. It was previously
// domain.SandboxSpec, which described a container, not a sandbox.
type ContainerSpec struct {
	ID               uuid.UUID
	BaseImage        string
	WorkDir          string
	WorkspaceDir     string            // Host directory to bind-mount into the container
	ExtraMounts      map[string]string // HostDir -> ContainerDir
	Cmd              []string          // Main process (the file watcher, for hot reload)
	Env              map[string]string
	MemoryLimit      int64 // bytes
	CPULimit         int64 // milli-cores
	DiskLimit        int64 // bytes
	NetworkID        string
	ExposedPort      *int // Port the application listens on
	Labels           map[string]string
	ExecutionProfile *ExecutionProfile
}
