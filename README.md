# Coralogix Operator
[![license](https://img.shields.io/github/license/coralogix/coralogix-operator.svg)](https://raw.githubusercontent.com/coralogix/coralogix-operator/master/LICENSE)
![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/coralogix/coralogix-operator.svg?include_prereleases&style=plastic)
![Go Report Card](https://goreportcard.com/badge/github.com/coralogix/coralogix-operator)
![e2e-tests](https://github.com/coralogix/coralogix-operator/actions/workflows/e2e-tests.yaml/badge.svg?style=plastic)

## Overview
The Coralogix Operator provides Kubernetes-native deployment and management for Coralogix, 
designed to simplify and automate the configuration of Coralogix APIs through Kubernetes [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

Please refer to the next note if you're using the latest version of the operator - [A note regarding webhooks and cert-manager](README.md#a-note-regarding-webhooks-and-cert-manager).

The operator integrates with Kubernetes by supporting a variety of custom resources and controllers to simplify Coralogix management, including:
- **Custom Resources for Coralogix:** Easily deploy and manage Coralogix features, using custom resources like
[Alerts](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/alerts), 
[RecordingRuleGroupSets](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/recordingrulegroupset),
[RuleGroups](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/rulegroups), [OutboundWebhooks](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/outboundwebhooks) and others.
For a complete list of available APIs and their details, refer to the [API documentation](https://github.com/coralogix/coralogix-operator/tree/master/docs/api.md).
For examples of custom resources, see the [samples directory](https://github.com/coralogix/coralogix-operator/tree/main/config/samples).
- **Prometheus Operator Integration:** The Operator leverages [Prometheus Operator](https://prometheus-operator.dev/)'s PrometheusRule CRD,
to simplify the transition to Coralogix by utilizing existing monitoring configurations.
For more details on this integration, see the [Prometheus Integration documentation](https://github.com/coralogix/coralogix-operator/tree/master/docs/prometheus-integration.md).
- **Running Multiple Instances:** The operator supports running multiple instances within a single cluster by using label selectors.
For more details, see the [Running Multiple Instances documentation](https://github.com/coralogix/coralogix-operator/tree/master/docs/multi-instance-operator.md).


## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.

### Helm Installation
1. For using the official helm chart, add the Coralogix repository and update it:
```sh
helm repo add coralogix https://cgx.jfrog.io/artifactory/coralogix-charts-virtual
helm repo update
```

2. Install the operator with Helm:
```sh
helm install <my-release> coralogix/coralogix-operator \
  --set secret.data.apiKey="<api-key>" \
  --set coralogixOperator.region="<region>"
```

 - The Prometheus-Operator integration assumes PrometheusRule CRD is installed. If you wish to disable this integration, add the `--set prometheusOperator.prometheusRules.enabled=false` flag.

## **A note regarding webhooks and cert-manager**
Webhooks are used to validate the custom resources before they are created in the cluster. They are also used to convert the old schema to the new schema.
For the webhook to work, cert-manager should be installed in the cluster.
Webhooks will be enabled by default in the operator installation, so make sure cert-manager is installed in the cluster.
A [certificate](./charts/coralogix-operator/templates/certificate.yaml) and an [issuer](./charts/coralogix-operator/templates/issuer.yaml) will be installed on the cluster as part of the cert-manager installation.

### consequences of disabling webhooks
If you disable the webhooks, the operator will not be able to validate the custom resources before they are created in the cluster.
If you are using an old schema of the custom resources, the operator will not be able to convert them to the new schema.
That means you will have to manually update the custom resources to the new schema.
v1alpha1/Alerts won't be supported if webhooks are disabled, as the storage version is v1beta1.
The PrometheusRule controller won't be able to track alerts that were created in a v1alpha1 schema.

3. To uninstall the operator, run:
```sh
helm delete <my-release>
```
 
### Local Installation with Kustomize
1. Clone the operator repository and navigate to the project directory:
```
git clone https://github.com/coralogix/coralogix-operator.git 
cd coralogix-operator
```

2. Set the Coralogix API key and region as environment variables:
```sh
export CORALOGIX_API_KEY="<api-key>"
export CORALOGIX_REGION="<region>"
```
For private domain set the `CORALOGIX_DOMAIN` environment variable.

3. For a custom operator image, build and push your image:
```sh
make docker-build docker-push IMG=<some-registry>/coralogix-operator:<tag> 
```

4. Deploy the operator to the cluster with the image specified by `IMG`:
```sh
make deploy IMG=<some-registry>/coralogix-operator:<tag> 
```
Note: This will install cert-manager and PrometheusRule CRD on the cluster if not already installed.

5. To uninstall the operator, run:
```sh
make undeploy
```

## Contributing
Please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).
It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources until the Desired state is reached on the cluster.