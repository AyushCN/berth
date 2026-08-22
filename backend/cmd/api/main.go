package main

import (
	"context"
	"log/slog"
	"net/http"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AyushCN/berth/internal/config"
	berthhttp "github.com/AyushCN/berth/internal/delivery/http"
	"github.com/AyushCN/berth/internal/delivery/http/handler"
	"github.com/AyushCN/berth/internal/infrastructure/containerd"
	"github.com/AyushCN/berth/internal/infrastructure/db"
	"github.com/AyushCN/berth/internal/infrastructure/github"
	"github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/AyushCN/berth/internal/repository"
	"github.com/AyushCN/berth/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize infrastructure
	if err := db.Init(cfg.DatabaseURL); err != nil {
		slog.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := redis.Init(cfg.RedisURL); err != nil {
		slog.Error("failed to init redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	// containerd runtime
	runtime, err := containerd.NewRuntime(os.Getenv("CONTAINERD_SOCK"))
	if err != nil {
		slog.Error("failed to init container runtime", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

	// NATS
	natsClient, err := nats.NewClient("nats://localhost:4222")
	if err != nil {
		slog.Warn("nats not available, continuing without real-time sync", "error", err)
		natsClient = nil
	}
	if natsClient != nil {
		defer natsClient.Close()
	}

	// OPA
	// opaEngine, err := opa.NewEngine()
	// if err != nil {
	// 	slog.Error("failed to init opa", "error", err)
	// 	os.Exit(1)
	// }
	// _ = opaEngine

	// Repositories
	queries := repository.New(db.Pool())
	userRepo := repository.NewUserRepository(queries)
	sandboxRepo := repository.NewSandboxRepository(queries)

	// OAuth client
	oauthClient := github.NewOAuthClient(cfg.GithubClientID, cfg.GithubClientSecret, "http://localhost:3000/api/auth/github/callback")

	// Usecases
	authUC := usecase.NewAuthUsecase(userRepo, oauthClient, cfg.JWTSecret)
	sandboxUC := usecase.NewSandboxUsecase(sandboxRepo, runtime, nil) // predictor nil for now
	workspaceDir := os.Getenv("WORKSPACE_ROOT")
	if workspaceDir == "" {
		home, _ := os.UserHomeDir()
		workspaceDir = filepath.Join(home, ".local", "state", "berth", "workspaces")
	}
	_ = os.MkdirAll(workspaceDir, 0755)
	fileUC := usecase.NewFileUsecase(workspaceDir)

	// Handlers
	deps := &berthhttp.Dependencies{
		AuthHandler:    handler.NewAuthHandler(authUC),
		SandboxHandler: handler.NewSandboxHandler(sandboxUC),
		FileHandler:    handler.NewFileHandler(fileUC),
		WSHandler:      handler.NewWSHandler(natsClient),
	}

	// Router
	r := berthhttp.NewRouter(cfg, deps)

	srv := &stdhttp.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "port", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
