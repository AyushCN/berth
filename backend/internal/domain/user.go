package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a platform user.
type User struct {
	ID                   uuid.UUID `json:"id"`
	Email                string    `json:"email"`
	Username             string    `json:"username"`
	GithubID             string    `json:"github_id"`
	GithubUsername       string    `json:"github_username"`
	GithubTokenEncrypted string    `json:"-"`
	AvatarURL            string    `json:"avatar_url"`
	MaxSandboxes         int       `json:"max_sandboxes"`
	MaxBuildsPerHour     int       `json:"max_builds_per_hour"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByGithubID(ctx context.Context, githubID string) (*User, error)
	Update(ctx context.Context, u *User) error
}
