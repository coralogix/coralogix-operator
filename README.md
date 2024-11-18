# Coralogix Operator
[![license](https://img.shields.io/github/license/coralogix/coralogix-operator.svg)](https://raw.githubusercontent.com/coralogix/coralogix-operator/master/LICENSE)
![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/coralogix/coralogix-operator.svg?include_prereleases&style=plastic)
![Go Report Card](https://goreportcard.com/badge/github.com/coralogix/coralogix-operator)
![e2e-tests](https://github.com/coralogix/coralogix-operator/actions/workflows/e2e-tests.yaml/badge.svg?style=plastic)

## Overview
The Coralogix Operator provides Kubernetes-native deployment and management for Coralogix, 
designed to simplify and automate the configuration of Coralogix APIs through Kubernetes [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The operator integrates with Kubernetes by supporting a variety of custom resources and controllers to simplify Coralogix management, including:

- **Custom Resources for Coralogix:** Easily deploy and manage Coralogix features, using custom resources like
[Alerts](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/alerts), 
[RecordingRuleGroupSets](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/recordingrulegroupset),
[RuleGroups](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/rulegroups), [OutboundWebhooks](https://github.com/coralogix/coralogix-operator/tree/master/config/samples/outboundwebhooks) and others.
For a complete list of available APIs and their details, refer to the [API documentation](https://github.com/coralogix/coralogix-operator/tree/master/docs/api.md).
- **Prometheus Operator Integration:** The Operator leverages [Prometheus Operator](https://prometheus-operator.dev/) CRDs like PrometheusRule and AlertmanagerConfig,
to simplify the transition to Coralogix by utilizing existing monitoring configurations.
For more details on this integration, see the [Prometheus Integration documentation](https://github.com/coralogix/coralogix-operator/tree/master/docs/prometheus-integration.md).

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.

### Running on the cluster
1. Clone the operator repository and navigate to the project directory:
```
git clone https://github.com/coralogix/coralogix-operator.git 
cd coralogix-operator
```

2. Install the CRDs into the cluster:
```sh
make install
```

3. Set the Coralogix API key and region as environment variables:
```sh
$ export CORALOGIX_API_KEY="<api-key>"
$ export CORALOGIX_REGION="<region>"
```
For private domain set the `CORALOGIX_DOMAIN` environment variable.

4. For a custom operator image, build and push your image to the location specified by `IMG`:
```sh
make docker-build docker-push IMG=<some-registry>/coralogix-operator:tag
```

5. Deploy the operator to the cluster with the image specified by `IMG`:
```sh
make deploy IMG=coralogixoperator/coralogix-operator:latest
```

6. Verify that the operator's pod is running:
```sh
kubectl get pods -n coralogix-operator-system
```

### Undeploy controller
Undeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
Please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).
It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources until the Desired state is reached on the cluster.
