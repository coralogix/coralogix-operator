# Coralogix Operator Metrics

| Name                           | Type    | Description                                                             | Labels                                                                          | 
|--------------------------------------|---------|-------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| cx_operator_info                     | Gauge   | Coralogix Operator information.                                         | go_version, operator_version, coralogix_url, label_selector, namespace_selector |
| cx_operator_resource_info            | Gauge   | Coralogix Operator custom resource information.                         | kind, name, namespace, status                                                   |

## Sending Metrics to Coralogix
The Coralogix Operator's metrics can be sent to Coralogix using Prometheus. 
This requires configuring Prometheus to both scrape the operator's metrics and send them to Coralogix.
### Scraping the Metrics
By default, the Coralogix Operator is deployed with:
- A [ServiceMonitor](../charts/coralogix-operator/templates/service_monitor.yaml) that instructs Prometheus to scrape the operator’s metrics endpoint.
- A [ClusterRole](../charts/coralogix-operator/templates/metrics_reader_role.yaml) that grants Prometheus access to the metrics.
    
To ensure proper scraping:
- Verify that Prometheus is configured to select the provided ServiceMonitor.
- Ensure that the ClusterRole is bound to Prometheus’s ServiceAccount so it has the necessary permissions.
### Sending the Metrics
To forward the collected metrics to Coralogix, follow [this guide](https://coralogix.com/docs/integrations/prometheus/prometheus-server/) to configure Prometheus accordingly.
