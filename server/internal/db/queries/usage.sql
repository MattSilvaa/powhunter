-- name: GetUsageTotals :one
-- Point-in-time counts for the usage metrics. A subscriber is a user with at
-- least one active alert: that is who the forecaster can actually notify.
SELECT (SELECT COUNT(*) FROM users)::bigint                                    AS users,
       (SELECT COUNT(*) FROM users WHERE phone IS NOT NULL AND phone <> '')::bigint AS users_with_phone,
       (SELECT COUNT(DISTINCT user_uuid) FROM user_alerts WHERE active)::bigint AS subscribers,
       (SELECT COUNT(*) FROM user_alerts WHERE active)::bigint                  AS active_alerts;

-- name: GetUsageSince :one
-- Activity since a cutoff. Called once per reporting window.
WITH window_start AS (SELECT sqlc.arg(since)::timestamptz AS since)
SELECT (SELECT COUNT(*)
        FROM users u, window_start w
        WHERE u.created_at >= w.since)::bigint                  AS signups,
       (SELECT COUNT(*)
        FROM users u, window_start w
        WHERE u.created_at >= w.since
          AND u.referral_code IS NOT NULL)::bigint              AS signups_referred,
       (SELECT COUNT(DISTINCT s.user_uuid)
        FROM sessions s, window_start w
        WHERE s.last_seen_at >= w.since)::bigint                AS active_users,
       (SELECT COUNT(*)
        FROM alert_history h, window_start w
        WHERE h.sent_at >= w.since)::bigint                     AS alerts_sent,
       (SELECT COUNT(DISTINCT h.user_uuid)
        FROM alert_history h, window_start w
        WHERE h.sent_at >= w.since)::bigint                     AS users_alerted,
       (SELECT COALESCE(SUM(h.sms_segments), 0)
        FROM alert_history h, window_start w
        WHERE h.sent_at >= w.since)::bigint                     AS sms_segments,
       (SELECT COUNT(*)
        FROM alert_links l, window_start w
        WHERE l.created_at >= w.since)::bigint                  AS links_sent,
       -- Links sent in the window that were opened at least once, whenever the
       -- click came. Dividing by links_sent gives the click-through rate.
       (SELECT COUNT(*)
        FROM alert_links l, window_start w
        WHERE l.created_at >= w.since
          AND l.clicks > 0)::bigint                             AS links_clicked;

-- name: GetActiveAlertsByResort :many
-- How many users follow each resort. Every resort is listed, including those
-- nobody follows, so a resort dropping to zero shows as zero rather than as a
-- series that silently disappears.
SELECT r.name,
       COUNT(a.id)::bigint AS active_alerts
FROM resorts r
         LEFT JOIN user_alerts a ON a.resort_uuid = r.uuid AND a.active
GROUP BY r.name
ORDER BY r.name;

-- name: GetSignupsBySource :many
-- New accounts since a cutoff, per signup source. Accounts with no tagged
-- source are counted as "direct", and those from a shared alert link as
-- "alert_share" unless the link was also tagged.
SELECT COALESCE(signup_source,
                CASE WHEN referral_code IS NOT NULL THEN 'alert_share' ELSE 'direct' END)::text AS source,
       COUNT(*)::bigint AS signups
FROM users
WHERE created_at >= sqlc.arg(since)::timestamptz
GROUP BY 1
ORDER BY signups DESC, source;
