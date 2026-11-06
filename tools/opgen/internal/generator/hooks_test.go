package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const hookFileHead = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"

func TestHooksKeyParse(t *testing.T) {
	for _, tt := range []struct {
		name, file, wantErr string
	}{
		{name: "every hook but delete", file: hookFileHead + "hooks: [validate, beforeCreate, beforeUpdate, afterCreate, afterUpdate, beforeDelete]\n"},
		{name: "delete alone", file: hookFileHead + "hooks: [delete]\n"},
		{name: "unknown hook", file: hookFileHead + "hooks: [beforeSave]\n", wantErr: "not a hook"},
		{name: "duplicate hook", file: hookFileHead + "hooks: [validate, validate]\n", wantErr: "listed twice"},
		{name: "empty list", file: hookFileHead + "hooks: []\n", wantErr: "the list is empty"},
		{name: "null list", file: hookFileHead + "hooks:\n", wantErr: "the list is empty"},
		{name: "delete with beforeDelete", file: hookFileHead + "hooks: [beforeDelete, delete]\n", wantErr: "declare one"},
		{name: "mode new", file: "resource: Thing\nmode: new\nhooks: [validate]\n", wantErr: "a new kind follows the contract"},
		{name: "status in mode new", file: "resource: Thing\nmode: new\nstatus: {externalId: {type: string}}\n", wantErr: "a new kind follows the contract"},
		{name: "every status type", file: hookFileHead + "status:\n  a: {type: string}\n  b: {type: int32}\n  c: {type: int64}\n  d: {type: bool, keepOnCleanup: true}\n"},
		{name: "status without hooks", file: hookFileHead + "status:\n  externalId: {type: string}\n"},
		{name: "status id", file: hookFileHead + "status:\n  id: {type: string}\n", wantErr: "every kind has this field"},
		{name: "status conditions", file: hookFileHead + "status:\n  conditions: {type: string}\n", wantErr: "every kind has this field"},
		{name: "status printableStatus", file: hookFileHead + "status:\n  printableStatus: {type: string}\n", wantErr: "every kind has this field"},
		{name: "status Go name of id", file: hookFileHead + "status:\n  ID: {type: string}\n", wantErr: "every kind has"},
		{name: "status Go name of conditions", file: hookFileHead + "status:\n  Conditions: {type: string}\n", wantErr: "every kind has"},
		{name: "status unknown type", file: hookFileHead + "status:\n  x: {type: float}\n", wantErr: "is not one of"},
		{name: "status no type", file: hookFileHead + "status:\n  x: {keepOnCleanup: true}\n", wantErr: "is not one of"},
		{name: "status bad name", file: hookFileHead + "status:\n  bad-name: {type: string}\n", wantErr: "not a status property name"},
		{name: "status same Go field", file: hookFileHead + "status:\n  external_id: {type: string}\n  externalId: {type: string}\n", wantErr: "the Go field"},
		{name: "status unknown key", file: hookFileHead + "status:\n  x: {type: string, pointer: false}\n", wantErr: "pointer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := overrides.Parse([]byte(tt.file))
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatal(err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("want an error with %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// A bad hooks line reaches the user as OVERRIDE_INVALID.
func TestBadHooksFileIsOverrideInvalid(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, kindsDir, "hookthing", overrides.FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(string) string{
		func(s string) string { return replaceOnce(t, s, "mode: existing", "mode: new") },
		func(s string) string { return strings.Replace(s, "beforeDelete]", "beforeDelete, delete]", 1) },
		func(s string) string { return strings.Replace(s, "hooks: [", "hooks: [nope, ", 1) },
	} {
		if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
			t.Fatal(err)
		}
		wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
	}
}

func readKindFile(t *testing.T, root, kind, suffix string) string {
	t.Helper()
	dir := controllerDir
	if suffix == "types" || suffix == "convert" {
		dir = apiDir
	}
	data, err := os.ReadFile(filepath.Join(root, dir, strings.ToLower(kind)+"_"+suffix+".go"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHookCallsAreGeneratedInTheLifecycleOrder(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller := readKindFile(t, root, "HookThing", "controller")
	creation, rest, _ := strings.Cut(controller, "func (r *HookThingReconciler) HandleCreation(")
	if !strings.Contains(creation, "func (r *HookThingReconciler) HandleCreationWithFollowUp(") {
		t.Fatal("a kind with afterCreate must implement HandleCreationWithFollowUp")
	}
	create := strings.Split(creation, "HandleCreationWithFollowUp(")[1]
	update := strings.Split(strings.Split(rest, "HandleUpdate(")[1], "HandleDeletion(")[0]
	deletion := strings.Split(strings.Split(rest, "HandleDeletion(")[1], "SavesHookStatus")[0]
	for _, tt := range []struct {
		name, code string
		order      []string
	}{
		{"create", create, []string{"r.validate(", "cr.CreateRequest(", "r.beforeCreate(", ").Execute()", "ptr.To(id)", "r.afterCreate("}},
		{"update", update, []string{"r.validate(", "cr.UpdateRequest(", "r.beforeUpdate(", ").Execute()", "r.afterUpdate("}},
		{"delete", deletion, []string{"r.beforeDelete(", ").Execute()"}},
	} {
		last := -1
		for _, call := range tt.order {
			i := strings.Index(tt.code, call)
			if i < 0 || i < last {
				t.Errorf("%s: %q is missing or out of order", tt.name, call)
			}
			last = i
		}
	}
	// A hook error has its own type, so the shared reconciler never reads it as a remote NotFound.
	for _, want := range []string{"NewValidationError(", "NewHookError("} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller never calls %s", want)
		}
	}
	// Every hook call has the safe-hook comment, and the comment names the guide.
	if got, want := strings.Count(controller, `See AGENTS.md "Hooks".`), 7; got != want {
		t.Errorf("want %d safe-hook comments, got %d", want, got)
	}
}

// With delete, the hook replaces the generated DELETE: the controller never calls the SDK DELETE.
func TestDeleteHookReplacesTheGeneratedDelete(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller := readKindFile(t, root, "HookDeleteThing", "controller")
	if !strings.Contains(controller, "r.delete(ctx, log, cr)") {
		t.Error("the controller does not call the delete hook")
	}
	if strings.Contains(controller, "DeleteHookDeleteThing(") || strings.Contains(controller, "cxsdk.IsNotFound") {
		t.Error("the controller still has the generated DELETE")
	}
	// afterCreate is not declared, so Create is the plain HandleCreation.
	if strings.Contains(controller, "HandleCreationWithFollowUp") {
		t.Error("a kind without afterCreate must keep HandleCreation")
	}
	// The Update response of HookDeleteThing is a resource; HookThing has an empty one.
	if !strings.Contains(readKindFile(t, root, "HookThing", "controller"), "r.afterUpdate(ctx, log, cr, resp)") {
		t.Error("no afterUpdate call")
	}
}

// A kind without hooks or status renders no hook code.
func TestKindWithoutHooksHasNoHookCode(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"Thing", "PatchThing", "RefThing", "BodyIDThing", "WrappedThing"} {
		controller := readKindFile(t, root, kind, "controller")
		for _, unwanted := range []string{"ClientSet", "NewHookError", "NewValidationError", "SavesHookStatus", "RemoteStatusFields", "WithFollowUp", "AGENTS.md"} {
			if strings.Contains(controller, unwanted) {
				t.Errorf("%s: the controller has %q", kind, unwanted)
			}
		}
		if types := readKindFile(t, root, kind, "types"); strings.Contains(types, "ExternalId") || strings.Contains(types, "Imported") {
			t.Errorf("%s: the types have an extra status field", kind)
		}
	}
}

// status without hooks adds the fields and the cleanup list, and needs no hook and no client set.
func TestStatusWithoutHooks(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, kindsDir, "thing", overrides.FileName)
	file := hookFileHead + "status:\n  externalId: {type: string}\n  imported: {type: bool, keepOnCleanup: true}\n"
	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readKindFile(t, root, "Thing", "types")
	for _, want := range []string{"ExternalId *string `json:\"externalId,omitempty\"`", "Imported *bool `json:\"imported,omitempty\"`"} {
		if !strings.Contains(types, want) {
			t.Errorf("the types miss %q", want)
		}
	}
	controller := readKindFile(t, root, "Thing", "controller")
	if !strings.Contains(controller, `{"status", "externalId"},`) || strings.Contains(controller, `"imported"`) {
		t.Errorf("the cleanup list must hold externalId and leave out the keepOnCleanup field:\n%s", controller)
	}
	for _, unwanted := range []string{"SavesHookStatus", "ClientSet", "NewHookError"} {
		if strings.Contains(controller, unwanted) {
			t.Errorf("the controller has %q without a hook", unwanted)
		}
	}
}

// <kind>_hooks.go has no generator header, so the generator never removes it. Removing the kind
// removes the generated files and keeps the hooks file.
func TestRemovingAKindKeepsItsHooksFile(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(root, controllerDir, "hookthing_hooks.go")
	if err := os.MkdirAll(filepath.Dir(hooks), 0o755); err != nil {
		t.Fatal(err)
	}
	const handwritten = "package v1alpha1\n\n// Handwritten hooks.\n"
	if err := os.WriteFile(hooks, []byte(handwritten), 0o644); err != nil {
		t.Fatal(err)
	}
	// The generator does not replace it either.
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}

	kinds, err := os.ReadFile(filepath.Join(root, kindsFile))
	if err != nil {
		t.Fatal(err)
	}
	without := replaceOnce(t, string(kinds), "  - kind: HookThing\n    resource: HookThing\n", "")
	if err := os.WriteFile(filepath.Join(root, kindsFile), []byte(without), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, kindsDir, "hookthing")); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{
		filepath.Join(controllerDir, "hookthing_controller.go"),
		filepath.Join(apiDir, "hookthing_types.go"),
		filepath.Join(apiDir, "hookthing_convert.go"),
	} {
		if _, err := os.Stat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Errorf("%s must be removed with its kind", gone)
		}
	}
	got, err := os.ReadFile(hooks)
	if err != nil || string(got) != handwritten {
		t.Fatalf("the hooks file must stay as it is: %v\n%s", err, got)
	}
}
