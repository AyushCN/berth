package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// ModelTrainer trains ML models from collected training data
type ModelTrainer struct {
	modelRepo       domain.ModelRepository
	trainingDataRepo domain.TrainingDataRepository
	modelDir        string
}

func NewModelTrainer(
	modelRepo domain.ModelRepository,
	trainingDataRepo domain.TrainingDataRepository,
	modelDir string,
) *ModelTrainer {
	return &ModelTrainer{
		modelRepo:        modelRepo,
		trainingDataRepo: trainingDataRepo,
		modelDir:         modelDir,
	}
}

// TrainModel trains a model for a specific prediction type
func (t *ModelTrainer) TrainModel(ctx context.Context, pType domain.PredictionType, algorithm string) (*domain.Model, error) {
	// Get training data
	data, err := t.trainingDataRepo.ListByType(ctx, pType, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get training data: %w", err)
	}

	if len(data) < 10 {
		return nil, fmt.Errorf("insufficient training data: %d samples (minimum 10)", len(data))
	}

	// Train based on algorithm
	var model *domain.Model
	switch algorithm {
	case "linear":
		model = t.trainLinearRegression(pType, data)
	case "random_forest":
		model = t.trainRandomForest(pType, data)
	case "xgboost":
		model = t.trainXGBoost(pType, data)
	default:
		model = t.trainLinearRegression(pType, data) // Default
	}

	model.CreatedAt = time.Now()
	model.UpdatedAt = time.Now()

	// Save model to disk
	if err := t.saveModel(model); err != nil {
		return nil, fmt.Errorf("failed to save model: %w", err)
	}

	// Persist to database
	if err := t.modelRepo.Create(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to persist model: %w", err)
	}

	return model, nil
}

func (t *ModelTrainer) trainLinearRegression(pType domain.PredictionType, data []*domain.TrainingData) *domain.Model {
	// Simple linear regression implementation
	// In production, this would use a proper ML library like Gonum or golearn
	
	// Extract features and labels
	X := t.extractFeatureMatrix(data)
	y := t.extractLabels(data, pType)

	// Simple gradient descent for linear regression
	weights := make([]float64, len(X[0]))
	bias := 0.0
	learningRate := 0.01
	epochs := 1000

	for epoch := 0; epoch < epochs; epoch++ {
		for i := range X {
			prediction := bias
			for j := range X[i] {
				prediction += weights[j] * X[i][j]
			}
			error := y[i] - prediction
			bias += learningRate * error
			for j := range weights {
				weights[j] += learningRate * error * X[i][j]
			}
		}
	}

	// Calculate metrics
	predictions := make([]float64, len(X))
	for i := range X {
		pred := bias
		for j := range X[i] {
			pred += weights[j] * X[i][j]
		}
		predictions[i] = pred
	}

	metrics := t.calculateMetrics(y, predictions)

	return &domain.Model{
		ID:         uuid.New(),
		Name:       fmt.Sprintf("%s_linear_regression", pType),
		Version:    fmt.Sprintf("v%d", time.Now().Unix()),
		Type:       pType,
		Algorithm:  "linear_regression",
		Parameters: map[string]any{
			"weights": weights,
			"bias":    bias,
			"features": t.getFeatureNames(data),
		},
		Metrics:   metrics,
		IsActive:  true,
	}
}

func (t *ModelTrainer) trainRandomForest(pType domain.PredictionType, data []*domain.TrainingData) *domain.Model {
	// Simplified random forest - in production use a proper implementation
	numTrees := 10
	maxDepth := 5

	trees := make([]map[string]any, numTrees)
	for i := 0; i < numTrees; i++ {
		trees[i] = t.buildDecisionTree(data, pType, maxDepth)
	}

	metrics := t.evaluateEnsemble(data, pType, trees)

	return &domain.Model{
		ID:         uuid.New(),
		Name:       fmt.Sprintf("%s_random_forest", pType),
		Version:    fmt.Sprintf("v%d", time.Now().Unix()),
		Type:       pType,
		Algorithm:  "random_forest",
		Parameters: map[string]any{
			"trees":      trees,
			"num_trees":  numTrees,
			"max_depth":  maxDepth,
			"features":   t.getFeatureNames(data),
		},
		Metrics:   metrics,
		IsActive:  true,
	}
}

