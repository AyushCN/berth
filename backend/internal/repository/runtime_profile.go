package repository

import (
	"context"
	"encoding/json"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// RuntimeProfileRepository wraps the sqlc-generated queries
type RuntimeProfileRepository struct {
	queries *Queries
}

func NewRuntimeProfileRepository(q *Queries) *RuntimeProfileRepository {
	return &RuntimeProfileRepository{queries: q}
}

func (r *RuntimeProfileRepository) Create(ctx context.Context, p *domain.RuntimeProfile) error {
	evidence, _ := json.Marshal(p.DetectionEvidence)
	_, err := r.queries.CreateRuntimeProfile(ctx, CreateRuntimeProfileParams{
		ProjectID:          uuidPtrToPgType(p.ProjectID),
		WorkspaceID:        uuidPtrToPgType(p.WorkspaceID),
		DetectionEvidence:  evidence,
		Language:           p.Language,
		Version:            pgText(p.Version),
		Framework:          pgText(p.Framework),
		PackageManager:     pgText(p.PackageManager),
		Architecture:       pgText(p.Architecture),
		Entrypoint:         pgText(p.Entrypoint),
		BuildCommand:       pgText(p.BuildCommand),
		StartCommand:       pgText(p.StartCommand),
		Port:               pgInt4(p.Port),
		DockerfileSource:   pgText(p.DockerfileSource),
		DockerfileContent:  pgText(p.DockerfileContent),
		RequiresDatabase:   pgBool(p.RequiresDatabase),
		RequiresRedis:      pgBool(p.RequiresRedis),
		Confidence:         float32(p.Confidence),
		Status:             string(p.Status),
		ConfirmedBy:        uuidPtrToPgType(p.ConfirmedBy),
		ConfirmedAt:        pgTimestamptz(p.ConfirmedAt),
	})
	return err
}

func (r *RuntimeProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RuntimeProfile, error) {
	row, err := r.queries.GetRuntimeProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToRuntimeProfile(row), nil
}

func (r *RuntimeProfileRepository) GetByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.RuntimeProfile, error) {
	rows, err := r.queries.GetRuntimeProfilesByProject(ctx, uuidToPgType(projectID))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.RuntimeProfile, len(rows))
	for i, row := range rows {
		result[i] = r.rowToRuntimeProfile(row)
	}
	return result, nil
}

func (r *RuntimeProfileRepository) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.RuntimeProfile, error) {
	rows, err := r.queries.GetRuntimeProfilesByWorkspace(ctx, uuidToPgType(workspaceID))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.RuntimeProfile, len(rows))
	for i, row := range rows {
		result[i] = r.rowToRuntimeProfile(row)
	}
	return result, nil
}

func (r *RuntimeProfileRepository) GetLatestByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.RuntimeProfile, error) {
	row, err := r.queries.GetLatestRuntimeProfileByWorkspace(ctx, uuidToPgType(workspaceID))
	if err != nil {
		return nil, err
	}
	return r.rowToRuntimeProfile(row), nil
}

func (r *RuntimeProfileRepository) Update(ctx context.Context, p *domain.RuntimeProfile) error {
	evidence, _ := json.Marshal(p.DetectionEvidence)
	_, err := r.queries.UpdateRuntimeProfile(ctx, UpdateRuntimeProfileParams{
		ID:                 p.ID,
		DetectionEvidence:  evidence,
		Language:           pgText(p.Language),
		Version:            pgText(p.Version),
		Framework:          pgText(p.Framework),
		PackageManager:     pgText(p.PackageManager),
		Architecture:       pgText(p.Architecture),
		Entrypoint:         pgText(p.Entrypoint),
		BuildCommand:       pgText(p.BuildCommand),
		StartCommand:       pgText(p.StartCommand),
		Port:               pgInt4(p.Port),
		DockerfileSource:   pgText(p.DockerfileSource),
		DockerfileContent:  pgText(p.DockerfileContent),
		RequiresDatabase:   pgBool(p.RequiresDatabase),
		RequiresRedis:      pgBool(p.RequiresRedis),
		Confidence:         pgFloat4(float32(p.Confidence)),
		Status:             pgText(string(p.Status)),
		ConfirmedBy:        uuidPtrToPgType(p.ConfirmedBy),
		ConfirmedAt:        pgTimestamptz(p.ConfirmedAt),
	})
	return err
}

func (r *RuntimeProfileRepository) Confirm(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.RuntimeProfile, error) {
	row, err := r.queries.ConfirmRuntimeProfile(ctx, ConfirmRuntimeProfileParams{
		ID:         id,
		ConfirmedBy: uuidToPgType(userID),
	})
	if err != nil {
		return nil, err
	}
	return r.rowToRuntimeProfile(row), nil
}

func (r *RuntimeProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteRuntimeProfile(ctx, id)
}

func (r *RuntimeProfileRepository) rowToRuntimeProfile(row RuntimeProfile) *domain.RuntimeProfile {
	var evidence map[string]any
	if len(row.DetectionEvidence) > 0 {
		json.Unmarshal(row.DetectionEvidence, &evidence)
	}
	return &domain.RuntimeProfile{
		ID:                row.ID,
		ProjectID:         pgTypeToUUID(row.ProjectID),
		WorkspaceID:       pgTypeToUUID(row.WorkspaceID),
		DetectionEvidence: evidence,
		Language:          row.Language,
		Version:           row.Version.String,
		Framework:         row.Framework.String,
		PackageManager:    row.PackageManager.String,
		Architecture:      row.Architecture.String,
		Entrypoint:        row.Entrypoint.String,
		BuildCommand:      row.BuildCommand.String,
		StartCommand:      row.StartCommand.String,
		Port:              int(row.Port.Int32),
		DockerfileSource:  row.DockerfileSource.String,
		DockerfileContent: row.DockerfileContent.String,
		RequiresDatabase:  row.RequiresDatabase.Bool,
		RequiresRedis:     row.RequiresRedis.Bool,
		Confidence:        float64(row.Confidence),
		Status:            domain.RuntimeProfileStatus(row.Status),
		ConfirmedBy:       pgTypeToUUID(row.ConfirmedBy),
		ConfirmedAt:       pgTimestamptzToTimePtr(row.ConfirmedAt),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}