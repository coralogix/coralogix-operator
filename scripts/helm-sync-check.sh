#!/bin/bash

set -e

# Paths
crds_path="config/crd/bases"
chart_crds_path="charts/coralogix-operator/templates/crds"
role_file="config/rbac/role.yaml"
chart_role_file="charts/coralogix-operator/templates/cluster_role.yaml"
webhook_file="config/webhook/manifests.yaml"
chart_webhook_file="charts/coralogix-operator/templates/webhook.yaml"

errors_found=0

echo "Validating CRDs..."

# Validate CRDs
crds_files=$(find "$crds_path" -type f -name "*.yaml")
for crd_file in $crds_files; do
    chart_crd_file="$chart_crds_path/$(basename $crd_file)"
    
    if [ -f "$chart_crd_file" ]; then
        if ! cmp -s "$crd_file" "$chart_crd_file"; then
            echo "CRD file $chart_crd_file is outdated, please run make helm-update-crds"
            errors_found=$((errors_found + 1))
        fi
    else
        echo "CRD file $chart_crd_file is missing in the Helm chart. Please run make helm-update-crds"
        errors_found=$((errors_found + 1))
    fi
done

echo "Validating role..."

# Enforce role changes if the role file has been modified
if git diff --name-only origin/main...HEAD | grep -q "^$role_file$"; then
    if ! git diff --name-only origin/main...HEAD | grep -q "^$chart_role_file$"; then
        echo "role file $role_file was modified, but the corresponding Helm chart file $chart_role_file was not updated. Please update the chart."
        errors_found=$((errors_found + 1))
    fi
fi

echo "Validating Webhook..."

# Enforce webhook changes if the webhook file has been modified
if git diff --name-only origin/main...HEAD | grep -q "^$webhook_file$"; then
    if ! git diff --name-only origin/main...HEAD | grep -q "^$chart_webhook_file$"; then
        echo "Webhook file $webhook_file was modified, but the corresponding Helm chart file $chart_webhook_file was not updated. Please update the chart."
        errors_found=$((errors_found + 1))
    fi
fi

# Final result
if [ "$errors_found" -gt 0 ]; then
    echo "Validation failed with $errors_found error(s)."
    exit 1
fi

echo "All validations passed. CRDs, role, and Webhook are up-to-date."
exit 0