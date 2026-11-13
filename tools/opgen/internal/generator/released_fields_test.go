package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// An optional line on a scalar that a request requires keeps the value type of the SDK field. The
// CR field is a pointer, and the request sets the value only when the CR has one.
func TestOptionalLineKeepsTheSDKValueType(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  Thing:\n    fields:\n      name: {optional: true}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if !strings.Contains(types, "Name *string `json:\"name,omitempty\"`") {
		t.Errorf("the CR field is not an optional pointer:\n%s", types)
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	for _, request := range []string{"toCreateThingRequest", "toReplaceThingRequest"} {
		code := convert[strings.Index(convert, "func (in *ThingSpec) "+request):]
		if end := strings.Index(code[1:], "\nfunc "); end > 0 {
			code = code[:end+1]
		}
		if !strings.Contains(code, "if in.Name != nil {\n\t\tout.Name = (*in.Name)\n\t}") {
			t.Errorf("%s does not set the SDK value from the pointer:\n%s", request, code)
		}
	}
}

// controller-gen reads an unquoted value that starts with a digit as a number, so the Enum marker
// quotes every value of such a list. A list of plain names keeps the short form.
func TestEnumMarkerQuotesValuesThatStartWithADigit(t *testing.T) {
	for _, tc := range []struct {
		names []string
		want  string
	}{
		{[]string{"unspecified", "low", "pagerDutyIncidents"}, "unspecified;low;pagerDutyIncidents"},
		{[]string{"unspecified", "1m", "5m"}, `{"unspecified","1m","5m"}`},
		{[]string{"7d", "v2"}, `{"7d","v2"}`},
		{[]string{"true", "maybe"}, `{"true","maybe"}`},
		{[]string{"a-b", "c"}, `{"a-b","c"}`},
	} {
		e := &enumMap{}
		for _, name := range tc.names {
			e.Values = append(e.Values, enumValue{Name: name})
		}
		if got := e.Names(); got != tc.want {
			t.Errorf("Names(%v) = %s, want %s", tc.names, got, tc.want)
		}
	}

	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "enums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    names: {THING_PRIORITY_LOW: 1m, THING_PRIORITY_PAGER_DUTY_INCIDENTS: 5m}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	if types := readGenerated(t, root, apiDir+"/thing_types.go"); !strings.Contains(types, `+kubebuilder:validation:Enum={"unspecified","1m","5m"}`) {
		t.Errorf("the Enum marker is not quoted:\n%s", types)
	}
}

const ignoredLine = "types:\n  HttpThingConfig:\n    ignored:\n      legacyHeaders:\n        type: '[]string'\n        doc: |-\n          LegacyHeaders has no effect.\n          Deprecated: ignored by the operator and by the API.\n"

// An ignored field is in the CR type, with its doc, and no request sends it.
func TestIgnoredFieldIsInTheCRAndNotInRequests(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, ignoredLine)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	want := "\t// LegacyHeaders has no effect.\n\t// Deprecated: ignored by the operator and by the API.\n\t// +optional\n\tLegacyHeaders []string `json:\"legacyHeaders,omitempty\"`"
	if !strings.Contains(types, want) {
		t.Errorf("the CR type has no ignored field:\n%s", types)
	}
	if convert := readGenerated(t, root, apiDir+"/thing_convert.go"); strings.Contains(convert, "LegacyHeaders") {
		t.Errorf("a request sends the ignored field:\n%s", convert)
	}
	if sample := readGenerated(t, root, "config/samples/v1alpha1/generated/thing.yaml"); strings.Contains(sample, "legacyHeaders") {
		t.Errorf("the sample sets the ignored field:\n%s", sample)
	}
}

func TestIgnoredFieldOfAComponentThatTheCRDoesNotHaveIsUnused(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, strings.Replace(ignoredLine, "HttpThingConfig", "PatchThingSome", 1))
	wantCode(t, generateFromInput(input, root, root), codeOverrideUnused)
}

