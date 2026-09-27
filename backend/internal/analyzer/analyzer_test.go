package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

func TestArchitectureDetector(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected string
	}{
		{
			name: "detects monorepo with pnpm-workspace.yaml",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte("packages:\n  - apps/*\n  - packages/*\n"), 0644)
			},
			expected: "MONOREPO",
		},
		{
			name: "detects monorepo with turbo.json",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "turbo.json"), []byte("{\"pipeline\":{}}\n"), 0644)
			},
			expected: "MONOREPO",
		},
		{
			name: "detects monorepo with nx.json",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "nx.json"), []byte("{\"namedInputs\":{}}\n"), 0644)
			},
			expected: "MONOREPO",
		},
		{
			name: "detects monorepo with package.json workspaces",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"test\",\"workspaces\":[\"packages/*\"]}\n"), 0644)
			},
			expected: "MONOREPO",
		},
		{
			name: "detects library with package.json main",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"mylib\",\"main\":\"index.js\"}\n"), 0644)
			},
			expected: "LIBRARY",
		},
		{
			name: "detects CLI with bin field",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"mycli\",\"bin\":{\"mycli\":\"./bin.js\"}}\n"), 0644)
			},
			expected: "CLI",
		},
		{
			name: "detects API with routes directory",
			setup: func(dir string) {
				os.Mkdir(filepath.Join(dir, "routes"), 0755)
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"api\"}\n"), 0644)
			},
			expected: "API",
		},
		{
			name: "detects API with openapi spec",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644)
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"api\"}\n"), 0644)
			},
			expected: "API",
		},
		{
			name: "detects full stack with frontend and backend",
			setup: func(dir string) {
				os.Mkdir(filepath.Join(dir, "app"), 0755)
				os.WriteFile(filepath.Join(dir, "app/page.tsx"), []byte("export default () => <div>Home</div>\n"), 0644)
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"fullstack\"}\n"), 0644)
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644)
			},
			expected: "FULL_STACK",
		},
		{
			name: "detects web app with frontend only",
			setup: func(dir string) {
				os.Mkdir(filepath.Join(dir, "app"), 0755)
				os.WriteFile(filepath.Join(dir, "app/page.tsx"), []byte("export default () => <div>Home</div>\n"), 0644)
				os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><body>Home</body></html>\n"), 0644)
			},
			expected: "WEB_APP",
		},
		{
			name: "detects web app with backend only",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0644)
			},
			expected: "WEB_APP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			detector := NewArchitectureDetector(dir)
			result := detector.Detect()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFrameworkDetector_Go(t *testing.T) {
	dir := t.TempDir()

	// Test Gin
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\nrequire github.com/gin-gonic/gin v1.9.0\n"), 0644)
	detector := NewFrameworkDetector(dir)
	fw, conf := detector.detectGo()
	if fw != "gin" {
		t.Errorf("expected gin, got %s", fw)
	}
	if conf["gin"] < 0.8 {
		t.Errorf("expected high confidence for gin, got %f", conf["gin"])
	}

	// Test Fiber
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\nrequire github.com/gofiber/fiber/v2 v2.50.0\n"), 0644)
	detector = NewFrameworkDetector(dir)
	fw, conf = detector.detectGo()
	if fw != "fiber" {
		t.Errorf("expected fiber, got %s", fw)
	}

	// Test Echo
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\nrequire github.com/labstack/echo/v4 v4.11.0\n"), 0644)
	detector = NewFrameworkDetector(dir)
	fw, conf = detector.detectGo()
	if fw != "echo" {
		t.Errorf("expected echo, got %s", fw)
	}

	// Test stdlib (no framework)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\nrequire net/http v0.0.0\n"), 0644)
	detector = NewFrameworkDetector(dir)
	fw, conf = detector.detectGo()
	if fw != "stdlib" {
		t.Errorf("expected stdlib, got %s", fw)
	}
}

