-- Migration: 000006_sandbox_to_environment_migration.down.sql
-- Reverts the sandbox to environment migration
--
-- This previously referenced a `metadata` column on environment_events, which
-- does not exist: the column is `payload` (migrations/000004). Rolling back
-- therefore failed with 'column "metadata" does not exist', so the migration
-- could never be undone.

-- Delete environment events migrated from sandbox activities
DELETE FROM environment_events
WHERE type = 'ACTIVITY'
  AND payload ? 'details';

-- Delete environments created from sandboxes
DELETE FROM environments
WHERE id IN (SELECT id FROM sandboxes);

-- Delete runtime profiles created for sandboxes
DELETE FROM runtime_profiles
WHERE workspace_id IN (SELECT id FROM workspaces WHERE id IN (SELECT id FROM sandboxes));

-- Delete workspaces created from sandboxes
DELETE FROM workspaces
WHERE id IN (SELECT id FROM sandboxes);
