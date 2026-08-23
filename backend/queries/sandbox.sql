-- name: CreateSandbox :one
INSERT INTO sandboxes (
    project_id, owner_id, name, git_url, git_branch, state,
    runtime_language, runtime_base_image, runtime_port, needs_db
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetSandboxByID :one
SELECT * FROM sandboxes WHERE id = $1;

-- name: ListSandboxesByProject :many
SELECT * FROM sandboxes WHERE project_id = $1 ORDER BY created_at DESC;

-- name: UpdateSandboxState :exec
UPDATE sandboxes SET state = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateSandboxContainer :exec
UPDATE sandboxes SET container_id = $2, public_url = $3, updated_at = NOW() WHERE id = $1;

-- name: DeleteSandbox :exec
DELETE FROM sandboxes WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, username, github_id, github_username, avatar_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1;

-- name: PopPendingSandbox :one
UPDATE sandboxes 
SET state = 'BUILDING', updated_at = NOW() 
WHERE id = (
    SELECT id 
    FROM sandboxes 
    WHERE state = 'PENDING' 
    ORDER BY created_at ASC 
    FOR UPDATE SKIP LOCKED 
    LIMIT 1
) 
RETURNING *;
