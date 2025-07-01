# Coralogix Operator Metrics

The Coralogix Operator exposes a set of metrics that provide visibility into its internal state and performance. 
These include both [standard operator metrics](https://book.kubebuilder.io/reference/metrics-reference) provided by controller-runtime, 
and custom metrics implemented by the Coralogix Operator.

## Custom Metrics
| Name | Type | Description | Labels |
|------|------|-------------|---------|
| `cx_operator_build_info` | Gauge | Coralogix Operator build information. | `go_version`, `operator_version`, `coralogix_url` |
| `cx_operator_resource_info` | Gauge | Coralogix Operator custom resource information. | `kind`, `name`, `namespace`, `status` |

## Accessing the Metrics
Metrics are exposed via the operator’s `/metrics` endpoint, which by default is served on port 8080.

To access them locally:
```bash
kubectl port-forward -n <operator-namespace> svc/coralogix-operator 8080:8080
```
Then, open your browser to `http://localhost:8080/metrics`.

## Sending Metrics to Coralogix
The operator's metrics can be sent to Coralogix using Prometheus Operator.
This requires configuring Prometheus to both scrape the operator's metrics and forward them to Coralogix.

### Scraping the Metrics with Prometheus
By default, the Coralogix Operator is deployed with:

- A [ServiceMonitor](../charts/coralogix-operator/templates/service_monitor.yaml) that instructs Prometheus to scrape the operator's metrics endpoint.
- A [ClusterRole](../charts/coralogix-operator/templates/metrics_reader_role.yaml) that grants access to read the metrics.
    
To ensure proper scraping:

- Verify that Prometheus is configured to select the provided `ServiceMonitor`.
- Ensure that the `ClusterRole` is bound to Prometheus's `ServiceAccount` so it has the necessary permissions (using a `ClusterRoleBinding`).

### Forwarding the Metrics to Coralogix
To forward the collected metrics to Coralogix, follow [this guide](https://coralogix.com/docs/integrations/prometheus/prometheus-server/) to configure Prometheus accordingly.
