-- migrations/007_signup_source.sql
--
-- Where a new account came from, read from the utm_source and utm_campaign
-- tags on the link the visitor first arrived by, such as a Reddit post tagged
-- utm_source=reddit&utm_campaign=cosnow. Counting signups per source shows
-- which marketing is worth repeating.
--
-- Plain text, and only set when an account is created, like referral_code: an
-- anonymous form supplies it, and a missing or odd value must never fail the
-- signup.
-- +goose Up

ALTER TABLE users ADD COLUMN signup_source TEXT;
ALTER TABLE users ADD COLUMN signup_campaign TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS signup_campaign;
ALTER TABLE users DROP COLUMN IF EXISTS signup_source;
