package usecase

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type GitUsecase struct {
	workspaceDir string
}

func NewGitUsecase(dir string) *GitUsecase {
	return &GitUsecase{workspaceDir: dir}
}

func (uc *GitUsecase) getSandboxDir(sandboxID uuid.UUID) string {
	return filepath.Join(uc.workspaceDir, sandboxID.String())
}

func (uc *GitUsecase) runGitCmd(ctx context.Context, sandboxID uuid.UUID, args ...string) (string, error) {
	dir := uc.getSandboxDir(sandboxID)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %v, stderr: %s", strings.Join(args, " "), err, errBuf.String())
	}
	return outBuf.String(), nil
}

type GitStatus struct {
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

func (uc *GitUsecase) GetStatus(ctx context.Context, sandboxID uuid.UUID) (*GitStatus, error) {
	// Ensure remote is fetched
	uc.runGitCmd(ctx, sandboxID, "fetch", "origin")

	branchOut, err := uc.runGitCmd(ctx, sandboxID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(branchOut)

	statusOut, err := uc.runGitCmd(ctx, sandboxID, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	dirty := strings.TrimSpace(statusOut) != ""

	// Get ahead/behind count
	var ahead, behind int
	revListOut, err := uc.runGitCmd(ctx, sandboxID, "rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...origin/%s", branch))
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(revListOut))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &ahead)
			fmt.Sscanf(parts[1], "%d", &behind)
		}
	}

	return &GitStatus{
		Branch: branch,
		Dirty:  dirty,
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

func (uc *GitUsecase) ListBranches(ctx context.Context, sandboxID uuid.UUID) ([]string, error) {
	uc.runGitCmd(ctx, sandboxID, "fetch", "origin")
	out, err := uc.runGitCmd(ctx, sandboxID, "branch", "-a", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, b := range strings.Split(strings.TrimSpace(out), "\n") {
		if b != "" && !strings.Contains(b, "->") {
			branches = append(branches, strings.TrimSpace(b))
		}
	}
	return branches, nil
}

func (uc *GitUsecase) Checkout(ctx context.Context, sandboxID uuid.UUID, branch string) error {
	// If it's a remote branch like origin/feat, checkout a local tracking branch
	if strings.HasPrefix(branch, "origin/") {
		localBranch := strings.TrimPrefix(branch, "origin/")
		_, err := uc.runGitCmd(ctx, sandboxID, "checkout", "-b", localBranch, branch)
		if err != nil {
			// fallback if local already exists
			_, err = uc.runGitCmd(ctx, sandboxID, "checkout", localBranch)
			return err
		}
		return nil
	}
	_, err := uc.runGitCmd(ctx, sandboxID, "checkout", branch)
	return err
}

func (uc *GitUsecase) Pull(ctx context.Context, sandboxID uuid.UUID) error {
	_, err := uc.runGitCmd(ctx, sandboxID, "pull", "--rebase")
	return err
}

func (uc *GitUsecase) CreateBranch(ctx context.Context, sandboxID uuid.UUID, branch string) error {
	_, err := uc.runGitCmd(ctx, sandboxID, "checkout", "-b", branch)
	if err != nil {
		return err
	}
	// push to origin
	_, err = uc.runGitCmd(ctx, sandboxID, "push", "-u", "origin", branch)
	return err
}

func (uc *GitUsecase) Commit(ctx context.Context, sandboxID uuid.UUID, message string) error {
	// Add all changes
	_, err := uc.runGitCmd(ctx, sandboxID, "add", ".")
	if err != nil {
		return err
	}
	// Commit
	_, err = uc.runGitCmd(ctx, sandboxID, "commit", "-m", message)
	return err
}

func (uc *GitUsecase) Push(ctx context.Context, sandboxID uuid.UUID) error {
	_, err := uc.runGitCmd(ctx, sandboxID, "push")
	return err
}

type CommitEntry struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	Date      string `json:"date"`
}

func (uc *GitUsecase) Log(ctx context.Context, sandboxID uuid.UUID) ([]CommitEntry, error) {
	out, err := uc.runGitCmd(ctx, sandboxID, "log", "-n", "50", "--pretty=format:%H|%h|%s|%an|%aI")
	if err != nil {
		return nil, err
	}

	var commits []CommitEntry
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) == 5 {
			commits = append(commits, CommitEntry{
				Hash:      parts[0],
				ShortHash: parts[1],
				Message:   parts[2],
				Author:    parts[3],
				Date:      parts[4],
			})
		}
	}
	return commits, nil
}
