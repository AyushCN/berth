-- User queries.
--
-- These lived in queries/sandbox.sql until the legacy sandbox model was
-- removed, which meant deleting the sandbox queries silently deleted the user
-- queries too. They are unrelated to sandboxes and belong here.

-- name: CreateUser :one
INSERT INTO users (id, email, username, github_id, github_username, avatar_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1;
