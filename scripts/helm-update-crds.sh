#!/bin/bash
# This shell script will update the Helm CRD files

set -e

# Paths
crds_path="charts/coralogix-operator/templates/crds"
bases_path="config/crd/bases"

echo "Updating Helm CRDs from $bases_path to $crds_path..."

# Ensure the target CRDs directory exists
mkdir -p "$crds_path"

# Copy and wrap each CRD from bases into Helm chart directory
for base_file in "$bases_path"/*.yaml; do
    crd_file="$crds_path/$(basename "$base_file")"

    {
        echo "{{- if .Values.crds.create }}"
        cat "$base_file"
        echo "{{- end }}"
    } > "$crd_file"

    echo "Wrapped and wrote CRD file: $crd_file"
done

# Cleanup: Remove files in the chart directory that no longer exist in config/crd/bases
for crd_file in "$crds_path"/*.yaml; do
    base_file="$bases_path/$(basename "$crd_file")"
    if [ ! -f "$base_file" ]; then
        rm "$crd_file"
        echo "Removed obsolete CRD file: $crd_file"
    fi
done

echo "Helm CRDs update completed successfully."
