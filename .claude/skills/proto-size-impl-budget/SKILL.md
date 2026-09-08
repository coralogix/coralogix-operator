---
name: proto-size-impl-budget
description: "Use when reviewing or adding an operator CRD. Estimate types+controller size from pinned SDK fields and flag 3x overruns. Do NOT use for small field tweaks or dashboard JSON-blob edits."
---

# SDK size vs CRD implementation budget

**Trigger:** A new or large `*_types.go` plus controller for one management API.

**Fix:** Count exported JSON-tagged fields on generated model structs in the **pinned** `coralogix-management-sdk` from `go.mod` (`go/openapi/gen/<service>`). Skip duplicated OpenAPI filter/error types. Do not use live proto HEAD. If the SDK has no types for that API, skip the ratio. Compare implementation lines (CRD Spec/Status types + Extract/Handle/Reconcile; skip tests and generated deepcopy). Operator has no flatten; spec is the source of truth.

- Expected: `2.7 × fields + 75`
- Typical band: about `2×fields+65` to `4.5×fields+80`

Flag only extremes: about **3× the midpoint** or more. In-band is not a pass; still read the code. Below-band is often a JSON blob (valid if that was the intent, as with Dashboard). Alert is a deep-expand class (~5 lines/field); do not score other APIs against it.

**Why:** The pinned SDK is the surface this operator can implement. Typical APIs are not linear enough for a linter. A large ratio still catches expanding a small API like a dashboard.
