package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	natsCore "github.com/nats-io/nats.go"
)

// containerWorkspaceDir is where the host workspace directory is mounted
// inside every sandbox and environment container.
const containerWorkspaceDir = "/workspace"

// defaultCPUMilliCores is 1 CPU. domain.SandboxSpec measures CPU in
// milli-cores while domain.Environment measures it in nanocpus.
const defaultCPUMilliCores = 1000

// cpuMilliCores converts the nanocpus used by domain.Environment (where
// 1e9 means one CPU) into the milli-cores domain.SandboxSpec expects.
func cpuMilliCores(nanocores int64) int64 {
	if nanocores <= 0 {
		return 0
	}
	return nanocores / 1_000_000
}

// subscribeEnvironment registers the environment lifecycle subscriptions.
//
// These subjects were published by the api with no subscriber at all, so
// every environment created over HTTP sat in CREATED forever: no clone, no
// container, no logs, no exec. Stop and delete were worse, because the api
// logged success and returned 204 while the container kept running.
func (w *SandboxWorker) subscribeEnvironment(ctx context.Context) {
	if w.natsClient == nil {
		slog.Warn("no NATS client; environment provisioning disabled")
		return
	}

	subs := []struct {
		subject string
		handler natsCore.MsgHandler
		label   string
	}{
		{domain.SubjectEnvironmentCreate, w.handleEnvironmentCreate, "create"},
		{domain.SubjectEnvironmentStop, w.handleEnvironmentStop, "stop"},
		{domain.SubjectEnvironmentDelete, w.handleEnvironmentDelete, "delete"},
	}

	for _, s := range subs {
		subject, handler, label := s.subject, s.handler, s.label
		if _, err := w.natsClient.Subscribe(subject, "", handler); err != nil {
			slog.Error("failed to subscribe to environment events", "subject", subject, "error", err)
			continue
		}
		slog.Info("subscribed to environment events", "subject", subject, "event", label)
	}
	_ = ctx
}

func (w *SandboxWorker) handleEnvironmentCreate(msg *natsCore.Msg) {
	var evt domain.EnvironmentCreateEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		slog.Error("invalid environment create event", "error", err)
		_ = msg.Term()
		return
	}
	if evt.EnvironmentID == uuid.Nil {
		slog.Error("environment create event missing environment_id")
		_ = msg.Term()
		return
	}
	_ = msg.Ack()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.processEnvironment(context.Background(), evt)
	}()
}

// processEnvironment provisions a single environment.
//
// The workspace is the source of truth for the repository: the NATS payload
// only accelerates delivery. That means a missed message is recoverable via
// reapPendingEnvironments, which is what makes the pipeline survive restarts.
func (w *SandboxWorker) processEnvironment(ctx context.Context, evt domain.EnvironmentCreateEvent) {
	env, err := w.envRepo.GetByID(ctx, evt.EnvironmentID)
	if err != nil {
		slog.Error("failed to load environment", "environment_id", evt.EnvironmentID, "error", err)
		return
	}

	ws, err := w.workspaceRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		slog.Error("failed to load workspace", "workspace_id", env.WorkspaceID, "error", err)
		return
	}

	// Already provisioned or torn down: nothing to do.
	switch env.State {
	case domain.EnvironmentStateRunning, domain.EnvironmentStateStopping, domain.EnvironmentStateDeleting:
		slog.Info("environment already provisioned, skipping", "environment_id", env.ID, "state", env.State)
		return
	}

	gitURL, gitBranch := ws.GitURL, ws.GitBranch
	if gitURL == "" {
		gitURL = evt.GitURL
	}
	if gitBranch == "" {
		gitBranch = evt.GitBranch
	}
	if gitURL == "" {
		slog.Error("no git url for environment, cannot provision",
			"environment_id", env.ID, "workspace_id", ws.ID)
		_ = w.envRepo.UpdateState(context.Background(), env.ID, domain.EnvironmentStateBuildFailed)
		return
	}

	slog.Info("provisioning environment",
		"environment_id", env.ID, "workspace_id", ws.ID, "git_url", gitURL, "git_branch", gitBranch)

	if err := w.envRepo.UpdateState(ctx, env.ID, domain.EnvironmentStateBuilding); err != nil {
		slog.Error("failed to mark environment BUILDING", "environment_id", env.ID, "error", err)
	}

	if err := os.MkdirAll(workspaceRoot(), 0755); err != nil {
		slog.Error("failed to create workspace root", "error", err)
		_ = w.envRepo.UpdateState(context.Background(), env.ID, domain.EnvironmentStateBuildFailed)
		return
	}

	// The workspace owns the checkout, so the on-disk directory is keyed by
	// workspace id. The api's file and git endpoints resolve the same way,
	// which is why they previously pointed at directories that were never
	// created.
	workspaceDir := filepath.Join(workspaceRoot(), ws.ID.String())

	res, err := w.provision(ctx, provisionRequest{
		ID:           env.ID,
		WorkspaceDir: workspaceDir,
		WorkDir:      containerWorkspaceDir,
		GitURL:       gitURL,
		GitBranch:    gitBranch,
		OwnerID:      ws.OwnerID,
		MemoryLimit:  env.MemoryLimit,
		CPULimit:     cpuMilliCores(env.CPULimit),
	})
	if err != nil {
		slog.Error("failed to provision environment", "environment_id", env.ID, "error", err)
		_ = w.envRepo.UpdateState(context.Background(), env.ID, domain.EnvironmentStateBuildFailed)
		return
	}

	if err := w.envRepo.UpdateContainerID(context.Background(), env.ID, res.ContainerID); err != nil {
		slog.Error("failed to persist container id", "environment_id", env.ID, "error", err)
	}
	if err := w.envRepo.Update(ctx, &domain.Environment{
		ID:        env.ID,
		Name:      env.Name,
		PublicURL: res.PublicURL,
		Port:      res.Port,
	}); err != nil {
		slog.Error("failed to persist public url and port", "environment_id", env.ID, "error", err)
	}
	if err := w.envRepo.UpdateState(context.Background(), env.ID, domain.EnvironmentStateRunning); err != nil {
		slog.Error("failed to mark environment RUNNING", "environment_id", env.ID, "error", err)
		return
	}

	w.afterProvision(env.ID, res)

	slog.Info("environment provisioned successfully",
		"environment_id", env.ID,
		"workspace_id", ws.ID,
		"container_id", res.ContainerID,
		"language", res.Profile.Language,
		"public_url", res.PublicURL,
		"timing_metrics", res.Timings,
	)
}

