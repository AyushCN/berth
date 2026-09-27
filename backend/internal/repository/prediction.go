package repository

import (
	"context"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PredictionRepository struct {
	db *pgxpool.Pool
}

func NewPredictionRepository(db *pgxpool.Pool) *PredictionRepository {
	return &PredictionRepository{db: db}
}

func (r *PredictionRepository) Create(ctx context.Context, prediction *domain.Prediction) error {
	query := `
		INSERT INTO predictions (id, type, workspace_id, input, output, confidence, model_version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		prediction.ID,
		prediction.Type,
		prediction.WorkspaceID,
		prediction.Input,
		prediction.Output,
		prediction.Confidence,
		prediction.ModelVersion,
		prediction.CreatedAt,
	)
	return err
}

func (r *PredictionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Prediction, error) {
	query := `
		SELECT id, type, workspace_id, input, output, confidence, model_version, created_at
		FROM predictions
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	return r.scanPrediction(row)
}

func (r *PredictionRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, pType domain.PredictionType, limit, offset int) ([]*domain.Prediction, error) {
	query := `
		SELECT id, type, workspace_id, input, output, confidence, model_version, created_at
		FROM predictions
		WHERE workspace_id = $1 AND type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, workspaceID, pType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var predictions []*domain.Prediction
	for rows.Next() {
		p, err := r.scanPrediction(rows)
		if err != nil {
			return nil, err
		}
		predictions = append(predictions, p)
	}
	return predictions, rows.Err()
}

func (r *PredictionRepository) scanPrediction(scanner interface {
	Scan(dest ...any) error
}) (*domain.Prediction, error) {
	var p domain.Prediction
	err := scanner.Scan(
		&p.ID,
		&p.Type,
		&p.WorkspaceID,
		&p.Input,
		&p.Output,
		&p.Confidence,
		&p.ModelVersion,
		&p.CreatedAt,
	)
	return &p, err
}

type ModelRepository struct {
	db *pgxpool.Pool
}

func NewModelRepository(db *pgxpool.Pool) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) Create(ctx context.Context, model *domain.Model) error {
	query := `
		INSERT INTO models (id, name, version, type, algorithm, parameters, metrics, onnx_path, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Version,
		model.Type,
		model.Algorithm,
		model.Parameters,
		model.Metrics,
		model.ONNXPath,
		model.IsActive,
		model.CreatedAt,
		model.UpdatedAt,
	)
	return err
}

func (r *ModelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Model, error) {
	query := `
		SELECT id, name, version, type, algorithm, parameters, metrics, onnx_path, is_active, created_at, updated_at
		FROM models
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	return r.scanModel(row)
}

func (r *ModelRepository) GetLatestByType(ctx context.Context, pType domain.PredictionType) (*domain.Model, error) {
	query := `
		SELECT id, name, version, type, algorithm, parameters, metrics, onnx_path, is_active, created_at, updated_at
		FROM models
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, query, pType)
	return r.scanModel(row)
}

func (r *ModelRepository) GetActiveByType(ctx context.Context, pType domain.PredictionType) (*domain.Model, error) {
	query := `
		SELECT id, name, version, type, algorithm, parameters, metrics, onnx_path, is_active, created_at, updated_at
		FROM models
		WHERE type = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, query, pType)
	return r.scanModel(row)
}

func (r *ModelRepository) List(ctx context.Context, pType domain.PredictionType) ([]*domain.Model, error) {
	query := `
		SELECT id, name, version, type, algorithm, parameters, metrics, onnx_path, is_active, created_at, updated_at
		FROM models
		WHERE type = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, pType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*domain.Model
	for rows.Next() {
		m, err := r.scanModel(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

func (r *ModelRepository) Update(ctx context.Context, model *domain.Model) error {
	query := `
		UPDATE models
		SET name = $2, version = $3, type = $4, algorithm = $5, parameters = $6,
			metrics = $7, onnx_path = $8, is_active = $9, updated_at = $10
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		model.ID,
		model.Name,
		model.Version,
		model.Type,
		model.Algorithm,
		model.Parameters,
		model.Metrics,
		model.ONNXPath,
		model.IsActive,
		model.UpdatedAt,
	)
	return err
}

func (r *ModelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM models WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *ModelRepository) scanModel(scanner interface {
	Scan(dest ...any) error
}) (*domain.Model, error) {
	var m domain.Model
	err := scanner.Scan(
		&m.ID,
		&m.Name,
		&m.Version,
		&m.Type,
		&m.Algorithm,
		&m.Parameters,
		&m.Metrics,
		&m.ONNXPath,
		&m.IsActive,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	return &m, err
}

type TrainingDataRepository struct {
	db *pgxpool.Pool
}

func NewTrainingDataRepository(db *pgxpool.Pool) *TrainingDataRepository {
	return &TrainingDataRepository{db: db}
}

func (r *TrainingDataRepository) Create(ctx context.Context, data *domain.TrainingData) error {
	query := `
		INSERT INTO training_data (id, workspace_id, build_id, features, labels, architecture, framework, language, cache_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		data.ID,
		data.WorkspaceID,
		data.BuildID,
		data.Features,
		data.Labels,
		data.Architecture,
		data.Framework,
		data.Language,
		data.CacheKey,
		data.CreatedAt,
	)
	return err
}

func (r *TrainingDataRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingData, error) {
	query := `
		SELECT id, workspace_id, build_id, features, labels, architecture, framework, language, cache_key, created_at
		FROM training_data
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	return r.scanTrainingData(row)
}

func (r *TrainingDataRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.TrainingData, error) {
	query := `
		SELECT id, workspace_id, build_id, features, labels, architecture, framework, language, cache_key, created_at
		FROM training_data
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*domain.TrainingData
	for rows.Next() {
		d, err := r.scanTrainingData(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *TrainingDataRepository) ListByType(ctx context.Context, pType domain.PredictionType, limit, offset int) ([]*domain.TrainingData, error) {
	query := `
		SELECT id, workspace_id, build_id, features, labels, architecture, framework, language, cache_key, created_at
		FROM training_data
		WHERE (labels->>'type') = $1 OR architecture = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, string(pType), string(pType), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*domain.TrainingData
	for rows.Next() {
		d, err := r.scanTrainingData(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *TrainingDataRepository) CountByType(ctx context.Context, pType domain.PredictionType) (int64, error) {
	query := `
		SELECT COUNT(*) FROM training_data
		WHERE (labels->>'type') = $1 OR architecture = $2
	`
	var count int64
	err := r.db.QueryRow(ctx, query, string(pType), string(pType)).Scan(&count)
	return count, err
}

func (r *TrainingDataRepository) DeleteOlderThan(ctx context.Context, pType domain.PredictionType, before time.Time) (int64, error) {
	query := `
		DELETE FROM training_data
		WHERE created_at < $1 AND ((labels->>'type') = $2 OR architecture = $3)
	`
	result, err := r.db.Exec(ctx, query, before, string(pType), string(pType))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), err
}

func (r *TrainingDataRepository) scanTrainingData(scanner interface {
	Scan(dest ...any) error
}) (*domain.TrainingData, error) {
	var d domain.TrainingData
	err := scanner.Scan(
		&d.ID,
		&d.WorkspaceID,
		&d.BuildID,
		&d.Features,
		&d.Labels,
		&d.Architecture,
		&d.Framework,
		&d.Language,
		&d.CacheKey,
		&d.CreatedAt,
	)
	return &d, err
}