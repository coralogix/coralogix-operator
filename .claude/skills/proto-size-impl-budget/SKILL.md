---
name: proto-size-impl-budget
description: "Use when reviewing or adding an operator CRD. Estimate types+controller Go from pinned SDK fields and flag 3x overruns. Do NOT use for small field tweaks or dashboard JSON-blob edits."
---

# SDK size vs CRD implementation budget

**Trigger:** A new or large `*_types.go` plus controller for one management API.

**Fix:** Count exported JSON-tagged fields on generated model structs in the **pinned** `coralogix-management-sdk` from `go.mod` (`go/openapi/gen/<service>`). Skip duplicated OpenAPI filter/error types. Do not use live proto HEAD. If the SDK has no types for that API, skip the ratio.

Count **all non-test, non-example `.go` lines** for that CRD: Spec/Status types, controller, helpers, and any generated client copied into this repo. Skip tests, examples, docs, and generated deepcopy. Compare:

- Expected: `6.5 × fields`
- Typical band: about `4×` to `9×`

Flag only extremes: about **3× the midpoint** or more. A copied OpenAPI/SDK client is included in the line count, so it shows up here. In-band is not a pass; still read the code. Below-band is often a JSON blob (valid if that was the intent, as with Dashboard). Alert sits in this band (~7×). Do not give it a separate formula.

**Why:** The pinned SDK is the surface this operator can implement. Typical APIs are not linear enough for a linter. Counting all Go, not only Extract/Handle, is what catches a dumped generated client.
