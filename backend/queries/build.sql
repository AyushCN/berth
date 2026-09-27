-- name: CreateBuildPlan :one
INSERT INTO build_plans (
    runtime_profile_id, base_image, dockerfile, build_args,
    install_command, build_command, start_command, working_dir,
    port, confidence, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetBuildPlan :one
SELECT * FROM build_plans WHERE id = $1;

-- name: GetBuildPlanByRuntimeProfile :one
SELECT * FROM build_plans WHERE runtime_profile_id = $1;

-- name: UpdateBuildPlan :one
UPDATE build_plans SET
    base_image = COALESCE(sqlc.narg('base_image'), base_image),
    dockerfile = COALESCE(sqlc.narg('dockerfile'), dockerfile),
    build_args = COALESCE(sqlc.narg('build_args'), build_args),
    install_command = COALESCE(sqlc.narg('install_command'), install_command),
    build_command = COALESCE(sqlc.narg('build_command'), build_command),
    start_command = COALESCE(sqlc.narg('start_command'), start_command),
    working_dir = COALESCE(sqlc.narg('working_dir'), working_dir),
    port = COALESCE(sqlc.narg('port'), port),
    confidence = COALESCE(sqlc.narg('confidence'), confidence),
    status = COALESCE(sqlc.narg('status'), status),
    error = COALESCE(sqlc.narg('error'), error),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBuildPlan :exec
DELETE FROM build_plans WHERE id = $1;

-- name: CreateBuild :one
INSERT INTO builds (
    build_plan_id, workspace_id, commit_hash, status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetBuild :one
SELECT * FROM builds WHERE id = $1;

-- name: GetBuildsByWorkspace :many
SELECT * FROM builds 
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetBuildsByBuildPlan :many
SELECT * FROM builds 
WHERE build_plan_id = $1
ORDER BY created_at DESC;

-- name: GetLatestBuildByWorkspace :one
SELECT * FROM builds 
WHERE workspace_id = $1
ORDER BY created_at DESC LIMIT 1;

-- name: UpdateBuild :one
UPDATE builds SET
    status = COALESCE(sqlc.narg('status'), status),
    image_id = COALESCE(sqlc.narg('image_id'), image_id),
    logs = COALESCE(sqlc.narg('logs'), logs),
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    finished_at = COALESCE(sqlc.narg('finished_at'), finished_at),
    error = COALESCE(sqlc.narg('error'), error),
    cache_hit = COALESCE(sqlc.narg('cache_hit'), cache_hit),
    build_duration_ms = COALESCE(sqlc.narg('build_duration_ms'), build_duration_ms)
WHERE id = $1
RETURNING *;

-- name: DeleteBuild :exec
DELETE FROM builds WHERE id = $1;

-- name: CreateImage :one
INSERT INTO images (
    build_id, workspace_id, tag, digest, size_bytes, base_image, labels
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetImage :one
SELECT * FROM images WHERE id = $1;

-- name: GetImageByTag :one
SELECT * FROM images WHERE tag = $1;

-- name: GetImagesByWorkspace :many
SELECT * FROM images 
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: UpdateImageLastUsed :exec
UPDATE images SET last_used_at = NOW() WHERE id = $1;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = $1;