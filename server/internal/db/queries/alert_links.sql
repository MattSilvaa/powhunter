-- name: InsertAlertLink :exec
INSERT INTO alert_links (code, alert_history_id, destination_url)
VALUES ($1, $2, $3);

-- name: RecordAlertLinkClick :one
-- Counts a click and returns where to send the visitor, in one round trip.
UPDATE alert_links
SET clicks           = clicks + 1,
    first_clicked_at = COALESCE(first_clicked_at, NOW()),
    last_clicked_at  = NOW()
WHERE code = $1
RETURNING destination_url;

-- name: GetAlertLinkDestination :one
-- For requests that should not count as a click, such as link previews.
SELECT destination_url
FROM alert_links
WHERE code = $1;

-- name: GetAlertLinkDetails :one
-- What the alert link page shows: the storm, never the person it was sent to.
SELECT r.uuid          AS resort_uuid,
       r.name          AS resort_name,
       h.snow_amount,
       h.forecast_date,
       l.destination_url
FROM alert_links l
         JOIN alert_history h ON h.id = l.alert_history_id
         JOIN resorts r ON r.uuid = h.resort_uuid
WHERE l.code = $1;
