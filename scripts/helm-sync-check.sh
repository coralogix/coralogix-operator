#!/bin/bash

set -e

# Paths
crds_path="config/crd/bases"
chart_crds_path="charts/coralogix-operator/crds"
role_file="config/rbac/role.yaml"
chart_role_file="charts/coralogix-operator/templates/cluster_role.yaml"

errors_found=0

echo "Validating CRDs..."

# Validate CRDs
crds_files=$(find "$crds_path" -type f -name "*.yaml")
for crd_file in $crds_files; do
    # Extract filename without the path
    crd_filename=$(basename "$crd_file")

    chart_crd_file="$chart_crds_path/$crd_filename"

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

# Final result
if [ "$errors_found" -gt 0 ]; then
    echo "Validation failed with $errors_found error(s)."
    exit 1
fi

echo "All validations passed. CRDs and roles are up-to-date."
exit 0
