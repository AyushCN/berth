package repository

import (
	"context"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SandboxRepository struct {
	queries *Queries
}

func NewSandboxRepository(q *Queries) *SandboxRepository {
	return &SandboxRepository{queries: q}
}

func (r *SandboxRepository) Create(ctx context.Context, s *domain.Sandbox) error {
	_, err := r.queries.CreateSandbox(ctx, CreateSandboxParams{
		ProjectID: pgtype.UUID{},
		OwnerID:   s.OwnerID,
		Name:      s.Name,
		GitUrl:    s.GitURL,
		GitBranch: s.GitBranch,
		State:     string(s.State),
	})
	return err
}

func (r *SandboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sandbox, error) {
	s, err := r.queries.GetSandboxByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainSandbox(s), nil
}

func (r *SandboxRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Sandbox, error) {
	rows, err := r.queries.ListSandboxesByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	var result []*domain.Sandbox
	for _, s := range rows {
		result = append(result, toDomainSandbox(s))
	}
	return result, nil
}

func (r *SandboxRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Sandbox, error) {
	rows, err := r.queries.ListSandboxesByProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	var result []*domain.Sandbox
	for _, s := range rows {
		result = append(result, toDomainSandbox(s))
	}
	return result, nil
}

func (r *SandboxRepository) UpdateState(ctx context.Context, id uuid.UUID, state domain.SandboxState) error {
	return r.queries.UpdateSandboxState(ctx, UpdateSandboxStateParams{
		ID:    id,
		State: string(state),
	})
}

func (r *SandboxRepository) UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error {
	return r.queries.UpdateContainerID(ctx, UpdateContainerIDParams{
		ID:          id,
		ContainerID: pgtype.Text{String: containerID, Valid: true},
	})
}

func (r *SandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteSandbox(ctx, id)
}

func (r *SandboxRepository) CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return r.queries.CountSandboxesByOwner(ctx, ownerID)
}

func toDomainSandbox(s Sandbox) *domain.Sandbox {
	sb := &domain.Sandbox{
		ID:        s.ID,
		OwnerID:   s.OwnerID,
		Name:      s.Name,
		GitURL:    s.GitUrl,
		GitBranch: s.GitBranch,
		State:     domain.SandboxState(s.State),
		CreatedAt: s.CreatedAt.Time,
		UpdatedAt: s.UpdatedAt.Time,
	}
	if s.ContainerID.Valid {
		sb.ContainerID = &s.ContainerID.String
	}
	if s.PublicUrl.Valid {
		sb.PublicURL = &s.PublicUrl.String
	}
	return sb
}
