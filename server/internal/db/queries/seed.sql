-- name: ClearResorts :exec
DELETE FROM resorts;

-- name: InsertResort :one
INSERT INTO resorts (
  uuid, name, url_host, url_pathname, latitude, longitude
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;