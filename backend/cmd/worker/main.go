package main

import (
	"context"
	"log/slog"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AyushCN/berth/internal/config"
	"github.com/AyushCN/berth/internal/infrastructure/docker"
	"github.com/AyushCN/berth/internal/infrastructure/db"
	natsInfra "github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/AyushCN/berth/internal/repository"
	"github.com/AyushCN/berth/internal/usecase"
	"github.com/AyushCN/berth/internal/worker"
)

func main() {
	logFile, err := os.OpenFile("/tmp/worker.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
		slog.SetDefault(logger)
	} else {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		slog.SetDefault(logger)
	}

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

	// Initialize Docker runtime
	runtime, err := docker.NewDockerRuntime(cfg.DockerHost, cfg.DockerNetwork, cfg.TraefikDomain)
	if err != nil {
		slog.Error("failed to init docker runtime", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

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

	// Prediction repositories
	modelRepo := repository.NewModelRepository(db.Pool())
	trainingDataRepo := repository.NewTrainingDataRepository(db.Pool())
	predictionRepo := repository.NewPredictionRepository(db.Pool())

	// Prediction service
	modelDir := os.Getenv("MODEL_DIR")
	if modelDir == "" {
		modelDir = "/tmp/berth/models"
	}
	_ = os.MkdirAll(modelDir, 0755)

	modelTrainer := usecase.NewModelTrainer(modelRepo, trainingDataRepo, modelDir)
	predictionService := usecase.NewPredictionService(modelTrainer, predictionRepo)
	dataCollector := usecase.NewDataCollector(trainingDataRepo, repository.NewBuildRepository(queries), repository.NewRuntimeProfileRepository(queries))

	// Inject data collector into sandbox worker
	sandboxWorker := worker.NewSandboxWorker(sandboxRepo, runtime, natsClient, dataCollector)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduled retraining
	go predictionService.ScheduledRetraining(ctx, 6*time.Hour)

	go sandboxWorker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("worker shutting down gracefully...")
	cancel()
}