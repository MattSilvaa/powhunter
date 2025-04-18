-- name: CreateUserAlert :one
INSERT INTO user_alerts (user_uuid, resort_uuid, min_snow_amount, notification_days)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetUserAlert :one
SELECT *
FROM user_alerts
WHERE user_uuid = $1
  AND resort_uuid = $2 LIMIT 1;

-- name: GetResortAlerts :many
SELECT *
FROM user_alerts
WHERE resort_uuid = $1
  and active = true;

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
         JOIN users u ON ua.user_uuid = u.id
         JOIN resorts r ON ua.resort_uuid = r.uuid
WHERE ua.active = true;
