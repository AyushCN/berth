package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ChangeRequestState represents the state of a change request
type ChangeRequestState string

const (
	ChangeRequestStateOpen     ChangeRequestState = "OPEN"
	ChangeRequestStateReview   ChangeRequestState = "REVIEW"
	ChangeRequestStateMerged   ChangeRequestState = "MERGED"
	ChangeRequestStateClosed   ChangeRequestState = "CLOSED"
	ChangeRequestStateConflict ChangeRequestState = "CONFLICT"
)

// ChangeRequest represents a request to merge changes from a fork to canonical
type ChangeRequest struct {
	ID                  uuid.UUID           `json:"id"`
	ProjectID           uuid.UUID           `json:"project_id"`
	SourceWorkspaceID   uuid.UUID           `json:"source_workspace_id"`
	TargetWorkspaceID   uuid.UUID           `json:"target_workspace_id"`
	Title               string              `json:"title"`
	Description         string              `json:"description,omitempty"`
	AuthorID            uuid.UUID           `json:"author_id"`
	State               ChangeRequestState  `json:"state"`
	Commits             []CommitInfo        `json:"commits,omitempty"`
	FilesChanged        []string            `json:"files_changed,omitempty"`
	ReviewerID          *uuid.UUID          `json:"reviewer_id,omitempty"`
	ReviewedAt          *time.Time          `json:"reviewed_at,omitempty"`
	MergedAt            *time.Time          `json:"merged_at,omitempty"`
	MergeCommitHash     string              `json:"merge_commit_hash,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// CommitInfo represents a commit in a change request
type CommitInfo struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// ChangeRequestRepository defines the interface for change request persistence
type ChangeRequestRepository interface {
	Create(ctx context.Context, cr *ChangeRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*ChangeRequest, error)
	GetByProject(ctx context.Context, projectID uuid.UUID) ([]*ChangeRequest, error)
	GetBySourceWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*ChangeRequest, error)
	GetByState(ctx context.Context, state ChangeRequestState) ([]*ChangeRequest, error)
	Update(ctx context.Context, cr *ChangeRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}