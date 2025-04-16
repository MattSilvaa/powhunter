-- name: CreateUserAlert :one
INSERT INTO user_alerts (
  user_id, resort_uuid, min_snow_amount, notification_days
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetUserAlert :one
SELECT * FROM user_alerts
WHERE user_id = $1 AND resort_uuid = $2
LIMIT 1;

-- name: UpdateUserAlert :one
UPDATE user_alerts
SET min_snow_amount = $3,
    notification_days = $4,
    active = $5
WHERE user_id = $1 AND resort_uuid = $2
RETURNING *;

-- name: ListActiveAlerts :many
SELECT
  ua.id, ua.user_id, u.email, u.phone,
  ua.resort_uuid, r.name as resort_name,
  ua.min_snow_amount, ua.notification_days
FROM user_alerts ua
JOIN users u ON ua.user_id = u.id
JOIN resorts r ON ua.resort_uuid = r.uuid
WHERE ua.active = true;

-- name: GetNewAlertMatches :many
SELECT
  u.id as user_id,
  u.email,
  u.phone,
  r.name as resort_name,
  r.uuid as resort_uuid,
  $2 as forecast_date,
  $3 as snow_amount,
  NULL as previous_snow_amount
FROM
  users u
JOIN
  alert_history ura ON u.id = ura.user_id
JOIN
  resorts r ON ura.resort_uuid = r.uuid
WHERE
  r.uuid = $1
  AND NOT EXISTS (
    SELECT 1 FROM alert_history ah
    WHERE ah.user_id = u.id
      AND ah.resort_uuid = r.uuid
      AND ah.forecast_date = $2
  )
  AND ura.min_snow_amount <= $3
  AND ura.days_ahead >= $4;

-- name: GetLatestAlertSnowAmount :one
SELECT amount_cm
FROM alert_history
WHERE user_id = $1
  AND resort_uuid = $2
  AND forecast_date = $3
ORDER BY created_at DESC
LIMIT 1;
