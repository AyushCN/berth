package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/AyushCN/berth/internal/domain"
)

type SandboxWorker struct {
	repo    domain.SandboxRepository
	runtime domain.ContainerRuntime
}

func NewSandboxWorker(repo domain.SandboxRepository, runtime domain.ContainerRuntime) *SandboxWorker {
	return &SandboxWorker{
		repo:    repo,
		runtime: runtime,
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
	// PopPendingSandbox sets state to BUILDING atomically
	sandbox, err := w.repo.PopPendingSandbox(ctx)
	if err != nil {
		// sql.ErrNoRows or pgx.ErrNoRows is expected, but depending on the adapter we might need to check the string.
		// For now we just ignore if no rows are returned. If it's a real error, we might log it, but we don't want to spam.
		if err.Error() != "no rows in result set" {
			slog.Debug("no pending sandboxes or error", "error", err)
		}
		return
	}

	slog.Info("processing pending sandbox", "sandbox_id", sandbox.ID)

	// Create and Start container
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	spec := domain.SandboxSpec{
		ID:        sandbox.ID,
		BaseImage: "node:18-alpine", // Default for now
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

	if err := w.runtime.StartSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to start container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	if err := w.repo.UpdateState(bgCtx, sandbox.ID, domain.StateRunning); err != nil {
		slog.Error("worker failed to set RUNNING state", "sandbox_id", sandbox.ID, "error", err)
	}

	slog.Info("sandbox started successfully", "sandbox_id", sandbox.ID)
}
