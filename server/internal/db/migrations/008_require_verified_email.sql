-- migrations/008_require_verified_email.sql
--
-- Alerts are now only texted to accounts whose email address is confirmed.
-- Signup was anonymous, so anyone could enter any email with any phone number
-- and the forecaster would text that number, at our expense.
--
-- Every account that exists today signed up under the old rules and is already
-- receiving alerts. Marking them confirmed keeps those alerts flowing. Only
-- accounts created from now on have to click the emailed link first.
-- +goose Up

UPDATE users
SET email_verified_at = COALESCE(created_at, NOW())
WHERE email_verified_at IS NULL;

-- +goose Down
-- Nothing to undo: which accounts were confirmed by this migration rather than
-- by a link is not recorded, and un-confirming real users would stop their alerts.
SELECT 1;
