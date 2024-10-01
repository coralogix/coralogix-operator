# Contributing to Coralogix Operator

Thank you for your interest in contributing to the Coralogix Operator. We welcome your contributions. Here you'll find information to help you get started with operator development.

You are contributing under the terms and conditions of the [Contributor License Agreement](LICENSE). [For signing](https://cla-assistant.io/coralogix/coralogix-operator).

Building the Operator
---------------------

### Requirements

- You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind)
- [Go](https://golang.org/doc/install) 1.20.x (for building the operator locally)
- [kuttl](https://kuttl.dev/) (for running e2e tests).
- [kubebuilder](https://book-v1.book.kubebuilder.io/getting_started/installation_and_setup.html) for developing the operator.

### Steps

Clone the repository locally.

```sh
$ git clone git@github.com:coralogix/coralogix-operator
```

Navigate to the operator directory and install it.
This command will build the operator and install the CRDs located at [crd directory](./config/crd) into the K8s cluster
specified in ~/.kube/config.

```sh
$ make install
```

Add the region and api-key as environment variables (or later as flags).

```sh
$ export CORALOGIX_API_KEY="xxx-xxx-xxx"
$ export CORALOGIX_REGION = "EU2"
```
For private domain the `domain` field or the environment variable `CORALOGIX_DOMAIN` have to be defined.

Run the operator locally
```sh
$ go run main.go
```
Or with `regin` and `api-key` flags
```sh
$ go run main.go -region EU2 -api-key xxx-xxx-xxx
```
For not running the prometheusRule controller set the `prometheus-rule-controller` flag to `false`
```sh
$ go run main.go -prometheus-rule-controller=false
```
Or build and push your image to a registry
```sh
make docker-build docker-push IMG=<some-registry>/coralogix-operator:tag
```
Then deploy it to the cluster
```sh
make deploy IMG=<some-registry>/coralogix-operator:tag
```

Running examples
```sh

---------------------
It's possible to use one of the samples in the [sample directory](./config/samples) or creating your own sample file.
Then apply it -

```sh
$ kubectl apply -f config/samples/alerts/standard_alert.yaml
```

Getting the resource status

```sh
$ kubectl get alerts.coralogix.com standard-alert-example -oyaml
```

Destroying the resource.

```sh
$ kubectl delete alerts.coralogix.com standard-alert-example
```

Developing
---------------------
We use [kubebuilder](https://book.kubebuilder.io/) for developing the operator.
When creating or updating CRDs remember to run 
```sh
make manifests
````

Running E2E Tests
---------------------
The test files are located at [./tests/e2e/](./tests/e2e).
In order to run the full e2e tests suite:
1. Add the api key and region as environment variables:
```sh
$ export CORALOGIX_API_KEY="<api-key>"
$ export CORALOGIX_REGION="<region>"
```
2. Run the tests:
```sh
$ make e2e-tests
````

Running Integration Tests
---------------------
We use [kuttl](https://kuttl.dev/) for integration tests.
The test files are located at [./tests/integration/](./tests/integration).
In order to run the full integration tests suite, run:
```sh
$ make integration-tests
````

*Note:* `kuttl` tests create real resources and in a case of failure some resources may not be removed.

Releases
---------------------
To determine the release convention we use [semantic-release](.releaserc.json) -
```sh
"releaseRules": [
          {"message": "major*", "release": "major"},
          {"message": "minor*", "release": "minor"},
          {"message": "patch*", "release": "patch"}
        ]
````