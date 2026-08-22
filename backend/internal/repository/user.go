package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/AyushCN/berth/internal/domain"
)

// UserRepository wraps sqlc-generated queries.
type UserRepository struct {
	queries *Queries
}

func NewUserRepository(q *Queries) *UserRepository {
	return &UserRepository{queries: q}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	_, err := r.queries.CreateUser(ctx, CreateUserParams{
		Email:              u.Email,
		Username:           pgtype.Text{String: u.Username, Valid: true},
		GithubID:           pgtype.Text{String: u.GithubID, Valid: true},
		GithubUsername:     pgtype.Text{String: u.GithubUsername, Valid: true},
		AvatarUrl:          pgtype.Text{String: u.AvatarURL, Valid: true},
	})
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainUser(u), nil
}

func (r *UserRepository) GetByGithubID(ctx context.Context, githubID string) (*domain.User, error) {
	u, err := r.queries.GetUserByGithubID(ctx, pgtype.Text{String: githubID, Valid: true})
	if err != nil {
		return nil, err
	}
	return toDomainUser(u), nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	// Only update token for now
	return r.queries.UpdateUserToken(ctx, UpdateUserTokenParams{
		ID:                   u.ID,
		GithubTokenEncrypted: pgtype.Text{String: u.GithubTokenEncrypted, Valid: true},
	})
}

func toDomainUser(u User) *domain.User {
	return &domain.User{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username.String,
		GithubID:  u.GithubID.String,
		AvatarURL: u.AvatarUrl.String,
	}
}
