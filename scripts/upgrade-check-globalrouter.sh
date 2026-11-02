#!/usr/bin/env bash
# Create a router with a released operator. Remove it if a later step fails.
set -euo pipefail

mode="${1:?usage: $0 before|cleanup}"
cr="upgrade-test-global-router"
namespace="default"
id_file="${RUNNER_TEMP:-/tmp}/upgrade-test-global-router.id"

jsonpath() {
    kubectl get globalrouter "$cr" -n "$namespace" -o jsonpath="$1"
}

gone() {
    ! kubectl get globalrouter "$cr" -n "$namespace" >/dev/null 2>&1
}

wait_for() {
    for _ in $(seq 1 90); do
        if "$@"; then
            return 0
        fi
        sleep 2
    done
    echo "Timed out waiting for the GlobalRouter."
    kubectl get globalrouter "$cr" -n "$namespace" -o jsonpath="{.status.printableStatus}" || true
    return 1
}

has_id() {
    [ -n "$(jsonpath '{.status.id}')" ]
}

case "$mode" in
before)
    name="${NAME:?NAME is required}"
    kubectl apply -f - <<YAML
apiVersion: coralogix.com/v1alpha1
kind: GlobalRouter
metadata:
  name: $cr
  namespace: $namespace
spec:
  name: $name
  description: Router created before the operator upgrade
  routingLabels:
    environment: $name
YAML
    wait_for has_id
    jsonpath '{.status.id}' > "$id_file"
    echo "The released operator created the GlobalRouter."
    ;;
cleanup)
    if gone; then
        echo "No GlobalRouter to clean up."
        exit 0
    fi
    kubectl delete globalrouter "$cr" -n "$namespace" --wait=false
    if wait_for gone; then
        echo "The operator deleted the GlobalRouter."
        exit 0
    fi
    echo "::warning::The upgrade test left a GlobalRouter. Delete it by hand."
    ;;
*)
    echo "usage: $0 before|cleanup"
    exit 2
    ;;
esac
