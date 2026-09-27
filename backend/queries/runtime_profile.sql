-- name: CreateRuntimeProfile :one
INSERT INTO runtime_profiles (
    project_id, workspace_id, detection_evidence, language, version, framework,
    package_manager, architecture, entrypoint, build_command, start_command,
    port, dockerfile_source, dockerfile_content, requires_database, requires_redis,
    confidence, status, confirmed_by, confirmed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
) RETURNING *;

-- name: GetRuntimeProfile :one
SELECT * FROM runtime_profiles WHERE id = $1;

-- name: GetRuntimeProfilesByProject :many
SELECT * FROM runtime_profiles WHERE project_id = $1 ORDER BY created_at DESC;

-- name: GetRuntimeProfilesByWorkspace :many
SELECT * FROM runtime_profiles WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: GetLatestRuntimeProfileByWorkspace :one
SELECT * FROM runtime_profiles 
WHERE workspace_id = $1 
ORDER BY created_at DESC LIMIT 1;

-- name: UpdateRuntimeProfile :one
UPDATE runtime_profiles SET
    detection_evidence = COALESCE(sqlc.narg('detection_evidence'), detection_evidence),
    language = COALESCE(sqlc.narg('language'), language),
    version = COALESCE(sqlc.narg('version'), version),
    framework = COALESCE(sqlc.narg('framework'), framework),
    package_manager = COALESCE(sqlc.narg('package_manager'), package_manager),
    architecture = COALESCE(sqlc.narg('architecture'), architecture),
    entrypoint = COALESCE(sqlc.narg('entrypoint'), entrypoint),
    build_command = COALESCE(sqlc.narg('build_command'), build_command),
    start_command = COALESCE(sqlc.narg('start_command'), start_command),
    port = COALESCE(sqlc.narg('port'), port),
    dockerfile_source = COALESCE(sqlc.narg('dockerfile_source'), dockerfile_source),
    dockerfile_content = COALESCE(sqlc.narg('dockerfile_content'), dockerfile_content),
    requires_database = COALESCE(sqlc.narg('requires_database'), requires_database),
    requires_redis = COALESCE(sqlc.narg('requires_redis'), requires_redis),
    confidence = COALESCE(sqlc.narg('confidence'), confidence),
    status = COALESCE(sqlc.narg('status'), status),
    confirmed_by = COALESCE(sqlc.narg('confirmed_by'), confirmed_by),
    confirmed_at = COALESCE(sqlc.narg('confirmed_at'), confirmed_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ConfirmRuntimeProfile :one
UPDATE runtime_profiles SET
    status = 'CONFIRMED',
    confirmed_by = $2,
    confirmed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRuntimeProfile :exec
DELETE FROM runtime_profiles WHERE id = $1;