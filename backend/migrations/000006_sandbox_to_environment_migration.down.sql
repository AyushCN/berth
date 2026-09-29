-- Migration: 000006_sandbox_to_environment_migration.down.sql
-- Reverts the sandbox to environment migration

-- Delete environment events created from sandbox activities
DELETE FROM environment_events 
WHERE type = 'ACTIVITY' 
AND metadata ? 'details';

-- Delete environments created from sandboxes
DELETE FROM environments 
WHERE id IN (SELECT id FROM sandboxes);

-- Delete runtime profiles created for sandboxes
DELETE FROM runtime_profiles 
WHERE workspace_id IN (SELECT id FROM workspaces WHERE id IN (SELECT id FROM sandboxes));

-- Delete workspaces created from sandboxes
DELETE FROM workspaces 
WHERE id IN (SELECT id FROM sandboxes);