package repository

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ domain.ProjectRepository = (*ProjectRepository)(nil)

type ProjectRepository struct {
	queries *Queries
}

func NewProjectRepository(q *Queries) *ProjectRepository {
	return &ProjectRepository{queries: q}
}

func (r *ProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	var desc pgtype.Text
	if p.Description != nil {
		desc = pgtype.Text{String: *p.Description, Valid: true}
	}
	created, err := r.queries.CreateProject(ctx, CreateProjectParams{
		Name:                p.Name,
		Description:         desc,
		OwnerOrganizationID: p.OwnerOrganizationID,
		CreatedByUserID:     p.CreatedByUserID,
		IsPublic:            pgtype.Bool{Bool: p.IsPublic, Valid: true},
	})
	if err == nil {
		p.ID = created.ID
		p.CreatedAt = created.CreatedAt.Time
		p.UpdatedAt = created.UpdatedAt.Time
	}
	return err
}

func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	p, err := r.queries.GetProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var desc *string
	if p.Description.Valid {
		desc = &p.Description.String
	}
	isPublic := false
	if p.IsPublic.Valid {
		isPublic = p.IsPublic.Bool
	}
	return &domain.Project{
		ID:                  p.ID,
		Name:                p.Name,
		Description:         desc,
		OwnerOrganizationID: p.OwnerOrganizationID,
		CreatedByUserID:     p.CreatedByUserID,
		IsPublic:            isPublic,
		CreatedAt:           p.CreatedAt.Time,
		UpdatedAt:           p.UpdatedAt.Time,
	}, nil
}

func (r *ProjectRepository) ListForOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Project, error) {
	rows, err := r.queries.ListProjectsForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var res []*domain.Project
	for _, p := range rows {
		var desc *string
		if p.Description.Valid {
			desc = &p.Description.String
		}
		isPublic := false
		if p.IsPublic.Valid {
			isPublic = p.IsPublic.Bool
		}
		res = append(res, &domain.Project{
			ID:                  p.ID,
			Name:                p.Name,
			Description:         desc,
			OwnerOrganizationID: p.OwnerOrganizationID,
			CreatedByUserID:     p.CreatedByUserID,
			IsPublic:            isPublic,
			CreatedAt:           p.CreatedAt.Time,
			UpdatedAt:           p.UpdatedAt.Time,
		})
	}
	return res, nil
}

func (r *ProjectRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	rows, err := r.queries.ListProjectsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var res []*domain.Project
	for _, p := range rows {
		var desc *string
		if p.Description.Valid {
			desc = &p.Description.String
		}
		isPublic := false
		if p.IsPublic.Valid {
			isPublic = p.IsPublic.Bool
		}
		res = append(res, &domain.Project{
			ID:                  p.ID,
			Name:                p.Name,
			Description:         desc,
			OwnerOrganizationID: p.OwnerOrganizationID,
			CreatedByUserID:     p.CreatedByUserID,
			IsPublic:            isPublic,
			CreatedAt:           p.CreatedAt.Time,
			UpdatedAt:           p.UpdatedAt.Time,
		})
	}
	return res, nil
}

func (r *ProjectRepository) AddCollaborator(ctx context.Context, pc *domain.ProjectCollaborator) error {
	var invitedBy pgtype.UUID
	if pc.InvitedByUserID != nil {
		invitedBy = pgtype.UUID{Bytes: *pc.InvitedByUserID, Valid: true}
	}
	created, err := r.queries.AddProjectCollaborator(ctx, AddProjectCollaboratorParams{
		ProjectID:       pc.ProjectID,
		UserID:          pc.UserID,
		Role:            string(pc.Role),
		InvitedByUserID: invitedBy,
	})
	if err == nil {
		pc.ID = created.ID
		pc.InvitedAt = created.InvitedAt.Time
	}
	return err
}

func (r *ProjectRepository) GetCollaborator(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*domain.ProjectCollaborator, error) {
	pc, err := r.queries.GetProjectCollaborator(ctx, GetProjectCollaboratorParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	var invitedBy *uuid.UUID
	if pc.InvitedByUserID.Valid {
		id := pc.InvitedByUserID.Bytes
		parsed, _ := uuid.FromBytes(id[:])
		invitedBy = &parsed
	}
	var acceptedAt *time.Time
	if pc.AcceptedAt.Valid {
		acceptedAt = &pc.AcceptedAt.Time
	}
	return &domain.ProjectCollaborator{
		ID:              pc.ID,
		ProjectID:       pc.ProjectID,
		UserID:          pc.UserID,
		Role:            domain.ProjectRole(pc.Role),
		InvitedByUserID: invitedBy,
		InvitedAt:       pc.InvitedAt.Time,
		AcceptedAt:      acceptedAt,
	}, nil
}

func (r *ProjectRepository) ListCollaborators(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectCollaborator, error) {
	rows, err := r.queries.ListProjectCollaborators(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var res []*domain.ProjectCollaborator
	for _, pc := range rows {
		var acceptedAt *time.Time
		if pc.AcceptedAt.Valid {
			acceptedAt = &pc.AcceptedAt.Time
		}
		var username, email, avatar *string
		if pc.Username.Valid {
			username = &pc.Username.String
		}
		if pc.Email != "" {
			e := pc.Email
			email = &e
		}
		if pc.AvatarUrl.Valid {
			avatar = &pc.AvatarUrl.String
		}
		res = append(res, &domain.ProjectCollaborator{
			ID:         pc.ID,
			ProjectID:  pc.ProjectID,
			UserID:     pc.UserID,
			Role:       domain.ProjectRole(pc.Role),
			InvitedAt:  pc.InvitedAt.Time,
			AcceptedAt: acceptedAt,
			Username:   username,
			Email:      email,
			AvatarURL:  avatar,
		})
	}
	return res, nil
}

func (r *ProjectRepository) UpdateRole(ctx context.Context, projectID uuid.UUID, userID uuid.UUID, role domain.ProjectRole) error {
	return r.queries.UpdateProjectRole(ctx, UpdateProjectRoleParams{
		ProjectID: projectID,
		UserID:    userID,
		Role:      string(role),
	})
}

func (r *ProjectRepository) RemoveCollaborator(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) error {
	return r.queries.RemoveProjectCollaborator(ctx, RemoveProjectCollaboratorParams{
		ProjectID: projectID,
		UserID:    userID,
	})
}

func (r *ProjectRepository) ListSandboxes(ctx context.Context, projectID uuid.UUID) ([]*domain.Sandbox, error) {
	rows, err := r.queries.ListSandboxesByProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return nil, err
	}
	var res []*domain.Sandbox
	for _, s := range rows {
		res = append(res, toDomainSandbox(s))
	}
	return res, nil
}
