-- name: CreateEnvironment :one
INSERT INTO environments (
    workspace_id, runtime_profile_id, name, state, container_id, image_id,
    public_url, port, memory_limit, cpu_limit
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetEnvironment :one
SELECT * FROM environments WHERE id = $1 AND deleted_at IS NULL;

-- name: GetEnvironmentsByWorkspace :many
SELECT * FROM environments 
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetActiveEnvironmentsByWorkspace :many
SELECT * FROM environments 
WHERE workspace_id = $1 AND deleted_at IS NULL 
AND state IN ('RUNNING', 'STARTING', 'BUILDING', 'READY')
ORDER BY created_at DESC;

-- name: UpdateEnvironment :one
UPDATE environments SET
    name = COALESCE(sqlc.narg('name'), name),
    runtime_profile_id = COALESCE(sqlc.narg('runtime_profile_id'), runtime_profile_id),
    container_id = COALESCE(sqlc.narg('container_id'), container_id),
    image_id = COALESCE(sqlc.narg('image_id'), image_id),
    public_url = COALESCE(sqlc.narg('public_url'), public_url),
    port = COALESCE(sqlc.narg('port'), port),
    memory_limit = COALESCE(sqlc.narg('memory_limit'), memory_limit),
    cpu_limit = COALESCE(sqlc.narg('cpu_limit'), cpu_limit),
    last_activity_at = COALESCE(sqlc.narg('last_activity_at'), last_activity_at),
    active_sessions = COALESCE(sqlc.narg('active_sessions'), active_sessions),
    suspended_at = COALESCE(sqlc.narg('suspended_at'), suspended_at),
    last_error = COALESCE(sqlc.narg('last_error'), last_error),
    restart_count = COALESCE(sqlc.narg('restart_count'), restart_count),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateEnvironmentState :one
UPDATE environments SET
    state = $2,
    last_error = COALESCE(sqlc.narg('last_error'), last_error),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateEnvironmentContainerID :one
UPDATE environments SET
    container_id = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateEnvironmentImageID :one
UPDATE environments SET
    image_id = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateEnvironmentActivity :one
UPDATE environments SET
    last_activity_at = NOW(),
    active_sessions = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteEnvironment :exec
UPDATE environments SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ListEnvironmentsByState :many
SELECT * FROM environments 
WHERE state = $1 AND deleted_at IS NULL
ORDER BY updated_at;

-- name: ListSuspendedEnvironments :many
SELECT * FROM environments 
WHERE state = 'SUSPENDED' AND deleted_at IS NULL
AND suspended_at < $1
ORDER BY suspended_at;

-- name: ListIdleRunningEnvironments :many
SELECT * FROM environments 
WHERE state = 'RUNNING' AND deleted_at IS NULL
AND active_sessions = 0
AND (last_activity_at IS NULL OR last_activity_at < $1)
ORDER BY last_activity_at;

-- name: CountEnvironmentsByStateAndRuntimeProfile :one
SELECT COUNT(*) FROM environments 
WHERE state = $1 AND deleted_at IS NULL
AND runtime_profile_id = $2;

-- name: CreateEnvironmentService :one
INSERT INTO environment_services (
    environment_id, name, type, image, port, config, state, container_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetEnvironmentServices :many
SELECT * FROM environment_services 
WHERE environment_id = $1
ORDER BY created_at;

-- name: GetEnvironmentService :one
SELECT * FROM environment_services WHERE id = $1;

-- name: UpdateEnvironmentService :one
UPDATE environment_services SET
    state = COALESCE(sqlc.narg('state'), state),
    container_id = COALESCE(sqlc.narg('container_id'), container_id),
    config = COALESCE(sqlc.narg('config'), config),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEnvironmentService :exec
DELETE FROM environment_services WHERE id = $1;