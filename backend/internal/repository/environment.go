package repository

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// EnvironmentRepository wraps the sqlc-generated queries for environment operations
type EnvironmentRepository struct {
	queries *Queries
}

func NewEnvironmentRepository(q *Queries) *EnvironmentRepository {
	return &EnvironmentRepository{queries: q}
}

func (r *EnvironmentRepository) Create(ctx context.Context, env *domain.Environment) error {
	_, err := r.queries.CreateEnvironment(ctx, CreateEnvironmentParams{
		WorkspaceID:      env.WorkspaceID,
		RuntimeProfileID: uuidPtrToPgType(env.RuntimeProfileID),
		Name:             env.Name,
		State:            string(env.State),
		ContainerID:      pgText(env.ContainerID),
		ImageID:          uuidPtrToPgType(env.ImageID),
		PublicUrl:        pgText(env.PublicURL),
		Port:             pgInt4(env.Port),
		MemoryLimit:      pgInt8(env.MemoryLimit),
		CpuLimit:         pgInt8(env.CPULimit),
	})
	return err
}

func (r *EnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	row, err := r.queries.GetEnvironment(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToEnvironment(row), nil
}

func (r *EnvironmentRepository) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Environment, error) {
	rows, err := r.queries.GetEnvironmentsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Environment, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEnvironment(row)
	}
	return result, nil
}

func (r *EnvironmentRepository) GetActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Environment, error) {
	rows, err := r.queries.GetActiveEnvironmentsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Environment, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEnvironment(row)
	}
	return result, nil
}

func (r *EnvironmentRepository) Update(ctx context.Context, env *domain.Environment) error {
	_, err := r.queries.UpdateEnvironment(ctx, UpdateEnvironmentParams{
		ID:                env.ID,
		Name:              pgText(env.Name),
		RuntimeProfileID:  uuidPtrToPgType(env.RuntimeProfileID),
		ContainerID:       pgText(env.ContainerID),
		ImageID:           uuidPtrToPgType(env.ImageID),
		PublicUrl:         pgText(env.PublicURL),
		Port:              pgInt4(env.Port),
		MemoryLimit:       pgInt8(env.MemoryLimit),
		CpuLimit:          pgInt8(env.CPULimit),
		LastActivityAt:    pgTimestamptz(env.LastActivityAt),
		ActiveSessions:    pgInt4(env.ActiveSessions),
		SuspendedAt:       pgTimestamptz(env.SuspendedAt),
		LastError:         pgText(env.LastError),
		RestartCount:      pgInt4(env.RestartCount),
	})
	return err
}

func (r *EnvironmentRepository) UpdateState(ctx context.Context, id uuid.UUID, state domain.EnvironmentState) error {
	_, err := r.queries.UpdateEnvironmentState(ctx, UpdateEnvironmentStateParams{
		ID:       id,
		State:    string(state),
		LastError: pgText(""),
	})
	return err
}

func (r *EnvironmentRepository) UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error {
	_, err := r.queries.UpdateEnvironmentContainerID(ctx, UpdateEnvironmentContainerIDParams{
		ID:           id,
		ContainerID:  pgText(containerID),
	})
	return err
}

func (r *EnvironmentRepository) UpdateImageID(ctx context.Context, id uuid.UUID, imageID uuid.UUID) error {
	_, err := r.queries.UpdateEnvironmentImageID(ctx, UpdateEnvironmentImageIDParams{
		ID:      id,
		ImageID: uuidToPgType(imageID),
	})
	return err
}

func (r *EnvironmentRepository) UpdateActivity(ctx context.Context, id uuid.UUID, activeSessions int) error {
	_, err := r.queries.UpdateEnvironmentActivity(ctx, UpdateEnvironmentActivityParams{
		ID:              id,
		ActiveSessions:  pgInt4(activeSessions),
	})
	return err
}

func (r *EnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteEnvironment(ctx, id)
}

func (r *EnvironmentRepository) ListByState(ctx context.Context, state domain.EnvironmentState) ([]*domain.Environment, error) {
	rows, err := r.queries.ListEnvironmentsByState(ctx, string(state))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Environment, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEnvironment(row)
	}
	return result, nil
}

func (r *EnvironmentRepository) ListSuspended(ctx context.Context, before time.Time) ([]*domain.Environment, error) {
	rows, err := r.queries.ListSuspendedEnvironments(ctx, pgTimestamptz(&before))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Environment, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEnvironment(row)
	}
	return result, nil
}

