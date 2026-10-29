#!/bin/bash
# Checks that a ViewFolder that the released operator created still works after the upgrade.
#
#   before:  create the ViewFolder with the released operator and wait for its id.
#   after:   with the new operator, wait for RemoteSynced with the same id, change the name, wait
#            for RemoteSynced of the new generation, then delete it and wait until it is gone.
#   cleanup: delete the ViewFolder if a failed step left it, through whichever operator runs, so
#            no run leaves the folder in Coralogix. It warns with the id when it cannot.
#
# NAME is the name of the folder in Coralogix; give each run its own.

set -euo pipefail

mode="${1:?usage: $0 before|after|cleanup}"
cr="upgrade-test-view-folder"
namespace="default"
name="${NAME:?NAME is required}"
id_file="${RUNNER_TEMP:-/tmp}/upgrade-test-view-folder.id"

# wait_for runs the condition until it succeeds, for at most 120 seconds.
wait_for() {
    for _ in $(seq 1 60); do
        if "$@"; then
            return 0
        fi
        sleep 2
    done
    echo "Timed out waiting for: $*"
    kubectl get viewfolder "$cr" -n "$namespace" -o yaml || true
    return 1
}

jsonpath() {
    kubectl get viewfolder "$cr" -n "$namespace" -o jsonpath="$1"
}

has_id() {
    [ -n "$(jsonpath '{.status.id}')" ]
}

synced() {
    local generation
    generation="$(jsonpath '{.metadata.generation}')"
    [ "$(jsonpath '{.status.conditions[?(@.type=="RemoteSynced")].status}')" = "True" ] &&
        [ "$(jsonpath '{.status.conditions[?(@.type=="RemoteSynced")].observedGeneration}')" = "$generation" ]
}

gone() {
    ! kubectl get viewfolder "$cr" -n "$namespace" >/dev/null 2>&1
}

case "$mode" in
before)
    kubectl apply -f - <<YAML
apiVersion: coralogix.com/v1alpha1
kind: ViewFolder
metadata:
  name: $cr
  namespace: $namespace
spec:
  name: $name
YAML
    wait_for has_id
    jsonpath '{.status.id}' > "$id_file"
    echo "The released operator created the folder $(cat "$id_file")."
    ;;
after)
    wait_for synced
    if [ "$(jsonpath '{.status.id}')" != "$(cat "$id_file")" ]; then
        echo "The id changed after the upgrade: $(cat "$id_file") -> $(jsonpath '{.status.id}')"
        exit 1
    fi
    kubectl patch viewfolder "$cr" -n "$namespace" --type merge -p "{\"spec\":{\"name\":\"$name-updated\"}}"
    wait_for synced
    kubectl delete viewfolder "$cr" -n "$namespace" --wait=false
    wait_for gone
    echo "The new operator synced, updated, and deleted the folder."
    ;;
cleanup)
    if gone; then
        echo "No folder to clean up."
        exit 0
    fi
    kubectl delete viewfolder "$cr" -n "$namespace" --wait=false
    if wait_for gone; then
        echo "The operator deleted the folder."
        exit 0
    fi
    echo "::warning::The upgrade test left the folder $(cat "$id_file" 2>/dev/null || echo "with an unknown id") in Coralogix. Delete it by hand."
    ;;
*)
    echo "usage: $0 before|after|cleanup"
    exit 2
    ;;
esac
