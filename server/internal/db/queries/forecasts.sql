-- name: CreateSnowForecast :one
INSERT INTO snow_forecasts (
  resort_uuid, forecast_date, predicted_snow_amount
) VALUES (
  $1, $2, $3
)
ON CONFLICT (resort_uuid, forecast_date) DO UPDATE
SET predicted_snow_amount = EXCLUDED.predicted_snow_amount,
    last_updated = NOW()
RETURNING *;

-- name: GetSnowForecasts :many
SELECT * FROM snow_forecasts
WHERE resort_uuid = $1
  AND forecast_date BETWEEN $2 AND $3
  AND predicted_snow_amount >= $4;
