# cxo-observer

## Overview
`cxo-observer` is a CLI tool that collects Kubernetes resources related to the Coralogix Operator installation, 
including both core operator components and custom resources (CRs) managed by the operator.
It is useful for support, debugging, and exporting the current state of your Coralogix Operator installation.  
The output is compressed into a `.tar.gz` file containing YAML files organized by 
resource group, version, kind, and namespace, to be easily inspected and shared.

Support requests and issues reports should include the output of this tool, including the affected resources.

### Features
- Collects Kubernetes resources created by the Coralogix Helm chart (e.g. Deployment, CRDs, ServiceAccount).
- Collects Coralogix custom resources across the entire cluster by default,
  with optional filtering by namespace and label selectors.

---
## Installation
### Prerequisites
- [Go](https://golang.org/doc/install) 1.16 or later

```bash
go install github.com/coralogix/coralogix-operator/tools/cxo-observer@<your-operator-version>
```

## Usage

```bash
cxo-observer [flags]
```

Example:
```bash
cxo-observer --chart-namespace=observability --namespace-selector=production,staging --label-selector=team=backend,app=api
```
If no `--namespace-selector` nor `--label-selector` is provided, all custom resources across the entire cluster will be collected.

### Flags
```bash
$ cxo-observer -h
Usage of cxo-observer:
  -chart-name string
        The name of Coralogix Operator Helm chart release. (default "coralogix-operator")
  -chart-namespace string
        The namespace of Coralogix Operator Helm chart release.
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
  -label-selector string
        A comma-separated list of key=value labels to filter custom resources.
  -namespace-selector string
        A comma-separated list of namespaces to filter custom resources.
  -zap-devel
        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
  -zap-encoder value
        Zap log encoding (one of 'json' or 'console')
  -zap-log-level value
        Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
  -zap-stacktrace-level value
        Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
  -zap-time-encoding value
        Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```

### Output Structure
```text
output/
├── operator-resources/
│   ├── crds/
│   │   ├── <crd-name>.yaml
│   │   └── ...
│   ├── deployment.yaml
│   ├── service.yaml   
│   └── ...
└── custom-resources/
    ├── <namespace>/
    │   └── <group>/<version>/<kind>/<name>.yaml
```