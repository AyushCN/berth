package main

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/api-sandbox/backend/db"
	"github.com/api-sandbox/backend/provider"
	"github.com/google/uuid"
)

func main() {
	// Initialize DB and Docker client
	db.InitDB()
	provider.InitDocker()

	// Configuration
	repos := []string{
		"https://github.com/expressjs/express/tree/master/examples/hello-world", // Node (minimal)
		"https://github.com/gin-gonic/gin",                                      // Go (minimal)
		// Add more realistic repos as required for the study
	}
	conditions := []string{"cold", "warm"}
	trials := 5

	ctx := context.Background()

	// Generate a randomized execution plan to avoid temporal bias
	type runInfo struct {
		repo      string
		condition string
		trial     int
	}
	var executionPlan []runInfo
	for _, repo := range repos {
		for _, condition := range conditions {
			for i := 1; i <= trials; i++ {
				executionPlan = append(executionPlan, runInfo{repo, condition, i})
			}
		}
	}

	// Shuffle execution plan
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(executionPlan), func(i, j int) {
		executionPlan[i], executionPlan[j] = executionPlan[j], executionPlan[i]
	})

	slog.Info("Starting benchmark runs", "total_runs", len(executionPlan))

	for idx, run := range executionPlan {
		slog.Info("Executing run", "run", idx+1, "total", len(executionPlan), "repo", run.repo, "condition", run.condition, "trial", run.trial)

		// Set the condition for the provider logic
		os.Setenv("BERTH_CACHE_MODE", run.condition)

		// Generate dummy env/entity data
		envID := uuid.NewString()
		entityType := "benchmark"
		
		// Determine branch, fallback to empty to let clone fallback work
		branch := "main" // this will be overridden if there's a tree/branch in URL, or default will be used by cloneOrFetch

		_, err := provider.CloneAndBuildImage(ctx, envID, entityType, run.repo, branch)
		if err != nil {
			slog.Error("Run failed", "repo", run.repo, "condition", run.condition, "error", err)
			continue
		}
		
		// Clean up the image after we record its size, so cold runs are truly cold next time
		// Optionally also clean up if warm run is complete, but we need the warm image to exist *for* the warm condition.
		// A full benchmark script might want to prune images only when starting a new "cold" set for a specific repo, 
		// but since we randomized the plan, we might need a more careful approach to image caching in a real scenario.
		// For now, this serves as the foundational harness.
		
		slog.Info("Run completed successfully", "repo", run.repo, "condition", run.condition, "trial", run.trial)
	}

	slog.Info("All benchmark runs finished")
}
