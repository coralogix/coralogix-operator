# Coralogix Operator - Production Deployment Guide

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [PrometheusRule Integration](#prometheusrule-integration)
6. [Template Conversion](#template-conversion)
7. [Priority Mapping](#priority-mapping)
8. [Monitoring and Troubleshooting](#monitoring-and-troubleshooting)
9. [Best Practices](#best-practices)
10. [Examples](#examples)

## Overview

The Coralogix Operator is a Kubernetes operator that automates the management of Coralogix resources through Custom Resource Definitions (CRDs). It provides seamless integration with Prometheus Operator, automatically converting PrometheusRule resources to Coralogix Alerts with intelligent template conversion from Go syntax to Tera syntax.

### Key Features

- **Automatic Template Conversion:** Converts Prometheus Go templates to Coralogix Tera templates
- **Priority Mapping:** Maps Prometheus severity labels to Coralogix priority levels
- **Duration Validation:** Automatically validates and clamps alert durations to valid ranges (1-2160 minutes)
- **Routing Labels:** Automatically adds routing.group labels for alert organization
- **PrometheusRule Integration:** Seamlessly integrates with existing PrometheusRule configurations

## Prerequisites

### Required

- Kubernetes cluster (v1.16+)
- [Prometheus Operator](https://prometheus-operator.dev/) installed (for PrometheusRule CRD support)
- Coralogix API key
- Coralogix account region

### Optional

- Helm 3.x (recommended for installation)
- kubectl configured to access your cluster

## Installation

### Step 1: Add Helm Repository

```bash
# Add the Coralogix Helm repository
helm repo add coralogix https://cgx.jfrog.io/artifactory/coralogix-charts-virtual
helm repo update
```

### Step 2: Install the Operator

```bash
helm install coralogix-operator coralogix/coralogix-operator \
  --namespace coralogix-operator \
  --create-namespace \
  --set secret.data.apiKey="<your-api-key>" \
  --set coralogixOperator.region="<your-region>" \
  --set coralogixOperator.image.repository="public.ecr.aws/w3s4j9x9/cx-operator-go-convertor" \
  --set coralogixOperator.image.tag="v1.0.0"
```

**Available Regions:**
- `us1` - US East
- `us2` - US West
- `eu1` - Europe
- `eu2` - Europe 2
- `ap1` - Asia Pacific
- `ap2` - Asia Pacific 2

### Step 3: Verify Installation

```bash
# Check operator pod status
kubectl get pods -n coralogix-operator

# Check operator logs
kubectl logs -n coralogix-operator -l app.kubernetes.io/name=coralogix-operator --tail=50

# Verify CRDs are installed
kubectl get crd | grep coralogix
```

### Step 4: Upgrade Existing Installation

If upgrading from a previous version:

```bash
helm upgrade coralogix-operator coralogix/coralogix-operator \
  --namespace coralogix-operator \
  --set secret.data.apiKey="<your-api-key>" \
  --set coralogixOperator.region="<your-region>" \
  --set coralogixOperator.image.repository="public.ecr.aws/w3s4j9x9/cx-operator-go-convertor" \
  --set coralogixOperator.image.tag="v1.0.0"
```

## Configuration

### Image Configuration

The operator uses a custom image with template conversion capabilities:

```yaml
coralogixOperator:
  image:
    repository: public.ecr.aws/w3s4j9x9/cx-operator-go-convertor
    tag: "v1.0.0"  # or "latest"
    pullPolicy: IfNotPresent
```

### Advanced Configuration

For complete configuration options, see the [Helm Chart README](./charts/coralogix-operator/README.md).

## PrometheusRule Integration

### Enabling PrometheusRule Processing

To enable the operator to process PrometheusRule resources, add the following label to your PrometheusRule:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: my-alerts
  namespace: my-namespace
  labels:
    app.coralogix.com/track-alerting-rules: "true"  # Required label
spec:
  groups:
    - name: my-alert-group
      rules:
        - alert: MyAlert
          expr: up == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            description: "Service is down"
```

### How It Works

1. The operator watches for PrometheusRule resources with the `app.coralogix.com/track-alerting-rules: "true"` label
2. For each alert in the PrometheusRule, the operator creates a Coralogix Alert CRD
3. Go templates in descriptions and labels are automatically converted to Tera templates
4. Severity labels are mapped to Coralogix priority levels
5. Alert durations are validated and clamped to valid ranges
6. Routing labels are automatically added

## Template Conversion

The operator automatically converts Prometheus Go template syntax to Coralogix Tera template syntax. See the [Template Conversion Guide](./PROMETHEUS_TEMPLATE_CONVERSION.md) for detailed information.

### Quick Reference

| Go Template | Tera Template |
|------------|---------------|
| `{{ $labels.pod }}` | `{{ alert.groups[0].keyValues.pod }}` |
| `{{ $value }}` | `{{ alert.value }}` |
| `{{ printf "%.2f" $value }}` | `{{ alert.value \| round(method="ceil", precision=2) }}` |

### Example

**Before (PrometheusRule):**
```yaml
annotations:
  description: 'Pod {{ $labels.pod }} has high CPU: {{ $value }}'
```

**After (Coralogix Alert):**
```yaml
spec:
  description: 'Pod {{ alert.groups[0].keyValues.pod }} has high CPU: {{ alert.value }}'
```

## Priority Mapping

The operator maps Prometheus `severity` labels to Coralogix `priority` levels:

| Prometheus Severity | Coralogix Priority | Notes |
|---------------------|-------------------|-------|
| `critical` | `p1` | Highest priority |
| `high`, `error` | `p2` | High priority |
| `moderate`, `warning` | `p3` | Medium priority (default) |
| `info` | `p4` | Low priority |
| `low` | `p5` | Lowest priority |
| No severity / Unknown / Dynamic | `p3` | Default fallback |

### Dynamic Severity

If a severity label contains Go template syntax (e.g., `{{ $labels.severity }}`), the operator will:
1. Default to `p3` priority
2. Not attempt to convert the template in the severity field

### Example

```yaml
labels:
  severity: moderate  # Maps to p3
```

## Duration Validation

The operator automatically validates and clamps alert durations:

- **Minimum:** 1 minute
- **Maximum:** 2160 minutes (36 hours)
- **Behavior:** Durations outside this range are automatically clamped to the nearest valid value

### Example

```yaml
# PrometheusRule
for: 2d  # 2880 minutes - exceeds maximum

# Automatically clamped to
for: 2160m  # Maximum allowed duration
```

## Automatic Labels

The operator automatically adds the following label to all alerts:

- `routing.group: main` - Used for alert routing and organization

## Monitoring and Troubleshooting

### Check Operator Status

```bash
# Check pod status
kubectl get pods -n coralogix-operator

# Check operator logs
kubectl logs -n coralogix-operator -l app.kubernetes.io/name=coralogix-operator --tail=100

# Check for errors
kubectl logs -n coralogix-operator -l app.kubernetes.io/name=coralogix-operator | grep -i error
```

### Check Alert Status

```bash
# List all alerts
kubectl get alert --all-namespaces

# Check specific alert
kubectl get alert <alert-name> -n <namespace> -o yaml

# Check alert sync status
kubectl get alert <alert-name> -n <namespace> -o jsonpath='{.status.conditions[?(@.type=="RemoteSynced")].status}'
```

### Common Issues

#### Alerts Not Created

1. **Check PrometheusRule label:**
   ```bash
   kubectl get prometheusrule <name> -n <namespace> -o yaml | grep track-alerting-rules
   ```
   Should show: `app.coralogix.com/track-alerting-rules: "true"`

2. **Check operator logs:**
   ```bash
   kubectl logs -n coralogix-operator -l app.kubernetes.io/name=coralogix-operator | grep -i prometheusrule
   ```

3. **Verify PrometheusRule exists:**
   ```bash
   kubectl get prometheusrule <name> -n <namespace>
   ```

#### Alerts Not Syncing

1. **Check alert status:**
   ```bash
   kubectl describe alert <alert-name> -n <namespace>
   ```

2. **Check for validation errors:**
   ```bash
   kubectl get alert <alert-name> -n <namespace> -o yaml | grep -A 10 "status:"
   ```

3. **Verify API key and region:**
   ```bash
   kubectl get secret -n coralogix-operator -o yaml | grep apiKey
   ```

#### Template Conversion Issues

1. **Verify conversion:**
   ```bash
   kubectl get alert <alert-name> -n <namespace> -o yaml | grep description
   ```
   Should show Tera syntax, not Go syntax.

2. **Check for unsupported patterns:**
   ```bash
   kubectl get alert <alert-name> -n <namespace> -o yaml | grep "conversion not supported"
   ```

## Best Practices

### 1. Use Standard Prometheus Template Syntax

Stick to standard patterns for best compatibility:
- `{{ $labels.<name> }}` for label references
- `{{ $value }}` for alert values
- Avoid complex template expressions

### 2. Organize PrometheusRules

- Group related alerts in the same PrometheusRule
- Use meaningful namespaces
- Add appropriate labels for filtering

### 3. Monitor Alert Status

Regularly check alert sync status:
```bash
kubectl get alert --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace,STATUS:.status.printableStatus
```

### 4. Validate Before Production

- Test template conversion with sample PrometheusRules
- Verify priority mappings
- Check duration validation
- Review converted alerts before deploying to production

### 5. Use Version Tags

Always specify version tags for the operator image:
```yaml
coralogixOperator:
  image:
    tag: "v1.0.0"  # Use specific version, not "latest"
```

## Examples

### Complete Example: PrometheusRule with Template Conversion

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: application-alerts
  namespace: production
  labels:
    app.coralogix.com/track-alerting-rules: "true"
    team: platform
spec:
  groups:
    - name: application
      rules:
        - alert: HighErrorRate
          expr: |
            sum(rate(http_requests_total{status=~"5.."}[5m])) by (service, environment)
            /
            sum(rate(http_requests_total[5m])) by (service, environment)
            > 0.05
          for: 10m
          labels:
            severity: high
            team: backend
          annotations:
            description: |
              Service {{ $labels.service }} in environment {{ $labels.environment }}
              has error rate of {{ $value | humanizePercentage }}
        
        - alert: HighLatency
          expr: |
            histogram_quantile(0.95,
              sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service)
            ) > 1.0
          for: 15m
          labels:
            severity: moderate
            team: platform
          annotations:
            description: |
              Service {{ $labels.service }} has p95 latency of {{ $value }}s
```

### Example: Checking Converted Alerts

```bash
# List alerts created from PrometheusRule
kubectl get alert -n production | grep application-alerts

# Check converted description
kubectl get alert application-alerts-higherrorrate-0 -n production -o jsonpath='{.spec.description}'

# Verify priority mapping
kubectl get alert application-alerts-higherrorrate-0 -n production -o jsonpath='{.spec.priority}'

# Check entity labels
kubectl get alert application-alerts-higherrorrate-0 -n production -o jsonpath='{.spec.entityLabels}'
```

### Example: Validation Script

```bash
#!/bin/bash
# validate-alerts.sh

NAMESPACE=${1:-production}

echo "Validating alerts in namespace: $NAMESPACE"
echo ""

# Check all alerts are synced
UNSYNCED=$(kubectl get alert -n $NAMESPACE -o json | \
  jq -r '.items[] | select(.status.printableStatus != "RemoteSynced") | .metadata.name')

if [ -z "$UNSYNCED" ]; then
  echo "✓ All alerts are synced"
else
  echo "✗ Unsynced alerts:"
  echo "$UNSYNCED"
fi

# Check for Go template syntax (should not exist)
echo ""
echo "Checking for unconverted Go templates..."
kubectl get alert -n $NAMESPACE -o yaml | grep -E '\{\{.*\$labels|\$value' && \
  echo "✗ Found unconverted Go templates!" || \
  echo "✓ No Go templates found (conversion successful)"

# Count alerts
TOTAL=$(kubectl get alert -n $NAMESPACE --no-headers | wc -l)
echo ""
echo "Total alerts: $TOTAL"
```

## Additional Resources

- [API Documentation](./api.md)
- [Metrics Documentation](./metrics.md)
- [Template Conversion Guide](./PROMETHEUS_TEMPLATE_CONVERSION.md)
- [Helm Chart README](./charts/coralogix-operator/README.md)

## Support

For issues, questions, or contributions:
- GitHub Issues: [coralogix-operator](https://github.com/coralogix/coralogix-operator/issues)
- Documentation: [Coralogix Docs](https://coralogix.com/docs/)

