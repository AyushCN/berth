DROP TABLE IF EXISTS sandbox_activities;
DROP TABLE IF EXISTS sandbox_changes;

ALTER TABLE sandboxes
DROP COLUMN commit_hash,
DROP COLUMN modified_by_user_id,
DROP COLUMN last_modified_at,
DROP COLUMN has_uncommitted_changes;
