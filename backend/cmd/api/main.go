package main

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/AyushCN/berth/internal/config"
	berthhttp "github.com/AyushCN/berth/internal/delivery/http"
	"github.com/AyushCN/berth/internal/delivery/http/handler"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/infrastructure/db"
	"github.com/AyushCN/berth/internal/infrastructure/github"
	"github.com/AyushCN/berth/internal/infrastructure/nats"
	"github.com/AyushCN/berth/internal/infrastructure/redis"
	"github.com/AyushCN/berth/internal/repository"
	"github.com/AyushCN/berth/internal/usecase"
	"github.com/AyushCN/berth/pkg/crypto"
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

	// Apply any pending schema migrations. Embedded in the binary, so a fresh
	// deployment provisions itself instead of needing a manual psql step.
	if _, err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("failed to migrate database schema", "error", err)
		os.Exit(1)
	}

	if cfg.Mode != "api" {
		slog.Error("api binary requires MODE=api", "mode", cfg.Mode)
		os.Exit(1)
	}

	// Initialize Redis
	if err := redis.Init(cfg.RedisURL); err != nil {
		slog.Error("failed to init redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	redisPubSub, err := redis.InitPubSub(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to init redis pubsub", "error", err)
		os.Exit(1)
	}
	defer redisPubSub.Close()

	// Repositories
	natsClient, err := nats.NewClient(cfg.NatsURL)
	if err != nil {
		slog.Warn("nats not available, continuing without real-time sync", "error", err)
		natsClient = nil
	}
	if natsClient != nil {
		defer natsClient.Close()
	}

	// Repositories
	queries := repository.New(db.Pool())
	userRepo := repository.NewUserRepository(queries)
	orgRepo := repository.NewOrganizationRepository(queries)
	projRepo := repository.NewProjectRepository(queries)
	shareLinkRepo := repository.NewShareLinkRepository(queries)
	workspaceRepo := repository.NewWorkspaceRepository(queries)
	workspaceMemberRepo := repository.NewWorkspaceMemberRepository(queries)
	changeRequestRepo := repository.NewChangeRequestRepository(queries)
	envRepo := repository.NewEnvironmentRepository(queries)

	// Prediction repositories
	modelRepo := repository.NewModelRepository(db.Pool())
	trainingDataRepo := repository.NewTrainingDataRepository(db.Pool())
	predictionRepo := repository.NewPredictionRepository(db.Pool())

	// OAuth client
	oauthClient := github.NewOAuthClient(cfg.GithubClientID, cfg.GithubClientSecret, cfg.FrontendURL+"/api/auth/github/callback")

	// Token encryption box. config.Load has already validated the key.
	tokenBox, err := crypto.NewBox(os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		slog.Error("failed to init token encryption", "error", err)
		os.Exit(1)
	}

	// Usecases
	orgUC := usecase.NewOrganizationUsecase(orgRepo)
	projUC := usecase.NewProjectUsecase(projRepo, orgRepo, workspaceRepo, workspaceMemberRepo)
	authUC := usecase.NewAuthUsecase(userRepo, oauthClient, cfg.JWTSecret, tokenBox, orgUC, projUC)
	envUC := usecase.NewEnvironmentUsecase(envRepo, workspaceRepo, projRepo, orgRepo, nil, natsClient)           // runtime nil in API mode
	shareLinkUC := usecase.NewShareLinkUsecase(shareLinkRepo, projRepo, workspaceRepo, workspaceMemberRepo, nil) // gitUC not yet initialized

	if cfg.Env != "production" {
		devUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		_, err := userRepo.GetByID(context.Background(), devUserID)
		if err != nil {
			devUser := &domain.User{
				ID:             devUserID,
				Email:          "dev@berth.local",
				Username:       "dev_user",
				GithubID:       "0",
				GithubUsername: "dev_user",
				AvatarURL:      "",
			}
			if createErr := userRepo.Create(context.Background(), devUser); createErr != nil {
				slog.Error("failed to create mock dev user", "error", createErr)
			} else {
				slog.Info("mock dev user initialized")
				org, orgErr := orgUC.Create(context.Background(), devUserID, "dev_user's Workspace")
				if orgErr == nil {
					desc := "Default project"
					_, _ = projUC.Create(context.Background(), devUserID, org.ID, "My Project", &desc, false)
				}
			}
		}
	}
	workspaceDir := os.Getenv("WORKSPACE_ROOT")
	if workspaceDir == "" {
		home, _ := os.UserHomeDir()
		workspaceDir = filepath.Join(home, ".local", "state", "berth", "workspaces")
	}
	_ = os.MkdirAll(workspaceDir, 0755)
	fileUC := usecase.NewFileUsecase(workspaceDir, envRepo, workspaceRepo, projRepo, envUC)

	// Initialize Docker runtime for Git operations in API mode
	gitRuntime, err := usecase.NewDockerRuntimeForGit(workspaceDir)
	if err != nil {
		slog.Warn("failed to init git runtime, git operations will use host filesystem", "error", err)
	}
	gitUC := usecase.NewGitUsecase(workspaceDir, userRepo, envRepo, workspaceRepo, projRepo, gitRuntime)
	changeRequestUC := usecase.NewChangeRequestUsecase(changeRequestRepo, workspaceRepo, workspaceMemberRepo, gitUC)
	shareLinkUC = usecase.NewShareLinkUsecase(shareLinkRepo, projRepo, workspaceRepo, workspaceMemberRepo, gitUC)

	// Activity tracker & warm pool.
	//
	// The api has no container runtime, so it cannot stop or start
	// containers itself; it asks the worker over NATS instead. The periodic
	// idle-suspend loop runs in the worker, which owns the Docker socket.
	// Running it in both processes would just race on the same rows.
	idleTimeout := 30 * time.Minute
	if cfg.Env == "production" {
		idleTimeout = 60 * time.Minute
	}
	activityTracker := usecase.NewActivityTracker(
		repository.NewEnvironmentRepository(queries),
		usecase.NewNATSContainerControl(natsClient),
		idleTimeout,
		5*time.Minute,
	)

	warmPool := usecase.NewWarmPool(
		repository.NewEnvironmentRepository(queries),
		nil, // unused: WarmPool stores the runtime but never calls it
		repository.NewRuntimeProfileRepository(queries),
		map[string]int{"node": 2, "python": 1, "go": 1},
		10*time.Minute,
	)
	go warmPool.Start(context.Background())

	// Prediction service
	modelDir := cfg.ModelDir
	_ = os.MkdirAll(modelDir, 0755)

	modelTrainer := usecase.NewModelTrainer(modelRepo, trainingDataRepo, modelDir)
	predictionService := usecase.NewPredictionService(modelTrainer, predictionRepo)

	// Start scheduled retraining (every 6 hours)
	go predictionService.ScheduledRetraining(context.Background(), 6*time.Hour)

	// Handlers
	deps := &berthhttp.Dependencies{
		AuthHandler:          handler.NewAuthHandler(authUC, cfg.FrontendURL),
		EnvironmentHandler:   handler.NewEnvironmentHandler(envUC, cfg.TraefikDomain),
		FileHandler:          handler.NewFileHandler(fileUC),
		WSHandler:            handler.NewWSHandler(redisPubSub, cfg.FrontendURL),
		GitHandler:           handler.NewGitHandler(gitUC),
		OrgHandler:           handler.NewOrganizationHandler(orgUC),
		ProjectHandler:       handler.NewProjectHandler(projUC),
		ShareLinkHandler:     handler.NewShareLinkHandler(shareLinkUC),
		ChangeRequestHandler: handler.NewChangeRequestHandler(changeRequestUC),
		ActivityHandler:      handler.NewActivityHandler(activityTracker, warmPool),
		PredictionHandler:    handler.NewPredictionHandler(predictionService),
	}

	// Router
	r := berthhttp.NewRouter(cfg, deps)

	srv := &stdhttp.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "port", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("received signal, shutting down gracefully...", "signal", sig)

	// Shutdown WebSocket hub
	if deps.WSHandler != nil {
		deps.WSHandler.Shutdown()
	}

	// Stop activity tracker and warm pool
	// Note: In a real implementation, we'd store these in deps and call Stop()
	// For now, context cancellation will stop them

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
