-- migrations/009_snow_report.sql
--
-- The home page shows real snow numbers instead of a sample card.
--
-- resort_daily_snowfall holds the forecaster's latest daily snowfall for each
-- resort: the past three weeks and the coming days. Every run overwrites the
-- days it fetched, so a row is always the newest value for that date. The API
-- reads it to decide between the live report and the season recap, which
-- keeps Open-Meteo out of the page-view path entirely.
--
-- resort_season_stats holds last season's totals per resort, computed once a
-- season from Open-Meteo's historical archive.
-- +goose Up

CREATE TABLE resort_daily_snowfall (
    resort_uuid UUID NOT NULL REFERENCES resorts(uuid) ON DELETE CASCADE,
    date DATE NOT NULL,
    snowfall_in DOUBLE PRECISION NOT NULL CHECK (snowfall_in >= 0),
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (resort_uuid, date)
);

-- The report and the prune both filter on date across every resort.
CREATE INDEX idx_resort_daily_snowfall_date ON resort_daily_snowfall(date);

CREATE TABLE resort_season_stats (
    resort_uuid UUID NOT NULL REFERENCES resorts(uuid) ON DELETE CASCADE,
    -- Written as "2025-26".
    season TEXT NOT NULL,
    season_start DATE NOT NULL,
    season_end DATE NOT NULL,
    total_snowfall_in DOUBLE PRECISION NOT NULL,
    days_6in_plus INTEGER NOT NULL,
    biggest_storm_in DOUBLE PRECISION NOT NULL,
    biggest_storm_start DATE,
    biggest_storm_end DATE,
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (resort_uuid, season)
);

-- +goose Down
DROP TABLE IF EXISTS resort_season_stats;
DROP TABLE IF EXISTS resort_daily_snowfall;
