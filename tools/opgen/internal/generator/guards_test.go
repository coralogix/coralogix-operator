package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const updateOnlyOverrides = "resource: BodyIDThing\nmode: existing\nvalidators: {inferred: false}\napi: {updateIDInBody: true}\n"

// updateOnlyContract has a limit field in the Replace body and the resource, but not in the Create
// body. The Replace body gives it a default, as the views API does for isCompactMode. Without the
// default the shared model already rejects the field; with it, the model calls the field managed.
func updateOnlyContract(t *testing.T) string {
	t.Helper()
	property := "                limit: {type: integer, format: int32, minimum: 1, maximum: 10, x-coralogix-presence: true}\n"
	withDefault := "                limit: {type: integer, format: int32, minimum: 1, maximum: 10, default: 1, x-coralogix-presence: true}\n"
	contract := syntheticContract(t)
	if strings.Count(contract, property) != 2 {
		t.Fatal("the fixture has no limit in both request bodies")
	}
	contract = strings.Replace(contract, property, "", 1)
	return strings.Replace(contract, property, withDefault, 1)
}

func checkUpdateOnly(t *testing.T, overrides string) error {
	t.Helper()
	dir := t.TempDir()
	openAPI, override := filepath.Join(dir, "openapi.yaml"), filepath.Join(dir, "behavior-overrides.yaml")
	if err := os.WriteFile(openAPI, []byte(updateOnlyContract(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte(overrides), 0o644); err != nil {
		t.Fatal(err)
	}
	return Check(CheckOptions{Resource: "BodyIDThing", OpenAPIPath: openAPI, OverridesPath: override})
}

// A field that only Update has must not vanish without a word.
func TestUpdateOnlyFieldStopsGeneration(t *testing.T) {
	err := checkUpdateOnly(t, updateOnlyOverrides)
	wantCode(t, err, codeUpdateOnlyField)
	if !strings.Contains(err.Error(), "types.BodyIDThing.fields.limit") || !strings.Contains(err.Error(), "skip: true") {
		t.Errorf("the report must name the field and the skip line:\n%v", err)
	}
}

func TestUpdateOnlyFieldCanBeSkipped(t *testing.T) {
	skipped := updateOnlyOverrides + "types:\n  BodyIDThing:\n    fields:\n      limit: {skip: true}\n"
	if err := checkUpdateOnly(t, skipped); err != nil {
		t.Fatalf("a skipped Update-only field must pass: %v", err)
	}
}

// The generator writes the entry of a kind in the marked block. A handwritten entry of the same
// kind outside the block would list it twice.
func TestDuplicateHandwrittenEntryStopsGeneration(t *testing.T) {
	root := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	block := func(lines string) string { return "# BEGIN generated kinds\n" + lines + "# END generated kinds\n" }
	entries := map[string][]string{
		projectFile:      {"kind: Thing"},
		crdKustomization: {"- bases/coralogix.com_things.yaml"},
		helmClusterRole:  {"- things", "- things/finalizers", "- things/status"},
	}
	out := output{entries: entries}

	write(projectFile, "- kind: Other\n"+block("  kind: Thing\n"))
	write(crdKustomization, "  - bases/coralogix.com_others.yaml\n"+block("  - bases/coralogix.com_things.yaml\n"))
	write(helmClusterRole, "      - others\n"+block("      - things\n      - things/finalizers\n      - things/status\n"))
	if report := duplicateEntries(root, out); len(report) != 0 {
		t.Fatalf("a kind listed only in the block is not a duplicate: %v", report)
	}

	// An entry of another kind with the same prefix is not a duplicate: ViewFolder next to View.
	write(projectFile, "  kind: ThingFolder\n"+block("  kind: Thing\n"))
	if report := duplicateEntries(root, out); len(report) != 0 {
		t.Fatalf("kind: ThingFolder is not kind: Thing: %v", report)
	}

	write(projectFile, "  kind: Thing\n"+block("  kind: Thing\n"))
	write(crdKustomization, "  - bases/coralogix.com_things.yaml\n"+block("  - bases/coralogix.com_things.yaml\n"))
	write(helmClusterRole, "      - things\n      - things/status\n"+block("      - things\n      - things/finalizers\n      - things/status\n"))
	report := duplicateEntries(root, out)
	if len(report) != 4 {
		t.Fatalf("want 4 duplicates (project, kustomization, role: things and things/status), got %d: %v", len(report), report)
	}
	for _, is := range report {
		if is.Code != codeDuplicateEntry || !strings.Contains(is.Remediation, "Remove the handwritten entry") {
			t.Errorf("unexpected issue: %+v", is)
		}
	}
}

// The whole run stops when a handwritten entry of a generated kind is left in a marked file, and
// it writes nothing. This runs the real path: render, collect the entries, check, publish.
func TestGenerateStopsOnHandwrittenEntry(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, "PROJECT")
	project, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte("- api:\n    crdVersion: v1\n  kind: Thing\n"), project...), 0o644); err != nil {
		t.Fatal(err)
	}
	err = generateFromInput(input, root, root)
	wantCode(t, err, codeDuplicateEntry)
	if !strings.Contains(err.Error(), "PROJECT") {
		t.Fatalf("the report must name the file:\n%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "api", "coralogix", "v1alpha1", "thing_types.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed run wrote a file: %v", err)
	}
}
