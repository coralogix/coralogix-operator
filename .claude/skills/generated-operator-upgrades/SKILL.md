---
name: generated-operator-upgrades
description: Keep operator upgrade checks generated when migrating a released custom resource to opgen.
---

# Generated operator upgrades

Use this skill when an existing operator kind moves from handwritten code to opgen.

- Keep the existing e2e tests and upgrade matrix running.
- Put live test values, dependencies, and the released chart spec in the kind's `tools/opgen/kinds/<kind>/e2e.yaml`.
- Generate the current e2e lifecycle test and the released chart upgrade check. Do not add handwritten test hooks for a generated kind.
- When codegen adds an upgrade script, wire its before, after, and always-run cleanup phases into the upgrade workflow. Gate each phase to chart versions that already contain the kind.
- Keep the released chart spec fixed to fields that chart accepts. A new API field belongs in the current generated e2e test. It must not enter the old chart's create step.
- Check the same status ID after upgrade. Then update and delete the resource with the new operator.
- Use a release that can create an isolated resource. An old GlobalRouter release that writes `router_default` is unsafe on a shared account.
- Simulate a new optional SDK field in a generator test. Verify that current outputs gain it and the released chart spec stays fixed.
- Remove the handwritten entry of the kind from every file that also has a `# BEGIN generated kinds` block: `PROJECT`, `config/crd/kustomization.yaml`, and the Helm `cluster_role.yaml`. Codegen adds the generated entry and does not remove the old one. A duplicate CRD entry breaks `kustomize build config/default`, and so `make install` and `make deploy`.
- Check the migration with `grep -c` of the kind in each of those files (one entry each), and run `bin/kustomize build config/default | grep -c 'name: <plural>.coralogix.com'` (one CRD).
