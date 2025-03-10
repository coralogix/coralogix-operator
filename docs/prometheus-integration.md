# Prometheus Integration
The Coralogix Operator integrates with [Prometheus Operator](https://prometheus-operator.dev/) PrometheusRule CRD, to simplify the transition to Coralogix. 
By using existing monitoring configurations, the operator makes it easier to adopt Coralogix’s advanced monitoring and alerting features.

The operator watches PrometheusRule resources and automatically creates Coralogix custom resources in the cluster, 
including Alerts and RecordingRuleGroupSets.

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
- `Alert.Spec.EntityLabels`: Set to `rule.Labels` property.
- `Alert.Spec.Priority`: Set to `rule.Labels["severity"]` value, with the next priority mapping:
  - `critical` -> `p1`
  - `error`    -> `p2`
  - `warning`  -> `p3`
  - `info`     -> `p4`
  - `low`      -> `p5`
  
- `Alert.Spec.AlertType.MetricThreshold.OfTheLast.SpecificValue`: Set to `rule.For` value.
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.ConditionType`: Set to `moreThan`.
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.Threshold`: Set to `0`.
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.ForOverPct`: Set to `100`.

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
apiVersion: coralogix.com/v1beta1
kind: Alert
metadata:
  labels:
    app.coralogix.com/track-alerting-rules: "true"
    app.kubernetes.io/managed-by: prometheus-example-rules
  name: prometheus-example-rules-example-alert-0
  namespace: default
spec:
  alertType:
    metricThreshold:
      metricFilter:
        promql: vector(1) > 0
      missingValues:
        minNonNullValuesPct: 0
        replaceWithZero: false
      rules:
        - condition:
            conditionType: moreThan
            forOverPct: 100
            ofTheLast:
              specificValue: 5m
            threshold: "0"
  description: app latency alert
  enabled: true
  entityLabels:
    severity: critical
    slack_channel: '#observability'
  name: example-alert
  phantomMode: false
  priority: p1

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
