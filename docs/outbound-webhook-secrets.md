# OutboundWebhook Secret references

Use a Kubernetes Secret for an outbound webhook credential. The Secret must be in the same namespace as the `OutboundWebhook`.

## Fields

| Webhook type | Secret reference field |
| --- | --- |
| PagerDuty | `pagerDuty.serviceKeySecretKeyRef` |
| Jira | `jira.apiTokenSecretKeyRef` |
| Generic HTTP | `genericWebhook.headersFromSecret.<header name>` |
| Generic HTTP, Slack, SendLog, Microsoft Teams, Jira, Opsgenie, Demisto | `<type>.urlSecretKeyRef` |

Use `urlSecretKeyRef` when the URL contains a credential. Use `headersFromSecret` for credential header values. Plain URLs and headers can stay inline.

For example, a PagerDuty webhook can read its service key from a Secret:

```yaml
spec:
  outboundWebhookType:
    pagerDuty:
      serviceKeySecretKeyRef:
        name: pagerduty-credentials
        key: service-key
```

The operator reads the Secret on create and update. A Secret change triggers a new reconcile. A missing Secret or key leaves the webhook unsynced. A reference is required even if its `optional` field is set to `true`.

## Move an existing webhook

1. Create a Secret through your normal secret management process.
2. Replace the inline value with its Secret reference in one manifest update. A field cannot have both sources.
3. Check that the `RemoteSynced` condition becomes true. Rotate any credential that was previously in a manifest or operator log.

The inline `pagerDuty.serviceKey` and `jira.apiToken` fields are deprecated but remain supported. No removal version is set.
