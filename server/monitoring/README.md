# Monitoring

Prometheus, Alertmanager and Pushgateway run on Render as **private services**:
none is reachable from the internet. Grafana is not deployed. It runs locally
from `docker-compose.yml` against whichever Prometheus you point it at, which
avoids paying for an always-on service you would only ever open from a desk.

Alerts do not depend on any of this. Alertmanager posts to `ALERT_WEBHOOK_URL`
whether or not a dashboard is open — the dashboard is for diagnosis after the
page, not for noticing the problem.

## Looking at production dashboards

### Register an SSH key first, once

Render authenticates SSH by public key only. Without one registered you get
`Permission denied (publickey)`, which reads like a broken service rather than a
missing key:

1. `cat ~/.ssh/id_ed25519.pub` — or `ssh-keygen -t ed25519` if you have none
2. Paste it at https://dashboard.render.com/settings#ssh-public-keys
3. Allow a minute to propagate

### Open the tunnel

Prometheus has no public address, so reach it through Render SSH. Take the
address from the service's dashboard page; it changes if the service is
recreated.

```sh
# Prometheus. Forwards local 9091 to Prometheus's 9090 inside Render.
ssh -N -L 9091:localhost:9090 srv-daep03v40ujc7385eqrg@ssh.oregon.render.com

# Alertmanager, for the smoke test below. A different service, so a different
# address — reusing the Prometheus one is the easy mistake.
ssh -N -L 9093:localhost:9093 srv-daep03v40ujc7385eqs0@ssh.oregon.render.com
```

Leave the tunnel running in its own terminal. Note local 9091 is also what the
compose Pushgateway binds, so do not run that service while a tunnel is open.

### Run Grafana against it

```sh
cd server
PROMETHEUS_URL=http://host.docker.internal:9091 docker compose up grafana
```

`PROMETHEUS_URL` is what points Grafana at the tunnel instead of the local
Prometheus it would otherwise default to. Omit it and the dashboard renders
against an empty local database and looks broken rather than empty.

Grafana is on http://localhost:3001, admin password from `GRAFANA_PASSWORD`
(`admin` by default locally). The Powhunter folder holds the provisioned
dashboard; it is read from `monitoring/grafana/dashboards/` on every start, so
edits made in the browser are not saved back to the repository.

**If every panel says "Datasource prometheus was not found"**, delete Grafana's
volume and start again:

```sh
docker compose down && docker volume rm powhunter_grafana_data
```

Grafana copies provisioned datasources into a SQLite database inside
`grafana_data`, so a datasource change in this repository does not necessarily
reach a Grafana that has already started once — the stale copy survives a plain
restart and produces exactly this message while the datasource page reports a
healthy connection. Nothing is lost by deleting the volume: the dashboards come
from files and the login comes from the environment.

### Doing this without disturbing your working branch

The dashboards live on `main`, which is rarely the branch you are working on. A
worktree checks it out beside your clone and leaves your working tree, staged
changes and untracked files completely alone:

```sh
git worktree add ~/code/powhunter-monitoring main
cd ~/code/powhunter-monitoring/server
# ...tunnel and docker compose up grafana as above
git worktree remove ~/code/powhunter-monitoring   # when finished
```

Keep it under your home directory rather than `/tmp`: Docker Desktop bind-mounts
only from shared paths, and `$HOME` is shared by default while `/tmp` may not
be.

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
| `PUSHGATEWAY_URL` | forecaster cron | The Pushgateway's **internal address from its Render dashboard**, e.g. `http://pushgateway-cyuc:9091`; unset, run metrics go nowhere |

The cron is not in `render.yaml`, so it cannot use `fromService` and the address
has to be copied by hand. Take the port from Prometheus's start-up line
(`scraping … pushgateway=…`) rather than from the service page: Render shows
`:10000` there while the service actually listens on 9091, which is the port
`fromService` reports and the one Prometheus is scraping. Push to a different
port than Prometheus scrapes and the metrics land nowhere anything reads.

## Checking it actually works

An alerting stack that has never fired is not known to work. Every check below
is a positive signal — something arriving — because every failure this stack has
had looked like silence, and silence is also what a healthy idle system looks
like.

With the Prometheus tunnel open:

```sh
# Every target and its health, one line each. All should read "up".
curl -s localhost:9091/api/v1/targets \
  | jq -r '.data.activeTargets[] | "\(.labels.job) \(.scrapeUrl) \(.health) \(.lastError)"'

# Did the last forecaster run's metrics arrive?
curl -s 'localhost:9091/api/v1/query?query=powhunter_forecaster_last_success_timestamp_seconds' \
  | jq '.data.result'
```

A value from that second query is worth more than it looks: it proves the cron
pushed, the Pushgateway held it, and Prometheus scraped the Pushgateway — three
links at once. Convert it with `date -u -d @<value>` and it should match the
run's own `forecast check complete` log line. An empty result while the
`powhunter-forecaster` target is `up` means `PUSHGATEWAY_URL` is not set on the
cron.

The API scrape can also be confirmed without any tunnel, from the API's own
logs, which fill with one of these every 15 seconds:

```
"msg":"request","method":"GET","path":"/metrics","status":200
```

401 means `METRICS_TOKEN` does not match the API's; nothing at all means
`API_TARGET` is wrong.

Finally, prove delivery. This is the link that fails silently while everything
upstream looks healthy, and no other check covers it — with the Alertmanager
tunnel open:

```sh
curl -XPOST localhost:9093/api/v2/alerts -H 'Content-Type: application/json' -d '[{
  "labels":{"alertname":"SmokeTest","severity":"critical","service":"api"},
  "annotations":{"summary":"Testing delivery end to end"}
}]'
```

An empty `{}` means Alertmanager accepted it; the message should reach the
webhook within about 30 seconds. Accepted but never delivered points at
`alertmanager.yml` or `ALERT_WEBHOOK_URL`, not at Prometheus.

`ForecasterNotRunning` fires thirteen hours after the last success, so it is the
slowest to test and the most important: it is the alert that catches a silent
alert pipeline.

## A note on Render addressing

Render does not address services by the name you give them: it appends a
generated suffix and exposes the service on port 10000, so `pushgateway:9091`
resolves to nothing there. That is why the first deploy of this stack scraped
nothing at all. `render.yaml` asks Render for each address with `fromService`
rather than hardcoding one, which also survives a service being recreated.
