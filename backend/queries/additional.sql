-- name: UpdateUserToken :exec
UPDATE users SET github_token_encrypted = $2, updated_at = NOW() WHERE id = $1;
