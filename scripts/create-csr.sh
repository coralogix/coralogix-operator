#!/bin/bash

[ -z ${service} ] && service=validation-webhook-svc
[ -z ${secret} ] && secret=validation-webhook-certs
[ -z ${namespace} ] && namespace=default

csrName=${service}.${namespace}
tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

cat <<EOF >> ${tmpdir}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
EOF

#openssl genrsa -out ${tmpdir}/server-key.pem 2048
#openssl req -new -key ${tmpdir}/server-key.pem -subj "/CN=system:node:${service}.${namespace}.svc /OU=system:nodes /O=system:nodes" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf
#
openssl genrsa -out ./certs/server-key.pem 2048
openssl req -new -key ./certs/server-key.pem -subj "/CN=system:node:${service}.${namespace}.svc /OU=system:nodes /O=system:nodes" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf

# clean-up any previously created CSR for our service. Ignore errors if not present.
kubectl delete csr ${csrName} 2>/dev/null || true

# create  server cert/key CSR and  send to k8s API
cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: ${csrName}
spec:
  groups:
  - system:authenticated
  request: $(cat ${tmpdir}/server.csr | base64 | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  expirationSeconds: 864000
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

# verify CSR has been created
while true; do
    kubectl get csr ${csrName}
    if [ "$?" -eq 0 ]; then
        break
    fi
done

# approve and fetch the signed certificate
kubectl certificate approve ${csrName}
# verify certificate has been signed
for x in $(seq 10); do
    serverCert=$(kubectl get csr ${csrName} -o jsonpath='{.status.certificate}')
    if [[ ${serverCert} != '' ]]; then
        break
    fi
    sleep 1
done
if [[ ${serverCert} == '' ]]; then
    echo "ERROR: After approving csr ${csrName}, the signed certificate did not appear on the resource. Giving up after 10 attempts." >&2
    exit 1
fi
echo ${serverCert} | openssl base64 -d -A -out ./certs/server-cert.pem

# clean-up pods service and webhook configuration
kubectl delete Pod validation-webhook 2>/dev/null || true
kubectl delete Service validation-webhook-svc 2>/dev/null || true
kubectl delete ValidatingWebhookConfiguration validating-webhook-configuration 2>/dev/null || true

Docker build -t webhook:0.1.0 ../.
kind load docker-image webhook:0.1.0

# create Pod and Service for the webhook
kubectl create -f  ./config/webhook/service.yaml

set -o errexit
set -o nounset
set -o pipefail


CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

# create the validating webhook configuration
cat <<EOF | kubectl create -f -
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  creationTimestamp: null
  name: validating-webhook-configuration
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    service:
      name: validation-webhook-svc
      namespace: system
      path: /validate-coralogix-coralogix-com-v1alpha1-alert
  failurePolicy: Fail
  name: valert.kb.io
  rules:
  - apiGroups:
    - coralogix.coralogix.com
    apiVersions:
    - v1alpha1
    operations:
    - CREATE
    - UPDATE
    resources:
    - alerts
  sideEffects: None
EOF