package repository

import (
	"context"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// WorkspaceRepository wraps the sqlc-generated queries for workspace operations
type WorkspaceRepository struct {
	queries *Queries
}

func NewWorkspaceRepository(q *Queries) *WorkspaceRepository {
	return &WorkspaceRepository{queries: q}
}

func (r *WorkspaceRepository) Create(ctx context.Context, w *domain.Workspace) error {
	created, err := r.queries.CreateWorkspace(ctx, CreateWorkspaceParams{
		ProjectID:       w.ProjectID,
		Name:            w.Name,
		Type:            string(w.Type),
		BaseWorkspaceID: uuidPtrToPgType(w.BaseWorkspaceID),
		OwnerID:         w.OwnerID,
		GitBranch:       w.GitBranch,
		GitUrl:          w.GitURL,
	})
	if err != nil {
		return err
	}
	// Update the workspace with database-generated values
	w.ID = created.ID
	w.CreatedAt = created.CreatedAt.Time
	w.UpdatedAt = created.UpdatedAt.Time
	w.DeletedAt = pgTimestamptzToTimePtr(created.DeletedAt)
	return nil
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	row, err := r.queries.GetWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToWorkspace(row), nil
}

func (r *WorkspaceRepository) GetByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Workspace, error) {
	rows, err := r.queries.GetWorkspacesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Workspace, len(rows))
	for i, row := range rows {
		result[i] = r.rowToWorkspace(row)
	}
	return result, nil
}

func (r *WorkspaceRepository) GetForks(ctx context.Context, baseWorkspaceID uuid.UUID) ([]*domain.Workspace, error) {
	rows, err := r.queries.GetForkWorkspaces(ctx, uuidToPgType(baseWorkspaceID))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Workspace, len(rows))
	for i, row := range rows {
		result[i] = r.rowToWorkspace(row)
	}
	return result, nil
}

func (r *WorkspaceRepository) GetCanonical(ctx context.Context, projectID uuid.UUID) (*domain.Workspace, error) {
	row, err := r.queries.GetCanonicalWorkspace(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return r.rowToWorkspace(row), nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, w *domain.Workspace) error {
	_, err := r.queries.UpdateWorkspace(ctx, UpdateWorkspaceParams{
		ID:                    w.ID,
		Name:                  pgText(w.Name),
		GitBranch:             pgText(w.GitBranch),
		CommitHash:            pgText(w.CommitHash),
		HasUncommittedChanges: pgBool(w.HasUncommittedChanges),
		LastSyncedAt:          pgTimestamptz(w.LastSyncedAt),
	})
	return err
}

func (r *WorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.SoftDeleteWorkspace(ctx, id)
}

// Workspace Members

func (r *WorkspaceRepository) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceMemberRole) error {
	_, err := r.queries.CreateWorkspaceMember(ctx, CreateWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        string(role),
	})
	return err
}

func (r *WorkspaceRepository) GetMembers(ctx context.Context, workspaceID uuid.UUID) ([]*domain.WorkspaceMember, error) {
	rows, err := r.queries.GetWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WorkspaceMember, len(rows))
	for i, row := range rows {
		result[i] = &domain.WorkspaceMember{
			ID:          row.ID,
			WorkspaceID: row.WorkspaceID,
			UserID:      row.UserID,
			Role:        domain.WorkspaceMemberRole(row.Role),
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return result, nil
}

func (r *WorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	row, err := r.queries.GetWorkspaceMember(ctx, GetWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		return nil, err
	}
	return &domain.WorkspaceMember{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        domain.WorkspaceMemberRole(row.Role),
		CreatedAt:   row.CreatedAt.Time,
	}, nil
}

func (r *WorkspaceRepository) GetUserWorkspaces(ctx context.Context, userID uuid.UUID) ([]*domain.Workspace, error) {
	rows, err := r.queries.GetUserWorkspaces(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Workspace, len(rows))
	for i, row := range rows {
		result[i] = r.rowToWorkspaceWithRole(row)
	}
	return result, nil
}

func (r *WorkspaceRepository) UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceMemberRole) error {
	_, err := r.queries.UpdateWorkspaceMember(ctx, UpdateWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        string(role),
	})
	return err
}