func (t *ModelTrainer) trainXGBoost(pType domain.PredictionType, data []*domain.TrainingData) *domain.Model {
	// Simplified XGBoost-like gradient boosting
	numRounds := 50
	learningRate := 0.1

	trees := make([]map[string]any, numRounds)
	residuals := t.extractLabels(data, pType)

	for i := 0; i < numRounds; i++ {
		tree := t.buildDecisionTreeForResiduals(data, residuals, 3)
		trees[i] = tree

		// Update residuals
		predictions := t.predictEnsemble(data, trees[:i+1])
		for j := range residuals {
			residuals[j] -= learningRate * predictions[j]
		}
	}

	metrics := t.evaluateEnsemble(data, pType, trees)

	return &domain.Model{
		ID:         uuid.New(),
		Name:       fmt.Sprintf("%s_xgboost", pType),
		Version:    fmt.Sprintf("v%d", time.Now().Unix()),
		Type:       pType,
		Algorithm:  "xgboost",
		Parameters: map[string]any{
			"trees":         trees,
			"num_rounds":    numRounds,
			"learning_rate": learningRate,
			"features":      t.getFeatureNames(data),
		},
		Metrics:   metrics,
		IsActive:  true,
	}
}

func (t *ModelTrainer) buildDecisionTree(data []*domain.TrainingData, pType domain.PredictionType, maxDepth int) map[string]any {
	// Simplified decision tree - returns a basic structure
	labels := t.extractLabels(data, pType)
	mean := 0.0
	for _, v := range labels {
		mean += v
	}
	mean /= float64(len(labels))

	return map[string]any{
		"type":  "leaf",
		"value": mean,
	}
}

func (t *ModelTrainer) buildDecisionTreeForResiduals(data []*domain.TrainingData, residuals []float64, maxDepth int) map[string]any {
	mean := 0.0
	for _, v := range residuals {
		mean += v
	}
	mean /= float64(len(residuals))

	return map[string]any{
		"type":  "leaf",
		"value": mean,
	}
}

func (t *ModelTrainer) evaluateEnsemble(data []*domain.TrainingData, pType domain.PredictionType, trees []map[string]any) map[string]float64 {
	labels := t.extractLabels(data, pType)
	predictions := t.predictEnsemble(data, trees)
	return t.calculateMetrics(labels, predictions)
}

func (t *ModelTrainer) predictEnsemble(data []*domain.TrainingData, trees []map[string]any) []float64 {
	predictions := make([]float64, len(data))
	for i := range data {
		sum := 0.0
		for _, tree := range trees {
			if val, ok := tree["value"].(float64); ok {
				sum += val
			}
		}
		predictions[i] = sum / float64(len(trees))
	}
	return predictions
}

func (t *ModelTrainer) calculateMetrics(actual, predicted []float64) map[string]float64 {
	if len(actual) != len(predicted) || len(actual) == 0 {
		return map[string]float64{}
	}

	mse := 0.0
	mae := 0.0
	ssRes := 0.0
	ssTot := 0.0
	meanActual := 0.0

	for _, v := range actual {
		meanActual += v
	}
	meanActual /= float64(len(actual))

	for i := range actual {
		diff := actual[i] - predicted[i]
		mse += diff * diff
		mae += math.Abs(diff)
		ssRes += diff * diff
		ssTot += (actual[i] - meanActual) * (actual[i] - meanActual)
	}

	mse /= float64(len(actual))
	mae /= float64(len(actual))
	var r2 float64
	if ssTot == 0 {
		r2 = 1.0 // Perfect fit when all actual values are the same
	} else {
		r2 = 1.0 - (ssRes / ssTot)
	}

	return map[string]float64{
		"mse":   mse,
		"mae":   mae,
		"rmse":  math.Sqrt(mse),
		"r2":    r2,
		"samples": float64(len(actual)),
	}
}

func (t *ModelTrainer) extractFeatureMatrix(data []*domain.TrainingData) [][]float64 {
	// Convert features to numeric matrix
	featureNames := t.getFeatureNames(data)
	X := make([][]float64, len(data))

	for i, d := range data {
		row := make([]float64, len(featureNames))
		for j, name := range featureNames {
			if val, ok := d.Features[name]; ok {
				row[j] = t.toFloat(val)
			}
		}
		X[i] = row
	}

	return X
}

func (t *ModelTrainer) getFeatureNames(data []*domain.TrainingData) []string {
	featureSet := make(map[string]bool)
	for _, d := range data {
		for k := range d.Features {
			featureSet[k] = true
		}
	}

	names := make([]string, 0, len(featureSet))
	for k := range featureSet {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (t *ModelTrainer) extractLabels(data []*domain.TrainingData, pType domain.PredictionType) []float64 {
	labels := make([]float64, len(data))
	labelKeys := t.getLabelKeys(pType)

	for i, d := range data {
		for _, key := range labelKeys {
			if val, ok := d.Labels[key]; ok {
				labels[i] = t.toFloat(val)
				break
			}
		}
	}

	return labels
}

func (t *ModelTrainer) getLabelKeys(pType domain.PredictionType) []string {
	switch pType {
	case domain.PredictionTypeBuildTime:
		return []string{"build_duration_ms"}
	case domain.PredictionTypeImageSize:
		return []string{"image_size_bytes", "size_bytes"}
	case domain.PredictionTypeCacheHit:
		return []string{"cache_hit"}
	case domain.PredictionTypeResourceUsage:
		return []string{"cpu_usage", "memory_usage"}
	case domain.PredictionTypeFailureRisk:
		return []string{"failure_risk", "status"}
	default:
		return []string{}
	}
}

func (t *ModelTrainer) toFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	case string:
		// Try to parse numeric strings
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0.0
	}
}

