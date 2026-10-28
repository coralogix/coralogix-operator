#!/usr/bin/env bash
# Checks that a branch raises the operator version above the base branch, and that
# cmd/main.go and the chart agree on it. Every merge to main is released from that version.
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

if ! [[ "$chart_version" == "$app_version" && "$chart_version" == "$operator_version" ]]; then
  echo "error: versions differ; run 'make bump' instead of editing them by hand" >&2
  exit 1
fi

if [[ "$chart_version" == "$base_version" ||
  "$(printf '%s\n%s\n' "$chart_version" "$base_version" | sort -V | tail -n1)" != "$chart_version" ]]; then
  echo "error: version $chart_version is not above $base_version on $base." >&2
  echo "Run 'make bump' (patch), or 'make bump BUMP=minor' / 'make bump BUMP=major', and commit the result." >&2
  exit 1
fi

echo "ok: this merge releases $chart_version"
