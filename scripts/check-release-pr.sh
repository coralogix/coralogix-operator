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

fail() {
  echo "not a release PR: $*" >&2
  exit 1
}

# A file's header runs from its "diff --git" line to its first hunk; only lines inside hunks
# count as changes, so an added "++ ..." line can't pass for a "+++" header. A header line
# other than index and the plain ---/+++ lines (a mode change, rename, new or deleted file) is
# rejected.
files=()
removed=()
added=()
path=""
in_header=false
while IFS= read -r line; do
  if [[ "$line" == "diff --git "* ]]; then
    path="${line##* b/}"
    files+=("$path")
    in_header=true
  elif $in_header; then
    case "$line" in
      "@@ "*) in_header=false ;;
      "index "* | "--- a/$path" | "+++ b/$path") ;;
      *) fail "unexpected header line in $path: $line" ;;
    esac
  else
    case "$line" in
      -*) removed+=("${line:1}") ;;
      +*) added+=("${line:1}") ;;
      " "* | "@@ "* | "\\ No newline at end of file") ;;
      *) fail "unexpected diff line in $path: $line" ;;
    esac
  fi
done

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
