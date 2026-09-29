-- Reverting 000009 would require recreating the legacy sandbox tables and
-- backfilling them from workspaces + environments. There is no automated way to
-- do that: the two models never had a lossless correspondence (sandboxes
-- carried git_url, git_branch, expires_at and a separate state machine, while
-- environments carry a workspace_id and a 12-value state enum), and the worker
-- no longer writes the legacy columns.
--
-- Restore from a backup taken before migration 000009 instead.

DO $$
BEGIN
    RAISE EXCEPTION 'migration 000009 is not reversible: the legacy sandboxes model was removed. Restore from a pre-000009 backup.';
END $$;
