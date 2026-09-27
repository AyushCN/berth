-- Migration: 000004_collaborative_platform.up.sql
-- Transforms Berth from single-user sandbox to collaborative platform
-- Adds: Workspaces, Forks, ShareLinks, RuntimeProfiles, BuildPlans, Builds, Images, Services, ChangeRequests

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- SHARE LINKS - Project invitation via link
-- ============================================================
CREATE TABLE IF NOT EXISTS share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    code TEXT UNIQUE NOT NULL, -- short invite code
    role TEXT NOT NULL DEFAULT 'EDITOR', -- VIEWER, EDITOR
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    max_uses INT,
    uses_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_share_links_project ON share_links(project_id);
CREATE INDEX idx_share_links_code ON share_links(code);

-- ============================================================
-- RUNTIME PROFILES - Detected application specification
-- ============================================================
CREATE TABLE IF NOT EXISTS runtime_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    workspace_id UUID, -- nullable, FK added after workspaces table
    detection_evidence JSONB NOT NULL, -- {language, framework, packageManager, files, confidence}
    language TEXT NOT NULL,
    version TEXT,
    framework TEXT,
    package_manager TEXT,
    architecture TEXT, -- WEB_APP, API, CLI, FULL_STACK, MONOREPO, LIBRARY
    entrypoint TEXT,
    build_command TEXT,
    start_command TEXT,
    port INT,
    dockerfile_source TEXT, -- USER_PROVIDED, GENERATED, COMPOSE
    dockerfile_content TEXT, -- if GENERATED
    requires_database BOOLEAN DEFAULT FALSE,
    requires_redis BOOLEAN DEFAULT FALSE,
    confidence REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'DETECTED', -- DETECTED, CONFIRMED, OVERRIDDEN
    confirmed_by UUID REFERENCES users(id),
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_runtime_profiles_project ON runtime_profiles(project_id);
CREATE INDEX idx_runtime_profiles_workspace ON runtime_profiles(workspace_id);

-- ============================================================
-- WORKSPACES - Editable code state (canonical + forks)
-- ============================================================
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g., "canonical", "alice-fork"
    type TEXT NOT NULL DEFAULT 'FORK', -- CANONICAL, FORK
    base_workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL, -- for forks
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- workspace owner
    git_branch TEXT NOT NULL DEFAULT 'main',
    commit_hash TEXT, -- last synced commit
    has_uncommitted_changes BOOLEAN DEFAULT FALSE,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_workspaces_project ON workspaces(project_id);
CREATE INDEX idx_workspaces_base ON workspaces(base_workspace_id);
CREATE INDEX idx_workspaces_owner ON workspaces(owner_id);

-- Workspace members - who can access this workspace
CREATE TABLE IF NOT EXISTS workspace_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'EDITOR', -- OWNER, EDITOR, VIEWER
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX idx_workspace_members_user ON workspace_members(user_id);

-- Add FK from runtime_profiles to workspaces
ALTER TABLE runtime_profiles 
    ADD CONSTRAINT fk_runtime_profiles_workspace 
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE SET NULL;

