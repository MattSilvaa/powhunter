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

# Defaults are the compose addresses. On Render every one of these is supplied
# by render.yaml from a fromService reference, because Render appends a
# generated suffix to a service's internal hostname and exposes it on port
# 10000: "pushgateway:9091" resolves to nothing there, which is how the first
# deploy of this stack collected no metrics at all.
API_TARGET="${API_TARGET:-host.docker.internal:8080}"
PUSHGATEWAY_TARGET="${PUSHGATEWAY_TARGET:-pushgateway:9091}"
ALERTMANAGER_TARGET="${ALERTMANAGER_TARGET:-alertmanager:9093}"

# A plain substitution rather than envsubst, which the base image does not ship.
sed -e "s|__API_TARGET__|${API_TARGET}|g" \
    -e "s|__PUSHGATEWAY_TARGET__|${PUSHGATEWAY_TARGET}|g" \
    -e "s|__ALERTMANAGER_TARGET__|${ALERTMANAGER_TARGET}|g" \
    /etc/prometheus/prometheus.yml > "$CONFIG_FILE"

# An unsubstituted placeholder means a target silently points at nothing, so
# fail loudly here instead of running blind.
if grep -q '__[A-Z_]*TARGET__' "$CONFIG_FILE"; then
    echo "failed to substitute a target into the Prometheus config" >&2
    grep -n '__[A-Z_]*TARGET__' "$CONFIG_FILE" >&2
    exit 1
fi

echo "scraping api=${API_TARGET} pushgateway=${PUSHGATEWAY_TARGET} alertmanager=${ALERTMANAGER_TARGET}" >&2

exec /bin/prometheus \
    --config.file="$CONFIG_FILE" \
    --storage.tsdb.path=/prometheus \
    --storage.tsdb.retention.time="${PROMETHEUS_RETENTION:-15d}" \
    --web.enable-lifecycle \
    "$@"
