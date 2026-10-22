#!/bin/bash
set -e

NAMESPACE=${1:-coralogix-operator-system}

echo "Fetching operator logs from namespace: $NAMESPACE"

# Get all pod names in the namespace
PODS=$(kubectl get pods -n "$NAMESPACE" --no-headers -o custom-columns=":metadata.name")

# If no pods are found, exit
if [ -z "$PODS" ]; then
    echo "Error: No pods found in namespace $NAMESPACE"
    exit 1
fi

# Count the number of pods
POD_COUNT=$(echo "$PODS" | wc -l)

# Fail if there is more than one pod
if [ "$POD_COUNT" -ne 1 ]; then
    echo "Error: Expected exactly one pod, but found $POD_COUNT in namespace $NAMESPACE!"
    exit 1
fi

# Extract the pod name correctly
POD_NAME=$(echo "$PODS" | head -n 1)
echo "Found operator pod: $POD_NAME"

echo "=== Operator Logs for $POD_NAME ==="
kubectl logs "$POD_NAME" -n "$NAMESPACE"
echo "=== End of Operator Logs ==="
