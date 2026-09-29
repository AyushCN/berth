package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/usecase"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	natsCore "github.com/nats-io/nats.go"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SandboxWorker struct {
	repo           domain.SandboxRepository
	runtime        domain.ContainerRuntime
	natsClient     *natsInfra.Client
	dataCollector  *usecase.DataCollector
	wg             sync.WaitGroup
}

func NewSandboxWorker(repo domain.SandboxRepository, runtime domain.ContainerRuntime, natsClient *natsInfra.Client, dataCollector *usecase.DataCollector) *SandboxWorker {
	return &SandboxWorker{
		repo:          repo,
		runtime:       runtime,
		natsClient:    natsClient,
		dataCollector: dataCollector,
	}
}

func workspaceRoot() string {
	if root := os.Getenv("WORKSPACE_ROOT"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "berth", "workspaces")
}

func (w *SandboxWorker) cleanupExpired(ctx context.Context) {
	expired, err := w.repo.ListExpiredSandboxes(ctx)
	if err != nil {
		slog.Error("failed to list expired sandboxes", "error", err)
		return
	}
	for _, sandbox := range expired {
		if sandbox.ContainerID != nil {
			cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err = w.runtime.DeleteSandbox(cleanupCtx, *sandbox.ContainerID)
			cancel()
			if err != nil {
				slog.Error("failed to remove expired sandbox container", "sandbox_id", sandbox.ID, "error", err)
				continue
			}
		}
		if err := os.RemoveAll(filepath.Join(workspaceRoot(), sandbox.ID.String())); err != nil {
			slog.Error("failed to remove expired sandbox workspace", "sandbox_id", sandbox.ID, "error", err)
			continue
		}
		if err := w.repo.Delete(ctx, sandbox.ID); err != nil {
			slog.Error("failed to delete expired sandbox record", "sandbox_id", sandbox.ID, "error", err)
			continue
		}
		slog.Info("expired sandbox cleaned up", "sandbox_id", sandbox.ID)
	}
}

func (w *SandboxWorker) Start(ctx context.Context) {
	slog.Info("sandbox worker started, connecting to NATS")

	if w.natsClient != nil {
		// Use non-durable subscription for create events since we have DB polling fallback
		// This avoids conflicts with existing durable consumers
		_, err := w.natsClient.Subscribe("berth.sandbox.create", "", func(msg *natsCore.Msg) {
			msg.Ack()
			// trigger the pending sandbox processor
			w.processPending(ctx)
		})
		if err != nil {
			slog.Warn("failed to subscribe to NATS create events, will rely on DB polling", "error", err)
		} else {
			slog.Info("subscribed to berth.sandbox.create events")
		}
		// Skip stop/delete subscriptions to avoid conflicts with existing durable consumers
		// These will be handled by API directly via Docker runtime
		slog.Info("skipping stop/delete NATS subscriptions (handled by API directly)")
	}

	// 10s fallback polling and one-minute expired sandbox cleanup.
	ticker := time.NewTicker(10 * time.Second)
	cleanupTicker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	defer cleanupTicker.Stop()

	slog.Info("worker ticker loop started", "poll_interval", "10s", "cleanup_interval", "1m")

	for {
		select {
		case <-ctx.Done():
			slog.Info("sandbox worker shutting down, waiting for active jobs to finish")
			w.wg.Wait()
			slog.Info("sandbox worker shutdown complete")
			return
		case <-ticker.C:
			slog.Info("ticker fired, checking for pending sandboxes")
			w.processPending(ctx)
		case <-cleanupTicker.C:
			w.cleanupExpired(ctx)
		}
	}
}

type sandboxLifecycleRequest struct {
	SandboxID   string `json:"sandbox_id"`
	ContainerID string `json:"container_id"`
}

