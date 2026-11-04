#!/usr/bin/env bash
# Exits 0 only when CI on a release PR's commit has passed: the latest run of every workflow
# that a release PR starts, every other workflow run on the commit, and every check from outside
# GitHub Actions (CodeQL, GitGuardian). Exits 2 while CI still runs, 1 when a check failed, and 3
# when GitHub can't be read.
# Release PR Approval approves and merges only on 0, and Release waits for 0 before publishing:
# main requires only the live tests result, so GitHub doesn't wait for the other checks.
#
# Usage: scripts/check-release-ci.sh <sha>
# GH_TOKEN and GH_REPO read the runs.
set -euo pipefail

sha="$1"

# Every workflow that a release PR starts, also listed under workflow_run in
# release-pr-approval.yaml. All runs of one push are created together, so once these exist,
# none is missing.
required=(Container crd-validation helm-sync-check live-tests sanity unit-tests "Version Bump")

# Workflow runs, not check runs: several workflows name their job "Tests", and the check-runs API
# returns only the newest check of each name. A re-run is a new attempt of the same run.
runs=$(gh api --paginate "repos/$GH_REPO/actions/runs?head_sha=$sha&event=pull_request&per_page=100" \
  --jq '.workflow_runs[] | [.name, .created_at, .status, (.conclusion // "")] | @tsv') || exit 3
# The newest run of each workflow, for a commit that was pushed more than once.
latest=$(sort -t $'\t' -k1,1 -k2,2r <<< "$runs" | awk -F'\t' 'NF && !seen[$1]++')
checks=$(gh api --paginate "repos/$GH_REPO/commits/$sha/check-runs?per_page=100" \
  --jq '.check_runs[] | select(.app.slug != "github-actions") | [.name, "", .status, (.conclusion // "")] | @tsv') || exit 3

# Not piped into grep -q: under pipefail, grep stopping early can fail the command before it.
started=$(cut -f1 <<< "$latest")
for workflow in "${required[@]}"; do
  if ! grep -qxF "$workflow" <<< "$started"; then
    echo "Waiting: workflow '$workflow' has not started on $sha"
    exit 2
  fi
done
all=$(printf '%s\n%s\n' "$latest" "$checks" | awk 'NF')
pending=$(awk -F'\t' '$3 != "completed" {print $1}' <<< "$all")
if [[ -n "$pending" ]]; then
  echo "Waiting for: $(paste -sd, - <<< "$pending")"
  exit 2
fi
failed=$(awk -F'\t' '$4 != "success" && $4 != "skipped" && $4 != "neutral" {print $1 " (" $4 ")"}' <<< "$all")
if [[ -n "$failed" ]]; then
  echo "Failed: $(paste -sd, - <<< "$failed"). Re-run them; approval follows when they pass."
  exit 1
fi
echo "CI on $sha has passed."
