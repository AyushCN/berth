package analyzer

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AyushCN/berth/internal/domain"
)

// ArchitectureType represents the detected architecture type of the application
type ArchitectureType string

const (
	ArchWebApp    = "WEB_APP"
	ArchAPI       = "API"
	ArchCLI       = "CLI"
	ArchFullStack = "FULL_STACK"
	ArchMonorepo  = "MONOREPO"
	ArchLibrary   = "LIBRARY"
	ArchUnknown   = "UNKNOWN"
)

// FrameworkType represents the detected framework
type FrameworkType string

const (
	// Go frameworks
	FrameworkGin    = "gin"
	FrameworkFiber  = "fiber"
	FrameworkEcho   = "echo"
	FrameworkChi    = "chi"
	FrameworkStdlib = "stdlib"

	// Rust frameworks
	FrameworkActix  = "actix"
	FrameworkAxum   = "axum"
	FrameworkRocket = "rocket"
	FrameworkSalvo  = "salvo"

	// Python frameworks
	FrameworkDjango    = "django"
	FrameworkFastAPI   = "fastapi"
	FrameworkFlask     = "flask"
	FrameworkStarlette = "starlette"
	FrameworkQuart     = "quart"
	FrameworkTornado   = "tornado"
	FrameworkBottle    = "bottle"

	// Node frameworks
	FrameworkNext      = "next"
	FrameworkVite      = "vite"
	FrameworkExpress   = "express"
	FrameworkNest      = "nest"
	FrameworkKoa       = "koa"
	FrameworkFastify   = "fastify"
	FrameworkNuxt      = "nuxt"
	FrameworkRemix     = "remix"
	FrameworkSvelteKit = "sveltekit"
	FrameworkAstro     = "astro"

	// Java frameworks
	FrameworkSpring    = "spring"
	FrameworkQuarkus   = "quarkus"
	FrameworkMicronaut = "micronaut"
	FrameworkVertx     = "vertx"

	FrameworkUnknown = "unknown"
)

