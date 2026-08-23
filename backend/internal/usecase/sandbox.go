package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type CreateEnvironmentRequest struct {
	Name      string `json:"name"`
	GitURL    string `json:"git_url"`
	GitBranch string `json:"git_branch"`
}

type SandboxUsecase struct {
	repo    domain.SandboxRepository
	runtime domain.ContainerRuntime
	ml      domain.PredictionService
}

func NewSandboxUsecase(repo domain.SandboxRepository, runtime domain.ContainerRuntime, ml domain.PredictionService) *SandboxUsecase {
	return &SandboxUsecase{repo: repo, runtime: runtime, ml: ml}
}

func (uc *SandboxUsecase) ListEnvironments(ctx context.Context, uid uuid.UUID) (any, error) {
	return uc.repo.ListByOwner(ctx, uid)
}

func (uc *SandboxUsecase) CreateEnvironment(ctx context.Context, uid uuid.UUID, req CreateEnvironmentRequest) (any, error) {
	env := &domain.Sandbox{
		ID:        uuid.New(),
		OwnerID:   uid,
		Name:      req.Name,
		GitURL:    req.GitURL,
		GitBranch: req.GitBranch,
		State:     domain.StatePending, // Worker will pick this up
	}

	if err := uc.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create sandbox in db: %w", err)
	}

	// No goroutine! The worker daemon polls PostgreSQL for PENDING sandboxes.
	return env, nil
}

func (uc *SandboxUsecase) GetEnvironment(ctx context.Context, id uuid.UUID) (any, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *SandboxUsecase) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sandbox.ContainerID != nil && uc.runtime != nil {
		if err := uc.runtime.DeleteSandbox(ctx, *sandbox.ContainerID); err != nil {
			slog.Error("failed to delete container", "error", err)
		}
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *SandboxUsecase) ExecCommand(ctx context.Context, id uuid.UUID, cmd []string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("exec is not supported in api-only mode")
	}
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if sandbox.ContainerID == nil {
		return "", fmt.Errorf("sandbox is not running (no container id)")
	}
	return uc.runtime.Exec(ctx, *sandbox.ContainerID, cmd)
}

func (uc *SandboxUsecase) GetLogs(ctx context.Context, id uuid.UUID, lines int) (string, error) {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if sandbox.ContainerID == nil {
		return "", fmt.Errorf("sandbox is not running (no container id)")
	}
	if uc.runtime != nil {
		return uc.runtime.GetLogs(ctx, *sandbox.ContainerID, lines)
	}

	// API Mode fallback (reads from shared host disk)
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".local", "state", "berth", "logs", *sandbox.ContainerID, "task.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
