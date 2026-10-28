package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

// checkWith runs opgen check on the synthetic contract after edit changes it. e2eKind names the
// e2e.yaml file of a kind in the seed root, "" for none.
func checkWith(t *testing.T, resource, e2eKind string, edit func(string) string) error {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
	options := CheckOptions{Resource: resource, OpenAPIPath: path}
	if e2eKind != "" {
		options.E2EPath = filepath.Join("testdata", "root", "tools", "opgen", "kinds", e2eKind, e2eFileName)
	}
	return Check(options)
}

func replaceOnce(t *testing.T, s, old, replacement string) string {
	t.Helper()
	if strings.Count(s, old) != 1 {
		t.Fatalf("want one %q", old)
	}
	return strings.Replace(s, old, replacement, 1)
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *EligibilityError
	if !errors.As(err, &e) {
		t.Fatalf("want an eligibility error with %s, got %v", code, err)
	}
	for _, is := range e.Report {
		if is.Code == code {
			return
		}
	}
	t.Fatalf("want %s, got:\n%s", code, e.Error())
}

func TestCheckAcceptsTheSyntheticContracts(t *testing.T) {
	for resource, e2eKind := range map[string]string{"Thing": "thing", "PatchThing": ""} {
		if err := checkWith(t, resource, e2eKind, func(s string) string { return s }); err != nil {
			t.Fatalf("%s: %v", resource, err)
		}
	}
}

// check reports a contract gap with the code of iac, and a shape that the operator renderer cannot
// write with the code of the operator checks.
func TestCheckReportsGaps(t *testing.T) {
	tests := []struct {
		name string
		edit func(t *testing.T, s string) string
		code string
	}{
		{
			name: "a required string that accepts an empty value",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "topic: {type: string, minLength: 1}", "topic: {type: string}")
			},
			code: "REQUIRED_VALUE_CAN_BE_EMPTY",
		},
		{
			// settings is visited first as a plain field, so the list reuses its Go type.
			name: "an immutable field in an object that a list also holds",
			edit: func(t *testing.T, s string) string {
				for _, c := range []struct{ indent, component string }{
					{"                ", "ThingSettingsCreate"},
					{"                ", "ThingSettingsUpdate"},
					{"        ", "ThingSettings"},
				} {
					settings := c.indent + "settings: {$ref: '#/components/schemas/" + c.component + "'}\n"
					history := c.indent + "history:\n" + c.indent + "  type: array\n" + c.indent + "  items: {$ref: '#/components/schemas/" + c.component + "'}\n"
					s = replaceOnce(t, s, settings, settings+history)
				}
				return s
			},
			code: codeShapeUnsupported,
		},
		{
			// The values of spec.labels have limits, so their Go type is ThingLabelsValue.
			name: "a component with the Go name of a map value type",
			edit: func(t *testing.T, s string) string {
				return strings.ReplaceAll(s, "ThingItem", "ThingLabelsValue")
			},
			code: codeNameCollision,
		},
		{
			// The shared module rejects it before the CRD walk, which has no cycle guard.
			name: "an object that holds itself",
			edit: func(t *testing.T, s string) string {
				weight := "        weight: {type: integer, format: int32, minimum: 1}\n"
				return replaceOnce(t, s, weight, weight+"        child: {$ref: '#/components/schemas/ThingItem'}\n")
			},
			code: "SCHEMA_RECURSIVE",
		},
		{
			name: "a number",
			edit: func(t *testing.T, s string) string {
				s = replaceOnce(t, s, "weight: {type: integer, format: int32, minimum: 1}", "weight: {type: number, format: double, minimum: 1}")
				return s
			},
			code: codeShapeUnsupported,
		},
		{
			name: "a made-up value that cannot match the pattern",
			edit: func(t *testing.T, s string) string {
				return replaceOnce(t, s, "endpoint: {type: string, minLength: 1, description: The endpoint URL.}",
					"endpoint: {type: string, minLength: 1, pattern: '^https://.+$', description: The endpoint URL.}")
			},
			code: codeE2ERequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCode(t, checkWith(t, "Thing", "thing", func(s string) string { return tt.edit(t, s) }), tt.code)
		})
	}
}

func TestEnumName(t *testing.T) {
	for value, want := range map[string]string{
		"ENTITY_TYPE_PAGER_DUTY_INCIDENTS": "pagerDutyIncidents",
		"ENTITY_TYPE_ALERTS":               "alerts",
		"OTHER_P1":                         "otherP1",
	} {
		if got := enumName(value, "ENTITY_TYPE_UNSPECIFIED"); got != want {
			t.Errorf("enumName(%s) = %s, want %s", value, got, want)
		}
	}
}

func TestKindNames(t *testing.T) {
	for kind, want := range map[string][2]string{
		"ViewFolder":   {"view-folder", "viewfolders"},
		"AIEvaluation": {"ai-evaluation", "aievaluations"},
		"IPAccess":     {"ip-access", "ipaccesses"},
		"Policy":       {"policy", "policies"},
	} {
		if got := [2]string{kebab(kind), plural(kind)}; got != want {
			t.Errorf("%s: got %v, want %v", kind, got, want)
		}
	}
}

