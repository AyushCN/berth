package usecase

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type GitUsecase struct {
	workspaceDir  string
	userRepo      domain.UserRepository
	envRepo       domain.EnvironmentRepository
	workspaceRepo domain.WorkspaceRepository
	runtime       domain.ContainerRuntime
}

func NewGitUsecase(dir string, userRepo domain.UserRepository, envRepo domain.EnvironmentRepository, workspaceRepo domain.WorkspaceRepository, runtime domain.ContainerRuntime) *GitUsecase {
	return &GitUsecase{
		workspaceDir:  dir,
		userRepo:      userRepo,
		envRepo:       envRepo,
		workspaceRepo: workspaceRepo,
		runtime:       runtime,
	}
}

// workspaceDirFor resolves the host checkout directory for an environment.
// The directory is keyed by workspace id, matching what the worker provisions.
func (uc *GitUsecase) workspaceDirFor(ctx context.Context, environmentID uuid.UUID) (string, error) {
	wsID, err := workspaceIDFor(ctx, uc.envRepo, environmentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(uc.workspaceDir, wsID.String()), nil
}

func (uc *GitUsecase) Authorize(ctx context.Context, environmentID, userID uuid.UUID) error {
	env, err := uc.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("environment not found")
	}

	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found")
	}

	if ws.OwnerID != userID {
		return fmt.Errorf("unauthorized")
	}
	return nil
}

// runGitCmdOnHost runs git command on host filesystem (for workspace operations)
func (uc *GitUsecase) runGitCmdOnHost(ctx context.Context, environmentID uuid.UUID, args ...string) (string, error) {
	dir, err := uc.workspaceDirFor(ctx, environmentID)
	if err != nil {
		return "", err
	}

	// Fail cleanly before spawning git. exec would otherwise fail inside
	// chdir and wrap the absolute host path (WORKSPACE_ROOT/<id>) in the
	// error, which the handlers return verbatim to the client.
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("workspace for environment %s is not available on this host", environmentID)
		}
		return "", fmt.Errorf("workspace for environment %s is not readable", environmentID)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	if err != nil {
		// git's own stderr is safe to surface; the exec error is not, because
		// it embeds the host path.
		detail := strings.TrimSpace(errBuf.String())
		if detail == "" {
			detail = "no output"
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), detail)
	}
	return outBuf.String(), nil
}

// runGitCmdInContainer runs git command inside the sandbox container
func (uc *GitUsecase) runGitCmdInContainer(ctx context.Context, environmentID uuid.UUID, args ...string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("container runtime not available")
	}

	env, err := uc.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		return "", fmt.Errorf("environment not found: %w", err)
	}

	if env.ContainerID == "" {
		return "", fmt.Errorf("environment has no container")
	}

	if env.State != domain.EnvironmentStateRunning {
		return "", fmt.Errorf("environment is not running")
	}

	cmd := append([]string{"git"}, args...)
	return uc.runtime.Exec(ctx, env.ContainerID, cmd)
}

func (uc *GitUsecase) runGitCmdInContainerWithToken(ctx context.Context, environmentID uuid.UUID, token string, args ...string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("container runtime not available")
	}

	env, err := uc.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		return "", fmt.Errorf("environment not found: %w", err)
	}

	if env.ContainerID == "" {
		return "", fmt.Errorf("environment has no container")
	}

	if env.State != domain.EnvironmentStateRunning {
		return "", fmt.Errorf("environment is not running")
	}

	// Add GITHUB_TOKEN to environment
	cmd := append([]string{"git"}, args...)
	envVars := map[string]string{
		"GITHUB_TOKEN": token,
	}
	return uc.runtime.ExecWithEnv(ctx, env.ContainerID, cmd, envVars)
}

func (uc *GitUsecase) GetChangedFilesHost(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) ([]string, error) {
	sourceDir, err := uc.workspaceDirFor(ctx, sourceEnvironmentID)
	if err != nil {
		return nil, err
	}

	// First, fetch changes from source
	_, err = uc.runGitCmdOnHost(ctx, targetEnvironmentID, "fetch", sourceDir)
	if err != nil {
		return nil, err
	}

	// Get the current branch of target
	branchOut, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(branchOut)

	// Get diff between target branch and source
	diffOut, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "diff", "--name-only", fmt.Sprintf("%s/%s", sourceDir, branch))
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(diffOut), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