// Analyze examines the workspace directory and returns a RuntimeProfile based on static heuristics.
func Analyze(workspaceDir string) *domain.RuntimeProfile {
	// Rule 1: Node.js (package.json)
	if _, err := os.Stat(filepath.Join(workspaceDir, "package.json")); err == nil {
		profile := &domain.RuntimeProfile{
			Language:    "node",
			BaseImage:   "docker.io/library/node:20-alpine",
			InstallCmd:  "npm install",
			StartCmd:    "npm start",
			ExposedPort: 3001,
		}

		// Sub-rule: Check for specific Node.js frameworks or scripts
		pkgJSONBytes, err := os.ReadFile(filepath.Join(workspaceDir, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if err := json.Unmarshal(pkgJSONBytes, &pkg); err == nil {
				scripts, ok := pkg["scripts"].(map[string]interface{})
				if ok {
					if _, hasDev := scripts["dev"]; hasDev {
						profile.StartCmd = "npm run dev"
					} else if _, hasStart := scripts["start"]; !hasStart {
						// If no start or dev script, check if it's a simple index.js
						if _, err := os.Stat(filepath.Join(workspaceDir, "index.js")); err == nil {
							profile.StartCmd = "node index.js"
						}
					}
				}

				// Optional: Check dependencies to dynamically assign ports
				deps, ok := pkg["dependencies"].(map[string]interface{})
				if ok {
					if _, hasNext := deps["next"]; hasNext {
						profile.ExposedPort = 3000
					} else if _, hasVite := deps["vite"]; hasVite {
						profile.ExposedPort = 5173
					}
				}
			}
		}
		return profile
	}

	// Rule 2: Python (requirements.txt)
	if _, err := os.Stat(filepath.Join(workspaceDir, "requirements.txt")); err == nil {
		profile := &domain.RuntimeProfile{
			Language:    "python",
			BaseImage:   "docker.io/library/python:3.11-slim",
			InstallCmd:  "pip install -r requirements.txt",
			StartCmd:    "python main.py", // Fallback start command
			ExposedPort: 8000,
		}

		// Attempt to find a web framework start script
		if _, err := os.Stat(filepath.Join(workspaceDir, "manage.py")); err == nil {
			profile.StartCmd = "python manage.py runserver 0.0.0.0:8000" // Django
		} else if _, err := os.Stat(filepath.Join(workspaceDir, "app.py")); err == nil {
			profile.StartCmd = "python app.py" // Flask / Generic
		} else if _, err := os.Stat(filepath.Join(workspaceDir, "back.py")); err == nil {
			// Check if it's Flask/FastAPI
			content, _ := os.ReadFile(filepath.Join(workspaceDir, "back.py"))
			if bytes.Contains(content, []byte("flask")) || bytes.Contains(content, []byte("Flask")) {
				profile.StartCmd = "FLASK_APP=back.py flask run --host=0.0.0.0 --port=8000 --reload"
			} else if bytes.Contains(content, []byte("fastapi")) || bytes.Contains(content, []byte("FastAPI")) {
				profile.StartCmd = "uvicorn back:app --host 0.0.0.0 --port 8000 --reload"
			} else {
				profile.StartCmd = "python back.py"
			}
		} else if _, err := os.Stat(filepath.Join(workspaceDir, "server.py")); err == nil {
			profile.StartCmd = "python server.py"
		} else if _, err := os.Stat(filepath.Join(workspaceDir, "run.py")); err == nil {
			profile.StartCmd = "python run.py"
		}

		return profile
	}

	// Rule 3: Go (go.mod)
	if _, err := os.Stat(filepath.Join(workspaceDir, "go.mod")); err == nil {
		return &domain.RuntimeProfile{
			Language:    "go",
			BaseImage:   "docker.io/library/golang:1.23-alpine",
			InstallCmd:  "go mod download",
			StartCmd:    "go run .",
			ExposedPort: 8080,
		}
	}

	// Rule 4: Rust (Cargo.toml)
	if _, err := os.Stat(filepath.Join(workspaceDir, "Cargo.toml")); err == nil {
		return &domain.RuntimeProfile{
			Language:    "rust",
			BaseImage:   "docker.io/library/rust:1.80-slim",
			InstallCmd:  "cargo build",
			StartCmd:    "cargo run",
			ExposedPort: 8080,
		}
	}

	// Fallback to minimal Alpine if nothing matches
	return &domain.RuntimeProfile{
		Language:    "unknown",
		BaseImage:   "docker.io/library/alpine:latest",
		InstallCmd:  "",
		StartCmd:    "sh -c 'while true; do sleep 3600; done'",
		ExposedPort: 80,
	}
}

// ArchitectureDetector detects the architecture type of the application
type ArchitectureDetector struct {
	workspaceDir string
}

func NewArchitectureDetector(workspaceDir string) *ArchitectureDetector {
	return &ArchitectureDetector{workspaceDir: workspaceDir}
}

func (d *ArchitectureDetector) Detect() string {
	// Check for monorepo indicators
	if d.isMonorepo() {
		return "MONOREPO"
	}

	// Check for library indicators
	if d.isLibrary() {
		return "LIBRARY"
	}

	// Check for CLI indicators
	if d.isCLI() {
		return "CLI"
	}

	// Check for API indicators
	if d.isAPI() {
		return "API"
	}

	// Check for full-stack indicators
	if d.isFullStack() {
		return "FULL_STACK"
	}

	// Check for web app indicators
	if d.isWebApp() {
		return "WEB_APP"
	}

	return "UNKNOWN"
}

func (d *ArchitectureDetector) isMonorepo() bool {
	monorepoFiles := []string{
		"pnpm-workspace.yaml",
		"yarn.workspace.yaml",
		"package.json",
		"turbo.json",
		"nx.json",
		"nx.workspace.json",
		"lerna.json",
		"rush.json",
	}

	for _, f := range monorepoFiles {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, f)); err == nil {
			if f == "package.json" {
				if d.hasWorkspacesField() {
					return true
				}
				continue
			}
			return true
		}
	}
	return false
}

func (d *ArchitectureDetector) hasWorkspacesField() bool {
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "package.json"))
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &pkg); err != nil {
		return false
	}
	_, ok := pkg["workspaces"]
	return ok
}

func (d *ArchitectureDetector) isLibrary() bool {
	libIndicators := []string{"lib/", "src/", "dist/", "types/"}
	for _, ind := range libIndicators {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, ind)); err == nil {
			return true
		}
	}
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "package.json"))
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &pkg); err != nil {
		return false
	}
	if _, ok := pkg["main"]; ok {
		return true
	}
	if _, ok := pkg["module"]; ok {
		return true
	}
	if _, ok := pkg["types"]; ok {
		return true
	}
	if _, ok := pkg["typings"]; ok {
		return true
	}
	return false
}

