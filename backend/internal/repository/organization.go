package repository

import (
	"context"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

var _ domain.OrganizationRepository = (*OrganizationRepository)(nil)

type OrganizationRepository struct {
	queries *Queries
}

func NewOrganizationRepository(q *Queries) *OrganizationRepository {
	return &OrganizationRepository{queries: q}
}

func (r *OrganizationRepository) Create(ctx context.Context, name string) (*domain.Organization, error) {
	o, err := r.queries.CreateOrganization(ctx, name)
	if err != nil {
		return nil, err
	}
	return &domain.Organization{
		ID:        o.ID,
		Name:      o.Name,
		CreatedAt: o.CreatedAt.Time,
	}, nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	o, err := r.queries.GetOrganizationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.Organization{
		ID:        o.ID,
		Name:      o.Name,
		CreatedAt: o.CreatedAt.Time,
	}, nil
}

func (r *OrganizationRepository) AddMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role domain.OrganizationRole) (*domain.OrganizationMember, error) {
	om, err := r.queries.AddOrganizationMember(ctx, AddOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           string(role),
	})
	if err != nil {
		return nil, err
	}
	return &domain.OrganizationMember{
		ID:             om.ID,
		OrganizationID: om.OrganizationID,
		UserID:         om.UserID,
		Role:           domain.OrganizationRole(om.Role),
		CreatedAt:      om.CreatedAt.Time,
	}, nil
}

func (r *OrganizationRepository) GetMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (*domain.OrganizationMember, error) {
	om, err := r.queries.GetOrganizationMember(ctx, GetOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
	if err != nil {
		return nil, err
	}
	return &domain.OrganizationMember{
		ID:             om.ID,
		OrganizationID: om.OrganizationID,
		UserID:         om.UserID,
		Role:           domain.OrganizationRole(om.Role),
		CreatedAt:      om.CreatedAt.Time,
	}, nil
}

func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*domain.OrganizationMember, error) {
	rows, err := r.queries.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var res []*domain.OrganizationMember
	for _, row := range rows {
		var username, email, avatar *string
		if row.Username.Valid {
			username = &row.Username.String
		}
		if row.Email != "" {
			e := row.Email
			email = &e
		}
		if row.AvatarUrl.Valid {
			avatar = &row.AvatarUrl.String
		}
		res = append(res, &domain.OrganizationMember{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			UserID:         row.UserID,
			Role:           domain.OrganizationRole(row.Role),
			CreatedAt:      row.CreatedAt.Time,
			Username:       username,
			Email:          email,
			AvatarURL:      avatar,
		})
	}
	return res, nil
}

func (r *OrganizationRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error) {
	rows, err := r.queries.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var res []*domain.Organization
	for _, row := range rows {
		res = append(res, &domain.Organization{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return res, nil
}

func (r *OrganizationRepository) UpdateRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role domain.OrganizationRole) error {
	return r.queries.UpdateOrganizationRole(ctx, UpdateOrganizationRoleParams{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           string(role),
	})
}

func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	return r.queries.RemoveOrganizationMember(ctx, RemoveOrganizationMemberParams{
		OrganizationID: orgID,
		UserID:         userID,
	})
}
