package usecase

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// PredictionService provides prediction capabilities
type PredictionService struct {
	modelTrainer   *ModelTrainer
	predictionRepo domain.PredictionRepository
}

func NewPredictionService(
	modelTrainer *ModelTrainer,
	predictionRepo domain.PredictionRepository,
) *PredictionService {
	return &PredictionService{
		modelTrainer:   modelTrainer,
		predictionRepo: predictionRepo,
	}
}

// PredictBuildTime predicts the build time for a workspace
func (s *PredictionService) PredictBuildTime(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	prediction, err := s.modelTrainer.Predict(ctx, domain.PredictionTypeBuildTime, features)
	if err != nil {
		return nil, err
	}

	prediction.WorkspaceID = workspaceID
	if err := s.predictionRepo.Create(ctx, prediction); err != nil {
		return nil, err
	}

	return prediction, nil
}

// PredictImageSize predicts the final image size
func (s *PredictionService) PredictImageSize(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	prediction, err := s.modelTrainer.Predict(ctx, domain.PredictionTypeImageSize, features)
	if err != nil {
		return nil, err
	}

	prediction.WorkspaceID = workspaceID
	if err := s.predictionRepo.Create(ctx, prediction); err != nil {
		return nil, err
	}

	return prediction, nil
}

// PredictCacheHit predicts whether a build will have a cache hit
func (s *PredictionService) PredictCacheHit(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	prediction, err := s.modelTrainer.Predict(ctx, domain.PredictionTypeCacheHit, features)
	if err != nil {
		return nil, err
	}

	prediction.WorkspaceID = workspaceID
	if err := s.predictionRepo.Create(ctx, prediction); err != nil {
		return nil, err
	}

	return prediction, nil
}

// PredictFailureRisk predicts the risk of build failure
func (s *PredictionService) PredictFailureRisk(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	prediction, err := s.modelTrainer.Predict(ctx, domain.PredictionTypeFailureRisk, features)
	if err != nil {
		return nil, err
	}

	prediction.WorkspaceID = workspaceID
	if err := s.predictionRepo.Create(ctx, prediction); err != nil {
		return nil, err
	}

	return prediction, nil
}

// GetPredictionHistory returns prediction history for a workspace
func (s *PredictionService) GetPredictionHistory(ctx context.Context, workspaceID uuid.UUID, pType domain.PredictionType, limit, offset int) ([]*domain.Prediction, error) {
	return s.predictionRepo.ListByWorkspace(ctx, workspaceID, pType, limit, offset)
}

// GetModelMetrics returns metrics for the active model of a type
func (s *PredictionService) GetModelMetrics(ctx context.Context, pType domain.PredictionType) (map[string]float64, error) {
	model, err := s.modelTrainer.GetBestModel(ctx, pType)
	if err != nil {
		return nil, err
	}
	return model.Metrics, nil
}

// RetrainModel triggers retraining for a specific prediction type
func (s *PredictionService) RetrainModel(ctx context.Context, pType domain.PredictionType, algorithm string) (*domain.Model, error) {
	return s.modelTrainer.TrainModel(ctx, pType, algorithm)
}

// ExportModelONNX exports a model to ONNX format
func (s *PredictionService) ExportModelONNX(ctx context.Context, modelID uuid.UUID) (string, error) {
	return s.modelTrainer.ExportONNX(ctx, modelID)
}

// ActivateModel activates a model for predictions
func (s *PredictionService) ActivateModel(ctx context.Context, modelID uuid.UUID) error {
	return s.modelTrainer.ActivateModel(ctx, modelID)
}

// PrepareFeatures prepares feature map from detection result
func (s *PredictionService) PrepareFeatures(result any) map[string]any {
	analyzerResult, ok := result.(*analyzer.DetectionResult)
	if !ok || analyzerResult == nil {
		return map[string]any{}
	}

	profile := analyzerResult.RuntimeProfile
	features := map[string]any{
		"language":           profile.Language,
		"framework":          profile.Framework,
		"architecture":       analyzerResult.Architecture,
		"has_dockerfile":     profile.DockerfileSource == "USER_PROVIDED",
		"base_image":         profile.BaseImage,
		"exposed_port":       profile.ExposedPort,
		"cache_key":          analyzerResult.CacheKey,
		"num_lockfiles":      len(analyzerResult.Lockfiles),
		"has_compose":        analyzerResult.DockerCompose != nil,
		"entry_point_count":  len(analyzerResult.EntryPoints),
		"ambiguous_entry":    analyzerResult.AmbiguousEntry,
	}

	if analyzerResult.DockerCompose != nil {
		features["num_services"] = len(analyzerResult.DockerCompose.Services)
	}

	lockfileTypes := make([]string, len(analyzerResult.Lockfiles))
	for i, lf := range analyzerResult.Lockfiles {
		lockfileTypes[i] = lf.LockfileType
	}
	features["lockfile_types"] = lockfileTypes

	entryTypes := make([]string, len(analyzerResult.EntryPoints))
	for i, ep := range analyzerResult.EntryPoints {
		entryTypes[i] = ep.Type
	}
	features["entry_types"] = entryTypes

	features["framework_confidences"] = analyzerResult.FrameworkConf

	return features
}

// ScheduledRetraining runs periodic model retraining
func (s *PredictionService) ScheduledRetraining(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.modelTrainer.RetrainAllModels(ctx)
		}
	}
}