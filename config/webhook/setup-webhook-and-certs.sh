#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Variables
CERT_DIR=$(mktemp -d)  # Create a temporary directory for certs
SERVICE=${2:-"webhook-service"}
SECRET_NAME=${3:-"webhook-server-cert"}
WEBHOOK_CONFIG_PATH=${4:-"${SCRIPT_DIR}/manifests.yaml"}  # Adjusted default path to generated manifests
SERVICE_WEBHOOK_CONFIG_PATH=${5:-"${SCRIPT_DIR}/service.yaml"}  # Adjusted default path to generated service manifests

# Allow namespace to be overridden manually
if [[ -n "$1" ]]; then
    NAMESPACE=$1
    echo "Using manually provided namespace: $NAMESPACE"
else
    # Automatically detect the namespace of the manager deployment
    NAMESPACE=$(kubectl get deployment -A -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.namespace}')
    if [[ -z "$NAMESPACE" ]]; then
        echo "Error: Could not detect the manager deployment namespace."
        exit 1
    fi
    echo "Detected manager namespace: $NAMESPACE"
fi

echo "Using temporary directory for certificates: ${CERT_DIR}"

# Ensure the namespace exists before creating the secret
kubectl get namespace $NAMESPACE || kubectl create namespace $NAMESPACE

# Generate CA certificate and key
openssl req -newkey rsa:2048 -nodes -keyout ${CERT_DIR}/ca.key -x509 -days 365 -out ${CERT_DIR}/ca.crt -subj "/CN=${SERVICE}.${NAMESPACE}.svc"

# Generate server certificate key
openssl genrsa -out ${CERT_DIR}/tls.key 2048

# Create a SAN config file to include SANs for the certificate
cat <<EOF > ${CERT_DIR}/san.conf
[ req ]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[ dn ]
CN = ${SERVICE}.${NAMESPACE}.svc

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE}.${NAMESPACE}.svc
DNS.2 = ${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

# Create a certificate signing request (CSR) with SANs
openssl req -new -key ${CERT_DIR}/tls.key -out ${CERT_DIR}/server.csr -config ${CERT_DIR}/san.conf

# Generate the server certificate signed with the CA certificate and SANs
openssl x509 -req -in ${CERT_DIR}/server.csr -CA ${CERT_DIR}/ca.crt -CAkey ${CERT_DIR}/ca.key -CAcreateserial -out ${CERT_DIR}/tls.crt -days 365 -extensions req_ext -extfile ${CERT_DIR}/san.conf

# Cleanup: Remove the CSR and config file as they're no longer needed
rm ${CERT_DIR}/server.csr ${CERT_DIR}/san.conf

# Check if all files are generated
if [[ -f "${CERT_DIR}/ca.crt" && -f "${CERT_DIR}/tls.crt" && -f "${CERT_DIR}/tls.key" ]]; then
    echo "Self-signed certificates have been generated successfully in the temporary directory: ${CERT_DIR}"
    echo "Files:"
    echo " - CA Certificate: ${CERT_DIR}/ca.crt"
    echo " - Server Certificate: ${CERT_DIR}/tls.crt"
    echo " - Server Key: ${CERT_DIR}/tls.key"

    # Delete the existing secret if it exists (to avoid immutability errors)
    kubectl delete secret ${SECRET_NAME} --namespace=${NAMESPACE} --ignore-not-found

    # Create the Kubernetes secret with the generated certificates
    kubectl create secret tls ${SECRET_NAME} \
    --cert=${CERT_DIR}/tls.crt \
    --key=${CERT_DIR}/tls.key \
    --namespace=${NAMESPACE}

    echo "Kubernetes secret '${SECRET_NAME}' has been created in the '${NAMESPACE}' namespace."

    # Extract the base64-encoded CA certificate
    CA_BUNDLE=$(cat ${CERT_DIR}/ca.crt | base64 | tr -d '\n')

    # Function to use appropriate sed command based on OS
    function sed_inplace() {
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "$1" "$2"
        else
            sed -i "$1" "$2"
        fi
    }

    # Update the caBundle in the webhook config file
    sed_inplace "s|caBundle: .*|caBundle: ${CA_BUNDLE}|" "${WEBHOOK_CONFIG_PATH}"

    # Replace the namespace in the service YAML file with the correct namespace
    sed_inplace "s|namespace:.*|namespace: ${NAMESPACE}|" "${SERVICE_WEBHOOK_CONFIG_PATH}"

    # Replace the service namespace in the ValidatingWebhookConfiguration YAML file
    sed_inplace "s|namespace:.*|namespace: ${NAMESPACE}|" "${WEBHOOK_CONFIG_PATH}"

    # Check if caBundle exists in the YAML file
    if grep -q "caBundle:" "${WEBHOOK_CONFIG_PATH}"; then
        # Replace the existing caBundle value with the new CA_BUNDLE
        sed_inplace "s|caBundle: .*|caBundle: ${CA_BUNDLE}|" "${WEBHOOK_CONFIG_PATH}"
        echo "Updated ${WEBHOOK_CONFIG_PATH} with the new CA_BUNDLE."
    else
        # Add the caBundle field under clientConfig with proper indentation
        awk '/clientConfig:/ {print; print "    caBundle: '"${CA_BUNDLE}"'"; next}1' "${WEBHOOK_CONFIG_PATH}" > "${WEBHOOK_CONFIG_PATH}.tmp" && mv "${WEBHOOK_CONFIG_PATH}.tmp" "${WEBHOOK_CONFIG_PATH}"
        echo "Added caBundle to ${WEBHOOK_CONFIG_PATH}."
    fi

    # Apply the ValidatingWebhookConfiguration
    kubectl apply -f ${WEBHOOK_CONFIG_PATH}

    echo "ValidatingWebhookConfiguration has been updated and applied."
else
    echo "Error: One or more certificate files are missing."
    exit 1
fi

# Apply the service in the correct namespace
kubectl apply -f ${SERVICE_WEBHOOK_CONFIG_PATH} --namespace=${NAMESPACE}

# Optional: Clean up the temporary directory after the script completes
rm -rf ${CERT_DIR}
echo "Temporary certificate directory ${CERT_DIR} has been removed."
