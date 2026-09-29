package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// WorkspaceType represents the type of workspace
type WorkspaceType string

const (
	WorkspaceTypeCanonical WorkspaceType = "CANONICAL"
	WorkspaceTypeFork      WorkspaceType = "FORK"
)

// WorkspaceMemberRole represents the role of a user in a workspace
type WorkspaceMemberRole string

const (
	WorkspaceMemberRoleOwner   WorkspaceMemberRole = "OWNER"
	WorkspaceMemberRoleEditor  WorkspaceMemberRole = "EDITOR"
	WorkspaceMemberRoleViewer  WorkspaceMemberRole = "VIEWER"
)

// Workspace represents an editable code state (canonical or fork)
type Workspace struct {
	ID                  uuid.UUID      `json:"id"`
	ProjectID           uuid.UUID      `json:"project_id"`
	Name                string         `json:"name"`
	Type                WorkspaceType  `json:"type"` // CANONICAL, FORK
	BaseWorkspaceID     *uuid.UUID     `json:"base_workspace_id,omitempty"` // for forks
	OwnerID             uuid.UUID      `json:"owner_id"` // workspace owner
	GitURL              string         `json:"git_url"`
	GitBranch           string         `json:"git_branch"`
	CommitHash          string         `json:"commit_hash,omitempty"`
	HasUncommittedChanges bool         `json:"has_uncommitted_changes"`
	LastSyncedAt        *time.Time     `json:"last_synced_at,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           *time.Time     `json:"deleted_at,omitempty"`
}

// WorkspaceMember represents a user's membership in a workspace
type WorkspaceMember struct {
	ID          uuid.UUID            `json:"id"`
	WorkspaceID uuid.UUID            `json:"workspace_id"`
	UserID      uuid.UUID            `json:"user_id"`
	Role        WorkspaceMemberRole  `json:"role"` // OWNER, EDITOR, VIEWER
	CreatedAt   time.Time            `json:"created_at"`
}

// WorkspaceRepository defines the interface for workspace persistence
type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	GetByID(ctx context.Context, id uuid.UUID) (*Workspace, error)
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]*Workspace, error)
	GetForks(ctx context.Context, baseWorkspaceID uuid.UUID) ([]*Workspace, error)
	GetCanonical(ctx context.Context, projectID uuid.UUID) (*Workspace, error)
	Update(ctx context.Context, workspace *Workspace) error
	Delete(ctx context.Context, id uuid.UUID) error // soft delete
	GetUserWorkspaces(ctx context.Context, userID uuid.UUID) ([]*Workspace, error)
}

// WorkspaceMemberRepository defines the interface for workspace member persistence
type WorkspaceMemberRepository interface {
	Create(ctx context.Context, member *WorkspaceMember) error
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*WorkspaceMember, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*WorkspaceMember, error)
	Get(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error)
	Update(ctx context.Context, member *WorkspaceMember) error
	Delete(ctx context.Context, workspaceID, userID uuid.UUID) error
}