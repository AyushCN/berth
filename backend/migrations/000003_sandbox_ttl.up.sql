ALTER TABLE sandboxes
    ALTER COLUMN expires_at SET DEFAULT (NOW() + INTERVAL '24 hours');

UPDATE sandboxes
SET expires_at = created_at + INTERVAL '24 hours'
WHERE expires_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sandboxes_expires_at ON sandboxes (expires_at);
