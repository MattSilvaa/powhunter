# Alertmanager carrying this repository's routing configuration.
#
# The webhook URL is a secret and stays out of the image: the entrypoint
# substitutes it at start-up from ALERT_WEBHOOK_URL, because Alertmanager
# expands no environment variables in its configuration.
FROM prom/alertmanager:latest

COPY alertmanager.yml /etc/alertmanager/alertmanager.yml
COPY entrypoint.sh /entrypoint.sh

USER root
# Render SSH starts the user's login shell, and the base image gives nobody
# /bin/false, so every SSH session closed the moment it opened. nobody keeps
# its uid, so nothing it already owns changes.
RUN chmod +x /entrypoint.sh && \
    sed -i 's|^nobody:\(.*\):/bin/false$|nobody:\1:/bin/sh|' /etc/passwd
USER nobody

ENTRYPOINT ["/entrypoint.sh"]
