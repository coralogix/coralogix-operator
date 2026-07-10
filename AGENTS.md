# AGENTS.md

Project guidance for AI coding agents working in this repository.

## Public Repo

This is the public `coralogix/coralogix-operator` repo. Keep internal ticket IDs, customer names, private URLs, kubeconfigs, API keys, and other secrets out of committed code, comments, docs, samples, and tests.

Use generic, public-safe names in examples and fixtures.

## Architecture

This is a Kubebuilder v4 operator for `coralogix.com` custom resources.

API types live under `api/coralogix/v1alpha1` and `api/coralogix/v1beta1`. Generated DeepCopy code lives in `zz_generated.deepcopy.go`.

`cmd/main.go` wires scheme registration, config, Coralogix SDK clients, controller registration, optional PrometheusRule integration, health checks, and metrics.

Most resource controllers live in `internal/controller/coralogix/v1alpha1`. `Alert` lives in `internal/controller/coralogix/v1beta1`.

Most resource controllers delegate lifecycle flow to `internal/controller/coralogix/coralogix-reconciler/`. The shared reconciler handles create/update/delete routing, finalizers, `RemoteSynced` conditions, metrics, selector mismatch cleanup, and missing remote resources.

`internal/config/` owns flags, environment variables, region/domain URL derivation, selectors, reconcile intervals, and controller-runtime client/scheme globals.

`internal/utils/` holds kind constants, labels/annotations, condition reasons, and small shared helpers. `internal/monitoring/` owns operator/resource metrics.

## API and Codegen

Treat Go API types and Kubebuilder markers as the public CRD contract. Preserve backward compatibility unless a breaking change is explicitly requested.

For Coralogix API semantics, use the pinned `coralogix-management-sdk` and, when SDK behavior is ambiguous, the protobuf definitions from the canonical `coralogix/cx-management-apis` repository. This repo does not commit protobuf definitions.

Pay attention to presence semantics: pointers, protobuf `optional`, wrappers, empty slices, and zero values are not interchangeable.

Do not hand-edit generated files: `zz_generated.deepcopy.go`, `config/crd/bases/*.yaml`, or `docs/api.md`. Change source types/markers and run the relevant generation target.

When changing a CRD field, update the full path:

- API type, JSON tag, validation/defaulting markers, and status fields.
- Controller model conversion and create/update/delete behavior.
- Samples under `config/samples/`.
- Relevant KUTTL/e2e fixtures.
- Generated DeepCopy, CRDs/RBAC, API docs, and Helm chart CRDs.

## Reconciliation

Create, update, delete, and selector-mismatch cleanup must be idempotent.

Deletion must tolerate remote resources that are already gone.

Failed create/update paths must not leave misleading status IDs or `RemoteSynced=True`.

Use the shared reconciler for condition/status transitions unless a controller has a concrete reason to own them.

Return contextual errors from controllers; do not log and return the same error.

Before changing finalizer or status order, consider partial failures, retries, conflicts, and remote resources created before Kubernetes status/finalizer updates succeed.

Preserve `observedGeneration` behavior and `RemoteSynced` reasons from `internal/utils/conditions.go`.

When a remote resource is missing during update, the shared reconciler removes `status.id` so the next reconciliation can recreate it.

## Review Checklist

- Check CRD compatibility: field names, JSON tags, enum strings, defaulting, validation markers, required/optional behavior, and status shape.
- Trace touched fields end to end: CR spec -> controller conversion -> Coralogix API request -> remote response -> status/conditions -> samples/docs.
- Check null, empty, pointer, wrapper, and zero-value drift between Kubernetes objects, Go structs, SDK models, protobufs, and remote API defaults.
- Check finalizer/status ordering for partial failures, retries, conflicts, remote not-found responses, and selector mismatch cleanup.
- Confirm generated artifacts are in sync after API changes: DeepCopy, CRDs, docs, Helm chart CRDs, and samples.
- Prefer regression tests for create, second reconcile, update, remove optional fields, delete when remote is already gone, and status-missing recovery.

## Runtime Configuration

