#!/usr/bin/env bash
# Records the answers of the real Coralogix API for the baseline fidelity cases of one kind.
# It writes internal/controller/coralogix/baseline/testdata/<kind>/fidelity/recorded.json.
# The cases create resources in the team of the API key, and delete them at the end.
#
# Usage: scripts/record-baseline-fidelity.sh <kind> <region-or-domain>
#   scripts/record-baseline-fidelity.sh viewfolder EU2
#   scripts/record-baseline-fidelity.sh viewfolder coralogix.com
#
# The script asks for the API key. It does not read it from the environment or a file.
set -euo pipefail

if [ $# -ne 2 ]; then
  echo "usage: $0 <kind> <region-or-domain>" >&2
  exit 2
fi
kind=$1
target=$2

cd "$(dirname "$0")/.."

upper=$(printf '%s' "$target" | tr '[:lower:]' '[:upper:]')
unset CORALOGIX_REGION CORALOGIX_DOMAIN
case "$upper" in
  AP1|AP2|AP3|EU1|EU2|US1|US2|US3) export CORALOGIX_REGION="$upper" ;;
  *) export CORALOGIX_DOMAIN="$target" ;;
esac

printf 'Coralogix API key: '; read -rs CORALOGIX_API_KEY; echo; export CORALOGIX_API_KEY

RECORD_FIDELITY="$kind" go test ./internal/controller/coralogix/baseline \
  -run '^TestRecordFidelity$' -count=1 -v

echo "Wrote internal/controller/coralogix/baseline/testdata/$kind/fidelity/recorded.json"
echo "Check it with: go test ./internal/controller/coralogix/baseline -run TestFakeMatchesRecordedAPI"
