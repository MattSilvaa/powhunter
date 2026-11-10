#!/usr/bin/env bash
# Small wrapper around the Render API for the deploy workflow.
#
#   render.sh live <service-id>              print the live deploy's id
#   render.sh deploy <service-id> <commit>   deploy a commit and wait for it
#   render.sh rollback <service-id> <deploy> roll back to an earlier deploy
#
# Needs RENDER_API_KEY. RENDER_API_URL and RENDER_WAIT_SECONDS can be
# overridden for testing.
set -euo pipefail

API="${RENDER_API_URL:-https://api.render.com/v1}"
WAIT_SECONDS="${RENDER_WAIT_SECONDS:-1200}"
POLL_SECONDS="${RENDER_POLL_SECONDS:-10}"

: "${RENDER_API_KEY:?RENDER_API_KEY is not set}"

render() {
	local method="$1" path="$2" body="${3:-}"
	local args=(-fsS --retry 3 --retry-all-errors -X "$method"
		-H "Authorization: Bearer $RENDER_API_KEY"
		-H "Accept: application/json")
	if [ -n "$body" ]; then
		args+=(-H "Content-Type: application/json" -d "$body")
	fi
	curl "${args[@]}" "$API$path"
}

live_deploy() {
	render GET "/services/$1/deploys?limit=20" |
		jq -r '[.[].deploy | select(.status == "live")][0].id // empty'
}

# wait_for <service-id> <deploy-id>: succeeds once the deploy is live.
wait_for() {
	local service="$1" deploy="$2" status
	local deadline=$((SECONDS + WAIT_SECONDS))

	while [ "$SECONDS" -lt "$deadline" ]; do
		status="$(render GET "/services/$service/deploys/$deploy" | jq -r '.status')"
		echo "deploy $deploy: $status" >&2
		case "$status" in
		live) return 0 ;;
		build_failed | update_failed | pre_deploy_failed | canceled | deactivated)
			echo "::error::Render deploy $deploy ended as $status" >&2
			return 1
			;;
		esac
		sleep "$POLL_SECONDS"
	done

	echo "::error::Render deploy $deploy was not live after ${WAIT_SECONDS}s" >&2
	return 1
}

case "${1:-}" in
live)
	live_deploy "$2"
	;;
deploy)
	service="$2" commit="$3"
	deploy="$(render POST "/services/$service/deploys" \
		"$(jq -nc --arg c "$commit" '{commitId: $c, clearCache: "do_not_clear"}')" |
		jq -r '.id')"
	echo "started deploy $deploy of $commit on $service" >&2
	wait_for "$service" "$deploy"
	;;
rollback)
	service="$2" target="$3"
	if [ "$(live_deploy "$service")" = "$target" ]; then
		echo "$service is already on $target; nothing to roll back" >&2
		exit 0
	fi
	deploy="$(render POST "/services/$service/rollback" \
		"$(jq -nc --arg d "$target" '{deployId: $d}')" | jq -r '.id')"
	echo "rolling $service back to $target (deploy $deploy)" >&2
	wait_for "$service" "$deploy"
	;;
*)
	echo "usage: $0 live|deploy|rollback ..." >&2
	exit 2
	;;
esac
