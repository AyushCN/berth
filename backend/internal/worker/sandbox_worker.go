package worker

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/AyushCN/berth/internal/domain"
)

type SandboxWorker struct {
	repo    domain.SandboxRepository
	runtime domain.ContainerRuntime
	ml      domain.PredictionService
}

func NewSandboxWorker(repo domain.SandboxRepository, runtime domain.ContainerRuntime, ml domain.PredictionService) *SandboxWorker {
	return &SandboxWorker{
		repo:    repo,
		runtime: runtime,
		ml:      ml,
	}
}

func (w *SandboxWorker) Start(ctx context.Context) {
	slog.Info("sandbox worker started, polling for PENDING sandboxes")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sandbox worker shutting down")
			return
		case <-ticker.C:
			w.processPending(ctx)
		}
	}
}

func (w *SandboxWorker) processPending(ctx context.Context) {
	// 1. Atomically claim a PENDING sandbox
	sandbox, err := w.repo.PopPendingSandbox(ctx)
	if err != nil {
		// pgx.ErrNoRows or other error
		return
	}

	slog.Info("processing pending sandbox", "sandbox_id", sandbox.ID, "git_url", sandbox.GitURL)

	// 2. Prepare workspace directory
	home, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(home, ".local", "state", "berth", "workspaces", sandbox.ID.String())
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		slog.Error("failed to create workspace dir", "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// 3. Clone repository
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cloneCmd := exec.CommandContext(bgCtx, "git", "clone", "--depth", "1", "-b", sandbox.GitBranch, sandbox.GitURL, workspaceDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		slog.Error("git clone failed", "error", err, "output", string(out))
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// 4. Predict runtime profile
	profile, err := w.ml.Predict(bgCtx, sandbox.GitURL, sandbox.GitBranch)
	if err != nil {
		slog.Warn("prediction failed, using fallback", "error", err)
		profile = &domain.RuntimeProfile{
			Language:    "node",
			BaseImage:   "node:20-alpine",
			InstallCmd:  "npm install",
			StartCmd:    "npm run dev",
			ExposedPort: 3000,
			NeedsDB:     false,
			Confidence:  0.0,
		}
	}

	// 5. Create container with bind mount and keep-alive command
	spec := domain.SandboxSpec{
		ID:           sandbox.ID,
		BaseImage:    profile.BaseImage,
		WorkDir:      "/app",
		WorkspaceDir: workspaceDir,
		Cmd:          []string{"sh", "-c", "while true; do sleep 1; done"},
		MemoryLimit:  512 * 1024 * 1024,
		CPULimit:     1000,
	}

	cid, err := w.runtime.CreateSandbox(bgCtx, spec)
	if err != nil {
		slog.Error("worker failed to create container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	if err := w.repo.UpdateContainerID(bgCtx, sandbox.ID, cid); err != nil {
		slog.Error("worker failed to update container id", "sandbox_id", sandbox.ID, "error", err)
	}

	// 6. Start container
	if err := w.runtime.StartSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to start container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// 7. Install dependencies inside container
	if profile.InstallCmd != "" {
		installArgs := []string{"sh", "-c", "cd /app && " + profile.InstallCmd}
		if out, err := w.runtime.Exec(bgCtx, cid, installArgs); err != nil {
			slog.Error("dependency install failed", "sandbox_id", sandbox.ID, "error", err, "output", out)
			// Don't fail the sandbox; let user fix via terminal
		} else {
			slog.Info("dependencies installed", "sandbox_id", sandbox.ID, "output", out)
		}
	}

	// 8. Mark RUNNING
	if err := w.repo.UpdateState(bgCtx, sandbox.ID, domain.StateRunning); err != nil {
		slog.Error("worker failed to set RUNNING state", "sandbox_id", sandbox.ID, "error", err)
		return
	}

	slog.Info("sandbox provisioned successfully",
		"sandbox_id", sandbox.ID,
		"container_id", cid,
		"language", profile.Language,
		"base_image", profile.BaseImage,
	)
}
