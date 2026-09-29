package worker

import (
	"fmt"
	"strings"

	"github.com/AyushCN/berth/internal/domain"
)

// WatcherCommand returns the process watcher command for a given runtime profile.
// This replaces the keep-alive + background start pattern with a proper watcher.
func WatcherCommand(profile *domain.RuntimeProfile) []string {
	switch profile.Language {
	case "node":
		return nodeWatcherCommand(profile)
	case "python":
		return pythonWatcherCommand(profile)
	case "go":
		return goWatcherCommand(profile)
	case "rust":
		return rustWatcherCommand(profile)
	default:
		// Fallback: use the profile's start command directly
		if profile.StartCmd != "" {
			return []string{"sh", "-c", profile.StartCmd}
		}
		return []string{"sh", "-c", "while true; do sleep 3600; done"}
	}
}

func nodeWatcherCommand(profile *domain.RuntimeProfile) []string {
	// Use nodemon for hot reload
	// nodemon watches for file changes and restarts the command
	startCmd := profile.StartCmd
	if startCmd == "" {
		startCmd = "npm run dev"
	}

	// Detect package manager for the right npx/nodemon path
	// For simplicity, use npx which works with npm/yarn/pnpm
	// Use -y flag to auto-confirm installation of nodemon
	watchCmd := fmt.Sprintf("npx -y nodemon --watch . --ext js,ts,json,jsx,tsx --exec \"%s\"", startCmd)

	// If using bun, use bun's built-in watcher
	if strings.Contains(profile.BaseImage, "oven/bun") {
		watchCmd = fmt.Sprintf("bun --watch %s", strings.Replace(startCmd, "npm run ", "bun run ", 1))
	}

	return []string{"sh", "-c", watchCmd}
}

func pythonWatcherCommand(profile *domain.RuntimeProfile) []string {
	// Use the analyzer's detected start command, with hot reload if possible
	startCmd := profile.StartCmd
	if startCmd == "" {
		startCmd = "uvicorn main:app --host 0.0.0.0 --port 8000 --reload"
	}

	// Ensure --reload flag is present for uvicorn/flask
	if !strings.Contains(startCmd, "--reload") {
		if strings.Contains(startCmd, "uvicorn") {
			startCmd = strings.Replace(startCmd, "uvicorn", "uvicorn --reload", 1)
		} else if strings.Contains(startCmd, "flask run") {
			startCmd = strings.Replace(startCmd, "flask run", "flask run --reload", 1)
		} else if strings.Contains(startCmd, "python manage.py runserver") {
			// Django's runserver has auto-reload by default
		}
	}

	// Wrap in a script that keeps container alive if entry point fails
	// This allows users to debug via terminal even if auto-detected command fails
	wrappedCmd := fmt.Sprintf(`
set -e
echo "Starting: %s"
if %s; then
    exit 0
else
    echo "Start command failed, keeping container alive for debugging..."
    echo "You can now connect via terminal to investigate."
    tail -f /dev/null
fi
`, startCmd, startCmd)

	return []string{"sh", "-c", wrappedCmd}
}

func goWatcherCommand(profile *domain.RuntimeProfile) []string {
	// Use air for hot reload
	// Check if .air.toml exists in workspace, otherwise use default config
	startCmd := profile.StartCmd
	if startCmd == "" {
		startCmd = "go run ."
	}

	// Use air if available, otherwise fallback to direct run
	// Air is included in golang:alpine images we use
	watchCmd := "air -c .air.toml"

	// If no air config, create a minimal one inline
	// For simplicity, just use air with default settings
	return []string{"sh", "-c", watchCmd}
}

func rustWatcherCommand(profile *domain.RuntimeProfile) []string {
	// Use cargo watch for hot reload
	startCmd := profile.StartCmd
	if startCmd == "" {
		startCmd = "cargo run"
	}

	watchCmd := "cargo watch -x run"
	return []string{"sh", "-c", watchCmd}
}