func (d *ArchitectureDetector) isCLI() bool {
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "package.json"))
	if err != nil {
		return false
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &pkg); err != nil {
		return false
	}
	if _, ok := pkg["bin"]; ok {
		return true
	}
	deps := make(map[string]bool)
	if depsMap, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k := range depsMap {
			deps[k] = true
		}
	}
	if devDepsMap, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k := range devDepsMap {
			deps[k] = true
		}
	}
	cliDeps := []string{"commander", "click", "cobra", "cliff", "urfave/cli", "kingpin", "argparse", "docopt"}
	for _, dep := range cliDeps {
		if deps[dep] {
			return true
		}
	}
	return false
}

func (d *ArchitectureDetector) isAPI() bool {
	apiDirs := []string{"routes/", "controllers/", "handlers/", "api/", "graphql/", "rest/", "endpoints/"}
	for _, dir := range apiDirs {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, dir)); err == nil {
			return true
		}
	}
	specFiles := []string{"schema.graphql", "schema.gql", "openapi.yaml", "openapi.json", "swagger.yaml", "swagger.json", "api.yaml", "api.json"}
	for _, f := range specFiles {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, f)); err == nil {
			return true
		}
	}
	return false
}

func (d *ArchitectureDetector) isFullStack() bool {
	hasFrontend := d.hasFrontend()
	hasBackend := d.hasBackend()
	return hasFrontend && hasBackend
}

func (d *ArchitectureDetector) hasFrontend() bool {
	frontendFiles := []string{
		"index.html", "public/index.html", "index.tsx", "index.tsx",
		"src/index.tsx", "src/index.jsx", "src/main.tsx", "src/main.jsx",
		"app/", "src/app/", "pages/", "src/pages/", "views/",
	}
	for _, f := range frontendFiles {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, f)); err == nil {
			return true
		}
	}
	return false
}

func (d *ArchitectureDetector) hasBackend() bool {
	backendFiles := []string{
		"requirements.txt", "pyproject.toml", "Pipfile", "poetry.lock",
		"go.mod", "go.sum", "Cargo.toml", "Cargo.lock",
		"pom.xml", "build.gradle", "build.gradle.kts", "gradlew",
		"composer.json", "Gemfile", "Gemfile.lock",
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	}
	for _, f := range backendFiles {
		if _, err := os.Stat(filepath.Join(d.workspaceDir, f)); err == nil {
			return true
		}
	}
	return false
}

func (d *ArchitectureDetector) isWebApp() bool {
	return d.hasFrontend() || d.hasBackend()
}

// FrameworkDetector detects the framework used by the application
type FrameworkDetector struct {
	workspaceDir string
}

func NewFrameworkDetector(workspaceDir string) *FrameworkDetector {
	return &FrameworkDetector{workspaceDir: workspaceDir}
}

func (d *FrameworkDetector) Detect() (string, map[string]float64) {
	if fw, conf := d.detectGo(); fw != "unknown" {
		return fw, conf
	}
	if fw, conf := d.detectRust(); fw != "unknown" {
		return fw, conf
	}
	if fw, conf := d.detectJava(); fw != "unknown" {
		return fw, conf
	}
	if fw, conf := d.detectPython(); fw != "unknown" {
		return fw, conf
	}
	if fw, conf := d.detectNode(); fw != "unknown" {
		return fw, conf
	}
	return "unknown", map[string]float64{}
}

func (d *FrameworkDetector) detectGo() (string, map[string]float64) {
	confidences := make(map[string]float64)
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "go.mod"))
	if err != nil {
		return "unknown", confidences
	}
	content := string(data)
	frameworks := map[string][]string{
		"gin":    {"github.com/gin-gonic/gin"},
		"fiber":  {"github.com/gofiber/fiber/v2", "github.com/gofiber/fiber"},
		"echo":   {"github.com/labstack/echo/v4", "github.com/labstack/echo"},
		"chi":    {"github.com/go-chi/chi/v5", "github.com/go-chi/chi"},
		"stdlib": {"net/http"},
	}
	maxConf := 0.0
	var detected string = "unknown"
	for fw, imports := range frameworks {
		conf := 0.0
		for _, imp := range imports {
			if strings.Contains(content, imp) {
				conf = 0.9
				break
			}
		}
		confidences[fw] = conf
		if conf > maxConf {
			maxConf = conf
			detected = fw
		}
	}
	return detected, confidences
}

