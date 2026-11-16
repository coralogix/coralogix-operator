#!/usr/bin/env bash
# Prepare one EU2 CI account, or look up its team and integration ids, then run a command.
# The SDK pin is the only copy of this SHA. Do not follow a branch.

set -euo pipefail
set +x
umask 077

sdk_commit=2ceba7bde4d1b47b5281855e6f56b655b3d9357a
sdk_url=https://github.com/coralogix/coralogix-management-sdk.git
sdk_dir=
fixture_dir=

cleanup() {
  if [ -n "$sdk_dir" ]; then
    rm -rf "$sdk_dir"
  fi
  if [ -n "$fixture_dir" ]; then
    rm -rf "$fixture_dir"
  fi
}
trap cleanup EXIT

usage() {
  echo 'usage: ci-account.sh prepare | ci-account.sh lookup command [args...]' >&2
  exit 2
}

require_env() {
  local name
  for name in "$@"; do
    if [ -z "${!name:-}" ]; then
      echo 'A team input is missing' >&2
      exit 1
    fi
  done
}

checkout_sdk() {
  sdk_dir="$(mktemp -d)"
  git -C "$sdk_dir" init -q
  git -C "$sdk_dir" remote add origin "$sdk_url"
  git -C "$sdk_dir" fetch -q --depth 1 origin "$sdk_commit"
  git -C "$sdk_dir" checkout -q --detach FETCH_HEAD
}

build_image() {
  local sdk_dir="$1"
  local context="$sdk_dir/tools/prepare-ci-team"
  # GitHub Actions sets ACTIONS_CACHE_URL. The same scope shares one image
  # cache across the prepare and lookup jobs. A local run has no cache.
  if [ -n "${ACTIONS_CACHE_URL:-}" ]; then
    if ! docker buildx inspect ci-prep >/dev/null 2>&1; then
      docker buildx create --name ci-prep --driver docker-container >/dev/null 2>&1
    fi
    if ! docker buildx build --builder ci-prep --load \
      --cache-from "type=gha,scope=prepare-ci-team-${sdk_commit}" \
      --cache-to "type=gha,mode=max,scope=prepare-ci-team-${sdk_commit}" \
      -t prepare-ci-team:ci \
      "$context" >/dev/null 2>&1; then
      echo 'Preparation image build failed' >&2
      exit 1
    fi
    return
  fi
  if ! docker build --quiet -t prepare-ci-team:ci "$context" >/dev/null 2>&1; then
    echo 'Preparation image build failed' >&2
    exit 1
  fi
}

prepare_account() {
  require_env \
    CORALOGIX_API_KEY \
    CORALOGIX_SEND_YOUR_DATA_API_KEY \
    PAGERDUTY_REST_API_KEY \
    PAGERDUTY_USER_EMAIL \
    CI_PREP_AI_APPLICATION_PREFIX \
    ARCHIVE_LOGS_BUCKET
  checkout_sdk
  build_image "$sdk_dir"
  if ! docker run --rm --read-only \
    --tmpfs /tmp:rw,nosuid,nodev,mode=1777,size=32m \
    --cap-drop=ALL --security-opt no-new-privileges \
    -e CORALOGIX_API_KEY \
    -e CORALOGIX_SEND_YOUR_DATA_API_KEY \
    -e PAGERDUTY_REST_API_KEY \
    -e PAGERDUTY_USER_EMAIL \
    -e CI_PREP_AI_APPLICATION_PREFIX \
    -e ARCHIVE_LOGS_BUCKET \
    prepare-ci-team:ci >/dev/null 2>&1; then
    echo 'EU2 account preparation failed' >&2
    exit 1
  fi
  echo 'EU2 account preparation passed'
}

read_lookup_ids() {
  python3 - "$1" <<'PY'
import json
import re
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    fixtures = json.load(handle)
team_id = fixtures.get("teamId")
slack_id = fixtures.get("slackIntegrationId")
pagerduty_id = fixtures.get("pagerDutyIntegrationId")
connector_key = fixtures.get("pagerDutyConnectorKey")
uuid = re.compile(r"[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}\Z")
ids = (slack_id, pagerduty_id, connector_key)
if (
    not isinstance(team_id, str)
    or not re.fullmatch(r"(?:0|[1-9][0-9]*)", team_id)
    or int(team_id) > 4294967295
    or any(not isinstance(value, str) or not uuid.fullmatch(value) for value in ids)
):
    raise SystemExit(1)
sys.stdout.write("\t".join((team_id, slack_id, pagerduty_id)))
PY
}

lookup_and_run() {
  require_env CORALOGIX_API_KEY
  if [ "$#" -eq 0 ]; then
    usage
  fi
  local fields
  checkout_sdk
  fixture_dir="$(mktemp -d)"
  chmod 700 "$fixture_dir"
  build_image "$sdk_dir"
  if ! docker run --rm --read-only \
    --tmpfs /tmp:rw,nosuid,nodev,mode=1777,size=32m \
    --cap-drop=ALL --security-opt no-new-privileges \
    --user "$(id -u):$(id -g)" \
    -e CORALOGIX_API_KEY \
    -e PREPARE_CI_TEAM_OUTPUT_FILE=/out/fixtures.json \
    -v "$fixture_dir:/out" \
    prepare-ci-team:ci lookup >/dev/null 2>&1; then
    echo 'Integration lookup failed' >&2
    exit 1
  fi
  if ! fields="$(read_lookup_ids "$fixture_dir/fixtures.json" 2>/dev/null)"; then
    echo 'Integration lookup output was invalid' >&2
    exit 1
  fi
  IFS=$'\t' read -r TEST_TEAM_ID SLACK_INTEGRATION_ID PD_INTEGRATION_ID <<< "$fields"
  export TEST_TEAM_ID SLACK_INTEGRATION_ID PD_INTEGRATION_ID
  "$@"
}

mode="${1:-}"
if [ "$#" -gt 0 ]; then
  shift
fi
case "$mode" in
  prepare)
    if [ "$#" -ne 0 ]; then
      usage
    fi
    prepare_account
    ;;
  lookup)
    lookup_and_run "$@"
    ;;
  *)
    usage
    ;;
esac
