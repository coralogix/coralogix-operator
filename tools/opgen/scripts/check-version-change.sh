#!/usr/bin/env bash
# Fails when the generator produces something new and version.Current stays the same.
#
# The generator version names what produced the generated files. The check builds opgen at the
# base commit and at HEAD, runs both on the same input, and compares what they produce:
#   - the files that `opgen generate` writes for every kind of tools/opgen/kinds.yaml;
#   - the `opgen check` report of every component of the pinned spec;
#   - the golden files of the generator tests, which hold the output for the synthetic specs.
# A change of code that changes none of these needs no new version. A change of only the kinds
# list or the files of a kind (tools/opgen/kinds.yaml, tools/opgen/kinds/) is input, not code, and
# tools/opgen/versions.yaml is output.
#
# usage: check-version-change.sh <base-commit>
# OPGEN_OLD and OPGEN_NEW name prebuilt binaries instead of building them, and SPEC names the
# spec instead of the pinned one (for the tests).
set -euo pipefail

base=${1:?usage: check-version-change.sh <base-commit>}
root=$(git rev-parse --show-toplevel)
cd "$root"

changed=$(git diff --name-only "$base"...HEAD -- tools/opgen | grep -Ev '^tools/opgen/(scripts/|kinds\.yaml$|kinds/|versions\.yaml$)' || true)
if [[ -z "$changed" ]]; then
  exit 0
fi

version_path=tools/opgen/internal/version/version.go
version_at() {
  git show "$1:$version_path" 2>/dev/null | awk -F'"' '/^[[:space:]]*const Current = "[^"]+"[[:space:]]*$/ { print $2 }'
}

base_version=$(version_at "$base" || true)
head_version=$(version_at HEAD || true)
if [[ -z "$head_version" ]]; then
  printf '%s\n' 'Generator code changed, but version.Current is missing or invalid.' >&2
  exit 1
fi
if [[ "$base_version" != "$head_version" ]]; then
  exit 0 # a new version is always allowed, and a new generator has no base to compare
fi

fail() {
  printf 'The generator output changed, but version.Current is still %s. Give the generator a new version.\n' "$head_version" >&2
  printf '%s\n' "$@" >&2
  exit 1
}

golden=$(git diff --name-only "$base"...HEAD -- tools/opgen/internal/generator/testdata/golden tools/opgen/internal/generator/testdata/crd)
if [[ -n "$golden" ]]; then
  fail "The golden files changed:" "$golden"
fi

work=$(mktemp -d)
cleanup() {
  git worktree remove --force "$work/base" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT

old=${OPGEN_OLD:-}
new=${OPGEN_NEW:-}
if [[ -z "$old" ]]; then
  git worktree add --quiet --detach "$work/base" "$base"
  (cd "$work/base/tools/opgen" && go build -o "$work/opgen-old" ./cmd/opgen)
  old=$work/opgen-old
fi
if [[ -z "$new" ]]; then
  (cd tools/opgen && go build -o "$work/opgen-new" ./cmd/opgen)
  new=$work/opgen-new
fi

# Both binaries read the same input: a copy of HEAD each, so one cannot see the files of the other.
for side in old new; do
  mkdir -p "$work/$side-tree" "$work/$side-out/components"
  git archive HEAD | tar -x -C "$work/$side-tree"
done

spec=${SPEC:-}
if [[ -z "$spec" ]]; then
  go mod download
  spec=$(go list -m -f '{{.Dir}}' github.com/coralogix/coralogix-management-sdk)/openapi.yaml
fi

# Each binary runs from the operator root, so it finds the pinned SDK from the operator go.mod.
for side in old new; do
  binary=$old
  [[ $side == new ]] && binary=$new
  (cd "$work/$side-tree" && "$binary" generate) >"$work/$side-out/generate.txt" 2>&1 || echo "exit $?" >>"$work/$side-out/generate.txt"
done

check() {
  local binary=$1 file=$2
  shift 2
  "$binary" check "$@" >"$file" 2>&1 || echo "exit $?" >>"$file"
}

components() {
  awk '
    /^components:[[:space:]]*$/ { in_components = 1; next }
    in_components && /^[^[:space:]]/ { exit }
    in_components && /^  schemas:[[:space:]]*$/ { in_schemas = 1; next }
    in_schemas && /^  [^[:space:]]/ { exit }
    in_schemas && /^    [^[:space:]#][^:]*:[[:space:]]*$/ { sub(/^    /, ""); sub(/:[[:space:]]*$/, ""); gsub(/["\047]/, ""); print }
  ' "$spec"
}
export -f check
export old new spec work
# shellcheck disable=SC2016 # the child shell expands them
components | xargs -P 8 -I{} bash -c '
  name=$1 file=$(printf "%s" "$1" | tr "/" "_").txt
  check "$old" "$work/old-out/components/$file" --resource "$name" --openapi "$spec"
  check "$new" "$work/new-out/components/$file" --resource "$name" --openapi "$spec"
' _ {}

# opgen writes all over the operator tree. A diff that cannot compare also counts as a difference.
differences=""
for pair in "old-tree new-tree" "old-out new-out"; do
  read -r left right <<<"$pair"
  if ! found=$(diff -r --brief "$work/$left" "$work/$right" 2>&1); then
    differences+="$found"$'\n'
  fi
done
if [[ -n "$differences" ]]; then
  fail "The base and HEAD generators produce different output:" "${differences//$work\//}"
fi
