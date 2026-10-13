# Prometheus carrying this repository's scrape configuration and alert rules.
#
# Render cannot mount repository files into a stock image, and populating a disk
# by hand would be an unversioned manual step — the same shape as the migrations
# that existed but were never applied, which is how production ran for months
# against a schema nobody had checked. Baking the config makes a change to an
# alert threshold a reviewable diff, and makes the running config whatever the
# last green build produced.
#
# Secrets are not baked. METRICS_TOKEN and API_TARGET arrive as environment
# variables and are resolved by the entrypoint, because Prometheus expands no
# environment variables of its own.
FROM prom/prometheus:v3

COPY prometheus.yml /etc/prometheus/prometheus.yml
COPY rules /etc/prometheus/rules
COPY entrypoint.sh /entrypoint.sh

USER root
# Render SSH starts the user's login shell, and the base image gives nobody
# /bin/false, so every SSH session closed the moment it opened and the tunnel
# to Prometheus in README.md could not be used. nobody keeps its uid, so files
# it already owns, such as the Prometheus data disk, stay writable.
RUN chmod +x /entrypoint.sh && \
    sed -i 's|^nobody:\(.*\):/bin/false$|nobody:\1:/bin/sh|' /etc/passwd
USER nobody

ENTRYPOINT ["/entrypoint.sh"]
