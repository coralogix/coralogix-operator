package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// foldContract gives Thing a flat Create field and an object that Update and Get nest, the shape
// of an API that takes a limit flat on Create and inside an object on Replace:
//
//	Create:  capacityLimit
//	Replace: capacity {limit (required), exceeded (readOnly)}
func foldContract(t *testing.T, extra string) func(string) string {
	return func(s string) string {
		const quota = "                quota: {type: string, format: int64, x-coralogix-presence: true}\n"
		if strings.Count(s, quota) != 2 {
			t.Fatal("the synthetic Create and Replace bodies changed")
		}
		i := strings.Index(s, quota) + len(quota)
		s = s[:i] + "                capacityLimit: {type: integer, format: int32, minimum: 1, maximum: 100, x-coralogix-presence: true}\n" + s[i:]
		i = strings.Index(s[i:], quota) + i + len(quota)
		s = s[:i] + "                capacity: {$ref: '#/components/schemas/ThingCapacity'}\n" + s[i:]
		resource := "        quota: {type: string, format: int64, description: \"The quota, a 64-bit number.\"}\n"
		s = replaceOnce(t, s, resource, resource+"        capacity: {$ref: '#/components/schemas/ThingCapacity'}\n")
		return replaceOnce(t, s, "    ThingKind:\n", "    ThingCapacity:\n      type: object\n      required: [limit]\n      properties:\n        limit: {type: integer, format: int32, minimum: 1, maximum: 100, description: The limit.}\n        exceeded: {type: boolean, readOnly: true}\n"+extra+"    ThingKind:\n")
	}
}

func foldSDK(t *testing.T, extra string) func(string) string {
	return func(s string) string {
		s = replaceOnce(t, s, "type CreateThingRequest struct {\n", "type CreateThingRequest struct {\n	CapacityLimit *int32 `json:\"capacityLimit,omitempty\"`\n")
		s = replaceOnce(t, s, "type ReplaceThingRequest struct {\n", "type ReplaceThingRequest struct {\n	Capacity *ThingCapacity `json:\"capacity,omitempty\"`\n")
		s = replaceOnce(t, s, "type Thing struct {\n", "type Thing struct {\n	Capacity *ThingCapacity `json:\"capacity,omitempty\"`\n")
		return replaceOnce(t, s, "type ThingKind string", "type ThingCapacity struct {\n	Limit int32 `json:\"limit\"`\n	Exceeded *bool `json:\"exceeded,omitempty\"`\n"+extra+"}\n\ntype ThingKind string")
	}
}

const foldOverrides = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: capacityLimit}}\n"

func writeOverrides(t *testing.T, root, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The CR keeps the flat Create field. Create sends it as it is. Update builds the object when the
// field is set, and leaves it out otherwise, so the API keeps its stored value or default.
func TestCreateAsKeepsTheFlatCreateField(t *testing.T) {
	root, input := skipRoot(t, foldContract(t, ""), foldSDK(t, ""))
	writeOverrides(t, root, foldOverrides)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if !strings.Contains(types, "CapacityLimit *int32") || strings.Contains(types, "Capacity ") {
		t.Fatalf("the CR must have the flat field and no object:\n%s", types)
	}
	if !strings.Contains(types, "+kubebuilder:validation:Maximum=100") {
		t.Fatal("the contract limits of the Create field are missing")
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	if !strings.Contains(convert, "out.CapacityLimit = in.CapacityLimit") {
		t.Fatalf("Create does not send the flat field:\n%s", convert)
	}
	update := convert[strings.Index(convert, "func (in *ThingSpec) toReplaceThingRequest"):]
	for _, want := range []string{"if in.CapacityLimit != nil {", "thingsservice.ThingCapacity{}", ".Limit = ", "out.Capacity = "} {
		if !strings.Contains(update, want) {
			t.Errorf("Update does not build the object (missing %q):\n%s", want, update)
		}
	}
	if strings.Contains(update, "Exceeded") {
		t.Error("Update sends a read-only property")
	}
}

// A line in validators.fields names the flat CR field.
func TestCreateAsTakesValidatorLinesOnTheFlatField(t *testing.T) {
	root, input := skipRoot(t, foldContract(t, ""), foldSDK(t, ""))
	writeOverrides(t, root, "resource: Thing\nmode: existing\nvalidators:\n  inferred: true\n  fields:\n    spec.capacityLimit: {inferred: false}\ntypes:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: capacityLimit}}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readGenerated(t, root, apiDir+"/thing_types.go"), "Maximum=100") {
		t.Fatal("inferred: false did not drop the limit of the flat field")
	}
}

// A writable property of the object that no line names has no CR field. Dropping it would be silent.
func TestCreateAsRejectsAWritablePropertyThatNoLineNames(t *testing.T) {
	editAPI := func(s string) string {
		s = foldContract(t, "        burst: {type: integer, format: int32}\n")(s)
		return s
	}
	editSDK := foldSDK(t, "	Burst *int32 `json:\"burst,omitempty\"`\n")
	root, input := skipRoot(t, editAPI, editSDK)
	writeOverrides(t, root, foldOverrides)
	err := generateFromInput(input, root, root)
	if err == nil || !strings.Contains(err.Error(), "burst") {
		t.Fatalf("want an error that names the property, got %v", err)
	}
}

func TestCreateAsLineIsChecked(t *testing.T) {
	const head = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"
	for name, tc := range map[string]string{
		"a new kind":            "resource: Thing\nmode: new\ntypes:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: capacityLimit}}\n",
		"a nested component":    head + "types:\n  ThingCapacity:\n    fields:\n      limit: {createAs: {x: y}}\n",
		"an empty target":       head + "types:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: ''}}\n",
		"the same target":       head + "types:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: a, other: a}}\n",
		"another key":           head + "types:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: a}, name: cap}\n",
		"skip on the same line": head + "types:\n  Thing:\n    fields:\n      capacity: {createAs: {limit: a}, skip: true}\n",
	} {
		if _, err := overrides.Parse([]byte(tc)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := overrides.Parse([]byte(foldOverrides)); err != nil {
		t.Fatalf("a valid line: %v", err)
	}
}

// A line that names a Create field that the API does not have changes nothing and stops the run.
func TestCreateAsRejectsAMissingCreateField(t *testing.T) {
	root, input := skipRoot(t, foldContract(t, ""), foldSDK(t, ""))
	writeOverrides(t, root, strings.Replace(foldOverrides, "capacityLimit", "missingLimit", 1))
	if err := generateFromInput(input, root, root); err == nil {
		t.Fatal("a createAs line with a missing Create field must stop the run")
	}
}
