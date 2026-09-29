package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/AyushCN/berth/pkg/crypto"
)

// Config holds all application configuration.
type Config struct {
	Mode               string // "api" or "worker"
	Env                string
	Port               string
	DatabaseURL        string
	RedisURL           string
	NatsURL            string
	JWTSecret          string
	GithubClientID     string
	GithubClientSecret string
	FrontendURL        string
	ContainerdSocket   string
	WorkspaceDir       string
	Runtime            string // "runsc" or "runc" (legacy)

	// Docker runtime settings
	DockerHost       string
	DockerNetwork    string
	TraefikDomain    string

	// ModelDir is where trained prediction artefacts are written.
	ModelDir string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Attempt to load .env file; ignore errors if it doesn't exist
	_ = godotenv.Load()

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

	// ENCRYPTION_KEY protects GitHub OAuth tokens at rest. Both the api
	// (encrypt on login) and the worker (decrypt to clone private repos)
	// need it, so it is required in every mode. Validated here so a missing
	// or malformed key fails with a clear message instead of a panic.
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required (32 raw bytes or 64 hex characters)")
	}
	if _, err := crypto.NewBox(encryptionKey); err != nil {
		return nil, fmt.Errorf("invalid ENCRYPTION_KEY: %w", err)
	}

	cfg := &Config{
		Mode:               mode,
		Env:                getEnv("ENV", "development"),
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		JWTSecret:          jwtSecret,
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		ContainerdSocket:   os.Getenv("CONTAINERD_SOCK"),
		Runtime:            getEnv("BERTH_RUNTIME", "runc"),
		DockerHost:       getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		DockerNetwork:    getEnv("DOCKER_NETWORK", "berth"),
		TraefikDomain:    getEnv("TRAEFIK_DOMAIN", ""),
		ModelDir:         getEnv("MODEL_DIR", "/tmp/berth/models"),
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