func (w *SandboxWorker) handleStopRequest(msg *natsCore.Msg) {
	var req sandboxLifecycleRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		slog.Error("invalid sandbox stop request", "error", err)
		_ = msg.Term()
		return
	}
	if err := w.runtime.StopSandbox(context.Background(), req.ContainerID); err != nil {
		slog.Error("failed to stop sandbox requested by API", "sandbox_id", req.SandboxID, "error", err)
		_ = msg.Nak()
		return
	}
	if id, err := uuid.Parse(req.SandboxID); err == nil {
		_ = w.repo.UpdateState(context.Background(), id, domain.StateStopped)
	}
	_ = msg.Ack()
}

func (w *SandboxWorker) handleDeleteRequest(msg *natsCore.Msg) {
	var req sandboxLifecycleRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		slog.Error("invalid sandbox delete request", "error", err)
		_ = msg.Term()
		return
	}
	if req.ContainerID != "" {
		if err := w.runtime.DeleteSandbox(context.Background(), req.ContainerID); err != nil {
			slog.Error("failed to delete sandbox requested by API", "sandbox_id", req.SandboxID, "error", err)
			_ = msg.Nak()
			return
		}
	}
	if id, err := uuid.Parse(req.SandboxID); err == nil {
		if err := os.RemoveAll(filepath.Join(workspaceRoot(), id.String())); err != nil {
			slog.Error("failed to remove sandbox workspace", "sandbox_id", req.SandboxID, "error", err)
			_ = msg.Nak()
			return
		}
		_ = w.repo.Delete(context.Background(), id)
	}
	_ = msg.Ack()
}

