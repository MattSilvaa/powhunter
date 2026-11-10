# Deploying

Merging to `main` deploys. Nobody deploys by hand, and nothing reaches
production without passing the same checks a pull request does.

```
pull request ──▶ Test: unit, integration, lint, client, functional
                    │ merge
                    ▼
push to main ──▶ Deploy: Test again ─▶ API ─▶ web app ─▶ smoke test prod ─▶ forecaster
                                         │ any failure
                                         ▼
                              roll back API and web app
                              open a "deploy-failure" issue
```

## The stages

**Functional tests** (`server/functional`). Black-box tests that call the app
over HTTP. In CI they run against a stack built on the runner: Postgres, the
migrated schema, the API and the static web build. They cover health, the
resort list, the snow report, security headers, that the web pages and their
scripts load, and the main user flow: sign up, sign in with the emailed link,
see and delete alerts, sign out. The sign-in link comes from the API log,
because without `RESEND_API_KEY` mail is logged instead of sent.

**Deploy** (`.github/workflows/deploy.yml`). Runs on every push to `main`, one
at a time, never cancelled midway:

1. Runs the full `Test` workflow again on the merged commit.
2. Records which release each service is on now.
3. Deploys that exact commit to the API, then the web app, through the Render
   API (`.github/scripts/render.sh`). The API's pre-deploy command runs
   migrations, so a failed migration stops the release before it takes traffic.
4. Runs the functional suite against production in read-only mode. It first
   waits for `/health` to report the new commit, so it cannot pass by testing
   the old release. Tests that write data are skipped, because they only run
   when they can read the API's mail log, which only CI has.
5. Deploys the forecaster last, so the job that texts people only ever runs
   code that passed step 4.

**Rollback.** If any step after step 2 fails, the API and web app go back to the
releases recorded in step 2. A service whose new build failed was never
replaced, so it is left alone.

**Issue.** Any failure opens an issue labelled `deploy-failure` with links to
the commit and the run, or comments on the one already open. The next
successful deploy closes it. GitHub emails you about new issues by default.

## One-time setup

Already in place: the API's pre-deploy command runs `./bin/migrate`, and the
workflow knows the Render service IDs (override them with repository variables
`RENDER_API_SERVICE_ID`, `RENDER_WEB_SERVICE_ID`, `RENDER_FORECASTER_SERVICE_ID`
if a service is ever recreated). Three steps remain, all in a browser:

1. **Add the Render API key to GitHub.** In Render, Account Settings → API Keys
   → Create API Key. In GitHub, Settings → Secrets and variables → Actions →
   New repository secret, named `RENDER_API_KEY`.
2. **Turn off auto-deploy in Render** for `powhunter-public-api`,
   `powhunter-web-static` and `powhunter-forecaster-cron-job` (Settings →
   Build & Deploy → Auto-Deploy: Off). Otherwise Render deploys every push
   itself, before any test has run. Leave the monitoring services as they are.
   Do this after step 1, so there is no window where nothing deploys.
3. **Protect `main`.** Settings → Rules → Rulesets (or Branches), require a pull
   request and the `Test` checks (Server, Server lint, Client, Functional) to
   pass before merging. A broken change is then stopped at the pull request, and
   the deploy-time run is a second line of defence.

`API_URL` and `WEB_URL` repository variables override the default production
addresses, `https://api.powhunter.app` and `https://powhunter.app`.

## Writing migrations that can be rolled back

A rollback reverts code, not the database. After a rollback the previous
release runs against the new schema, so every migration must work with the
code before it:

- Add columns as nullable or with a default; add tables freely.
- Never rename or drop in the same release that stops using something. Ship the
  code that stops using it first, then drop it in a later release.

## Cost

Nothing new to pay for. GitHub Actions is free for public repositories, the
Render API is free, and there is no staging server: the functional tests run
against a throwaway stack on the CI runner instead.

## Running it locally

```bash
make test-functional   # against the API on :8080 and the web app on :3000
```

To deploy or roll back by hand, use `.github/scripts/render.sh` with
`RENDER_API_KEY` set, or rerun the workflow from the Actions tab.
