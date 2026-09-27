-- name: CreateShareLink :one
INSERT INTO share_links (
    project_id, code, role, created_by, expires_at, max_uses
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetShareLinkByCode :one
SELECT * FROM share_links
WHERE code = $1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
AND (max_uses IS NULL OR uses_count < max_uses);

-- name: GetShareLinksByProject :many
SELECT * FROM share_links
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: UpdateShareLink :one
UPDATE share_links
SET uses_count = uses_count + 1,
    revoked_at = COALESCE(sqlc.narg('revoked_at'), revoked_at)
WHERE id = $1
RETURNING *;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = $1;

-- name: IncrementShareLinkUses :exec
UPDATE share_links SET uses_count = uses_count + 1 WHERE id = $1;

-- name: GetShareLinkByID :one
SELECT * FROM share_links WHERE id = $1;