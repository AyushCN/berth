package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectUnauthorized = errors.New("unauthorized project action")
)

type ProjectUsecase struct {
	projRepo         domain.ProjectRepository
	orgRepo          domain.OrganizationRepository
	workspaceRepo    domain.WorkspaceRepository
	workspaceMemberRepo domain.WorkspaceMemberRepository
}

func NewProjectUsecase(projRepo domain.ProjectRepository, orgRepo domain.OrganizationRepository, workspaceRepo domain.WorkspaceRepository, workspaceMemberRepo domain.WorkspaceMemberRepository) *ProjectUsecase {
	return &ProjectUsecase{
		projRepo:          projRepo,
		orgRepo:           orgRepo,
		workspaceRepo:     workspaceRepo,
		workspaceMemberRepo: workspaceMemberRepo,
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

	// Create canonical workspace for the project
	canonicalWorkspace := &domain.Workspace{
		ID:               uuid.New(),
		ProjectID:        p.ID,
		Name:             "canonical",
		Type:             domain.WorkspaceTypeCanonical,
		OwnerID:          userID,
		GitBranch:        "main",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := u.workspaceRepo.Create(ctx, canonicalWorkspace); err != nil {
		// Log error but don't fail project creation
		// The workspace can be created later
		_ = err
	} else {
		// Add creator as workspace member
		member := &domain.WorkspaceMember{
			ID:          uuid.New(),
			WorkspaceID: canonicalWorkspace.ID,
			UserID:      userID,
			Role:        domain.WorkspaceMemberRoleOwner,
			CreatedAt:   time.Now(),
		}
		_ = u.workspaceMemberRepo.Create(ctx, member)
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

// ListEnvironments returns the environments in a project. The HTTP route is
// still /projects/:id/sandboxes for frontend compatibility, but the data now
// comes from the environments table via the project's workspaces.
func (u *ProjectUsecase) ListEnvironments(ctx context.Context, currentUserID uuid.UUID, projectID uuid.UUID) ([]*domain.Environment, error) {
	if _, err := u.GetByID(ctx, currentUserID, projectID); err != nil {
		return nil, err
	}
	return u.projRepo.ListEnvironments(ctx, projectID)
}