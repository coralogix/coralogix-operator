package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// wrappedIDBeside is the WrappedThing contract whose Update item has no id: the Update body has
// the id only beside the wrapper.
func wrappedIDBeside(t *testing.T) string {
	t.Helper()
	return replaceOnce(t, wrappedContract(t, true, false),
		"    WrappedThingUpdate:\n      type: object\n      required: [name]\n      properties:\n        id: {type: string}\n",
		"    WrappedThingUpdate:\n      type: object\n      required: [name]\n      properties:\n")
}

// opgen reads from the contract that Update sends the id beside the request wrapper, and tells the
// shared model, which would otherwise look for the id in the wrapper.
func TestUpdateIDBesideTheWrapperComesFromTheContract(t *testing.T) {
	files, err := renderWrapped(t, wrappedIDBeside(t), wrappedFile(wrappedRenames))
	if err != nil {
		t.Fatal(err)
	}
	convert := string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"])
	update := functionBody(t, convert, ") UpdateRequest(")
	if !strings.Contains(update, "out.Id = ") || !strings.Contains(update, "in.Status.ID") {
		t.Errorf("Update must send the status id beside the wrapper:\n%s", update)
	}
}

// wrappedGetOnly adds two fields that only Get has: an enum without an UNSPECIFIED zero value,
// which the shared contract checks reject, and a plain string.
func wrappedGetOnly(t *testing.T) string {
	t.Helper()
	return wrappedIDBeside(t) + "        state: {$ref: '#/components/schemas/WrappedState'}\n        createdAt: {type: string}\n" +
		"    WrappedState:\n      type: string\n      enum: [PENDING, ACTIVE]\n"
}

