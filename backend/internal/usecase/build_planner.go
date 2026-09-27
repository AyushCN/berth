package usecase

import (
	"context"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
)

// BuildPlanner coordinates build strategies to generate build plans
type BuildPlanner struct {
	strategies []BuildStrategy
}

func NewBuildPlanner() *BuildPlanner {
	return &BuildPlanner{
		strategies: []BuildStrategy{
			&ComposeBuildStrategy{},
			&PythonBuildStrategy{},
			&NodeBuildStrategy{},
			&GoBuildStrategy{},
			&RustBuildStrategy{},
			&JavaBuildStrategy{},
			&FallbackBuildStrategy{}, // Always last
		},
	}
}

// RegisterStrategy adds a new build strategy
func (p *BuildPlanner) RegisterStrategy(strategy BuildStrategy) {
	p.strategies = append(p.strategies, strategy)
}

// GenerateBuildPlan analyzes the detection result and generates a build plan
func (p *BuildPlanner) GenerateBuildPlan(ctx context.Context, result *analyzer.DetectionResult) (*domain.BuildPlan, error) {
	// Find the first matching strategy
	for _, strategy := range p.strategies {
		if strategy.Detect(result) {
			return strategy.GenerateBuildPlan(ctx, result)
		}
	}

	// Should never reach here since FallbackStrategy always matches
	fallback := &FallbackBuildStrategy{}
	return fallback.GenerateBuildPlan(ctx, result)
}

// GetStrategyNames returns the names of all registered strategies
func (p *BuildPlanner) GetStrategyNames() []string {
	names := make([]string, len(p.strategies))
	for i, s := range p.strategies {
		names[i] = s.Name()
	}
	return names
}