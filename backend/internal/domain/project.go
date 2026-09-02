package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProjectRole string

const (
	ProjectRoleOwner       ProjectRole = "OWNER"
	ProjectRoleCollaborator ProjectRole = "COLLABORATOR"
	ProjectRoleViewer      ProjectRole = "VIEWER"
)

type Project struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Description         *string   `json:"description,omitempty"`
	OwnerOrganizationID uuid.UUID `json:"owner_organization_id"`
	CreatedByUserID     uuid.UUID `json:"created_by_user_id"`
	IsPublic            bool      `json:"is_public"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProjectCollaborator struct {
	ID              uuid.UUID   `json:"id"`
	ProjectID       uuid.UUID   `json:"project_id"`
	UserID          uuid.UUID   `json:"user_id"`
	Role            ProjectRole `json:"role"`
	InvitedByUserID *uuid.UUID  `json:"invited_by_user_id,omitempty"`
	InvitedAt       time.Time   `json:"invited_at"`
	AcceptedAt      *time.Time  `json:"accepted_at,omitempty"`

	// Joined data
	Username  *string `json:"username,omitempty"`
	Email     *string `json:"email,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	ListForOrg(ctx context.Context, orgID uuid.UUID) ([]*Project, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Project, error)
	AddCollaborator(ctx context.Context, pc *ProjectCollaborator) error
	GetCollaborator(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*ProjectCollaborator, error)
	ListCollaborators(ctx context.Context, projectID uuid.UUID) ([]*ProjectCollaborator, error)
	UpdateRole(ctx context.Context, projectID uuid.UUID, userID uuid.UUID, role ProjectRole) error
	RemoveCollaborator(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) error
	ListSandboxes(ctx context.Context, projectID uuid.UUID) ([]*Sandbox, error)
}
