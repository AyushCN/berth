package usecase

import (
	"context"
	"errors"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectUnauthorized = errors.New("unauthorized project action")
)

type ProjectUsecase struct {
	projRepo domain.ProjectRepository
	orgRepo  domain.OrganizationRepository
}

func NewProjectUsecase(projRepo domain.ProjectRepository, orgRepo domain.OrganizationRepository) *ProjectUsecase {
	return &ProjectUsecase{
		projRepo: projRepo,
		orgRepo:  orgRepo,
	}
}

func (u *ProjectUsecase) Create(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, name string, description *string, isPublic bool) (*domain.Project, error) {
	// Must be an admin or owner of the org to create a project
	member, err := u.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil || (member.Role != domain.OrgRoleOwner && member.Role != domain.OrgRoleAdmin) {
		return nil, ErrProjectUnauthorized
	}

	p := &domain.Project{
		Name:                name,
		Description:         description,
		OwnerOrganizationID: orgID,
		CreatedByUserID:     userID,
		IsPublic:            isPublic,
	}

	err = u.projRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Add creator as OWNER
	err = u.projRepo.AddCollaborator(ctx, &domain.ProjectCollaborator{
		ProjectID: p.ID,
		UserID:    userID,
		Role:      domain.ProjectRoleOwner,
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (u *ProjectUsecase) GetByID(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.Project, error) {
	p, err := u.projRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}
	if p.IsPublic {
		return p, nil
	}
	// Verify access
	_, err = u.projRepo.GetCollaborator(ctx, projectID, userID)
	if err != nil {
		// Maybe user has org level access?
		member, orgErr := u.orgRepo.GetMember(ctx, p.OwnerOrganizationID, userID)
		if orgErr != nil || (member.Role != domain.OrgRoleOwner && member.Role != domain.OrgRoleAdmin) {
			return nil, ErrProjectUnauthorized
		}
	}
	return p, nil
}

func (u *ProjectUsecase) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	return u.projRepo.ListForUser(ctx, userID)
}

func (u *ProjectUsecase) ListForOrg(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) ([]*domain.Project, error) {
	// Verify org membership
	_, err := u.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return nil, ErrProjectUnauthorized
	}
	return u.projRepo.ListForOrg(ctx, orgID)
}

func (u *ProjectUsecase) AddCollaborator(ctx context.Context, currentUserID uuid.UUID, projectID uuid.UUID, newUserID uuid.UUID, role domain.ProjectRole) (*domain.ProjectCollaborator, error) {
	collab, err := u.projRepo.GetCollaborator(ctx, projectID, currentUserID)
	if err != nil || collab.Role != domain.ProjectRoleOwner {
		return nil, ErrProjectUnauthorized
	}
	pc := &domain.ProjectCollaborator{
		ProjectID:       projectID,
		UserID:          newUserID,
		Role:            role,
		InvitedByUserID: &currentUserID,
	}
	err = u.projRepo.AddCollaborator(ctx, pc)
	if err != nil {
		return nil, err
	}
	return pc, nil
}

func (u *ProjectUsecase) ListCollaborators(ctx context.Context, currentUserID uuid.UUID, projectID uuid.UUID) ([]*domain.ProjectCollaborator, error) {
	_, err := u.projRepo.GetCollaborator(ctx, projectID, currentUserID)
	if err != nil {
		return nil, ErrProjectUnauthorized
	}
	return u.projRepo.ListCollaborators(ctx, projectID)
}

func (u *ProjectUsecase) RemoveCollaborator(ctx context.Context, currentUserID uuid.UUID, projectID uuid.UUID, targetUserID uuid.UUID) error {
	collab, err := u.projRepo.GetCollaborator(ctx, projectID, currentUserID)
	if err != nil {
		return ErrProjectUnauthorized
	}
	if currentUserID != targetUserID && collab.Role != domain.ProjectRoleOwner {
		return ErrProjectUnauthorized
	}
	return u.projRepo.RemoveCollaborator(ctx, projectID, targetUserID)
}

func (u *ProjectUsecase) ListSandboxes(ctx context.Context, currentUserID uuid.UUID, projectID uuid.UUID) ([]*domain.Sandbox, error) {
	_, err := u.GetByID(ctx, currentUserID, projectID)
	if err != nil {
		return nil, err
	}
	return u.projRepo.ListSandboxes(ctx, projectID)
}
