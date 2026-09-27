-- Migration: 000005_prediction_engine.down.sql
-- Reverts: Predictions, Models, TrainingData tables

DROP TABLE IF EXISTS training_data;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS predictions;