package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const nestedMaskOverride = "resource: MaskThing\nmode: existing\nvalidators: {inferred: true}\napi:\n  updateMask:\n    nested: [rule]\n"

// checkMask checks MaskThing, or another resource, against the synthetic contract after edit.
func checkMask(t *testing.T, resource, override string, edit func(string) string) error {
	t.Helper()
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	openAPIPath, overridePath := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	text := string(contract)
	if edit != nil {
		text = edit(text)
	}
	if err := os.WriteFile(openAPIPath, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: resource, OpenAPIPath: openAPIPath, OverridesPath: overridePath})
}

// api.updateMask.nested names the fields of an object instead of the object, and only the oneOf arm
// that the CR sets. The golden MaskThing shows the method; this test pins its parts.
func TestNestedUpdateMask(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files := checkLifecycleFixture(t, "MaskThing", string(contract), nestedMaskOverride)
	convert := string(files["api/coralogix/v1alpha1/maskthing_convert.go"])
	for _, want := range []string{
		"func (in *MaskThing) UpdateMask() string {",
		`paths := []string{"name", "description", "rule.enabled", "rule.note"}`,
		"if in.Spec.Rule.Some != nil {\n\t\tpaths = append(paths, \"rule.some\")",
		"if in.Spec.Rule.Prefix != nil {\n\t\tpaths = append(paths, \"rule.prefix\")",
	} {
		if !strings.Contains(convert, want) {
			t.Errorf("maskthing_convert.go omits %q", want)
		}
	}
	if strings.Contains(convert, "const MaskThingUpdateMask") {
		t.Error("a mask with arms is a method, not a const")
	}
	controller := string(files["internal/controller/coralogix/v1alpha1/maskthing_controller.go"])
	if !strings.Contains(controller, ".UpdateMask(cr.UpdateMask())") {
		t.Error("the controller does not send the mask of the CR")
	}
}

// A kind without the key keeps its const mask, so released kinds do not change.
func TestUpdateMaskWithoutNestedStaysAConst(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files := checkLifecycleFixture(t, "PatchThing", string(contract), "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\n")
	if !strings.Contains(string(files["api/coralogix/v1alpha1/patchthing_convert.go"]), `const PatchThingUpdateMask = "name,description,enabled,target"`) {
		t.Fatal("the PatchThing mask changed")
	}
}

// A mask in the body takes the mask of the CR too.
func TestNestedUpdateMaskInBody(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := replaceOnce(t, string(contract), "      parameters:\n        - {name: updateMask, in: query, required: true, schema: {type: string, pattern: '^[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*(,[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*)*$'}}\n", "")
	text = replaceOnce(t, text, "title: UpdateMaskThingRequest\n              type: object\n              properties:\n", "title: UpdateMaskThingRequest\n              type: object\n              properties:\n                updateMask: {type: string, pattern: '^[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*(,[a-z][A-Za-z0-9]*(\\.[a-z][A-Za-z0-9]*)*)*$'}\n")
	files := checkLifecycleFixture(t, "MaskThing", text, nestedMaskOverride+"  updateMaskInBody: true\n")
	if !strings.Contains(string(files["api/coralogix/v1alpha1/maskthing_convert.go"]), "out.UpdateMask = ptr.To(in.UpdateMask())") {
		t.Fatal("the Update body does not hold the mask of the CR")
	}
}

func TestNestedUpdateMaskChecks(t *testing.T) {
	const patchHead = "resource: PatchThing\nmode: existing\nvalidators: {inferred: true}\n"
	for _, tt := range []struct {
		name, resource, override string
		edit                     func(string) string
		code                     string
	}{
		{name: "an object that is only a oneOf", resource: "PatchThing", override: patchHead + "api:\n  updateMask:\n    nested: [target]\n", code: codeShapeUnsupported},
		{
			name: "an optional object", resource: "MaskThing", override: nestedMaskOverride, code: codeShapeUnsupported,
			edit: func(s string) string {
				s = strings.Replace(s, "              title: CreateMaskThingRequest\n              type: object\n              required: [name, rule]", "              title: CreateMaskThingRequest\n              type: object\n              required: [name]", 1)
				return strings.Replace(s, "    MaskThing:\n      type: object\n      required: [id, name, rule]", "    MaskThing:\n      type: object\n      required: [id, name]", 1)
			},
		},
		{name: "a field that is not an object", resource: "MaskThing", override: strings.Replace(nestedMaskOverride, "[rule]", "[name]", 1), code: codeShapeUnsupported},
		{name: "a field that Update does not have", resource: "MaskThing", override: strings.Replace(nestedMaskOverride, "[rule]", "[missing]", 1), code: codeShapeUnsupported},
		{name: "a full-replace Update", resource: "Thing", override: "resource: Thing\nmode: existing\nvalidators: {inferred: true}\napi:\n  updateMask:\n    nested: [config]\n", code: codeShapeUnsupported},
		{
			name: "a field with requestValueBy", resource: "MaskThing", code: codeShapeUnsupported,
			override: nestedMaskOverride + "types:\n  MaskThingRule:\n    fields:\n      note: {requestValueBy: {some: SOME, prefix: PREFIX}}\n",
		},
		{
			name: "a oneOf that permits no arm", resource: "MaskThing", override: nestedMaskOverride, code: codeShapeUnsupported,
			edit: func(s string) string {
				return strings.Replace(s, "      oneOf:\n        - required: [some]\n        - required: [prefix]\n    PatchThingKind:", "      oneOf:\n        - required: [some]\n        - required: [prefix]\n        - not:\n            anyOf:\n              - required: [some]\n              - required: [prefix]\n    PatchThingKind:", 1)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, checkMask(t, tt.resource, tt.override, tt.edit), tt.code)
		})
	}
	for _, tt := range []struct{ name, override, want string }{
		{name: "no field", override: strings.Replace(nestedMaskOverride, "    nested: [rule]\n", "    nested: []\n", 1), want: "name an object field in nested"},
		{name: "a dotted path", override: strings.Replace(nestedMaskOverride, "[rule]", "[rule.some]", 1), want: "must be a top-level field"},
		{name: "a field listed twice", override: strings.Replace(nestedMaskOverride, "[rule]", "[rule, rule]", 1), want: "is listed twice"},
		{name: "a new kind", override: strings.Replace(nestedMaskOverride, "mode: existing\nvalidators: {inferred: true}\n", "mode: new\n", 1), want: "mode new sets no api key"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkMask(t, "MaskThing", tt.override, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want an error with %q, got %v", tt.want, err)
			}
		})
	}
	t.Run("the golden override", func(t *testing.T) {
		if err := checkMask(t, "MaskThing", nestedMaskOverride, nil); err != nil {
			t.Fatal(err)
		}
	})
}
