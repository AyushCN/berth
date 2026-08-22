package usecase

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/AyushCN/berth/internal/domain"
	"log/slog"
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
		State:     domain.StateBuilding,
	}

	if err := uc.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create sandbox in db: %w", err)
	}

	go func(id uuid.UUID, spec domain.SandboxSpec) {
		bgCtx := context.Background()
		containerID, err := uc.runtime.CreateSandbox(bgCtx, spec)
		if err != nil {
			slog.Error("failed to create container", "error", err)
			_ = uc.repo.UpdateState(bgCtx, id, domain.StateFailed)
			return
		}
		
		if err := uc.runtime.StartSandbox(bgCtx, containerID); err != nil {
			slog.Error("failed to start container", "error", err)
			_ = uc.repo.UpdateState(bgCtx, id, domain.StateFailed)
			return
		}

		_ = uc.repo.UpdateState(bgCtx, id, domain.StateRunning)
	}(env.ID, domain.SandboxSpec{ID: env.ID, BaseImage: "node:18-alpine"})

	return env, nil
}

func (uc *SandboxUsecase) GetEnvironment(ctx context.Context, id uuid.UUID) (any, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *SandboxUsecase) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	if err := uc.runtime.DeleteSandbox(ctx, id.String()); err != nil {
		slog.Error("failed to delete container", "error", err)
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *SandboxUsecase) ExecCommand(ctx context.Context, id uuid.UUID, cmd []string) (string, error) {
	return uc.runtime.Exec(ctx, id.String(), cmd)
}

func (uc *SandboxUsecase) GetLogs(ctx context.Context, id uuid.UUID, lines int) (string, error) {
	return uc.runtime.GetLogs(ctx, id.String(), lines)
}
