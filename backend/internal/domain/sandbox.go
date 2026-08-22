package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RuntimeProfile represents the predicted runtime configuration for a sandbox.
type RuntimeProfile struct {
	Language    string // "node", "python", "go", "rust", "other"
	BaseImage   string // e.g., "node:20-alpine"
	InstallCmd  string // e.g., "npm install"
	StartCmd    string // e.g., "npm run dev"
	WorkDir     string // e.g., "/app"
	ExposedPort int    // e.g., 3000
	NeedsDB     bool   // whether a sidecar DB is predicted
	Confidence  float64
}

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	StateIdle     SandboxState = "IDLE"
	StateBuilding SandboxState = "BUILDING"
	StateRunning  SandboxState = "RUNNING"
	StateStopped  SandboxState = "STOPPED"
	StateFailed   SandboxState = "FAILED"
)

// Sandbox is the core aggregate root for an ephemeral dev environment.
type Sandbox struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	GitURL      string
	GitBranch   string
	State       SandboxState
	Profile     *RuntimeProfile
	ContainerID *string // containerd container ID
	PublicURL   *string
	Port        *int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
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
	ProfileHash string   // hash of RuntimeProfile for matching
	ContainerID string   // containerd container ID
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
	UpdateState(ctx context.Context, id uuid.UUID, state SandboxState) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
}

// PredictionService defines the interface for the ML prediction microservice.
// Implemented by the Python gRPC service client.
type PredictionService interface {
	Predict(ctx context.Context, gitURL string, branch string) (*RuntimeProfile, error)
}

// ContainerRuntime defines the interface for containerd/gVisor operations.
// This abstracts containerd so we can mock it in tests.
type ContainerRuntime interface {
	CreateSandbox(ctx context.Context, spec SandboxSpec) (string, error) // returns containerID
	StartSandbox(ctx context.Context, containerID string) error
	StopSandbox(ctx context.Context, containerID string) error
	DeleteSandbox(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cmd []string) (string, error)
	GetLogs(ctx context.Context, containerID string, tail int) (string, error)
}

// SandboxSpec is the specification passed to the container runtime.
type SandboxSpec struct {
	ID          uuid.UUID
	BaseImage   string
	WorkDir     string
	Env         map[string]string
	MemoryLimit int64 // bytes
	CPULimit    int64 // milli-cores
	DiskLimit   int64 // bytes
	NetworkID   string
}
