package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RuntimeProfileStatus represents the status of a runtime profile
type RuntimeProfileStatus string

const (
	RuntimeProfileStatusDetected   RuntimeProfileStatus = "DETECTED"
	RuntimeProfileStatusConfirmed  RuntimeProfileStatus = "CONFIRMED"
	RuntimeProfileStatusOverridden RuntimeProfileStatus = "OVERRIDDEN"
)

// RuntimeProfile represents the detected application specification
type RuntimeProfile struct {
	ID                uuid.UUID      `json:"id"`
	ProjectID         *uuid.UUID     `json:"project_id,omitempty"`
	WorkspaceID       *uuid.UUID     `json:"workspace_id,omitempty"`
	DetectionEvidence map[string]any `json:"detection_evidence"`
	Language          string         `json:"language"`
	Version           string         `json:"version,omitempty"`
	Framework         string         `json:"framework,omitempty"`
	PackageManager    string         `json:"package_manager,omitempty"`
	Architecture      string         `json:"architecture,omitempty"` // WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
	Entrypoint        string         `json:"entrypoint,omitempty"`
	BuildCommand      string         `json:"build_command,omitempty"`
	StartCommand      string         `json:"start_command,omitempty"`
	Port              int            `json:"port,omitempty"`

	// Legacy fields for analyzer compatibility
	BaseImage   string `json:"base_image,omitempty"`
	InstallCmd  string `json:"install_cmd,omitempty"`
	StartCmd    string `json:"start_cmd,omitempty"`
	ExposedPort int    `json:"exposed_port,omitempty"`
	WorkDir     string `json:"work_dir,omitempty"`

	DockerfileSource  string               `json:"dockerfile_source,omitempty"` // USER_PROVIDED, GENERATED, COMPOSE
	DockerfileContent string               `json:"dockerfile_content,omitempty"`
	RequiresDatabase  bool                 `json:"requires_database"`
	RequiresRedis     bool                 `json:"requires_redis"`
	Confidence        float64              `json:"confidence"`
	Status            RuntimeProfileStatus `json:"status"`
	ConfirmedBy       *uuid.UUID           `json:"confirmed_by,omitempty"`
	ConfirmedAt       *time.Time           `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

// RuntimeProfileRepository defines the interface for runtime profile persistence
type RuntimeProfileRepository interface {
	Create(ctx context.Context, profile *RuntimeProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*RuntimeProfile, error)
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]*RuntimeProfile, error)
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*RuntimeProfile, error)
	Update(ctx context.Context, profile *RuntimeProfile) error
	Delete(ctx context.Context, id uuid.UUID) error
}
