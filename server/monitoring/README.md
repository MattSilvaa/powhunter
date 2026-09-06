# Monitoring

Prometheus, Alertmanager and Pushgateway run on Render as **private services**:
none is reachable from the internet. Grafana is not deployed. It runs locally
from `docker-compose.yml` against whichever Prometheus you point it at, which
avoids paying for an always-on service you would only ever open from a desk.

Alerts do not depend on any of this. Alertmanager posts to `ALERT_WEBHOOK_URL`
whether or not a dashboard is open — the dashboard is for diagnosis after the
page, not for noticing the problem.

## Looking at production dashboards

Prometheus has no public address, so reach it through Render SSH:

```sh
# Forward local 9091 to Prometheus inside Render. Take the SSH address from the
# service's dashboard page.
ssh -N -L 9091:localhost:9090 <prometheus-service>@ssh.oregon.render.com
```

Then run Grafana locally against the tunnel:

```sh
cd server
PROMETHEUS_URL=http://host.docker.internal:9091 docker compose up grafana
```

Grafana is on http://localhost:3001, admin password from `GRAFANA_PASSWORD`
(`admin` by default locally). The Powhunter folder holds the provisioned
dashboard; it is read from `monitoring/grafana/dashboards/` on every start, so
edits made in the browser are not saved back to the repository.

## Running the whole stack locally

```sh
cd server
docker compose up prometheus alertmanager pushgateway grafana
```

Prometheus scrapes the API on the host at `host.docker.internal:8080`. Set
`METRICS_TOKEN` to the same value the API uses, or the scrape returns 401 and
the `powhunter-api` target shows as DOWN.

## Why the configs are baked into images

Render cannot mount repository files into a stock image. The alternative,
populating a disk by hand, is an unversioned manual step that no review sees and
no rebuild reproduces — the same shape as the migrations that existed for months
without ever being applied. Baking them means an alert threshold change is a
reviewable diff and the running config is whatever the last green build made.

Secrets are not baked. Neither Prometheus nor Alertmanager expands environment
variables in its configuration, so each has an `entrypoint.sh` that resolves
them at start-up:

| Variable | Service | Purpose |
|---|---|---|
| `METRICS_TOKEN` | prometheus | Bearer token for the API scrape; must match the API's |
| `API_TARGET` | prometheus | `host.docker.internal:8080` locally; on Render supplied by `fromService` |
| `PUSHGATEWAY_TARGET` | prometheus | `pushgateway:9091` locally; on Render supplied by `fromService` |
| `ALERTMANAGER_TARGET` | prometheus | `alertmanager:9093` locally; on Render supplied by `fromService` |
| `ALERT_WEBHOOK_URL` | alertmanager | Where alerts are delivered |
| `PUSHGATEWAY_URL` | forecaster cron | The Pushgateway's **internal address from its Render dashboard**, e.g. `http://pushgateway-cyuc:10000`; unset, run metrics go nowhere |

## Checking it actually works

An alerting stack that has never fired is not known to work.

- Prometheus targets page: `powhunter-api` should be **UP**. DOWN means either
  `API_TARGET` is wrong or `METRICS_TOKEN` does not match the API's.
- After a forecaster run, `powhunter_forecaster_last_success_timestamp_seconds`
  should be present. Absent means `PUSHGATEWAY_URL` is not set on the cron.
- Force an alert and confirm it reaches the webhook. `ForecasterNotRunning`
  fires thirteen hours after the last success, so it is the slowest to test and
  the most important: it is the alert that catches a silent alert pipeline.

## A note on Render addressing

Render does not address services by the name you give them: it appends a
generated suffix and exposes the service on port 10000, so `pushgateway:9091`
resolves to nothing there. That is why the first deploy of this stack scraped
nothing at all. `render.yaml` asks Render for each address with `fromService`
rather than hardcoding one, which also survives a service being recreated.
