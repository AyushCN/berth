package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// FallbackBuildStrategy handles unknown project builds
type FallbackBuildStrategy struct{}

func (s *FallbackBuildStrategy) Name() string {
	return "fallback"
}

func (s *FallbackBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	// Always matches as last resort
	return true
}

func (s *FallbackBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
	profile := result.RuntimeProfile
	if profile == nil {
		return nil, fmt.Errorf("no runtime profile")
	}

	plan := &domain.BuildPlan{
		ID:              uuid.New(),
		RuntimeProfileID: profile.ID,
		BaseImage:       "docker.io/library/alpine:latest",
		Dockerfile:      s.generateDockerfile(profile),
		BuildArgs:       map[string]string{},
		InstallCommand:  "",
		BuildCommand:    "",
		StartCommand:    "sh -c 'while true; do sleep 3600; done'",
		WorkingDir:      "/app",
		Port:            8080,
		Confidence:      0.1,
		Status:          domain.BuildPlanStatusReady,
	}

	return plan, nil
}

func (s *FallbackBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile) string {
	return `# Fallback Dockerfile - no recognized build system detected
FROM alpine:latest

WORKDIR /app

COPY . .

EXPOSE 8080

CMD ["sh", "-c", "while true; do sleep 3600; done"]
`
}