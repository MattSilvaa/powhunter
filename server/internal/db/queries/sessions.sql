-- name: CreateLoginToken :one
INSERT INTO login_tokens (user_uuid, token_hash, expires_at)
VALUES ($1, $2, $3) RETURNING *;

-- name: ConsumeLoginToken :one
-- Redeeming a token is a conditional UPDATE rather than a SELECT followed by an
-- UPDATE: only one caller can win the race, so a link forwarded to someone else
-- cannot be redeemed twice.
UPDATE login_tokens
SET consumed_at = NOW()
WHERE token_hash = $1
  AND consumed_at IS NULL
  AND expires_at > NOW() RETURNING user_uuid;

-- name: DeleteExpiredLoginTokens :exec
DELETE FROM login_tokens
WHERE expires_at < NOW();

-- name: CreateSession :one
INSERT INTO sessions (user_uuid, token_hash, expires_at)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetSessionByTokenHash :one
-- Returns the session together with its user so authenticating a request is a
-- single round trip. Expired rows never match, so a stale cookie reads as
-- signed out even before the reaper removes it.
SELECT s.uuid          as session_uuid,
       s.expires_at,
       u.uuid          as user_uuid,
       u.email,
       u.phone,
       u.email_verified_at
FROM sessions s
         JOIN users u ON s.user_uuid = u.uuid
WHERE s.token_hash = $1
  AND s.expires_at > NOW() LIMIT 1;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = NOW(),
    expires_at   = $2
WHERE token_hash = $1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < NOW();

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified_at = COALESCE(email_verified_at, NOW()),
    updated_at        = NOW()
WHERE uuid = $1;
