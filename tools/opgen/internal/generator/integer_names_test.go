package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func integerNamesOverrides(line string) string {
	return "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingSettings:\n    fields:\n      retries: " + line + "\n"
}

const retriesNames = "{integerNames: [{name: Never, value: 0}, {name: Once, value: 1}, {name: Twice, value: 2}]}"

// The CR names the integers of the field. The CRD accepts only the names, and Create and Update send
// the integer of the name.
func TestIntegerNamesShowAnIntegerAsNames(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeOverrides(t, root, integerNamesOverrides(retriesNames))
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	for _, want := range []string{
		"// +kubebuilder:validation:Enum=Never;Once;Twice\n",
		"// The API holds each name as an integer: Never=0, Once=1, Twice=2.\n",
		"Retries *string `json:\"retries,omitempty\"`",
	} {
		if !strings.Contains(types, want) {
			t.Errorf("the types have no %q", want)
		}
	}
	// The integer limits of the contract do not apply to a name.
	if strings.Contains(types, "Maximum=10") {
		t.Error("the name field keeps the integer limits")
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	for _, want := range []string{
		"map[string]int32{\n\t\"Never\": 0,\n\t\"Once\":  1,\n\t\"Twice\": 2,\n}",
		"return 0, fmt.Errorf(\"unknown value %q\", value)",
		"err := thingSettingsRetriesToAPI(*in.Retries)",
	} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion has no %q", want)
		}
	}
	// The sample sets a name, never an integer.
	if sample := readGenerated(t, root, "config/samples/v1alpha1/generated/thing.yaml"); !strings.Contains(sample, "retries: Never") {
		t.Errorf("the sample does not name the value:\n%s", sample)
	}
}

func TestIntegerNamesRejectOtherShapes(t *testing.T) {
	for name, tc := range map[string]struct{ overrides, code string }{
		"a string field": {
			"resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n      description: " + retriesNames + "\n",
			codeOverrideInvalid,
		},
		"a value above the API maximum": {integerNamesOverrides("{integerNames: [{name: Many, value: 11}]}"), codeOverrideInvalid},
		"a value below the API minimum": {integerNamesOverrides("{integerNames: [{name: Less, value: -1}]}"), codeOverrideInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			root, input := skipRoot(t, nil, nil)
			writeOverrides(t, root, tc.overrides)
			wantCode(t, generateFromInput(input, root, root), tc.code)
		})
	}
}

