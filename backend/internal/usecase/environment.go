package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/google/uuid"
)

type EnvironmentCreateRequest struct {
	Name           string     `json:"name"`
	GitURL         string     `json:"git_url"`
	GitBranch      string     `json:"git_branch"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	RuntimeProfile *uuid.UUID `json:"runtime_profile_id,omitempty"`
}

type EnvironmentUsecase struct {
	envRepo       domain.EnvironmentRepository
	workspaceRepo domain.WorkspaceRepository
	projectRepo   domain.ProjectRepository
	orgRepo       domain.OrganizationRepository
	runtime       domain.ContainerRuntime
	natsClient    *natsInfra.Client
}

func NewEnvironmentUsecase(envRepo domain.EnvironmentRepository, workspaceRepo domain.WorkspaceRepository, projectRepo domain.ProjectRepository, orgRepo domain.OrganizationRepository, runtime domain.ContainerRuntime, natsClient *natsInfra.Client) *EnvironmentUsecase {
	return &EnvironmentUsecase{
		envRepo:       envRepo,
		workspaceRepo: workspaceRepo,
		projectRepo:   projectRepo,
		orgRepo:       orgRepo,
		runtime:       runtime,
		natsClient:    natsClient,
	}
}

func (uc *EnvironmentUsecase) ListEnvironments(ctx context.Context, uid uuid.UUID) ([]*domain.Environment, error) {
	// Get user's workspaces first, then get environments for those workspaces
	workspaces, err := uc.workspaceRepo.GetUserWorkspaces(ctx, uid)
	if err != nil {
		return nil, err
	}

	var allEnvs []*domain.Environment
	for _, ws := range workspaces {
		envs, err := uc.envRepo.GetByWorkspace(ctx, ws.ID)
		if err != nil {
			continue
		}
		allEnvs = append(allEnvs, envs...)
	}
	return allEnvs, nil
}

func (uc *EnvironmentUsecase) CreateEnvironment(ctx context.Context, uid uuid.UUID, req EnvironmentCreateRequest) (*domain.Environment, error) {
	// Verify project access if provided
	var projectID uuid.UUID
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		_, err := uc.projectRepo.GetCollaborator(ctx, *req.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to create environment in project: %w", err)
		}
		projectID = *req.ProjectID
	} else {
		// Create or get a default project for the user
		org, err := uc.getOrCreateUserOrg(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("failed to get/create user org: %w", err)
		}
		proj, err := uc.getOrCreateDefaultProject(ctx, uid, org.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get/create default project: %w", err)
		}
		projectID = proj.ID
	}

	// Create workspace first
	ws := &domain.Workspace{
		ID:                    uuid.New(),
		ProjectID:             projectID,
		Name:                  req.Name,
		Type:                  domain.WorkspaceTypeCanonical,
		BaseWorkspaceID:       nil,
		OwnerID:               uid,
		GitURL:                req.GitURL,
		GitBranch:             req.GitBranch,
		CommitHash:            "",
		HasUncommittedChanges: false,
	}

	if err := uc.workspaceRepo.Create(ctx, ws); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create environment
	var runtimeProfileID *uuid.UUID
	if req.RuntimeProfile != nil {
		runtimeProfileID = req.RuntimeProfile
	}

	env := &domain.Environment{
		ID:               uuid.New(),
		WorkspaceID:      ws.ID,
		RuntimeProfileID: runtimeProfileID,
		Name:             req.Name,
		State:            domain.EnvironmentStateCreated,
		MemoryLimit:      512 * 1024 * 1024, // 512 MiB default
		CPULimit:         1000000000,        // 1 CPU default
	}

	if err := uc.envRepo.Create(ctx, env); err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	// Publish NATS event for worker.
	//
	// The URL is also persisted on the workspace, so a worker that misses this
	// message (or starts later) can still recover by polling environments in
	// CREATED. Prefer the persisted copy so the two can never disagree.
	if uc.natsClient != nil {
		payload, err := json.Marshal(domain.EnvironmentCreateEvent{
			EnvironmentID: env.ID,
			WorkspaceID:   ws.ID,
			GitURL:        ws.GitURL,
			GitBranch:     ws.GitBranch,
			OwnerID:       uid,
		})
		if err != nil {
			slog.Warn("failed to marshal environment create event", "error", err)
		} else if err := uc.natsClient.Publish(domain.SubjectEnvironmentCreate, payload); err != nil {
			slog.Warn("failed to publish environment create event to NATS", "error", err)
		} else {
			slog.Info("published environment create event to NATS", "environment_id", env.ID)
		}
	}

	return env, nil
}

func (uc *EnvironmentUsecase) GetEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) (*domain.Environment, error) {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return nil, fmt.Errorf("unauthorized to get environment")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to get environment: %w", err)
		}
	}

	return env, nil
}

func (uc *EnvironmentUsecase) GetPreviewEnvironment(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	return uc.envRepo.GetByID(ctx, id)
}

func (uc *EnvironmentUsecase) DeleteEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to delete environment")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to delete environment")
		}
	}

	// Stop and delete container if exists
	if env.ContainerID != "" && uc.runtime != nil {
		if err := uc.runtime.DeleteSandbox(ctx, env.ContainerID); err != nil {
			return fmt.Errorf("failed to delete environment container: %w", err)
		}
	} else if env.ContainerID != "" {
		if uc.natsClient == nil {
			return fmt.Errorf("worker cleanup is unavailable")
		}
		payload, _ := json.Marshal(domain.EnvironmentLifecycleEvent{
			EnvironmentID: id,
			WorkspaceID:   ws.ID,
			ContainerID:   env.ContainerID,
		})
		if err := uc.natsClient.Publish(domain.SubjectEnvironmentDelete, payload); err != nil {
			return fmt.Errorf("failed to request environment cleanup: %w", err)
		}
	}

	// Delete workspace directory
	workspaceRoot := os.Getenv("WORKSPACE_ROOT")
	if workspaceRoot == "" {
		home, _ := os.UserHomeDir()
		workspaceRoot = filepath.Join(home, ".local", "state", "berth", "workspaces")
	}
	workspaceDir := filepath.Join(workspaceRoot, ws.ID.String())
	if err := os.RemoveAll(workspaceDir); err != nil {
		slog.Error("failed to delete workspace dir", "workspace_id", ws.ID, "error", err)
	}

	// Soft delete environment and workspace
	if err := uc.envRepo.Delete(ctx, id); err != nil {
		return err
	}
	return uc.workspaceRepo.Delete(ctx, ws.ID)
}

func (uc *EnvironmentUsecase) ExecCommand(ctx context.Context, uid uuid.UUID, id uuid.UUID, cmd []string) (string, error) {
	if uc.runtime == nil {
		return "", fmt.Errorf("exec is not supported in api-only mode")
	}
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return "", err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return "", fmt.Errorf("unauthorized to exec in environment")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return "", fmt.Errorf("unauthorized to exec in environment")
		}
	}

	if env.ContainerID == "" {
		return "", fmt.Errorf("environment is not running (no container id)")
	}
	return uc.runtime.Exec(ctx, env.ContainerID, cmd)
}

func (uc *EnvironmentUsecase) GetLogs(ctx context.Context, uid uuid.UUID, id uuid.UUID, lines int) (string, error) {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return "", err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return "", fmt.Errorf("unauthorized to get logs")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil {
			return "", fmt.Errorf("unauthorized to get logs: %w", err)
		}
	}

	if env.ContainerID == "" {
		return "", fmt.Errorf("environment is not running (no container id)")
	}
	if uc.runtime != nil {
		return uc.runtime.GetLogs(ctx, env.ContainerID, lines)
	}

	// API Mode fallback (reads from shared host disk)
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".local", "state", "berth", "logs", env.ContainerID, "task.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (uc *EnvironmentUsecase) ForkEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID, req EnvironmentCreateRequest) (*domain.Environment, error) {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment to fork: %w", err)
	}

	// Verify user has access to the original environment
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return nil, fmt.Errorf("unauthorized to fork environment")
		}
		_, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to fork environment: %w", err)
		}
	}

	// Verify user has access to the target project (if provided)
	if req.ProjectID != nil && *req.ProjectID != uuid.Nil {
		_, err := uc.projectRepo.GetCollaborator(ctx, *req.ProjectID, uid)
		if err != nil {
			return nil, fmt.Errorf("unauthorized to create environment in target project: %w", err)
		}
	}

	// Create new workspace as fork
	newWS := &domain.Workspace{
		ID:                    uuid.New(),
		ProjectID:             ws.ProjectID,
		Name:                  req.Name,
		Type:                  domain.WorkspaceTypeFork,
		BaseWorkspaceID:       &ws.ID,
		OwnerID:               uid,
		GitURL:                ws.GitURL,
		GitBranch:             ws.GitBranch,
		CommitHash:            ws.CommitHash,
		HasUncommittedChanges: false,
	}

	if err := uc.workspaceRepo.Create(ctx, newWS); err != nil {
		return nil, fmt.Errorf("failed to create fork workspace: %w", err)
	}

	// Create new environment for fork
	newEnv := &domain.Environment{
		ID:               uuid.New(),
		WorkspaceID:      newWS.ID,
		RuntimeProfileID: env.RuntimeProfileID,
		Name:             req.Name,
		State:            domain.EnvironmentStateCreated,
		MemoryLimit:      env.MemoryLimit,
		CPULimit:         env.CPULimit,
	}

	if err := uc.envRepo.Create(ctx, newEnv); err != nil {
		return nil, fmt.Errorf("failed to create fork environment: %w", err)
	}

	// Publish NATS event for worker.
	// Include GitURL: it was previously omitted here, so a fork reached the
	// worker with no repository to clone even once a subscriber existed.
	if uc.natsClient != nil {
		payload, err := json.Marshal(domain.EnvironmentCreateEvent{
			EnvironmentID: newEnv.ID,
			WorkspaceID:   newWS.ID,
			GitURL:        ws.GitURL,
			GitBranch:     newWS.GitBranch,
			OwnerID:       uid,
		})
		if err != nil {
			slog.Warn("failed to marshal fork environment create event", "error", err)
		} else if err := uc.natsClient.Publish(domain.SubjectEnvironmentCreate, payload); err != nil {
			slog.Warn("failed to publish fork environment create event to NATS", "error", err)
		} else {
			slog.Info("published fork environment create event to NATS", "environment_id", newEnv.ID)
		}
	}

	return newEnv, nil
}

func (uc *EnvironmentUsecase) StopEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to stop environment")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to stop environment")
		}
	}

	workerHandlesStop := false
	if env.ContainerID != "" && uc.runtime != nil {
		if err := uc.runtime.StopSandbox(ctx, env.ContainerID); err != nil {
			return fmt.Errorf("failed to stop environment container: %w", err)
		}
	} else if env.ContainerID != "" {
		if uc.natsClient == nil {
			return fmt.Errorf("worker stop service is unavailable")
		}
		payload, _ := json.Marshal(domain.EnvironmentLifecycleEvent{
			EnvironmentID: id,
			WorkspaceID:   ws.ID,
			ContainerID:   env.ContainerID,
		})
		if err := uc.natsClient.Publish(domain.SubjectEnvironmentStop, payload); err != nil {
			return fmt.Errorf("failed to request environment stop: %w", err)
		}
		workerHandlesStop = true
	}

	if workerHandlesStop {
		return nil
	}
	return uc.envRepo.UpdateState(ctx, id, domain.EnvironmentStateStopped)
}

func (uc *EnvironmentUsecase) RestartEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to restart environment")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to restart environment")
		}
	}

	if env.ContainerID != "" && uc.runtime != nil {
		if err := uc.runtime.DeleteSandbox(ctx, env.ContainerID); err != nil {
			slog.Error("failed to delete container for restart", "error", err)
		}
	}

	// Reset container ID so it picks up a new one
	_ = uc.envRepo.UpdateContainerID(ctx, id, "")
	return uc.envRepo.UpdateState(ctx, id, domain.EnvironmentStateCreated)
}

func (uc *EnvironmentUsecase) StartEnvironment(ctx context.Context, uid uuid.UUID, id uuid.UUID) error {
	env, err := uc.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify user has access via workspace
	ws, err := uc.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return err
	}

	if ws.OwnerID != uid {
		if ws.ProjectID == uuid.Nil {
			return fmt.Errorf("unauthorized to start environment")
		}
		collab, err := uc.projectRepo.GetCollaborator(ctx, ws.ProjectID, uid)
		if err != nil || collab.Role == domain.ProjectRoleViewer {
			return fmt.Errorf("unauthorized to start environment")
		}
	}

	// Only allow starting from a state that has nothing running. CRASHED is
	// included because a crashed container is exactly what the user wants back.
	// SUSPENDED is handled by the separate resume endpoint.
	switch env.State {
	case domain.EnvironmentStateStopped, domain.EnvironmentStateBuildFailed, domain.EnvironmentStateCrashed:
	case domain.EnvironmentStateRunning, domain.EnvironmentStateStarting, domain.EnvironmentStateBuilding,
		domain.EnvironmentStateCreated, domain.EnvironmentStateReady:
		return fmt.Errorf("environment is already %s", env.State)
	default:
		return fmt.Errorf("can only start environment in stopped, failed or crashed state, current state: %s", env.State)
	}

	// Prefer restarting the existing container. Resetting to CREATED instead
	// makes the worker rebuild from scratch: re-clone, reinstall every
	// dependency, recommit the image. The container is still present after a
	// stop, so ask the worker to start it.
	if env.ContainerID != "" && uc.natsClient != nil {
		payload, err := json.Marshal(domain.EnvironmentLifecycleEvent{
			EnvironmentID: env.ID,
			WorkspaceID:   ws.ID,
			ContainerID:   env.ContainerID,
		})
		if err != nil {
			slog.Warn("failed to marshal environment start request", "error", err)
		} else if err := uc.natsClient.Publish(domain.SubjectEnvironmentStart, payload); err != nil {
			slog.Warn("failed to request environment start, falling back to rebuild", "error", err)
		} else {
			return nil
		}
	}

	// No container, or no message bus: rebuild from scratch.
	return uc.envRepo.UpdateState(ctx, id, domain.EnvironmentStateCreated)
}

// getOrCreateUserOrg gets or creates an organization for the user
func (uc *EnvironmentUsecase) getOrCreateUserOrg(ctx context.Context, uid uuid.UUID) (*domain.Organization, error) {
	// Try to find existing org for user
	orgs, err := uc.orgRepo.ListForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	if len(orgs) > 0 {
		return orgs[0], nil
	}
	// Create new org
	return uc.orgRepo.Create(ctx, fmt.Sprintf("%s's Workspace", uid.String()[:8]))
}

// getOrCreateDefaultProject gets or creates a default project in the org
func (uc *EnvironmentUsecase) getOrCreateDefaultProject(ctx context.Context, uid, orgID uuid.UUID) (*domain.Project, error) {
	projs, err := uc.projectRepo.ListForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	for _, p := range projs {
		if p.OwnerOrganizationID == orgID {
			return p, nil
		}
	}
	p := &domain.Project{
		ID:                  uuid.New(),
		Name:                "My Project",
		Description:         &[]string{"Default project"}[0],
		OwnerOrganizationID: orgID,
		CreatedByUserID:     uid,
		IsPublic:            false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	if err := uc.projectRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
