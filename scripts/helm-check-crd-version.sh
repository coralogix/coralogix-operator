#!/bin/bash

crds_path="charts/coralogix-operator/templates/crds"
bases_path="config/crd/bases"

errors_found=0

# Get a list of YAML files in the two paths
crds_files=$(find "$crds_path" -type f -name "*.yaml")

# Compare the contents of corresponding files
for crd_file in $crds_files; do
    base_file="$bases_path/$(basename $crd_file)"
    
    if [ -f "$base_file" ]; then
        if ! cmp -s "$crd_file" "$base_file"; then
            echo "CRD file $crd_file is outdated, please run make helm-update-crds"
            errors_found=$((errors_found + 1))
        fi
    else
        echo "Base file not found for $crd_file"
        errors_found=$((errors_found + 1))
    fi
done

if [ "$errors_found" -gt 0 ]; then
    exit 1
fi

echo "CRDS are up to date"
