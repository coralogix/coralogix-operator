package generator

import (
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// numericStringRoot adds a text field that holds an unsigned decimal number, as the API of a limit
// sends it, to the settings of the synthetic Thing.
func numericStringRoot(t *testing.T, pattern string) (string, func() error) {
	t.Helper()
	field := "cap: {type: string, minLength: 1, maxLength: 20, pattern: '" + pattern + "', default: '0', x-coralogix-presence: true, description: The cap.}"
	root, input := skipRoot(t, func(api string) string {
		for _, schema := range []string{"ThingSettings", "ThingSettingsCreate", "ThingSettingsUpdate"} {
			api = strings.Replace(api, "    "+schema+":\n      type: object\n", "    "+schema+":\n      type: object\n", 1)
			i := strings.Index(api, "    "+schema+":\n")
			j := strings.Index(api[i:], "      properties:\n") + i + len("      properties:\n")
			api = api[:j] + "        " + field + "\n" + api[j:]
		}
		return api
	}, func(sdk string) string {
		sdk = strings.Replace(sdk, "type ThingSettings struct {\n", "type ThingSettings struct {\n\tCap *string `json:\"cap,omitempty\"`\n", 1)
		sdk = strings.Replace(sdk, "type ThingSettingsCreate struct {\n", "type ThingSettingsCreate struct {\n\tCap *string `json:\"cap,omitempty\"`\n", 1)
		return strings.Replace(sdk, "type ThingSettingsUpdate struct {\n", "type ThingSettingsUpdate struct {\n\tCap *string `json:\"cap,omitempty\"`\n", 1)
	})
	return root, func() error { return generateFromInput(input, root, root) }
}

func numericStringOverrides(line string) string {
	return "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingSettings:\n    fields:\n      cap: " + line + "\n"
}

// The CR shows the text as an int64, and Create and Update send its decimal text. The limits of the
// text stay out of the integer.
func TestNumericStringShowsATextNumberAsAnInteger(t *testing.T) {
	root, generate := numericStringRoot(t, "^[0-9]+$")
	writeOverrides(t, root, numericStringOverrides("{numericString: true}"))
	if err := generate(); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if !strings.Contains(types, "Cap *int64 `json:\"cap,omitempty\"`") {
		t.Errorf("the cap is not an optional int64:\n%s", types)
	}
	if strings.Contains(types, "MaxLength=20") || strings.Contains(types, "validation:Pattern") {
		t.Error("the integer keeps the limits of the text")
	}
	if !strings.Contains(types, "+kubebuilder:default=0\n") {
		t.Error("the default of the cap is not the integer 0")
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	for _, want := range []string{"if in.Cap != nil {", "strconv.FormatInt(*in.Cap, 10)"} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion has no %q:\n%s", want, convert)
		}
	}
}

func TestNumericStringRejectsOtherShapes(t *testing.T) {
	for name, tc := range map[string]struct{ overrides, pattern string }{
		"a text with another pattern": {numericStringOverrides("{numericString: true}"), "^[a-z]+$"},
		"an integer field": {
			"resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingSettings:\n    fields:\n      retries: {numericString: true}\n",
			"^[0-9]+$",
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, generate := numericStringRoot(t, tc.pattern)
			writeOverrides(t, root, tc.overrides)
			wantCode(t, generate(), codeOverrideInvalid)
		})
	}
}

func TestNumericStringParsing(t *testing.T) {
	for _, tc := range []struct {
		name, mode, line string
		valid            bool
	}{
		{"existing kind", "existing", "{numericString: true}", true},
		{"a new kind", "new", "{numericString: true}", false},
		{"with a released name and optional", "existing", "{numericString: true, name: limit, optional: true}", true},
		{"with skip", "existing", "{skip: true, numericString: true}", false},
		{"with integerNames", "existing", "{numericString: true, integerNames: [{name: Once, value: 1}]}", false},
		{"with joinedBy", "existing", "{numericString: true, joinedBy: ','}", false},
		{"with secretRef", "existing", "{numericString: true, secretRef: true}", false},
	} {
		data := numericStringOverrides(tc.line)
		if tc.mode == "new" {
			data = "resource: Thing\nmode: new\ntypes:\n  ThingSettings:\n    fields:\n      cap: " + tc.line + "\n"
		}
		if _, err := overrides.Parse([]byte(data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}
