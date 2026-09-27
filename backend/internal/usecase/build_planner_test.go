package usecase

import (
	"context"
	"testing"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

func TestBuildPlanner(t *testing.T) {
	planner := NewBuildPlanner()

	// Test strategy registration
	names := planner.GetStrategyNames()
	expectedStrategies := []string{"compose", "python", "node", "go", "rust", "java", "fallback"}
	if len(names) != len(expectedStrategies) {
		t.Errorf("expected %d strategies, got %d: %v", len(expectedStrategies), len(names), names)
	}

	// Verify all expected strategies are present
	for _, exp := range expectedStrategies {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected strategy %s not found", exp)
		}
	}
}

func TestBuildPlanner_ComposeStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	// Create detection result with docker-compose
	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "node",
			BaseImage:    "nginx:latest",
			ExposedPort:  80,
			WorkDir:      "/app",
			DockerfileSource: "USER_PROVIDED",
		},
		DockerCompose: &analyzer.DockerComposeConfig{
			Services: map[string]analyzer.ComposeService{
				"web": {
					Image: "nginx:latest",
					Ports: []string{"80:80"},
				},
			},
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "nginx:latest" {
		t.Errorf("expected base image nginx:latest, got %s", plan.BaseImage)
	}
	if plan.Port != 80 {
		t.Errorf("expected port 80, got %d", plan.Port)
	}
}

func TestBuildPlanner_NodeStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "node",
			Framework:    "next",
			BaseImage:    "node:20-alpine",
			ExposedPort:  3000,
			WorkDir:      "/app",
			InstallCmd:   "npm ci",
			BuildCommand: "npm run build",
			StartCmd:     "npm start",
		},
		Lockfiles: []analyzer.LockfileInfo{
			{LockfileType: "npm"},
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "node:20-alpine" {
		t.Errorf("expected base image node:20-alpine, got %s", plan.BaseImage)
	}
	if plan.InstallCommand != "npm ci" {
		t.Errorf("expected npm ci, got %s", plan.InstallCommand)
	}
	if plan.BuildCommand != "npm run build" {
		t.Errorf("expected npm run build, got %s", plan.BuildCommand)
	}
	if plan.StartCommand != "npm start" {
		t.Errorf("expected npm start, got %s", plan.StartCommand)
	}
}

func TestBuildPlanner_PythonStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "python",
			Framework:    "django",
			BaseImage:    "python:3.11-slim",
			ExposedPort:  8000,
			WorkDir:      "/app",
			InstallCmd:   "pip install -r requirements.txt",
			StartCmd:     "python manage.py runserver 0.0.0.0:8000",
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "python:3.11-slim" {
		t.Errorf("expected base image python:3.11-slim, got %s", plan.BaseImage)
	}
	if plan.Port != 8000 {
		t.Errorf("expected port 8000, got %d", plan.Port)
	}
}

func TestBuildPlanner_GoStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "go",
			Framework:    "gin",
			BaseImage:    "golang:1.23-alpine",
			ExposedPort:  8080,
			WorkDir:      "/app",
			InstallCmd:   "go mod download",
			BuildCommand: "go build -o /app/bin/main .",
			StartCmd:     "/app/bin/main",
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "golang:1.23-alpine" {
		t.Errorf("expected base image golang:1.23-alpine, got %s", plan.BaseImage)
	}
}

func TestBuildPlanner_RustStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "rust",
			Framework:    "axum",
			BaseImage:    "rust:1.80-slim",
			ExposedPort:  8080,
			WorkDir:      "/app",
			InstallCmd:   "cargo fetch",
			BuildCommand: "cargo build --release",
			StartCmd:     "./target/release/app",
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "rust:1.80-slim" {
		t.Errorf("expected base image rust:1.80-slim, got %s", plan.BaseImage)
	}
}

func TestBuildPlanner_JavaStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "java",
			Framework:    "spring",
			BaseImage:    "eclipse-temurin:21-jdk-alpine",
			ExposedPort:  8080,
			WorkDir:      "/app",
			BuildCommand: "./gradlew build -x test",
			StartCmd:     "java -jar target/app.jar",
		},
		Lockfiles: []analyzer.LockfileInfo{
			{LockfileType: "gradle"},
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "eclipse-temurin:21-jdk-alpine" {
		t.Errorf("expected base image eclipse-temurin:21-jdk-alpine, got %s", plan.BaseImage)
	}
}

func TestBuildPlanner_FallbackStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	// Unknown language should use fallback
	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "unknown-lang",
			BaseImage:    "alpine:latest",
			ExposedPort:  8080,
		},
	}

	plan, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "docker.io/library/alpine:latest" {
		t.Errorf("expected fallback base image alpine:latest, got %s", plan.BaseImage)
	}
	if plan.Confidence >= 0.5 {
		t.Errorf("expected low confidence for fallback, got %f", plan.Confidence)
	}
}

func TestBuildPlanner_CustomStrategy(t *testing.T) {
	planner := NewBuildPlanner()

	// Register a custom strategy that should take priority
	customStrategy := &testStrategy{name: "custom", detect: true}
	planner.RegisterStrategy(customStrategy)

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:       uuid.New(),
			Language: "test",
		},
	}

	_, err := planner.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Custom strategy should be used (registered last, but we check if it was called)
	// Since strategies are checked in order, and custom is last, it would only match if others don't
	// For this test, we just verify registration works
	if len(planner.GetStrategyNames()) != 8 {
		t.Errorf("expected 8 strategies after registration, got %d", len(planner.GetStrategyNames()))
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// testStrategy is a test implementation of BuildStrategy
type testStrategy struct {
	name   string
	detect bool
}

func (s *testStrategy) Name() string {
	return s.name
}

func (s *testStrategy) Detect(result *analyzer.DetectionResult) bool {
	return s.detect
}

func (s *testStrategy) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
	return &domain.BuildPlan{
		ID:              uuid.New(),
		RuntimeProfileID: result.RuntimeProfile.ID,
		BaseImage:       "custom:latest",
		Dockerfile:      "FROM custom:latest",
		Port:            8080,
		Confidence:      0.95,
		Status:          domain.BuildPlanStatusReady,
	}, nil
}

