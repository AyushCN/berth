-- 000009: drop the legacy `sandboxes` model.
--
-- Migration 000006 copied sandboxes into workspaces + environments but was a
-- data copy, not a rename, so both models were live at once. The worker drove
-- sandboxes while the api wrote environments, which meant:
--
--   - berth.environment.create had no subscriber, so every environment created
--     over HTTP sat in CREATED forever
--   - stop and delete published messages nobody consumed, so the api returned
--     204 while the container kept running
--   - cleanupExpired deleted sandboxes rows on a 60s timer, including rows for
--     environments that were still live
--
-- The worker now consumes berth.environment.* and environments are the only
-- runnable model, so the table and its satellites go away.
--
-- environments is the surviving record: migration 000006 reused each sandbox id
-- as the workspace and environment id, and the worker has been writing
-- container_id, port and public_url to environments since then.

DROP TABLE IF EXISTS sandbox_activities;
DROP TABLE IF EXISTS sandbox_changes;
DROP TABLE IF EXISTS sandbox_logs;
DROP TABLE IF EXISTS sandboxes;

-- audit_logs was never read or written by any code path and has no domain
-- type, repository or handler.
DROP TABLE IF EXISTS audit_logs;
