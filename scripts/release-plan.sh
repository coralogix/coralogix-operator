#!/usr/bin/env bash
# Decides how to release a version, from the tags and release-x.y branches on the remote.
#
# Usage: scripts/release-plan.sh <version> [remote]
#
# Prints key=value lines for $GITHUB_OUTPUT:
#   version        X.Y.Z
#   tag            vX.Y.Z
#   kind           minor (X.Y.0, cut from main) or patch (X.Y.Z with Z > 0, on release-X.Y)
#   branch         release-X.Y
#   branch_exists  true when release-X.Y is already on the remote
#   latest         true when the version is above every released version
#
# A minor run whose branch already exists (a previous run cut it and then failed) reuses the
# branch instead of cutting it again.
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 <version> [remote]" >&2
  exit 2
fi

version="${1#v}"
remote="${2:-origin}"

if [[ ! "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "error: '$1' is not a release version (expected X.Y.Z, for example 2.16.0)" >&2
  exit 1
fi
major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
patch="${BASH_REMATCH[3]}"
tag="v$version"
branch="release-$major.$minor"

# Only plain vX.Y.Z tags count as releases.
tags=$(git ls-remote --tags --refs "$remote" 'v*' |
  sed -E 's#.*refs/tags/##' |
  grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true)

if grep -qxF "$tag" <<<"$tags"; then
  echo "error: tag $tag already exists" >&2
  exit 1
fi

branch_exists=false
if [[ -n "$(git ls-remote --heads "$remote" "refs/heads/$branch")" ]]; then
  branch_exists=true
fi

highest=$(sort -V <<<"$tags" | tail -n1)
highest="${highest#v}"

# True when $1 sorts strictly above $2 as a version.
version_gt() {
  [[ "$1" != "$2" && "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n1)" == "$1" ]]
}

latest=true
if [[ -n "$highest" ]] && ! version_gt "$version" "$highest"; then
  latest=false
fi

if [[ "$patch" == "0" ]]; then
  kind=minor
  # A new minor or major line is cut from main, which is ahead of every release line.
  if [[ "$latest" != "true" ]]; then
    echo "error: $version is not above the highest release v$highest; a new line must be the newest" >&2
    exit 1
  fi
else
  kind=patch
  if [[ "$branch_exists" != "true" ]]; then
    echo "error: $version is a patch release, but branch $branch does not exist" >&2
    exit 1
  fi
  previous="v$major.$minor.$((patch - 1))"
  if ! grep -qxF "$previous" <<<"$tags"; then
    echo "error: $version skips a patch: tag $previous does not exist" >&2
    exit 1
  fi
fi

cat <<EOF
version=$version
tag=$tag
kind=$kind
branch=$branch
branch_exists=$branch_exists
latest=$latest
EOF
