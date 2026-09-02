-- name: CreateOrganization :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations WHERE id = $1 LIMIT 1;

-- name: AddOrganizationMember :one
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrganizationMember :one
SELECT * FROM organization_members 
WHERE organization_id = $1 AND user_id = $2 LIMIT 1;

-- name: ListOrganizationMembers :many
SELECT om.id, om.organization_id, om.user_id, om.role, om.created_at, u.username, u.email, u.avatar_url 
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1
ORDER BY om.created_at ASC;

-- name: ListOrganizationsForUser :many
SELECT o.id, o.name, o.created_at, om.role
FROM organizations o
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1
ORDER BY o.created_at DESC;

-- name: UpdateOrganizationRole :exec
UPDATE organization_members
SET role = $3
WHERE organization_id = $1 AND user_id = $2;

-- name: RemoveOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = $1 AND user_id = $2;
