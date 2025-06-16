-- name: CheckAlertSent :one
SELECT EXISTS(SELECT 1
              FROM alert_history
              WHERE user_uuid = $1
                AND resort_uuid = $2
                AND forecast_date = $3) as alert_sent;

-- name: GetLastAlertSnowAmount :one
SELECT snow_amount
FROM alert_history
WHERE user_uuid = $1
  AND resort_uuid = $2
  AND forecast_date = $3
ORDER BY sent_at DESC LIMIT 1;

-- name: InsertAlertHistory :exec
INSERT INTO alert_history (user_uuid, resort_uuid, forecast_date, snow_amount, sent_at)
VALUES ($1, $2, $3, $4, NOW());