- `CORALOGIX_API_KEY` is required.
- Exactly one of `CORALOGIX_REGION` (`AP1`, `AP2`, `AP3`, `EU1`, `EU2`, `US1`, `US2`, `US3`) or `CORALOGIX_DOMAIN` is required.
- Optional selectors: `LABEL_SELECTOR` and `NAMESPACE_SELECTOR`.
- Per-kind reconcile intervals use `<KIND>_RECONCILE_INTERVAL_SECONDS`; nonzero values must be at least 30 seconds.

## Commands

```bash
make build              # Generate code and build bin/manager
make run                # Run the controller against the current kubeconfig
make lint               # Run golangci-lint
make unit-tests         # Generate manifests/code, install envtest, run controller tests
make manifests          # Regenerate CRDs and RBAC
make generate           # Regenerate DeepCopy methods
make generate-api-docs  # Regenerate docs/api.md
make helm-sync-check    # Verify chart CRDs match generated CRDs
make helm-update-crds   # Sync generated CRDs into the Helm chart
make helm-sync-docs     # Regenerate chart README with helm-docs
```

Cluster/API tests:

```bash
make integration-tests  # kubectl kuttl test
make e2e-tests          # go test ./tests/e2e/ -ginkgo.v -v
```

`make unit-tests` uses envtest assets for Kubernetes `1.30.3` and writes `cover.out`. Integration and e2e tests require a Kubernetes cluster and real Coralogix access.

## Validation

Use focused unit/envtest coverage for controller behavior, validation, status transitions, selector behavior, and idempotent delete/update flows.

Use KUTTL or e2e coverage when Kubernetes wiring, CRDs, Helm packaging, or real API behavior is part of the risk.

Before finishing code changes, prefer:

```bash
make generate
make manifests
make lint
make unit-tests
```

For CRD or chart changes, also run:

```bash
make generate-api-docs
make helm-sync-check
```

If tests cannot be run, state the skipped command and why it was skipped.

## Repo Map

Use this map to choose the smallest useful reading set before opening broad directories.

- `cmd/main.go` wires manager setup, SDK clients, schemes, controllers, health checks, and metrics.
- `api/coralogix/v1alpha1/*_types.go` and `api/coralogix/v1beta1/*_types.go` define CRD specs/statuses and Kubebuilder markers.
- `internal/controller/coralogix/coralogix-reconciler/coralogix_reconciler.go` is the shared lifecycle engine.
- `internal/controller/coralogix/v1alpha1/*_controller.go` and `internal/controller/coralogix/v1beta1/*_controller.go` contain per-kind logic and model conversion.
- `internal/controller/prometheusrule_controller.go` handles Prometheus Operator integration.
- `internal/config/` owns runtime config, selectors, reconcile intervals, and controller-runtime globals.
- `internal/utils/` holds kind names, labels/annotations, condition reasons, and shared helpers.
- `config/crd/bases/`, `config/rbac/`, `config/manager/`, `config/default/`, and `config/prometheus/` are generated or assembled deployment manifests.
- `config/samples/` contains public sample CRs.
- `charts/coralogix-operator/` contains the Helm chart.
- `docs/api.md` is generated from CRDs. `docs/prometheus-integration.md` and `docs/metrics.md` are hand-written.
- `tests/e2e/` contains Go/Ginkgo tests that hit real Coralogix APIs. `tests/integration/` contains KUTTL scenarios.
- `tools/cxo-observer/` is a supporting observer tool.

Primary resource kinds:

- `v1beta1`: `Alert`.
- `v1alpha1`: `AIEvaluation`, `AlertScheduler`, `ApiKey`, `ArchiveLogsTarget`, `ArchiveMetricsTarget`, `Connector`, `CustomEnrichment`, `CustomRole`, `Dashboard`, `DashboardsFolder`, `Enrichment`, `Events2Metric`, `GlobalRouter`, `Group`, `Integration`, `IPAccess`, `OutboundWebhook`, `Preset`, `QuotaAllocationRuleSet`, `RecordingRuleGroupSet`, `RuleGroup`, `Scope`, `SLO`, `TCOLogsPolicies`, `TCOTracesPolicies`, `View`, `ViewFolder`.

Quick navigation:

```bash
rg "<Kind>" api internal/controller config/samples tests
rg "HandleCreation|HandleUpdate|HandleDeletion" internal/controller/coralogix
rg "<json_or_status_field>" api internal/controller config/samples
```
