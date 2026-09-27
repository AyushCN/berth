package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type CreateChangeRequestRequest struct {
	ProjectID          uuid.UUID
	SourceWorkspaceID  uuid.UUID
	TargetWorkspaceID  uuid.UUID
	Title              string
	Description        string
	AuthorID           uuid.UUID
}

type UpdateChangeRequestRequest struct {
	ID          uuid.UUID
	Title       *string
	Description *string
	State       *domain.ChangeRequestState
	ReviewerID  *uuid.UUID
}

type ChangeRequestUsecase struct {
	changeRequestRepo domain.ChangeRequestRepository
	workspaceRepo     domain.WorkspaceRepository
	workspaceMemberRepo domain.WorkspaceMemberRepository
	gitRepo           domain.GitRepository
}

func NewChangeRequestUsecase(
	changeRequestRepo domain.ChangeRequestRepository,
	workspaceRepo domain.WorkspaceRepository,
	workspaceMemberRepo domain.WorkspaceMemberRepository,
	gitRepo domain.GitRepository,
) *ChangeRequestUsecase {
	return &ChangeRequestUsecase{
		changeRequestRepo:     changeRequestRepo,
		workspaceRepo:         workspaceRepo,
		workspaceMemberRepo:   workspaceMemberRepo,
		gitRepo:               gitRepo,
	}
}

func (uc *ChangeRequestUsecase) CreateChangeRequest(ctx context.Context, req CreateChangeRequestRequest) (*domain.ChangeRequest, error) {
	// Verify source workspace exists and belongs to project
	sourceWorkspace, err := uc.workspaceRepo.GetByID(ctx, req.SourceWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("source workspace not found: %w", err)
	}

	// Verify target workspace exists and is canonical
	targetWorkspace, err := uc.workspaceRepo.GetByID(ctx, req.TargetWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("target workspace not found: %w", err)
	}

	if targetWorkspace.Type != domain.WorkspaceTypeCanonical {
		return nil, fmt.Errorf("target workspace must be canonical")
	}

	if sourceWorkspace.ProjectID != targetWorkspace.ProjectID {
		return nil, fmt.Errorf("source and target workspaces must belong to same project")
	}

	// Verify author has access to source workspace
	member, err := uc.workspaceMemberRepo.Get(ctx, req.SourceWorkspaceID, req.AuthorID)
	if err != nil || (member.Role != domain.WorkspaceMemberRoleOwner && member.Role != domain.WorkspaceMemberRoleEditor) {
		return nil, fmt.Errorf("unauthorized to create change request from this workspace")
	}

	// Get commits and files changed from source workspace via git
	commits, err := uc.gitRepo.GetCommits(ctx, req.SourceWorkspaceID)
	if err != nil {
		commits = []domain.CommitInfo{}
	}

	filesChanged, err := uc.gitRepo.GetChangedFiles(ctx, req.SourceWorkspaceID, req.TargetWorkspaceID)
	if err != nil {
		filesChanged = []string{}
	}

	cr := &domain.ChangeRequest{
		ID:                  uuid.New(),
		ProjectID:           sourceWorkspace.ProjectID,
		SourceWorkspaceID:   req.SourceWorkspaceID,
		TargetWorkspaceID:   req.TargetWorkspaceID,
		Title:               req.Title,
		Description:         req.Description,
		AuthorID:            req.AuthorID,
		State:               domain.ChangeRequestStateOpen,
		Commits:             commits,
		FilesChanged:        filesChanged,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := uc.changeRequestRepo.Create(ctx, cr); err != nil {
		return nil, fmt.Errorf("failed to create change request: %w", err)
	}

	return cr, nil
}

func (uc *ChangeRequestUsecase) GetChangeRequest(ctx context.Context, id uuid.UUID) (*domain.ChangeRequest, error) {
	return uc.changeRequestRepo.GetByID(ctx, id)
}

func (uc *ChangeRequestUsecase) ListChangeRequests(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*domain.ChangeRequest, error) {
	return uc.changeRequestRepo.GetByProject(ctx, projectID)
}

func (uc *ChangeRequestUsecase) ListChangeRequestsBySource(ctx context.Context, workspaceID uuid.UUID) ([]*domain.ChangeRequest, error) {
	return uc.changeRequestRepo.GetBySourceWorkspace(ctx, workspaceID)
}

func (uc *ChangeRequestUsecase) UpdateChangeRequest(ctx context.Context, req UpdateChangeRequestRequest) (*domain.ChangeRequest, error) {
	cr, err := uc.changeRequestRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("change request not found: %w", err)
	}

	if req.Title != nil {
		cr.Title = *req.Title
	}
	if req.Description != nil {
		cr.Description = *req.Description
	}
	if req.State != nil {
		cr.State = *req.State
	}
	if req.ReviewerID != nil {
		cr.ReviewerID = req.ReviewerID
	}

	cr.UpdatedAt = time.Now()

	if err := uc.changeRequestRepo.Update(ctx, cr); err != nil {
		return nil, fmt.Errorf("failed to update change request: %w", err)
	}

	return cr, nil
}

