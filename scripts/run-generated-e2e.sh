#!/usr/bin/env bash
# Runs the generated live e2e specs of the opgen kinds on a local Kind cluster.
#
# It builds the cluster and the operator image, installs the chart, and runs the specs that you
# pick. The work is done by scripts/run-e2e.sh. That script asks for the Coralogix API key with a
# hidden prompt and for the team id, so the key never enters the shell history or the transcript.
#
# usage: scripts/run-generated-e2e.sh [generated|coverage|all] [Kind ...]
#   generated  the full, update, delete, and minimal specs of each kind (default)
#   coverage   the coverage steps: the enum values and oneOf arms that the full spec does not send
#   all        both
# Kind names are the kinds of tools/opgen/kinds.yaml. Without a Kind, every generated kind runs.
#
# examples:
#   scripts/run-generated-e2e.sh coverage                  # every coverage step
#   scripts/run-generated-e2e.sh all Preset GlobalRouter   # both suites of two kinds
#
# The run takes about 3 minutes of setup, and then about 1 minute for each step.
# Environment: CLUSTER_NAME, IMAGE_TAG, and CORALOGIX_REGION (default EU2), as in run-e2e.sh.
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)

mode=${1:-generated}
if [[ $# -gt 0 ]]; then
  shift
fi
case "$mode" in
  generated | coverage | all) ;;
  *)
    echo "usage: $0 [generated|coverage|all] [Kind ...]" >&2
    exit 2
    ;;
esac

kinds=("$@")
if [[ ${#kinds[@]} -eq 0 ]]; then
  while IFS= read -r kind; do
    kinds+=("$kind")
  done < <(sed -n 's/^  - kind: //p' "$REPO_ROOT/tools/opgen/kinds.yaml")
fi

# The Describe names are "<Kind> (generated)" and "<Kind> (coverage)".
suites=()
case "$mode" in
  generated) suites=(generated) ;;
  coverage) suites=(coverage) ;;
  all) suites=(generated coverage) ;;
esac
focus=""
for kind in "${kinds[@]}"; do
  for suite in "${suites[@]}"; do
    # Ginkgo matches the focus against the whole spec text.
    focus+="${focus:+|}${kind} \\(${suite}\\)"
  done
done

echo "focus: $focus"
exec "$REPO_ROOT/scripts/run-e2e.sh" "$focus"
