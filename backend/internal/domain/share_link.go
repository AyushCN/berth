package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ShareLink represents a project invitation link
type ShareLink struct {
	ID        uuid.UUID  `json:"id"`
	ProjectID uuid.UUID  `json:"project_id"`
	Code      string     `json:"code"`
	Role      string     `json:"role"` // VIEWER, EDITOR
	CreatedBy uuid.UUID  `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UsesCount int        `json:"uses_count"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// ShareLinkRepository defines the interface for share link persistence
type ShareLinkRepository interface {
	Create(ctx context.Context, link *ShareLink) error
	GetByID(ctx context.Context, id uuid.UUID) (*ShareLink, error)
	GetByCode(ctx context.Context, code string) (*ShareLink, error)
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]*ShareLink, error)
	Update(ctx context.Context, link *ShareLink) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementUses(ctx context.Context, id uuid.UUID) error
}
