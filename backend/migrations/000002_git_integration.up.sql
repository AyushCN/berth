ALTER TABLE sandboxes
ADD COLUMN has_uncommitted_changes BOOLEAN DEFAULT FALSE,
ADD COLUMN last_modified_at TIMESTAMPTZ,
ADD COLUMN modified_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN commit_hash TEXT;

CREATE TABLE IF NOT EXISTS sandbox_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID NOT NULL REFERENCES sandboxes(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    change_type TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    diff TEXT,
    committed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sandbox_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID REFERENCES sandboxes(id) ON DELETE CASCADE,
    activity_type TEXT NOT NULL,
    data JSONB,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sandbox_activities_sandbox ON sandbox_activities(sandbox_id);
CREATE INDEX idx_sandbox_activities_created_at ON sandbox_activities(created_at);
CREATE INDEX idx_sandbox_changes_sandbox ON sandbox_changes(sandbox_id);
