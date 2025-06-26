# coralogix-operator

![Version: 0.4.7](https://img.shields.io/badge/Version-0.4.7-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.4.7](https://img.shields.io/badge/AppVersion-0.4.7-informational?style=flat-square)

Coralogix Operator Helm Chart

**Homepage:** <https://github.com/coralogix/coralogix-operator>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Coralogix | <platform@coralogix.com> |  |

## Source Code

* <https://github.com/coralogix/coralogix-operator>

## Requirements

Kubernetes: `>=1.16.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |
| certificate.create | bool | `true` | Specifies whether a Certificate should be created. |
| coralogixOperator | object | `{"domain":"","image":{"pullPolicy":"IfNotPresent","repository":"coralogixrepo/coralogix-operator","tag":""},"labelSelector":{},"leaderElection":{"enabled":true},"namespaceSelector":{},"prometheusRules":{"enabled":true},"reconcileIntervalSeconds":{"alert":"","alertScheduler":"","apiKey":"","customRole":"","dashboard":"","dashboardsFolder":"","group":"","integration":"","outboundWebhook":"","prometheusRule":"","recordingRuleGroupSet":"","ruleGroup":"","scope":"","tcoLogsPolicies":"","tcoTracesPolicies":"","view":"","viewFolder":""},"region":"","resources":{},"securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true},"webhooks":{"enabled":true}}` | Coralogix operator container config |
| coralogixOperator.domain | string | `""` | Coralogix Account Domain |
| coralogixOperator.image | object | `{"pullPolicy":"IfNotPresent","repository":"coralogixrepo/coralogix-operator","tag":""}` | Coralogix operator Image |
| coralogixOperator.labelSelector | object | `{}` | A selector to filter custom resources (by the custom resources' labels). {} matches all custom resources. Cannot be set to nil. |
| coralogixOperator.leaderElection | object | `{"enabled":true}` | Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager. |
| coralogixOperator.namespaceSelector | object | `{}` | A selector to filter namespaces (by the namespace's labels). {} matches all namespaces. Cannot be set to nil. |
| coralogixOperator.reconcileIntervalSeconds | object | `{"alert":"","alertScheduler":"","apiKey":"","customRole":"","dashboard":"","dashboardsFolder":"","group":"","integration":"","outboundWebhook":"","prometheusRule":"","recordingRuleGroupSet":"","ruleGroup":"","scope":"","tcoLogsPolicies":"","tcoTracesPolicies":"","view":"","viewFolder":""}` | The interval in seconds to reconcile each custom resource |
| coralogixOperator.region | string | `""` | Coralogix Account Region |
| coralogixOperator.resources | object | `{}` | resource config for Coralogix operator |
| coralogixOperator.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Security context for Coralogix operator container |
| deployment.replicas | int | `1` | How many coralogix-operator pods to run |
| fullnameOverride | string | `""` | Provide a name to substitute for the full names of resources |
| imagePullSecrets | list | `[]` |  |
| issuer.create | bool | `true` | Specifies whether an Issuer should be created. |
| nameOverride | string | `""` | Provide a name in place of coralogix-operator for `app:` labels |
| nodeSelector | object | `{}` | ref: https://kubernetes.io/docs/user-guide/node-selection/ |
| podAnnotations | object | `{}` | Annotations to add to the operator pod |
| secret | object | `{"annotations":{},"create":true,"data":{"apiKey":""},"labels":{},"secretKeyReference":{}}` | Configuration for Coralogix operator secret |
| secret.annotations | object | `{}` | Annotations to add to the Coralogix operator secret |
| secret.create | bool | `true` | Indicates if the Coralogix operator secret should be created |
| secret.data | object | `{"apiKey":""}` | Coralogix operator secret data |
| secret.labels | object | `{}` | Labels to add to the Coralogix operator secret |
| secret.secretKeyReference | object | `{}` | secret.data and secret.secretKeyReference should be mutually exclusive. |
| securityContext | object | `{"fsGroup":2000,"runAsGroup":2000,"runAsNonRoot":true,"runAsUser":2000,"seccompProfile":{"type":"RuntimeDefault"}}` | ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/ |
| serviceAccount | object | `{"annotations":{},"create":true,"name":""}` | ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | If not set and create is true, a name is generated using the fullname template |
| serviceMonitor | object | `{"create":true}` | Service monitor for Prometheus to use. |
| serviceMonitor.create | bool | `true` | Specifies whether a service monitor should be created. |
| tolerations | list | `[]` | ref: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ |
| validatingWebhookConfiguration.create | bool | `true` | Specifies whether a ValidatingWebhookConfiguration should be created. |

