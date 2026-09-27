-- Migration: 000005_prediction_engine.up.sql
-- Adds: Predictions, Models, TrainingData tables for ML-based predictions

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- PREDICTIONS - Model predictions for builds
-- ============================================================
CREATE TABLE IF NOT EXISTS predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type TEXT NOT NULL, -- BUILD_TIME, IMAGE_SIZE, CACHE_HIT, RESOURCE_USAGE, FAILURE_RISK
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    input JSONB NOT NULL,
    output JSONB NOT NULL,
    confidence REAL NOT NULL DEFAULT 0,
    model_version TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_predictions_workspace ON predictions(workspace_id);
CREATE INDEX idx_predictions_type ON predictions(type);
CREATE INDEX idx_predictions_created ON predictions(created_at);

-- ============================================================
-- MODELS - Trained ML models
-- ============================================================
CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    type TEXT NOT NULL, -- BUILD_TIME, IMAGE_SIZE, CACHE_HIT, RESOURCE_USAGE, FAILURE_RISK
    algorithm TEXT NOT NULL, -- linear_regression, random_forest, xgboost
    parameters JSONB NOT NULL,
    metrics JSONB NOT NULL,
    onnx_path TEXT,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_models_type ON models(type);
CREATE INDEX idx_models_active ON models(type, is_active) WHERE is_active = TRUE;
CREATE UNIQUE INDEX idx_models_name_version ON models(name, version);

-- ============================================================
-- TRAINING DATA - Collected build data for model training
-- ============================================================
CREATE TABLE IF NOT EXISTS training_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    build_id UUID REFERENCES builds(id) ON DELETE SET NULL,
    features JSONB NOT NULL,
    labels JSONB NOT NULL,
    architecture TEXT, -- WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
    framework TEXT,
    language TEXT,
    cache_key TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_training_data_workspace ON training_data(workspace_id);
CREATE INDEX idx_training_data_type ON training_data(architecture, framework, language);
CREATE INDEX idx_training_data_created ON training_data(created_at);