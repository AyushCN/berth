package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// NodeBuildStrategy handles Node.js project builds
type NodeBuildStrategy struct{}

func (s *NodeBuildStrategy) Name() string {
	return "node"
}

func (s *NodeBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.RuntimeProfile != nil && result.RuntimeProfile.Language == "node"
}

func (s *NodeBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
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

func (s *NodeBuildStrategy) selectBaseImage(profile *domain.RuntimeProfile) string {
	if profile.BaseImage != "" {
		return profile.BaseImage
	}
	return "docker.io/library/node:20-alpine"
}

func (s *NodeBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) string {
	// Check if there's a user-provided Dockerfile
	if profile.DockerfileSource == "USER_PROVIDED" && profile.DockerfileContent != "" {
		return profile.DockerfileContent
	}

	// Determine package manager
	packageManager := "npm"
	if result.Lockfiles != nil {
		for _, lf := range result.Lockfiles {
			if lf.LockfileType == "yarn" {
				packageManager = "yarn"
			} else if lf.LockfileType == "pnpm" {
				packageManager = "pnpm"
			} else if lf.LockfileType == "bun" {
				packageManager = "bun"
			}
		}
	}

	var installCmd, buildCmd, startCmd string

	switch packageManager {
	case "yarn":
		installCmd = "yarn install --frozen-lockfile"
		buildCmd = "yarn build"
		startCmd = "yarn start"
	case "pnpm":
		installCmd = "pnpm install --frozen-lockfile"
		buildCmd = "pnpm build"
		startCmd = "pnpm start"
	case "bun":
		installCmd = "bun install"
		buildCmd = "bun run build"
		startCmd = "bun run start"
	default:
		installCmd = "npm ci"
		buildCmd = "npm run build"
		startCmd = "npm start"
	}

	// Override with profile values if set
	if profile.InstallCmd != "" {
		installCmd = profile.InstallCmd
	}
	if profile.BuildCommand != "" {
		buildCmd = profile.BuildCommand
	}
	if profile.StartCmd != "" {
		startCmd = profile.StartCmd
	}

	// Check for specific framework
	framework := profile.Framework
	if framework == "next" {
		startCmd = "npm run start"
	} else if framework == "vite" {
		startCmd = "npm run dev"
	}

	dockerfile := fmt.Sprintf(`FROM %s

WORKDIR /app

COPY package*.json ./
RUN %s

COPY . .

RUN %s

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, buildCmd, profile.ExposedPort, startCmd)

	return dockerfile
}