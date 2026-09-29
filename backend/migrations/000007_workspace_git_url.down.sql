-- Revert 000007: drop workspaces.git_url.
--
-- Any environments provisioned from this column become unprovisionable again
-- (there is no other record of the repository), so take a backup first if the
-- column holds data you still need.

ALTER TABLE workspaces DROP COLUMN IF EXISTS git_url;