func (uc *ChangeRequestUsecase) MergeChangeRequest(ctx context.Context, changeRequestID, reviewerID uuid.UUID) (*domain.ChangeRequest, error) {
	cr, err := uc.changeRequestRepo.GetByID(ctx, changeRequestID)
	if err != nil {
		return nil, fmt.Errorf("change request not found: %w", err)
	}

	if cr.State != domain.ChangeRequestStateOpen && cr.State != domain.ChangeRequestStateReview {
		return nil, fmt.Errorf("change request cannot be merged in state: %s", cr.State)
	}

	// Verify reviewer has permission to merge (must be owner of target workspace)
	targetWorkspace, err := uc.workspaceRepo.GetByID(ctx, cr.TargetWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("target workspace not found: %w", err)
	}

	member, err := uc.workspaceMemberRepo.Get(ctx, targetWorkspace.ID, reviewerID)
	if err != nil || member.Role != domain.WorkspaceMemberRoleOwner {
		return nil, fmt.Errorf("unauthorized to merge change request")
	}

	// Merge the changes via git
	mergeCommit, err := uc.gitRepo.Merge(ctx, cr.SourceWorkspaceID, cr.TargetWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to merge changes: %w", err)
	}

	cr.State = domain.ChangeRequestStateMerged
	cr.ReviewerID = &reviewerID
	cr.ReviewedAt = timePtr(time.Now())
	cr.MergedAt = timePtr(time.Now())
	cr.MergeCommitHash = mergeCommit
	cr.UpdatedAt = time.Now()

	if err := uc.changeRequestRepo.Update(ctx, cr); err != nil {
		return nil, fmt.Errorf("failed to update change request: %w", err)
	}

	return cr, nil
}

func (uc *ChangeRequestUsecase) CloseChangeRequest(ctx context.Context, changeRequestID, reviewerID uuid.UUID) (*domain.ChangeRequest, error) {
	cr, err := uc.changeRequestRepo.GetByID(ctx, changeRequestID)
	if err != nil {
		return nil, fmt.Errorf("change request not found: %w", err)
	}

	if cr.State == domain.ChangeRequestStateMerged {
		return nil, fmt.Errorf("cannot close merged change request")
	}

	// Verify reviewer has permission
	targetWorkspace, err := uc.workspaceRepo.GetByID(ctx, cr.TargetWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("target workspace not found: %w", err)
	}

	member, err := uc.workspaceMemberRepo.Get(ctx, targetWorkspace.ID, reviewerID)
	if err != nil || member.Role != domain.WorkspaceMemberRoleOwner {
		return nil, fmt.Errorf("unauthorized to close change request")
	}

	cr.State = domain.ChangeRequestStateClosed
	cr.ReviewerID = &reviewerID
	cr.ReviewedAt = timePtr(time.Now())
	cr.UpdatedAt = time.Now()

	if err := uc.changeRequestRepo.Update(ctx, cr); err != nil {
		return nil, fmt.Errorf("failed to update change request: %w", err)
	}

	return cr, nil
}