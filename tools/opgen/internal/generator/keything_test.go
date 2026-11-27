package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The golden KeyThing has the shapes of the API keys service. Create nests the lists in an object
// that the CR inlines (inline, requestAs), Update wraps each list and sends an empty wrapper for a
// list that the CR leaves out (unwrap, sendNullAsEmpty), and enabled is in Update only
// (updateDefault).
func TestKeyThingRequests(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	conversion := readGenerated(t, root, apiDir+"/keything_convert.go")
	create := functionBody(t, conversion, ") toCreateKeyThingRequest(")
	for _, want := range []string{
		"&keythingsservice.CreateKeyThingRequestKeyPermissions{}",
		".Presets = in.Presets",
		".Permissions = in.Permissions",
		"out.KeyPermissions = *obj",
	} {
		if !strings.Contains(create, want) {
			t.Errorf("Create misses %q:\n%s", want, create)
		}
	}
	if strings.Contains(create, "Enabled") {
		t.Errorf("Create cannot send enabled:\n%s", create)
	}
	update := functionBody(t, conversion, ") toUpdateKeyThingRequest(")
	for _, want := range []string{
		"out.IsEnabled = in.Enabled",
		"out.NewName = &",
		"keythingsservice.KeyThingPresets{}",
		".Presets = []string{}",
		".Permissions = []string{}",
		"out.Presets = &w",
		"out.Permissions = &w",
	} {
		if !strings.Contains(update, want) {
			t.Errorf("Update misses %q:\n%s", want, update)
		}
	}
	if strings.Contains(update, "KeyPermissions") {
		t.Errorf("Update holds the lists at the top level:\n%s", update)
	}
	types := readGenerated(t, root, apiDir+"/keything_types.go")
	for _, want := range []string{
		"// +kubebuilder:default=true\n\tEnabled *bool `json:\"enabled,omitempty\"`",
		"Presets []string `json:\"presets,omitempty\"`",
		"Permissions []string `json:\"permissions,omitempty\"`",
	} {
		if !strings.Contains(types, want) {
			t.Errorf("the types miss %q:\n%s", want, types)
		}
	}
	if strings.Contains(types, "KeyPermissions") {
		t.Error("the CR shows the lists of keyPermissions at the top level")
	}
}

// writeKeyThingOverrides replaces the behavior-overrides file of KeyThing.
func writeKeyThingOverrides(t *testing.T, root, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, kindsDir, "keything", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func keyThingOverrides(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "root", kindsDir, "keything", "behavior-overrides.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A request that nests an inlined object needs a requestAs line for each field of the object, or
// the request would lose the field.
func TestInlineOfAnObjectThatARequestNestsNeedsRequestAs(t *testing.T) {
	root, input := syntheticRoot(t)
	writeKeyThingOverrides(t, root, replaceOnce(t, keyThingOverrides(t), "requestAs: keyPermissions.permissions, ", ""))
	err := generateFromInput(input, root, root)
	if err == nil || !strings.Contains(err.Error(), "INLINE_UNSUPPORTED") || !strings.Contains(err.Error(), "keyPermissions.permissions") {
		t.Fatalf("want INLINE_UNSUPPORTED for keyPermissions.permissions, got %v", err)
	}
}

// A field that only Update has is in the CR only with an updateDefault line, which states the
// value that Create stores.
func TestUpdateOnlyFieldNeedsUpdateDefault(t *testing.T) {
	root, input := syntheticRoot(t)
	writeKeyThingOverrides(t, root, replaceOnce(t, keyThingOverrides(t), ", updateDefault: 'true'", ""))
	err := generateFromInput(input, root, root)
	if err == nil || !strings.Contains(err.Error(), "FIELD_LIFECYCLE_UNSUPPORTED") {
		t.Fatalf("want the lifecycle error of the shared model, got %v", err)
	}
}

func TestUpdateDefaultAndItemNameLines(t *testing.T) {
	const head = "resource: S\nmode: existing\nvalidators: {inferred: true}\n"
	for _, c := range []struct {
		name, file, want string
	}{
		{"updateDefault on a nested component", head + "types:\n  Inner:\n    fields:\n      on: {updateDefault: 'true'}\n", "updateDefault: use it on a released resource field"},
		{"updateDefault in a new kind", "resource: S\nmode: new\ntypes:\n  S:\n    fields:\n      on: {updateDefault: 'true'}\n", "updateDefault: use it on a released resource field"},
		{"updateDefault with requestAs", head + "types:\n  S:\n    fields:\n      on: {updateDefault: 'true', requestAs: enabled}\n", "updateDefault: the line cannot set"},
		{"itemName in a new kind", "resource: S\nmode: new\ntypes:\n  S:\n    fields:\n      presets: {itemName: name}\n", "itemName: a new kind follows the contract"},
		{"itemName that is no property name", head + "types:\n  S:\n    fields:\n      presets: {itemName: 'a.b'}\n", "is not a property name"},
		{"itemName with unwrap", head + "types:\n  S:\n    fields:\n      presets: {itemName: name, unwrap: true}\n", "itemName: the line cannot set"},
		{"skip with itemName", head + "types:\n  S:\n    fields:\n      presets: {skip: true, itemName: name}\n", "skip cannot change another property"},
		{"skip with updateDefault", head + "types:\n  S:\n    fields:\n      on: {skip: true, updateDefault: 'true'}\n", "skip cannot change another property"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := overrides.Parse([]byte(c.file))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q, got %v", c.want, err)
			}
		})
	}
}
