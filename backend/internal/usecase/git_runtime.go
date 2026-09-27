package usecase

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/infrastructure/docker"
)

// NewDockerRuntimeForGit creates a Docker runtime for git operations
// Returns nil if Docker is not available
func NewDockerRuntimeForGit(workspaceDir string) (domain.ContainerRuntime, error) {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	dockerNetwork := os.Getenv("DOCKER_NETWORK")
	if dockerNetwork == "" {
		dockerNetwork = "berth"
	}

	traefikDomain := os.Getenv("TRAEFIK_DOMAIN")
	if traefikDomain == "" {
		traefikDomain = "localhost"
	}

	runtime, err := docker.NewDockerRuntime(dockerHost, dockerNetwork, traefikDomain)
	if err != nil {
		slog.Warn("Docker runtime not available for git operations, using host filesystem fallback", "error", err)
		return nil, fmt.Errorf("docker not available: %w", err)
	}

	slog.Info("Docker runtime initialized for git operations")
	return runtime, nil
}