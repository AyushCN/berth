-- Recreate the prediction tables by replaying migrations
-- 000005_prediction_engine.up.sql, which is the authoritative definition.
--
-- The rows are not restored: the feature never worked, so the tables were empty
-- or held only build metadata.

\i 000005_prediction_engine.up.sql
