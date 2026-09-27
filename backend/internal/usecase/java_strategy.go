package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// JavaBuildStrategy handles Java project builds
type JavaBuildStrategy struct{}

func (s *JavaBuildStrategy) Name() string {
	return "java"
}

func (s *JavaBuildStrategy) Detect(result *analyzer.DetectionResult) bool {
	return result.RuntimeProfile != nil && result.RuntimeProfile.Language == "java"
}

func (s *JavaBuildStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
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

func (s *JavaBuildStrategy) selectBaseImage(profile *domain.RuntimeProfile) string {
	if profile.BaseImage != "" {
		return profile.BaseImage
	}
	return "docker.io/library/eclipse-temurin:21-jdk-alpine"
}

func (s *JavaBuildStrategy) generateDockerfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) string {
	// Check if there's a user-provided Dockerfile
	if profile.DockerfileSource == "USER_PROVIDED" && profile.DockerfileContent != "" {
		return profile.DockerfileContent
	}

	var installCmd, buildCmd, startCmd string

	// Determine build tool
	buildTool := "maven"
	if result != nil {
		for _, lf := range result.Lockfiles {
			if lf.LockfileType == "gradle" {
				buildTool = "gradle"
				break
			}
		}
	}

	if profile.InstallCmd != "" {
		installCmd = profile.InstallCmd
	} else if buildTool == "gradle" {
		installCmd = "./gradlew dependencies"
	} else {
		installCmd = "mvn dependency:go-offline"
	}

	if profile.BuildCommand != "" {
		buildCmd = profile.BuildCommand
	} else if buildTool == "gradle" {
		buildCmd = "./gradlew build -x test"
	} else {
		buildCmd = "mvn package -DskipTests"
	}

	if profile.StartCmd != "" {
		startCmd = profile.StartCmd
	} else {
		startCmd = "java -jar target/app.jar"
	}

	var dockerfile string
	if buildTool == "gradle" {
		dockerfile = fmt.Sprintf(`FROM %s

WORKDIR /app

COPY gradlew gradle.properties gradle ./gradle/
COPY build.gradle settings.gradle ./
RUN %s

COPY . .

RUN %s

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, buildCmd, profile.ExposedPort, startCmd)
	} else {
		dockerfile = fmt.Sprintf(`FROM %s

WORKDIR /app

COPY pom.xml ./
RUN %s

COPY src ./src
RUN %s

EXPOSE %d

CMD ["%s"]
`, s.selectBaseImage(profile), installCmd, buildCmd, profile.ExposedPort, startCmd)
	}

	return dockerfile
}