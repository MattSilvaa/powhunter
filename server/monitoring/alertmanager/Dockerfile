# Alertmanager carrying this repository's routing configuration.
#
# The webhook URL is a secret and stays out of the image: the entrypoint
# substitutes it at start-up from ALERT_WEBHOOK_URL, because Alertmanager
# expands no environment variables in its configuration.
FROM prom/alertmanager:latest

COPY alertmanager.yml /etc/alertmanager/alertmanager.yml
COPY entrypoint.sh /entrypoint.sh

USER root
RUN chmod +x /entrypoint.sh
USER nobody

ENTRYPOINT ["/entrypoint.sh"]
