package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EnvironmentState represents the lifecycle state of an environment
type EnvironmentState string

const (
	EnvironmentStateCreated      EnvironmentState = "CREATED"
	EnvironmentStateBuilding     EnvironmentState = "BUILDING"
	EnvironmentStateBuildFailed  EnvironmentState = "BUILD_FAILED"
	EnvironmentStateReady        EnvironmentState = "READY"
	EnvironmentStateStarting     EnvironmentState = "STARTING"
	EnvironmentStateRunning      EnvironmentState = "RUNNING"
	EnvironmentStateStopping     EnvironmentState = "STOPPING"
	EnvironmentStateStopped      EnvironmentState = "STOPPED"
	EnvironmentStateSuspending   EnvironmentState = "SUSPENDING"
	EnvironmentStateSuspended    EnvironmentState = "SUSPENDED"
	EnvironmentStateCrashed      EnvironmentState = "CRASHED"
	EnvironmentStateDeleting     EnvironmentState = "DELETING"
)

// IsTerminal returns true if the state is terminal (no further transitions without explicit action)
func (s EnvironmentState) IsTerminal() bool {
	switch s {
	case EnvironmentStateBuildFailed, EnvironmentStateCrashed, EnvironmentStateDeleting:
		return true
	default:
		return false
	}
}

// IsActive returns true if the environment is running or starting
func (s EnvironmentState) IsActive() bool {
	return s == EnvironmentStateRunning || s == EnvironmentStateStarting
}

// CanStart returns true if the environment can be started
func (s EnvironmentState) CanStart() bool {
	return s == EnvironmentStateReady || s == EnvironmentStateStopped || s == EnvironmentStateSuspended || s == EnvironmentStateCrashed
}

// CanStop returns true if the environment can be stopped
func (s EnvironmentState) CanStop() bool {
	return s == EnvironmentStateRunning || s == EnvironmentStateStarting
}

// CanSuspend returns true if the environment can be suspended
func (s EnvironmentState) CanSuspend() bool {
	return s == EnvironmentStateRunning
}

// CanBuild returns true if the environment can be built
func (s EnvironmentState) CanBuild() bool {
	return s == EnvironmentStateCreated || s == EnvironmentStateBuildFailed
}

// Environment represents a runnable instance
type Environment struct {
	ID                  uuid.UUID         `json:"id"`
	WorkspaceID         uuid.UUID         `json:"workspace_id"`
	RuntimeProfileID    *uuid.UUID        `json:"runtime_profile_id,omitempty"`
	Name                string            `json:"name"`
	State               EnvironmentState  `json:"state"`
	ContainerID         string            `json:"container_id,omitempty"`
	ImageID             *uuid.UUID        `json:"image_id,omitempty"`
	PublicURL           string            `json:"public_url,omitempty"`
	Port                int               `json:"port,omitempty"`
	MemoryLimit         int64             `json:"memory_limit"`  // bytes
	CPULimit            int64             `json:"cpu_limit"`     // nanocpus
	LastActivityAt      *time.Time        `json:"last_activity_at,omitempty"`
	ActiveSessions      int               `json:"active_sessions"`
	SuspendedAt         *time.Time        `json:"suspended_at,omitempty"`
	LastError           string            `json:"last_error,omitempty"`
	RestartCount        int               `json:"restart_count"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	DeletedAt           *time.Time        `json:"deleted_at,omitempty"`
}

// EnvironmentService represents a sidecar service (database, cache, etc.)
type EnvironmentService struct {
	ID              uuid.UUID `json:"id"`
	EnvironmentID   uuid.UUID `json:"environment_id"`
	Name            string    `json:"name"`            // postgres, redis, mysql, etc.
	Type            string    `json:"type"`            // DATABASE, CACHE, MESSAGE_QUEUE, OTHER
	Image           string    `json:"image"`
	Port            int       `json:"port,omitempty"`
	Config          map[string]any `json:"config"`    // env vars, credentials
	State           string    `json:"state"`           // CREATED, RUNNING, STOPPED, FAILED
	ContainerID     string    `json:"container_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EnvironmentRepository defines the interface for environment persistence
type EnvironmentRepository interface {
	Create(ctx context.Context, env *Environment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Environment, error)
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*Environment, error)
	GetActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*Environment, error)
	Update(ctx context.Context, env *Environment) error
	UpdateState(ctx context.Context, id uuid.UUID, state EnvironmentState) error
	UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error
	UpdateImageID(ctx context.Context, id uuid.UUID, imageID uuid.UUID) error
	UpdateActivity(ctx context.Context, id uuid.UUID, activeSessions int) error
	Delete(ctx context.Context, id uuid.UUID) error // soft delete
	ListByState(ctx context.Context, state EnvironmentState) ([]*Environment, error)
	ListSuspended(ctx context.Context, before time.Time) ([]*Environment, error)
	ListIdleRunning(ctx context.Context, before time.Time) ([]*Environment, error)
	CountByStateAndRuntimeProfile(ctx context.Context, state string, runtimeProfileID uuid.UUID) (int64, error)
}

// EnvironmentServiceRepository defines the interface for environment service persistence
type EnvironmentServiceRepository interface {
	Create(ctx context.Context, svc *EnvironmentService) error
	GetByEnvironment(ctx context.Context, environmentID uuid.UUID) ([]*EnvironmentService, error)
	GetByID(ctx context.Context, id uuid.UUID) (*EnvironmentService, error)
	Update(ctx context.Context, svc *EnvironmentService) error
	Delete(ctx context.Context, id uuid.UUID) error
}