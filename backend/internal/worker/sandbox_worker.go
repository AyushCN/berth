package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/usecase"
	natsCore "github.com/nats-io/nats.go"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/AyushCN/berth/pkg/crypto"
)

type SandboxWorker struct {
	repo          domain.SandboxRepository
	userRepo      domain.UserRepository
	envRepo       domain.EnvironmentRepository
	workspaceRepo domain.WorkspaceRepository
	runtime       domain.ContainerRuntime
	natsClient    *natsInfra.Client
	dataCollector *usecase.DataCollector
	tokenBox      *crypto.Box
	wg            sync.WaitGroup
}

func NewSandboxWorker(
	repo domain.SandboxRepository,
	userRepo domain.UserRepository,
	envRepo domain.EnvironmentRepository,
	workspaceRepo domain.WorkspaceRepository,
	runtime domain.ContainerRuntime,
	natsClient *natsInfra.Client,
	dataCollector *usecase.DataCollector,
	tokenBox *crypto.Box,
) *SandboxWorker {
	return &SandboxWorker{
		repo:          repo,
		userRepo:      userRepo,
		envRepo:       envRepo,
		workspaceRepo: workspaceRepo,
		runtime:       runtime,
		natsClient:    natsClient,
		dataCollector: dataCollector,
		tokenBox:      tokenBox,
	}
}

func workspaceRoot() string {
	if root := os.Getenv("WORKSPACE_ROOT"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "berth", "workspaces")
}

// gitCacheRoot holds bare mirrors shared by every sandbox and environment
// built from the same repository and branch.
func gitCacheRoot() string {
	if root := os.Getenv("GIT_CACHE_ROOT"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "berth", "cache", "git")
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

	// Environments are the live model: the api publishes create/stop/delete
	// and nothing else consumes them, so the worker must subscribe here.
	w.subscribeEnvironment(ctx)

	// 10s fallback polling and one-minute expired sandbox cleanup.
	ticker := time.NewTicker(10 * time.Second)
	cleanupTicker := time.NewTicker(time.Minute)
	envReaper := time.NewTicker(environmentReapInterval)
	defer ticker.Stop()
	defer cleanupTicker.Stop()
	defer envReaper.Stop()

	slog.Info("worker ticker loop started",
		"poll_interval", "10s", "cleanup_interval", "1m", "environment_reap_interval", environmentReapInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("sandbox worker shutting down, waiting for active jobs to finish")
			w.wg.Wait()
			slog.Info("sandbox worker shutdown complete")
			return
		case <-ticker.C:
			w.processPending(ctx)
		case <-cleanupTicker.C:
			w.cleanupExpired(ctx)
		case <-envReaper.C:
			w.reapPendingEnvironments(ctx)
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

	popStart := time.Now()
	sandbox, err := w.repo.PopPendingSandbox(ctx)
	if err != nil {
		slog.Info("PopPendingSandbox returned error", "error", err)
		return
	}
	if sandbox == nil {
		return
	}

	slog.Info("processing pending sandbox", "sandbox_id", sandbox.ID, "git_url", sandbox.GitURL)

	if err := w.repo.UpdateState(ctx, sandbox.ID, domain.StateBuilding); err != nil {
		slog.Error("failed to update state to BUILDING", "sandbox_id", sandbox.ID, "error", err)
	}

	if err := os.MkdirAll(workspaceRoot(), 0755); err != nil {
		slog.Error("failed to create workspace root", "error", err)
		_ = w.repo.UpdateState(ctx, sandbox.ID, domain.StateFailed)
		return
	}

	res, err := w.provision(ctx, provisionRequest{
		ID:           sandbox.ID,
		WorkspaceDir: filepath.Join(workspaceRoot(), sandbox.ID.String()),
		WorkDir:      containerWorkspaceDir,
		GitURL:       sandbox.GitURL,
		GitBranch:    sandbox.GitBranch,
		OwnerID:      sandbox.OwnerID,
		MemoryLimit:  512 * 1024 * 1024,
		CPULimit:     defaultCPUMilliCores,
	})
	if err != nil {
		slog.Error("failed to provision sandbox", "sandbox_id", sandbox.ID, "error", err)
		_ = w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateFailed)
		return
	}

	if err := w.repo.UpdateContainerAndURL(context.Background(), sandbox.ID, res.ContainerID, res.PublicURL, res.Port); err != nil {
		slog.Error("failed to update container id and url", "sandbox_id", sandbox.ID, "error", err)
	}
	if err := w.repo.UpdateState(context.Background(), sandbox.ID, domain.StateRunning); err != nil {
		slog.Error("worker failed to set RUNNING state", "sandbox_id", sandbox.ID, "error", err)
		return
	}

	w.afterProvision(sandbox.ID, res)

	slog.Info("sandbox provisioned successfully",
		"sandbox_id", sandbox.ID,
		"container_id", res.ContainerID,
		"language", res.Profile.Language,
		"base_image", res.Profile.BaseImage,
		"timing_metrics", res.Timings,
		"total", time.Since(popStart).String(),
	)
}

// afterProvision runs the work that is common to every successfully
// provisioned container regardless of which model created it: training-data
// collection and the interactive shell bridge.
func (w *SandboxWorker) afterProvision(id uuid.UUID, res *provisionResult) {
	if w.dataCollector != nil && res.Profile != nil {
		profile := res.Profile
		profile.WorkspaceID = &id
		detection := &analyzer.DetectionResult{
			RuntimeProfile: profile,
			Architecture:   profile.Architecture,
			Framework:      profile.Framework,
			CacheKey:       "",
			Lockfiles:      []analyzer.LockfileInfo{},
			EntryPoints:    []analyzer.EntryPoint{},
		}
		if err := w.dataCollector.CollectFromProfile(context.Background(), profile, detection); err != nil {
			slog.Warn("failed to collect training data", "id", id, "error", err)
		}
	}

	if w.natsClient != nil {
		go w.StartInteractiveShell(context.Background(), id.String(), res.ContainerID)
	}
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

// resolveGitToken returns the owner's decrypted GitHub token, or an empty
// string when no token is available (anonymous clone of a public repo).
//
// Decryption goes through the same crypto.Box the api uses to encrypt on
// login. An earlier version of this file hand-rolled a decryptor that
// expected hex(nonce || ciphertext) with a 32-byte nonce, which is not the
// format crypto.Encrypt ever produced; it failed on every token and the
// error was discarded, so the "authenticated clone" log line was a lie.
func (w *SandboxWorker) resolveGitToken(ctx context.Context, ownerID uuid.UUID) (string, error) {
	if w.tokenBox == nil || ownerID == uuid.Nil {
		return "", nil
	}
	user, err := w.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return "", fmt.Errorf("failed to load owner %s: %w", ownerID, err)
	}
	if user == nil || user.GithubTokenEncrypted == "" {
		return "", nil
	}
	return w.tokenBox.Decrypt(user.GithubTokenEncrypted)
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
