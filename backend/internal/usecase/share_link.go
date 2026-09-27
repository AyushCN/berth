package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type CreateShareLinkRequest struct {
	ProjectID   uuid.UUID
	Role        string // VIEWER, EDITOR
	CreatedBy   uuid.UUID
	ExpiresAt   *time.Time
	MaxUses     *int
}

type JoinViaShareLinkRequest struct {
	Code   string
	UserID uuid.UUID
}

type ShareLinkUsecase struct {
	shareLinkRepo       domain.ShareLinkRepository
	projectRepo         domain.ProjectRepository
	workspaceRepo       domain.WorkspaceRepository
	workspaceMemberRepo domain.WorkspaceMemberRepository
	gitRepo             domain.GitRepository
}

func NewShareLinkUsecase(
	shareLinkRepo domain.ShareLinkRepository,
	projectRepo domain.ProjectRepository,
	workspaceRepo domain.WorkspaceRepository,
	workspaceMemberRepo domain.WorkspaceMemberRepository,
	gitRepo domain.GitRepository,
) *ShareLinkUsecase {
	return &ShareLinkUsecase{
		shareLinkRepo:       shareLinkRepo,
		projectRepo:         projectRepo,
		workspaceRepo:       workspaceRepo,
		workspaceMemberRepo: workspaceMemberRepo,
		gitRepo:             gitRepo,
	}
}

func (uc *ShareLinkUsecase) CreateShareLink(ctx context.Context, req CreateShareLinkRequest) (*domain.ShareLink, error) {
	// Verify project exists and user has permission
	_, err := uc.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Verify user is project owner or collaborator
	member, err := uc.projectRepo.GetCollaborator(ctx, req.ProjectID, req.CreatedBy)
	if err != nil || (member.Role != domain.ProjectRoleOwner && member.Role != domain.ProjectRoleCollaborator) {
		return nil, fmt.Errorf("unauthorized to create share link")
	}

	// Generate unique code
	code, err := uc.generateUniqueCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	link := &domain.ShareLink{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		Code:      code,
		Role:      req.Role,
		CreatedBy: req.CreatedBy,
		ExpiresAt: req.ExpiresAt,
		MaxUses:   req.MaxUses,
		UsesCount: 0,
		CreatedAt: time.Now(),
	}

	if err := uc.shareLinkRepo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to create share link: %w", err)
	}

	return link, nil
}

func (uc *ShareLinkUsecase) GetShareLinks(ctx context.Context, projectID, userID uuid.UUID) ([]*domain.ShareLink, error) {
	// Verify user has access to project
	_, err := uc.projectRepo.GetCollaborator(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}

	return uc.shareLinkRepo.GetByProject(ctx, projectID)
}

func (uc *ShareLinkUsecase) RevokeShareLink(ctx context.Context, linkID, userID uuid.UUID) error {
	link, err := uc.shareLinkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}
	// Verify user owns the project
	_, err = uc.projectRepo.GetCollaborator(ctx, link.ProjectID, userID)
	if err != nil {
		return fmt.Errorf("unauthorized")
	}

	now := time.Now()
	link.RevokedAt = &now
	return uc.shareLinkRepo.Update(ctx, link)
}