func (uc *GitUsecase) GetDiffHost(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) (string, error) {
	sourceDir, err := uc.workspaceDirFor(ctx, sourceEnvironmentID)
	if err != nil {
		return "", err
	}

	branchOut, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(branchOut)

	return uc.runGitCmdOnHost(ctx, targetEnvironmentID, "diff", fmt.Sprintf("%s/%s", sourceDir, branch))
}

func (uc *GitUsecase) MergeHost(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) (string, error) {
	sourceDir, err := uc.workspaceDirFor(ctx, sourceEnvironmentID)
	if err != nil {
		return "", err
	}

	branchOut, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(branchOut)

	// Fetch from source
	if _, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "fetch", sourceDir); err != nil {
		return "", err
	}

	// Merge
	mergeMsg := fmt.Sprintf("Merge changes from environment %s", sourceEnvironmentID)
	if _, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "merge", "--no-ff", "-m", mergeMsg, fmt.Sprintf("%s/%s", sourceDir, branch)); err != nil {
		return "", err
	}

	commitOut, err := uc.runGitCmdOnHost(ctx, targetEnvironmentID, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commitOut), nil
}

func (uc *GitUsecase) GetChangedFiles(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) ([]string, error) {
	return uc.GetChangedFilesHost(ctx, sourceEnvironmentID, targetEnvironmentID)
}

func (uc *GitUsecase) GetDiff(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) (string, error) {
	return uc.GetDiffHost(ctx, sourceEnvironmentID, targetEnvironmentID)
}

func (uc *GitUsecase) Merge(ctx context.Context, sourceEnvironmentID, targetEnvironmentID uuid.UUID) (string, error) {
	return uc.MergeHost(ctx, sourceEnvironmentID, targetEnvironmentID)
}

func (uc *GitUsecase) PushWithToken(ctx context.Context, environmentID uuid.UUID, token string) error {
	if err := uc.Authorize(ctx, environmentID, uuid.Nil); err != nil {
		return err
	}

	if uc.runtime == nil {
		return fmt.Errorf("container runtime not available")
	}

	env, err := uc.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	if env.ContainerID == "" {
		return fmt.Errorf("environment has no container")
	}

	if env.State != domain.EnvironmentStateRunning {
		return fmt.Errorf("environment is not running")
	}

	// Use container for git push with token
	cmd := []string{"push", "origin", "HEAD:refs/heads/main"}
	if _, err := uc.runGitCmdInContainerWithToken(ctx, environmentID, token, cmd...); err != nil {
		return err
	}
	return nil
}

