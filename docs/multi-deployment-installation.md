# Running Multiple Deployments of the Coralogix Operator

The Coralogix Operator supports running multiple deployments within the same Kubernetes cluster, each managing only a subset of custom resources. 
This is achieved using **`label-selector`** flag, which filters custom resources based on specific labels,
and **`namespace-selector`** flag, which filters custom resources based on the namespace they are deployed in.

## How It Works

- By setting the `--label-selector` flag, the operator will **only reconcile resources that match the specified label selector**. 
- By setting the `--namespace-selector` flag, the operator will **only reconcile resources that are deployed in the specified namespaces**.
This allows for multiple independent deployments of the operator, each managing a different subset of resources.

#### Example: Deploying an Operator using the label-selector flag
```sh
helm install coralogix-operator-staging coralogix/coralogix-operator \
  --set secret.data.apiKey="stg-api-key" \
  --set coralogixOperator.region="eu2" \
  --set coralogixOperator.labelSelector="env=stg,team=a"
```
This operator installation will **only reconcile custom resources** labeled:
```yaml
metadata:
  labels:
    ...
    env: stg
    team: a
    ...
```

#### Example: Deploying an Operator using the namespace-selector flag
```sh
helm install coralogix-operator-staging coralogix/coralogix-operator \
  --set secret.data.apiKey="stg-api-key" \
  --set coralogixOperator.region="eu2" \
  --set coralogixOperator.namespaceSelector="staging,production"
```
This operator installation will **only reconcile custom resources** deployed in either the `staging` or `production` namespaces.

---
 
This setup requires **isolation** between multiple operator deployments, to prevent conflicts between resources managed by different installations.
Leaving both the `--label-selector` and `--namespace-selector` flags empty will cause the operator to reconcile all resources, which may lead to conflicts between instances.
