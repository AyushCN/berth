package usecase

import (
	"context"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
)

// BuildStrategy defines the interface for build strategies
type BuildStrategy interface {
	// Name returns the name of the strategy
	Name() string
	// Detect returns true if this strategy can handle the given detection result
	Detect(result *analyzer.DetectionResult) bool
	// GenerateBuildPlan creates a build plan from the detection result
	GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error)
}