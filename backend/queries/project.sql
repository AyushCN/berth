-- name: CreateProject :one
INSERT INTO projects (name, description, owner_organization_id, created_by_user_id, is_public)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = $1 LIMIT 1;

-- name: ListProjectsForOrg :many
SELECT * FROM projects WHERE owner_organization_id = $1 ORDER BY created_at DESC;

-- name: ListProjectsForUser :many
SELECT p.*, pc.role
FROM projects p
JOIN project_collaborators pc ON p.id = pc.project_id
WHERE pc.user_id = $1
ORDER BY p.created_at DESC;

-- name: AddProjectCollaborator :one
INSERT INTO project_collaborators (project_id, user_id, role, invited_by_user_id, invited_at)
VALUES ($1, $2, $3, $4, NOW())
RETURNING *;

-- name: GetProjectCollaborator :one
SELECT * FROM project_collaborators 
WHERE project_id = $1 AND user_id = $2 LIMIT 1;

-- name: ListProjectCollaborators :many
SELECT pc.id, pc.project_id, pc.user_id, pc.role, pc.invited_at, pc.accepted_at, u.username, u.email, u.avatar_url 
FROM project_collaborators pc
JOIN users u ON u.id = pc.user_id
WHERE pc.project_id = $1
ORDER BY pc.invited_at ASC;

-- name: UpdateProjectRole :exec
UPDATE project_collaborators
SET role = $3
WHERE project_id = $1 AND user_id = $2;

-- name: RemoveProjectCollaborator :exec
DELETE FROM project_collaborators
WHERE project_id = $1 AND user_id = $2;


