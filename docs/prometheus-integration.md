# Prometheus Integration

The Coralogix Operator integrates with the [Prometheus Operator](https://prometheus-operator.dev/) PrometheusRule CRD, to simplify the transition to Coralogix.
By using existing monitoring configurations, the operator makes it easier to adopt Coralogix's advanced monitoring and alerting features.

The operator watches PrometheusRule resources and automatically creates Coralogix custom resources in the cluster including Alerts and RecordingRuleGroupSets.

## PrometheusRule Integration

### Alerts

PrometheusRule alerts can be used to configure Coralogix Metric Alerts. Since Coralogix Metric Alerts provide more advanced alerting capabilities than PrometheusRule alerts, this integration is ideal for quickly setting up basic alerts. To leverage the full capabilities of Coralogix Metric Alerts, you should manage the alerts directly through the Coralogix Alert custom resource.

To enable the operator to monitor alerts in a PrometheusRule, add the following annotation to the PrometheusRule:

```yaml
app.coralogix.com/track-alerting-rules: "true"
```

The operator will create a Coralogix Alert in the PrometheusRule namespace, for each alert in the PrometheusRule.

The following Coralogix Alert properties are derived from the PrometheusRule alerting rule:

- `Alert.Spec.Name`: Set to `rule.Alert` value
- `Alert.Spec.Description`: Set to `rule.Annotations["description"]` value
- `Alert.Spec.EntityLabels`: Set to `rule.Labels` property, plus the PrometheusRule's own labels that start with `routing.` (see [Notification routing](#notification-routing))
- `Alert.Spec.Priority`: Set to `rule.Labels["severity"]` value, with the next priority mapping:
    - `critical` -> `p1`
    - `error` -> `p2`
    - `warning` -> `p3`
    - `info` -> `p4`
    - `low` -> `p5`
- `Alert.Spec.AlertType.MetricThreshold.OfTheLast.DynamicDuration`: Set to `rule.For` value
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.ConditionType`: Set to `moreThan`
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.Threshold`: Set to `0`
- `Alert.Spec.AlertType.MetricThreshold.Rules[0].Condition.ForOverPct`: Set to `100`

Other properties will not be overridden by the operator and can be modified directly in the Coralogix Alert resource.

#### Notification routing

Alerts created from a PrometheusRule can be delivered through Notification Center [Global Routers](https://coralogix.com/docs/user-guides/notification-center/routing/introduction/), which match alerts by their routing labels.

To route the Alerts generated from a PrometheusRule, add this annotation to the PrometheusRule, with the router's `notifyOn` as its value (`triggeredOnly` or `triggeredAndResolved`):

```yaml
app.coralogix.com/alert-router-notify-on: triggeredAndResolved
```

The operator then sets `Alert.Spec.NotificationGroup.Router` to `{notifyOn: <value>, mode: routingLabels}` on each generated Alert, so Coralogix sends its notifications to every Global Router whose routing labels match the Alert's. The rest of the notification group, such as `groupByKeys` and `webhooks`, is kept. `destinations` is removed, because an Alert can have a router or destinations but not both. A phantom Alert (`phantomMode: true`) gets no router, because it can't have a notification group. An invalid annotation value fails the reconcile and leaves routers as they are; the PrometheusRule's other changes still sync. Without the annotation, the operator leaves the notification group as it is.

Routing labels are entity labels with the `routing.` prefix, such as `routing.environment`. Prometheus does not allow dots in rule label names, so set them as labels on the PrometheusRule itself. The operator copies every PrometheusRule label that starts with `routing.` into the entity labels of each generated Alert:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    app.coralogix.com/track-alerting-rules: "true"
    routing.environment: production
    routing.team: payments
  annotations:
    app.coralogix.com/alert-router-notify-on: triggeredAndResolved
  name: payments-rules
spec:
  groups:
    - name: payments.rules
      rules:
        - alert: payments-error-rate
          expr: rate(payment_errors_total[5m]) > 1
          for: 5m
          labels:
            severity: critical
```

The generated Alert gets these entity labels and router:

```yaml
spec:
  entityLabels:
    routing.environment: production
    routing.team: payments
    severity: critical
  notificationGroup:
    router:
      mode: routingLabels
      notifyOn: triggeredAndResolved
```

#### Example

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
              dynamicDuration: 5m
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

### Recording Rules

PrometheusRule recording rules can be used to configure the Coralogix RecordingRuleGroupSet.

To enable the operator to monitor recording rules in a PrometheusRule, add the following annotation to the PrometheusRule:

```yaml
app.coralogix.com/track-recording-rules: "true"
```

The operator will create a Coralogix RecordingRuleGroupSet in the PrometheusRule namespace, containing all the PrometheusRule's recording rules.

#### Example

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
