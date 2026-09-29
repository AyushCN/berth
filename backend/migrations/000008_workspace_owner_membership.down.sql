-- Revert 000008.
--
-- Only removes membership rows that this migration inserted for a workspace
-- owner. Rows created by WorkspaceRepository.Create after the code fix are
-- indistinguishable from backfilled ones, so rolling this back can leave a
-- workspace without an owner membership row again.

DELETE FROM workspace_members wm
USING workspaces w
WHERE wm.workspace_id = w.id
  AND wm.user_id = w.owner_id
  AND wm.role = 'OWNER';
