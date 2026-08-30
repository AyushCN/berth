package repository

import (
	"context"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ domain.SandboxRepository = (*SandboxRepository)(nil)

type SandboxRepository struct {
	queries *Queries
}

func NewSandboxRepository(q *Queries) *SandboxRepository {
	return &SandboxRepository{queries: q}
}

func (r *SandboxRepository) Create(ctx context.Context, s *domain.Sandbox) error {
	created, err := r.queries.CreateSandbox(ctx, CreateSandboxParams{
		ProjectID:          pgtype.UUID{},
		OwnerID:            s.OwnerID,
		Name:               s.Name,
		GitUrl:             s.GitURL,
		GitBranch:          s.GitBranch,
		State:              string(s.State),
		RuntimeLanguage:    pgtype.Text{},
		RuntimeBaseImage:   pgtype.Text{},
		RuntimePort:        pgtype.Int4{},
		NeedsDb:            pgtype.Bool{},
	})
	if err == nil {
		s.ID = created.ID
		s.CreatedAt = created.CreatedAt.Time
		s.UpdatedAt = created.UpdatedAt.Time
	}
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
	return r.queries.UpdateSandboxContainer(ctx, UpdateSandboxContainerParams{
		ID:          id,
		ContainerID: pgtype.Text{String: containerID, Valid: true},
		PublicUrl:   pgtype.Text{Valid: false},
	})
}

func (r *SandboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteSandbox(ctx, id)
}

func (r *SandboxRepository) CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return r.queries.CountSandboxesByOwner(ctx, ownerID)
}

func (r *SandboxRepository) PopPendingSandbox(ctx context.Context) (*domain.Sandbox, error) {
	s, err := r.queries.PopPendingSandbox(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainSandbox(s), nil
}

func (r *SandboxRepository) UpdateGitTracking(ctx context.Context, id uuid.UUID, hasChanges bool, modifiedBy *uuid.UUID, commitHash *string) error {
	var modifiedByUUID pgtype.UUID
	if modifiedBy != nil {
		modifiedByUUID = pgtype.UUID{Bytes: *modifiedBy, Valid: true}
	}
	var commitHashStr pgtype.Text
	if commitHash != nil {
		commitHashStr = pgtype.Text{String: *commitHash, Valid: true}
	}
	return r.queries.UpdateSandboxGitTracking(ctx, UpdateSandboxGitTrackingParams{
		ID:                    id,
		HasUncommittedChanges: pgtype.Bool{Bool: hasChanges, Valid: true},
		ModifiedByUserID:      modifiedByUUID,
		CommitHash:            commitHashStr,
	})
}

func (r *SandboxRepository) LogActivity(ctx context.Context, sandboxID uuid.UUID, userID uuid.UUID, activityType string, data []byte) error {
	_, err := r.queries.CreateSandboxActivity(ctx, CreateSandboxActivityParams{
		SandboxID:    pgtype.UUID{Bytes: sandboxID, Valid: true},
		ActivityType: activityType,
		Data:         data,
		UserID:       pgtype.UUID{Bytes: userID, Valid: true},
	})
	return err
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
	if s.HasUncommittedChanges.Valid {
		sb.HasUncommittedChanges = s.HasUncommittedChanges.Bool
	}
	if s.LastModifiedAt.Valid {
		sb.LastModifiedAt = &s.LastModifiedAt.Time
	}
	if s.ModifiedByUserID.Valid {
		id := s.ModifiedByUserID.Bytes
		parsed, _ := uuid.FromBytes(id[:])
		sb.ModifiedByUserID = &parsed
	}
	if s.CommitHash.Valid {
		sb.CommitHash = &s.CommitHash.String
	}
	return sb
}
