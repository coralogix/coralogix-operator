#!/bin/bash

# Variables
CERT_DIR=${1:-"/tmp/k8s-webhook-server/serving-certs"}
CERT_NAME="webhook-server"
NAMESPACE=${2:-"default"}
SERVICE=${3:-"webhook-service"}
SECRET_NAME=${4:-"webhook-server-cert"}

# Create certificate directory
mkdir -p ${CERT_DIR}

# Generate CA certificate and key
openssl req -newkey rsa:2048 -nodes -keyout ${CERT_DIR}/ca.key -x509 -days 365 -out ${CERT_DIR}/ca.crt -subj "/CN=${SERVICE}.${NAMESPACE}.svc"

# Generate server certificate key
openssl genrsa -out ${CERT_DIR}/tls.key 2048

# Create a certificate signing request (CSR)
openssl req -new -key ${CERT_DIR}/tls.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ${CERT_DIR}/server.csr

# Generate the server certificate signed with the CA certificate
openssl x509 -req -in ${CERT_DIR}/server.csr -CA ${CERT_DIR}/ca.crt -CAkey ${CERT_DIR}/ca.key -CAcreateserial -out ${CERT_DIR}/tls.crt -days 365

# Cleanup: Remove the CSR as it's no longer needed
rm ${CERT_DIR}/server.csr

# Check if all files are generated
if [[ -f "${CERT_DIR}/ca.crt" && -f "${CERT_DIR}/tls.crt" && -f "${CERT_DIR}/tls.key" ]]; then
    echo "Self-signed certificates have been generated successfully in the directory: ${CERT_DIR}"
    echo "Files:"
    echo " - CA Certificate: ${CERT_DIR}/ca.crt"
    echo " - Server Certificate: ${CERT_DIR}/tls.crt"
    echo " - Server Key: ${CERT_DIR}/tls.key"
else
    echo "Error: One or more certificate files are missing."
    exit 1
fi


Docker build -t webhook:0.1.0 ../.
kind load docker-image webhook:latest