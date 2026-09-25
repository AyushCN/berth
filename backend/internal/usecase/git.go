package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/pkg/crypto"
	"github.com/google/uuid"
)

type GitUsecase struct {
	workspaceDir string
	userRepo     domain.UserRepository
	sandboxRepo  domain.SandboxRepository
}

func NewGitUsecase(dir string, userRepo domain.UserRepository, sandboxRepo domain.SandboxRepository) *GitUsecase {
	return &GitUsecase{workspaceDir: dir, userRepo: userRepo, sandboxRepo: sandboxRepo}
}

func (uc *GitUsecase) getSandboxDir(sandboxID uuid.UUID) string {
	return filepath.Join(uc.workspaceDir, sandboxID.String())
}

func (uc *GitUsecase) Authorize(ctx context.Context, sandboxID, userID uuid.UUID) error {
	sandbox, err := uc.sandboxRepo.GetByID(ctx, sandboxID)
	if err != nil || sandbox.OwnerID != userID {
		return fmt.Errorf("sandbox not found")
	}
	return nil
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

func (uc *GitUsecase) Checkout(ctx context.Context, sandboxID uuid.UUID, branch string, force ...bool) error {
	forceCheckout := len(force) > 0 && force[0]

	if forceCheckout {
		// Stash any dirty changes first to allow forced checkout
		uc.runGitCmd(ctx, sandboxID, "stash", "--include-untracked")
	}
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

// Push creates or reuses a per-sandbox branch and authenticates with the owner's OAuth token.
func (uc *GitUsecase) Push(ctx context.Context, sandboxID, userID uuid.UUID) (string, error) {
	if err := uc.Authorize(ctx, sandboxID, userID); err != nil {
		return "", err
	}
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user.GithubTokenEncrypted == "" {
		return "", fmt.Errorf("GitHub authorization is missing; sign in with GitHub again")
	}
	token, err := crypto.Decrypt(user.GithubTokenEncrypted)
	if err != nil {
		return "", fmt.Errorf("could not decrypt GitHub authorization; sign in with GitHub again")
	}
	if token == "" {
		return "", fmt.Errorf("GitHub authorization is missing; sign in with GitHub again")
	}
	// Get current branch
	branchOut, err := uc.runGitCmd(ctx, sandboxID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	currentBranch := strings.TrimSpace(branchOut)

	// Push edits on a predictable Berth branch, leaving the source branch intact.
	sandboxBranch := fmt.Sprintf("berth/%s", sandboxID.String())
	if currentBranch != sandboxBranch {
		if _, err := uc.runGitCmd(ctx, sandboxID, "checkout", "-b", sandboxBranch); err != nil {
			if _, checkoutErr := uc.runGitCmd(ctx, sandboxID, "checkout", sandboxBranch); checkoutErr != nil {
				return "", fmt.Errorf("could not create or switch to %s: %w", sandboxBranch, checkoutErr)
			}
		}
		currentBranch = sandboxBranch
	}

	if _, err := uc.runGitCmdWithToken(ctx, sandboxID, token, "push", "-u", "origin", currentBranch); err != nil {
		return "", fmt.Errorf("GitHub push failed; verify the OAuth grant has repository contents write access: %w", err)
	}
	return currentBranch, nil
}

func (uc *GitUsecase) runGitCmdWithToken(ctx context.Context, sandboxID uuid.UUID, token string, args ...string) (string, error) {
	dir := uc.getSandboxDir(sandboxID)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	encoded := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraheader", "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic "+encoded)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git push failed: %v, stderr: %s", err, errBuf.String())
	}
	return outBuf.String(), nil
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