func (r *WorkspaceRepository) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	return r.queries.DeleteWorkspaceMember(ctx, DeleteWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
}

func (r *WorkspaceRepository) rowToWorkspace(row Workspace) *domain.Workspace {
	return &domain.Workspace{
		ID:                    row.ID,
		ProjectID:             row.ProjectID,
		Name:                  row.Name,
		Type:                  domain.WorkspaceType(row.Type),
		BaseWorkspaceID:       pgTypeToUUID(row.BaseWorkspaceID),
		OwnerID:               row.OwnerID,
		GitURL:                row.GitUrl,
		GitBranch:             row.GitBranch,
		CommitHash:            row.CommitHash.String,
		HasUncommittedChanges: row.HasUncommittedChanges.Bool,
		LastSyncedAt:          pgTimestamptzToTimePtr(row.LastSyncedAt),
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
		DeletedAt:             pgTimestamptzToTimePtr(row.DeletedAt),
	}
}

func (r *WorkspaceRepository) rowToWorkspaceWithRole(row GetUserWorkspacesRow) *domain.Workspace {
	return &domain.Workspace{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Type:      domain.WorkspaceType(row.Type),
		BaseWorkspaceID: pgTypeToUUID(row.BaseWorkspaceID),
		OwnerID:   row.OwnerID,
		GitURL:    row.GitUrl,
		GitBranch: row.GitBranch,
		CommitHash: row.CommitHash.String,
		HasUncommittedChanges: row.HasUncommittedChanges.Bool,
		LastSyncedAt: pgTimestamptzToTimePtr(row.LastSyncedAt),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
		DeletedAt: pgTimestamptzToTimePtr(row.DeletedAt),
	}
}

// WorkspaceMemberRepository wraps the sqlc-generated queries for workspace member operations
type WorkspaceMemberRepository struct {
	queries *Queries
}

func NewWorkspaceMemberRepository(q *Queries) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{queries: q}
}

func (r *WorkspaceMemberRepository) Create(ctx context.Context, member *domain.WorkspaceMember) error {
	_, err := r.queries.CreateWorkspaceMember(ctx, CreateWorkspaceMemberParams{
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		Role:        string(member.Role),
	})
	return err
}

func (r *WorkspaceMemberRepository) Get(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	row, err := r.queries.GetWorkspaceMember(ctx, GetWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		return nil, err
	}
	return &domain.WorkspaceMember{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        domain.WorkspaceMemberRole(row.Role),
		CreatedAt:   row.CreatedAt.Time,
	}, nil
}

func (r *WorkspaceMemberRepository) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.WorkspaceMember, error) {
	rows, err := r.queries.GetWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WorkspaceMember, len(rows))
	for i, row := range rows {
		result[i] = &domain.WorkspaceMember{
			ID:          row.ID,
			WorkspaceID: row.WorkspaceID,
			UserID:      row.UserID,
			Role:        domain.WorkspaceMemberRole(row.Role),
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return result, nil
}

func (r *WorkspaceMemberRepository) Update(ctx context.Context, member *domain.WorkspaceMember) error {
	_, err := r.queries.UpdateWorkspaceMember(ctx, UpdateWorkspaceMemberParams{
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		Role:        string(member.Role),
	})
	return err
}

func (r *WorkspaceMemberRepository) Delete(ctx context.Context, workspaceID, userID uuid.UUID) error {
	return r.queries.DeleteWorkspaceMember(ctx, DeleteWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
}

func (r *WorkspaceMemberRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.WorkspaceMember, error) {
	rows, err := r.queries.GetWorkspaceMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.WorkspaceMember, len(rows))
	for i, row := range rows {
		result[i] = &domain.WorkspaceMember{
			ID:          row.ID,
			WorkspaceID: row.WorkspaceID,
			UserID:      row.UserID,
			Role:        domain.WorkspaceMemberRole(row.Role),
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return result, nil
}