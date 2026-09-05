#!/bin/sh
# Prometheus expands no environment variables in its configuration, so both the
# scrape token and the API's address have to be resolved before it starts.
# Doing it here keeps one prometheus.yml serving both local compose and Render,
# and keeps the token in the environment rather than the repository or image.
set -eu

TOKEN_FILE=/tmp/metrics_token
CONFIG_FILE=/tmp/prometheus.yml

if [ -z "${METRICS_TOKEN:-}" ]; then
    echo "METRICS_TOKEN is not set; the API scrape will be rejected with 401" >&2
fi

printf '%s' "${METRICS_TOKEN:-}" > "$TOKEN_FILE"
chmod 600 "$TOKEN_FILE"

# host.docker.internal reaches the API when it runs on the developer's machine;
# on Render the API is another service, addressed by name over the internal
# network.
API_TARGET="${API_TARGET:-host.docker.internal:8080}"

# A plain substitution rather than envsubst, which the base image does not ship.
sed "s|__API_TARGET__|${API_TARGET}|g" /etc/prometheus/prometheus.yml > "$CONFIG_FILE"

if grep -q '__API_TARGET__' "$CONFIG_FILE"; then
    echo "failed to substitute API_TARGET into the Prometheus config" >&2
    exit 1
fi

echo "scraping the API at ${API_TARGET}" >&2

exec /bin/prometheus \
    --config.file="$CONFIG_FILE" \
    --storage.tsdb.path=/prometheus \
    --storage.tsdb.retention.time="${PROMETHEUS_RETENTION:-15d}" \
    --web.enable-lifecycle \
    "$@"
