package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

const secretThingHead = "resource: SecretThing\nmode: existing\nvalidators: {inferred: true}\napi:\n  requestWrapper: item\n  idProperty: id\n  serverSetID: true\n  updateIDInBody: true\n"

func writeSecretThingOverrides(t *testing.T, root, typesBlock string) {
	t.Helper()
	path := filepath.Join(root, kindsDir, "secretthing", "behavior-overrides.yaml")
	if err := os.WriteFile(path, []byte(secretThingHead+"types:\n  SecretThingSettings:\n    fields:\n      fields: {collection: list}\n"+typesBlock), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A valueFrom line adds a Secret sibling. Create and Update send the literal, or the value that
// they read from the Secret, in the one API property, and never send the reference.
func TestValueFromReadsTheSecretInBothRequests(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	convert := readGenerated(t, root, apiDir+"/secretthing_convert.go")
	if got := strings.Count(convert, "ResolveSecretKeyRef(ctx, namespace, *in."); got != 6 {
		t.Errorf("%d Secret reads, want 6 (three fields, in Create and in Update):\n%s", got, convert)
	}
	if strings.Contains(convert, "out.SecretKeyRef") || strings.Contains(convert, "out.ApiKeySecretKeyRef") {
		t.Error("a request sends the Secret reference")
	}
	controller := readGenerated(t, root, controllerDir+"/secretthing_controller.go")
	if !strings.Contains(controller, `+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`) {
		t.Error("the controller lacks the Secret read marker")
	}
	thing := readGenerated(t, root, controllerDir+"/thing_controller.go")
	if strings.Contains(thing, "resources=secrets") {
		t.Error("a kind without valueFrom has the Secret read marker")
	}
}

// The CR sets the string or the Secret. The rule needs exactly one when the contract requires the
// string, or the line says requireOne, and at most one otherwise.
func TestValueFromRule(t *testing.T) {
	for name, tc := range map[string]struct {
		line, rule, message string
	}{
		"contract requires the string": {"apiKey: {valueFrom: {secret: keyRef}}", "(has(self.apiKey) ? 1 : 0) + (has(self.keyRef) ? 1 : 0) == 1", "Exactly one of apiKey or keyRef must be set"},
		"requireOne":                   {"token: {valueFrom: {secret: tokenRef}, requireOne: true}", "(has(self.token) ? 1 : 0) + (has(self.tokenRef) ? 1 : 0) == 1", "Exactly one of token or tokenRef must be set"},
		"optional":                     {"token: {valueFrom: {secret: tokenRef}}", "(has(self.token) ? 1 : 0) + (has(self.tokenRef) ? 1 : 0) <= 1", "At most one of token or tokenRef can be set"},
	} {
		root, input := syntheticRoot(t)
		writeSecretThingOverrides(t, root, "  SecretThing:\n    fields:\n      id: {readOnly: true}\n      "+tc.line+"\n")
		if err := generateFromInput(input, root, root); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		types := readGenerated(t, root, apiDir+"/secretthing_types.go")
		if !strings.Contains(types, `rule="`+tc.rule+`",message="`+tc.message+`"`) {
			t.Errorf("%s: want the rule %q with the message %q:\n%s", name, tc.rule, tc.message, types)
		}
	}
}

// A field that the contract requires is optional in the CR, so the minimal spec keeps it: without
// it or its Secret, the exactly-one rule rejects the spec.
func TestValueFromMinimalSpecKeepsTheRequiredString(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	test := readGenerated(t, root, controllerDir+"/secretthing_coverage_test.go")
	minimal := test[strings.Index(test, `name: "minimal spec"`):]
	if !strings.Contains(minimal[:strings.Index(minimal, "},")], `"apiKey"`) {
		t.Errorf("the minimal spec lacks the required apiKey:\n%s", minimal)
	}
}

func TestValueFromRejectsOtherShapes(t *testing.T) {
	for _, tc := range map[string]struct {
		types, code string
	}{
		"an object":                {"  SecretThing:\n    fields:\n      id: {readOnly: true}\n      settings: {valueFrom: {secret: ref}}\n", codeOverrideInvalid},
		"a sibling that exists":    {"  SecretThing:\n    fields:\n      id: {readOnly: true}\n      apiKey: {valueFrom: {secret: token}}\n", codeNameCollision},
		"an unknown property":      {"  SecretThing:\n    fields:\n      id: {readOnly: true}\n      missing: {valueFrom: {secret: ref}}\n", codeOverrideUnused},
		"a field of a nested kind": {"  SecretThing:\n    fields:\n      id: {readOnly: true}\n  SecretThingField:\n    fields:\n      value: {valueFrom: {secret: fieldName}}\n", codeNameCollision},
	} {
		root, input := syntheticRoot(t)
		writeSecretThingOverrides(t, root, tc.types)
		wantCode(t, generateFromInput(input, root, root), tc.code)
	}
}

func TestValueFromParsing(t *testing.T) {
	const existing = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Item:\n    fields:\n"
	for _, tc := range []struct {
		name, data string
		valid      bool
	}{
		{"existing kind", existing + "      value: {valueFrom: {secret: secretKeyRef}}\n", true},
		{"with requireOne", existing + "      value: {valueFrom: {secret: secretKeyRef}, requireOne: true}\n", true},
		{"with a renamed field", existing + "      value: {name: literal, valueFrom: {secret: secretKeyRef}}\n", true},
		{"a new kind", "resource: Thing\nmode: new\ntypes:\n  Item:\n    fields:\n      value: {valueFrom: {secret: secretKeyRef}}\n", false},
		{"no secret", existing + "      value: {valueFrom: {}}\n", false},
		{"not a name", existing + "      value: {valueFrom: {secret: a.b}}\n", false},
		{"the name of the field", existing + "      value: {valueFrom: {secret: value}}\n", false},
		{"the renamed name of the field", existing + "      value: {name: literal, valueFrom: {secret: literal}}\n", false},
		{"configMap is not supported", existing + "      value: {valueFrom: {configMap: x}}\n", false},
		{"with skip", existing + "      value: {valueFrom: {secret: ref}, skip: true}\n", false},
		{"with required", existing + "      value: {valueFrom: {secret: ref}, required: true}\n", false},
		{"with ref", existing + "      value: {valueFrom: {secret: ref}, ref: {kind: Thing}}\n", false},
		{"with joinedBy", existing + "      value: {valueFrom: {secret: ref}, joinedBy: \",\"}\n", false},
		{"with sendNullAsEmpty", existing + "      value: {valueFrom: {secret: ref}, sendNullAsEmpty: true}\n", false},
		{"with requestAs", existing + "      value: {valueFrom: {secret: ref}, requestAs: other}\n", false},
		{"with inline", existing + "      value: {valueFrom: {secret: ref}, inline: true}\n", false},
	} {
		if _, err := overrides.Parse([]byte(tc.data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}