func TestIntegerNamesParsing(t *testing.T) {
	for _, tc := range []struct {
		name, mode, line string
		valid            bool
	}{
		{"existing kind", "existing", retriesNames, true},
		{"a new kind", "new", retriesNames, false},
		{"a name listed twice", "existing", "{integerNames: [{name: Once, value: 1}, {name: Once, value: 2}]}", false},
		{"a value listed twice", "existing", "{integerNames: [{name: Once, value: 1}, {name: Again, value: 1}]}", false},
		{"a name that starts with a digit", "existing", "{integerNames: [{name: 1st, value: 1}]}", false},
		{"a name with a space", "existing", "{integerNames: [{name: 'Once more', value: 1}]}", false},
		{"a boolean name", "existing", "{integerNames: [{name: 'true', value: 1}]}", false},
		{"with skip", "existing", "{skip: true, integerNames: [{name: Once, value: 1}]}", false},
		{"with ref", "existing", "{ref: {kind: Thing}, integerNames: [{name: Once, value: 1}]}", false},
		{"with a name and required", "existing", "{name: tries, required: true, integerNames: [{name: Once, value: 1}]}", true},
	} {
		data := integerNamesOverrides(tc.line)
		if tc.mode == "new" {
			data = "resource: Thing\nmode: new\ntypes:\n  ThingSettings:\n    fields:\n      retries: " + tc.line + "\n"
		}
		if _, err := overrides.Parse([]byte(data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}

// A fixture of another API version, such as the v1beta1 Alert, uses the package of that version.
func TestE2EFixtureOfAnotherVersion(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "values:\n  destinations: [https://example.com]\n"+
		"fixtures:\n  - {kind: Alert, version: v1beta1, suffix: -alert, statusField: ID, spec: {name: '@{run}-alert'}}\n"+
		"  - {kind: Thing, suffix: -thing, statusField: ID, spec: {name: '@{run}-thing'}}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	test := readGenerated(t, root, "tests/e2e/thing_generated_test.go")
	for _, want := range []string{
		"\tcoralogixv1beta1 \"" + operatorModule + "/api/coralogix/v1beta1\"\n",
		"fixture := &coralogixv1beta1.Alert{",
		"current := &coralogixv1beta1.Alert{}",
		"fixture := &coralogixv1alpha1.Thing{",
	} {
		if !strings.Contains(test, want) {
			t.Errorf("the e2e test has no %q", want)
		}
	}
	if strings.Count(test, "api/coralogix/v1beta1\"") != 1 {
		t.Error("the v1beta1 package is not imported once")
	}

	root, input = syntheticRoot(t)
	writeThingE2E(t, root, "values:\n  destinations: [https://example.com]\n"+
		"fixtures:\n  - {kind: Alert, version: beta, suffix: -alert, statusField: ID, spec: {name: '@{run}-alert'}}\n")
	wantCode(t, generateFromInput(input, root, root), codeE2EInvalid)
}

// A reference with a version names a handwritten kind of that API version.
func TestRefKindOfAnotherVersion(t *testing.T) {
	root := t.TempDir()
	file, err := overrides.Parse([]byte("resource: BodyIDThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  BodyIDThing:\n    fields:\n      alertIds: {ref: {kind: Alert, version: v1beta1}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	wantShapeCode(t, refKindIssues(root, nil, file), codeOverrideInvalid)
	// An Alert types file of v1alpha1 is not the Alert of v1beta1.
	writeTypes := func(version string) {
		dir := filepath.Join(root, filepath.Dir(apiDir), version)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "alert_types.go"), []byte("package "+version+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTypes("v1alpha1")
	wantShapeCode(t, refKindIssues(root, nil, file), codeOverrideInvalid)
	writeTypes("v1beta1")
	if err := refKindIssues(root, nil, file); err != nil {
		t.Fatal(err)
	}

	// A listed kind has no types file before the run writes it. An explicit v1alpha1 reference to
	// it passes, as one without a version does; a v1beta1 reference to it does not.
	listed := []kindEntry{{Kind: "Folder", Resource: "Folder"}}
	for version, ok := range map[string]bool{"": true, "v1alpha1": true, "v1beta1": false} {
		ref := "{kind: Folder}"
		if version != "" {
			ref = "{kind: Folder, version: " + version + "}"
		}
		file, err := overrides.Parse([]byte("resource: BodyIDThing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  BodyIDThing:\n    fields:\n      alertIds: {ref: " + ref + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		err = refKindIssues(root, listed, file)
		if ok && err != nil {
			t.Errorf("version %q: %v", version, err)
		}
		if !ok {
			wantShapeCode(t, err, codeOverrideInvalid)
		}
	}
}

// An API default of an integerNames field is written as its name, because the CR holds names.
func TestIntegerNamesDefault(t *testing.T) {
	// The contract declares retries in the resource and in both requests, with the same default.
	withDefault := func(value string) func(string) string {
		return func(api string) string {
			const field = "retries: {type: integer, format: int32, minimum: 0, maximum: 10,"
			if strings.Count(api, field) != 3 {
				t.Fatalf("the synthetic contract has %d retries fields, want 3", strings.Count(api, field))
			}
			return strings.ReplaceAll(api, field, field+" default: "+value+",")
		}
	}
	root, input := skipRoot(t, withDefault("1"), nil)
	writeOverrides(t, root, integerNamesOverrides(retriesNames))
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if !strings.Contains(types, "// +kubebuilder:default="+markerString("Once")+"\n") || strings.Contains(types, "+kubebuilder:default=1\n") {
		t.Errorf("the default is not the name of the integer:\n%s", types)
	}

	// A default with no name has no CR spelling.
	root, input = skipRoot(t, withDefault("5"), nil)
	writeOverrides(t, root, integerNamesOverrides(retriesNames))
	wantCode(t, generateFromInput(input, root, root), codeShapeUnsupported)
}