func (r *EnvironmentRepository) ListIdleRunning(ctx context.Context, before time.Time) ([]*domain.Environment, error) {
	rows, err := r.queries.ListIdleRunningEnvironments(ctx, pgTimestamptz(&before))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Environment, len(rows))
	for i, row := range rows {
		result[i] = r.rowToEnvironment(row)
	}
	return result, nil
}

func (r *EnvironmentRepository) CountByStateAndRuntimeProfile(ctx context.Context, state string, runtimeProfileID uuid.UUID) (int64, error) {
	count, err := r.queries.CountEnvironmentsByStateAndRuntimeProfile(ctx, CountEnvironmentsByStateAndRuntimeProfileParams{
		State:            state,
		RuntimeProfileID: pgtype.UUID{Bytes: runtimeProfileID, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Environment Services

func (r *EnvironmentRepository) CreateService(ctx context.Context, svc *domain.EnvironmentService) error {
	_, err := r.queries.CreateEnvironmentService(ctx, CreateEnvironmentServiceParams{
		EnvironmentID: svc.EnvironmentID,
		Name:          svc.Name,
		Type:          svc.Type,
		Image:         svc.Image,
		Port:          pgInt4(svc.Port),
		Config:        mapToJSONB(svc.Config),
		State:         svc.State,
		ContainerID:   pgText(svc.ContainerID),
	})
	return err
}

func (r *EnvironmentRepository) GetServices(ctx context.Context, environmentID uuid.UUID) ([]*domain.EnvironmentService, error) {
	rows, err := r.queries.GetEnvironmentServices(ctx, environmentID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.EnvironmentService, len(rows))
	for i, row := range rows {
		result[i] = &domain.EnvironmentService{
			ID:            row.ID,
			EnvironmentID: row.EnvironmentID,
			Name:          row.Name,
			Type:          row.Type,
			Image:         row.Image,
			Port:          int(row.Port.Int32),
			Config:        jsonbToMap(row.Config),
			State:         row.State,
			ContainerID:   row.ContainerID.String,
			CreatedAt:     row.CreatedAt.Time,
			UpdatedAt:     row.UpdatedAt.Time,
		}
	}
	return result, nil
}

func (r *EnvironmentRepository) GetService(ctx context.Context, id uuid.UUID) (*domain.EnvironmentService, error) {
	row, err := r.queries.GetEnvironmentService(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.EnvironmentService{
		ID:            row.ID,
		EnvironmentID: row.EnvironmentID,
		Name:          row.Name,
		Type:          row.Type,
		Image:         row.Image,
		Port:          int(row.Port.Int32),
		Config:        jsonbToMap(row.Config),
		State:         row.State,
		ContainerID:   row.ContainerID.String,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}, nil
}

func (r *EnvironmentRepository) UpdateService(ctx context.Context, svc *domain.EnvironmentService) error {
	_, err := r.queries.UpdateEnvironmentService(ctx, UpdateEnvironmentServiceParams{
		ID:        svc.ID,
		State:     pgText(svc.State),
		ContainerID: pgText(svc.ContainerID),
		Config:    mapToJSONB(svc.Config),
	})
	return err
}

func (r *EnvironmentRepository) DeleteService(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteEnvironmentService(ctx, id)
}

func (r *EnvironmentRepository) rowToEnvironment(row Environment) *domain.Environment {
	return &domain.Environment{
		ID:               row.ID,
		WorkspaceID:      row.WorkspaceID,
		RuntimeProfileID: pgTypeToUUID(row.RuntimeProfileID),
		Name:             row.Name,
		State:            domain.EnvironmentState(row.State),
		ContainerID:      row.ContainerID.String,
		ImageID:          pgTypeToUUID(row.ImageID),
		PublicURL:        row.PublicUrl.String,
		Port:             int(row.Port.Int32),
		MemoryLimit:      row.MemoryLimit.Int64,
		CPULimit:         row.CpuLimit.Int64,
		LastActivityAt:   pgTimestamptzToTimePtr(row.LastActivityAt),
		ActiveSessions:   int(row.ActiveSessions.Int32),
		SuspendedAt:      pgTimestamptzToTimePtr(row.SuspendedAt),
		LastError:        row.LastError.String,
		RestartCount:     int(row.RestartCount.Int32),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		DeletedAt:        pgTimestamptzToTimePtr(row.DeletedAt),
	}
}