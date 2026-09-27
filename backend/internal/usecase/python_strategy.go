package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// PythonBuildStrategy handles Python project builds
type PythonBuildStrategy struct{}

func (s *PythonBuildStrategy) Name() string {
	return "python"
}

func (s *PythonBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.RuntimeProfile != nil && result.RuntimeProfile.Language == "python"
}

func (s *PythonBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
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

func (s *PythonBuildStrategy) selectBaseImage(profile *domain.RuntimeProfile) string {
	if profile.BaseImage != "" {
		return profile.BaseImage
	}
	return "docker.io/library/python:3.11-slim"
}

func (s *PythonBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) string {
	var dockerfile string

	// Check if there's a user-provided Dockerfile
	if profile.DockerfileSource == "USER_PROVIDED" && profile.DockerfileContent != "" {
		return profile.DockerfileContent
	}

	var installCmd, startCmd string

	if profile.InstallCmd != "" {
		installCmd = profile.InstallCmd
	} else {
		installCmd = "pip install -r requirements.txt"
	}

	if profile.StartCmd != "" {
		startCmd = profile.StartCmd
	} else {
		startCmd = "python main.py"
	}

	dockerfile = fmt.Sprintf(`FROM %s

WORKDIR /app

COPY requirements.txt .
RUN %s

COPY . .

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, profile.ExposedPort, startCmd)

	return dockerfile
}