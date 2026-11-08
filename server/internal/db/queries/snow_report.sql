-- name: UpsertDailySnowfall :exec
INSERT INTO resort_daily_snowfall (resort_uuid, date, snowfall_in, fetched_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (resort_uuid, date) DO UPDATE
SET snowfall_in = EXCLUDED.snowfall_in,
    fetched_at  = EXCLUDED.fetched_at;

-- name: PruneDailySnowfall :execrows
-- Only the last few weeks matter to the report; older rows are dead weight.
DELETE FROM resort_daily_snowfall
WHERE date < $1;

-- name: ListDailySnowfallSince :many
SELECT r.uuid AS resort_uuid,
       r.name AS resort_name,
       d.date,
       d.snowfall_in,
       d.fetched_at
FROM resort_daily_snowfall d
JOIN resorts r ON r.uuid = d.resort_uuid
WHERE d.date >= $1
ORDER BY r.name, d.date;

-- name: UpsertSeasonStats :exec
INSERT INTO resort_season_stats (
    resort_uuid, season, season_start, season_end, total_snowfall_in,
    days_6in_plus, biggest_storm_in, biggest_storm_start, biggest_storm_end, computed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (resort_uuid, season) DO UPDATE
SET season_start        = EXCLUDED.season_start,
    season_end          = EXCLUDED.season_end,
    total_snowfall_in   = EXCLUDED.total_snowfall_in,
    days_6in_plus       = EXCLUDED.days_6in_plus,
    biggest_storm_in    = EXCLUDED.biggest_storm_in,
    biggest_storm_start = EXCLUDED.biggest_storm_start,
    biggest_storm_end   = EXCLUDED.biggest_storm_end,
    computed_at         = NOW();

-- name: ListSeasonStats :many
SELECT r.name AS resort_name,
       s.season,
       s.season_start,
       s.season_end,
       s.total_snowfall_in,
       s.days_6in_plus,
       s.biggest_storm_in,
       s.biggest_storm_start,
       s.biggest_storm_end,
       s.computed_at
FROM resort_season_stats s
JOIN resorts r ON r.uuid = s.resort_uuid
WHERE s.season = $1
ORDER BY s.total_snowfall_in DESC, r.name;

-- name: ListResortsMissingSeasonStats :many
SELECT r.*
FROM resorts r
WHERE NOT EXISTS (
    SELECT 1 FROM resort_season_stats s
    WHERE s.resort_uuid = r.uuid AND s.season = $1
)
ORDER BY r.name;
