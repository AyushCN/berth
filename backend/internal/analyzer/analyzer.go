package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/AyushCN/berth/internal/domain"
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
			ExposedPort: 3000,
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
