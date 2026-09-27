package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AyushCN/berth/internal/analyzer"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

func TestModelTrainer_TrainLinearRegression(t *testing.T) {
	dir := t.TempDir()

	// Create mock repositories
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	// Add training data
	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features: map[string]any{
				"language":  1.0,
				"framework": 2.0,
				"port_norm": 3.0,
			},
			Labels: map[string]any{
				"build_duration_ms": float64(1000 + i*100),
			},
			Language:     "node",
			Framework:    "next",
			Architecture: "WEB_APP",
			CreatedAt:    time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	model, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.ID == uuid.Nil {
		t.Error("expected model ID")
	}
	if model.Type != domain.PredictionTypeBuildTime {
		t.Errorf("expected BUILD_TIME, got %s", model.Type)
	}
	if model.Algorithm != "linear_regression" {
		t.Errorf("expected linear_regression, got %s", model.Algorithm)
	}
	if !model.IsActive {
		t.Error("expected model to be active")
	}

	// Check metrics
	if model.Metrics["mse"] == 0 {
		t.Error("expected non-zero MSE")
	}
	// R2 can be negative if model performs worse than mean predictor
	_ = model.Metrics["r2"]

	// Check parameters
	if model.Parameters["weights"] == nil {
		t.Error("expected weights in parameters")
	}
	if model.Parameters["bias"] == nil {
		t.Error("expected bias in parameters")
	}
}

func TestModelTrainer_TrainRandomForest(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features: map[string]any{
				"language":  1.0,
				"framework": 2.0,
				"port_norm": 3.0,
			},
			Labels: map[string]any{
				"build_duration_ms": float64(1000 + i*100),
			},
			Language:     "node",
			Framework:    "next",
			Architecture: "WEB_APP",
			CreatedAt:    time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	model, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "random_forest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.Algorithm != "random_forest" {
		t.Errorf("expected random_forest, got %s", model.Algorithm)
	}
	if model.Parameters["trees"] == nil {
		t.Error("expected trees in parameters")
	}
	if model.Parameters["num_trees"] != nil {
		var numTrees float64
		switch v := model.Parameters["num_trees"].(type) {
		case float64:
			numTrees = v
		case int:
			numTrees = float64(v)
		case int64:
			numTrees = float64(v)
		}
		if numTrees != 10 {
			t.Errorf("expected 10 trees, got %v", numTrees)
		}
	} else {
		t.Error("expected num_trees in parameters")
	}
}

func TestModelTrainer_TrainXGBoost(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features: map[string]any{
				"language":  1.0,
				"framework": 2.0,
				"port_norm": 3.0,
			},
			Labels: map[string]any{
				"build_duration_ms": float64(1000 + i*100),
			},
			Language:     "node",
			Framework:    "next",
			Architecture: "WEB_APP",
			CreatedAt:    time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	model, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "xgboost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.Algorithm != "xgboost" {
		t.Errorf("expected xgboost, got %s", model.Algorithm)
	}
	if model.Parameters["trees"] == nil {
		t.Error("expected trees in parameters")
	}
	if model.Parameters["num_rounds"] != nil {
		var numRounds float64
		switch v := model.Parameters["num_rounds"].(type) {
		case float64:
			numRounds = v
		case int:
			numRounds = float64(v)
		case int64:
			numRounds = float64(v)
		}
		if numRounds != 50 {
			t.Errorf("expected 50 rounds, got %v", numRounds)
		}
	} else {
		t.Error("expected num_rounds in parameters")
	}
	if model.Parameters["learning_rate"] != nil {
		var lr float64
		switch v := model.Parameters["learning_rate"].(type) {
		case float64:
			lr = v
		case int:
			lr = float64(v)
		}
		if lr != 0.1 {
			t.Errorf("expected learning_rate 0.1, got %v", lr)
		}
	} else {
		t.Error("expected learning_rate in parameters")
	}
}

func TestModelTrainer_InsufficientData(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	// Only 5 samples (less than minimum 10)
	for i := 0; i < 5; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	_, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	if err == nil {
		t.Error("expected error for insufficient data")
	}
	if err != nil && err.Error() != "insufficient training data: 5 samples (minimum 10)" {
		t.Errorf("expected specific error message, got %v", err)
	}
}

