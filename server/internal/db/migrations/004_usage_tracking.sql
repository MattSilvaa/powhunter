-- migrations/004_usage_tracking.sql
--
-- Supports the usage metrics. Twilio bills per message segment, not per alert,
-- so the segment count of each delivered SMS is what turns "alerts sent" into
-- a cost. Rows sent before this column existed are counted as one segment,
-- which is what the original alert wording fits in.
-- +goose Up

ALTER TABLE alert_history
    ADD COLUMN sms_segments INTEGER NOT NULL DEFAULT 1
        CHECK (sms_segments > 0);

-- The usage metrics count rows inside recent windows on every refresh.
CREATE INDEX idx_alert_history_sent_at ON alert_history(sent_at);
CREATE INDEX idx_users_created_at ON users(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_alert_history_sent_at;
ALTER TABLE alert_history DROP COLUMN IF EXISTS sms_segments;
