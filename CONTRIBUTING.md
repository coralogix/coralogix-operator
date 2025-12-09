# Contributing to Coralogix Operator

Thank you for your interest in contributing to the Coralogix Operator. We welcome your contributions. Here you'll find information to help you get started with operator development.

You are contributing under the terms and conditions of the [Contributor License Agreement](LICENSE). [For signing](https://cla-assistant.io/coralogix/coralogix-operator).

### Building the Operator
---------------------
1. Clone the operator repository and navigate to the project directory:
```
git clone https://github.com/coralogix/coralogix-operator/v2.git 
cd coralogix-operator
```

2. Set the Coralogix API key and region as environment variables:
```sh
export CORALOGIX_API_KEY="<api-key>"
export CORALOGIX_REGION="<region>"
```

3. For a custom operator image, build and push your image to a registry:
```sh
make docker-build docker-push IMG=<some-registry>/coralogix-operator:<tag> 
```

4. Deploy the operator to the cluster with the image specified by `IMG`:
```sh
make deploy IMG=<some-registry>/coralogix-operator:<tag> 
```

Note: This will install Prometheus Operator CRDs on the cluster if not already installed.

5. To uninstall the operator, run:
```sh
make undeploy
```

### Running examples
---------------------
The project provides a [sample directory](./config/samples) for examples of custom resources.

To use an example, apply the resource to the cluster:
```sh
$ kubectl apply -f config/samples/v1beta1/alerts/metric_threshold.yaml
```

Getting the resource:
```sh
$ kubectl get alert metric-threshold -o yaml
```

Deleting the resource.
```sh
$ kubectl delete alert metric-threshold
```

Developing
---------------------
We use [kubebuilder](https://book.kubebuilder.io/) for developing the operator.
When creating or updating CRDs remember to run:
```sh
make manifests generate
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