-- name: UpsertResort :one
-- Keyed on name so re-seeding keeps existing UUIDs and the alerts that
-- reference them, rather than deleting and recreating every resort.
INSERT INTO resorts (
  uuid, name, url_host, url_pathname, latitude, longitude
) VALUES (
  $1, $2, $3, $4, $5, $6
)
ON CONFLICT (name) DO UPDATE SET
  url_host = EXCLUDED.url_host,
  url_pathname = EXCLUDED.url_pathname,
  latitude = EXCLUDED.latitude,
  longitude = EXCLUDED.longitude
RETURNING *;
