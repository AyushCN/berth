package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AyushCN/berth/internal/config"
	"github.com/AyushCN/berth/internal/infrastructure/containerd"
	"github.com/AyushCN/berth/internal/infrastructure/db"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/AyushCN/berth/internal/repository"
	"github.com/AyushCN/berth/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Set mode explicitly so Load() handles validation
	os.Setenv("MODE", "worker")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load worker config", "error", err)
		os.Exit(1)
	}

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

	if cfg.ContainerdSocket == "" {
		slog.Error("CONTAINERD_SOCK is required for worker mode")
		os.Exit(1)
	}

	runtime, err := containerd.NewRuntime(cfg.ContainerdSocket)
	if err != nil {
		slog.Error("failed to init container runtime", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

	queries := repository.New(db.Pool())
	sandboxRepo := repository.NewSandboxRepository(queries)

	sandboxWorker := worker.NewSandboxWorker(sandboxRepo, runtime)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sandboxWorker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("worker shutting down gracefully...")
	cancel() // Stop the worker polling loops
}
