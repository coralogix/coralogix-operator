package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// A description line replaces the contract description of the CR field, in the Go type and in the CRD.
func TestDescriptionOverrideReplacesTheContractText(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeOverrides(t, root, sendNullOverrides("      description: {description: The released text of the field.}\n"))
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	field := types[:strings.Index(types, "\tDescription *string")]
	field = field[strings.LastIndex(field, "\n\n"):]
	if !strings.Contains(field, "// The released text of the field.") {
		t.Errorf("the field keeps the contract description:\n%s", field)
	}
	if strings.Count(field, "//") != 2 {
		t.Errorf("the field has more doc than the override and +optional:\n%s", field)
	}
}

func TestDescriptionOverrideOnAnUnknownFieldIsRejected(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeOverrides(t, root, sendNullOverrides("      missing: {description: text}\n"))
	if err := generateFromInput(input, root, root); err == nil {
		t.Error("a description of an unknown property must be rejected")
	}
}

func TestDescriptionOverrideParsing(t *testing.T) {
	for _, tc := range []struct {
		name, mode, line string
		valid            bool
	}{
		{"existing kind", "existing", "description: {description: text}", true},
		{"with a name", "existing", "description: {description: text, name: summary}", true},
		{"a new kind", "new", "description: {description: text}", false},
		{"with skip", "existing", "description: {description: text, skip: true}", false},
		{"blank", "existing", "description: {description: ' '}", false},
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
