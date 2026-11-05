package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func sendNullOverrides(lines string) string {
	return "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n" + lines
}

// An unset optional string is sent as "", and an unset optional list as [], in Create and Update.
func TestSendNullAsEmptySendsAnEmptyValue(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeOverrides(t, root, sendNullOverrides("      description: {sendNullAsEmpty: true}\n      items: {sendNullAsEmpty: true}\n"))
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	create := convert[strings.Index(convert, "func (in *ThingSpec) toCreateThingRequest"):strings.Index(convert, "func (in *ThingSpec) toReplaceThingRequest")]
	update := convert[strings.Index(convert, "func (in *ThingSpec) toReplaceThingRequest"):]
	for name, code := range map[string]string{"Create": create, "Update": update} {
		for _, want := range []string{"if in.Description != nil {", "[]thingsservice.ThingItem{}"} {
			if !strings.Contains(code, want) {
				t.Errorf("%s: missing %q", name, want)
			}
		}
		if !regexp.MustCompile(`v\d+ := ""\n\s*out\.Description = &v\d+`).MatchString(code) {
			t.Errorf("%s: an unset description is not sent as an empty string:\n%s", name, code)
		}
		if strings.Count(code, "} else {") < 2 {
			t.Errorf("%s: the else branches are missing", name)
		}
	}
	// A field without the key keeps the released rule: nothing is sent for an unset value.
	if strings.Contains(create[strings.Index(create, "out.Quota"):], ` := ""`) {
		t.Error("a field without the key sends an empty value")
	}
}

func TestSendNullAsEmptyRejectsOtherShapes(t *testing.T) {
	for name, line := range map[string]string{
		"a required field":    "      name: {sendNullAsEmpty: true}\n",
		"a bool":              "      enabled: {sendNullAsEmpty: true}\n",
		"an object":           "      settings: {sendNullAsEmpty: true}\n",
		"a map":               "      labels: {sendNullAsEmpty: true}\n",
		"a quantity":          "      ratio: {sendNullAsEmpty: true}\n",
		"a generated date":    "      createTime: {sendNullAsEmpty: true}\n",
		"an unknown property": "      missing: {sendNullAsEmpty: true}\n",
	} {
		root, input := skipRoot(t, nil, nil)
		writeOverrides(t, root, sendNullOverrides(line))
		if err := generateFromInput(input, root, root); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	root, input := skipRoot(t, nil, nil)
	writeOverrides(t, root, sendNullOverrides("      name: {sendNullAsEmpty: true}\n"))
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}

func TestSendNullAsEmptyParsing(t *testing.T) {
	for _, tc := range []struct {
		name, mode, line string
		valid            bool
	}{
		{"existing kind", "existing", "description: {sendNullAsEmpty: true}", true},
		{"a new kind", "new", "description: {sendNullAsEmpty: true}", false},
		{"with skip", "existing", "description: {sendNullAsEmpty: true, skip: true}", false},
		{"with createAs", "existing", "description: {sendNullAsEmpty: true, createAs: {a: b}}", false},
		{"with optional", "existing", "description: {sendNullAsEmpty: true, optional: true}", true},
	} {
		data := "resource: Thing\nmode: " + tc.mode + "\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n      " + tc.line + "\n"
		if tc.mode == "new" {
			data = "resource: Thing\nmode: new\ntypes:\n  Thing:\n    fields:\n      " + tc.line + "\n"
		}
		if _, err := overrides.Parse([]byte(data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}