func TestFrameworkDetector_Rust(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		content  string
		expected string
	}{
		{content: "[dependencies]\naxum = \"0.7\"\n", expected: "axum"},
		{content: "[dependencies]\nactix-web = \"4\"\n", expected: "actix"},
		{content: "[dependencies]\nrocket = \"0.5\"\n", expected: "rocket"},
		{content: "[dependencies]\nsalvo = \"0.60\"\n", expected: "salvo"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(tt.content), 0644)
			detector := NewFrameworkDetector(dir)
			fw, conf := detector.detectRust()
			if fw != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, fw)
			}
			if conf[tt.expected] < 0.8 {
				t.Errorf("expected high confidence for %s, got %f", tt.expected, conf[tt.expected])
			}
		})
	}
}

func TestFrameworkDetector_Java(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		content  string
		expected string
	}{
		{content: "dependencies {\n    implementation 'org.springframework.boot:spring-boot-starter-web'\n}", expected: "spring"},
		{content: "dependencies {\n    implementation 'io.quarkus:quarkus-resteasy'\n}", expected: "quarkus"},
		{content: "dependencies {\n    implementation 'io.micronaut:micronaut-http'\n}", expected: "micronaut"},
		{content: "dependencies {\n    implementation 'io.vertx:vertx-web'\n}", expected: "vertx"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(tt.content), 0644)
			detector := NewFrameworkDetector(dir)
			fw, _ := detector.detectJava()
			if fw != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, fw)
			}
		})
	}
}

func TestFrameworkDetector_Python(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		content  string
		expected string
	}{
		{content: "django\n", expected: "django"},
		{content: "fastapi\n", expected: "fastapi"},
		{content: "flask\n", expected: "flask"},
		{content: "starlette\n", expected: "starlette"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(tt.content), 0644)
			detector := NewFrameworkDetector(dir)
			fw, _ := detector.detectPython()
			if fw != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, fw)
			}
		})
	}
}

func TestFrameworkDetector_Node(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		content  string
		expected string
	}{
		{content: `{"dependencies":{"next":"14.0.0"}}`, expected: "next"},
		{content: `{"dependencies":{"vite":"5.0.0"}}`, expected: "vite"},
		{content: `{"dependencies":{"express":"4.18.0"}}`, expected: "express"},
		{content: `{"dependencies":{"@nestjs/core":"10.0.0"}}`, expected: "nest"},
		{content: `{"dependencies":{"fastify":"4.0.0"}}`, expected: "fastify"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			os.WriteFile(filepath.Join(dir, "package.json"), []byte(tt.content), 0644)
			detector := NewFrameworkDetector(dir)
			fw, _ := detector.detectNode()
			if fw != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, fw)
			}
		})
	}
}

func TestEnhancedAnalyzer_DetectEntryPoints(t *testing.T) {
	dir := t.TempDir()

	// Create a main.py and main.go (ambiguous)
	os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	analyzer := NewEnhancedAnalyzer(dir)
	result := analyzer.Analyze()

	if !result.AmbiguousEntry {
		t.Error("expected ambiguous entry with multiple entry points")
	}

	if len(result.EntryPoints) < 2 {
		t.Errorf("expected at least 2 entry points, got %d", len(result.EntryPoints))
	}

	// Check for main.py
	found := false
	for _, ep := range result.EntryPoints {
		if ep.Path == "main.py" || ep.Path == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected main.py or main.go in entry points")
	}
}

func TestEnhancedAnalyzer_DockerCompose(t *testing.T) {
	dir := t.TempDir()

	compose := `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    environment:
      NODE_ENV: production
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
`

	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	analyzer := NewEnhancedAnalyzer(dir)
	result := analyzer.Analyze()

	if result.DockerCompose == nil {
		t.Fatal("expected docker compose config")
	}

	if len(result.DockerCompose.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.DockerCompose.Services))
	}

	if result.DockerCompose.Services["web"].Image != "nginx:latest" {
		t.Errorf("expected web image nginx:latest, got %s", result.DockerCompose.Services["web"].Image)
	}

	if len(result.DockerCompose.Services["web"].Ports) != 1 || result.DockerCompose.Services["web"].Ports[0] != "80:80" {
		t.Errorf("expected port 80:80 for web, got %v", result.DockerCompose.Services["web"].Ports)
	}

	if result.DockerCompose.Services["web"].Environment["NODE_ENV"] != "production" {
		t.Errorf("expected NODE_ENV=production, got %v", result.DockerCompose.Services["web"].Environment)
	}
}