// reapPendingEnvironments is the recovery path for environments that were
// created while no worker was listening, or whose NATS message was lost.
// Safe to call on a timer: it only touches environments still in CREATED.
func (w *SandboxWorker) reapPendingEnvironments(ctx context.Context) {
	pending, err := w.envRepo.ListByState(ctx, domain.EnvironmentStateCreated)
	if err != nil {
		slog.Warn("failed to list pending environments", "error", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	slog.Info("reaping pending environments", "count", len(pending))
	for _, env := range pending {
		// Claim it before doing any work so a concurrent tick or message
		// does not provision the same environment twice.
		if err := w.envRepo.UpdateState(ctx, env.ID, domain.EnvironmentStateStarting); err != nil {
			slog.Warn("failed to claim environment", "environment_id", env.ID, "error", err)
			continue
		}
		if err := w.envRepo.UpdateState(ctx, env.ID, domain.EnvironmentStateCreated); err != nil {
			continue
		}
		evt := domain.EnvironmentCreateEvent{EnvironmentID: env.ID, WorkspaceID: env.WorkspaceID}
		w.processEnvironment(ctx, evt)
	}
}

func (w *SandboxWorker) handleEnvironmentStop(msg *natsCore.Msg) {
	evt, ok := w.decodeLifecycle(msg, "stop")
	if !ok {
		return
	}
	_ = msg.Ack()

	if evt.ContainerID == "" {
		slog.Warn("stop event without container id; loading from database", "environment_id", evt.EnvironmentID)
		env, err := w.envRepo.GetByID(context.Background(), evt.EnvironmentID)
		if err != nil {
			slog.Error("failed to load environment for stop", "environment_id", evt.EnvironmentID, "error", err)
			return
		}
		evt.ContainerID = env.ContainerID
	}
	if evt.ContainerID == "" {
		slog.Info("environment has no container to stop", "environment_id", evt.EnvironmentID)
		_ = w.envRepo.UpdateState(context.Background(), evt.EnvironmentID, domain.EnvironmentStateStopped)
		return
	}

	if err := w.runtime.StopSandbox(context.Background(), evt.ContainerID); err != nil {
		slog.Error("failed to stop environment container", "environment_id", evt.EnvironmentID, "error", err)
		_ = msg.Nak()
		return
	}
	if err := w.envRepo.UpdateState(context.Background(), evt.EnvironmentID, domain.EnvironmentStateStopped); err != nil {
		slog.Error("failed to mark environment STOPPED", "environment_id", evt.EnvironmentID, "error", err)
	}
	slog.Info("environment stopped", "environment_id", evt.EnvironmentID, "container_id", evt.ContainerID)
}

func (w *SandboxWorker) handleEnvironmentDelete(msg *natsCore.Msg) {
	evt, ok := w.decodeLifecycle(msg, "delete")
	if !ok {
		return
	}
	_ = msg.Ack()

	if evt.ContainerID != "" {
		if err := w.runtime.DeleteSandbox(context.Background(), evt.ContainerID); err != nil {
			slog.Error("failed to delete environment container", "environment_id", evt.EnvironmentID, "error", err)
			_ = msg.Nak()
			return
		}
	}

	// Remove the checkout. The workspace may already be soft-deleted by the
	// api, so resolve the directory from the event when possible.
	if evt.WorkspaceID != uuid.Nil {
		if err := os.RemoveAll(filepath.Join(workspaceRoot(), evt.WorkspaceID.String())); err != nil {
			slog.Warn("failed to remove workspace directory", "workspace_id", evt.WorkspaceID, "error", err)
		}
	}

	if err := w.envRepo.Delete(context.Background(), evt.EnvironmentID); err != nil {
		slog.Error("failed to soft delete environment", "environment_id", evt.EnvironmentID, "error", err)
		return
	}
	slog.Info("environment deleted", "environment_id", evt.EnvironmentID)
}

func (w *SandboxWorker) decodeLifecycle(msg *natsCore.Msg, event string) (domain.EnvironmentLifecycleEvent, bool) {
	var evt domain.EnvironmentLifecycleEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		slog.Error("invalid environment event", "event", event, "error", err)
		_ = msg.Term()
		return evt, false
	}
	if evt.EnvironmentID == uuid.Nil {
		slog.Error("environment event missing environment_id", "event", event)
		_ = msg.Term()
		return evt, false
	}
	return evt, true
}

// environmentReapInterval is how often unprovisioned environments are picked
// up. Deliberately slower than the legacy sandbox poll because the api now
// persists the repository URL, so recovery does not need to be aggressive.
const environmentReapInterval = 30 * time.Second