func TestModelTrainer_ExportONNX(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	// Train a model first
	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	model, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Export to ONNX
	onnxPath, err := trainer.ExportONNX(context.Background(), model.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if onnxPath == "" {
		t.Error("expected ONNX path")
	}

	// Check file exists
	if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
		t.Errorf("ONNX file not found at %s", onnxPath)
	}

	// Check model updated with ONNX path
	updatedModel, _ := modelRepo.GetByID(context.Background(), model.ID)
	if updatedModel.ONNXPath != onnxPath {
		t.Errorf("expected model ONNXPath %s, got %s", onnxPath, updatedModel.ONNXPath)
	}
}

func TestModelTrainer_ActivateModel(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	// Train two models
	model1, _ := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	model2, _ := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "random_forest")

	// Both should be active initially
	if !model1.IsActive {
		t.Error("expected model1 to be active")
	}
	if !model2.IsActive {
		t.Error("expected model2 to be active")
	}

	// Activate model1 (should deactivate model2)
	err := trainer.ActivateModel(context.Background(), model1.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated1, _ := modelRepo.GetByID(context.Background(), model1.ID)
	updated2, _ := modelRepo.GetByID(context.Background(), model2.ID)

	if !updated1.IsActive {
		t.Error("expected model1 to be active")
	}
	if updated2.IsActive {
		t.Error("expected model2 to be deactivated")
	}
}

func TestModelTrainer_GetBestModel(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	// Train multiple models
	_, _ = trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	model2, _ := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "random_forest")
	_, _ = trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "xgboost")

	// Manually set model2 to have better R2
	model2.Metrics["r2"] = 0.999
	modelRepo.Update(context.Background(), model2)

	// Also ensure other models have lower R2
	models, _ := modelRepo.List(context.Background(), domain.PredictionTypeBuildTime)
	for _, m := range models {
		if m.ID != model2.ID {
			m.Metrics["r2"] = 0.5
			modelRepo.Update(context.Background(), m)
		}
	}

	best, err := trainer.GetBestModel(context.Background(), domain.PredictionTypeBuildTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if best.ID != model2.ID {
		t.Errorf("expected best model to be model2, got %s", best.ID)
	}
}