func (d *FrameworkDetector) detectRust() (string, map[string]float64) {
	confidences := make(map[string]float64)
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "Cargo.toml"))
	if err != nil {
		return "unknown", confidences
	}
	content := string(data)
	frameworks := map[string][]string{
		"actix":  {"actix-web", "actix"},
		"axum":   {"axum"},
		"rocket": {"rocket"},
		"salvo":  {"salvo"},
	}
	maxConf := 0.0
	var detected string = "unknown"
	for fw, imports := range frameworks {
		conf := 0.0
		for _, imp := range imports {
			if strings.Contains(content, imp) {
				conf = 0.9
				break
			}
		}
		confidences[fw] = conf
		if conf > maxConf {
			maxConf = conf
			detected = fw
		}
	}
	return detected, confidences
}

func (d *FrameworkDetector) detectJava() (string, map[string]float64) {
	confidences := make(map[string]float64)
	buildFiles := []string{"pom.xml", "build.gradle", "build.gradle.kts"}
	var content string
	for _, f := range buildFiles {
		data, err := os.ReadFile(filepath.Join(d.workspaceDir, f))
		if err == nil {
			content += string(data)
		}
	}
	if content == "" {
		return "unknown", confidences
	}
	frameworks := map[string][]string{
		"spring":    {"spring-boot", "spring-boot-starter", "org.springframework"},
		"quarkus":   {"quarkus", "io.quarkus"},
		"micronaut": {"micronaut", "io.micronaut"},
		"vertx":     {"vertx", "io.vertx"},
	}
	maxConf := 0.0
	var detected string = "unknown"
	for fw, imports := range frameworks {
		conf := 0.0
		for _, imp := range imports {
			if strings.Contains(content, imp) {
				conf = 0.9
				break
			}
		}
		confidences[fw] = conf
		if conf > maxConf {
			maxConf = conf
			detected = fw
		}
	}
	return detected, confidences
}

func (d *FrameworkDetector) detectPython() (string, map[string]float64) {
	confidences := make(map[string]float64)
	files := []string{"pyproject.toml", "requirements.txt", "setup.py", "Pipfile", "poetry.lock"}
	var content string
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(d.workspaceDir, f))
		if err == nil {
			content += string(data)
		}
	}
	if content == "" {
		return "unknown", confidences
	}
	frameworks := map[string][]string{
		"django":    {"django", "django.contrib"},
		"fastapi":   {"fastapi"},
		"flask":     {"flask"},
		"starlette": {"starlette"},
		"quart":     {"quart"},
		"tornado":   {"tornado"},
		"bottle":    {"bottle"},
	}
	maxConf := 0.0
	var detected string = "unknown"
	for fw, imports := range frameworks {
		conf := 0.0
		for _, imp := range imports {
			if strings.Contains(content, imp) {
				conf = 0.9
				break
			}
		}
		confidences[fw] = conf
		if conf > maxConf {
			maxConf = conf
			detected = fw
		}
	}
	return detected, confidences
}

func (d *FrameworkDetector) detectNode() (string, map[string]float64) {
	confidences := make(map[string]float64)
	data, err := os.ReadFile(filepath.Join(d.workspaceDir, "package.json"))
	if err != nil {
		return "unknown", confidences
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &pkg); err != nil {
		return "unknown", confidences
	}
	deps := make(map[string]bool)
	if depsMap, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k := range depsMap {
			deps[k] = true
		}
	}
	if devDepsMap, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k := range devDepsMap {
			deps[k] = true
		}
	}
	frameworks := map[string][]string{
		"next":      {"next"},
		"vite":      {"vite"},
		"express":   {"express"},
		"nest":      {"@nestjs/core", "@nestjs/common"},
		"koa":       {"koa"},
		"fastify":   {"fastify"},
		"nuxt":      {"nuxt"},
		"remix":     {"@remix-run/react", "@remix-run/node"},
		"sveltekit": {"@sveltejs/kit"},
		"astro":     {"astro"},
	}
	maxConf := 0.0
	var detected string = "unknown"
	for fw, imports := range frameworks {
		conf := 0.0
		for _, imp := range imports {
			if deps[imp] {
				conf = 0.9
				break
			}
		}
		confidences[fw] = conf
		if conf > maxConf {
			maxConf = conf
			detected = fw
		}
	}
	return detected, confidences
}

