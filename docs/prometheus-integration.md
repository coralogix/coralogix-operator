# Prometheus Integration
The Coralogix Operator integrates with [Prometheus Operator](https://prometheus-operator.dev/) CRDs, 
such as PrometheusRule and AlertmanagerConfig, to simplify the transition to Coralogix. 
By using existing monitoring configurations, the operator makes it easier to adopt Coralogix’s advanced monitoring and alerting features.

The operator watches PrometheusRule and AlertmanagerConfig resources and automatically creates Coralogix custom resources in the cluster, 
including Alerts, RecordingRuleGroupSets and OutboundWebhooks.

## PrometheusRule Integration
### Alerts:  
PrometheusRule alerts can be used to configure Coralogix Metric Alerts. Since Coralogix Metric Alerts 
provide more advanced alerting capabilities than PrometheusRule alerts, this integration is ideal for 
quickly setting up basic alerts. To leverage the full capabilities of Coralogix Metric Alerts, 
you should manage the alerts directly through the Coralogix Alert custom resource.

To enable the operator to monitor alerts in a PrometheusRule, add the following annotation to the PrometheusRule:
```yaml
app.coralogix.com/track-alerting-rules: "true"
```
The operator will create a Coralogix Alert in the PrometheusRule namespace, for each alert in the PrometheusRule. 

The following Coralogix Alert properties are derived from the PrometheusRule alerting rule:
- `Alert.Spec.Name`: Set to `rule.Alert` value.
- `Alert.Spec.Description`: Set to `rule.Annotations["description"]` value.
- `Alert.Spec.Labels`: Set to `rule.Labels` property.
- `Alert.Spec.Severity`: Set to `rule.Labels["severity"]` value.
- `Alert.Spec.AlertType.Metric.Promql.Conditions.TimeWindow`: Set to `rule.For` value.
- `Alert.Spec.AlertType.Metric.Promql.Conditions.AlertWhen`: Set to `More`.
- `Alert.Spec.AlertType.Metric.Promql.Conditions.Threshold`: Set to `0`.
- `Alert.Spec.AlertType.Metric.Promql.Conditions.sampleThresholdPercentage`: Set to `100`.

Other properties will not be overridden by the operator and can be modified directly in the Coralogix Alert resource.

#### Example:
For the following PrometheusRule:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    app.coralogix.com/track-alerting-rules: "true"
  name: prometheus-example-rules
spec:
  groups:
    - name: example.rules
      interval: "60s"
      rules:
        - alert: example-alert
          expr: vector(1) > 0
          for: 5m
          annotations:
            description: "app latency alert"
          labels:
            severity: critical
            slack_channel: "#observability"
```
The following Coralogix Alert will be created:
```yaml
apiVersion: coralogix.com/v1alpha1
kind: Alert
metadata:
  labels:
    app.kubernetes.io/managed-by: prometheus-example-rules
  name: prometheus-example-rules-example-alert-0
  namespace: default
spec:
  active: true
  alertType:
    metric:
      promql:
        conditions:
          alertWhen: More
          minNonNullValuesPercentage: 0
          sampleThresholdPercentage: 100
          threshold: "0"
          timeWindow: FiveMinutes
        searchQuery: vector(1) > 0
  description: app latency alert
  labels:
    managed-by: coralogix-operator
    severity: critical
    slack_channel: '#observability'
  name: example-alert
  severity: Critical
```

### Recording Rules:
PrometheusRule recording rules can be used to configure Coralogix RecordingRuleGroupSet.

To enable the operator to monitor recording rules in a PrometheusRule, add the following annotation to the PrometheusRule:
```yaml
app.coralogix.com/track-recording-rules: "true"
```
The operator will create a Coralogix RecordingRuleGroupSet in the PrometheusRule namespace, 
containing all the PrometheusRule's recording rules.

#### Example:
For the following PrometheusRule:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    app.coralogix.com/track-recording-rules: "true"
  name: prometheus-example-rules
spec:
  groups:
    - name: first.group
      interval: "60s"
      rules:
        - record: example-record-1
          expr: vector(1)
        - record: example-record-2
          expr: vector(2)
    - name: second.group
      interval: "60s"
      rules:
        - record: example-record-3
          expr: vector(3)
        - record: example-record-4
          expr: vector(4)
```
The following Coralogix RecordingRuleGroupSet will be created:
```yaml
apiVersion: v1
items:
  - apiVersion: coralogix.com/v1alpha1
    kind: RecordingRuleGroupSet
    metadata:
      name: prometheus-example-rules
      namespace: default
    spec:
      groups:
        - intervalSeconds: 60
          name: first.group
          rules:
            - expr: vector(1)
              record: example-record-1
            - expr: vector(2)
              record: example-record-2
        - intervalSeconds: 60
          name: second.group
          rules:
            - expr: vector(3)
              record: example-record-3
            - expr: vector(4)
              record: example-record-4
```

## AlertmanagerConfig Integration
### Receivers: 
Receivers defined in an AlertmanagerConfig resource can be used to create Coralogix OutboundWebhooks. 

To enable the operator to monitor an AlertmanagerConfig, add the following annotation to the AlertmanagerConfig:
```yaml
app.coralogix.com/track-alertmanager-config: "true"
```
The operator will create a Coralogix OutboundWebhook in the AlertmanagerConfig namespace, for each receiver defined in the AlertmanagerConfig.

#### Example:
For the following receiver in an AlertmanagerConfig:
```yaml
  receivers:
    - name: opsgenie-general
      opsgenieConfigs:
        - sendResolved: true
          apiURL: https://api.opsgenie.com/
```
The following Coralogix OutboundWebhook will be created:
```yaml
apiVersion: coralogix.com/v1alpha1
kind: OutboundWebhook
metadata:
  finalizers:
    - outbound-webhook.coralogix.com/finalizer
  name: opsgenie-general.opsgenie.0
  namespace: default
spec:
  name: opsgenie-general.opsgenie.0
  outboundWebhookType:
    opsgenie:
      url: https://api.opsgenie.com/
```

## Combining PrometheusRule and AlertmanagerConfig configurations
The operator can link OutboundWebhooks created from an AlertmanagerConfig receivers to Alerts created from a PrometheusRule.
This linkage is based on the routes defined in the AlertmanagerConfig. According to the routes matchers and grouping,
the operator will populate the Alert's notification groups, which contain the relevant OutboundWebhooks.

To enable this functionality, add the following annotation to the PrometheusRule:
```yaml
app.coralogix.com/managed-by-alertmanager-config: "true"
```

#### Example:
Adding the following route to the AlertmanagerConfig:
```yaml
  route:
    groupBy:
      - alertname
      - namespace
      - severity
    receiver: opsgenie-general
    routes:
      - receiver: opsgenie-general
        matchers:
          - matchType: "=~"
            name: opsgenie_team
            value: ".+"
        groupBy:
          - coralogix.metadata.sdkId
```
Will add the following notification group to the Coralogix Alert:
```yaml
 notificationGroups:
  - groupByFields:
    - alertname
    - namespace
    - severity
    notifications:
    - integrationName: opsgenie-general.opsgenie.0
      notifyOn: TriggeredAndResolved
      retriggeringPeriodMinutes: 240
```
