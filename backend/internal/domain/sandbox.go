package domain

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// SandboxRuntimeProfile contains the runtime configuration detected from a repository.
// Deprecated: Use RuntimeProfile from runtime_profile.go instead
type SandboxRuntimeProfile struct {
	Language    string // "node", "python", "go", "rust", "other"
	BaseImage   string // e.g., "node:20-alpine"
	InstallCmd  string // e.g., "npm install"
	StartCmd    string // e.g., "npm run dev"
	WorkDir     string // e.g., "/app"
	ExposedPort int    // e.g., 3000
}

// SandboxRuntime represents the container runtime to use for a sandbox.
type SandboxRuntime string

const (
	// RuntimeGVisor uses gVisor (runsc) for strong isolation
	RuntimeGVisor SandboxRuntime = "gvisor"
	// RuntimeRunc uses standard runc for lighter-weight isolation
	RuntimeRunc SandboxRuntime = "runc"
)

// NetworkMode defines the network isolation mode for a sandbox.
type NetworkMode string

const (
	// NetworkModeHost uses host networking (no isolation)
	NetworkModeHost NetworkMode = "host"
	// NetworkModeCNI uses CNI for isolated network namespace
	NetworkModeCNI NetworkMode = "cni"
	// NetworkModeNone disables networking entirely
	NetworkModeNone NetworkMode = "none"
)

// FilesystemMode defines the filesystem isolation mode for a sandbox.
type FilesystemMode string

const (
	// FilesystemModeBindMount uses bind mounts for workspace (current behavior)
	FilesystemModeBindMount FilesystemMode = "bindmount"
	// FilesystemModeOverlay uses overlay filesystem for copy-on-write
	FilesystemModeOverlay FilesystemMode = "overlay"
	// FilesystemModeRO makes root filesystem read-only with explicit writable mounts
	FilesystemModeRO FilesystemMode = "readonly"
)

// ExecutionProfile defines the complete security and execution profile for a sandbox.
type ExecutionProfile struct {
	// Runtime specifies the container runtime to use
	Runtime SandboxRuntime `json:"runtime"`

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

	// SeccompProfile specifies a custom seccomp profile (empty = default)
	SeccompProfile string `json:"seccomp_profile"`

	// DiskLimit is the disk quota in bytes
	DiskLimit int64 `json:"disk_limit"`

	// NetworkEgressPolicy defines the egress network policy
	NetworkEgressPolicy string `json:"network_egress_policy"`
}

// DefaultExecutionProfile returns a secure default execution profile for untrusted workloads.
func DefaultExecutionProfile() *ExecutionProfile {
	return &ExecutionProfile{
		Runtime:           RuntimeGVisor,
		Rootless:          true,
		NetworkMode:       NetworkModeCNI,
		FilesystemMode:    FilesystemModeRO,
		Capabilities:      []string{}, // Drop all capabilities
		CPUQuota:          1000,       // 1 CPU
		MemoryLimit:       512 * 1024 * 1024, // 512 MiB
		PidsLimit:         256,
		ReadOnlyRootFS:    true,
		NoNewPrivileges:   true,
		SeccompProfile:    "default",
		DiskLimit:         2 * 1024 * 1024 * 1024, // 2 GiB
		NetworkEgressPolicy: "default",
	}
}

// TrustedExecutionProfile returns a relaxed execution profile for trusted workloads.
func TrustedExecutionProfile() *ExecutionProfile {
	return &ExecutionProfile{
		Runtime:           RuntimeRunc,
		Rootless:          true,
		NetworkMode:       NetworkModeHost,
		FilesystemMode:    FilesystemModeBindMount,
		Capabilities:      []string{"CHOWN", "DAC_OVERRIDE", "SETGID", "SETUID"},
		CPUQuota:          2000,
		MemoryLimit:       1024 * 1024 * 1024, // 1 GiB
		PidsLimit:         512,
		ReadOnlyRootFS:    false,
		NoNewPrivileges:   false,
		SeccompProfile:    "",
		DiskLimit:         5 * 1024 * 1024 * 1024, // 5 GiB
		NetworkEgressPolicy: "trusted",
	}
}

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	StateIdle     SandboxState = "IDLE"
	StatePending  SandboxState = "PENDING"
	StateBuilding SandboxState = "BUILDING"
	StateRunning  SandboxState = "RUNNING"
	StateStopped  SandboxState = "STOPPED"
	StateFailed   SandboxState = "FAILED"
)

