
-- name: ListSandboxesByOwner :many
SELECT * FROM sandboxes WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: CountSandboxesByOwner :one
SELECT COUNT(*) FROM sandboxes WHERE owner_id = $1;

-- name: UpdateUserToken :exec
UPDATE users SET github_token_encrypted = $2, updated_at = NOW() WHERE id = $1;
