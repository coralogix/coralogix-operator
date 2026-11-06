---
name: write-opgen-hook
description: Use when you add or change a hook of a generated kind (hooks key of behavior-overrides.yaml, <kind>_hooks.go). Holds the safe-hook rule, the safe pattern, and the test pattern. Do NOT use for a kind that needs no hook.
---

# Write an opgen hook

A hook is handwritten Go code that runs at a fixed point of the generated lifecycle. Read "Hooks" in `AGENTS.md` first. It has the keys, the signatures, the call order, and the error reasons.

## The rule: a hook must be safe to run many times

Every hook can run many times.

- `afterUpdate` runs on every reconcile interval, also when nothing changed.
- `afterUpdate` also repairs what a failed `afterCreate` missed.
- Delete hooks (`beforeDelete`, `delete`) run again after a failure, and on a selector mismatch.
- `beforeCreate` and `beforeUpdate` run again on every attempt.

**Safe** means: running the hook 1 time or 100 times leaves the same result.

```go
// Safe: make sure the Secret is right.
err := c.Get(ctx, key, existing)
if apierrors.IsNotFound(err) { return c.Create(ctx, desired) }
if err != nil { return err }
existing.Data = desired.Data
return c.Update(ctx, existing)
```

```go
// Unsafe: the second run fails with AlreadyExists, every time.
return c.Create(ctx, desired)
```

```go
// Unsafe: each run adds one more item.
body.Retentions = append(body.Retentions, extra)
```

```go
// Safe: set the whole field.
body.Retentions = buildRetentions(cr)
```

Checklist:

- Read the current state. Create it if it is missing. Update it if it differs.
- Do not create a remote object or a Kubernetes object blindly.
- Do not append to a body field that the generated code filled, unless the hook builds the whole field.
- Delete hooks accept a resource that is already gone. 404 is success.
- Keep API calls few. `afterUpdate` repeats on each interval.
- Return errors. Do not log and return.
- A hook error is never a remote NotFound. The shared reconciler checks `HookError` first, so a 404 from another API does not remove `status.id`.

## Where the code goes

- File: `internal/controller/coralogix/v1alpha1/<kind>_hooks.go`. It has no generated header.
- Each hook is an unexported method on `<Kind>Reconciler`. The signatures are in `AGENTS.md`.
- Put the `// +kubebuilder:rbac:...` markers that the hooks need in this file. Run `make manifests`.
- Use `r.ClientSet` for another Coralogix API. Use `config.GetClient()` for Kubernetes.
- Set an extra status field only if the kind declares it under `status:`. Use `keepOnCleanup: true` for a field that must survive a selector mismatch.

## Test pattern: call the hook twice

For each hook, call it twice on the same input. Check two things:

1. The result is the same.
2. No second remote or Kubernetes object exists.

```go
func TestAfterCreateIsSafeToRepeat(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	config.InitClient(fakeClient)
	r := &ThingReconciler{ClientSet: fakeClientSet}
	cr := newThing()

	require.NoError(t, r.afterCreate(ctx, logr.Discard(), cr, resp))
	first := snapshot(t, fakeClient) // the objects that the hook wrote

	require.NoError(t, r.afterCreate(ctx, logr.Discard(), cr, resp)) // the second run
	require.Equal(t, first, snapshot(t, fakeClient), "a second run must change nothing")
}
```

For a body hook, build the body twice and call the hook on a fresh body each time. Check that two runs on one body give the same body:

```go
body := &sdk.CreateThingRequest{}
require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, body))
once := deepCopy(body) // a JSON round trip is enough
require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, body))
require.Equal(t, once, body, "a second run must not add to the body")
```

For a delete hook, also call it when the resource is already gone. It must return no error.

## After you change a hook

- Run `make unit-tests`, `make lint`, and `make opgen-test`.
- The hook signature comes from the generated code. If `go build` says `r.beforeCreate undefined`, the method is missing. If `unused` says a method is unused, the `hooks` key does not declare it.
