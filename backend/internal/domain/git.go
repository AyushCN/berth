package domain

import (
	"context"

	"github.com/google/uuid"
)

// GitStatus represents the git status of a sandbox
type GitStatus struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

// CommitEntry represents a commit in the log
type CommitEntry struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Date      string `json:"date"`
}

// GitRepository defines the interface for git operations within a sandbox
type GitRepository interface {
	// GetCommits returns the commit history for a sandbox
	GetCommits(ctx context.Context, sandboxID uuid.UUID) ([]CommitEntry, error)

	// GetChangedFiles returns the list of files changed between two sandboxes
	GetChangedFiles(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) ([]string, error)

	// GetDiff returns the diff between two sandboxes
	GetDiff(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error)

	// Commit commits changes in a sandbox
	Commit(ctx context.Context, sandboxID uuid.UUID, message string) error

	// Push pushes commits to remote
	Push(ctx context.Context, sandboxID uuid.UUID, userID uuid.UUID) (string, error)

	// Pull pulls changes from remote
	Pull(ctx context.Context, sandboxID uuid.UUID) error

	// Merge merges changes from source sandbox into target sandbox
	// Returns the merge commit hash
	Merge(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error)

	// CreateBranch creates a new branch in the sandbox
	CreateBranch(ctx context.Context, sandboxID uuid.UUID, branchName string) error

	// Checkout checks out a branch in the sandbox
	Checkout(ctx context.Context, sandboxID uuid.UUID, branchName string, force bool) error

	// GetStatus returns the git status for a sandbox
	GetStatus(ctx context.Context, sandboxID uuid.UUID) (*GitStatus, error)

	// ListBranches returns all branches for a sandbox
	ListBranches(ctx context.Context, sandboxID uuid.UUID) ([]string, error)

	// Log returns the commit log for a sandbox
	Log(ctx context.Context, sandboxID uuid.UUID) ([]CommitEntry, error)
}
