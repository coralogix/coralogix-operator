# Running Multiple Instances of the Coralogix Operator

The Coralogix Operator supports running multiple instances within the same Kubernetes cluster, each managing only a subset of custom resources. 
This is achieved using the **`label-selector`** flag, which filters custom resources based on specific Kubernetes labels.

## How It Works

By setting the `--label-selector` flag, an instance of the operator will **only reconcile resources that match the specified label selector**. 
This allows for multiple independent instances of the operator, each managing a different subset of resources.

#### Example: Deploying an Operator for staging environment and team a
```sh
helm install coralogix-operator-staging coralogix/coralogix-operator \
  --set secret.data.apiKey="stg-api-key" \
  --set coralogixOperator.domain="app.stg.domain" \
  --set coralogixOperator.labelSelector="env=stg,team=a"
```
This instance will **only reconcile custom resources** labeled:
```yaml
metadata:
  labels:
    ...
    env: stg
    team: a
    ...
```

---
 
This setup requires **isolation** between multiple operator instances, to prevent conflicts between resources managed by different instances.
Leaving the `--label-selector` flag empty will cause the operator to reconcile all resources, which may lead to conflicts between instances.

