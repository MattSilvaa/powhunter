-- name: CreateUserAlert :one
INSERT INTO user_alerts (user_uuid, resort_uuid, min_snow_amount, notification_days)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpsertUnconfirmedUserAlert :exec
-- Anonymous signup for an unconfirmed account. Signing up again for the same
-- resort must not fail: the first confirmation email may never have arrived,
-- and a 409 would both block a fresh link and roll back the phone update that
-- stops a squatter keeping their number on someone else's address. The latest
-- thresholds win, as the latest phone does.
INSERT INTO user_alerts (user_uuid, resort_uuid, min_snow_amount, notification_days)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_uuid, resort_uuid) DO UPDATE
    SET min_snow_amount   = EXCLUDED.min_snow_amount,
        notification_days = EXCLUDED.notification_days;

-- name: GetUserAlert :one
SELECT *
FROM user_alerts
WHERE user_uuid = $1
  AND resort_uuid = $2 LIMIT 1;

-- name: GetResortAlerts :many
-- Only accounts with a confirmed email are alerted. Signup is anonymous, so an
-- unconfirmed account may carry a phone number its owner never agreed to text.
SELECT *
FROM user_alerts
WHERE resort_uuid = $1
  AND active = true
  AND user_uuid IN (SELECT uuid FROM users WHERE email_verified_at IS NOT NULL);

-- name: UpdateUserAlert :one
UPDATE user_alerts
SET min_snow_amount   = $3,
    notification_days = $4,
    active            = $5
WHERE user_uuid = $1
  AND resort_uuid = $2 RETURNING *;

-- name: ListActiveAlerts :many
SELECT ua.id,
       ua.user_uuid,
       u.email,
       u.phone,
       ua.resort_uuid,
       r.name as resort_name,
       ua.min_snow_amount,
       ua.notification_days
FROM user_alerts ua
         JOIN users u ON ua.user_uuid = u.uuid
         JOIN resorts r ON ua.resort_uuid = r.uuid
WHERE ua.active = true;

-- The three queries below key off the authenticated user's UUID rather than an
-- email supplied by the caller. Taking the identity from the session is what
-- stops one person reading or deleting another person's alerts.

-- name: GetUserAlertsByUserUUID :many
SELECT ua.id,
       ua.user_uuid,
       ua.resort_uuid,
       r.name as resort_name,
       ua.min_snow_amount,
       ua.notification_days,
       ua.active,
       ua.created_at
FROM user_alerts ua
         JOIN resorts r ON ua.resort_uuid = r.uuid
WHERE ua.user_uuid = $1 AND ua.active = true;

-- name: DeleteUserAlertByUserUUID :exec
DELETE FROM user_alerts
WHERE user_uuid = $1
  AND resort_uuid = $2;

-- name: DeleteAllUserAlertsByUserUUID :exec
DELETE FROM user_alerts
WHERE user_uuid = $1;

-- name: UpdateUserAlertSettingsByUserUUID :one
-- Changes only the thresholds. Unlike UpdateUserAlert it leaves active alone,
-- and it keys off the session's user so one account cannot edit another's.
UPDATE user_alerts
SET min_snow_amount   = $3,
    notification_days = $4
WHERE user_uuid = $1
  AND resort_uuid = $2
  AND active = true RETURNING *;