func (t *ModelTrainer) saveModel(model *domain.Model) error {
	if err := os.MkdirAll(t.modelDir, 0755); err != nil {
		return err
	}

	modelPath := filepath.Join(t.modelDir, fmt.Sprintf("%s_%s.json", model.Name, model.Version))
	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(modelPath, data, 0644)
}

// ExportONNX exports a model to ONNX format
func (t *ModelTrainer) ExportONNX(ctx context.Context, modelID uuid.UUID) (string, error) {
	model, err := t.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		return "", fmt.Errorf("model not found: %w", err)
	}

	// In production, this would convert the model to ONNX format
	// For now, we'll create a mock ONNX file
	onnxPath := filepath.Join(t.modelDir, fmt.Sprintf("%s_%s.onnx", model.Name, model.Version))
	
	// Create a simple ONNX-like structure (placeholder)
	onnxContent := fmt.Sprintf(`{
  "ir_version": 8,
  "producer_name": "berth-prediction-engine",
  "producer_version": "1.0.0",
  "model_version": "%s",
  "doc_string": "Model for %s prediction",
  "graph": {
    "name": "%s",
    "input": [],
    "output": [],
    "node": [],
    "initializer": []
  }
}`, model.Version, model.Type, model.Name)

	if err := os.WriteFile(onnxPath, []byte(onnxContent), 0644); err != nil {
		return "", err
	}

	// Update model with ONNX path
	model.ONNXPath = onnxPath
	model.UpdatedAt = time.Now()
	if err := t.modelRepo.Update(ctx, model); err != nil {
		return "", err
	}

	return onnxPath, nil
}

// ActivateModel sets a model as the active model for its type
func (t *ModelTrainer) ActivateModel(ctx context.Context, modelID uuid.UUID) error {
	model, err := t.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		return err
	}

	// Deactivate other models of same type
	models, err := t.modelRepo.List(ctx, model.Type)
	if err != nil {
		return err
	}

	for _, m := range models {
		if m.ID != modelID && m.IsActive {
			m.IsActive = false
			if err := t.modelRepo.Update(ctx, m); err != nil {
				return err
			}
		}
	}

	model.IsActive = true
	model.UpdatedAt = time.Now()
	return t.modelRepo.Update(ctx, model)
}

// GetModelByID returns a model by ID
func (t *ModelTrainer) GetModelByID(ctx context.Context, modelID uuid.UUID) (*domain.Model, error) {
	return t.modelRepo.GetByID(ctx, modelID)
}

// GetBestModel returns the best model for a prediction type based on metrics
func (t *ModelTrainer) GetBestModel(ctx context.Context, pType domain.PredictionType) (*domain.Model, error) {
	models, err := t.modelRepo.List(ctx, pType)
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found for type %s", pType)
	}

	// Sort by R2 score (higher is better)
	sort.Slice(models, func(i, j int) bool {
		r2i := models[i].Metrics["r2"]
		r2j := models[j].Metrics["r2"]
		return r2i > r2j
	})

	return models[0], nil
}

// RetrainAllModels retrains models for all prediction types
func (t *ModelTrainer) RetrainAllModels(ctx context.Context) error {
	types := []domain.PredictionType{
		domain.PredictionTypeBuildTime,
		domain.PredictionTypeImageSize,
		domain.PredictionTypeCacheHit,
		domain.PredictionTypeResourceUsage,
		domain.PredictionTypeFailureRisk,
	}

	algorithms := []string{"linear", "random_forest", "xgboost"}

	for _, pType := range types {
		for _, algo := range algorithms {
			_, err := t.TrainModel(ctx, pType, algo)
			if err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}

// Mock prediction function for testing
func (t *ModelTrainer) Predict(ctx context.Context, pType domain.PredictionType, features map[string]any) (*domain.Prediction, error) {
	model, err := t.GetBestModel(ctx, pType)
	if err != nil {
		return nil, err
	}

	// Simple mock prediction
	output := map[string]any{
		"predicted_value": rand.Float64() * 1000,
		"model_version":   model.Version,
	}

	return &domain.Prediction{
		ID:           uuid.New(),
		Type:         pType,
		Input:        features,
		Output:       output,
		Confidence:   0.85,
		ModelVersion: model.Version,
		CreatedAt:    time.Now(),
	}, nil
}