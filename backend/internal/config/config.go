package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration.
type Config struct {
	Env          string
	Port         string
	DatabaseURL  string
	RedisURL     string
	JWTSecret    string
	GithubClientID     string
	GithubClientSecret string
	PredictionAddr     string // gRPC address for Python prediction service
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
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
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return &Config{
		Env:                getEnv("ENV", "development"),
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		JWTSecret:          jwtSecret,
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		PredictionAddr:     getEnv("PREDICTION_ADDR", "localhost:50051"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
