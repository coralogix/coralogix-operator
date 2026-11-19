package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const optionalGroupOverride = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n" +
	"groups:\n  extras: {fields: [description, quota], optional: true}\n" +
	"exactlyOneOf:\n  - [extras, settings]\n"

// An optional group is a pointer that a CR can leave out, and a request skips the fields of a
// missing group. exactlyOneOf makes the group and another field a choice: one CEL rule, and a sample
// and an e2e sweep that send each side.
func TestOptionalGroupAndExactlyOneOf(t *testing.T) {
	root, input := syntheticRoot(t)
	kindDir := filepath.Join(root, kindsDir, "thing")
	if err := os.WriteFile(filepath.Join(kindDir, "behavior-overrides.yaml"), []byte(optionalGroupOverride), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	read := func(path ...string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(path...))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	types := read(root, apiDir, "thing_types.go")
	for _, want := range []string{
		"Extras *ThingSpecExtras `json:\"extras,omitempty\"`",
		`Exactly one of extras, settings must be set`,
	} {
		if !strings.Contains(types, want) {
			t.Errorf("thing_types.go omits %q", want)
		}
	}
	convert := read(root, apiDir, "thing_convert.go")
	for _, want := range []string{"if in.Extras != nil {", "in.Extras.Quota"} {
		if !strings.Contains(convert, want) {
			t.Errorf("thing_convert.go omits %q", want)
		}
	}
	// The sweep sends each side of the choice, and no spec sends both.
	e2e := read(root, "tests", "e2e", "thing_generated_test.go")
	for _, want := range []string{`"extras": {`, `"settings": {`} {
		if !strings.Contains(e2e, want) {
			t.Errorf("the e2e test never sends %s", want)
		}
	}
}

func TestExactlyOneOfChecks(t *testing.T) {
	const head = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"
	for _, tt := range []struct {
		name, override, code string
	}{
		{name: "an optional group holds a required field", override: head + "groups:\n  extras: {fields: [name, description], optional: true}\n", code: codeShapeUnsupported},
		{name: "a required field is not an arm", override: head + "exactlyOneOf:\n  - [name, settings]\n", code: codeOverrideInvalid},
		{name: "an unknown field", override: head + "exactlyOneOf:\n  - [missing, settings]\n", code: codeOverrideUnused},
		{name: "an arm of another oneOf", override: head + "types:\n  ThingConfig:\n    exactlyOneOf:\n      - [http, queue]\n", code: codeOverrideInvalid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, checkG14(t, "Thing", tt.override), tt.code)
		})
	}
	t.Run("one name is not a choice", func(t *testing.T) {
		if err := checkG14(t, "Thing", head+"exactlyOneOf:\n  - [settings]\n"); err == nil {
			t.Fatal("a set of one field passed")
		}
	})
	t.Run("a valid set", func(t *testing.T) {
		if err := checkG14(t, "Thing", optionalGroupOverride); err != nil {
			t.Fatal(err)
		}
	})
}

// A field of a required group is read straight from the spec. A field of an optional group is read
// only when the group is set, so sourceField names the group to test.
func TestSourceFieldGuardsAnOptionalGroup(t *testing.T) {
	leaf := &crField{API: "quota", JSON: "quota", GoName: "Quota"}
	for _, optional := range []bool{false, true} {
		group := &crField{API: "extras", JSON: "extras", GoName: "Extras", group: true, Optional: optional, object: &crType{Fields: []*crField{leaf}}}
		field, src, guard := sourceField(&crType{Fields: []*crField{group}}, "quota")
		wantGuard := ""
		if optional {
			wantGuard = "in.Extras"
		}
		if field != leaf || src != "in.Extras.Quota" || guard != wantGuard {
			t.Errorf("optional=%v: got %v, %q, %q", optional, field, src, guard)
		}
	}
}
