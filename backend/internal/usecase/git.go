package usecase

import (
	"bytes"
	"context"
	"fmt"
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
	runtime      domain.ContainerRuntime
}

func NewGitUsecase(dir string, userRepo domain.UserRepository, sandboxRepo domain.SandboxRepository, runtime domain.ContainerRuntime) *GitUsecase {
	return &GitUsecase{workspaceDir: dir, userRepo: userRepo, sandboxRepo: sandboxRepo, runtime: runtime}
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

// runGitCmdOnHost runs git command on host filesystem (for workspace operations)
func (uc *GitUsecase) runGitCmdOnHost(ctx context.Context, sandboxID uuid.UUID, args ...string) (string, error) {
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

// runGitCmdInContainer runs git command inside the sandbox container
func (uc *GitUsecase) runGitCmdInContainer(ctx context.Context, sandboxID uuid.UUID, args ...string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("container runtime not available")
	}

	sandbox, err := uc.sandboxRepo.GetByID(ctx, sandboxID)
	if err != nil {
		return "", fmt.Errorf("sandbox not found: %w", err)
	}

	if sandbox.ContainerID == nil || *sandbox.ContainerID == "" {
		return "", fmt.Errorf("sandbox has no container")
	}

	if sandbox.State != domain.StateRunning {
		return "", fmt.Errorf("sandbox is not running")
	}

	cmd := append([]string{"git"}, args...)
	return uc.runtime.Exec(ctx, *sandbox.ContainerID, cmd)
}

func (uc *GitUsecase) runGitCmdInContainerWithToken(ctx context.Context, sandboxID uuid.UUID, token string, args ...string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("container runtime not available")
	}

	sandbox, err := uc.sandboxRepo.GetByID(ctx, sandboxID)
	if err != nil {
		return "", fmt.Errorf("sandbox not found: %w", err)
	}

	if sandbox.ContainerID == nil || *sandbox.ContainerID == "" {
		return "", fmt.Errorf("sandbox has no container")
	}

	if sandbox.State != domain.StateRunning {
		return "", fmt.Errorf("sandbox is not running")
	}

	// Pass GITHUB_TOKEN as environment variable
	env := map[string]string{
		"GITHUB_TOKEN": token,
	}

	return uc.runtime.ExecWithEnv(ctx, *sandbox.ContainerID, append([]string{"git"}, args...), env)
}

// --- Host-based operations (workspace-level) ---

