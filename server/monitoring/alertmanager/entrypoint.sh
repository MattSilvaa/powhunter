#!/bin/sh
# Alertmanager expands no environment variables in its configuration. The
# committed config used ${ALERT_WEBHOOK_URL} as though it did, so every alert
# would have been posted to that literal string and silently dropped: the stack
# would have looked healthy while delivering nothing. Substituting here is what
# makes the receiver real.
set -eu

CONFIG_FILE=/tmp/alertmanager.yml

if [ -z "${ALERT_WEBHOOK_URL:-}" ]; then
    echo "ALERT_WEBHOOK_URL is not set; alerts will fire but reach nobody" >&2
fi

sed "s|__ALERT_WEBHOOK_URL__|${ALERT_WEBHOOK_URL:-}|g" \
    /etc/alertmanager/alertmanager.yml > "$CONFIG_FILE"

if grep -q '__ALERT_WEBHOOK_URL__' "$CONFIG_FILE"; then
    echo "failed to substitute ALERT_WEBHOOK_URL into the Alertmanager config" >&2
    exit 1
fi

exec /bin/alertmanager \
    --config.file="$CONFIG_FILE" \
    --storage.path=/alertmanager \
    "$@"
