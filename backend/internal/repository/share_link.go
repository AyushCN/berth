package repository

import (
	"context"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// ShareLinkRepository wraps the sqlc-generated queries
type ShareLinkRepository struct {
	queries *Queries
}

func NewShareLinkRepository(q *Queries) *ShareLinkRepository {
	return &ShareLinkRepository{queries: q}
}

func (r *ShareLinkRepository) Create(ctx context.Context, link *domain.ShareLink) error {
	maxUses := 0
	if link.MaxUses != nil {
		maxUses = *link.MaxUses
	}
	_, err := r.queries.CreateShareLink(ctx, CreateShareLinkParams{
		ProjectID: link.ProjectID,
		Code:      link.Code,
		Role:      link.Role,
		CreatedBy: link.CreatedBy,
		ExpiresAt: pgTimestamptz(link.ExpiresAt),
		MaxUses:   pgInt4(maxUses),
	})
	return err
}

func (r *ShareLinkRepository) GetByCode(ctx context.Context, code string) (*domain.ShareLink, error) {
	row, err := r.queries.GetShareLinkByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return r.rowToShareLink(row), nil
}

func (r *ShareLinkRepository) GetByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ShareLink, error) {
	rows, err := r.queries.GetShareLinksByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ShareLink, len(rows))
	for i, row := range rows {
		result[i] = r.rowToShareLink(row)
	}
	return result, nil
}

func (r *ShareLinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ShareLink, error) {
	row, err := r.queries.GetShareLinkByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToShareLink(row), nil
}

func (r *ShareLinkRepository) Update(ctx context.Context, link *domain.ShareLink) error {
	_, err := r.queries.UpdateShareLink(ctx, UpdateShareLinkParams{
		ID:        link.ID,
		RevokedAt: pgTimestamptz(link.RevokedAt),
	})
	return err
}

func (r *ShareLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteShareLink(ctx, id)
}

func (r *ShareLinkRepository) IncrementUses(ctx context.Context, id uuid.UUID) error {
	return r.queries.IncrementShareLinkUses(ctx, id)
}

func (r *ShareLinkRepository) rowToShareLink(row ShareLink) *domain.ShareLink {
	return &domain.ShareLink{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Code:      row.Code,
		Role:      row.Role,
		CreatedBy: row.CreatedBy,
		ExpiresAt: pgTimestamptzToTimePtr(row.ExpiresAt),
		MaxUses:   pgInt4ToIntPtr(row.MaxUses),
		UsesCount: int(row.UsesCount.Int32),
		CreatedAt: row.CreatedAt.Time,
		RevokedAt: pgTimestamptzToTimePtr(row.RevokedAt),
	}
}
