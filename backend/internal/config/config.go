package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all application configuration.
type Config struct {
	Mode               string // "api" or "worker"
	Env                string
	Port               string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	GithubClientID     string
	GithubClientSecret string
	FrontendURL        string
	PredictionAddr     string // gRPC address for Python prediction service
	ContainerdSocket   string
	WorkspaceDir       string
	Runtime            string // "runsc" or "runc"
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "api"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://berth:berth@localhost:5432/berth?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if mode == "api" && jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required in api mode")
	}

	cfg := &Config{
		Mode:               mode,
		Env:                getEnv("ENV", "development"),
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		PredictionAddr:     getEnv("PREDICTION_ADDR", "http://localhost:50052"),
		ContainerdSocket:   os.Getenv("CONTAINERD_SOCK"),
		Runtime:            getEnv("BERTH_RUNTIME", "runc"),
	}

	workspaceDir := os.Getenv("WORKSPACE_ROOT")
	if workspaceDir == "" {
		home, _ := os.UserHomeDir()
		workspaceDir = filepath.Join(home, ".local", "state", "berth", "workspaces")
	}
	cfg.WorkspaceDir = workspaceDir

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
