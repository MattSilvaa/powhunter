-- name: CreateMagicLinkToken :one
INSERT INTO magic_link_tokens (
  token, user_uuid, expires_at, purpose
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetMagicLinkToken :one
SELECT * FROM magic_link_tokens
WHERE token = $1 AND expires_at > NOW() AND used_at IS NULL
LIMIT 1;

-- name: UseMagicLinkToken :exec
UPDATE magic_link_tokens
SET used_at = NOW()
WHERE token = $1;

-- name: CleanupExpiredTokens :exec
DELETE FROM magic_link_tokens
WHERE expires_at < NOW() OR used_at IS NOT NULL;

-- name: CreateUserSession :one
INSERT INTO user_sessions (
  session_id, user_uuid, expires_at
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetUserSession :one
SELECT * FROM user_sessions
WHERE session_id = $1 AND expires_at > NOW()
LIMIT 1;

-- name: UpdateSessionLastUsed :exec
UPDATE user_sessions
SET last_used_at = NOW()
WHERE session_id = $1;

-- name: DeleteUserSession :exec
DELETE FROM user_sessions
WHERE session_id = $1;

-- name: CleanupExpiredSessions :exec
DELETE FROM user_sessions
WHERE expires_at < NOW();

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified = TRUE, verified_at = NOW()
WHERE uuid = $1;

-- name: GetUserWithVerification :one
SELECT * FROM users
WHERE uuid = $1 LIMIT 1;