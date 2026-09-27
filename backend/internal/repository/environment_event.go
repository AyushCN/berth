package repository

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// EnvironmentEventRepository wraps the sqlc-generated queries
type EnvironmentEventRepository struct {
	queries *Queries
}

func NewEnvironmentEventRepository(q *Queries) *EnvironmentEventRepository {
	return &EnvironmentEventRepository{queries: q}
}

func (r *EnvironmentEventRepository) Create(ctx context.Context, event *domain.EnvironmentEvent) error {
	_, err := r.queries.CreateEnvironmentEvent(ctx, CreateEnvironmentEventParams{
		EnvironmentID: uuidFromPtr(event.EnvironmentID),
		WorkspaceID:   event.WorkspaceID,
		ProjectID:     event.ProjectID,
		UserID:        uuidPtrToPgType(event.UserID),
		Type:          string(event.Type),
		Payload:       mapToJSONB(event.Payload),
	})
	return err
}

func (r *EnvironmentEventRepository) GetByEnvironment(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*domain.EnvironmentEvent, error) {
	rows, err := r.queries.GetEnvironmentEvents(ctx, GetEnvironmentEventsParams{
		EnvironmentID: environmentID,
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.EnvironmentEvent, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEvent(row)
	}
	return result, nil
}

func (r *EnvironmentEventRepository) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.EnvironmentEvent, error) {
	rows, err := r.queries.GetWorkspaceEvents(ctx, GetWorkspaceEventsParams{
		WorkspaceID: workspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.EnvironmentEvent, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEventFromWorkspaceRow(row)
	}
	return result, nil
}

func (r *EnvironmentEventRepository) GetByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*domain.EnvironmentEvent, error) {
	rows, err := r.queries.GetProjectEvents(ctx, GetProjectEventsParams{
		ProjectID: projectID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.EnvironmentEvent, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEventFromProjectRow(row)
	}
	return result, nil
}

func (r *EnvironmentEventRepository) GetRecent(ctx context.Context, projectID uuid.UUID, since time.Time, limit int) ([]*domain.EnvironmentEvent, error) {
	rows, err := r.queries.GetRecentProjectEvents(ctx, GetRecentProjectEventsParams{
		ProjectID: projectID,
		CreatedAt: pgTimestamptz(&since),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.EnvironmentEvent, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEventFromRecentRow(row)
	}
	return result, nil
}

func (r *EnvironmentEventRepository) rowToEvent(row GetEnvironmentEventsRow) *domain.EnvironmentEvent {
	return &domain.EnvironmentEvent{
		ID:            row.ID,
		EnvironmentID: uuidPtr(&row.EnvironmentID),
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		UserID:        pgTypeToUUID(row.UserID),
		Type:          domain.EnvironmentEventType(row.Type),
		Payload:       jsonbToMap(row.Payload),
		CreatedAt:     row.CreatedAt.Time,
	}
}

func (r *EnvironmentEventRepository) rowToEventFromWorkspaceRow(row GetWorkspaceEventsRow) *domain.EnvironmentEvent {
	return &domain.EnvironmentEvent{
		ID:            row.ID,
		EnvironmentID: uuidPtr(&row.EnvironmentID),
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		UserID:        pgTypeToUUID(row.UserID),
		Type:          domain.EnvironmentEventType(row.Type),
		Payload:       jsonbToMap(row.Payload),
		CreatedAt:     row.CreatedAt.Time,
	}
}

func (r *EnvironmentEventRepository) rowToEventFromProjectRow(row GetProjectEventsRow) *domain.EnvironmentEvent {
	return &domain.EnvironmentEvent{
		ID:            row.ID,
		EnvironmentID: uuidPtr(&row.EnvironmentID),
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		UserID:        pgTypeToUUID(row.UserID),
		Type:          domain.EnvironmentEventType(row.Type),
		Payload:       jsonbToMap(row.Payload),
		CreatedAt:     row.CreatedAt.Time,
	}
}

func (r *EnvironmentEventRepository) rowToEventFromRecentRow(row GetRecentProjectEventsRow) *domain.EnvironmentEvent {
	envID := uuidToPgType(row.EnvironmentID)
	userID := row.UserID
	return &domain.EnvironmentEvent{
		ID:            row.ID,
		EnvironmentID: pgTypeToUUID(envID),
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		UserID:        pgTypeToUUID(userID),
		Type:          domain.EnvironmentEventType(row.Type),
		Payload:       jsonbToMap(row.Payload),
		CreatedAt:     row.CreatedAt.Time,
	}
}