package config

import (
	"fmt"
	"os"
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
	PredictionAddr     string // gRPC address for Python prediction service
	ContainerdSocket   string
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

	return &Config{
		Mode:               mode,
		Env:                getEnv("ENV", "development"),
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		PredictionAddr:     getEnv("PREDICTION_ADDR", "localhost:50051"),
		ContainerdSocket:   os.Getenv("CONTAINERD_SOCK"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
