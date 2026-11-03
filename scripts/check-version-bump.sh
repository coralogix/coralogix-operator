#!/usr/bin/env bash
# Checks the operator version on a branch: cmd/main.go and the chart must agree on it, and a
# changed version must be above the base branch, whose version must already be released. Merging
# a raised version releases it; see `make bump`.
#
# Usage: scripts/check-version-bump.sh [base-ref]   (default: origin/main)
set -euo pipefail

base="${1:-origin/main}"
main_go="cmd/main.go"
chart="charts/coralogix-operator/Chart.yaml"

chart_version=$(sed -n -E 's/^version: *//p' "$chart")
app_version=$(sed -n -E 's/^appVersion: *//p' "$chart")
operator_version=$(sed -n -E 's/^const OperatorVersion = "(.*)"$/\1/p' "$main_go")
base_version=$(git show "$base:$chart" | sed -n -E 's/^version: *//p')

echo "base $base: $base_version"
echo "Chart.yaml version: $chart_version, appVersion: $app_version"
echo "$main_go OperatorVersion: $operator_version"

if [[ ! "$chart_version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "error: version '$chart_version' is not X.Y.Z without leading zeros; run 'make bump' instead of editing it by hand" >&2
  exit 1
fi

if ! [[ "$chart_version" == "$app_version" && "$chart_version" == "$operator_version" ]]; then
  echo "error: versions differ; run 'make bump' instead of editing them by hand" >&2
  exit 1
fi

if [[ "$chart_version" == "$base_version" ]]; then
  echo "ok: version unchanged; this merge releases nothing"
  exit 0
fi

if [[ "$(printf '%s\n%s\n' "$chart_version" "$base_version" | sort -V | tail -n1)" != "$chart_version" ]]; then
  echo "error: version $chart_version is below $base_version on $base." >&2
  echo "Merge $base into this branch and run 'make bump' again." >&2
  exit 1
fi

# A release that failed before its tag exists must be retried first: once a higher version is on
# the base branch, Release can only publish that one, and the failed version would be skipped.
status=0
git ls-remote --exit-code --tags origin "refs/tags/v$base_version" > /dev/null || status=$?
if [[ "$status" == 2 ]]; then
  echo "error: v$base_version on $base is not released yet. Retry the Release workflow first; once v$base_version is tagged, run 'make bump' again." >&2
  exit 1
elif [[ "$status" != 0 ]]; then
  echo "error: could not look up the tag v$base_version on origin" >&2
  exit 1
fi

echo "ok: this merge releases $chart_version"