-- ============================================================
-- ENVIRONMENTS - Runnable instances (replaces sandboxes)
-- ============================================================
CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    runtime_profile_id UUID REFERENCES runtime_profiles(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'CREATED', -- CREATED, BUILDING, BUILD_FAILED, READY, STARTING, RUNNING, STOPPING, STOPPED, SUSPENDING, SUSPENDED, CRASHED, DELETING
    container_id TEXT,
    image_id UUID, -- FK to images table
    public_url TEXT,
    port INT,
    -- Resource limits
    memory_limit BIGINT DEFAULT 536870912, -- 512MB
    cpu_limit BIGINT DEFAULT 1000000000, -- 1000m CPU
    -- Activity tracking
    last_activity_at TIMESTAMPTZ,
    active_sessions INT DEFAULT 0,
    suspended_at TIMESTAMPTZ,
    -- Error tracking
    last_error TEXT,
    restart_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_environments_workspace ON environments(workspace_id);
CREATE INDEX idx_environments_state ON environments(state);
CREATE INDEX idx_environments_activity ON environments(last_activity_at);

-- Environment services (sidecars: postgres, redis, etc.)
CREATE TABLE IF NOT EXISTS environment_services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- postgres, redis, mysql, etc.
    type TEXT NOT NULL, -- DATABASE, CACHE, MESSAGE_QUEUE, OTHER
    image TEXT NOT NULL,
    port INT,
    config JSONB, -- env vars, credentials, etc.
    state TEXT NOT NULL DEFAULT 'CREATED',
    container_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_env_services_environment ON environment_services(environment_id);

-- ============================================================
-- BUILD PLANS & BUILDS - Image construction pipeline
-- ============================================================
CREATE TABLE IF NOT EXISTS build_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    runtime_profile_id UUID NOT NULL REFERENCES runtime_profiles(id) ON DELETE CASCADE,
    base_image TEXT NOT NULL,
    dockerfile TEXT NOT NULL,
    build_args JSONB,
    install_command TEXT,
    build_command TEXT,
    start_command TEXT,
    working_dir TEXT DEFAULT '/app',
    port INT,
    confidence REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING', -- PENDING, READY, FAILED
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_build_plans_runtime_profile ON build_plans(runtime_profile_id);

CREATE TABLE IF NOT EXISTS builds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_plan_id UUID NOT NULL REFERENCES build_plans(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    commit_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'QUEUED', -- QUEUED, BUILDING, SUCCESS, FAILED, CANCELLED
    image_id UUID, -- FK to images after success
    logs TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error TEXT,
    cache_hit BOOLEAN DEFAULT FALSE,
    build_duration_ms BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_builds_workspace ON builds(workspace_id);
CREATE INDEX idx_builds_build_plan ON builds(build_plan_id);
CREATE INDEX idx_builds_status ON builds(status);

-- ============================================================
-- IMAGES - Built container images
-- ============================================================
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    build_id UUID REFERENCES builds(id) ON DELETE SET NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tag TEXT NOT NULL, -- e.g., "berth/workspace-abc123:commit-hash"
    digest TEXT, -- sha256 digest
    size_bytes BIGINT,
    base_image TEXT,
    labels JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_images_workspace ON images(workspace_id);
CREATE INDEX idx_images_tag ON images(tag);

-- ============================================================
-- CHANGE REQUESTS - Editor → Owner workflow
-- ============================================================
CREATE TABLE IF NOT EXISTS change_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    target_workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE, -- usually canonical
    title TEXT NOT NULL,
    description TEXT,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'OPEN', -- OPEN, REVIEW, MERGED, CLOSED, CONFLICT
    commits JSONB, -- list of commit hashes/messages
    files_changed JSONB, -- list of file paths
    reviewer_id UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    merged_at TIMESTAMPTZ,
    merge_commit_hash TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_change_requests_project ON change_requests(project_id);
CREATE INDEX idx_change_requests_source ON change_requests(source_workspace_id);
CREATE INDEX idx_change_requests_state ON change_requests(state);

-- ============================================================
-- ENVIRONMENT EVENTS - Activity log for WebSocket broadcasting
-- ============================================================
CREATE TABLE IF NOT EXISTS environment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type TEXT NOT NULL, -- STATE_CHANGED, FILE_CHANGED, TERMINAL_OUTPUT, BUILD_STARTED, BUILD_FINISHED, USER_JOINED, USER_LEFT, CHANGE_REQUEST_CREATED, etc.
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_env_events_environment ON environment_events(environment_id);
CREATE INDEX idx_env_events_workspace ON environment_events(workspace_id);
CREATE INDEX idx_env_events_project ON environment_events(project_id);
CREATE INDEX idx_env_events_created ON environment_events(created_at);

-- ============================================================
-- MIGRATE EXISTING SANDBOXES TO NEW SCHEMA
-- ============================================================

-- Create canonical workspaces for existing projects
INSERT INTO workspaces (id, project_id, name, type, owner_id, git_branch, created_at, updated_at)
SELECT 
    gen_random_uuid(),
    s.project_id,
    'canonical',
    'CANONICAL',
    s.owner_id,
    s.git_branch,
    s.created_at,
    s.updated_at
FROM sandboxes s
WHERE s.project_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- Create fork workspaces for sandboxes without project (personal sandboxes)
INSERT INTO workspaces (id, project_id, name, type, owner_id, git_branch, created_at, updated_at)
SELECT 
    s.id, -- reuse sandbox ID as workspace ID for migration
    p.id,
    s.name,
    'FORK',
    s.owner_id,
    s.git_branch,
    s.created_at,
    s.updated_at
FROM sandboxes s
JOIN projects p ON p.created_by_user_id = s.owner_id
WHERE s.project_id IS NULL
ON CONFLICT DO NOTHING;

-- Add workspace members for existing project collaborators
INSERT INTO workspace_members (workspace_id, user_id, role)
SELECT w.id, pc.user_id, 
    CASE pc.role 
        WHEN 'OWNER' THEN 'OWNER'
        WHEN 'ADMIN' THEN 'EDITOR'
        WHEN 'COLLABORATOR' THEN 'EDITOR'
        WHEN 'VIEWER' THEN 'VIEWER'
        ELSE 'EDITOR'
    END
FROM workspaces w
JOIN projects p ON w.project_id = p.id
JOIN project_collaborators pc ON p.id = pc.project_id
WHERE w.type = 'CANONICAL'
AND pc.accepted_at IS NOT NULL
ON CONFLICT DO NOTHING;

-- Add workspace owner as member
INSERT INTO workspace_members (workspace_id, user_id, role)
SELECT w.id, w.owner_id, 'OWNER'
FROM workspaces w
ON CONFLICT DO NOTHING;

-- Create environments from sandboxes
INSERT INTO environments (id, workspace_id, name, state, container_id, public_url, port, memory_limit, cpu_limit, created_at, updated_at)
SELECT 
    s.id,
    w.id,
    s.name,
    CASE s.state
        WHEN 'PENDING' THEN 'CREATED'
        WHEN 'BUILDING' THEN 'BUILDING'
        WHEN 'RUNNING' THEN 'RUNNING'
        WHEN 'STOPPED' THEN 'STOPPED'
        WHEN 'FAILED' THEN 'CRASHED'
        ELSE 'CREATED'
    END,
    s.container_id,
    s.public_url,
    COALESCE(s.runtime_port, 3000),
    536870912,
    1000000000,
    s.created_at,
    s.updated_at
FROM sandboxes s
JOIN workspaces w ON (
    (s.project_id IS NOT NULL AND w.project_id = s.project_id AND w.type = 'CANONICAL')
    OR (s.project_id IS NULL AND w.id = s.id)
)
ON CONFLICT DO NOTHING;

-- Create runtime profiles from sandbox data
INSERT INTO runtime_profiles (project_id, workspace_id, detection_evidence, language, version, framework, package_manager, architecture, entrypoint, build_command, start_command, port, dockerfile_source, confidence, status, created_at, updated_at)
SELECT 
    w.project_id,
    w.id,
    jsonb_build_object(
        'language', COALESCE(s.runtime_language, 'unknown'),
        'files', '[]'::jsonb,
        'confidence', 0.8
    ),
    COALESCE(s.runtime_language, 'unknown'),
    NULL,
    NULL,
    NULL,
    'WEB_APP',
    NULL,
    NULL,
    NULL,
    COALESCE(s.runtime_port, 3000),
    'GENERATED',
    0.8,
    'DETECTED',
    s.created_at,
    s.updated_at
FROM sandboxes s
JOIN workspaces w ON (
    (s.project_id IS NOT NULL AND w.project_id = s.project_id AND w.type = 'CANONICAL')
    OR (s.project_id IS NULL AND w.id = s.id)
)
WHERE s.runtime_language IS NOT NULL
ON CONFLICT DO NOTHING;