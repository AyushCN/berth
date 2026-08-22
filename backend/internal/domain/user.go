package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a platform user.
type User struct {
	ID                   uuid.UUID
	Email                string
	Username             string
	GithubID             string
	GithubUsername       string
	GithubTokenEncrypted string
	AvatarURL            string
	MaxSandboxes         int
	MaxBuildsPerHour     int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByGithubID(ctx context.Context, githubID string) (*User, error)
	Update(ctx context.Context, u *User) error
}
