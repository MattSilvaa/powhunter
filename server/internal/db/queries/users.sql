-- name: CreateUser :one
INSERT INTO users (
  email, phone
) VALUES (
  $1, $2
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByUUID :one
SELECT * FROM users
WHERE uuid = $1 LIMIT 1;

-- name: GetOrCreateUserByEmail :one
-- Requesting a login link for an address that has no account creates one, so
-- signup and login are the same flow. DO UPDATE rather than DO NOTHING because
-- DO NOTHING returns no row on conflict.
INSERT INTO users (email)
VALUES ($1)
ON CONFLICT (email) DO UPDATE
    SET updated_at = NOW()
RETURNING *;

-- name: UpsertUser :one
-- Get-or-create in a single statement. A read-then-write race meant two
-- concurrent signups with the same email both attempted an insert, and one
-- failed on the unique constraint as a 500.
INSERT INTO users (email, phone)
VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE
    SET phone = COALESCE(EXCLUDED.phone, users.phone)
RETURNING *;
