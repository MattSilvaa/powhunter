-- name: ListResorts :many
SELECT * FROM resorts
ORDER BY name;

-- name: GetResortByUUID :one
SELECT * FROM resorts
WHERE uuid = $1 LIMIT 1;
