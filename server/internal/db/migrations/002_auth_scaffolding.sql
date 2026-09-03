-- migrations/002_auth_scaffolding.sql
--
-- Schema groundwork for user signup and login. No application code reads these
-- columns yet; landing them separately keeps the authentication change focused
-- on behaviour rather than schema.
-- +goose Up

-- Credential and verification state for the accounts that already exist.
ALTER TABLE users
    ADD COLUMN password_hash TEXT,
    ADD COLUMN email_verified_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

-- Sessions are looked up by a hash of the cookie value, never the value itself,
-- so a database leak does not hand over usable sessions.
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_uuid ON sessions(user_uuid);
-- Supports reaping expired sessions without a full scan.
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Bound the alert preferences at the database level so an out-of-range value
-- cannot be stored even if a future caller skips validation. A zero or negative
-- threshold would alert on every forecast.
ALTER TABLE user_alerts
    ADD CONSTRAINT user_alerts_min_snow_amount_check
        CHECK (min_snow_amount > 0 AND min_snow_amount <= 100),
    ADD CONSTRAINT user_alerts_notification_days_check
        CHECK (notification_days >= 1 AND notification_days <= 30);

-- Application logic already avoids sending the same alert twice, but nothing
-- enforced it. Collapse any historical duplicates, then make them impossible.
DELETE FROM alert_history a
    USING alert_history b
WHERE a.id < b.id
  AND a.user_uuid = b.user_uuid
  AND a.resort_uuid = b.resort_uuid
  AND a.forecast_date = b.forecast_date;

ALTER TABLE alert_history
    ADD CONSTRAINT alert_history_user_resort_date_key
        UNIQUE (user_uuid, resort_uuid, forecast_date);

-- These duplicate indexes that already exist implicitly, or are prefixes of
-- composite indexes above, so they cost write throughput for no read benefit.
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_uuid;
DROP INDEX IF EXISTS idx_resorts_uuid;
DROP INDEX IF EXISTS idx_user_alerts_user_id;
DROP INDEX IF EXISTS idx_user_alerts_resort_uuid;

-- +goose Down
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_resorts_uuid ON resorts(uuid);
CREATE INDEX idx_user_alerts_user_id ON user_alerts(user_uuid);
CREATE INDEX idx_user_alerts_resort_uuid ON user_alerts(resort_uuid);

ALTER TABLE alert_history DROP CONSTRAINT IF EXISTS alert_history_user_resort_date_key;

ALTER TABLE user_alerts
    DROP CONSTRAINT IF EXISTS user_alerts_min_snow_amount_check,
    DROP CONSTRAINT IF EXISTS user_alerts_notification_days_check;

DROP TABLE IF EXISTS sessions;

ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS email_verified_at,
    DROP COLUMN IF EXISTS updated_at;