// GetChangedFilesHost returns the list of files changed between two workspaces (host)
func (uc *GitUsecase) GetChangedFilesHost(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) ([]string, error) {
	sourceDir := uc.getSandboxDir(sourceSandboxID)
	targetDir := uc.getSandboxDir(targetSandboxID)

	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", fmt.Sprintf("%s...%s", sourceDir, targetDir))
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// GetDiffHost returns the diff between two workspaces (host)
func (uc *GitUsecase) GetDiffHost(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error) {
	sourceDir := uc.getSandboxDir(sourceSandboxID)
	targetDir := uc.getSandboxDir(targetSandboxID)

	cmd := exec.CommandContext(ctx, "git", "diff", fmt.Sprintf("%s...%s", sourceDir, targetDir))
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(output), nil
}

// MergeHost merges changes from source workspace into target workspace (host)
func (uc *GitUsecase) MergeHost(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error) {
	sourceDir := uc.getSandboxDir(sourceSandboxID)

	if _, err := uc.runGitCmdOnHost(ctx, targetSandboxID, "fetch", sourceDir); err != nil {
		return "", fmt.Errorf("failed to fetch source: %w", err)
	}

	branchOut, err := uc.runGitCmdOnHost(ctx, targetSandboxID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(branchOut)

	if _, err := uc.runGitCmdOnHost(ctx, targetSandboxID, "merge", "--no-ff", "-m", fmt.Sprintf("Merge changes from sandbox %s", sourceSandboxID), fmt.Sprintf("%s/%s", sourceDir, branch)); err != nil {
		return "", fmt.Errorf("merge failed: %w", err)
	}

	commitOut, err := uc.runGitCmdOnHost(ctx, targetSandboxID, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get merge commit: %w", err)
	}
	return strings.TrimSpace(commitOut), nil
}

// --- Container-based operations (sandbox-level) ---

func (uc *GitUsecase) GetStatus(ctx context.Context, sandboxID uuid.UUID) (*domain.GitStatus, error) {
	uc.runGitCmdInContainer(ctx, sandboxID, "fetch", "origin")

	branchOut, err := uc.runGitCmdInContainer(ctx, sandboxID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(branchOut)

	statusOut, err := uc.runGitCmdInContainer(ctx, sandboxID, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	dirty := strings.TrimSpace(statusOut) != ""

	var ahead, behind int
	revListOut, err := uc.runGitCmdInContainer(ctx, sandboxID, "rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...origin/%s", branch))
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(revListOut))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &ahead)
			fmt.Sscanf(parts[1], "%d", &behind)
		}
	}

	return &domain.GitStatus{
		Branch: branch,
		Dirty:  dirty,
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

func (uc *GitUsecase) ListBranches(ctx context.Context, sandboxID uuid.UUID) ([]string, error) {
	uc.runGitCmdInContainer(ctx, sandboxID, "fetch", "origin")
	out, err := uc.runGitCmdInContainer(ctx, sandboxID, "branch", "-a", "--format=%(refname:short)")
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

func (uc *GitUsecase) Checkout(ctx context.Context, sandboxID uuid.UUID, branchName string, force bool) error {
	forceCheckout := force

	if forceCheckout {
		uc.runGitCmdInContainer(ctx, sandboxID, "stash", "--include-untracked")
	}

	if strings.HasPrefix(branchName, "origin/") {
		localBranch := strings.TrimPrefix(branchName, "origin/")
		_, err := uc.runGitCmdInContainer(ctx, sandboxID, "checkout", "-b", localBranch, branchName)
		if err != nil {
			_, err = uc.runGitCmdInContainer(ctx, sandboxID, "checkout", localBranch)
			return err
		}
		return nil
	}
	_, err := uc.runGitCmdInContainer(ctx, sandboxID, "checkout", branchName)
	return err
}

func (uc *GitUsecase) Pull(ctx context.Context, sandboxID uuid.UUID) error {
	_, err := uc.runGitCmdInContainer(ctx, sandboxID, "pull", "--rebase")
	return err
}

func (uc *GitUsecase) CreateBranch(ctx context.Context, sandboxID uuid.UUID, branch string) error {
	_, err := uc.runGitCmdInContainer(ctx, sandboxID, "checkout", "-b", branch)
	return err
}

func (uc *GitUsecase) Commit(ctx context.Context, sandboxID uuid.UUID, message string) error {
	_, err := uc.runGitCmdInContainer(ctx, sandboxID, "add", ".")
	if err != nil {
		return err
	}
	_, err = uc.runGitCmdInContainer(ctx, sandboxID, "commit", "-m", message)
	return err
}

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

	branchOut, err := uc.runGitCmdInContainer(ctx, sandboxID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	currentBranch := strings.TrimSpace(branchOut)

	sandboxBranch := fmt.Sprintf("berth/%s", sandboxID.String())
	if currentBranch != sandboxBranch {
		if _, err := uc.runGitCmdInContainer(ctx, sandboxID, "checkout", "-b", sandboxBranch); err != nil {
			if _, checkoutErr := uc.runGitCmdInContainer(ctx, sandboxID, "checkout", sandboxBranch); checkoutErr != nil {
				return "", fmt.Errorf("could not create or switch to %s: %w", sandboxBranch, checkoutErr)
			}
		}
		currentBranch = sandboxBranch
	}

	if _, err := uc.runGitCmdInContainerWithToken(ctx, sandboxID, token, "push", "-u", "origin", currentBranch); err != nil {
		return "", fmt.Errorf("GitHub push failed; verify the OAuth grant has repository contents write access: %w", err)
	}
	return currentBranch, nil
}

func (uc *GitUsecase) Log(ctx context.Context, sandboxID uuid.UUID) ([]domain.CommitEntry, error) {
	out, err := uc.runGitCmdInContainer(ctx, sandboxID, "log", "-n", "50", "--pretty=format:%H|%h|%s|%an|%aI")
	if err != nil {
		return nil, err
	}

	var commits []domain.CommitEntry
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) == 5 {
			commits = append(commits, domain.CommitEntry{
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

func (uc *GitUsecase) GetCommits(ctx context.Context, sandboxID uuid.UUID) ([]domain.CommitInfo, error) {
	commits, err := uc.Log(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	var commitInfos []domain.CommitInfo
	for _, c := range commits {
		commitInfos = append(commitInfos, domain.CommitInfo{
			Hash:    c.Hash,
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date,
		})
	}
	return commitInfos, nil
}

// Interface implementation methods for domain.GitRepository

// GetChangedFiles implements domain.GitRepository
func (uc *GitUsecase) GetChangedFiles(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) ([]string, error) {
	return uc.GetChangedFilesHost(ctx, sourceSandboxID, targetSandboxID)
}

// GetDiff implements domain.GitRepository
func (uc *GitUsecase) GetDiff(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error) {
	return uc.GetDiffHost(ctx, sourceSandboxID, targetSandboxID)
}

// Merge implements domain.GitRepository
func (uc *GitUsecase) Merge(ctx context.Context, sourceSandboxID, targetSandboxID uuid.UUID) (string, error) {
	return uc.MergeHost(ctx, sourceSandboxID, targetSandboxID)
}