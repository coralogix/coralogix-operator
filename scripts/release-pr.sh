#!/usr/bin/env bash
# Opens a release PR: a branch from origin/main with the version bumped by `make bump-version`.
# Merging the PR releases the new version (see .github/workflows/release.yaml).
#
# Usage: scripts/release-pr.sh [patch|minor|major]   (default: patch)
#
# Run it through `make bump BUMP=...`. Needs a clean working tree. With `gh` installed it
# opens the PR and turns on auto-merge for it; otherwise it prints what to open.
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
  echo "Then turn on auto-merge (squash) for it, so it merges once Release PR Approval approves it."
  exit 0
fi

if url=$(gh pr view "$branch" --json url,state --jq 'select(.state == "OPEN") | .url' 2> /dev/null) &&
  [[ -n "$url" ]]; then
  echo "The release PR is already open, now with this push: $url"
else
  gh pr create --base main --head "$branch" --title "$title" \
    --body "Releases $version. Merging this PR publishes the image, the Helm chart and the GitHub release."
fi

# Auto-merge with your token, not the bot's: GitHub starts no workflows for a merge made with
# GITHUB_TOKEN, so Release would not run. The PR then merges once it is approved.
if gh pr merge "$branch" --auto --squash; then
  echo "Auto-merge is on: the PR merges once Release PR Approval approves it, and the merge releases $version."
else
  echo "warning: could not turn on auto-merge; merge the PR yourself once it is approved." >&2
fi
