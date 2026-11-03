#!/usr/bin/env bash
# Opens a release PR: a branch from origin/main with the version bumped by `make bump-version`.
# Merging the PR releases the new version (see .github/workflows/release.yaml).
#
# Usage: scripts/release-pr.sh [patch|minor|major]   (default: patch)
#
# Run it through `make bump BUMP=...`. Needs a clean working tree. With `gh` installed it
# opens the PR; otherwise it prints what to open. Release PR Approval merges it once every check
# has passed.
set -euo pipefail

part="${1:-patch}"
case "$part" in
  patch | minor | major) ;;
  *)
    echo "usage: $0 [patch|minor|major]" >&2
    exit 2
    ;;
esac

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: the working tree has uncommitted changes; commit or set them aside first" >&2
  exit 1
fi

git fetch origin main
# A failed release must be retried before the next bump; the Version Bump check refuses it too
# (scripts/check-version-bump.sh).
current=$(git show origin/main:charts/coralogix-operator/Chart.yaml | sed -n -E 's/^version: *//p')
status=0
git ls-remote --exit-code --tags origin "refs/tags/v$current" > /dev/null || status=$?
if [[ "$status" == 2 ]]; then
  echo "error: v$current on main is not released yet. Retry the Release workflow from the Actions tab first, then run 'make bump' again" >&2
  exit 1
elif [[ "$status" != 0 ]]; then
  echo "error: could not look up the tag v$current on origin" >&2
  exit 1
fi
# CI refuses a release PR from anyone else (scripts/check-releaser.sh); this only says so early.
if command -v gh > /dev/null && login=$(gh api user --jq .login 2> /dev/null) &&
  releasers=$(git show origin/main:.github/RELEASERS 2> /dev/null); then
  listed=$(sed -e 's/#.*//' -e 's/[[:space:]]//g' <<< "$releasers")
  if ! grep -qixF "$login" <<< "$listed"; then
    echo "error: only the people in .github/RELEASERS (coralogix/team-eco-system) release the operator, and $login is not listed" >&2
    exit 1
  fi
fi
start=$(git symbolic-ref --quiet --short HEAD || git rev-parse HEAD)
git switch --quiet --detach origin/main
# On a failure, drop the bump and go back to where you were.
trap 'git checkout --quiet -- . && git checkout --quiet "$start"' ERR

make bump-version BUMP="$part"
version=$(sed -n -E 's/^version: *//p' charts/coralogix-operator/Chart.yaml)
# Not release-*: pushing a branch with that name publishes the chart (helm-chart-deploy.yml).
branch="prepare-v$version"
title="Release $version"
# -C and --force: a retry after a failed run replaces the branch that run left behind. The
# script creates the branch and nothing else writes to it.
git switch --quiet -C "$branch"
git commit --quiet -am "$title"
git push --force --set-upstream origin "$branch"
trap - ERR
git checkout --quiet "$start"

if ! command -v gh > /dev/null; then
  echo "Branch $branch is pushed. Open the PR against main: $title"
  echo "Release PR Approval merges it once every check on it has passed."
  exit 0
fi

if url=$(gh pr view "$branch" --json url,state --jq 'select(.state == "OPEN") | .url' 2> /dev/null) &&
  [[ -n "$url" ]]; then
  echo "The release PR is already open, now with this push: $url"
else
  gh pr create --base main --head "$branch" --title "$title" \
    --body "Releases $version. Merging this PR publishes the image, the Helm chart and the GitHub release."
fi

# No auto-merge: main requires no checks, so auto-merge would merge on any approval while tests
# still run. Release PR Approval merges the PR once every check has passed, then starts Release.
echo "Release PR Approval merges the PR once every check on it has passed; that releases $version."