func TestEnhancedAnalyzer_Lockfiles(t *testing.T) {
	dir := t.TempDir()

	// Create multiple lockfiles
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{\"lockfileVersion\":2}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "Cargo.lock"), []byte("# Cargo lockfile\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.sum"), []byte("github.com/gin-gonic/gin v1.9.0\n"), 0644)

	analyzer := NewEnhancedAnalyzer(dir)
	result := analyzer.Analyze()

	if len(result.Lockfiles) < 3 {
		t.Errorf("expected at least 3 lockfiles, got %d", len(result.Lockfiles))
	}

	lockfileTypes := make(map[string]bool)
	for _, lf := range result.Lockfiles {
		lockfileTypes[lf.LockfileType] = true
		if lf.Hash == "" {
			t.Errorf("expected hash for %s", lf.Path)
		}
	}

	if !lockfileTypes["npm"] || !lockfileTypes["cargo"] || !lockfileTypes["go"] {
		t.Errorf("expected npm, cargo, go lockfiles, got %v", lockfileTypes)
	}
}

func TestEnhancedAnalyzer_CacheKey(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{\"lockfileVersion\":2}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.sum"), []byte("github.com/gin-gonic/gin v1.9.0\n"), 0644)

	analyzer := NewEnhancedAnalyzer(dir)
	result := analyzer.Analyze()

	if result.CacheKey == "" {
		t.Error("expected cache key")
	}
	if len(result.CacheKey) != 16 {
		t.Errorf("expected 16 char cache key, got %d", len(result.CacheKey))
	}

	// Same lockfiles should produce same cache key
	analyzer2 := NewEnhancedAnalyzer(dir)
	result2 := analyzer2.Analyze()
	if result.CacheKey != result2.CacheKey {
		t.Error("cache key should be deterministic")
	}
}

func TestAnalyze_Node(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","scripts":{"dev":"vite"},"dependencies":{"vite":"5.0.0"}}`), 0644)

	profile := Analyze(dir)

	if profile.Language != "node" {
		t.Errorf("expected node, got %s", profile.Language)
	}
	if profile.StartCmd != "npm run dev" {
		t.Errorf("expected 'npm run dev', got %s", profile.StartCmd)
	}
	if profile.ExposedPort != 5173 {
		t.Errorf("expected port 5173 for vite, got %d", profile.ExposedPort)
	}
}

func TestAnalyze_Python(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("django\n"), 0644)
	os.WriteFile(filepath.Join(dir, "manage.py"), []byte("# manage.py\n"), 0644)

	profile := Analyze(dir)

	if profile.Language != "python" {
		t.Errorf("expected python, got %s", profile.Language)
	}
	if profile.StartCmd != "python manage.py runserver 0.0.0.0:8000" {
		t.Errorf("expected django start command, got %s", profile.StartCmd)
	}
}

func TestAnalyze_Go(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644)

	profile := Analyze(dir)

	if profile.Language != "go" {
		t.Errorf("expected go, got %s", profile.Language)
	}
	if profile.StartCmd != "go run ." {
		t.Errorf("expected 'go run .', got %s", profile.StartCmd)
	}
}

func TestAnalyze_Rust(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\"\nversion = \"0.1.0\"\n"), 0644)

	profile := Analyze(dir)

	if profile.Language != "rust" {
		t.Errorf("expected rust, got %s", profile.Language)
	}
	if profile.StartCmd != "cargo run" {
		t.Errorf("expected 'cargo run', got %s", profile.StartCmd)
	}
}

func TestAnalyze_Unknown(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644)

	profile := Analyze(dir)

	if profile.Language != "unknown" {
		t.Errorf("expected unknown, got %s", profile.Language)
	}
}

