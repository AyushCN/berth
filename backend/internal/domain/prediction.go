package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PredictionType represents the type of prediction
type PredictionType string

const (
	PredictionTypeBuildTime     PredictionType = "BUILD_TIME"
	PredictionTypeImageSize     PredictionType = "IMAGE_SIZE"
	PredictionTypeCacheHit      PredictionType = "CACHE_HIT"
	PredictionTypeResourceUsage PredictionType = "RESOURCE_USAGE"
	PredictionTypeFailureRisk   PredictionType = "FAILURE_RISK"
)

// Prediction represents a model prediction
type Prediction struct {
	ID           uuid.UUID      `json:"id"`
	Type         PredictionType `json:"type"`
	WorkspaceID  uuid.UUID      `json:"workspace_id"`
	Input        map[string]any `json:"input"`
	Output       map[string]any `json:"output"`
	Confidence   float64        `json:"confidence"`
	ModelVersion string         `json:"model_version"`
	CreatedAt    time.Time      `json:"created_at"`
}

// TrainingData represents a single training data point
type TrainingData struct {
	ID           uuid.UUID      `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspace_id"`
	BuildID      *uuid.UUID     `json:"build_id,omitempty"`
	Features     map[string]any `json:"features"`
	Labels       map[string]any `json:"labels"`
	Architecture string         `json:"architecture"`
	Framework    string         `json:"framework"`
	Language     string         `json:"language"`
	CacheKey     string         `json:"cache_key"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Model represents a trained ML model
type Model struct {
	ID         uuid.UUID          `json:"id"`
	Name       string             `json:"name"`
	Version    string             `json:"version"`
	Type       PredictionType     `json:"type"`
	Algorithm  string             `json:"algorithm"`
	Parameters map[string]any     `json:"parameters"`
	Metrics    map[string]float64 `json:"metrics"`
	ONNXPath   string             `json:"onnx_path,omitempty"`
	IsActive   bool               `json:"is_active"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// ModelRepository defines the interface for model persistence
type ModelRepository interface {
	Create(ctx context.Context, model *Model) error
	GetByID(ctx context.Context, id uuid.UUID) (*Model, error)
	GetLatestByType(ctx context.Context, pType PredictionType) (*Model, error)
	GetActiveByType(ctx context.Context, pType PredictionType) (*Model, error)
	List(ctx context.Context, pType PredictionType) ([]*Model, error)
	Update(ctx context.Context, model *Model) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TrainingDataRepository defines the interface for training data persistence
type TrainingDataRepository interface {
	Create(ctx context.Context, data *TrainingData) error
	GetByID(ctx context.Context, id uuid.UUID) (*TrainingData, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*TrainingData, error)
	ListByType(ctx context.Context, pType PredictionType, limit, offset int) ([]*TrainingData, error)
	CountByType(ctx context.Context, pType PredictionType) (int64, error)
	DeleteOlderThan(ctx context.Context, pType PredictionType, before time.Time) (int64, error)
}

// PredictionRepository defines the interface for prediction persistence
type PredictionRepository interface {
	Create(ctx context.Context, prediction *Prediction) error
	GetByID(ctx context.Context, id uuid.UUID) (*Prediction, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, pType PredictionType, limit, offset int) ([]*Prediction, error)
}

// PredictionService defines the interface for prediction service
type PredictionService interface {
	PredictBuildTime(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*Prediction, error)
	PredictImageSize(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*Prediction, error)
	PredictCacheHit(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*Prediction, error)
	PredictFailureRisk(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*Prediction, error)
	GetPredictionHistory(ctx context.Context, workspaceID uuid.UUID, pType PredictionType, limit, offset int) ([]*Prediction, error)
	GetModelMetrics(ctx context.Context, pType PredictionType) (map[string]float64, error)
	RetrainModel(ctx context.Context, pType PredictionType, algorithm string) (*Model, error)
	ExportModelONNX(ctx context.Context, modelID uuid.UUID) (string, error)
	ActivateModel(ctx context.Context, modelID uuid.UUID) error
	PrepareFeatures(result any) map[string]any
}
