package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func syntheticContract(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReleasedFieldPresenceAndName(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      name: {name: title}\n      limit: {required: true}\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/bodyidthing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/bodyidthing_convert.go"])
	for _, want := range []string{"Title string `json:\"title\"`", "Limit int32 `json:\"limit\"`"} {
		if !strings.Contains(types, want) {
			t.Errorf("types lack %q", want)
		}
	}
	if strings.Count(convert, "in.Title") != 2 || strings.Count(convert, "out.Name =") != 2 || strings.Count(convert, "out.Limit =") != 2 {
		t.Error("both request conversions must use the API properties")
	}
	if !strings.Contains(string(files["config/samples/v1alpha1/generated/bodyidthing.yaml"]), "title:") {
		t.Error("sample must use the CR property")
	}
}

func TestNestedReleasedFieldPresenceAndName(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingItem:\n    fields:\n      name: {name: label}\n      weight: {required: true}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "Label string `json:\"label\"`") || !strings.Contains(types, "Weight int32 `json:\"weight\"`") {
		t.Error("nested CR fields must keep their released names and presence")
	}
	if !strings.Contains(convert, "in.Label") || !strings.Contains(convert, "out.Name =") || !strings.Contains(convert, "out.Weight =") {
		t.Error("nested request conversions must use API properties")
	}
}

func TestReleasedEnumValues(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW]\n    rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    names: {THING_PRIORITY_LOW: alerts}\n    aliases: {unspecified: alerts}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	convert := string(files["api/coralogix/v1alpha1/thing_convert.go"])
	if !strings.Contains(types, "+kubebuilder:validation:Enum=unspecified;alerts") {
		t.Error("CR enum marker has the wrong values")
	}
	if !strings.Contains(convert, `"unspecified": thingsservice.THINGPRIORITY_THING_PRIORITY_LOW`) ||
		!strings.Contains(convert, `"alerts":      thingsservice.THINGPRIORITY_THING_PRIORITY_LOW`) {
		t.Error("alias and accepted value must map to the checked SDK constant")
	}
}

func TestReleasedRequiredAndNameLinesMustChangeOutput(t *testing.T) {
	for _, line := range []string{"name: {required: true}", "name: {name: name}", "limit: {optional: true}"} {
		t.Run(line, func(t *testing.T) {
			file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      " + line + "\n"
			dir := t.TempDir()
			openAPI := filepath.Join(dir, "openapi.yaml")
			override := filepath.Join(dir, "behavior-overrides.yaml")
			if err := os.WriteFile(openAPI, []byte(syntheticContract(t)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: override}), codeOverrideUnused)
		})
	}
}

func TestReleasedOptionalField(t *testing.T) {
	file := "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\ntypes:\n  BodyIDThing:\n    fields:\n      name: {optional: true}\n"
	files := checkLifecycleFixture(t, "BodyIDThing", syntheticContract(t), file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/bodyidthing_types.go"]), "Name *string `json:\"name,omitempty\"`") {
		t.Error("released name must be optional in the CR")
	}
}

func TestReleasedEnumVerbatimTrimmedAndBusinessFirst(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    trimPrefix: THING_PRIORITY_\n    verbatim: true\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	types := string(files["api/coralogix/v1alpha1/thing_types.go"])
	if !strings.Contains(types, "Enum=unspecified;LOW;PAGER_DUTY_INCIDENTS") {
		t.Error("verbatim and trimPrefix must set the CR enum values")
	}
	contract := strings.Replace(syntheticContract(t), "enum: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", "enum: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", 1)
	file = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    verbatim: true\n"
	files = checkLifecycleFixture(t, "Thing", contract, file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/thing_types.go"]), "Enum=THING_PRIORITY_LOW;THING_PRIORITY_PAGER_DUTY_INCIDENTS") {
		t.Error("a business first value must stay accepted")
	}
}

func TestReleasedEnumNameKeepsOtherDefaultNames(t *testing.T) {
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    zero: unspecified\n    values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]\n    names: {THING_PRIORITY_LOW: alerts}\n"
	files := checkLifecycleFixture(t, "Thing", syntheticContract(t), file)
	if !strings.Contains(string(files["api/coralogix/v1alpha1/thing_types.go"]), "Enum=unspecified;alerts;pagerDutyIncidents") {
		t.Error("the name override must keep the other CR spelling")
	}
}

func TestCompatibilityLinesRejectInvalidValues(t *testing.T) {
	cases := []struct {
		resource, fields string
		code             string
	}{
		{"BodyIDThing", "types:\n  BodyIDThing:\n    fields:\n      name: {name: limit}\n", codeNameCollision},
		{"BodyIDThing", "types:\n  BodyIDThing:\n    fields:\n      name: {required: true, optional: true}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [NOT_IN_API]}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_PAGER_DUTY_INCIDENTS]}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], names: {THING_PRIORITY_LOW: low}}\n", codeOverrideInvalid},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS], names: {THING_PRIORITY_LOW: same, THING_PRIORITY_PAGER_DUTY_INCIDENTS: same}}\n", codeNameCollision},
		{"Thing", "enums:\n  ThingPriority: {zero: unspecified, values: [THING_PRIORITY_LOW], rejected: [THING_PRIORITY_PAGER_DUTY_INCIDENTS], aliases: {unspecified: missing}}\n", codeOverrideInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.resource+tc.fields, func(t *testing.T) {
			head := "resource: " + tc.resource + "\nmode: existing\nvalidators: {inferred: true}\n"
			if tc.resource == "BodyIDThing" {
				head = "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\n"
			}
			dir := t.TempDir()
			openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
			if err := os.WriteFile(openAPI, []byte(syntheticContract(t)), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(head+tc.fields), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode(t, Check(CheckOptions{Resource: tc.resource, OpenAPIPath: openAPI, OverridesPath: override}), tc.code)
		})
	}
}

func TestEnumLineWithoutOutputChangeIsUnused(t *testing.T) {
	contract := strings.Replace(syntheticContract(t), "enum: [THING_PRIORITY_UNSPECIFIED, THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", "enum: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]", 1)
	file := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority: {values: [THING_PRIORITY_LOW, THING_PRIORITY_PAGER_DUTY_INCIDENTS]}\n"
	dir := t.TempDir()
	openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPI, []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, Check(CheckOptions{Resource: "Thing", OpenAPIPath: openAPI, OverridesPath: override}), codeOverrideUnused)
}

func TestEnumMarkerDelimitersAreRejected(t *testing.T) {
	for name, line := range map[string]string{
		"zero":    "zero: 'unspec,ified'\n    values: [THING_PRIORITY_LOW]",
		"names":   "values: [THING_PRIORITY_LOW]\n    names: {THING_PRIORITY_LOW: 'foo,bar'}",
		"aliases": "values: [THING_PRIORITY_LOW]\n    aliases: {'foo,bar': low}",
	} {
		t.Run(name, func(t *testing.T) {
			data := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\nenums:\n  ThingPriority:\n    " + line + "\n"
			if _, err := overrides.Parse([]byte(data)); err == nil || !strings.Contains(err.Error(), "enums.ThingPriority") {
				t.Fatalf("Parse() error = %v, want an enum path error", err)
			}
		})
	}
}
