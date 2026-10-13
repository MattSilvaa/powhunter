-- migrations/005_alert_links.sql
--
-- A short link in each alert SMS, redirecting through the API to the resort's
-- mountain report. Counting the clicks shows which alerts people act on, which
-- is the number a resort or an affiliate partner will ask for.
--
-- One row per delivered alert, so a link always belongs to exactly one user,
-- resort and forecast. Clicks are a counter rather than a row each: the
-- question is whether an alert was acted on, not a clickstream.
-- +goose Up

CREATE TABLE alert_links (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    alert_history_id INTEGER NOT NULL UNIQUE REFERENCES alert_history(id) ON DELETE CASCADE,
    destination_url TEXT NOT NULL,
    clicks INTEGER NOT NULL DEFAULT 0 CHECK (clicks >= 0),
    first_clicked_at TIMESTAMP WITH TIME ZONE,
    last_clicked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- The usage metrics count links sent inside recent windows.
CREATE INDEX idx_alert_links_created_at ON alert_links(created_at);

-- +goose Down
DROP TABLE IF EXISTS alert_links;
