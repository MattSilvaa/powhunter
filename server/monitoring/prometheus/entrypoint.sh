#!/bin/sh
# Prometheus does not expand environment variables in its configuration, so the
# scrape token has to reach it as a file. Writing it at start-up keeps the
# secret in Render's environment rather than in the repository or the image.
set -eu

TOKEN_FILE=/tmp/metrics_token

if [ -z "${METRICS_TOKEN:-}" ]; then
    echo "METRICS_TOKEN is not set; the API scrape will be rejected" >&2
fi

printf '%s' "${METRICS_TOKEN:-}" > "$TOKEN_FILE"
chmod 600 "$TOKEN_FILE"

exec /bin/prometheus \
    --config.file=/etc/prometheus/prometheus.yml \
    --storage.tsdb.path=/prometheus \
    --storage.tsdb.retention.time="${PROMETHEUS_RETENTION:-15d}" \
    --web.enable-lifecycle \
    "$@"
