#!/usr/bin/env bash
# Runs on the production host (piped over SSH by .github/workflows/release.yml).
# Mirrors .deploy.sh: fast-forward the /srv git checkout to the released
# commit, rebuild the service, and restore the previous image on failure.
#
# Usage: deploy-remote.sh <target-commit> <expected-version>
set -euo pipefail

readonly TARGET_COMMIT="$1"
readonly EXPECTED_VERSION="$2"
readonly REMOTE_DIR="/srv/gtm-mcp-server"
readonly SERVICE="gtm-mcp"
readonly IMAGE="gtm-mcp-server:latest"
readonly ROLLBACK_IMAGE="gtm-mcp-server:rollback"
readonly HEALTH_URL="https://mcp.gtmeditor.com/health"
readonly COMPOSE=(docker compose --project-directory "$REMOTE_DIR" -f "$REMOTE_DIR/docker-compose.yml")

deployment_started=0
rollback_ready=0

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

rollback() {
    (( rollback_ready == 1 )) || return 0

    printf '==> Restoring rollback image\n' >&2
    docker image tag "$ROLLBACK_IMAGE" "$IMAGE" || return 1
    "${COMPOSE[@]}" up -d --no-build --force-recreate "$SERVICE" || return 1
}

# EXIT rather than ERR so failures reported through die() also roll back.
on_exit() {
    local exit_code=$?
    trap - EXIT
    (( exit_code == 0 )) && return 0

    printf 'error: deployment failed (exit %s)\n' "$exit_code" >&2
    if (( deployment_started == 1 )); then
        if rollback; then
            printf 'error: previous image restored; remote source remains at the new commit\n' >&2
        else
            printf 'error: automatic rollback also failed; inspect the service immediately\n' >&2
        fi
    fi
    exit "$exit_code"
}

# The script arrives on stdin (ssh ... bash -s), so run the body with stdin
# closed; otherwise a command that reads stdin could swallow the remaining lines.
main() {
    trap on_exit EXIT

    printf '==> Checking production preconditions\n'
    test -d "$REMOTE_DIR/.git"
    test -f "$REMOTE_DIR/docker-compose.yml"
    test -f "$REMOTE_DIR/.env"
    docker info >/dev/null

    [[ "$(git -C "$REMOTE_DIR" branch --show-current)" == main ]] || die "remote checkout must be on main"
    [[ -z "$(git -C "$REMOTE_DIR" status --porcelain --untracked-files=normal)" ]] || die "remote checkout is not clean"
    grep -Fxq "$SERVICE" <<< "$("${COMPOSE[@]}" config --services)" || die "Compose service '$SERVICE' was not found"

    git -C "$REMOTE_DIR" fetch --quiet origin main
    git -C "$REMOTE_DIR" cat-file -e "${TARGET_COMMIT}^{commit}"
    git -C "$REMOTE_DIR" merge-base --is-ancestor "$TARGET_COMMIT" origin/main \
        || die "release commit $TARGET_COMMIT is not on main"

    readonly REMOTE_COMMIT="$(git -C "$REMOTE_DIR" rev-parse HEAD)"
    printf '    remote: %s\n' "$REMOTE_COMMIT"
    printf '    target: %s\n' "$TARGET_COMMIT"
    printf '    version: %s\n' "$EXPECTED_VERSION"

    # .deploy.sh may already have shipped this commit, or a newer one from main.
    if git -C "$REMOTE_DIR" merge-base --is-ancestor "$TARGET_COMMIT" "$REMOTE_COMMIT"; then
        printf '==> Production already includes %s; nothing to deploy\n' "$TARGET_COMMIT"
        return 0
    fi
    git -C "$REMOTE_DIR" merge-base --is-ancestor "$REMOTE_COMMIT" "$TARGET_COMMIT" \
        || die "remote commit is not an ancestor of target; refusing a divergent deployment"

    readonly OLD_CONTAINER="$("${COMPOSE[@]}" ps -q "$SERVICE")"

    if docker image inspect "$IMAGE" >/dev/null 2>&1; then
        printf '==> Saving rollback image\n'
        docker image tag "$IMAGE" "$ROLLBACK_IMAGE"
        rollback_ready=1
    fi

    printf '==> Fast-forwarding production source\n'
    git -C "$REMOTE_DIR" merge --quiet --ff-only "$TARGET_COMMIT"

    printf '==> Building and restarting %s\n' "$SERVICE"
    deployment_started=1
    "${COMPOSE[@]}" up -d --build --remove-orphans --force-recreate "$SERVICE"

    container_id=''
    container_health=''
    for attempt in {1..36}; do
        container_id="$("${COMPOSE[@]}" ps -q "$SERVICE")"
        if [[ -n "$container_id" ]]; then
            container_health="$(docker inspect --format={{.State.Health.Status}} "$container_id")"
            if [[ "$container_health" == healthy || "$container_health" == running ]]; then
                break
            fi
        fi
        printf '    waiting for container health (%s/36, status=%s)\n' "$attempt" "${container_health:-missing}"
        sleep 5
    done

    [[ "$container_health" == healthy || "$container_health" == running ]] \
        || die "container did not become healthy"
    if [[ -n "$OLD_CONTAINER" ]]; then
        [[ "$container_id" != "$OLD_CONTAINER" ]] || die "container was not recreated"
    fi

    printf '==> Verifying public health endpoint\n'
    response=''
    actual_status=''
    actual_version=''
    for attempt in {1..24}; do
        response="$(curl -fsS --max-time 10 "$HEALTH_URL" || true)"
        actual_status="$(jq -r '.status // empty' <<< "$response" 2>/dev/null || true)"
        actual_version="$(jq -r '.version // empty' <<< "$response" 2>/dev/null || true)"
        if [[ "$actual_status" == healthy && "$actual_version" == "$EXPECTED_VERSION" ]]; then
            break
        fi
        printf '    waiting for public health (%s/24, status=%s, version=%s)\n' \
            "$attempt" "${actual_status:-missing}" "${actual_version:-missing}"
        sleep 5
    done

    [[ "$actual_status" == healthy ]] || die "public endpoint did not report healthy"
    [[ "$actual_version" == "$EXPECTED_VERSION" ]] \
        || die "public endpoint reports version '$actual_version', expected '$EXPECTED_VERSION'"

    deployment_started=0
    printf '==> Deployed %s (version %s) successfully\n' "$TARGET_COMMIT" "$EXPECTED_VERSION"
}

main "$@" </dev/null
