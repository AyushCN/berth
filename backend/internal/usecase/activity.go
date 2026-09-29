package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type ActivityTracker struct {
	repo                 domain.EnvironmentRepository
	control              containerControl
	idleTimeout          time.Duration
	suspendCheckInterval time.Duration
	mu                   sync.Mutex
	running              bool
	stopCh               chan struct{}
}

// NewActivityTracker builds a tracker. control must not be nil: it decides how
// containers are stopped and started. The worker passes a Docker-backed
// controller; the api passes one that asks the worker over NATS.
func NewActivityTracker(
	repo domain.EnvironmentRepository,
	control containerControl,
	idleTimeout time.Duration,
	suspendCheckInterval time.Duration,
) *ActivityTracker {
	if control == nil {
		// A nil controller here would panic on first use, which is exactly
		// how this class of bug reached main before.
		control = natsContainerControl{}
	}
	return &ActivityTracker{
		repo:                 repo,
		control:              control,
		idleTimeout:          idleTimeout,
		suspendCheckInterval: suspendCheckInterval,
		stopCh:               make(chan struct{}),
	}
}

func (at *ActivityTracker) Start(ctx context.Context) {
	at.mu.Lock()
	if at.running {
		at.mu.Unlock()
		return
	}
	at.running = true
	at.mu.Unlock()

	slog.Info("activity tracker started", "idle_timeout", at.idleTimeout, "check_interval", at.suspendCheckInterval)

	go at.runSuspendLoop(ctx)
}

func (at *ActivityTracker) Stop() {
	at.mu.Lock()
	defer at.mu.Unlock()
	if !at.running {
		return
	}
	at.running = false
	close(at.stopCh)
	slog.Info("activity tracker stopped")
}

func (at *ActivityTracker) RecordActivity(ctx context.Context, environmentID uuid.UUID) error {
	return at.repo.UpdateActivity(ctx, environmentID, 0) // 0 means just update timestamp
}

func (at *ActivityTracker) RecordSessionStart(ctx context.Context, environmentID uuid.UUID) error {
	return at.repo.UpdateActivity(ctx, environmentID, 1) // increment sessions
}

func (at *ActivityTracker) RecordSessionEnd(ctx context.Context, environmentID uuid.UUID) error {
	return at.repo.UpdateActivity(ctx, environmentID, -1) // decrement sessions
}

func (at *ActivityTracker) runSuspendLoop(ctx context.Context) {
	ticker := time.NewTicker(at.suspendCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-at.stopCh:
			return
		case <-ticker.C:
			at.checkAndSuspendIdleEnvironments(ctx)
		}
	}
}

func (at *ActivityTracker) checkAndSuspendIdleEnvironments(ctx context.Context) {
	// Find environments that are RUNNING and idle
	idleThreshold := time.Now().Add(-at.idleTimeout)
	environments, err := at.repo.ListIdleRunning(ctx, idleThreshold)
	if err != nil {
		slog.Error("failed to list idle running environments", "error", err)
		return
	}

	for _, env := range environments {
		// Double-check the environment is still running and idle
		if env.State != domain.EnvironmentStateRunning {
			continue
		}
		if env.ActiveSessions > 0 {
			continue
		}
		if env.LastActivityAt != nil && env.LastActivityAt.After(idleThreshold) {
			continue
		}

		slog.Info("suspending idle environment", "environment_id", env.ID, "idle_since", env.LastActivityAt)

		// Update state to SUSPENDING
		if err := at.repo.UpdateState(ctx, env.ID, domain.EnvironmentStateSuspending); err != nil {
			slog.Error("failed to update state to suspending", "environment_id", env.ID, "error", err)
			continue
		}

		// Stop the container. If this fails, revert to RUNNING rather than
		// claiming the environment is suspended while its container lives on.
		if env.ContainerID != "" {
			if err := at.control.Stop(ctx, env); err != nil {
				slog.Error("failed to stop idle environment container", "environment_id", env.ID, "error", err)
				_ = at.repo.UpdateState(ctx, env.ID, domain.EnvironmentStateRunning)
				continue
			}
		}

		// Update state to SUSPENDED - keep ContainerID so we can resume later
		now := time.Now()
		if err := at.repo.Update(ctx, &domain.Environment{
			ID:          env.ID,
			State:       domain.EnvironmentStateSuspended,
			SuspendedAt: &now,
			// ContainerID: keep it for resume
			UpdatedAt: now,
		}); err != nil {
			slog.Error("failed to update environment to suspended", "environment_id", env.ID, "error", err)
		}
	}
}

// ResumeEnvironment resumes a suspended environment
func (at *ActivityTracker) ResumeEnvironment(ctx context.Context, environmentID uuid.UUID) error {
	env, err := at.repo.GetByID(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	if env.State != domain.EnvironmentStateSuspended {
		return fmt.Errorf("environment is not suspended")
	}

	// Update state to STARTING
	if err := at.repo.UpdateState(ctx, environmentID, domain.EnvironmentStateStarting); err != nil {
		return fmt.Errorf("failed to update state to starting: %w", err)
	}

	// Start the container again
	if env.ContainerID != "" {
		if err := at.control.Start(ctx, env); err != nil {
			slog.Error("failed to start suspended environment container", "environment_id", env.ID, "error", err)
			// Revert state
			_ = at.repo.UpdateState(ctx, environmentID, domain.EnvironmentStateSuspended)
			return fmt.Errorf("failed to start sandbox: %w", err)
		}
	}

	now := time.Now()
	if err := at.repo.Update(ctx, &domain.Environment{
		ID:          environmentID,
		State:       domain.EnvironmentStateRunning,
		SuspendedAt: nil,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("failed to update environment to running: %w", err)
	}

	return nil
}
