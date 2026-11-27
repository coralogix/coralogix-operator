package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"go.yaml.in/yaml/v4"
)

// A new optional API field reaches every generated OutboundWebhook surface without a new override.
func TestOutboundWebhookNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "OutboundWebhook", Resource: "OutgoingWebhook"}
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
		doc, err := (&documents{raw: spec}).forKind(entry, file)
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
	after := render(withOutboundWebhookProbeNote(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines: %s", name, strings.Join(removed, "; "))
		}
	}
	for _, check := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/outboundwebhook_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/outboundwebhook_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"api/coralogix/v1alpha1/outboundwebhook_convert.go", "out.ProbeNote = in.ProbeNote"},
		{"config/samples/v1alpha1/generated/outboundwebhook.yaml", "probeNote:"},
		{"tests/e2e/outboundwebhook_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[check.file])), normalizeSpace(check.text)) {
			t.Errorf("%s has no %q", check.file, check.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/outboundwebhook_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("probeNote appears in %d requests; want Create and Update", n)
	}
	if string(after["scripts/upgrade-check-outboundwebhook.sh"]) != string(before["scripts/upgrade-check-outboundwebhook.sh"]) {
		t.Error("a new API field changed the released chart upgrade check")
	}
}

func withOutboundWebhookProbeNote(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	object := func(value any, keys ...string) map[string]any {
		t.Helper()
		current := value.(map[string]any)
		for _, key := range keys {
			current = current[key].(map[string]any)
		}
		return current
	}
	field := map[string]any{"type": "string", "maxLength": 20}
	// Create and Update wrap the webhook in data, which is OutgoingWebhookInputData.
	object(doc, "components", "schemas", "OutgoingWebhookInputData", "properties")["probeNote"] = field
	object(doc, "components", "schemas", "OutgoingWebhook", "properties")["probeNote"] = field
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