// A skip of a field that only Get has changes no request. It is used when the contract checks fail
// under the field without it.
func TestSkipOfAFieldThatOnlyGetHas(t *testing.T) {
	_, err := renderWrapped(t, wrappedGetOnly(t), wrappedFile(wrappedRenames))
	if err == nil || !strings.Contains(err.Error(), "ENUM_ZERO_INVALID") {
		t.Fatalf("want the enum error of the shared checks, got %v", err)
	}
	files, err := renderWrapped(t, wrappedGetOnly(t), wrappedFile(wrappedRenames+"      state: {skip: true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(files["api/coralogix/v1alpha1/wrappedthing_types.go"]), "State") {
		t.Error("the CR must not have the skipped field")
	}
	_, err = renderWrapped(t, wrappedGetOnly(t), wrappedFile(wrappedRenames+"      state: {skip: true}\n      createdAt: {skip: true}\n"))
	wantCode(t, err, codeOverrideUnused)
	if err == nil || !strings.Contains(err.Error(), "createdAt") {
		t.Fatalf("want the unused skip of createdAt, got %v", err)
	}
}

// setByHookFile declares the hooks that set the field, and the spec field of a handwritten type
// that they read.
func setByHookFile(lines string) string {
	return wrappedFile(lines) + "spec:\n  noteSource: {type: WrappedNoteSource, value: true}\nhooks: [beforeCreate, beforeUpdate]\n"
}

// A field that a hook sets is not in the CR, and the conversions leave it out. Its requestAs
// object stays in the requests for the other fields of the object.
func TestSetByHookLeavesTheFieldToTheHooks(t *testing.T) {
	files, err := renderWrapped(t, wrappedIDBeside(t), setByHookFile(
		"      title: {requestAs: heading}\n      note: {requestAs: extra.note, setByHook: true}\n      tag: {requestAs: extra.tag}\n"))
	if err != nil {
		t.Fatal(err)
	}
	types := string(files["api/coralogix/v1alpha1/wrappedthing_types.go"])
	if strings.Contains(types, "Note *string") {
		t.Errorf("the CR must not have the field that a hook sets:\n%s", types)
	}
	if !strings.Contains(types, "NoteSource WrappedNoteSource `json:\"noteSource,omitempty\"`") {
		t.Errorf("a value spec field is the type itself:\n%s", types)
	}
	convert := string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"])
	for _, fn := range []string{"toWrappedThingCreateCreate", "toWrappedThingUpdateUpdate"} {
		body := functionBody(t, convert, ") "+fn+"(")
		if strings.Contains(body, ".Note =") {
			t.Errorf("%s must leave the field to the hook:\n%s", fn, body)
		}
		if !strings.Contains(body, ".Tag = ") || strings.Count(body, "out.Extra = ") != 1 {
			t.Errorf("%s must still build the object for the other field:\n%s", fn, body)
		}
	}
	if !strings.Contains(string(files["internal/controller/coralogix/v1alpha1/wrappedthing_controller.go"]), "r.beforeCreate(ctx, log, cr, body)") {
		t.Error("the controller must call the hook")
	}
}

func TestSetByHookOverrideRules(t *testing.T) {
	head := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nhooks: [beforeCreate, beforeUpdate]\ntypes:\n"
	if _, err := overrides.Parse([]byte(head + "  Thing:\n    fields:\n      note: {requestAs: extra.note, setByHook: true}\n")); err != nil {
		t.Fatalf("requestAs and setByHook go together: %v", err)
	}
	for name, file := range map[string]string{
		"without hooks":                  "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nhooks: [beforeCreate]\ntypes:\n  Thing:\n    fields:\n      note: {setByHook: true}\n",
		"a new kind":                     "resource: Thing\nmode: new\nhooks: [beforeCreate, beforeUpdate]\ntypes:\n  Thing:\n    fields:\n      note: {setByHook: true}\n",
		"a nested field":                 head + "  ThingItem:\n    fields:\n      name: {setByHook: true}\n",
		"with skip":                      head + "  Thing:\n    fields:\n      note: {setByHook: true, skip: true}\n",
		"with another key":               head + "  Thing:\n    fields:\n      note: {setByHook: true, name: memo}\n",
		"a blank skip":                   head + "  Thing:\n    fields:\n      note: {setByHook: false}\n",
		"a spec type that is not a name": "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nspec:\n  extra: {type: 'map[string]string', value: true}\nhooks: [beforeCreate]\n",
	} {
		if _, err := overrides.Parse([]byte(file)); err == nil {
			t.Errorf("%s: the file parses", name)
		}
	}
}

// An e2e.yaml can leave out the update step, with the reason, when the API accepts no other value
// of a field that the step can change. An upgrade field with a dot patches a path below a spec field.
func TestE2EUpdateSkipAndNestedUpgradeField(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "values:\n  destinations: [https://example.com]\nupdate:\n  skip: the API accepts no other value\n"+
		"upgrade:\n  from: '1.0'\n  spec: {name: '@{run}', labels: {team: a}}\n  field: labels.team\n  value: b\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	test := readGenerated(t, root, "tests/e2e/thing_generated_test.go")
	if strings.Contains(test, `It("updates the Thing"`) || !strings.Contains(test, "// No update step: the API accepts no other value") {
		t.Errorf("the update step must be left out with its reason:\n%s", test)
	}
	script := readGenerated(t, root, "scripts/upgrade-check-thing.sh")
	if !strings.Contains(script, `'{} | setpath(["spec"] + ($field | split(".")); $value)'`) {
		t.Errorf("the upgrade check must patch the nested path:\n%s", script)
	}
	for name, tc := range map[string]struct{ e2e, code string }{
		"a skip with a field":     {"values:\n  destinations: [https://example.com]\nupdate:\n  skip: reason\n  field: name\n", codeE2EInvalid},
		"a blank skip":            {"values:\n  destinations: [https://example.com]\nupdate:\n  skip: ' '\n", codeE2EInvalid},
		"a path outside the spec": {"values:\n  destinations: [https://example.com]\nupgrade:\n  from: '1.0'\n  spec: {name: '@{run}'}\n  field: labels.team\n  value: b\n", codeE2EInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			writeThingE2E(t, root, tc.e2e)
			wantCode(t, generateFromInput(input, root, root), tc.code)
		})
	}
}

// Create and Update always build the requestAs object of the field that a hook sets, so the hook
// sets the field in a present object, also when the CR sets no other field of the object.
func TestSetByHookBuildsTheObjectsOfItsPath(t *testing.T) {
	files, err := renderWrapped(t, wrappedIDBeside(t), setByHookFile(
		"      title: {requestAs: heading}\n      note: {requestAs: extra.note, setByHook: true}\n      tag: {requestAs: extra.tag}\n"))
	if err != nil {
		t.Fatal(err)
	}
	convert := string(files["api/coralogix/v1alpha1/wrappedthing_convert.go"])
	for _, fn := range []string{"toWrappedThingCreateCreate", "toWrappedThingUpdateUpdate"} {
		body := functionBody(t, convert, ") "+fn+"(")
		if strings.Contains(body, ".Note =") {
			t.Errorf("%s must leave the field to the hook:\n%s", fn, body)
		}
		if strings.Count(body, "out.Extra = ") != 1 || strings.Count(body, "if len(in.Tag) != 0 {") != 1 {
			t.Errorf("%s must build the object of the field whether or not the CR sets tag:\n%s", fn, body)
		}
	}
}