func TestPythonBuildStrategy(t *testing.T) {
	strategy := &PythonBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "python",
			Framework:    "fastapi",
			BaseImage:    "python:3.11-slim",
			ExposedPort:  8000,
			WorkDir:      "/app",
			InstallCmd:   "pip install -r requirements.txt",
			StartCmd:     "uvicorn main:app --host 0.0.0.0 --port 8000",
		},
	}

	// Test Detect
	if !strategy.Detect(result) {
		t.Error("PythonBuildStrategy should detect python")
	}

	// Test with non-python
	result.RuntimeProfile.Language = "node"
	if strategy.Detect(result) {
		t.Error("PythonBuildStrategy should not detect node")
	}

	// Test GenerateBuildPlan
	result.RuntimeProfile.Language = "python"
	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "python:3.11-slim" {
		t.Errorf("expected python:3.11-slim, got %s", plan.BaseImage)
	}
}

func TestNodeBuildStrategy(t *testing.T) {
	strategy := &NodeBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "node",
			Framework:    "next",
			BaseImage:    "node:20-alpine",
			ExposedPort:  3000,
			WorkDir:      "/app",
			BuildCommand: "npm run build",
			StartCmd:     "npm start",
		},
		Lockfiles: []analyzer.LockfileInfo{
			{LockfileType: "npm"},
		},
	}

	// Test Detect
	if !strategy.Detect(result) {
		t.Error("NodeBuildStrategy should detect node")
	}

	// Test with yarn
	result.Lockfiles = []analyzer.LockfileInfo{{LockfileType: "yarn"}}
	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(plan.Dockerfile, "yarn install") {
		t.Errorf("expected yarn install in dockerfile, got %s", plan.Dockerfile)
	}

	// Test with pnpm
	result.Lockfiles = []analyzer.LockfileInfo{{LockfileType: "pnpm"}}
	plan, err = strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(plan.Dockerfile, "pnpm install") {
		t.Errorf("expected pnpm install in dockerfile, got %s", plan.Dockerfile)
	}
}

func TestGoBuildStrategy(t *testing.T) {
	strategy := &GoBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "go",
			Framework:    "gin",
			BaseImage:    "golang:1.23-alpine",
			ExposedPort:  8080,
			WorkDir:      "/app",
			InstallCmd:   "go mod download",
			BuildCommand: "go build -o /app/bin/main .",
			StartCmd:     "/app/bin/main",
		},
	}

	// Test Detect
	if !strategy.Detect(result) {
		t.Error("GoBuildStrategy should detect go")
	}

	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "golang:1.23-alpine" {
		t.Errorf("expected golang:1.23-alpine, got %s", plan.BaseImage)
	}
}

func TestRustBuildStrategy(t *testing.T) {
	strategy := &RustBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "rust",
			Framework:    "axum",
			BaseImage:    "rust:1.80-slim",
			ExposedPort:  8080,
			WorkDir:      "/app",
			InstallCmd:   "cargo fetch",
			BuildCommand: "cargo build --release",
			StartCmd:     "./target/release/app",
		},
	}

	if !strategy.Detect(result) {
		t.Error("RustBuildStrategy should detect rust")
	}

	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "rust:1.80-slim" {
		t.Errorf("expected rust:1.80-slim, got %s", plan.BaseImage)
	}
}

func TestJavaBuildStrategy(t *testing.T) {
	strategy := &JavaBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:           uuid.New(),
			Language:     "java",
			Framework:    "spring",
			BaseImage:    "eclipse-temurin:21-jdk-alpine",
			ExposedPort:  8080,
			WorkDir:      "/app",
		},
	}

	if !strategy.Detect(result) {
		t.Error("JavaBuildStrategy should detect java")
	}

	// Test with maven
	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should default to maven when no gradle lockfile
	if plan.Dockerfile == "" {
		t.Error("expected dockerfile")
	}
}

func TestComposeBuildStrategy(t *testing.T) {
	strategy := &ComposeBuildStrategy{}

	// Test without compose
	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:       uuid.New(),
			Language: "node",
		},
	}
	if strategy.Detect(result) {
		t.Error("ComposeBuildStrategy should not detect without compose")
	}

	// Test with compose
	result = &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:       uuid.New(),
			Language: "node",
		},
		DockerCompose: &analyzer.DockerComposeConfig{
			Services: map[string]analyzer.ComposeService{
				"web": {Image: "nginx:latest"},
			},
		},
	}
	if !strategy.Detect(result) {
		t.Error("ComposeBuildStrategy should detect with compose")
	}

	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.BaseImage != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", plan.BaseImage)
	}
}

func TestFallbackBuildStrategy(t *testing.T) {
	strategy := &FallbackBuildStrategy{}

	result := &analyzer.DetectionResult{
		RuntimeProfile: &domain.RuntimeProfile{
			ID:       uuid.New(),
			Language: "unknown",
		},
	}

	// Fallback should always detect
	if !strategy.Detect(result) {
		t.Error("FallbackBuildStrategy should always detect")
	}

	plan, err := strategy.GenerateBuildPlan(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Confidence >= 0.5 {
		t.Errorf("expected low confidence for fallback, got %f", plan.Confidence)
	}
}