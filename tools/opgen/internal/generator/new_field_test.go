package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"go.yaml.in/yaml/v4"
)

// A field that a later contract adds to a released kind flows from the contract into the CRD
// type, the conversions, the sample, and the e2e test without a behavior-overrides line or
// handwritten code. The test adds an optional field to the pinned contract of ViewFolder and
// renders it with the committed behavior-overrides file: the rendering keeps every line of the
// rendering of the pinned contract, and adds the field.
func TestReleasedKindTakesANewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "ViewFolder", Resource: "ViewFolder"}
	dir := filepath.Join(input.ModuleRoot, entry.dir())
	file, err := readOverrides(filepath.Join(dir, "behavior-overrides.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	e2e, err := readE2E(filepath.Join(dir, e2eFileName))
	if err != nil {
		t.Fatal(err)
	}
	render := func(spec []byte) map[string][]byte {
		t.Helper()
		doc, err := model.Load(spec)
		if err != nil {
			t.Fatal(err)
		}
		v, err := validate(doc, entry, input.SDKModule, input.Module, e2e, file)
		if err != nil {
			t.Fatal(err)
		}
		out, err := renderAll([]*validated{v}, input.SDKModule, input.Module)
		if err != nil {
			t.Fatal(err)
		}
		return out.files
	}
	before := render(input.OpenAPI)
	after := render(withNewField(t, input.OpenAPI))

	for name, text := range after {
		if removed := removedLines(string(before[name]), string(text)); len(removed) != 0 {
			t.Errorf("%s: a new field changed lines that were there before:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/viewfolder_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/viewfolder_convert.go", "out.ProbeNote = in.ProbeNote"},
		{"config/samples/v1alpha1/generated/viewfolder.yaml", "probeNote:"},
		{"tests/e2e/viewfolder_generated_test.go", `"probeNote":`},
	} {
		got := normalizeSpace(string(after[want.file]))
		if !strings.Contains(got, normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/viewfolder_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Replace request, got %d conversions", n)
	}
}

// withNewField adds the optional field probeNote to the Create body, the Replace body, and the
// resource of ViewFolder.
func withNewField(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	added := 0
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if title := v["title"]; title == "ViewFolder" || title == "CreateViewFolderRequest" {
				if props, ok := v["properties"].(map[string]any); ok {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note."}
					added++
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(doc)
	if added < 3 {
		t.Fatalf("want probeNote in the Create body, the Replace body, and the resource, added it %d times", added)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// removedLines returns the lines of before that after does not have. Space is normalized, because
// gofmt aligns the lines next to a new field again, and so is a trailing comma, because a JSON
// value before a new one gets a comma.
func removedLines(before, after string) []string {
	key := func(line string) string { return strings.TrimSuffix(normalizeSpace(line), ",") }
	count := map[string]int{}
	for _, line := range strings.Split(before, "\n") {
		count[key(line)]++
	}
	for _, line := range strings.Split(after, "\n") {
		count[key(line)]--
	}
	var removed []string
	for line, n := range count {
		for range n {
			removed = append(removed, line)
		}
	}
	slices.Sort(removed)
	return removed
}

func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }
