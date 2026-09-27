package usecase

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// DataCollector collects training data from builds and analyses
type DataCollector struct {
	trainingDataRepo domain.TrainingDataRepository
	buildRepo        BuildRepository
	runtimeRepo      RuntimeProfileRepository
}

type BuildRepository interface {
	GetBuildsByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.Build, error)
	GetLatestBuildByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.Build, error)
}

type RuntimeProfileRepository interface {
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.RuntimeProfile, error)
}

func NewDataCollector(
	trainingDataRepo domain.TrainingDataRepository,
	buildRepo BuildRepository,
	runtimeRepo RuntimeProfileRepository,
) *DataCollector {
	return &DataCollector{
		trainingDataRepo: trainingDataRepo,
		buildRepo:        buildRepo,
		runtimeRepo:      runtimeRepo,
	}
}

// CollectFromBuild collects training data from a completed build
func (c *DataCollector) CollectFromBuild(ctx context.Context, build *domain.Build, plan *domain.BuildPlan, profile *domain.RuntimeProfile, result *analyzer.DetectionResult) error {
	if build.Status != domain.BuildStatusSuccess && build.Status != domain.BuildStatusFailed {
		return nil // Only collect from completed builds
	}

	features := c.extractFeatures(profile, plan, result)
	labels := c.extractLabels(build)

	data := &domain.TrainingData{
		ID:           uuid.New(),
		WorkspaceID:  build.WorkspaceID,
		BuildID:      &build.ID,
		Features:     features,
		Labels:       labels,
		Architecture: profile.Architecture,
		Framework:    profile.Framework,
		Language:     profile.Language,
		CacheKey:     result.CacheKey,
		CreatedAt:    time.Now(),
	}

	return c.trainingDataRepo.Create(ctx, data)
}

// CollectFromProfile collects training data from a runtime profile
func (c *DataCollector) CollectFromProfile(ctx context.Context, profile *domain.RuntimeProfile, result *analyzer.DetectionResult) error {
	features := c.extractFeaturesFromProfile(profile, result)
	labels := map[string]any{} // No labels for profile-only data

	data := &domain.TrainingData{
		ID:           uuid.New(),
		WorkspaceID:  *profile.WorkspaceID,
		BuildID:      nil,
		Features:     features,
		Labels:       labels,
		Architecture: profile.Architecture,
		Framework:    profile.Framework,
		Language:     profile.Language,
		CacheKey:     result.CacheKey,
		CreatedAt:    time.Now(),
	}

	return c.trainingDataRepo.Create(ctx, data)
}

func (c *DataCollector) extractFeatures(profile *domain.RuntimeProfile, plan *domain.BuildPlan, result *analyzer.DetectionResult) map[string]any {
	features := map[string]any{
		"language":           profile.Language,
		"framework":          profile.Framework,
		"architecture":       profile.Architecture,
		"has_dockerfile":     profile.DockerfileSource == "USER_PROVIDED",
		"base_image":         plan.BaseImage,
		"exposed_port":       profile.ExposedPort,
		"cache_key":          result.CacheKey,
		"num_lockfiles":      len(result.Lockfiles),
		"has_compose":        result.DockerCompose != nil,
		"num_services":       0,
		"entry_point_count":  len(result.EntryPoints),
		"ambiguous_entry":    result.AmbiguousEntry,
	}

	if result.DockerCompose != nil {
		features["num_services"] = len(result.DockerCompose.Services)
	}

	// Lockfile types
	lockfileTypes := make([]string, len(result.Lockfiles))
	for i, lf := range result.Lockfiles {
		lockfileTypes[i] = lf.LockfileType
	}
	features["lockfile_types"] = lockfileTypes

	// Entry point types
	entryTypes := make([]string, len(result.EntryPoints))
	for i, ep := range result.EntryPoints {
		entryTypes[i] = ep.Type
	}
	features["entry_types"] = entryTypes

	// Framework confidences
	features["framework_confidences"] = result.FrameworkConf

	return features
}

func (c *DataCollector) extractFeaturesFromProfile(profile *domain.RuntimeProfile, result *analyzer.DetectionResult) map[string]any {
	return c.extractFeatures(profile, &domain.BuildPlan{
		BaseImage: profile.BaseImage,
		Port:      profile.ExposedPort,
	}, result)
}

func (c *DataCollector) extractLabels(build *domain.Build) map[string]any {
	labels := map[string]any{
		"build_duration_ms": build.BuildDurationMs,
		"cache_hit":         build.CacheHit,
		"status":            string(build.Status),
	}

	if build.ImageID != nil {
		labels["image_id"] = build.ImageID.String()
	}

	if build.Logs != "" {
		// Extract image size from logs if available
		labels["logs_length"] = len(build.Logs)
	}

	return labels
}

// CollectBatch collects training data for all completed builds in a workspace
func (c *DataCollector) CollectBatch(ctx context.Context, workspaceID uuid.UUID) error {
	builds, err := c.buildRepo.GetBuildsByWorkspace(ctx, workspaceID, 1000, 0)
	if err != nil {
		return err
	}

	profiles, err := c.runtimeRepo.GetByWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}

	profileMap := make(map[uuid.UUID]*domain.RuntimeProfile)
	for _, p := range profiles {
		profileMap[p.ID] = p
	}

	for _, build := range builds {
		if build.Status != domain.BuildStatusSuccess && build.Status != domain.BuildStatusFailed {
			continue
		}

		profile := profileMap[build.BuildPlanID]
		if profile == nil {
			continue
		}

		// Create a minimal detection result for historical data
		result := &analyzer.DetectionResult{
			RuntimeProfile: profile,
			Architecture:   profile.Architecture,
			Framework:      profile.Framework,
			CacheKey:       "",
			Lockfiles:      []analyzer.LockfileInfo{},
			EntryPoints:    []analyzer.EntryPoint{},
		}

		plan := &domain.BuildPlan{
			BaseImage: profile.BaseImage,
			Port:      profile.ExposedPort,
		}

		if err := c.CollectFromBuild(ctx, build, plan, profile, result); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// GetTrainingDataset retrieves training data for a specific prediction type
func (c *DataCollector) GetTrainingDataset(ctx context.Context, pType domain.PredictionType, limit, offset int) ([]*domain.TrainingData, error) {
	return c.trainingDataRepo.ListByType(ctx, pType, limit, offset)
}