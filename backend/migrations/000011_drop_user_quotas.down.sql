-- Restore the unenforced quota columns with their original defaults.
-- Safe: nothing reads them, so re-adding changes no behaviour.

ALTER TABLE users ADD COLUMN IF NOT EXISTS max_sandboxes INT DEFAULT 5;
ALTER TABLE users ADD COLUMN IF NOT EXISTS max_builds_per_hour INT DEFAULT 10;
