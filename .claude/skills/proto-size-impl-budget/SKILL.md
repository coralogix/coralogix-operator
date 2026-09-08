---
name: proto-size-impl-budget
description: "Use when reviewing or adding an operator CRD. Estimate types+controller size from proto fields and flag 3x overruns. Do NOT use for small field tweaks or dashboard JSON-blob edits."
---

# Proto size vs CRD implementation budget

**Trigger:** A new or large `*_types.go` plus controller for one management API.

**Fix:** Count numbered proto fields for that API in the public `coralogix/cx-management-apis` repo. If protos are missing, count fields on the generated SDK structs. If neither exists, skip the ratio. Compare implementation lines (CRD Spec/Status types + Extract/Handle/Reconcile; skip tests and generated deepcopy). Operator has no flatten; spec is the source of truth.

- Expected: `2.4 × fields + 75`
- Typical band: about `1.5×fields+50` to `4×fields+90`

Flag only extremes: about **3× the midpoint** or more. In-band is not a pass; still read the code. Below-band is often a JSON blob (valid if that was the intent, as with Dashboard). Alert is a deep-expand class (~4 lines/field); do not score other APIs against it. Use field count, not proto lines.

**Why:** Typical APIs are not linear enough for a linter. A large ratio still catches expanding a small proto like a dashboard.