// Publish never replaces a file without the generator header, and one problem writes nothing.
func TestPublishKeepsHandwrittenFiles(t *testing.T) {
	root, input := syntheticRoot(t)
	handwritten := filepath.Join(root, "api", "coralogix", "v1alpha1", "thing_types.go")
	if err := os.MkdirAll(filepath.Dir(handwritten), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handwritten, []byte("package v1alpha1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := generateFromInput(input, root, root)
	if err == nil || !strings.Contains(err.Error(), "has no "+version.Tool+" header") {
		t.Fatalf("want a refusal to replace the handwritten file, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "utils", "generated_kinds.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed run wrote a file: %v", err)
	}
	project, err := os.ReadFile(filepath.Join(root, "PROJECT"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(project), "kind: Thing") {
		t.Fatal("a failed run changed PROJECT")
	}
}

// A block needs both marker lines.
func TestPublishNeedsMarkers(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, "config", "crd", "kustomization.yaml")
	if err := os.WriteFile(path, []byte("resources: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, root); err == nil || !strings.Contains(err.Error(), blockBegin) {
		t.Fatalf("want an error about the markers, got %v", err)
	}
}

// A generated file of a kind that is no longer listed is removed. A handwritten file stays.
func TestPublishRemovesFilesOfRemovedKinds(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	handwritten := filepath.Join(root, "api", "coralogix", "v1alpha1", "handwritten_types.go")
	if err := os.WriteFile(handwritten, []byte("package v1alpha1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, kindsFile), []byte("kinds:\n  - {kind: PatchThing, resource: PatchThing}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"thing", "refthing", "bodyidthing"} {
		if err := os.RemoveAll(filepath.Join(root, kindsDir, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"api/coralogix/v1alpha1/thing_types.go", "tests/e2e/thing_generated_test.go", "config/samples/v1alpha1/generated/thing.yaml"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s of the removed kind is still there: %v", rel, err)
		}
	}
	if _, err := os.Stat(handwritten); err != nil {
		t.Errorf("the handwritten file is gone: %v", err)
	}
}

func TestReadKindsRejectsUnknownKeysAndDirectories(t *testing.T) {
	root, _ := syntheticRoot(t)
	if err := os.WriteFile(filepath.Join(root, kindsFile), []byte("kinds:\n  - {kind: Thing, resource: Thing, extra: 1}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readKinds(root); err == nil || !strings.Contains(err.Error(), "extra") {
		t.Fatalf("want an unknown key error, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, kindsFile), []byte("kinds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readKinds(root); err == nil || !strings.Contains(err.Error(), "belongs to no kind") {
		t.Fatalf("want an error about kinds/thing, got %v", err)
	}
}

// A generated name that a handwritten file of the same package declares stops the run.
func TestHandwrittenNameCollision(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, "api", "coralogix", "v1alpha1", "handwritten_types.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package v1alpha1\n\ntype ThingConfig struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantCode(t, generateFromInput(input, root, root), codeNameCollision)
	if _, err := os.Stat(filepath.Join(root, "api", "coralogix", "v1alpha1", "thing_types.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed run wrote a file: %v", err)
	}
}

// Two kinds of one run that declare the same Go name stop the run before it writes. The kind
// ThingConfig makes the root type ThingConfig, and the kind Thing makes it for its component.
func TestGeneratedNameCollision(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, "tools", "opgen", "kinds.yaml")
	kinds, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(kinds, "  - kind: ThingConfig\n    resource: PatchThing\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	err = generateFromInput(input, root, root)
	wantCode(t, err, codeNameCollision)
	if !strings.Contains(err.Error(), "ThingConfig is also declared in the generated file") {
		t.Fatalf("want the clash between the two kinds, got:\n%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "api", "coralogix", "v1alpha1", "thing_types.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed run wrote a file: %v", err)
	}
}

// A list item or a map value in a wrapper stops the renderer, as a wrapped field does. Only a
// behavior-overrides line makes a wrapper, so the test builds the type itself.
func TestWrappedElementIsUnsupported(t *testing.T) {
	item := &model.Type{Kind: model.String, Wrappers: []model.Wrapper{{}}}
	for _, kind := range []model.Kind{model.List, model.Map} {
		b := &builder{kind: "Thing", types: map[string]*crType{}, enums: map[string]*enumMap{}}
		_, err := b.field("Thing", "Thing", "tags", "", &model.Type{Kind: kind, Elem: item}, false, "spec.tags", false)
		s, ok := isShapeError(err)
		if !ok || s.code != codeShapeUnsupported || s.path != "spec.tags[]" {
			t.Errorf("%s: want %s at spec.tags[], got %v", kind, codeShapeUnsupported, err)
		}
	}
}
