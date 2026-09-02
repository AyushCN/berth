package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OrganizationRole string

const (
	OrgRoleOwner  OrganizationRole = "OWNER"
	OrgRoleAdmin  OrganizationRole = "ADMIN"
	OrgRoleMember OrganizationRole = "MEMBER"
)

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type OrganizationMember struct {
	ID             uuid.UUID        `json:"id"`
	OrganizationID uuid.UUID        `json:"organization_id"`
	UserID         uuid.UUID        `json:"user_id"`
	Role           OrganizationRole `json:"role"`
	CreatedAt      time.Time        `json:"created_at"`

	// Joined data from User
	Username  *string `json:"username,omitempty"`
	Email     *string `json:"email,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type OrganizationRepository interface {
	Create(ctx context.Context, name string) (*Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	AddMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role OrganizationRole) (*OrganizationMember, error)
	GetMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*OrganizationMember, error)
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]*OrganizationMember, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*Organization, error)
	UpdateRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role OrganizationRole) error
	RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error
}