func (uc *GitUsecase) GetStatusHost(ctx context.Context, environmentID uuid.UUID) (string, error) {
	out, err := uc.runGitCmdOnHost(ctx, environmentID, "status", "--porcelain")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (uc *GitUsecase) GetBranchesHost(ctx context.Context, environmentID uuid.UUID) ([]string, error) {
	out, err := uc.runGitCmdOnHost(ctx, environmentID, "branch", "-a")
	if err != nil {
		return nil, err
	}
	branches := strings.Split(strings.TrimSpace(out), "\n")
	if len(branches) == 1 && branches[0] == "" {
		return []string{}, nil
	}
	return branches, nil
}

func (uc *GitUsecase) CreateBranchHost(ctx context.Context, environmentID uuid.UUID, branchName string) error {
	_, err := uc.runGitCmdOnHost(ctx, environmentID, "checkout", "-b", branchName)
	return err
}

func (uc *GitUsecase) CheckoutHost(ctx context.Context, environmentID uuid.UUID, branchName string) error {
	_, err := uc.runGitCmdOnHost(ctx, environmentID, "checkout", branchName)
	return err
}

func (uc *GitUsecase) GetLogsHost(ctx context.Context, environmentID uuid.UUID, limit int) (string, error) {
	// "--oneline -20" must be two separate arguments. As one argument git
	// rejects it as an unknown option, so this always returned an error.
	out, err := uc.runGitCmdOnHost(ctx, environmentID, "log", "--oneline", fmt.Sprintf("-%d", limit))
	if err != nil {
		return "", err
	}
	return out, nil
}

// Wrapper methods for handler compatibility
func (uc *GitUsecase) ListBranches(ctx context.Context, environmentID uuid.UUID) ([]string, error) {
	return uc.GetBranchesHost(ctx, environmentID)
}

func (uc *GitUsecase) GetBranches(ctx context.Context, environmentID uuid.UUID) ([]string, error) {
	return uc.GetBranchesHost(ctx, environmentID)
}

func (uc *GitUsecase) CreateBranch(ctx context.Context, environmentID uuid.UUID, branchName string) error {
	return uc.CreateBranchHost(ctx, environmentID, branchName)
}

func (uc *GitUsecase) Checkout(ctx context.Context, environmentID uuid.UUID, branchName string, force bool) error {
	// Force checkout by adding -f flag if force is true
	if force {
		_, err := uc.runGitCmdOnHost(ctx, environmentID, "checkout", "-f", branchName)
		return err
	}
	return uc.CheckoutHost(ctx, environmentID, branchName)
}

func (uc *GitUsecase) Pull(ctx context.Context, environmentID uuid.UUID) error {
	_, err := uc.runGitCmdOnHost(ctx, environmentID, "pull")
	return err
}

func (uc *GitUsecase) Commit(ctx context.Context, environmentID uuid.UUID, message string) error {
	_, err := uc.runGitCmdOnHost(ctx, environmentID, "commit", "-m", message)
	return err
}

// commitLogFormat requests every field the UI renders. The previous
// `--oneline` output carries neither author nor date, so Author and Date were
// hardcoded to "". The frontend then called formatDistanceToNow(new Date("")),
// which throws RangeError: Invalid time value, and relativeTime("") rendered
// "NaNd ago".
//
// Fields are separated with \x1f (unit separator) so commit subjects
// containing spaces or the other delimiters still parse.
const commitLogFormat = "--pretty=format:%H%x1f%h%x1f%an%x1f%aI%x1f%s"

const commitLogSeparator = "\x1f"

// GetCommits returns the commit history for an environment
func (uc *GitUsecase) GetCommits(ctx context.Context, environmentID uuid.UUID) ([]domain.CommitEntry, error) {
	out, err := uc.runGitCmdOnHost(ctx, environmentID, "log", commitLogFormat, "-20")
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return []domain.CommitEntry{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	commits := make([]domain.CommitEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, commitLogSeparator, 5)
		if len(parts) < 5 {
			continue
		}
		commits = append(commits, domain.CommitEntry{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[4],
			Author:    parts[2],
			Date:      parts[3],
		})
	}
	return commits, nil
}

// Push pushes commits to remote with user token
func (uc *GitUsecase) Push(ctx context.Context, environmentID uuid.UUID, userID uuid.UUID) (string, error) {
	// This would need the user's token - for now just do a regular push
	_, err := uc.runGitCmdOnHost(ctx, environmentID, "push")
	if err != nil {
		return "", err
	}
	return "main", nil
}

// GetStatus returns the git status for an environment
func (uc *GitUsecase) GetStatus(ctx context.Context, environmentID uuid.UUID) (*domain.GitStatus, error) {
	out, err := uc.runGitCmdOnHost(ctx, environmentID, "status", "--porcelain", "--branch")
	if err != nil {
		return nil, err
	}

	status := &domain.GitStatus{}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Parse branch info: ## branchname...origin/branchname [ahead X, behind Y]
			branchInfo := strings.TrimPrefix(line, "## ")
			parts := strings.Fields(branchInfo)
			if len(parts) > 0 {
				status.Branch = parts[0]
			}
			// Parse ahead/behind
			for _, part := range parts {
				if strings.HasPrefix(part, "[ahead") {
					fmt.Sscanf(part, "[ahead %d", &status.Ahead)
				} else if strings.HasPrefix(part, "behind") {
					fmt.Sscanf(part, "behind %d]", &status.Behind)
				}
			}
		} else if line != "" {
			status.Dirty = true
		}
	}
	return status, nil
}

// Log returns the commit log for an environment
func (uc *GitUsecase) Log(ctx context.Context, environmentID uuid.UUID) ([]domain.CommitEntry, error) {
	return uc.GetCommits(ctx, environmentID)
}
