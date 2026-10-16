#!/usr/bin/env bash
# Commits files from the working tree onto a remote branch with the GitHub GraphQL API.
#
# Usage: scripts/signed-commit.sh <branch> <expected-head-sha> <message> <file>...
#
# GitHub signs commits created by createCommitOnBranch, so they pass required-signature rules
# without a GPG key in CI. The commit lands only if the branch head is still
# <expected-head-sha>. Prints the new commit sha.
#
# Needs gh authenticated (GH_TOKEN) and GITHUB_REPOSITORY set to owner/name.
set -euo pipefail

if [[ $# -lt 4 ]]; then
  echo "usage: $0 <branch> <expected-head-sha> <message> <file>..." >&2
  exit 2
fi

branch="$1"
head="$2"
message="$3"
shift 3
repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set to owner/name}"

additions=$(for file in "$@"; do
  jq -n --arg path "$file" --arg contents "$(base64 <"$file" | tr -d '\n')" \
    '{path: $path, contents: $contents}'
done | jq -s .)

query='mutation($input: CreateCommitOnBranchInput!) {
  createCommitOnBranch(input: $input) { commit { oid } }
}'

jq -n \
  --arg query "$query" \
  --arg repo "$repo" \
  --arg branch "$branch" \
  --arg head "$head" \
  --arg message "$message" \
  --argjson additions "$additions" \
  '{
    query: $query,
    variables: {
      input: {
        branch: {repositoryNameWithOwner: $repo, branchName: $branch},
        expectedHeadOid: $head,
        message: {headline: $message},
        fileChanges: {additions: $additions}
      }
    }
  }' |
  gh api graphql --input - --jq '.data.createCommitOnBranch.commit.oid'
