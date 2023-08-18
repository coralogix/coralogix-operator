#!/bin/bash
# This shell script will update the helm CRDs files

crds_path="charts/coralogix-operator/templates/crds"
bases_path="config/crd/bases"

# Get a list of YAML files in the two paths
crds_files=$(find "$crds_path" -type f -name "*.yaml")

# Replace the contents of corresponding files
for crd_file in $crds_files; do
    base_file="$bases_path/$(basename $crd_file)"
    
    if [ -f "$base_file" ]; then
        cp "$base_file" "$crd_file"
        echo "Replaced CRD file: $crd_file"
    else
        echo "Base file not found for $crd_file"
    fi
done
