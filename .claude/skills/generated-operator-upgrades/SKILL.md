---
name: generated-operator-upgrades
description: Keep operator upgrade checks generated when migrating a released custom resource to opgen.
---

# Generated operator upgrades

Use this skill when an existing operator kind moves from handwritten code to opgen.

- Keep the existing e2e tests and upgrade matrix running.
- Put live test values, dependencies, and the released chart spec in the kind's `tools/opgen/kinds/<kind>/e2e.yaml`.
- Generate the current e2e lifecycle test and the released chart upgrade check. Do not add handwritten test hooks for a generated kind.
- Keep the released chart spec fixed to fields that chart accepts. A new API field belongs in the current generated e2e test. It must not enter the old chart's create step.
- Check the same status ID after upgrade. Then update and delete the resource with the new operator.
- Use a release that can create an isolated resource. An old GlobalRouter release that writes `router_default` is unsafe on a shared account.
- Simulate a new optional SDK field in a generator test. Verify that current outputs gain it and the released chart spec stays fixed.
