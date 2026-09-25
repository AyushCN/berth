DROP INDEX IF EXISTS idx_sandboxes_expires_at;
ALTER TABLE sandboxes ALTER COLUMN expires_at DROP DEFAULT;
