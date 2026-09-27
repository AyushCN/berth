-- name: CreateEnvironmentEvent :one
INSERT INTO environment_events (
    environment_id, workspace_id, project_id, user_id, type, payload
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetEnvironmentEvents :many
SELECT ee.*, u.username, u.avatar_url
FROM environment_events ee
LEFT JOIN users u ON ee.user_id = u.id
WHERE ee.environment_id = $1
ORDER BY ee.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetWorkspaceEvents :many
SELECT ee.*, u.username, u.avatar_url
FROM environment_events ee
LEFT JOIN users u ON ee.user_id = u.id
WHERE ee.workspace_id = $1
ORDER BY ee.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetProjectEvents :many
SELECT ee.*, u.username, u.avatar_url
FROM environment_events ee
LEFT JOIN users u ON ee.user_id = u.id
WHERE ee.project_id = $1
ORDER BY ee.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetRecentProjectEvents :many
SELECT ee.*, u.username, u.avatar_url
FROM environment_events ee
LEFT JOIN users u ON ee.user_id = u.id
WHERE ee.project_id = $1 AND ee.created_at > $2
ORDER BY ee.created_at DESC
LIMIT $3;