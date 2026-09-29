-- Migration: 000006_sandbox_to_environment_migration.up.sql
-- Migrates data from legacy sandboxes table to new workspaces + environments model

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- Step 1: Create RuntimeProfiles for existing sandboxes if needed
-- ============================================================
-- First, ensure we have runtime profiles for the languages used in sandboxes
-- We'll create default profiles for each language
--
-- detection_evidence is JSONB NOT NULL with no default (migrations/000004), so
-- it has to be supplied. This INSERT omitted it, which made the migration abort
-- with 'null value in column "detection_evidence"' on any database that already
-- had rows in sandboxes. Fresh databases only passed because the table was
-- empty.

INSERT INTO runtime_profiles (id, workspace_id, detection_evidence, language, version, framework, package_manager, architecture, entrypoint, build_command, start_command, port, dockerfile_source, dockerfile_content, requires_database, requires_redis, confidence, status, created_at, updated_at)
SELECT 
    uuid_generate_v4(),
    NULL, -- workspace_id will be set after workspace creation
    jsonb_build_object('migratedFrom', 'sandboxes', 'sandboxId', s.id, 'language', s.runtime_language),
    s.runtime_language,
    '',
    '',
    CASE 
        WHEN s.runtime_language = 'node' THEN 'npm'
        WHEN s.runtime_language = 'python' THEN 'pip'
        WHEN s.runtime_language = 'go' THEN 'go'
        WHEN s.runtime_language = 'rust' THEN 'cargo'
        ELSE ''
    END,
    'WEB_APP',
    '',
    CASE 
        WHEN s.runtime_language = 'node' THEN 'npm install'
        WHEN s.runtime_language = 'python' THEN 'pip install -r requirements.txt'
        WHEN s.runtime_language = 'go' THEN 'go mod download'
        WHEN s.runtime_language = 'rust' THEN 'cargo build'
        ELSE ''
    END,
    CASE 
        WHEN s.runtime_language = 'node' THEN 'npm run dev'
        WHEN s.runtime_language = 'python' THEN 'uvicorn main:app --host 0.0.0.0 --port 8000 --reload'
        WHEN s.runtime_language = 'go' THEN 'go run .'
        WHEN s.runtime_language = 'rust' THEN 'cargo run'
        ELSE ''
    END,
    s.runtime_port,
    '',
    '',
    false,
    false,
    1.0,
    'DETECTED',
    NOW(),
    NOW()
FROM sandboxes s
WHERE s.runtime_language IS NOT NULL AND s.runtime_language != ''
  AND NOT EXISTS (
    SELECT 1 FROM runtime_profiles rp 
    WHERE rp.language = s.runtime_language 
    AND rp.port = s.runtime_port
  )
ON CONFLICT DO NOTHING;

-- ============================================================
-- Step 2: Create Workspaces from Sandboxes
-- ============================================================
-- Each sandbox becomes a workspace with its git state
-- Use a default project for sandboxes without project_id

INSERT INTO workspaces (id, project_id, name, type, base_workspace_id, owner_id, git_branch, commit_hash, has_uncommitted_changes, last_synced_at, created_at, updated_at)
SELECT 
    s.id, -- Use same ID for workspace
    COALESCE(s.project_id, (SELECT id FROM projects LIMIT 1)),
    s.name,
    'CANONICAL',
    NULL,
    s.owner_id,
    s.git_branch,
    s.commit_hash,
    s.has_uncommitted_changes,
    s.last_modified_at,
    s.created_at,
    s.updated_at
FROM sandboxes s
WHERE NOT EXISTS (
    SELECT 1 FROM workspaces w WHERE w.id = s.id
);

-- ============================================================
-- Step 3: Link Workspaces to RuntimeProfiles
-- ============================================================
-- Update the runtime_profiles created in step 1 to link to the workspaces

UPDATE runtime_profiles rp
SET workspace_id = w.id
FROM workspaces w
JOIN sandboxes s ON s.id = w.id
WHERE rp.workspace_id IS NULL
  AND rp.language = s.runtime_language
  AND rp.port = s.runtime_port;

-- ============================================================
-- Step 4: Create Environments from Sandboxes
-- ============================================================
-- Each sandbox becomes an environment linked to its workspace

INSERT INTO environments (id, workspace_id, runtime_profile_id, name, state, container_id, image_id, public_url, port, memory_limit, cpu_limit, created_at, updated_at)
SELECT 
    s.id, -- Use same ID for environment
    w.id,
    rp.id,
    s.name,
    CASE s.state
        WHEN 'IDLE' THEN 'CREATED'
        WHEN 'PENDING' THEN 'CREATED'
        WHEN 'BUILDING' THEN 'BUILDING'
        WHEN 'RUNNING' THEN 'RUNNING'
        WHEN 'STOPPED' THEN 'STOPPED'
        WHEN 'FAILED' THEN 'BUILD_FAILED'
        ELSE 'CREATED'
    END,
    s.container_id,
    NULL, -- image_id - would need docker image tracking
    s.public_url,
    s.runtime_port,
    536870912, -- 512 MiB default
    1000000000, -- 1 CPU default
    s.created_at,
    s.updated_at
FROM sandboxes s
JOIN workspaces w ON w.id = s.id
LEFT JOIN runtime_profiles rp ON rp.workspace_id = w.id
WHERE NOT EXISTS (
    SELECT 1 FROM environments e WHERE e.id = s.id
);

-- ============================================================
-- Step 5: Migrate Sandbox Activities to Environment Events
-- ============================================================
INSERT INTO environment_events (id, environment_id, workspace_id, project_id, type, payload, created_at)
SELECT 
    sa.id,
    sa.sandbox_id,
    w.id,
    COALESCE(w.project_id, (SELECT id FROM projects LIMIT 1)),
    'ACTIVITY',
    jsonb_build_object('activity_type', sa.activity_type, 'data', sa.data),
    sa.created_at
FROM sandbox_activities sa
JOIN workspaces w ON w.id = sa.sandbox_id
WHERE EXISTS (SELECT 1 FROM environments e WHERE e.id = sa.sandbox_id);

-- ============================================================
-- Step 6: Migrate Sandbox Changes to Change Requests (partial)
-- ============================================================
-- Sandbox changes represent file changes, not exactly change requests
-- We'll skip this for now as the models don't map 1:1

-- ============================================================
-- Step 7: Migrate Sandbox Logs (keep for debugging)
-- ============================================================
-- sandbox_logs table references sandboxes, we'll keep it for now
-- but add a foreign key to environments for future queries

-- ============================================================
-- Step 8: Create indexes for new foreign keys
-- ============================================================
-- The foreign keys are already defined in the schema

-- ============================================================
-- Verification queries
-- ============================================================
-- SELECT COUNT(*) FROM sandboxes;
-- SELECT COUNT(*) FROM workspaces;
-- SELECT COUNT(*) FROM environments;
-- SELECT COUNT(*) FROM runtime_profiles;
-- SELECT COUNT(*) FROM environment_events;