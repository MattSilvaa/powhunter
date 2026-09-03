-- migrations/003_login_tokens.sql
--
-- Single-use tokens backing the magic-link login flow. Like sessions, only a
-- hash of the token is stored, so a database leak does not hand out logins.
-- +goose Up

CREATE TABLE login_tokens (
    id SERIAL PRIMARY KEY,
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    -- Set when the token is redeemed. Redemption is a conditional UPDATE on this
    -- column, which is what makes a token genuinely single-use: two concurrent
    -- redemptions of the same link cannot both match.
    consumed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_tokens_user_uuid ON login_tokens(user_uuid);
-- Supports reaping expired tokens without a full scan.
CREATE INDEX idx_login_tokens_expires_at ON login_tokens(expires_at);

-- +goose Down
DROP TABLE IF EXISTS login_tokens;
