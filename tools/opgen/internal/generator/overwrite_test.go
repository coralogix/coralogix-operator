package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The golden ThingPolicies is an overwrite resource: its status has no id, every reconcile takes
// the Update path, and Delete writes the list empty.
func TestOverwriteKindHasNoStatusID(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thingpolicies_types.go")
	if strings.Contains(types, "ID *string") {
		t.Error("an overwrite resource has no status id")
	}
	if !strings.Contains(types, "func (in *ThingPolicies) HasIDInStatus() bool {\n\treturn true\n}") {
		t.Error("an overwrite resource always takes the Update path")
	}
	controller := readGenerated(t, root, controllerDir+"/thingpolicies_controller.go")
	for _, want := range []string{
		"Policies: []thingpoliciesservice.CreateThingPolicyRequest{},",
		"return coralogixreconciler.AddFinalizer(ctx, log, cr, r)",
		"cxsdk.IsNotFound(apiErr)",
	} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller lacks %q", want)
		}
	}
	if strings.Contains(controller, "Status.ID") {
		t.Error("the controller of an overwrite resource reads no status id")
	}
	e2e := readGenerated(t, root, e2eDir+"/thingpolicies_generated_test.go")
	if !strings.Contains(e2e, "Ordered, Serial") {
		t.Error("the e2e test of an overwrite resource runs alone")
	}
	script := readGenerated(t, root, "scripts/upgrade-check-thingpolicies.sh")
	if strings.Contains(script, "status.id") || !strings.Contains(script, "--type json") {
		t.Errorf("the upgrade check of an overwrite resource has no id and patches the list:\n%s", script)
	}
}

// An inline line shows the fields of an object beside the fields of its component, a request
// wrapper of a nested component moves the fields back into it, and joinedBy joins a list.
func TestInlineWrapperAndJoinedFields(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thingpolicies_types.go")
	for _, want := range []string{"Pattern *string", "Levels []string", "Tags []string", "Disabled *bool"} {
		if !strings.Contains(types, want) {
			t.Errorf("the item lacks %q", want)
		}
	}
	for _, unwanted := range []string{"Rules *", "OtherRules", "Enabled", "XValidation"} {
		if strings.Contains(types, unwanted) {
			t.Errorf("the item must not have %q", unwanted)
		}
	}
	convert := readGenerated(t, root, apiDir+"/thingpolicies_convert.go")
	for _, want := range []string{
		"obj3.Disabled = in.Disabled",
		`strings.Join(in.Tags, ",")`,
		"out.Rules = *obj5",
		"out.Item = *obj3",
	} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion lacks %q:\n%s", want, convert)
		}
	}
}

func TestNewKeysAreChecked(t *testing.T) {
	const head = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"
	for name, tc := range map[string]string{
		"inline in a new kind":           "resource: Thing\nmode: new\ntypes:\n  Item:\n    fields:\n      rules: {inline: true}\n",
		"inline on the resource":         head + "types:\n  Thing:\n    fields:\n      rules: {inline: true}\n",
		"inline with another key":        head + "types:\n  Item:\n    fields:\n      rules: {inline: true, name: x}\n",
		"inline with skip":               head + "types:\n  Item:\n    fields:\n      rules: {inline: true, skip: true}\n",
		"joinedBy in a new kind":         "resource: Thing\nmode: new\ntypes:\n  Item:\n    fields:\n      tags: {joinedBy: \",\"}\n",
		"joinedBy with a ref":            head + "types:\n  Item:\n    fields:\n      tags: {joinedBy: \",\", ref: {kind: Thing}}\n",
		"requestWrapper on the resource": head + "types:\n  Thing:\n    requestWrapper: item\n",
		"requestWrapper in a new kind":   "resource: Thing\nmode: new\ntypes:\n  Item:\n    requestWrapper: item\n",
		"requestWrapper that is no name": head + "types:\n  Item:\n    requestWrapper: a.b\n",
		"overwrite without operations":   head + "api:\n  overwrite: {}\n",
		"overwrite with requestWrapper":  head + "api:\n  requestWrapper: x\n  overwrite: {}\n  operations: {create: W, get: G, update: W, delete: W}\n",
	} {
		if _, err := overrides.Parse([]byte(tc)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := overrides.Parse([]byte(head + "api:\n  overwrite: {}\n  operations: {create: W, get: G, update: W, delete: W}\ntypes:\n  Item:\n    requestWrapper: item\n    fields:\n      rules: {inline: true}\n      tags: {joinedBy: \",\"}\n")); err != nil {
		t.Errorf("the overwrite keys must be accepted: %v", err)
	}
}

// Two CR paths of one component share a Go type when their validators lines are the same.
func TestEqualNestedValidationSharesAType(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: Thing\nmode: existing\nvalidators:\n  inferred: true\n  fields:\n" +
		"    spec.a.names: {add: ['+kubebuilder:validation:MinItems=1']}\n" +
		"    spec.b.names: {add: ['+kubebuilder:validation:MinItems=1']}\n" +
		"    spec.c.names: {add: ['+kubebuilder:validation:MinItems=2']}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{file: file}
	if !b.sameNestedValidation("spec.a", "spec.b") {
		t.Error("equal lines must share a type")
	}
	if b.sameNestedValidation("spec.a", "spec.c") || b.sameNestedValidation("spec.a", "spec.d") {
		t.Error("different lines need their own types")
	}
}

// An inlined arm drops the rule of its group, also when every arm is settable.
func TestInlinedArmDropsItsGroup(t *testing.T) {
	typ := &model.Type{Kind: model.Object, Groups: []model.OneOfGroup{{Arms: []string{"a", "b"}}}, Fields: []*model.Field{
		{Name: "a", Behavior: model.Normal, Create: &model.Attrs{}},
		{Name: "b", Behavior: model.Normal, Create: &model.Attrs{}},
	}}
	if groups := settableGroups(typ, map[string]bool{"a": true}); len(groups) != 0 {
		t.Errorf("want no group, got %v", groups)
	}
	if groups := settableGroups(typ, nil); len(groups) != 1 {
		t.Errorf("a group of settable arms keeps its rule, got %v", groups)
	}
}

// A joinedBy field cannot take the API default of the joined string.
func TestJoinedFieldRejectsADefault(t *testing.T) {
	file, err := overrides.Parse([]byte("resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Item:\n    fields:\n      tags: {joinedBy: \",\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{file: file, used: map[overrides.Line]bool{}, usedValidators: map[string]bool{}}
	def := ""
	f := &crField{API: "tags", JSON: "tags", joinedBy: ",", typ: &model.Type{Kind: model.String}}
	if err := b.contractDefault(f, &model.Attrs{Default: &def}, "spec.tags"); err == nil {
		t.Error("want an error for a default on a joinedBy field")
	}
}
