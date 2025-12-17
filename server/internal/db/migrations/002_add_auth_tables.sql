-- migrations/002_add_auth_tables.sql
-- +goose Up

-- Table to store magic link tokens for authentication
CREATE TABLE magic_link_tokens (
    id SERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    purpose VARCHAR(50) NOT NULL DEFAULT 'login' -- 'login' or 'signup'
);

-- Table to store user sessions (JWT tokens)
CREATE TABLE user_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL UNIQUE,
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add verified email field to users table
ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN verified_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

-- Indexes for performance
CREATE INDEX idx_magic_link_tokens_token ON magic_link_tokens(token);
CREATE INDEX idx_magic_link_tokens_user_uuid ON magic_link_tokens(user_uuid);
CREATE INDEX idx_magic_link_tokens_expires_at ON magic_link_tokens(expires_at);
CREATE INDEX idx_user_sessions_session_id ON user_sessions(session_id);
CREATE INDEX idx_user_sessions_user_uuid ON user_sessions(user_uuid);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_user_sessions_expires_at;
DROP INDEX IF EXISTS idx_user_sessions_user_uuid;
DROP INDEX IF EXISTS idx_user_sessions_session_id;
DROP INDEX IF EXISTS idx_magic_link_tokens_expires_at;
DROP INDEX IF EXISTS idx_magic_link_tokens_user_uuid;
DROP INDEX IF EXISTS idx_magic_link_tokens_token;

ALTER TABLE users DROP COLUMN IF EXISTS verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;

DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS magic_link_tokens;