func TestEnhancedAnalyzer_FullAnalysis(t *testing.T) {
	dir := t.TempDir()

	// Create a realistic project structure
	os.Mkdir(filepath.Join(dir, "app"), 0755)
	os.Mkdir(filepath.Join(dir, "components"), 0755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","scripts":{"dev":"next dev"},"dependencies":{"next":"14.0.0","react":"18.0.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{\"lockfileVersion\":2}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "app/page.tsx"), []byte("export default () => <div>Hello</div>;\n"), 0644)
	os.WriteFile(filepath.Join(dir, "components/Header.tsx"), []byte("export default function Header() { return <header>App</header> }\n"), 0644)

	analyzer := NewEnhancedAnalyzer(dir)
	result := analyzer.Analyze()

	// Check basic detection
	if result.RuntimeProfile == nil {
		t.Fatal("expected runtime profile")
	}
	if result.RuntimeProfile.Language != "node" {
		t.Errorf("expected node, got %s", result.RuntimeProfile.Language)
	}
	if result.Framework != "next" {
		t.Errorf("expected next framework, got %s", result.Framework)
	}
	if result.Architecture != "FULL_STACK" {
		t.Errorf("expected FULL_STACK architecture, got %s", result.Architecture)
	}

	// Check lockfiles
	if len(result.Lockfiles) == 0 {
		t.Error("expected lockfiles")
	}

	// Check cache key
	if result.CacheKey == "" {
		t.Error("expected cache key")
	}

	// Check entry points
	if len(result.EntryPoints) == 0 {
		t.Error("expected entry points")
	}
}

func TestArchitectureDetector_IsLibrary(t *testing.T) {
	dir := t.TempDir()

	// Test with lib directory
	os.Mkdir(filepath.Join(dir, "lib"), 0755)
	detector := NewArchitectureDetector(dir)
	if !detector.isLibrary() {
		t.Error("expected library detection with lib/ directory")
	}

	// Test with package.json main
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "package.json"), []byte("{\"name\":\"test\",\"main\":\"index.js\"}\n"), 0644)
	detector = NewArchitectureDetector(dir2)
	if !detector.isLibrary() {
		t.Error("expected library detection with package.json main")
	}
}

func TestArchitectureDetector_IsCLI(t *testing.T) {
	dir := t.TempDir()

	// Test with commander dependency
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"cli\",\"dependencies\":{\"commander\":\"10.0.0\"}}\n"), 0644)
	detector := NewArchitectureDetector(dir)
	if !detector.isCLI() {
		t.Error("expected CLI detection with commander")
	}

	// Test with cobra
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "package.json"), []byte("{\"name\":\"cli\",\"dependencies\":{\"cobra\":\"1.8.0\"}}\n"), 0644)
	detector = NewArchitectureDetector(dir2)
	if !detector.isCLI() {
		t.Error("expected CLI detection with cobra")
	}
}

func TestArchitectureDetector_IsAPI(t *testing.T) {
	dir := t.TempDir()

	// Test with routes directory
	os.Mkdir(filepath.Join(dir, "routes"), 0755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"api\"}\n"), 0644)
	detector := NewArchitectureDetector(dir)
	if !detector.isAPI() {
		t.Error("expected API detection with routes/ directory")
	}

	// Test with openapi.yaml
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644)
	os.WriteFile(filepath.Join(dir2, "package.json"), []byte("{\"name\":\"api\"}\n"), 0644)
	detector = NewArchitectureDetector(dir2)
	if !detector.isAPI() {
		t.Error("expected API detection with openapi.yaml")
	}
}

func TestArchitectureDetector_HasWorkspacesField(t *testing.T) {
	dir := t.TempDir()

	// With workspaces
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{\"name\":\"mono\",\"workspaces\":[\"packages/*\"]}\n"), 0644)
	detector := NewArchitectureDetector(dir)
	if !detector.hasWorkspacesField() {
		t.Error("expected workspaces field detection")
	}

	// Without workspaces
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir2, "package.json"), []byte("{\"name\":\"regular\"}\n"), 0644)
	detector = NewArchitectureDetector(dir2)
	if detector.hasWorkspacesField() {
		t.Error("expected no workspaces field")
	}
}

// Helper to create a mock runtime profile
func createMockProfile() *domain.RuntimeProfile {
	return &domain.RuntimeProfile{
		ID:              uuid.New(),
		Language:        "node",
		Framework:       "next",
		Architecture:    "WEB_APP",
		BaseImage:       "node:20-alpine",
		InstallCmd:      "npm ci",
		BuildCommand:    "npm run build",
		StartCmd:        "npm start",
		ExposedPort:     3000,
		WorkDir:         "/app",
		DockerfileSource: "GENERATED",
		Confidence:      0.9,
	}
}