func (w *SandboxWorker) processPending(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	dbPopStart := time.Now()
	sandbox, err := w.repo.PopPendingSandbox(ctx)
	if err != nil {
		slog.Info("PopPendingSandbox returned error", "error", err)
		return
	}
	if sandbox == nil {
		slog.Info("PopPendingSandbox returned nil sandbox")
		return
	}
	dbPopDuration := time.Since(dbPopStart)

	slog.Info("processing pending sandbox", "sandbox_id", sandbox.ID, "git_url", sandbox.GitURL, "db_pop_duration", dbPopDuration)

	// Validate Git URL before any filesystem or network operation
	if err := validateGitURL(sandbox.GitURL); err != nil {
		slog.Error("invalid git url", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// Validate branch name (prevent flag injection)
	if strings.HasPrefix(sandbox.GitBranch, "-") {
		slog.Error("invalid git branch", "sandbox_id", sandbox.ID, "branch", sandbox.GitBranch)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// Prepare workspace directory
	workspaceDir := filepath.Join(workspaceRoot(), sandbox.ID.String())

	success := false
	var cid string
	defer func() {
		if !success {
			_ = os.RemoveAll(workspaceDir)
			if cid != "" {
				// Fire and forget cleanup
				go func(id string) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					_ = w.runtime.DeleteSandbox(ctx, id)
				}(cid)
			}
		}
	}()

	// Clone repository in background
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cloneDone := make(chan error, 1)
	var cloneDuration time.Duration

	go func() {
		cloneStart := time.Now()

		branchPart := sandbox.GitBranch
		if branchPart == "" {
			branchPart = "HEAD"
		}
		hash := sha256.Sum256([]byte(sandbox.GitURL + "@" + branchPart))
		cacheKey := fmt.Sprintf("%x", hash)[:16]

		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".local", "state", "berth", "cache", "git", cacheKey)

		var err error
		if _, statErr := os.Stat(cacheDir); os.IsNotExist(statErr) {
			slog.Info("git cache miss, performing cold clone", "sandbox_id", sandbox.ID)
			os.MkdirAll(filepath.Dir(cacheDir), 0755)

			cloneArgs := []string{"clone", "--bare"}
			if sandbox.GitBranch != "" {
				cloneArgs = append(cloneArgs, "-b", sandbox.GitBranch)
			}
			cloneArgs = append(cloneArgs, sandbox.GitURL, cacheDir)

			cloneCmd := exec.CommandContext(bgCtx, "git", cloneArgs...)
			if out, cmdErr := cloneCmd.CombinedOutput(); cmdErr != nil {
				slog.Error("git cold clone failed", "error", cmdErr, "output", string(out))
				err = cmdErr
			}
		} else {
			slog.Info("git cache hit", "sandbox_id", sandbox.ID)
		}

		if err == nil {
			// Remove workspace dir if it exists (git clone creates it)
			_ = os.RemoveAll(workspaceDir)
			localCloneArgs := []string{"clone", "--local", "--shared", cacheDir, workspaceDir}
			slog.Info("running git local clone", "args", localCloneArgs)
			localCmd := exec.CommandContext(bgCtx, "git", localCloneArgs...)
			if out, cmdErr := localCmd.CombinedOutput(); cmdErr != nil {
				slog.Error("git local clone failed", "error", cmdErr, "output", string(out))
				err = cmdErr
			} else {
				slog.Info("git local clone output", "output", string(out))
				// Verify workspace has files
				entries, _ := os.ReadDir(workspaceDir)
				slog.Info("workspace contents after clone", "count", len(entries))
				for _, e := range entries {
					slog.Info("workspace entry", "name", e.Name(), "is_dir", e.IsDir())
				}
			}
		}

		cloneDuration = time.Since(cloneStart)
		cloneDone <- err
	}()

	// Wait for the repository clone before detecting its runtime.
	if err := <-cloneDone; err != nil {
		slog.Error("worker failing due to clone error")
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	slog.Info("git clone completed", "sandbox_id", sandbox.ID, "duration", cloneDuration)

	// Analyze the repository using the rule-based engine
	analyzeStart := time.Now()
	profile := analyzer.Analyze(workspaceDir)
	analyzeDuration := time.Since(analyzeStart)
	slog.Info("static analysis completed", "sandbox_id", sandbox.ID, "language", profile.Language, "duration", analyzeDuration)

	// Allocate a dynamic port for this sandbox
	allocatedPort, err := getFreePort()
	if err != nil {
		slog.Error("failed to allocate free port", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	slog.Info("allocated dynamic port", "sandbox_id", sandbox.ID, "port", allocatedPort)

	// Set PORT environment variable so the app listens on the exposed port
	env := map[string]string{
		"PORT": strconv.Itoa(allocatedPort),
	}

	// PHASE 1: Create container with sleep infinity to keep it alive for dependency installation
	keepaliveCmd := []string{"sleep", "infinity"}
	execProfile := domain.DefaultExecutionProfile()
	spec := domain.SandboxSpec{
		ID:               sandbox.ID,
		BaseImage:        profile.BaseImage,
		WorkDir:          "/workspace",
		WorkspaceDir:     workspaceDir,
		Cmd:              keepaliveCmd,
		MemoryLimit:      512 * 1024 * 1024,
		CPULimit:         1000,
		ExposedPort:      &allocatedPort,
		Env:              env,
		ExecutionProfile: execProfile,
		Labels: map[string]string{
			"berth.language": profile.Language,
		},
	}

	createStart := time.Now()
	var errCreate error
	cid, errCreate = w.runtime.CreateSandbox(bgCtx, spec)
	if errCreate != nil {
		slog.Error("worker failed to create container", "sandbox_id", sandbox.ID, "error", errCreate)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	createDuration := time.Since(createStart)

	startStart := time.Now()
	if err := w.runtime.StartSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to start keepalive container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	startDuration := time.Since(startStart)

	// PHASE 2: Install dependencies inside running container
	var installDuration time.Duration
	if profile.InstallCmd != "" {
		installStart := time.Now()
		installArgs := []string{"sh", "-c", "cd /workspace && " + profile.InstallCmd}
		slog.Info("executing install command", "sandbox_id", sandbox.ID, "cmd", installArgs)
		if out, err := w.runtime.Exec(bgCtx, cid, installArgs); err != nil {
			slog.Error("dependency install failed", "sandbox_id", sandbox.ID, "error", err, "output", out)
			_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
			return
		} else {
			installDuration = time.Since(installStart)
			slog.Info("dependencies installed successfully", "sandbox_id", sandbox.ID, "install_duration", installDuration)
		}
	}

	// PHASE 3: Stop container, commit to new image, recreate with watcher command
	stopStart := time.Now()
	if err := w.runtime.StopSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to stop container after install", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	stopDuration := time.Since(stopStart)

	// Commit container to new image with installed dependencies
	commitImage := fmt.Sprintf("berth-%s:latest", sandbox.ID.String()[:8])
	commitStart := time.Now()
	if err := w.runtime.CommitContainer(bgCtx, cid, commitImage); err != nil {
		slog.Error("worker failed to commit container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	commitDuration := time.Since(commitStart)
	slog.Info("committed container with installed deps", "sandbox_id", sandbox.ID, "image", commitImage)

	// Delete old container
	if err := w.runtime.DeleteSandbox(bgCtx, cid); err != nil {
		slog.Warn("failed to delete old container", "sandbox_id", sandbox.ID, "error", err)
	}

	// PHASE 4: Create new container from committed image with watcher command
	watcherCmd := WatcherCommand(profile)
	watcherSpec := domain.SandboxSpec{
		ID:               sandbox.ID,
		BaseImage:        commitImage,
		WorkDir:          "/workspace",
		WorkspaceDir:     workspaceDir,
		Cmd:              watcherCmd,
		MemoryLimit:      512 * 1024 * 1024,
		CPULimit:         1000,
		ExposedPort:      &allocatedPort,
		Env:              env,
		ExecutionProfile: execProfile,
		Labels: map[string]string{
			"berth.language": profile.Language,
		},
	}

	createStart2 := time.Now()
	cid, errCreate = w.runtime.CreateSandbox(bgCtx, watcherSpec)
	if errCreate != nil {
		slog.Error("worker failed to create watcher container", "sandbox_id", sandbox.ID, "error", errCreate)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	createDuration2 := time.Since(createStart2)

	startStart2 := time.Now()
	if err := w.runtime.StartSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to start watcher container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	startDuration2 := time.Since(startStart2)

	_ = stopDuration
	_ = commitDuration
	_ = createDuration2
	_ = startDuration2

	apiHost := os.Getenv("API_PUBLIC_HOST")
	if apiHost == "" {
		apiHost = "http://localhost:8080"
	}
	// With Traefik, preview URL is sandbox-id.domain
	publicURL := fmt.Sprintf("http://%s.%s/", sandbox.ID, strings.TrimPrefix(apiHost, "http://"))
	if strings.HasPrefix(apiHost, "http://") {
		host := strings.TrimPrefix(apiHost, "http://")
		publicURL = fmt.Sprintf("http://%s.%s/", sandbox.ID, host)
	} else {
		publicURL = fmt.Sprintf("%s/p/%s/", apiHost, sandbox.ID)
	}
	if err := w.repo.UpdateContainerAndURL(context.Background(), sandbox.ID, cid, publicURL, allocatedPort); err != nil {
		slog.Error("worker failed to update container id and url", "sandbox_id", sandbox.ID, "error", err)
	}

	if err := w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateRunning); err != nil {
		slog.Error("worker failed to set RUNNING state", "sandbox_id", sandbox.ID, "error", err)
		return
	}

	// Collect training data for prediction engine
	if w.dataCollector != nil {
		// Set WorkspaceID on profile for data collection
		profile.WorkspaceID = &sandbox.ID
		detectionResult := &analyzer.DetectionResult{
			RuntimeProfile: profile,
			Architecture:   profile.Architecture,
			Framework:      profile.Framework,
			CacheKey:       "",
			Lockfiles:      []analyzer.LockfileInfo{},
			EntryPoints:    []analyzer.EntryPoint{},
		}
		if err := w.dataCollector.CollectFromProfile(context.Background(), profile, detectionResult); err != nil {
			slog.Warn("failed to collect training data", "sandbox_id", sandbox.ID, "error", err)
		}
	}

	if w.natsClient != nil {
		go w.StartInteractiveShell(context.Background(), sandbox.ID.String(), cid)
	}

	totalDuration := time.Since(dbPopStart)
	success = true
	slog.Info("sandbox provisioned successfully",
		"sandbox_id", sandbox.ID,
		"container_id", cid,
		"language", profile.Language,
		"base_image", profile.BaseImage,
		"timing_metrics", map[string]any{
			"db_pop":  dbPopDuration.String(),
			"clone":   cloneDuration.String(),
			"create":  createDuration.String(),
			"start":   startDuration.String(),
			"install": installDuration.String(),
			"total":   totalDuration.String(),
		},
	)
}

func validateGitURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only https scheme allowed, got %s", u.Scheme)
	}
	if u.Host != "github.com" {
		return fmt.Errorf("only github.com allowed, got %s", u.Host)
	}
	if strings.Contains(u.Path, "..") {
		return fmt.Errorf("path traversal detected")
	}
	if strings.HasPrefix(u.Path, "-") {
		return fmt.Errorf("path looks like a flag")
	}
	return nil
}

// StartInteractiveShell starts a PTY-backed shell inside the container
// and bridges I/O to NATS subject sandbox.<id>.input and .output
func (w *SandboxWorker) StartInteractiveShell(ctx context.Context, sandboxID, containerID string) {
	inSubject := "sandbox." + sandboxID + ".input"
	outSubject := "sandbox." + sandboxID + ".output"

	var shellMutex sync.Mutex
	var shellStarted bool
	var stdin io.WriteCloser

	// Subscribe to inputs
	sub, err := w.natsClient.Subscribe(inSubject, "", func(msg *natsCore.Msg) {
		shellMutex.Lock()
		defer shellMutex.Unlock()

		if !shellStarted {
			slog.Info("starting interactive shell lazily", "sandbox_id", sandboxID)
			in, out, waitFunc, err := w.runtime.ExecPTY(ctx, containerID, []string{"/bin/sh"})
			if err != nil {
				slog.Error("failed to start interactive shell", "error", err)
				msg.Ack()
				return
			}
			stdin = in
			shellStarted = true

			// Bridge stdout to NATS
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := out.Read(buf)
					if n > 0 {
						_ = w.natsClient.Publish(outSubject, buf[:n])
					}
					if err != nil {
						slog.Info("shell stdout closed", "sandbox_id", sandboxID)
						break
					}
				}
			}()

			// Wait for shell exit
			go func() {
				if err := waitFunc(); err != nil {
					slog.Warn("interactive shell exited with error", "error", err)
				} else {
					slog.Info("interactive shell exited cleanly", "sandbox_id", sandboxID)
				}

				shellMutex.Lock()
				shellStarted = false
				if stdin != nil {
					stdin.Close()
				}
				shellMutex.Unlock()
			}()
		}

		// Forward input to shell
		if shellStarted && stdin != nil {
			_, err := stdin.Write(msg.Data)
			if err != nil {
				slog.Error("failed to write to shell stdin", "error", err)
			}
		}

		msg.Ack()
	})

	if err != nil {
		slog.Error("failed to subscribe to terminal events", "error", err)
		return
	}

	// Keep subscription alive until context is canceled
	<-ctx.Done()
	sub.Unsubscribe()

	shellMutex.Lock()
	if shellStarted && stdin != nil {
		stdin.Close()
	}
	shellMutex.Unlock()
}

// getFreePort asks the kernel for a free open port that is ready to use
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
