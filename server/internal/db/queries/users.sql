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
--
-- Signup is anonymous, so the phone supplied here never replaces the number on
-- a confirmed account: that would let anyone who knows an address redirect the
-- owner's SMS alerts to themselves. Changing it is done signed in, through
-- UpdateUserPhone. An unconfirmed account does take the new number, or someone
-- who signed up first with a victim's address would keep their own phone on it
-- after the victim signed up and confirmed.
--
-- The referral code and signup source are only recorded for a new account:
-- someone who already signed up was not brought in by the link they used this
-- time.
INSERT INTO users (email, phone, referral_code, signup_source, signup_campaign)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (email) DO UPDATE
    SET phone = CASE
                    WHEN users.email_verified_at IS NULL THEN EXCLUDED.phone
                    ELSE COALESCE(users.phone, EXCLUDED.phone)
                END
RETURNING *;

-- name: UpdateUserPhone :one
UPDATE users
SET phone      = $2,
    updated_at = NOW()
WHERE uuid = $1
RETURNING *;
