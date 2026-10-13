-- migrations/006_referrals.sql
--
-- The alert link page lets a subscriber text a storm to friends, with a signup
-- link carrying the code of the alert it came from. Recording that code on the
-- new account is what shows whether sharing brings people in.
--
-- Plain text rather than a foreign key: the code arrives from an anonymous
-- signup form, and a stale or mistyped one must never fail the signup itself.
-- +goose Up

ALTER TABLE users ADD COLUMN referral_code TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS referral_code;