func TestIgnoredFieldParsing(t *testing.T) {
	for _, tc := range []struct {
		name, mode, body string
		valid            bool
	}{
		{"an existing kind", "existing", "legacy: {type: '[]string', doc: Has no effect.}", true},
		{"a string", "existing", "legacy: {type: string, doc: Has no effect.}", true},
		{"a bool", "existing", "legacy: {type: bool, doc: Has no effect.}", true},
		{"a new kind", "new", "legacy: {type: string, doc: Has no effect.}", false},
		{"an object type", "existing", "legacy: {type: 'map[string]string', doc: Has no effect.}", false},
		{"no doc", "existing", "legacy: {type: string}", false},
		{"a bad name", "existing", "'legacy.headers': {type: string, doc: Has no effect.}", false},
		{"a field line of the same name", "existing", "endpoint: {type: string, doc: Has no effect.}\n    fields:\n      endpoint: {required: true}", false},
	} {
		data := "resource: Thing\nmode: " + tc.mode + "\n"
		if tc.mode == "existing" {
			data += "validators: {inferred: true}\n"
		}
		data += "types:\n  HttpThingConfig:\n    ignored:\n      " + tc.body + "\n"
		if _, err := overrides.Parse([]byte(data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}

// fallbackAPI adds a fallback ThingItem beside the items list, so one component is at two CR
// paths.
func fallbackAPI(api string) string {
	api = strings.Replace(api, "        items:\n          type: array\n          items: {$ref: '#/components/schemas/ThingItem'}\n          description: Ordered items.\n",
		"        items:\n          type: array\n          items: {$ref: '#/components/schemas/ThingItem'}\n          description: Ordered items.\n        fallback: {$ref: '#/components/schemas/ThingItem'}\n", 1)
	return strings.ReplaceAll(api, "                items:\n                  type: array\n                  items: {$ref: '#/components/schemas/ThingItem'}\n",
		"                items:\n                  type: array\n                  items: {$ref: '#/components/schemas/ThingItem'}\n                fallback: {$ref: '#/components/schemas/ThingItem'}\n")
}

func fallbackSDK(sdk string) string {
	return strings.ReplaceAll(sdk, "\tItems        []ThingItem", "\tFallback     *ThingItem `json:\"fallback,omitempty\"`\n\tItems        []ThingItem")
}

// One component at two CR paths with the same validators lines is one struct. Different lines
// make a struct per path.
func TestSameValidationAtTwoPathsSharesOneStruct(t *testing.T) {
	for _, tc := range []struct {
		name, validators string
		split            bool
	}{
		{"the same lines", "    spec.items[].name: {inferred: false}\n    spec.fallback.name: {inferred: false}\n", false},
		{"a line at one path", "    spec.items[].name: {inferred: false}\n", true},
		{"different lines", "    spec.items[].name: {inferred: false}\n    spec.fallback.weight: {inferred: false}\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, input := skipRoot(t, fallbackAPI, fallbackSDK)
			writeOverrides(t, root, "resource: Thing\nmode: existing\nvalidators:\n  inferred: true\n  fields:\n"+tc.validators)
			if err := generateFromInput(input, root, root); err != nil {
				t.Fatal(err)
			}
			types := readGenerated(t, root, apiDir+"/thing_types.go")
			if split := strings.Contains(types, "type ThingItemAtSpecFallback"); split != tc.split {
				t.Errorf("a struct per path = %t, want %t:\n%s", split, tc.split, types)
			}
			if !tc.split {
				item := types[strings.Index(types, "type ThingItem struct"):]
				item = item[:strings.Index(item, "\n}\n")]
				if strings.Contains(item, "MinLength=1") || !strings.Contains(types, "Fallback *ThingItem `") {
					t.Errorf("the fallback does not use the shared struct without the dropped limit:\n%s", types)
				}
			}
		})
	}
}

// A minimal line can name a required group with an object of its optional fields, for a field that
// the live API requires and the released CR keeps optional inside a group.
func TestMinimalSetsAnOptionalFieldOfARequiredGroup(t *testing.T) {
	for _, tc := range []struct {
		name, minimal string
		valid         bool
	}{
		{"an optional field of the group", "minimal:\n  options: {description: kept}\n", true},
		{"a field that is not in the group", "minimal:\n  options: {quota: '5'}\n", false},
		{"an empty object", "minimal:\n  options: {}\n", false},
		{"a value that is not an object", "minimal:\n  options: kept\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, input := skipRoot(t, nil, nil)
			writeOverrides(t, root, "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ngroups:\n  options: {fields: [description]}\n")
			e2e := "values:\n  destinations: [https://example.com]\n" + tc.minimal
			if err := os.WriteFile(filepath.Join(root, kindsDir, "thing", e2eFileName), []byte(e2e), 0o644); err != nil {
				t.Fatal(err)
			}
			err := generateFromInput(input, root, root)
			if !tc.valid {
				wantCode(t, err, codeE2EInvalid)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			test := readGenerated(t, root, "tests/e2e/thing_generated_test.go")
			minimal := test[strings.Index(test, "thingMinimalSpec = `"):]
			minimal = minimal[:strings.Index(minimal, "}`")]
			if !strings.Contains(minimal, `"description": "kept"`) {
				t.Errorf("the minimal spec has no group field:\n%s", minimal)
			}
		})
	}
}
