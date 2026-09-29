package repository

import (
	"context"
	"encoding/json"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// ChangeRequestRepository wraps the sqlc-generated queries
type ChangeRequestRepository struct {
	queries *Queries
}

func NewChangeRequestRepository(q *Queries) *ChangeRequestRepository {
	return &ChangeRequestRepository{queries: q}
}

func (r *ChangeRequestRepository) Create(ctx context.Context, cr *domain.ChangeRequest) error {
	commits, _ := json.Marshal(cr.Commits)
	files, _ := json.Marshal(cr.FilesChanged)
	_, err := r.queries.CreateChangeRequest(ctx, CreateChangeRequestParams{
		ProjectID:         cr.ProjectID,
		SourceWorkspaceID: cr.SourceWorkspaceID,
		TargetWorkspaceID: cr.TargetWorkspaceID,
		Title:             cr.Title,
		Description:       pgText(cr.Description),
		AuthorID:          cr.AuthorID,
		Commits:           commits,
		FilesChanged:      files,
	})
	return err
}

func (r *ChangeRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChangeRequest, error) {
	row, err := r.queries.GetChangeRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToChangeRequest(row), nil
}

func (r *ChangeRequestRepository) GetByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ChangeRequest, error) {
	rows, err := r.queries.GetChangeRequestsByProject(ctx, GetChangeRequestsByProjectParams{
		ProjectID: projectID,
		Limit:     100,
		Offset:    0,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ChangeRequest, len(rows))
	for i, row := range rows {
		result[i] = r.rowToChangeRequestFromProjectRow(row)
	}
	return result, nil
}

func (r *ChangeRequestRepository) GetBySourceWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.ChangeRequest, error) {
	rows, err := r.queries.GetChangeRequestsBySourceWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ChangeRequest, len(rows))
	for i, row := range rows {
		result[i] = r.rowToChangeRequestFromSourceRow(row)
	}
	return result, nil
}

func (r *ChangeRequestRepository) GetByState(ctx context.Context, state domain.ChangeRequestState) ([]*domain.ChangeRequest, error) {
	rows, err := r.queries.GetChangeRequestsByState(ctx, string(state))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ChangeRequest, len(rows))
	for i, row := range rows {
		result[i] = r.rowToChangeRequestFromStateRow(row)
	}
	return result, nil
}

func (r *ChangeRequestRepository) Update(ctx context.Context, cr *domain.ChangeRequest) error {
	commits, _ := json.Marshal(cr.Commits)
	files, _ := json.Marshal(cr.FilesChanged)
	_, err := r.queries.UpdateChangeRequest(ctx, UpdateChangeRequestParams{
		ID:              cr.ID,
		Title:           pgText(cr.Title),
		Description:     pgText(cr.Description),
		State:           pgText(string(cr.State)),
		Commits:         commits,
		FilesChanged:    files,
		ReviewerID:      uuidPtrToPgType(cr.ReviewerID),
		ReviewedAt:      pgTimestamptz(cr.ReviewedAt),
		MergedAt:        pgTimestamptz(cr.MergedAt),
		MergeCommitHash: pgText(cr.MergeCommitHash),
	})
	return err
}

func (r *ChangeRequestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteChangeRequest(ctx, id)
}

func (r *ChangeRequestRepository) rowToChangeRequest(row GetChangeRequestRow) *domain.ChangeRequest {
	return &domain.ChangeRequest{
		ID:                row.ID,
		ProjectID:         row.ProjectID,
		SourceWorkspaceID: row.SourceWorkspaceID,
		TargetWorkspaceID: row.TargetWorkspaceID,
		Title:             row.Title,
		Description:       row.Description.String,
		AuthorID:          row.AuthorID,
		State:             domain.ChangeRequestState(row.State),
		ReviewerID:        pgTypeToUUID(row.ReviewerID),
		ReviewedAt:        pgTimestamptzToTimePtr(row.ReviewedAt),
		MergedAt:          pgTimestamptzToTimePtr(row.MergedAt),
		MergeCommitHash:   row.MergeCommitHash.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func (r *ChangeRequestRepository) rowToChangeRequestFromProjectRow(row GetChangeRequestsByProjectRow) *domain.ChangeRequest {
	return &domain.ChangeRequest{
		ID:                row.ID,
		ProjectID:         row.ProjectID,
		SourceWorkspaceID: row.SourceWorkspaceID,
		TargetWorkspaceID: row.TargetWorkspaceID,
		Title:             row.Title,
		Description:       row.Description.String,
		AuthorID:          row.AuthorID,
		State:             domain.ChangeRequestState(row.State),
		ReviewerID:        pgTypeToUUID(row.ReviewerID),
		ReviewedAt:        pgTimestamptzToTimePtr(row.ReviewedAt),
		MergedAt:          pgTimestamptzToTimePtr(row.MergedAt),
		MergeCommitHash:   row.MergeCommitHash.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func (r *ChangeRequestRepository) rowToChangeRequestFromSourceRow(row GetChangeRequestsBySourceWorkspaceRow) *domain.ChangeRequest {
	return &domain.ChangeRequest{
		ID:                row.ID,
		ProjectID:         row.ProjectID,
		SourceWorkspaceID: row.SourceWorkspaceID,
		TargetWorkspaceID: row.TargetWorkspaceID,
		Title:             row.Title,
		Description:       row.Description.String,
		AuthorID:          row.AuthorID,
		State:             domain.ChangeRequestState(row.State),
		ReviewerID:        pgTypeToUUID(row.ReviewerID),
		ReviewedAt:        pgTimestamptzToTimePtr(row.ReviewedAt),
		MergedAt:          pgTimestamptzToTimePtr(row.MergedAt),
		MergeCommitHash:   row.MergeCommitHash.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func (r *ChangeRequestRepository) rowToChangeRequestFromStateRow(row GetChangeRequestsByStateRow) *domain.ChangeRequest {
	return &domain.ChangeRequest{
		ID:                row.ID,
		ProjectID:         row.ProjectID,
		SourceWorkspaceID: row.SourceWorkspaceID,
		TargetWorkspaceID: row.TargetWorkspaceID,
		Title:             row.Title,
		Description:       row.Description.String,
		AuthorID:          row.AuthorID,
		State:             domain.ChangeRequestState(row.State),
		ReviewerID:        pgTypeToUUID(row.ReviewerID),
		ReviewedAt:        pgTimestamptzToTimePtr(row.ReviewedAt),
		MergedAt:          pgTimestamptzToTimePtr(row.MergedAt),
		MergeCommitHash:   row.MergeCommitHash.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}