func (uc *ShareLinkUsecase) JoinViaShareLink(ctx context.Context, req JoinViaShareLinkRequest) (*domain.Workspace, error) {
	link, err := uc.shareLinkRepo.GetByCode(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired share link")
	}

	if link.RevokedAt != nil {
		return nil, fmt.Errorf("share link has been revoked")
	}

	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("share link has expired")
	}

	if link.MaxUses != nil && link.UsesCount >= *link.MaxUses {
		return nil, fmt.Errorf("share link has reached maximum uses")
	}

	// Get the canonical workspace for the project
	canonicalWorkspace, err := uc.workspaceRepo.GetCanonical(ctx, link.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project has no canonical workspace: %w", err)
	}

	// Check if user already has a workspace in this project
	userWorkspaces, err := uc.workspaceRepo.GetByProject(ctx, link.ProjectID)
	if err == nil {
		for _, ws := range userWorkspaces {
			member, _ := uc.workspaceMemberRepo.Get(ctx, ws.ID, req.UserID)
			if member != nil {
				// User already has a workspace, return it
				return ws, nil
			}
		}
	}

	// Create workspace based on role
	var workspace *domain.Workspace
	if link.Role == "VIEWER" {
		// Viewers get access to canonical workspace
		workspace = canonicalWorkspace
	} else {
		// Editors get a fork workspace
		workspace, err = uc.createForkWorkspace(ctx, canonicalWorkspace, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to create fork workspace: %w", err)
		}
	}

	// Add user as workspace member
	memberRole := domain.WorkspaceMemberRoleEditor
	if link.Role == "VIEWER" {
		memberRole = domain.WorkspaceMemberRoleViewer
	}

	member := &domain.WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: workspace.ID,
		UserID:      req.UserID,
		Role:        memberRole,
		CreatedAt:   time.Now(),
	}

	if err := uc.workspaceMemberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add workspace member: %w", err)
	}

	// Add user as project collaborator if not already
	_, err = uc.projectRepo.GetCollaborator(ctx, link.ProjectID, req.UserID)
	if err != nil {
		projectCollaborator := &domain.ProjectCollaborator{
			ID:              uuid.New(),
			ProjectID:       link.ProjectID,
			UserID:          req.UserID,
			Role:            domain.ProjectRole(link.Role),
			InvitedByUserID: &link.CreatedBy,
			InvitedAt:       time.Now(),
			AcceptedAt:      timePtr(time.Now()),
		}
		if err := uc.projectRepo.AddCollaborator(ctx, projectCollaborator); err != nil {
			slog.Warn("failed to add project collaborator", "project_id", link.ProjectID, "user_id", req.UserID, "error", err)
		}
	}

	// Increment share link uses
	uc.shareLinkRepo.IncrementUses(ctx, link.ID)

	return workspace, nil
}

func (uc *ShareLinkUsecase) createForkWorkspace(ctx context.Context, canonical *domain.Workspace, userID uuid.UUID) (*domain.Workspace, error) {
	workspace := &domain.Workspace{
		ID:               uuid.New(),
		ProjectID:        canonical.ProjectID,
		Name:             fmt.Sprintf("%s-fork", userID.String()[:8]),
		Type:             domain.WorkspaceTypeFork,
		BaseWorkspaceID:  &canonical.ID,
		OwnerID:          userID,
		GitBranch:        canonical.GitBranch,
		CommitHash:       canonical.CommitHash,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := uc.workspaceRepo.Create(ctx, workspace); err != nil {
		return nil, err
	}

	// Clone canonical workspace files to fork workspace
	if uc.gitRepo != nil && canonical.GitBranch != "" {
		// Fetch the canonical branch to ensure it's available
		if err := uc.gitRepo.Pull(ctx, canonical.ID); err != nil {
			slog.Warn("failed to pull canonical workspace before forking", "workspace_id", canonical.ID, "error", err)
		}

		// Create a new branch for the fork
		forkBranch := fmt.Sprintf("fork/%s", workspace.ID.String()[:8])
		if err := uc.gitRepo.CreateBranch(ctx, canonical.ID, forkBranch); err != nil {
			slog.Warn("failed to create fork branch", "workspace_id", canonical.ID, "error", err)
		}
		
		// The fork workspace will use this branch when its environment is created
		workspace.GitBranch = forkBranch
		if err := uc.workspaceRepo.Update(ctx, workspace); err != nil {
			slog.Warn("failed to update fork workspace with branch", "workspace_id", workspace.ID, "error", err)
		}
	}

	return workspace, nil
}

func (uc *ShareLinkUsecase) generateUniqueCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		code := hex.EncodeToString(bytes)

		// Check if code exists
		_, err := uc.shareLinkRepo.GetByCode(ctx, code)
		if err != nil {
			// Code doesn't exist, use it
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code after 10 attempts")
}

func timePtr(t time.Time) *time.Time {
	return &t
}