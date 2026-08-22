package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swordrookie/berth/internal/config"
	"github.com/swordrookie/berth/internal/infrastructure/db"
	"github.com/swordrookie/berth/internal/infrastructure/redis"
	"github.com/swordrookie/berth/internal/delivery/http/handler"
	"github.com/swordrookie/berth/internal/delivery/http/middleware"
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

	// Setup router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check (no auth)
	r.GET("/health", handler.HealthCheck)

	// API routes
	api := r.Group("/api")
	api.Use(middleware.RateLimit())
	{
		api.POST("/auth/github", handler.GithubLogin)
		api.GET("/auth/github/callback", handler.GithubCallback)

		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(cfg.JWTSecret))
		{
			authenticated.GET("/user/me", handler.GetMe)
			authenticated.GET("/environments", handler.ListEnvironments)
			authenticated.POST("/environments", handler.CreateEnvironment)
		}
	}

	srv := &http.Server{
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
