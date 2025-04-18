-- migrations/001_create_schema.sql
-- +goose Up
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE resorts (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE,
    url_host VARCHAR(255),
    url_pathname VARCHAR(255),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION
);

CREATE TABLE user_alerts (
    id SERIAL PRIMARY KEY,
    user_uuid UUID REFERENCES users(uuid) ON DELETE CASCADE,
    resort_uuid UUID REFERENCES resorts(uuid) ON DELETE CASCADE,
    min_snow_amount INTEGER NOT NULL,
    notification_days INTEGER NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, resort_uuid)
);

CREATE TABLE alert_history (
    id SERIAL PRIMARY KEY,
    user_uuid UUID REFERENCES users(uuid) ON DELETE CASCADE,
    resort_uuid UUID REFERENCES resorts(uuid) ON DELETE CASCADE,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    forecast_date DATE NOT NULL,
    snow_amount INTEGER NOT NULL
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_resorts_uuid ON resorts(uuid);
CREATE INDEX idx_user_alerts_user_id ON user_alerts(user_uuid);
CREATE INDEX idx_user_alerts_resort_uuid ON user_alerts(resort_uuid);
CREATE INDEX idx_user_alerts_resort_uuid_active ON user_alerts(resort_uuid, active);
CREATE INDEX idx_user_alerts_active ON user_alerts(active);
CREATE INDEX idx_user_alerts_combined ON user_alerts(user_uuid, resort_uuid, active);
CREATE INDEX idx_alert_history_combined ON alert_history(user_uuid, resort_uuid, forecast_date);


-- +goose Down
DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS user_alerts;
DROP TABLE IF EXISTS resorts;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";