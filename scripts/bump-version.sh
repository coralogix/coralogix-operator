#!/usr/bin/env bash
# Bumps the operator version: OperatorVersion in cmd/main.go and version/appVersion in the
# chart's Chart.yaml.
#
# Usage: scripts/bump-version.sh [patch|minor|major]   (default: patch)
#
# `make bump-version` runs this and then refreshes the version badges in the chart README.
set -euo pipefail

part="${1:-patch}"
main_go="cmd/main.go"
chart="charts/coralogix-operator/Chart.yaml"

current=$(sed -n -E 's/^version: *//p' "$chart")
operator=$(sed -n -E 's/^const OperatorVersion = "(.*)"$/\1/p' "$main_go")
if [[ "$current" != "$operator" ]]; then
  echo "error: $chart version $current and $main_go OperatorVersion $operator differ" >&2
  exit 1
fi
if [[ ! "$current" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "error: current version '$current' is not X.Y.Z without leading zeros" >&2
  exit 1
fi
major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
patch="${BASH_REMATCH[3]}"

case "$part" in
  patch) next="$major.$minor.$((patch + 1))" ;;
  minor) next="$major.$((minor + 1)).0" ;;
  major) next="$((major + 1)).0.0" ;;
  *)
    echo "usage: $0 [patch|minor|major]" >&2
    exit 2
    ;;
esac

# sed -i.bak works with both GNU and BSD sed.
sed -i.bak -E "s/^const OperatorVersion = \".*\"$/const OperatorVersion = \"$next\"/" "$main_go"
sed -i.bak -E -e "s/^version:.*/version: $next/" -e "s/^appVersion:.*/appVersion: $next/" "$chart"
rm -f "$main_go.bak" "$chart.bak"

echo "$current -> $next"