// DetectionResult represents the result of analysis
type DetectionResult struct {
	RuntimeProfile *domain.RuntimeProfile
	Architecture   string
	Framework      string
	FrameworkConf  map[string]float64
	EntryPoints    []EntryPoint
	AmbiguousEntry bool
	DockerCompose  *DockerComposeConfig
	Lockfiles      []LockfileInfo
	CacheKey       string
}

type EntryPoint struct {
	Path        string
	Type        string
	Confidence  float64
	Description string
}

type DockerComposeConfig struct {
	Services map[string]ComposeService
	Networks map[string]interface{}
	Volumes  map[string]interface{}
}

type ComposeService struct {
	Image       string
	Build       map[string]interface{}
	Ports       []string
	Environment map[string]string
	Volumes     []string
	DependsOn   []string
	Command     string
	Entrypoint  []string
}

type LockfileInfo struct {
	Path         string
	LockfileType string
	Hash         string
	Packages     int
}

// EnhancedAnalyzer provides comprehensive analysis
type EnhancedAnalyzer struct {
	workspaceDir string
	archDetector *ArchitectureDetector
	fwDetector   *FrameworkDetector
}

func NewEnhancedAnalyzer(workspaceDir string) *EnhancedAnalyzer {
	return &EnhancedAnalyzer{
		workspaceDir: workspaceDir,
		archDetector: NewArchitectureDetector(workspaceDir),
		fwDetector:   NewFrameworkDetector(workspaceDir),
	}
}

func (e *EnhancedAnalyzer) Analyze() *DetectionResult {
	baseProfile := Analyze(e.workspaceDir)

	arch := NewArchitectureDetector(e.workspaceDir).Detect()

	framework, fwConf := NewFrameworkDetector(e.workspaceDir).Detect()

	entryPoints, ambiguous := e.detectEntryPoints()

	compose := e.parseDockerCompose()

	lockfiles := e.detectLockfiles()

	cacheKey := e.generateCacheKey()

	return &DetectionResult{
		RuntimeProfile: baseProfile,
		Architecture:   arch,
		Framework:      framework,
		FrameworkConf:  fwConf,
		EntryPoints:    entryPoints,
		AmbiguousEntry: ambiguous,
		DockerCompose:  compose,
		Lockfiles:      lockfiles,
		CacheKey:       cacheKey,
	}
}

func (e *EnhancedAnalyzer) enhanceProfile(base *domain.RuntimeProfile, arch, framework string) *domain.RuntimeProfile {
	if base == nil {
		return nil
	}
	base.Architecture = arch
	base.Framework = framework
	return base
}

func (e *EnhancedAnalyzer) detectEntryPoints() ([]EntryPoint, bool) {
	var entries []EntryPoint

	// Check for Dockerfile
	if _, err := os.Stat(filepath.Join(e.workspaceDir, "Dockerfile")); err == nil {
		entries = append(entries, EntryPoint{
			Path:        "Dockerfile",
			Type:        "dockerfile",
			Confidence:  1.0,
			Description: "Dockerfile found",
		})
	}

	// Check for docker-compose
	if _, err := os.Stat(filepath.Join(e.workspaceDir, "docker-compose.yml")); err == nil {
		entries = append(entries, EntryPoint{
			Path:        "docker-compose.yml",
			Type:        "compose",
			Confidence:  1.0,
			Description: "Docker Compose file found",
		})
	}
	if _, err := os.Stat(filepath.Join(e.workspaceDir, "docker-compose.yaml")); err == nil {
		entries = append(entries, EntryPoint{
			Path:        "docker-compose.yaml",
			Type:        "compose",
			Confidence:  1.0,
			Description: "Docker Compose file found",
		})
	}

	// Check for main entry points
	entryCandidates := map[string]string{
		"main.py":          "python",
		"app.py":           "python",
		"manage.py":        "python",
		"main.go":          "go",
		"main.rs":          "rust",
		"Cargo.toml":       "rust",
		"main.java":        "java",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"main.js":          "node",
		"index.js":         "node",
		"server.js":        "node",
		"app.js":           "node",
		"package.json":     "node",
		"requirements.txt": "python",
		"pyproject.toml":   "python",
		"setup.py":         "python",
		"Pipfile":          "python",
		"poetry.lock":      "python",
		"go.mod":           "go",
	}

	found := []string{}
	for path := range entryCandidates {
		if _, err := os.Stat(filepath.Join(e.workspaceDir, path)); err == nil {
			entries = append(entries, EntryPoint{
				Path:        path,
				Type:        "main",
				Confidence:  0.8,
				Description: fmt.Sprintf("Entry point: %s", path),
			})
			found = append(found, path)
		}
	}

	ambiguous := len(found) > 1

	return entries, ambiguous
}

