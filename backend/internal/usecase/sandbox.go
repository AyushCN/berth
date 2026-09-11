package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AyushCN/berth/internal/domain"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/google/uuid"
)

type CreateEnvironmentRequest struct {
	Name      string     `json:"name"`
	GitURL    string     `json:"git_url"`
	GitBranch string     `json:"git_branch"`
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
}

type SandboxUsecase struct {
	repo        domain.SandboxRepository
	projectRepo domain.ProjectRepository
	runtime     domain.ContainerRuntime
	ml          domain.PredictionService
	natsClient  *natsInfra.Client
}

func NewSandboxUsecase(repo domain.SandboxRepository, projectRepo domain.ProjectRepository, runtime domain.ContainerRuntime, ml domain.PredictionService, natsClient *natsInfra.Client) *SandboxUsecase {
	return &SandboxUsecase{repo: repo, projectRepo: projectRepo, runtime: runtime, ml: ml, natsClient: natsClient}
}

func (uc *SandboxUsecase) ListEnvironments(ctx context.Context, uid uuid.UUID) (any, error) {
	return uc.repo.ListByOwner(ctx, uid)
}

func (uc *SandboxUsecase) CreateEnvironment(ctx context.Context, uid uuid.UUID, req CreateEnvironmentRequest) (any, error) {
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		_, err := uc.projectRepo.GetCollaborator(ctx, *req.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to create sandbox in project: %w", err)
		}
	}

	env := &domain.Sandbox{
		ID:        uuid.New(),
		OwnerID:   uid,
		Name:      req.Name,
		GitURL:    req.GitURL,
		GitBranch: req.GitBranch,
		State:     domain.StatePending, // Worker will pick this up
	}
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		env.ProjectID = *req.ProjectID
	}

	if err := uc.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create sandbox in db: %w", err)
	}

	if uc.natsClient != nil {
		payload := fmt.Sprintf(`{"sandbox_id":"%s","git_url":"%s","git_branch":"%s","owner_id":"%s"}`, env.ID, env.GitURL, env.GitBranch, env.OwnerID)
		if err := uc.natsClient.Publish("berth.sandbox.create", []byte(payload)); err != nil {
			slog.Warn("failed to publish sandbox create event to NATS", "error", err)
		} else {
			slog.Info("published sandbox create event to NATS", "sandbox_id", env.ID)
		}
	}

	// The worker daemon also polls PostgreSQL for PENDING sandboxes as fallback.
	return env, nil
}

func (uc *SandboxUsecase) GetEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) (any, error) {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Verify user has access to the sandbox
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return nil, fmt.Errorf("unauthorized to get sandbox")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to get sandbox: %w", err)
		}
	}
	return sandbox, nil
}

func (uc *SandboxUsecase) GetPreviewEnvironment(ctx context.Context, id uuid.UUID) (*domain.Sandbox, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *SandboxUsecase) DeleteEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// Verify user has access to delete the sandbox
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to delete sandbox")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to delete sandbox")
		}
	}
	if sandbox.ContainerID != nil && uc.runtime != nil {
		if err := uc.runtime.DeleteSandbox(ctx, *sandbox.ContainerID); err != nil {
			slog.Error("failed to delete container", "error", err)
		}
	}

	// Clean up workspace directory
	home, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(home, ".local", "state", "berth", "workspaces", id.String())
	if err := os.RemoveAll(workspaceDir); err != nil {
		slog.Error("failed to delete workspace dir", "sandbox_id", id, "error", err)
	}

	return uc.repo.Delete(ctx, id)
}

func (uc *SandboxUsecase) ExecCommand(ctx context.Context, uid uuid.UUID, id uuid.UUID, cmd []string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("exec is not supported in api-only mode")
	}
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return "", fmt.Errorf("unauthorized to exec in sandbox")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return "", fmt.Errorf("unauthorized to exec in sandbox")
		}
	}
	if sandbox.ContainerID == nil {
		return "", fmt.Errorf("sandbox is not running (no container id)")
	}
	return uc.runtime.Exec(ctx, *sandbox.ContainerID, cmd)
}

func (uc *SandboxUsecase) GetLogs(ctx context.Context, uid uuid.UUID, id uuid.UUID, lines int) (string, error) {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return "", fmt.Errorf("unauthorized to get logs")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil {
			return "", fmt.Errorf("unauthorized to get logs: %w", err)
		}
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

func (uc *SandboxUsecase) ForkEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID, req CreateEnvironmentRequest) (any, error) {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox to fork: %w", err)
	}
	
	// Verify user has access to the original sandbox
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return nil, fmt.Errorf("unauthorized to fork sandbox")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to fork sandbox: %w", err)
		}
	}

	// Verify user has access to the target project (if provided)
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		_, err := uc.projectRepo.GetCollaborator(ctx, *req.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to create sandbox in target project: %w", err)
		}
	}

	env := &domain.Sandbox{
		ID:        uuid.New(),
		OwnerID:   uid,
		Name:      req.Name,
		GitURL:    sandbox.GitURL,
		GitBranch: sandbox.GitBranch,
		State:     domain.StatePending, // Worker will pick this up
	}
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		env.ProjectID = *req.ProjectID
	}

	if err := uc.repo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create forked sandbox in db: %w", err)
	}

	return env, nil
}

func (uc *SandboxUsecase) StopEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to stop sandbox")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to stop sandbox")
		}
	}
	if sandbox.ContainerID != nil && uc.runtime != nil {
		if err := uc.runtime.StopSandbox(ctx, *sandbox.ContainerID); err != nil {
			slog.Error("failed to stop container", "error", err)
		}
	}
	return uc.repo.UpdateState(ctx, id, domain.StateStopped)
}

func (uc *SandboxUsecase) RestartEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	sandbox, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sandbox.OwnerID != uid {
		if sandbox.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to restart sandbox")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, sandbox.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to restart sandbox")
		}
	}
	if sandbox.ContainerID != nil && uc.runtime != nil {
		if err := uc.runtime.DeleteSandbox(ctx, *sandbox.ContainerID); err != nil {
			slog.Error("failed to delete container for restart", "error", err)
		}
	}
	// Reset container ID so it picks up a new one
	_ = uc.repo.UpdateContainerID(ctx, id, "")
	return uc.repo.UpdateState(ctx, id, domain.StatePending)
}