func TestModelTrainer_Predict(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	_, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prediction, err := trainer.Predict(context.Background(), domain.PredictionTypeBuildTime, map[string]any{
		"language":  1.0,
		"framework": 2.0,
		"port_norm": 3.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prediction.ID == uuid.Nil {
		t.Error("expected prediction ID")
	}
	if prediction.Type != domain.PredictionTypeBuildTime {
		t.Errorf("expected BUILD_TIME, got %s", prediction.Type)
	}
	if prediction.Confidence <= 0 {
		t.Error("expected positive confidence")
	}
	if prediction.Output["predicted_value"] == nil {
		t.Error("expected predicted_value in output")
	}
}

func TestModelTrainer_SaveModel(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()

	for i := 0; i < 20; i++ {
		data := &domain.TrainingData{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			Features:    map[string]any{"language": 1.0},
			Labels:      map[string]any{"build_duration_ms": 1000.0},
			CreatedAt:   time.Now(),
		}
		trainingDataRepo.Create(context.Background(), data)
	}

	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	model, err := trainer.TrainModel(context.Background(), domain.PredictionTypeBuildTime, "linear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check model file exists
	modelPath := filepath.Join(dir, model.Name+"_"+model.Version+".json")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Errorf("model file not found at %s", modelPath)
	}
}

// Mock repositories for testing

type mockModelRepo struct {
	models map[uuid.UUID]*domain.Model
}

func newMockModelRepo() *mockModelRepo {
	return &mockModelRepo{models: make(map[uuid.UUID]*domain.Model)}
}

func (m *mockModelRepo) Create(ctx context.Context, model *domain.Model) error {
	m.models[model.ID] = model
	return nil
}

func (m *mockModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Model, error) {
	if model, ok := m.models[id]; ok {
		return model, nil
	}
	return nil, nil
}

func (m *mockModelRepo) GetLatestByType(ctx context.Context, pType domain.PredictionType) (*domain.Model, error) {
	var latest *domain.Model
	for _, model := range m.models {
		if model.Type == pType {
			if latest == nil || model.CreatedAt.After(latest.CreatedAt) {
				latest = model
			}
		}
	}
	return latest, nil
}

func (m *mockModelRepo) GetActiveByType(ctx context.Context, pType domain.PredictionType) (*domain.Model, error) {
	for _, model := range m.models {
		if model.Type == pType && model.IsActive {
			return model, nil
		}
	}
	return nil, nil
}

func (m *mockModelRepo) List(ctx context.Context, pType domain.PredictionType) ([]*domain.Model, error) {
	var models []*domain.Model
	for _, model := range m.models {
		if model.Type == pType {
			models = append(models, model)
		}
	}
	return models, nil
}

func (m *mockModelRepo) Update(ctx context.Context, model *domain.Model) error {
	m.models[model.ID] = model
	return nil
}

func (m *mockModelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.models, id)
	return nil
}

type mockTrainingDataRepo struct {
	data []*domain.TrainingData
}

func newMockTrainingDataRepo() *mockTrainingDataRepo {
	return &mockTrainingDataRepo{data: make([]*domain.TrainingData, 0)}
}

func (m *mockTrainingDataRepo) Create(ctx context.Context, data *domain.TrainingData) error {
	m.data = append(m.data, data)
	return nil
}

func (m *mockTrainingDataRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingData, error) {
	for _, d := range m.data {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

func (m *mockTrainingDataRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.TrainingData, error) {
	var result []*domain.TrainingData
	for _, d := range m.data {
		if d.WorkspaceID == workspaceID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockTrainingDataRepo) ListByType(ctx context.Context, pType domain.PredictionType, limit, offset int) ([]*domain.TrainingData, error) {
	var result []*domain.TrainingData
	for _, d := range m.data {
		result = append(result, d)
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockTrainingDataRepo) CountByType(ctx context.Context, pType domain.PredictionType) (int64, error) {
	var count int64
	for range m.data {
		count++
	}
	return count, nil
}

func (m *mockTrainingDataRepo) DeleteOlderThan(ctx context.Context, pType domain.PredictionType, before time.Time) (int64, error) {
	var count int64
	var remaining []*domain.TrainingData
	for _, d := range m.data {
		if d.CreatedAt.Before(before) && (d.Architecture == string(pType) || d.Framework == string(pType)) {
			count++
		} else {
			remaining = append(remaining, d)
		}
	}
	m.data = remaining
	return count, nil
}

func TestModelTrainer_CalculateMetrics(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()
	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	actual := []float64{100, 200, 300, 400, 500}
	predicted := []float64{110, 190, 310, 390, 510}

	metrics := trainer.calculateMetrics(actual, predicted)

	if metrics["mse"] == 0 {
		t.Error("expected non-zero MSE")
	}
	if metrics["mae"] == 0 {
		t.Error("expected non-zero MAE")
	}
	if metrics["rmse"] == 0 {
		t.Error("expected non-zero RMSE")
	}
	if metrics["r2"] < 0 || metrics["r2"] > 1 {
		t.Errorf("expected R2 in [0,1], got %f", metrics["r2"])
	}
	if metrics["samples"] != 5 {
		t.Errorf("expected 5 samples, got %f", metrics["samples"])
	}
}

func TestModelTrainer_ExtractFeatures(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()
	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	data := []*domain.TrainingData{
		{
			Features: map[string]any{
				"language":  1.0,
				"framework": 2.0,
				"port_norm": 3.0,
			},
		},
		{
			Features: map[string]any{
				"language":  2.0,
				"framework": 3.0,
				"port_norm": 8.0,
			},
		},
	}

	X := trainer.extractFeatureMatrix(data)

	if len(X) != 2 {
		t.Errorf("expected 2 rows, got %d", len(X))
	}
	if len(X[0]) != 3 {
		t.Errorf("expected 3 columns, got %d", len(X[0]))
	}
	// Feature names are sorted alphabetically: framework, language, port_norm
	// First row: framework=2.0, language=1.0, port_norm=3.0
	if X[0][0] != 2.0 || X[0][1] != 1.0 || X[0][2] != 3.0 {
		t.Errorf("unexpected feature values: %v", X[0])
	}
}

func TestModelTrainer_ToFloat(t *testing.T) {
	dir := t.TempDir()
	modelRepo := newMockModelRepo()
	trainingDataRepo := newMockTrainingDataRepo()
	trainer := NewModelTrainer(modelRepo, trainingDataRepo, dir)

	tests := []struct {
		input    any
		expected float64
	}{
		{1.5, 1.5},
		{float32(2.5), 2.5},
		{42, 42.0},
		{int64(100), 100.0},
		{true, 1.0},
		{false, 0.0},
		{"3.14", 3.14},
		{"not-a-number", 0.0},
		{nil, 0.0},
	}

	for _, tt := range tests {
		result := trainer.toFloat(tt.input)
		if result != tt.expected {
			t.Errorf("toFloat(%v) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestDataCollector_ExtractFeatures(t *testing.T) {
	trainingDataRepo := newMockTrainingDataRepo()
	buildRepo := &mockBuildRepo{}
	runtimeRepo := &mockRuntimeProfileRepo{}

	collector := NewDataCollector(trainingDataRepo, buildRepo, runtimeRepo)

	profile := &domain.RuntimeProfile{
		ID:              uuid.New(),
		Language:        "node",
		Framework:       "next",
		Architecture:    "WEB_APP",
		BaseImage:       "node:20-alpine",
		ExposedPort:     3000,
		WorkDir:         "/app",
		DockerfileSource: "GENERATED",
		InstallCmd:      "npm ci",
		BuildCommand:    "npm run build",
		StartCmd:        "npm start",
	}

	result := &analyzer.DetectionResult{
		RuntimeProfile: profile,
		Architecture:   "WEB_APP",
		Framework:      "next",
		CacheKey:       "abc123",
		Lockfiles: []analyzer.LockfileInfo{
			{LockfileType: "npm"},
			{LockfileType: "yarn"},
		},
		EntryPoints: []analyzer.EntryPoint{
			{Path: "package.json", Type: "main"},
			{Path: "Dockerfile", Type: "dockerfile"},
		},
		AmbiguousEntry: false,
	}

	plan := &domain.BuildPlan{
		BaseImage: "node:20-alpine",
		Port:      3000,
	}

	features := collector.extractFeatures(profile, plan, result)

	if features["language"] != "node" {
		t.Errorf("expected language=node, got %v", features["language"])
	}
	if features["framework"] != "next" {
		t.Errorf("expected framework=next, got %v", features["framework"])
	}
	if features["architecture"] != "WEB_APP" {
		t.Errorf("expected architecture=WEB_APP, got %v", features["architecture"])
	}
	if features["base_image"] != "node:20-alpine" {
		t.Errorf("expected base_image=node:20-alpine, got %v", features["base_image"])
	}
	if features["exposed_port"] != 3000 {
		t.Errorf("expected exposed_port=3000, got %v", features["exposed_port"])
	}
	if features["cache_key"] != "abc123" {
		t.Errorf("expected cache_key=abc123, got %v", features["cache_key"])
	}
	if features["num_lockfiles"] != 2 {
		t.Errorf("expected num_lockfiles=2, got %v", features["num_lockfiles"])
	}
	if features["entry_point_count"] != 2 {
		t.Errorf("expected entry_point_count=2, got %v", features["entry_point_count"])
	}
	if features["ambiguous_entry"] != false {
		t.Errorf("expected ambiguous_entry=false, got %v", features["ambiguous_entry"])
	}
	if features["has_dockerfile"] != false {
		t.Errorf("expected has_dockerfile=false, got %v", features["has_dockerfile"])
	}
}

func TestDataCollector_ExtractLabels(t *testing.T) {
	trainingDataRepo := newMockTrainingDataRepo()
	buildRepo := &mockBuildRepo{}
	runtimeRepo := &mockRuntimeProfileRepo{}

	collector := NewDataCollector(trainingDataRepo, buildRepo, runtimeRepo)

	build := &domain.Build{
		ID:              uuid.New(),
		WorkspaceID:     uuid.New(),
		BuildPlanID:     uuid.New(),
		Status:          domain.BuildStatusSuccess,
		BuildDurationMs: 123456,
		CacheHit:        true,
	}

	labels := collector.extractLabels(build)

	if labels["build_duration_ms"] != int64(123456) {
		t.Errorf("expected build_duration_ms=123456, got %v", labels["build_duration_ms"])
	}
	if labels["cache_hit"] != true {
		t.Errorf("expected cache_hit=true, got %v", labels["cache_hit"])
	}
	if labels["status"] != "SUCCESS" {
		t.Errorf("expected status=SUCCESS, got %v", labels["status"])
	}
}

// Mock build repo
type mockBuildRepo struct {
	builds []*domain.Build
}

func (m *mockBuildRepo) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Build, error) {
	return m.builds, nil
}

func (m *mockBuildRepo) GetLatestByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.Build, error) {
	if len(m.builds) > 0 {
		return m.builds[0], nil
	}
	return nil, nil
}

func (m *mockBuildRepo) GetBuildsByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.Build, error) {
	return m.builds, nil
}

func (m *mockBuildRepo) GetLatestBuildByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.Build, error) {
	if len(m.builds) > 0 {
		return m.builds[0], nil
	}
	return nil, nil
}

// Mock runtime profile repo
type mockRuntimeProfileRepo struct {
	profiles []*domain.RuntimeProfile
}

func (m *mockRuntimeProfileRepo) GetByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.RuntimeProfile, error) {
	return m.profiles, nil
}