func (e *EnhancedAnalyzer) parseDockerCompose() *DockerComposeConfig {
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, f := range composeFiles {
		path := filepath.Join(e.workspaceDir, f)
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var compose map[string]interface{}
			if err := yaml.Unmarshal(data, &compose); err != nil {
				continue
			}
			services := make(map[string]ComposeService)
			if servicesMap, ok := compose["services"].(map[string]interface{}); ok {
				for name, svc := range servicesMap {
					if svcMap, ok := svc.(map[string]interface{}); ok {
						svc := ComposeService{}
						if img, ok := svcMap["image"].(string); ok {
							svc.Image = img
						}
						if build, ok := svcMap["build"].(map[string]interface{}); ok {
							svc.Build = build
						}
						if ports, ok := svcMap["ports"].([]interface{}); ok {
							for _, p := range ports {
								if ps, ok := p.(string); ok {
									svc.Ports = append(svc.Ports, ps)
								}
							}
						}
						if env, ok := svcMap["environment"].(map[string]interface{}); ok {
							svc.Environment = make(map[string]string)
							for k, v := range env {
								svc.Environment[k] = fmt.Sprintf("%v", v)
							}
						}
						if vols, ok := svcMap["volumes"].([]interface{}); ok {
							for _, v := range vols {
								if vs, ok := v.(string); ok {
									svc.Volumes = append(svc.Volumes, vs)
								}
							}
						}
						if deps, ok := svcMap["depends_on"].([]interface{}); ok {
							for _, d := range deps {
								if ds, ok := d.(string); ok {
									svc.DependsOn = append(svc.DependsOn, ds)
								}
							}
						}
						if cmd, ok := svcMap["command"].(string); ok {
							svc.Command = cmd
						}
						if ep, ok := svcMap["entrypoint"].([]interface{}); ok {
							for _, e := range ep {
								if es, ok := e.(string); ok {
									svc.Entrypoint = append(svc.Entrypoint, es)
								}
							}
						}
						services[name] = svc
					}
				}
				return &DockerComposeConfig{
					Services: services,
				}
			}
		}
	}
	return nil
}

func (e *EnhancedAnalyzer) detectLockfiles() []LockfileInfo {
	lockfiles := []struct {
		path         string
		lockfileType string
	}{
		{"package-lock.json", "npm"},
		{"yarn.lock", "yarn"},
		{"pnpm-lock.yaml", "pnpm"},
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"Cargo.lock", "cargo"},
		{"go.sum", "go"},
		{"go.mod", "go"},
		{"requirements.txt", "pip"},
		{"requirements-dev.txt", "pip"},
		{"Pipfile.lock", "pipenv"},
		{"poetry.lock", "poetry"},
		{"composer.lock", "composer"},
		{"Gemfile.lock", "bundler"},
		{"pom.xml", "maven"},
		{"build.gradle", "gradle"},
		{"build.gradle.kts", "gradle"},
		{"gradle.lockfile", "gradle"},
	}

	var result []LockfileInfo
	for _, lf := range lockfiles {
		path := filepath.Join(e.workspaceDir, lf.path)
		if _, err := os.Stat(path); err == nil {
			data, _ := os.ReadFile(path)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))[:16]
			result = append(result, LockfileInfo{
				Path:         lf.path,
				LockfileType: lf.lockfileType,
				Hash:         hash,
				Packages:     0,
			})
		}
	}
	return result
}

func (e *EnhancedAnalyzer) generateCacheKey() string {
	lockfiles := e.detectLockfiles()
	var parts []string
	for _, lf := range lockfiles {
		parts = append(parts, lf.LockfileType+":"+lf.Hash)
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", h)[:16]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
