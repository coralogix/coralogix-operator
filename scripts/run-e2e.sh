#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
cd "$REPO_ROOT"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <ginkgo-focus>" >&2
  echo "Requires CORALOGIX_API_KEY and TEST_TEAM_ID (prompted if unset)." >&2
  exit 2
fi

FOCUS=$1
CLUSTER_NAME=${CLUSTER_NAME:-coralogix-operator-e2e}
IMAGE_REPOSITORY=${IMAGE_REPOSITORY:-coralogix-operator}
IMAGE_TAG=${IMAGE_TAG:-e2e-local}
CORALOGIX_REGION=${CORALOGIX_REGION:-EU2}
IMAGE="${IMAGE_REPOSITORY}:v${IMAGE_TAG}"

if [[ -z "${CORALOGIX_API_KEY:-}" ]]; then
  read -r -s -p "Coralogix API key: " CORALOGIX_API_KEY
  printf '\n'
fi

if [[ -z "$CORALOGIX_API_KEY" ]]; then
  echo "CORALOGIX_API_KEY cannot be empty" >&2
  exit 1
fi

if [[ -z "${TEST_TEAM_ID:-}" ]]; then
  read -r -p "Coralogix team ID (TEST_TEAM_ID): " TEST_TEAM_ID
fi

if [[ -z "$TEST_TEAM_ID" ]]; then
  echo "TEST_TEAM_ID cannot be empty" >&2
  exit 1
fi
export TEST_TEAM_ID

umask 077
API_KEY_FILE=$(mktemp)
trap 'rm -f "$API_KEY_FILE"' EXIT
printf '%s' "$CORALOGIX_API_KEY" > "$API_KEY_FILE"
export -n CORALOGIX_API_KEY

unset CORALOGIX_DOMAIN
export CORALOGIX_REGION

kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
kind create cluster --name "$CLUSTER_NAME"
kubectl config use-context "kind-$CLUSTER_NAME"

make docker-build IMG="$IMAGE"
kind load docker-image "$IMAGE" --name "$CLUSTER_NAME"

helm upgrade --install cx ./charts/coralogix-operator \
  --namespace coralogix-operator-system \
  --create-namespace \
  --set serviceMonitor.create=false \
  --set coralogixOperator.image.repository="$IMAGE_REPOSITORY" \
  --set coralogixOperator.image.tag="$IMAGE_TAG" \
  --set coralogixOperator.image.pullPolicy=IfNotPresent \
  --set coralogixOperator.prometheusRules.enabled=false \
  --set-file secret.data.apiKey="$API_KEY_FILE" \
  --set coralogixOperator.region="$CORALOGIX_REGION"

kubectl -n coralogix-operator-system rollout status \
  deployment/cx-coralogix-operator

export CORALOGIX_API_KEY
go test ./tests/e2e \
  -run '^TestE2E$' \
  -v \
  -ginkgo.v \
  -ginkgo.focus="$FOCUS"
