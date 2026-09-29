-- 000007: persist the repository URL on workspaces.
--
-- The git URL was previously only ever transmitted in the
-- berth.environment.create NATS payload and never written to the database.
-- That made the provisioning pipeline unrecoverable: a worker that missed
-- the message, or any restart, had no way to learn what to clone, and there
-- was no database-polling fallback for environments the way sandboxes had.
--
-- The workspace owns the checkout, so the URL belongs here rather than on
-- environments (an environment is a runnable instance of a workspace).

ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS git_url TEXT NOT NULL DEFAULT '';

-- Backfill from the legacy sandboxes table. Migration 000006 copied each
-- sandbox into a workspace reusing the same id, so the join is on id.
UPDATE workspaces w
SET git_url = s.git_url
FROM sandboxes s
WHERE s.id = w.id
  AND w.git_url = ''
  AND s.git_url IS NOT NULL
  AND s.git_url <> '';

-- Surface the incomplete rows rather than leaving them silently empty.
DO $$
DECLARE
    missing INTEGER;
BEGIN
    SELECT count(*) INTO missing FROM workspaces WHERE git_url = '';
    IF missing > 0 THEN
        RAISE NOTICE 'workspaces.git_url backfilled; % workspace(s) still have no git_url and cannot be provisioned', missing;
    END IF;
END $$;
