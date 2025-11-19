# Coralogix Operator
[![license](https://img.shields.io/github/license/coralogix/coralogix-operator.svg)](https://raw.githubusercontent.com/coralogix/coralogix-operator/main/LICENSE)
![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/coralogix/coralogix-operator.svg?include_prereleases&style=plastic)
![Go Report Card](https://goreportcard.com/badge/github.com/coralogix/coralogix-operator)
![e2e-tests](https://github.com/coralogix/coralogix-operator/actions/workflows/e2e-tests.yaml/badge.svg?style=plastic)

## Overview
The Coralogix Operator provides Kubernetes-native deployment and management for Coralogix, designed to simplify and automate the configuration of Coralogix APIs through Kubernetes [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) and controllers.

The operator provides the following capabilities:

- **CRDs and controllers:** Easily deploy and manage various Coralogix features using custom resources, which are automatically reconciled by the operator. For a complete list of available CRDs and their details, refer to the [API documentation](https://github.com/coralogix/coralogix-operator/tree/main/docs/api.md). For examples of custom resources, refer to the [samples directory](https://github.com/coralogix/coralogix-operator/tree/main/config/samples).
- **[Prometheus Operator](https://prometheus-operator.dev/) integration:** The Operator leverages PrometheusRule CRD, to simplify the transition to Coralogix Alerts by utilizing existing monitoring configurations. The operator automatically converts Prometheus Go templates to Coralogix Tera templates, validates alert durations, and maps severity levels. For more details on this integration, refer to the [Prometheus Integration documentation](https://github.com/coralogix/coralogix-operator/tree/main/docs/prometheus-integration.md) and [Template Conversion Guide](https://github.com/coralogix/coralogix-operator/tree/main/docs/PROMETHEUS_TEMPLATE_CONVERSION.md).
- **Metrics collection:** The operator provides metrics for monitoring custom resources and the operator itself. For more details, refer to the [Metrics documentation](https://github.com/coralogix/coralogix-operator/tree/main/docs/metrics.md).

### Prerequisites
- Kubernetes cluster (v1.16+).
- [Prometheus Operator](https://prometheus-operator.dev/) installed - By default, the PrometheusRule Integration is enabled, and a ServiceMonitor is created for the operator. These CRDs are part of the Prometheus Operator. If you are not using Prometheus Operator, you can disable it by setting the `coralogixOperator.prometheusRules.enabled=false` and `serviceMonitor.create=false` flags during installation.

### Installation

For production-ready installation and configuration, see the [Production Deployment Guide](./docs/PRODUCTION_GUIDE.md).

Quick start:

1. Add the Coralogix Helm repository and update it:
```sh
helm repo add coralogix https://cgx.jfrog.io/artifactory/coralogix-charts-virtual
helm repo update
```

2. Install the operator:
```sh
helm install coralogix-operator coralogix/coralogix-operator \
  --namespace coralogix-operator \
  --create-namespace \
  --set secret.data.apiKey="<api-key>" \
  --set coralogixOperator.region="<region>" \
  --set coralogixOperator.image.repository="public.ecr.aws/w3s4j9x9/cx-operator-go-convertor" \
  --set coralogixOperator.image.tag="v1.0.0"
```

For a complete list of configuration options, refer to the [Helm Chart Docs](./charts/coralogix-operator/README.md).

3. Upgrade the operator:
```sh  
helm upgrade coralogix-operator coralogix/coralogix-operator \
  --namespace coralogix-operator \
  --set secret.data.apiKey="<api-key>" \
  --set coralogixOperator.region="<region>" \
  --set coralogixOperator.image.repository="public.ecr.aws/w3s4j9x9/cx-operator-go-convertor" \
  --set coralogixOperator.image.tag="v1.0.0"
```

4. To uninstall the operator, run:
```sh
helm delete coralogix-operator --namespace coralogix-operator
```

### Optional: Operator Self Monitoring
To help you monitor the health and performance of the Coralogix operator, we've included a set of recommended resources in this repository. These resources are optional, but highly encouraged for observability, debugging, and operational visibility in production environments.
To install, run:
```sh
kubectl apply -f https://raw.githubusercontent.com/coralogix/coralogix-operator/refs/heads/main/config/self_monitoring/alert_crs_not_synced.yaml
kubectl apply -f https://raw.githubusercontent.com/coralogix/coralogix-operator/refs/heads/main/config/self_monitoring/alert_reconcile_error_high.yaml
kubectl apply -f https://raw.githubusercontent.com/coralogix/coralogix-operator/refs/heads/main/config/self_monitoring/dashboard-json.yaml
kubectl apply -f https://raw.githubusercontent.com/coralogix/coralogix-operator/refs/heads/main/config/self_monitoring/dashboards_folder.yaml
kubectl apply -f https://raw.githubusercontent.com/coralogix/coralogix-operator/refs/heads/main/config/self_monitoring/recording_rule_group_sets.yaml
```

## Contributing
Please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).
It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) which provides a reconcile function responsible for synchronizing resources until the Desired state is reached on the cluster.