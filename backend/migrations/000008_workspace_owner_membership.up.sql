-- 000008: backfill workspace_members for workspace owners.
--
-- WorkspaceRepository.Create did not insert the owner's workspace_members
-- row. GetUserWorkspaces (and therefore EnvironmentUsecase.ListEnvironments)
-- resolves access through workspace_members, not workspaces.owner_id, so every
-- workspace created through the API was invisible to its own owner and any
-- environment inside it never appeared in the list.
--
-- The repository now inserts the row on create. This backfills the rows that
-- were missed.

INSERT INTO workspace_members (workspace_id, user_id, role)
SELECT w.id, w.owner_id, 'OWNER'
FROM workspaces w
WHERE w.deleted_at IS NULL
  AND w.owner_id IS NOT NULL
  ON CONFLICT (workspace_id, user_id) DO NOTHING;

DO $$
DECLARE
    orphans INTEGER;
BEGIN
    SELECT count(*) INTO orphans
    FROM workspaces w
    WHERE w.deleted_at IS NULL
      AND NOT EXISTS (
          SELECT 1 FROM workspace_members wm
          WHERE wm.workspace_id = w.id AND wm.user_id = w.owner_id
      );

    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 000008: % workspace(s) still have no owner membership row', orphans;
    END IF;
END $$;
