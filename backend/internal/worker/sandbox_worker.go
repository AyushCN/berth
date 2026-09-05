package worker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	natsCore "github.com/nats-io/nats.go"
)

type SandboxWorker struct {
	repo       domain.SandboxRepository
	runtime    domain.ContainerRuntime
	ml         domain.PredictionService
	natsClient *natsInfra.Client
	wg         sync.WaitGroup
}

func NewSandboxWorker(repo domain.SandboxRepository, runtime domain.ContainerRuntime, ml domain.PredictionService, natsClient *natsInfra.Client) *SandboxWorker {
	return &SandboxWorker{
		repo:       repo,
		runtime:    runtime,
		ml:         ml,
		natsClient: natsClient,
	}
}

func (w *SandboxWorker) Start(ctx context.Context) {
	slog.Info("sandbox worker started, connecting to NATS")

	if w.natsClient != nil {
		_, err := w.natsClient.Subscribe("berth.sandbox.create", "worker-group", func(msg *natsCore.Msg) {
			msg.Ack()
			// trigger the pending sandbox processor
			w.processPending(ctx)
		})
		if err != nil {
			slog.Error("failed to subscribe to NATS", "error", err)
		} else {
			slog.Info("subscribed to berth.sandbox.create events")
		}
	}

	// 10s fallback polling
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sandbox worker shutting down, waiting for active jobs to finish")
			w.wg.Wait()
			slog.Info("sandbox worker shutdown complete")
			return
		case <-ticker.C:
			w.processPending(ctx)
		}
	}
}

func (w *SandboxWorker) processPending(ctx context.Context) {
	w.wg.Add(1)
	defer w.wg.Done()

	dbPopStart := time.Now()
	sandbox, err := w.repo.PopPendingSandbox(ctx)
	if err != nil {
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
	home, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(home, ".local", "state", "berth", "workspaces", sandbox.ID.String())
	
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

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		slog.Error("failed to create workspace dir", "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	// Clone repository in background
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
			
			cloneArgs := []string{"clone", "--bare", "--depth", "1"}
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
			localCloneArgs := []string{"clone", "--local", "--shared", cacheDir, workspaceDir}
			localCmd := exec.CommandContext(bgCtx, "git", localCloneArgs...)
			if out, cmdErr := localCmd.CombinedOutput(); cmdErr != nil {
				slog.Error("git local clone failed", "error", cmdErr, "output", string(out))
				err = cmdErr
			}
		}

		cloneDuration = time.Since(cloneStart)
		cloneDone <- err
	}()

	// Wait for clone to finish before predicting
	if err := <-cloneDone; err != nil {
		slog.Error("worker failing due to clone error")
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	slog.Info("git clone completed", "sandbox_id", sandbox.ID, "duration", cloneDuration)

	// Predict runtime profile using the cloned repository
	predictStart := time.Now()
	profile, err := w.ml.Predict(bgCtx, sandbox.GitURL, sandbox.GitBranch, workspaceDir)
	if err != nil {
		slog.Warn("prediction failed, using fallback", "error", err)
		profile = &domain.RuntimeProfile{
			Language:    "node",
			BaseImage:   "docker.io/library/node:20-alpine",
			InstallCmd:  "npm install",
			StartCmd:    "npm run dev",
			ExposedPort: 3000,
			NeedsDB:     false,
			Confidence:  0.0,
		}
	}
	predictDuration := time.Since(predictStart)

	// Create container with bind mount and keep-alive command
	spec := domain.SandboxSpec{
		ID:           sandbox.ID,
		BaseImage:    profile.BaseImage,
		WorkDir:      "/mnt",
		WorkspaceDir: workspaceDir,
		Cmd:          []string{"sh", "-c", "while true; do sleep 1; done"},
		MemoryLimit:  512 * 1024 * 1024,
		CPULimit:     1000,
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

	if err := w.repo.UpdateContainerID(bgCtx, sandbox.ID, cid); err != nil {
		slog.Error("worker failed to update container id", "sandbox_id", sandbox.ID, "error", err)
	}

	startStart := time.Now()
	if err := w.runtime.StartSandbox(bgCtx, cid); err != nil {
		slog.Error("worker failed to start container", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}
	startDuration := time.Since(startStart)

	// Determine lockfile hash for dependency caching
	var lockfile string
	var cacheTarget string
	if _, err := os.Stat(filepath.Join(workspaceDir, "package-lock.json")); err == nil {
		lockfile = "package-lock.json"
		cacheTarget = "node_modules"
	} else if _, err := os.Stat(filepath.Join(workspaceDir, "go.sum")); err == nil {
		lockfile = "go.sum"
		cacheTarget = "vendor"
	}

	var depCachePath string
	var cacheHit bool
	if lockfile != "" {
		if b, err := os.ReadFile(filepath.Join(workspaceDir, lockfile)); err == nil {
			sum := sha256.Sum256(b)
			hash := fmt.Sprintf("%x", sum)[:16]
			depCachePath = filepath.Join(home, ".local", "state", "berth", "cache", "deps", profile.Language, hash)
			if _, err := os.Stat(depCachePath); err == nil {
				cacheHit = true
			}
		}
	}

	// Install dependencies inside container
	var installDuration time.Duration
	if profile.InstallCmd != "" {
		if cacheHit {
			slog.Info("dependency cache hit", "sandbox_id", sandbox.ID, "path", depCachePath)
			// Copy cached dependencies into workspace
			cmd := exec.Command("cp", "-a", depCachePath+"/.", filepath.Join(workspaceDir, cacheTarget)+"/")
			_ = os.MkdirAll(filepath.Join(workspaceDir, cacheTarget), 0755)
			if err := cmd.Run(); err != nil {
				slog.Warn("failed to restore dependency cache", "error", err)
			}
		} else {
			installStart := time.Now()
			installArgs := []string{"sh", "-c", "cd /mnt && " + profile.InstallCmd}
			if out, err := w.runtime.Exec(bgCtx, cid, installArgs); err != nil {
				slog.Error("dependency install failed", "sandbox_id", sandbox.ID, "error", err, "output", out)
			} else {
				installDuration = time.Since(installStart)
				slog.Info("dependencies installed", "sandbox_id", sandbox.ID, "output", out, "install_duration", installDuration)
				
				// Save to cache
				if depCachePath != "" {
					_ = os.MkdirAll(depCachePath, 0755)
					cmd := exec.Command("cp", "-a", filepath.Join(workspaceDir, cacheTarget)+"/.", depCachePath+"/")
					if err := cmd.Run(); err != nil {
						slog.Warn("failed to save dependency cache", "error", err)
					}
				}
			}
		}
	}

	if err := w.repo.UpdateState(bgCtx, sandbox.ID, domain.StateRunning); err != nil {
		slog.Error("worker failed to set RUNNING state", "sandbox_id", sandbox.ID, "error", err)
		return
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
			"predict": predictDuration.String(),
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
