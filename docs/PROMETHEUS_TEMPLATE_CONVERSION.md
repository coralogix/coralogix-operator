# Prometheus Template Conversion Guide

## Overview

The Coralogix Operator automatically converts Prometheus Go template syntax to Coralogix Tera template syntax when processing PrometheusRule resources. This allows you to use your existing PrometheusRule definitions with Go templates, and the operator will automatically convert them to the Tera syntax required by Coralogix.

## Template Conversion

### Go Template to Tera Template Mapping

The operator automatically converts the following Go template patterns to Tera syntax:

| Go Template | Tera Template | Description |
|------------|---------------|-------------|
| `{{ $labels.<name> }}` | `{{ alert.groups[0].keyValues.<name> }}` | Label values |
| `{{ $value }}` | `{{ alert.value }}` | Alert value |
| `{{ printf "format" $value }}` | `{{ alert.value \| round(...) }}` | Formatted values |

### Conversion Examples

#### Example 1: Simple Label Reference

**PrometheusRule (Go Template):**
```yaml
- alert: High CPU Usage
  annotations:
    description: 'Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has high CPU usage'
  expr: sum by (pod, namespace) (rate(container_cpu_usage_seconds_total[5m])) > 0.8
  labels:
    severity: warning
```

**Converted Coralogix Alert (Tera Template):**
```yaml
spec:
  description: 'Pod {{ alert.groups[0].keyValues.pod }} in namespace {{ alert.groups[0].keyValues.namespace }} has high CPU usage'
  entityLabels:
    severity: warning
    routing.group: main
  priority: p3
```

#### Example 2: Value Reference

**PrometheusRule (Go Template):**
```yaml
- alert: Pod Restart Loop
  annotations:
    description: 'Pod {{ $labels.pod }} has restarted {{ $value }} times'
  expr: rate(kube_pod_container_status_restarts_total[1h]) > 5
  labels:
    severity: moderate
```

**Converted Coralogix Alert (Tera Template):**
```yaml
spec:
  description: 'Pod {{ alert.groups[0].keyValues.pod }} has restarted {{ alert.value }} times'
  entityLabels:
    severity: moderate
    routing.group: main
  priority: p3
```

#### Example 3: Complex Expression

**PrometheusRule (Go Template):**
```yaml
- alert: API Error Rate High
  annotations:
    description: 'API {{ $labels.api }} has high error rate in environment {{ $labels.environment }}'
  expr: sum by (api, environment) (rate(http_requests_total{status=~"5.."}[5m])) / sum by (api, environment) (rate(http_requests_total[5m])) > 0.05
  labels:
    severity: high
```

**Converted Coralogix Alert (Tera Template):**
```yaml
spec:
  description: 'API {{ alert.groups[0].keyValues.api }} has high error rate in environment {{ alert.groups[0].keyValues.environment }}'
  entityLabels:
    severity: high
    routing.group: main
  priority: p2
```

## Supported Template Patterns

The operator recognizes and converts the following Go template patterns:

1. **Standalone label references:**
   - `{{ $labels.<name> }}` → `{{ alert.groups[0].keyValues.<name> }}`
   - `{{ $labels.<name> }}` (no spaces) → `{{alert.groups[0].keyValues.<name>}}`

2. **Value references:**
   - `{{ $value }}` → `{{ alert.value }}`
   - `$value` in expressions → `alert.value`

3. **Printf patterns:**
   - `{{ printf "%.2f" $value }}` → `{{ alert.value \| round(method="ceil", precision=2) }}`
   - `{{ printf "%d" $value }}` → `{{ alert.value \| round(method="ceil", precision=0) }}`

4. **Complex expressions:**
   - Labels within template blocks are automatically converted
   - Nested expressions are preserved and converted appropriately

## Conversion Detection

The operator automatically detects Go template syntax in:
- Alert descriptions (`annotations.description`)
- Entity labels (values in `labels`)

If Go template syntax is detected, the operator will:
1. Convert all Go template patterns to Tera syntax
2. Preserve the original structure and formatting where possible
3. Handle edge cases and unsupported patterns gracefully

## Unsupported Patterns

If the operator encounters template patterns that cannot be converted, it will:
- For descriptions: Attempt conversion and preserve what can be converted
- For entity labels: Replace with `[field conversion not supported]` placeholder

## Best Practices

1. **Use standard Prometheus template syntax:** Stick to `{{ $labels.<name> }}` and `{{ $value }}` patterns for best compatibility.

2. **Test your templates:** After deployment, verify that the converted templates work as expected in Coralogix.

3. **Check alert status:** Monitor the `RemoteSynced` status of created alerts to ensure successful conversion and synchronization.

4. **Review converted alerts:** Use `kubectl get alert <name> -o yaml` to review the converted template syntax.

## Troubleshooting

### Template Not Converting

If templates are not being converted:

1. **Check PrometheusRule label:** Ensure the PrometheusRule has the label:
   ```yaml
   labels:
     app.coralogix.com/track-alerting-rules: "true"
   ```

2. **Verify operator is running:** Check operator logs for conversion errors:
   ```bash
   kubectl logs -n coralogix-operator -l app.kubernetes.io/name=coralogix-operator | grep -i template
   ```

3. **Check alert status:** Verify the alert was created and check its status:
   ```bash
   kubectl get alert <alert-name> -o yaml
   ```

### Conversion Errors

If you see conversion errors in operator logs:

1. **Review the original template:** Ensure it follows standard Prometheus template syntax
2. **Check for unsupported patterns:** Some complex template expressions may not be supported
3. **Simplify the template:** Break down complex templates into simpler expressions

### Validation

To validate template conversion:

```bash
# Get the converted alert
kubectl get alert <alert-name> -o yaml

# Check for Go template syntax (should not be present)
kubectl get alert <alert-name> -o yaml | grep -E '\{\{.*\$labels|\$value'

# Verify Tera syntax is present
kubectl get alert <alert-name> -o yaml | grep -E 'alert\.groups\[0\]\.keyValues|alert\.value'
```

