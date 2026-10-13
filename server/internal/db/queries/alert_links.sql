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
