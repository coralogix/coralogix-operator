#!/bin/bash
# Fails when a CRD is larger than 256 KiB. A client-side apply (kubectl apply, make install, or
# Argo CD without server-side apply) stores the whole object in an annotation, and the API server
# rejects an annotation over 256 KiB with "metadata.annotations: Too long".

set -e

limit=262144
errors_found=0

for crd_file in config/crd/bases/*.yaml; do
    size=$(wc -c < "$crd_file" | tr -d ' ')
    if [ "$size" -gt "$limit" ]; then
        echo "CRD file $crd_file has $size bytes, more than the limit of $limit bytes."
        errors_found=$((errors_found + 1))
    fi
done

if [ "$errors_found" -gt 0 ]; then
    echo "Validation failed with $errors_found error(s). Shorten the descriptions of the kind, or split it."
    exit 1
fi

echo "All CRDs are within $limit bytes."
