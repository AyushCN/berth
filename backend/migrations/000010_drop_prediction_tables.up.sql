-- 000010: drop the prediction/ML tables.
--
-- The prediction engine was never reachable. BuildPlanner was only invoked
-- from tests, so no rows were ever written to builds/build_plans and the data
-- collector never ran, which meant ModelTrainer could never accumulate the
-- samples it requires and every /api/predictions/* endpoint failed with "no
-- models found for type ...".
--
-- The Go code for it (ONNX inference, model trainer, prediction service, build
-- planner and the per-language build strategies) has been removed, along with
-- the gRPC server, the protobufs and the HTTP routes.
--
-- These tables are dropped so the schema does not keep implying a feature that
-- no longer exists. Recreating them is a straight replay of migrations
-- 000005_prediction_engine.up.sql if the feature returns.

-- Order matters: builds has a foreign key to build_plans, so it must go first.
-- Dropping build_plans on its own fails with "cannot drop table build_plans
-- because other objects depend on it".
DROP TABLE IF EXISTS predictions;
DROP TABLE IF EXISTS training_data;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS builds;
DROP TABLE IF EXISTS build_plans;
