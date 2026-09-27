package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// GoBuildStrategy handles Go project builds
type GoBuildStrategy struct{}

func (s *GoBuildStrategy) Name() string {
	return "go"
}

func (s *GoBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.RuntimeProfile != nil && result.RuntimeProfile.Language == "go"
}

func (s *GoBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
	profile := result.RuntimeProfile
	if profile == nil {
		return nil, fmt.Errorf("no runtime profile")
	}

	plan := &domain.BuildPlan{
		ID:              uuid.New(),
		RuntimeProfileID: profile.ID,
		BaseImage:       s.selectBaseImage(profile),
		Dockerfile:      s.generateDockerfile(profile, result),
		BuildArgs:       map[string]string{},
		InstallCommand:  profile.InstallCmd,
		BuildCommand:    profile.BuildCommand,
		StartCommand:    profile.StartCmd,
		WorkingDir:      profile.WorkDir,
		Port:            profile.ExposedPort,
		Confidence:      0.9,
		Status:          domain.BuildPlanStatusReady,
	}

	return plan, nil
}

func (s *GoBuildStrategy) selectBaseImage(profile *domain.RuntimeProfile) string {
	if profile.BaseImage != "" {
		return profile.BaseImage
	}
	return "docker.io/library/golang:1.23-alpine"
}

func (s *GoBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) string {
	// Check if there's a user-provided Dockerfile
	if profile.DockerfileSource == "USER_PROVIDED" && profile.DockerfileContent != "" {
		return profile.DockerfileContent
	}

	var installCmd, buildCmd, startCmd string

	if profile.InstallCmd != "" {
		installCmd = profile.InstallCmd
	} else {
		installCmd = "go mod download"
	}

	if profile.BuildCommand != "" {
		buildCmd = profile.BuildCommand
	} else {
		buildCmd = "go build -o /app/bin/main ."
	}

	if profile.StartCmd != "" {
		startCmd = profile.StartCmd
	} else {
		startCmd = "/app/bin/main"
	}

	dockerfile := fmt.Sprintf(`FROM %s

WORKDIR /app

COPY go.mod go.sum ./
RUN %s

COPY . .

RUN %s

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, buildCmd, profile.ExposedPort, startCmd)

	return dockerfile
}