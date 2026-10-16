#!/usr/bin/env bash
# Writes a release version into the operator and the Helm chart.
#
# Usage: scripts/release-bump.sh <version>
#
# Updates OperatorVersion in cmd/main.go and version/appVersion in the chart's Chart.yaml.
# Run `make helm-sync-docs` afterwards to refresh the version badges in the chart README.
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <version>" >&2
  exit 2
fi

version="${1#v}"
main_go="cmd/main.go"
chart="charts/coralogix-operator/Chart.yaml"

# sed -i.bak works with both GNU and BSD sed.
sed -i.bak -E "s/^const OperatorVersion = \".*\"$/const OperatorVersion = \"$version\"/" "$main_go"
sed -i.bak -E -e "s/^version:.*/version: $version/" -e "s/^appVersion:.*/appVersion: $version/" "$chart"
rm -f "$main_go.bak" "$chart.bak"

grep -qxF "const OperatorVersion = \"$version\"" "$main_go" ||
  { echo "error: OperatorVersion not found in $main_go" >&2; exit 1; }
grep -qxF "version: $version" "$chart" ||
  { echo "error: version not found in $chart" >&2; exit 1; }
grep -qxF "appVersion: $version" "$chart" ||
  { echo "error: appVersion not found in $chart" >&2; exit 1; }
