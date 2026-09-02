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
	"github.com/AyushCN/berth/internal/infrastructure/ml"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/AyushCN/berth/internal/repository"
	"github.com/AyushCN/berth/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if os.Getenv("MODE") == "" {
		os.Setenv("MODE", "worker")
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load worker config", "error", err)
		os.Exit(1)
	}

	if cfg.Mode != "worker" {
		slog.Error("worker binary requires MODE=worker", "mode", cfg.Mode)
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

	runtime, err := containerd.NewRuntime(cfg.ContainerdSocket, cfg.Runtime)
	if err != nil {
		slog.Error("failed to init container runtime", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

	// Prediction service client
	predictor := ml.NewClient(cfg.PredictionAddr)

	var natsClient *natsInfra.Client
	if cfg.NatsURL != "" {
		nc, err := natsInfra.NewClient(cfg.NatsURL)
		if err != nil {
			slog.Error("failed to connect to NATS", "error", err)
		} else {
			natsClient = nc
			defer natsClient.Close()
		}
	}

	queries := repository.New(db.Pool())
	sandboxRepo := repository.NewSandboxRepository(queries)

	sandboxWorker := worker.NewSandboxWorker(sandboxRepo, runtime, predictor, natsClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sandboxWorker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("worker shutting down gracefully...")
	cancel()
}