// Sandbox is the core aggregate root for an ephemeral dev environment.
// Deprecated: Use Workspace + Environment instead
type Sandbox struct {
	ID                    uuid.UUID            `json:"id"`
	ProjectID             uuid.UUID            `json:"project_id"`
	OwnerID               uuid.UUID            `json:"owner_id"`
	Name                  string               `json:"name"`
	GitURL                string               `json:"git_url"`
	GitBranch             string               `json:"git_branch"`
	State                 SandboxState         `json:"state"`
	Profile               *SandboxRuntimeProfile `json:"profile,omitempty"`
	ExecutionProfile      *ExecutionProfile    `json:"execution_profile,omitempty"`
	ContainerID           *string              `json:"container_id,omitempty"`
	PublicURL             *string              `json:"public_url,omitempty"`
	Port                  *int                 `json:"port,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
	ExpiresAt             *time.Time           `json:"expires_at,omitempty"`
	HasUncommittedChanges bool                 `json:"has_uncommitted_changes"`
	LastModifiedAt        *time.Time           `json:"last_modified_at,omitempty"`
	ModifiedByUserID      *uuid.UUID           `json:"modified_by_user_id,omitempty"`
	CommitHash            *string              `json:"commit_hash,omitempty"`
}

// IsActive returns true if the sandbox is running or building.
func (s *Sandbox) IsActive() bool {
	return s.State == StateBuilding || s.State == StateRunning
}

// CanEdit returns true if the sandbox is in an editable state.
func (s *Sandbox) CanEdit() bool {
	return s.State == StateRunning || s.State == StateIdle
}

// WarmPoolEntry represents a pre-warmed sandbox waiting for assignment.
type WarmPoolEntry struct {
	ID          uuid.UUID
	ProfileHash string // hash of RuntimeProfile for matching
	ContainerID string // containerd container ID
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// SandboxRepository defines the interface for sandbox persistence.
// This is implemented by the PostgreSQL repository (sqlc-generated + wrapper).
type SandboxRepository interface {
	Create(ctx context.Context, s *Sandbox) error
	GetByID(ctx context.Context, id uuid.UUID) (*Sandbox, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Sandbox, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*Sandbox, error)
	ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*Sandbox, error)
	UpdateState(ctx context.Context, id uuid.UUID, state SandboxState) error
	UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error
	UpdateContainerAndURL(ctx context.Context, id uuid.UUID, containerID string, publicURL string, port int) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
	PopPendingSandbox(ctx context.Context) (*Sandbox, error)
	ListExpiredSandboxes(ctx context.Context) ([]*Sandbox, error)
	UpdateGitTracking(ctx context.Context, id uuid.UUID, hasChanges bool, modifiedBy *uuid.UUID, commitHash *string) error
	LogActivity(ctx context.Context, sandboxID uuid.UUID, userID uuid.UUID, activityType string, data []byte) error
}

// ContainerRuntime defines the interface for containerd/gVisor operations.
// This abstracts containerd so we can mock it in tests.
type ContainerRuntime interface {
	CreateSandbox(ctx context.Context, spec SandboxSpec) (string, error) // returns containerID
	StartSandbox(ctx context.Context, containerID string) error
	StopSandbox(ctx context.Context, containerID string) error
	DeleteSandbox(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cmd []string) (string, error)
	ExecWithEnv(ctx context.Context, containerID string, cmd []string, env map[string]string) (string, error)
	ExecPTY(ctx context.Context, containerID string, cmd []string) (stdin io.WriteCloser, stdout io.Reader, wait func() error, err error)
	GetLogs(ctx context.Context, containerID string, tail int) (string, error)
}

// SandboxSpec is the specification passed to the container runtime.
type SandboxSpec struct {
	ID              uuid.UUID
	BaseImage       string
	WorkDir         string
	WorkspaceDir    string             // Host directory to bind-mount into container
	ExtraMounts     map[string]string  // HostDir -> ContainerDir
	Cmd             []string           // Container main process (watcher for hot reload)
	Env             map[string]string
	MemoryLimit     int64              // bytes
	CPULimit        int64              // milli-cores
	DiskLimit       int64              // bytes
	NetworkID       string
	ExposedPort     *int               // Port the application exposes (for Traefik)
	Labels          map[string]string  // Additional Docker labels (e.g., for Traefik)
	ExecutionProfile  *ExecutionProfile  // Security and execution profile
}
