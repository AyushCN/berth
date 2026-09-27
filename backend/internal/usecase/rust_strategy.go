package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// RustBuildStrategy handles Rust project builds
type RustBuildStrategy struct{}

func (s *RustBuildStrategy) Name() string {
	return "rust"
}

func (s *RustBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.RuntimeProfile != nil && result.RuntimeProfile.Language == "rust"
}

func (s *RustBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
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

func (s *RustBuildStrategy) selectBaseImage(profile *domain.RuntimeProfile) string {
	if profile.BaseImage != "" {
		return profile.BaseImage
	}
	return "docker.io/library/rust:1.80-slim"
}

func (s *RustBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) string {
	// Check if there's a user-provided Dockerfile
	if profile.DockerfileSource == "USER_PROVIDED" && profile.DockerfileContent != "" {
		return profile.DockerfileContent
	}

	var installCmd, buildCmd, startCmd string

	if profile.InstallCmd != "" {
		installCmd = profile.InstallCmd
	} else {
		installCmd = "cargo fetch"
	}

	if profile.BuildCommand != "" {
		buildCmd = profile.BuildCommand
	} else {
		buildCmd = "cargo build --release"
	}

	if profile.StartCmd != "" {
		startCmd = profile.StartCmd
	} else {
		startCmd = "./target/release/app"
	}

	dockerfile := fmt.Sprintf(`FROM %s

WORKDIR /app

COPY Cargo.toml Cargo.lock ./
RUN %s

COPY . .

RUN %s

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, buildCmd, profile.ExposedPort, startCmd)

	return dockerfile
}