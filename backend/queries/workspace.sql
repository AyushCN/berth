-- name: CreateWorkspace :one
INSERT INTO workspaces (
    project_id, name, type, base_workspace_id, owner_id, git_branch
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = $1 AND deleted_at IS NULL;

-- name: GetWorkspacesByProject :many
SELECT * FROM workspaces 
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY 
    CASE type WHEN 'CANONICAL' THEN 0 ELSE 1 END,
    created_at DESC;

-- name: GetForkWorkspaces :many
SELECT * FROM workspaces 
WHERE base_workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetCanonicalWorkspace :one
SELECT * FROM workspaces 
WHERE project_id = $1 AND type = 'CANONICAL' AND deleted_at IS NULL;

-- name: UpdateWorkspace :one
UPDATE workspaces SET
    name = COALESCE(sqlc.narg('name'), name),
    git_branch = COALESCE(sqlc.narg('git_branch'), git_branch),
    commit_hash = COALESCE(sqlc.narg('commit_hash'), commit_hash),
    has_uncommitted_changes = COALESCE(sqlc.narg('has_uncommitted_changes'), has_uncommitted_changes),
    last_synced_at = COALESCE(sqlc.narg('last_synced_at'), last_synced_at),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteWorkspace :exec
UPDATE workspaces SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: CreateWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: GetWorkspaceMembers :many
SELECT wm.*, u.username, u.email, u.avatar_url
FROM workspace_members wm
JOIN users u ON wm.user_id = u.id
WHERE wm.workspace_id = $1
ORDER BY 
    CASE wm.role WHEN 'OWNER' THEN 0 WHEN 'EDITOR' THEN 1 ELSE 2 END,
    wm.created_at;

-- name: GetWorkspaceMember :one
SELECT wm.*, u.username, u.email, u.avatar_url
FROM workspace_members wm
JOIN users u ON wm.user_id = u.id
WHERE wm.workspace_id = $1 AND wm.user_id = $2;

-- name: GetUserWorkspaces :many
SELECT w.*, wm.role as member_role
FROM workspaces w
JOIN workspace_members wm ON w.id = wm.workspace_id
WHERE wm.user_id = $1 AND w.deleted_at IS NULL
ORDER BY w.updated_at DESC;

-- name: UpdateWorkspaceMember :one
UPDATE workspace_members SET role = $3 WHERE workspace_id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteWorkspaceMember :exec
DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2;

-- name: GetWorkspaceMembersByUser :many
SELECT wm.*, u.username, u.email, u.avatar_url
FROM workspace_members wm
JOIN users u ON wm.user_id = u.id
WHERE wm.user_id = $1
ORDER BY wm.created_at;