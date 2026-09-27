package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BuildPlanStatus represents the status of a build plan
type BuildPlanStatus string

const (
	BuildPlanStatusPending BuildPlanStatus = "PENDING"
	BuildPlanStatusReady   BuildPlanStatus = "READY"
	BuildPlanStatusFailed  BuildPlanStatus = "FAILED"
)

// BuildPlan represents the plan for building a container image
type BuildPlan struct {
	ID              uuid.UUID       `json:"id"`
	RuntimeProfileID uuid.UUID      `json:"runtime_profile_id"`
	BaseImage       string          `json:"base_image"`
	Dockerfile      string          `json:"dockerfile"`
	BuildArgs       map[string]string `json:"build_args"`
	InstallCommand  string          `json:"install_command,omitempty"`
	BuildCommand    string          `json:"build_command,omitempty"`
	StartCommand    string          `json:"start_command"`
	WorkingDir      string          `json:"working_dir"`
	Port            int             `json:"port"`
	Confidence      float64         `json:"confidence"`
	Status          BuildPlanStatus `json:"status"`
	Error           string          `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// BuildPlanRepository defines the interface for build plan persistence
type BuildPlanRepository interface {
	Create(ctx context.Context, plan *BuildPlan) error
	GetByID(ctx context.Context, id uuid.UUID) (*BuildPlan, error)
	GetByRuntimeProfile(ctx context.Context, runtimeProfileID uuid.UUID) (*BuildPlan, error)
	Update(ctx context.Context, plan *BuildPlan) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BuildStatus represents the status of a build
type BuildStatus string

const (
	BuildStatusQueued     BuildStatus = "QUEUED"
	BuildStatusBuilding   BuildStatus = "BUILDING"
	BuildStatusSuccess    BuildStatus = "SUCCESS"
	BuildStatusFailed     BuildStatus = "FAILED"
	BuildStatusCancelled  BuildStatus = "CANCELLED"
)

// Build represents a container image build
type Build struct {
	ID              uuid.UUID   `json:"id"`
	BuildPlanID     uuid.UUID   `json:"build_plan_id"`
	WorkspaceID     uuid.UUID   `json:"workspace_id"`
	CommitHash      string      `json:"commit_hash"`
	Status          BuildStatus `json:"status"`
	ImageID         *uuid.UUID  `json:"image_id,omitempty"`
	Logs            string      `json:"logs,omitempty"`
	StartedAt       *time.Time  `json:"started_at,omitempty"`
	FinishedAt      *time.Time  `json:"finished_at,omitempty"`
	Error           string      `json:"error,omitempty"`
	CacheHit        bool        `json:"cache_hit"`
	BuildDurationMs int64       `json:"build_duration_ms,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

// BuildRepository defines the interface for build persistence
type BuildRepository interface {
	Create(ctx context.Context, build *Build) error
	GetByID(ctx context.Context, id uuid.UUID) (*Build, error)
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*Build, error)
	GetByBuildPlan(ctx context.Context, buildPlanID uuid.UUID) ([]*Build, error)
	GetLatestByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Build, error)
	Update(ctx context.Context, build *Build) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Image represents a built container image
type Image struct {
	ID          uuid.UUID `json:"id"`
	BuildID     *uuid.UUID `json:"build_id,omitempty"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Tag         string    `json:"tag"`
	Digest      string    `json:"digest,omitempty"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	BaseImage   string    `json:"base_image"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// ImageRepository defines the interface for image persistence
type ImageRepository interface {
	Create(ctx context.Context, image *Image) error
	GetByID(ctx context.Context, id uuid.UUID) (*Image, error)
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*Image, error)
	GetByTag(ctx context.Context, tag string) (*Image, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}