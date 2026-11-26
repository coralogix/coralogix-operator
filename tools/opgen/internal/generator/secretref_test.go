package generator

import (
	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func writeArmThingOverrides(t *testing.T, root, types string) {
	t.Helper()
	data := "resource: ArmThing\nmode: existing\nvalidators: {inferred: true}\napi: {updateIDInBody: true}\n" + types
	if err := os.WriteFile(filepath.Join(root, kindsDir, "armthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A secretRef field reads its value from the Secret on each write, and the controller indexes and
// watches the Secrets that a CR names.
func TestSecretRefReadsTheSecretAndWatchesIt(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	convert := readGenerated(t, root, apiDir+"/armthing_convert.go")
	for _, want := range []string{
		`ResolveSecretValue(ctx, namespace, "spec.hook.token", in.Token, in.TokenSecretKeyRef)`,
		`ResolveSecretValues(ctx, namespace, "spec.hook.headers", in.Headers, in.HeadersFromSecret)`,
		"func (in *ArmThing) SecretNames() []string {",
		"for _, ref := range in.HeadersFromSecret {\n\t\tnames[ref.Name] = true\n\t}",
		"if in.Quiet != nil {\n\t\tin.Quiet.addSecretNames(names)\n\t}",
	} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion has no %q", want)
		}
	}
	controller := readGenerated(t, root, controllerDir+"/armthing_controller.go")
	for _, want := range []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
		"Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.forSecret))",
		"For(&coralogixv1alpha1.ArmThing{}, builder.WithPredicates(config.GetConfig().Selector.Predicate()))",
		"client.MatchingFields{armthingSecretIndex: obj.GetName()}",
	} {
		if !strings.Contains(controller, want) {
			t.Errorf("the controller has no %q", want)
		}
	}
	// A kind without a secretRef line keeps its controller as it was.
	if thing := readGenerated(t, root, controllerDir+"/thing_controller.go"); strings.Contains(thing, "Secret") {
		t.Error("the Thing controller watches Secrets")
	}
}

// An inArms field leaves the top of the spec. Each arm holds it, and the request sends the value of
// the arm that the CR sets. The conversion of an empty arm still sends an empty object.
func TestInArmsShowsATopLevelFieldInEachArm(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/armthing_types.go")
	spec := types[strings.Index(types, "type ArmThingSpec struct"):]
	spec = spec[:strings.Index(spec, "\n}\n")]
	if strings.Contains(spec, "Url") {
		t.Errorf("the spec keeps the URL:\n%s", spec)
	}
	if strings.Count(types, "Url *string `json:\"url,omitempty\"`") != 2 {
		t.Error("the two arms do not hold the URL")
	}
	convert := readGenerated(t, root, apiDir+"/armthing_convert.go")
	for _, want := range []string{
		`ResolveSecretValue(ctx, namespace, "spec.quiet.url", in.Quiet.Url, in.Quiet.UrlSecretKeyRef)`,
		"if in.Quiet != nil {\n\t\tout.Quiet = map[string]interface{}{}\n\t}",
	} {
		if !strings.Contains(convert, want) {
			t.Errorf("the conversion has no %q", want)
		}
	}
	if sample := readGenerated(t, root, "config/samples/v1alpha1/generated/armthing.yaml"); !strings.Contains(sample, "    url: sample-url\n") {
		t.Errorf("the sample sets no URL in the arm:\n%s", sample)
	}
}

func TestSecretRefAndInArmsRejectOtherShapes(t *testing.T) {
	for name, types := range map[string]string{
		"inArms naming a field that is not an arm": "types:\n  ArmThing:\n    fields:\n      url: {inArms: [name]}\n",
		"inArms on an arm":                         "types:\n  ArmThing:\n    fields:\n      hook: {inArms: [quiet]}\n",
	} {
		t.Run(name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			writeArmThingOverrides(t, root, types)
			wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
		})
	}
	t.Run("secretRef on an optional string", func(t *testing.T) {
		root, input := syntheticRoot(t)
		writeArmThingOverrides(t, root, "types:\n  ArmThing:\n    fields:\n      url: {required: true, secretRef: true, inArms: [hook, quiet]}\n  ArmThingHook:\n    fields:\n      note: {secretRef: true}\n")
		if err := generateFromInput(input, root, root); err != nil {
			t.Fatal(err)
		}
		types := readGenerated(t, root, apiDir+"/armthing_types.go")
		if !strings.Contains(types, `rule="!(has(self.note) && has(self.noteSecretKeyRef))",message="Only one of note or noteSecretKeyRef can be set"`) {
			t.Error("an optional string has no at-most-one rule")
		}
		// The empty API default would fill note and break the rule for a CR that names only the Secret.
		if i := strings.Index(types, "Note *string"); i < 0 || strings.Contains(types[strings.LastIndex(types[:i], "// A note"):i], "kubebuilder:default") {
			t.Error("a secretRef string keeps its API default")
		}
	})
	t.Run("secretRef on a field with a default", func(t *testing.T) {
		// Kubernetes would fill the literal, so a CR that names only the Secret would break the rule.
		root, input := syntheticRoot(t)
		data := "resource: ArmThing\nmode: existing\napi: {updateIDInBody: true}\n" +
			"validators:\n  inferred: true\n  fields:\n    spec.hook.note:\n      add: ['+kubebuilder:default=none']\n" +
			"types:\n  ArmThing:\n    fields:\n      url: {required: true, secretRef: true, inArms: [hook, quiet]}\n  ArmThingHook:\n    fields:\n      note: {secretRef: true}\n"
		if err := os.WriteFile(filepath.Join(root, kindsDir, "armthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		err := generateFromInput(input, root, root)
		wantCode(t, err, codeOverrideInvalid)
		if err == nil || !strings.Contains(err.Error(), "spec.hook.note") {
			t.Errorf("the error does not name the field: %v", err)
		}
	})
	t.Run("secretRef on an integer", func(t *testing.T) {
		root, input := skipRoot(t, nil, nil)
		writeOverrides(t, root, "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  ThingSettings:\n    fields:\n      retries: {secretRef: true}\n")
		wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
	})
	t.Run("secretRef on a map with value limits", func(t *testing.T) {
		root, input := skipRoot(t, nil, nil)
		writeOverrides(t, root, "resource: Thing\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  Thing:\n    fields:\n      labels: {secretRef: true}\n")
		wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
	})
}

func TestSecretRefAndInArmsParsing(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		valid      bool
	}{
		{"secretRef on an existing kind", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      token: {secretRef: true}\n", true},
		{"secretRef on a new kind", "resource: A\nmode: new\ntypes:\n  A:\n    fields:\n      token: {secretRef: true}\n", true},
		{"secretRef with skip", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      token: {secretRef: true, skip: true}\n", false},
		{"secretRef with optional", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      token: {secretRef: true, optional: true}\n", false},
		{"secretRef with ref", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      token: {secretRef: true, ref: {kind: Thing}}\n", false},
		{"inArms", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      url: {required: true, secretRef: true, inArms: [b, c]}\n", true},
		{"inArms on a new kind", "resource: A\nmode: new\ntypes:\n  A:\n    fields:\n      url: {inArms: [b]}\n", false},
		{"inArms on a nested component", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  B:\n    fields:\n      url: {inArms: [b]}\n", false},
		{"inArms with an arm twice", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      url: {inArms: [b, b]}\n", false},
		{"inArms with a name", "resource: A\nmode: existing\nvalidators: {inferred: true}\ntypes:\n  A:\n    fields:\n      url: {name: address, inArms: [b]}\n", false},
	} {
		if _, err := overrides.Parse([]byte(tc.data)); (err == nil) != tc.valid {
			t.Errorf("%s: valid = %t, want %t: %v", tc.name, err == nil, tc.valid, err)
		}
	}
}

// sharedArmRoot is the synthetic root with an ArmThing whose two arms are one component.
func sharedArmRoot(t *testing.T, validators string) (string, source.Input) {
	t.Helper()
	root, input := editedRoot(t, func(api string) string {
		return strings.ReplaceAll(api, "quiet: {$ref: '#/components/schemas/ArmThingQuiet'}", "quiet: {$ref: '#/components/schemas/ArmThingHook'}")
	}, "go/openapi/gen/arm_things_service/arm_things.go", func(sdk string) string {
		return strings.ReplaceAll(sdk, "Quiet map[string]interface{}", "Quiet *ArmThingHook")
	})
	data := "resource: ArmThing\nmode: existing\napi: {updateIDInBody: true}\nvalidators:\n  inferred: true\n" + validators +
		"types:\n  ArmThing:\n    fields:\n      url: {required: true, inArms: [hook]}\n"
	if err := os.WriteFile(filepath.Join(root, kindsDir, "armthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, input
}

// The arm hook shares its component with the arm quiet. inArms gives hook a struct of its own,
// so the URL and its rule never reach quiet.
func TestInArmsGivesASharedArmItsOwnType(t *testing.T) {
	root, input := sharedArmRoot(t, "")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := normalizeSpace(readGenerated(t, root, apiDir+"/armthing_types.go"))
	if n := strings.Count(types, "Url string `json:\"url\"`"); n != 1 {
		t.Errorf("the URL is in %d structs; want only the hook arm", n)
	}
	if !strings.Contains(types, "Hook *ArmThingHookAtSpecHook") || !strings.Contains(types, "Quiet *ArmThingHook `") {
		t.Error("the hook arm has no struct of its own, or quiet lost the shared one")
	}
	shared := types[strings.Index(types, "type ArmThingHook struct {"):]
	if shared = shared[:strings.Index(shared, "}")]; strings.Contains(shared, "Url") {
		t.Error("the struct of quiet has the URL of hook")
	}
}

// inferred: false at the arm path drops the contract limits of the copy in that arm.
func TestInArmsInfersValidationAtTheArmPath(t *testing.T) {
	root, input := skipRoot(t, func(api string) string {
		return strings.ReplaceAll(api, "url: {type: string, minLength: 0, x-coralogix-presence: true, description: The URL that the thing calls.}",
			"url: {type: string, minLength: 1, x-coralogix-presence: true, description: The URL that the thing calls.}")
	}, nil)
	data := "resource: ArmThing\nmode: existing\napi: {updateIDInBody: true}\n" +
		"validators:\n  inferred: true\n  fields:\n    spec.quiet.url: {inferred: false}\n" +
		"types:\n  ArmThing:\n    fields:\n      url: {required: true, inArms: [hook, quiet]}\n"
	if err := os.WriteFile(filepath.Join(root, kindsDir, "armthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/armthing_types.go")
	if n := strings.Count(types, "+kubebuilder:validation:MinLength=1\n\tUrl"); n != 1 {
		t.Errorf("the contract limit of the URL is in %d arms; want only hook", n)
	}
}

// A validators line at the top-level path of a field that inArms moves has no field to apply to.
func TestInArmsRejectsValidationAtTheTopLevelPath(t *testing.T) {
	root, input := syntheticRoot(t)
	data := "resource: ArmThing\nmode: existing\napi: {updateIDInBody: true}\n" +
		"validators:\n  inferred: true\n  fields:\n    spec.url: {inferred: false}\n" +
		"types:\n  ArmThing:\n    fields:\n      url: {required: true, inArms: [hook, quiet]}\n"
	if err := os.WriteFile(filepath.Join(root, kindsDir, "armthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}

// A secretRef field counts as set when the CR names only its Secret: in the guard of a request
// object (requestAs with a dot), and in an exactlyOneOf rule.
func TestSecretRefCountsTheSecretAsSet(t *testing.T) {
	const head = "resource: WrappedThing\nmode: existing\nvalidators:\n  inferred: true\napi:\n  requestWrapper: item\n  serverSetID: true\n  idProperty: id\n  updateIDInBody: true\n  requestValues:\n    create:\n      label: {source: spec.name}\n      requestId: {source: uuid, value: \"cx-operator-create-\"}\n    update:\n      label: {source: spec.name}\n      requestId: {source: uuid, value: \"cx-operator-replace-\"}\n"
	const fields = "types:\n  WrappedThing:\n    fields:\n      id:\n        readOnly: true\n      title: {requestAs: heading}\n      note: {requestAs: extra.note, secretRef: true}\n"
	generate := func(t *testing.T, data string) string {
		t.Helper()
		root, input := syntheticRoot(t)
		if err := os.WriteFile(filepath.Join(root, kindsDir, "wrappedthing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := generateFromInput(input, root, root); err != nil {
			t.Fatal(err)
		}
		return root
	}
	t.Run("request object guard", func(t *testing.T) {
		root := generate(t, head+fields)
		convert := readGenerated(t, root, apiDir+"/wrappedthing_convert.go")
		if n := strings.Count(convert, "in.Note != nil || in.NoteSecretKeyRef != nil"); n != 2 {
			t.Errorf("the extra object is guarded by the Secret in %d requests; want Create and Update", n)
		}
	})
	t.Run("exactlyOneOf", func(t *testing.T) {
		root := generate(t, head+"exactlyOneOf:\n  - [note, title]\n"+fields)
		types := readGenerated(t, root, apiDir+"/wrappedthing_types.go")
		if !strings.Contains(types, `((has(self.note) || has(self.noteSecretKeyRef)) ? 1 : 0) + (has(self.title) ? 1 : 0) == 1`) {
			t.Error("the exactlyOneOf rule does not count noteSecretKeyRef as note")
		}
	})
}
