#!/usr/bin/env bash
# Prints the live e2e shard matrix, or checks that every e2e spec runs in one shard.
#
# tests/e2e/live-shards.yaml lists the Describe names that stay together.
# The shared shard runs those specs. The rest shard is the catch-all: its skip
# regex is the shared regex, so it runs every other spec that `make e2e-tests`
# runs. A spec that matches no shared name is not dropped. Coverage specs and
# the Group SCIM spec stay out of both shards. They have their own jobs.
#
# usage: scripts/e2e-shards.sh matrix
#        scripts/e2e-shards.sh check
# GINKGO names the ginkgo binary (default bin/ginkgo). SHARDS_FILE names the shard file.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"

shards_file=${SHARDS_FILE:-tests/e2e/live-shards.yaml}
ginkgo=${GINKGO:-bin/ginkgo}
# Keep this equal to the default of E2E_GINKGO_LABEL_FILTER in the Makefile.
label_filter='!scim-migration && !coverage'

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is missing" >&2
  exit 2
fi

# pattern prints the ginkgo regex for the shared shard.
# Ginkgo matches the suite description plus the spec text, so the Describe name
# is not at the start of the string. The name is matched as a word followed by
# a space. That keeps View apart from ViewFolder, and Alert apart from AlertSet.
pattern() {
  local names name alt dups
  names=$(grep -v '^#' "$shards_file" | jq -r '.shared[]')
  if [ -z "$names" ]; then
    echo "shared shard lists no specs" >&2
    exit 1
  fi
  dups=$(printf '%s\n' "$names" | sed '/^$/d' | sort | uniq -d)
  if [ -n "$dups" ]; then
    echo "shared shard lists a name twice:" >&2
    printf '%s\n' "$dups" >&2
    exit 1
  fi
  alt=""
  while IFS= read -r name; do
    if [ -z "$name" ]; then
      continue
    fi
    if ! [[ "$name" =~ ^[A-Za-z0-9]+$ ]]; then
      echo "shard name has characters a regex would read specially: $name" >&2
      exit 1
    fi
    if [ -n "$alt" ]; then
      alt="${alt}|${name}"
    else
      alt="$name"
    fi
  done <<< "$names"
  # A trailing space, not $, so GNU make does not treat $ as a variable.
  printf '%s' "(?:^| )(?:${alt}) "
}

matrix() {
  local focus
  focus=$(pattern)
  # rest skips the shared regex and sets no focus, so it is the catch-all.
  jq -n --arg focus "$focus" '{
    include: [
      {name: "shared", focus: $focus, skip: ""},
      {name: "rest", focus: "", skip: $focus}
    ]
  }'
}

# specs_for <focus> <skip> prints the full text of each selected spec, sorted.
specs_for() {
  local focus=$1 skip=$2
  local -a args
  args=(--dry-run --label-filter="$label_filter" --json-report="$tmp/report.json")
  if [ -n "$focus" ]; then
    args+=(--focus "$focus")
  fi
  if [ -n "$skip" ]; then
    args+=(--skip "$skip")
  fi
  # Flags go before the package. Ginkgo rejects a flag that follows it.
  args+=(./tests/e2e/)
  if ! "$ginkgo" "${args[@]}" > "$tmp/ginkgo.log" 2>&1; then
    cat "$tmp/ginkgo.log" >&2
    exit 1
  fi
  jq -r '.[].SpecReports[]
    | select(.LeafNodeType == "It" and .State == "passed")
    | (.ContainerHierarchyTexts + [.LeafNodeText]) | join(" ")' "$tmp/report.json" | sort
}

check() {
  local makefile_filter focus
  makefile_filter=$(sed -n 's/^E2E_GINKGO_LABEL_FILTER ?= //p' Makefile)
  if [ "$makefile_filter" != "$label_filter" ]; then
    echo "E2E_GINKGO_LABEL_FILTER is '$makefile_filter'. The shard check expects '$label_filter'." >&2
    exit 1
  fi
  if [[ ! -x "$ginkgo" ]]; then
    echo "ginkgo is missing: run 'make ginkgo' or set GINKGO" >&2
    exit 2
  fi
  focus=$(pattern)
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  specs_for "" "" > "$tmp/all.txt"
  specs_for "$focus" "" > "$tmp/shared.txt"
  specs_for "" "$focus" > "$tmp/rest.txt"
  # The catch-all is every spec the shared regex does not match.
  comm -23 "$tmp/all.txt" "$tmp/shared.txt" > "$tmp/catch-all.txt"
  echo "shared shard: $(wc -l < "$tmp/shared.txt" | tr -d ' ') specs"
  echo "rest shard (catch-all): $(wc -l < "$tmp/rest.txt" | tr -d ' ') specs"

  cat "$tmp/shared.txt" "$tmp/rest.txt" | sort > "$tmp/shard-sorted.txt"
  status=0
  if ! cmp -s "$tmp/catch-all.txt" "$tmp/rest.txt"; then
    echo "rest shard is not the catch-all. Specs the shared regex misses that the rest shard does not run:" >&2
    comm -23 "$tmp/catch-all.txt" "$tmp/rest.txt" >&2
    status=1
  fi
  missing=$(comm -23 "$tmp/all.txt" <(sort -u "$tmp/shard-sorted.txt"))
  if [ -n "$missing" ]; then
    echo "e2e specs that run in no shard:" >&2
    printf '%s\n' "$missing" >&2
    status=1
  fi
  doubled=$(uniq -d "$tmp/shard-sorted.txt")
  if [ -n "$doubled" ]; then
    echo "e2e specs that run in more than one shard:" >&2
    printf '%s\n' "$doubled" >&2
    status=1
  fi
  extra=$(comm -13 "$tmp/all.txt" <(sort -u "$tmp/shard-sorted.txt"))
  if [ -n "$extra" ]; then
    echo "shard specs that make e2e-tests does not run:" >&2
    printf '%s\n' "$extra" >&2
    status=1
  fi
  if [ "$status" -eq 0 ]; then
    echo "every e2e spec runs in exactly one shard ($(wc -l < "$tmp/all.txt" | tr -d ' ') specs)"
  fi
  exit "$status"
}

case "${1:-}" in
  matrix) matrix ;;
  check) check ;;
  *)
    echo "usage: scripts/e2e-shards.sh matrix|check" >&2
    exit 2
    ;;
esac
