#!/usr/bin/env bash
# Checks that a diff is a release PR from `make release-pr` and nothing else: it changes only
# the version lines in cmd/main.go, Chart.yaml and the chart README, from the base version to a
# higher one. The Release PR Approval workflow approves a PR that passes.
#
# Usage: scripts/check-release-pr.sh <base-version> < diff
set -euo pipefail

base_version="$1"
chart="charts/coralogix-operator/Chart.yaml"
expected=$(printf '%s\n' cmd/main.go "$chart" charts/coralogix-operator/README.md | sort)

files=()
removed=()
added=()
while IFS= read -r line; do
  case "$line" in
    "diff --git "*) files+=("${line##* b/}") ;;
    "--- "* | "+++ "*) ;;
    -*) removed+=("${line:1}") ;;
    +*) added+=("${line:1}") ;;
  esac
done

fail() {
  echo "not a release PR: $*" >&2
  exit 1
}

if [[ "$(printf '%s\n' "${files[@]}" | sort)" != "$expected" ]]; then
  fail "it changes $(printf '%s ' "${files[@]}")instead of exactly $(echo "$expected" | tr '\n' ' ')"
fi

new_version=""
for line in "${added[@]}"; do
  if [[ "$line" =~ ^version:\ *([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    new_version="${BASH_REMATCH[1]}"
  fi
done
[[ -n "$new_version" ]] || fail "no new version in $chart"
if ! printf '%s\n' "${removed[@]}" | grep -qx "version: *$base_version"; then
  fail "it doesn't start from $base_version on the base branch; merge the base branch and run 'make bump' again"
fi
if [[ "$new_version" == "$base_version" ||
  "$(printf '%s\n%s\n' "$new_version" "$base_version" | sort -V | tail -n1)" != "$new_version" ]]; then
  fail "version $new_version is not above $base_version on the base branch"
fi

# Each removed line, with the base version replaced by the new one, must be the added line.
if [[ ${#removed[@]} -ne ${#added[@]} ]]; then
  fail "it removes ${#removed[@]} lines and adds ${#added[@]}"
fi
for i in "${!removed[@]}"; do
  if [[ "${removed[$i]//"$base_version"/$new_version}" != "${added[$i]}" ]]; then
    fail "the change '${removed[$i]}' -> '${added[$i]}' is not a version bump"
  fi
done

echo "ok: release PR for $new_version"
