package worker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/usecase"
	"github.com/AyushCN/berth/pkg/crypto"
	natsCore "github.com/nats-io/nats.go"

	"github.com/google/uuid"
)

// Worker provisions and manages environment containers.
//
// It was previously SandboxWorker and also carried the legacy sandbox intake
// path (a berth.sandbox.create subscription, PopPendingSandbox polling and a
// cleanupExpired sweep over the sandboxes table). That table is gone; see
// migration 000009.
type Worker struct {
	userRepo      domain.UserRepository
	envRepo       domain.EnvironmentRepository
	workspaceRepo domain.WorkspaceRepository
	runtime       domain.ContainerRuntime
	natsClient    *natsInfra.Client
	dataCollector *usecase.DataCollector
	tokenBox      *crypto.Box
	wg            sync.WaitGroup
}

func NewWorker(
	userRepo domain.UserRepository,
	envRepo domain.EnvironmentRepository,
	workspaceRepo domain.WorkspaceRepository,
	runtime domain.ContainerRuntime,
	natsClient *natsInfra.Client,
	dataCollector *usecase.DataCollector,
	tokenBox *crypto.Box,
) *Worker {
	return &Worker{
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

// gitCacheRoot holds bare mirrors shared by every environment built from the
// same repository and branch.
func gitCacheRoot() string {
	if root := os.Getenv("GIT_CACHE_ROOT"); root != "" {
		return root
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "berth", "cache", "git")
}

// Start subscribes to the environment lifecycle subjects and then runs until
// ctx is cancelled.
//
// There is no separate "pending" database poll: reapPendingEnvironments
// already covers everything in CREATED, so a single timer is enough. The legacy
// version polled both sandboxes (every 10s) and swept expired sandboxes (every
// 60s); the expiry sweep deleted sandboxes rows for environments that were
// still live.
func (w *Worker) Start(ctx context.Context) {
	slog.Info("worker started")

	w.subscribeEnvironment(ctx)

	reaper := time.NewTicker(environmentReapInterval)
	defer reaper.Stop()

	slog.Info("worker ticker loop started", "environment_reap_interval", environmentReapInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down, waiting for active jobs to finish")
			w.wg.Wait()
			slog.Info("worker shutdown complete")
			return
		case <-reaper.C:
			w.reapPendingEnvironments(ctx)
		}
	}
}

// afterProvision runs the work that is common to every successfully
// provisioned container: training-data collection and the interactive shell
// bridge.
func (w *Worker) afterProvision(id uuid.UUID, res *provisionResult) {
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

// StartInteractiveShell starts a PTY-backed shell inside the container and
// bridges I/O over the NATS subjects sandbox.<id>.input / .output.
//
// The subject prefix is still "sandbox" because the websocket client in
// components/terminal.tsx subscribes to it; renaming both sides together is a
// separate change.
func (w *Worker) StartInteractiveShell(ctx context.Context, id, containerID string) {
	inSubject := "sandbox." + id + ".input"
	outSubject := "sandbox." + id + ".output"

	var shellMutex sync.Mutex
	var shellStarted bool
	var stdin io.WriteCloser

	sub, err := w.natsClient.Subscribe(inSubject, "", func(msg *natsCore.Msg) {
		shellMutex.Lock()
		defer shellMutex.Unlock()

		if !shellStarted {
			slog.Info("starting interactive shell lazily", "id", id)
			in, out, waitFunc, err := w.runtime.ExecPTY(ctx, containerID, []string{"/bin/sh"})
			if err != nil {
				slog.Error("failed to start interactive shell", "error", err)
				msg.Ack()
				return
			}
			stdin = in
			shellStarted = true

			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := out.Read(buf)
					if n > 0 {
						if pubErr := w.natsClient.Publish(outSubject, buf[:n]); pubErr != nil {
							slog.Error("failed to publish shell output", "error", pubErr)
						}
					}
					if err != nil {
						slog.Debug("shell output stream closed", "id", id)
						return
					}
				}
			}()

			go func() {
				if err := waitFunc(); err != nil {
					slog.Debug("shell exited", "id", id, "error", err)
				}
			}()
		}

		if stdin == nil {
			msg.Ack()
			return
		}
		if _, err := stdin.Write(msg.Data); err != nil {
			slog.Error("failed to write to shell stdin", "error", err)
		}
		msg.Ack()
	})
	if err != nil {
		slog.Error("failed to subscribe to shell input", "id", id, "error", err)
		return
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
}

// resolveGitToken returns the owner's decrypted GitHub token, or an empty
// string when no token is available (anonymous clone of a public repo).
//
// Decryption goes through the same crypto.Box the api uses to encrypt on
// login. An earlier version of this file hand-rolled a decryptor that expected
// hex(nonce || ciphertext) with a 32-byte nonce, which is not the format
// crypto.Encrypt ever produced; it failed on every token and the error was
// discarded, so the "authenticated clone" log line was a lie.
func (w *Worker) resolveGitToken(ctx context.Context, ownerID uuid.UUID) (string, error) {
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

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve tcp addr: %w", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to listen on tcp: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
