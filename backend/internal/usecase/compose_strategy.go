package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// ComposeBuildStrategy handles Docker Compose project builds
type ComposeBuildStrategy struct{}

func (s *ComposeBuildStrategy) Name() string {
	return "compose"
}

func (s *ComposeBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.DockerCompose != nil && len(result.DockerCompose.Services) > 0
}

func (s *ComposeBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
	profile := result.RuntimeProfile
	if profile == nil {
		return nil, fmt.Errorf("no runtime profile")
	}

	// For compose, we generate a build plan for the main service
	// In practice, this would create a multi-service build
	mainService := s.findMainService(result.DockerCompose)
	if mainService == nil {
		// No main service found, use first service
		for _, svc := range result.DockerCompose.Services {
			mainService = &svc
			break
		}
	}

	plan := &domain.BuildPlan{
		ID:              uuid.New(),
		RuntimeProfileID: profile.ID,
		BaseImage:       s.getBaseImage(mainService),
		Dockerfile:      s.generateDockerfile(mainService, result),
		BuildArgs:       s.getBuildArgs(mainService),
		InstallCommand:  "",
		BuildCommand:    "",
		StartCommand:    s.getStartCommand(mainService),
		WorkingDir:      "/app",
		Port:            s.getPort(mainService),
		Confidence:      0.85,
		Status:          domain.BuildPlanStatusReady,
	}

	return plan, nil
}

func (s *ComposeBuildStrategy) findMainService(compose *analyzer.DockerComposeConfig) *analyzer.ComposeService {
	// Look for a service with "web", "app", "api", "main" in name
	for name, svc := range compose.Services {
		if name == "web" || name == "app" || name == "api" || name == "main" || name == "server" {
			return &svc
		}
	}
	return nil
}

func (s *ComposeBuildStrategy) getBaseImage(svc *analyzer.ComposeService) string {
	if svc != nil && svc.Image != "" {
		return svc.Image
	}
	return "docker.io/library/alpine:latest"
}

func (s *ComposeBuildStrategy) getStartCommand(svc *analyzer.ComposeService) string {
	if svc != nil && svc.Command != "" {
		return svc.Command
	}
	if svc != nil && len(svc.Entrypoint) > 0 {
		return svc.Entrypoint[0]
	}
	return "sh -c 'while true; do sleep 3600; done'"
}

func (s *ComposeBuildStrategy) getPort(svc *analyzer.ComposeService) int {
	if svc != nil && len(svc.Ports) > 0 {
		// Parse port from "host:container" format
		portStr := svc.Ports[0]
		// Simple parsing - extract container port
		for i := len(portStr) - 1; i >= 0; i-- {
			if portStr[i] == ':' {
				portStr = portStr[i+1:]
				break
			}
		}
		// Try to parse as int
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		if port > 0 {
			return port
		}
	}
	return 8080
}

func (s *ComposeBuildStrategy) getBuildArgs(svc *analyzer.ComposeService) map[string]string {
	args := map[string]string{}
	if svc != nil && svc.Build != nil {
		if buildCtx, ok := svc.Build["context"].(string); ok {
			args["BUILD_CONTEXT"] = buildCtx
		}
		if dockerfile, ok := svc.Build["dockerfile"].(string); ok {
			args["DOCKERFILE"] = dockerfile
		}
	}
	return args
}

func (s *ComposeBuildStrategy) generateDockerfile(svc *analyzer.ComposeService, result *analyzer.DetectionResult) string {
	// If service has build config with dockerfile, use it
	if svc != nil && svc.Build != nil {
		if dockerfile, ok := svc.Build["dockerfile"].(string); ok {
			// Return the user-provided dockerfile path
			return fmt.Sprintf("# Using docker-compose build config\n# Dockerfile: %s", dockerfile)
		}
		if context, ok := svc.Build["context"].(string); ok {
			return fmt.Sprintf("# Using docker-compose build config\n# Build context: %s", context)
		}
	}

	// Fallback - generate a basic dockerfile
	return fmt.Sprintf(`# Generated from docker-compose
FROM %s

WORKDIR /app

COPY . .

EXPOSE %d

CMD ["%s"]
`, s.getBaseImage(svc), s.getPort(svc), s.getStartCommand(svc))
}