-- name: CreateChangeRequest :one
INSERT INTO change_requests (
    project_id, source_workspace_id, target_workspace_id,
    title, description, author_id, commits, files_changed
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetChangeRequest :one
SELECT cr.*, 
    sw.name as source_workspace_name,
    tw.name as target_workspace_name,
    u.username as author_username,
    u.avatar_url as author_avatar
FROM change_requests cr
JOIN workspaces sw ON cr.source_workspace_id = sw.id
JOIN workspaces tw ON cr.target_workspace_id = tw.id
JOIN users u ON cr.author_id = u.id
WHERE cr.id = $1;

-- name: GetChangeRequestsByProject :many
SELECT cr.*, 
    sw.name as source_workspace_name,
    tw.name as target_workspace_name,
    u.username as author_username,
    u.avatar_url as author_avatar,
    ru.username as reviewer_username
FROM change_requests cr
JOIN workspaces sw ON cr.source_workspace_id = sw.id
JOIN workspaces tw ON cr.target_workspace_id = tw.id
JOIN users u ON cr.author_id = u.id
LEFT JOIN users ru ON cr.reviewer_id = ru.id
WHERE cr.project_id = $1
ORDER BY cr.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetChangeRequestsBySourceWorkspace :many
SELECT cr.*, 
    sw.name as source_workspace_name,
    tw.name as target_workspace_name,
    u.username as author_username,
    u.avatar_url as author_avatar
FROM change_requests cr
JOIN workspaces sw ON cr.source_workspace_id = sw.id
JOIN workspaces tw ON cr.target_workspace_id = tw.id
JOIN users u ON cr.author_id = u.id
WHERE cr.source_workspace_id = $1
ORDER BY cr.created_at DESC;

-- name: GetChangeRequestsByState :many
SELECT cr.*, 
    sw.name as source_workspace_name,
    tw.name as target_workspace_name,
    u.username as author_username,
    u.avatar_url as author_avatar
FROM change_requests cr
JOIN workspaces sw ON cr.source_workspace_id = sw.id
JOIN workspaces tw ON cr.target_workspace_id = tw.id
JOIN users u ON cr.author_id = u.id
WHERE cr.state = $1
ORDER BY cr.created_at DESC;

-- name: UpdateChangeRequest :one
UPDATE change_requests SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    state = COALESCE(sqlc.narg('state'), state),
    commits = COALESCE(sqlc.narg('commits'), commits),
    files_changed = COALESCE(sqlc.narg('files_changed'), files_changed),
    reviewer_id = COALESCE(sqlc.narg('reviewer_id'), reviewer_id),
    reviewed_at = COALESCE(sqlc.narg('reviewed_at'), reviewed_at),
    merged_at = COALESCE(sqlc.narg('merged_at'), merged_at),
    merge_commit_hash = COALESCE(sqlc.narg('merge_commit_hash'), merge_commit_hash),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteChangeRequest :exec
DELETE FROM change_requests WHERE id = $1;