-- name: CreateAlertHistory :one
INSERT INTO alert_history (
  user_id, resort_uuid, forecast_date, snow_amount
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: CheckAlertSent :one
SELECT EXISTS(
  SELECT 1 FROM alert_history
  WHERE user_id = $1
    AND resort_uuid = $2
        AND forecast_date = $3
) as alert_sent;