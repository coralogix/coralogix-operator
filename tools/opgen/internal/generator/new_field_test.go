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
		{"api/coralogix/v1alpha1/viewfolder_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"api/coralogix/v1alpha1/viewfolder_types.go", "+kubebuilder:default=`memo`"},
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

// A new GlobalRouter field enters the current generated test. The chart 2.0 create spec stays
// fixed, because that chart cannot accept fields added after its release.
func TestGlobalRouterNewFieldKeepsReleasedUpgradeSpec(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "GlobalRouter", Resource: "GlobalRouter"}
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
	after := render(withNewGlobalRouterField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/globalrouter_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/globalrouter_convert.go", "out.ProbeNote = in.ProbeNote"},
		{"config/samples/v1alpha1/generated/globalrouter.yaml", "probeNote:"},
		{"tests/e2e/globalrouter_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if string(after["scripts/upgrade-check-globalrouter.sh"]) != string(before["scripts/upgrade-check-globalrouter.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

func withNewGlobalRouterField(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	router := schemas["GlobalRouter"].(map[string]any)
	properties := router["properties"].(map[string]any)
	properties["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A test note."}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
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
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20, "default": "memo"}
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

// A new View field flows into the type, both requests, the sample, and the current generated
// test with no behavior-overrides line. The chart 0.5 create spec stays fixed.
func TestViewNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "View", Resource: "View"}
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
	after := render(withNewViewField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/view_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/view_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/view.yaml", "probeNote:"},
		{"tests/e2e/view_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/view_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Replace request, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-view.sh"]) != string(before["scripts/upgrade-check-view.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewViewField adds the optional field probeNote to every schema with the View properties:
// the Create body, the Replace body, and the resource. The bodies are inline in the contract.
func withNewViewField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasTime := props["timeSelection"]
				_, hasFilters := props["filters"]
				_, hasFolder := props["folderId"]
				if hasTime && hasFilters && hasFolder {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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

// A field that a later contract adds to Events2Metric reaches the CRD type, both conversions, the
// sample, and the e2e test with no new behavior-overrides line. The released chart upgrade check
// stays as it is. The createAs, sendNullAsEmpty, and nested skip lines of the kind do not stop it.
func TestEvents2MetricNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "Events2Metric", Resource: "E2M"}
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
	after := render(withNewE2MField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/events2metric_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/events2metric_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/events2metric.yaml", "probeNote:"},
		{"tests/e2e/events2metric_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/events2metric_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Replace request, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-events2metric.sh"]) != string(before["scripts/upgrade-check-events2metric.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewE2MField adds the optional field probeNote to every schema with the E2M properties: the
// Create body, the Replace body, and the resource. The bodies are inline in the contract.
func withNewE2MField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasLabels := props["metricLabels"]
				_, hasFields := props["metricFields"]
				_, hasLogs := props["logsQuery"]
				if hasLabels && hasFields && hasLogs {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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

// A field that a later contract adds to a TCO policy reaches the CRD type, the request, the sample,
// and the e2e test of each TCO policies kind with no new behavior-overrides line. The request holds
// the field in the policy wrapper. The released chart upgrade check stays as it is.
func TestTCOPoliciesNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	spec := withNewPolicyField(t, input.OpenAPI)
	for _, kind := range []string{"TCOLogsPolicies", "TCOTracesPolicies", "TCORumPolicies"} {
		t.Run(kind, func(t *testing.T) {
			entry := kindEntry{Kind: kind, Resource: "GetCompanyPoliciesResponse"}
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
			before, after := render(input.OpenAPI), render(spec)
			lower := strings.ToLower(kind)
			for name, content := range after {
				if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
					t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
				}
			}
			for _, want := range []struct{ file, text string }{
				{"api/coralogix/v1alpha1/" + lower + "_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
				{"api/coralogix/v1alpha1/" + lower + "_types.go", "+kubebuilder:validation:MaxLength=20"},
				{"api/coralogix/v1alpha1/" + lower + "_convert.go", ".ProbeNote = in.ProbeNote"},
				{"config/samples/v1alpha1/generated/" + lower + ".yaml", "probeNote:"},
				{"tests/e2e/" + lower + "_generated_test.go", `"probeNote":`},
			} {
				if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
					t.Errorf("%s has no %q", want.file, want.text)
				}
			}
			script := "scripts/upgrade-check-" + lower + ".sh"
			if string(after[script]) != string(before[script]) {
				t.Error("a new field changed the released chart upgrade check")
			}
		})
	}
}

// withNewPolicyField adds the optional field probeNote to the policy that a request writes and to
// the policy that Get returns.
func withNewPolicyField(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range []string{"CreateGenericPolicyRequest", "Policy"} {
		props := schemas[name].(map[string]any)["properties"].(map[string]any)
		props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A field that a later contract adds to the SLO reaches the CRD type, the conversion, the sample,
// and the e2e test with no new behavior-overrides line. Create and Replace share the body, so one
// conversion sends it in both. The groups, ignored, optional, and enum lines of the kind do not stop
// it, and the released chart upgrade check stays as it is.
func TestSLONewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "SLO", Resource: "Slo"}
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
		// The same loader as generation: it gives an exclusive bound its inclusive form.
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
	after := render(withNewSLOField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/slo_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/slo_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/slo.yaml", "probeNote:"},
		{"tests/e2e/slo_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/slo_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 1 {
		t.Errorf("want probeNote in the shared request body, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-slo.sh"]) != string(before["scripts/upgrade-check-slo.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewSLOField adds the optional field probeNote to every schema with the SLO properties: the
// Create body, the Replace body, and the resource. The bodies are inline in the contract.
func withNewSLOField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasTimeFrame := props["sloTimeFrame"]
				_, hasTarget := props["targetThresholdPercentage"]
				_, hasRequest := props["requestBasedMetricSli"]
				if hasTimeFrame && hasTarget && hasRequest {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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

// A new field of the S3 target, and a new field that Get nests in archiveSpec and the POST sends at
// the top level, flow into the type, the request, the sample, and the generated test with no
// behavior-overrides line. The chart 0.5 upgrade spec stays fixed.
func TestArchiveLogsTargetNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "ArchiveLogsTarget", Resource: "v2.Target"}
	dir := filepath.Join(input.ModuleRoot, entry.dir())
	render := func(spec []byte) map[string][]byte {
		t.Helper()
		// Each render reads the file again: reshape records its result in the file.
		file, err := readOverrides(filepath.Join(dir, "behavior-overrides.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		e2e, err := readE2E(filepath.Join(dir, e2eFileName))
		if err != nil {
			t.Fatal(err)
		}
		// The same loader as generation: it reshapes the contract of the singleton.
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
	after := render(withNewArchiveFields(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/archivelogstarget_types.go", "StorageClass *string `json:\"storageClass,omitempty\"`"},
		{"api/coralogix/v1alpha1/archivelogstarget_types.go", "CompressionLevel *int32 `json:\"compressionLevel,omitempty\"`"},
		{"api/coralogix/v1alpha1/archivelogstarget_convert.go", "out.StorageClass = in.StorageClass"},
		{"api/coralogix/v1alpha1/archivelogstarget_convert.go", "out.CompressionLevel = in.CompressionLevel"},
		{"config/samples/v1alpha1/generated/archivelogstarget.yaml", "storageClass:"},
		{"config/samples/v1alpha1/generated/archivelogstarget.yaml", "compressionLevel:"},
		{"tests/e2e/archivelogstarget_generated_test.go", `"storageClass":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if string(after["scripts/upgrade-check-archivelogstarget.sh"]) != string(before["scripts/upgrade-check-archivelogstarget.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewArchiveFields adds storageClass to the S3 target, which Get and the POST share, and
// compressionLevel to the archiveSpec of Get and to the top level of the POST body.
func withNewArchiveFields(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	props := func(name string) map[string]any {
		return schemas[name].(map[string]any)["properties"].(map[string]any)
	}
	props("S3TargetSpec")["storageClass"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "Storage class."}
	level := map[string]any{"type": "integer", "format": "int32", "x-coralogix-presence": true, "description": "Compression level."}
	props("ArchiveSpec")["compressionLevel"] = level
	set := doc["paths"].(map[string]any)["/logs/data-setup/v2"].(map[string]any)["post"].(map[string]any)
	body := set["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	body["properties"].(map[string]any)["compressionLevel"] = level
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A field that a later contract adds to the AI evaluation reaches the CRD type, both request
// conversions, the sample, and the e2e test with no new behavior-overrides line. The unsupported
// arms, the immutable fields, and updateMask: omit of the kind do not stop it.
func TestAIEvaluationNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "AIEvaluation", Resource: "AiEvaluation"}
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
	after := render(withNewAIEvaluationField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/aievaluation_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/aievaluation_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/aievaluation.yaml", "probeNote:"},
		{"tests/e2e/aievaluation_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/aievaluation_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update body, got %d conversions", n)
	}
}

// withNewAIEvaluationField adds the optional field probeNote to every schema with the evaluation
// properties: the Create body, the Update body, and the resource. The bodies are inline in the
// contract.
func withNewAIEvaluationField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasApplication := props["application"]
				_, hasThreshold := props["threshold"]
				_, hasConfig := props["config"]
				if hasApplication && hasThreshold && hasConfig {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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
		t.Fatalf("want probeNote in the Create body, the Update body, and the resource, added it %d times", added)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A new optional field of the family reaches the type, both family conversions, the nested update
// mask, the sample, and the e2e test with no new override line. The mask names it as family.probeNote.
func TestConfigurationGroupNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "ConfigurationGroup", Resource: "ConfigurationGroup"}
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
	after := render(withNewFamilyField(t, input.OpenAPI))
	for name, content := range after {
		for _, line := range removedLines(string(before[name]), string(content)) {
			// The mask paths are one line, so a new path changes it.
			if !strings.HasPrefix(strings.TrimSpace(line), "paths := []string{") {
				t.Errorf("%s: a new field changed an existing line: %s", name, line)
			}
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/configurationgroup_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/configurationgroup_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"api/coralogix/v1alpha1/configurationgroup_convert.go", `"family.probeNote"`},
		{"config/samples/v1alpha1/generated/configurationgroup.yaml", "probeNote:"},
		{"tests/e2e/configurationgroup_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/configurationgroup_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update family, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-configurationgroup.sh"]) != string(before["scripts/upgrade-check-configurationgroup.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewFamilyField adds probeNote to the family of the configuration group: the Create, Update,
// and resource components.
func withNewFamilyField(t *testing.T, spec []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range []string{"ConfigurationFamilyCreate", "ConfigurationFamilyUpdate", "ConfigurationFamily"} {
		props := schemas[name].(map[string]any)["properties"].(map[string]any)
		props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A new rule field reaches the generated CR, both requests, the sample, and the live test.
// The handwritten hooks only change the rules that the generated conversion built.
func TestQuotaAllocationRuleSetNewRuleFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		t.Fatal(err)
	}
	entry := kindEntry{Kind: "QuotaAllocationRuleSet", Resource: "QuotaAllocationEntityTypeRuleSet"}
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
	var doc map[string]any
	if err := yaml.Unmarshal(input.OpenAPI, &doc); err != nil {
		t.Fatal(err)
	}
	props := doc["components"].(map[string]any)["schemas"].(map[string]any)["QuotaAllocationEntityTypeRule"].(map[string]any)["properties"].(map[string]any)
	props["probeNote"] = map[string]any{"type": "string", "description": "A test note.", "maxLength": 20}
	changed, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	after := render(changed)
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed old lines: %s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/quotaallocationruleset_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/quotaallocationruleset_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"api/coralogix/v1alpha1/quotaallocationruleset_convert.go", "out.ProbeNote = in.ProbeNote"},
		{"config/samples/v1alpha1/generated/quotaallocationruleset.yaml", "probeNote:"},
		{"tests/e2e/quotaallocationruleset_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/quotaallocationruleset_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in Create and Update, got %d conversions", n)
	}
}

// A new field of Integration reaches the type, both conversions, the sample, and the e2e test with
// no new override line. The fields that the hooks set stay out of the conversions.
func TestIntegrationNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "Integration", Resource: "DeployedIntegrationInformation"}
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
	after := render(withComponentField(t, input.OpenAPI, "DeployedIntegrationInformation", "IntegrationMetadata"))
	for name, text := range after {
		if removed := removedLines(string(before[name]), string(text)); len(removed) != 0 {
			t.Errorf("%s: a new field changed lines that were there before:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/integration_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"config/samples/v1alpha1/generated/integration.yaml", "probeNote:"},
		{"tests/e2e/integration_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	convert := string(after["api/coralogix/v1alpha1/integration_convert.go"])
	if n := strings.Count(convert, "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update request, got %d conversions", n)
	}
	if strings.Contains(convert, ".Parameters =") {
		t.Error("the hooks set the parameters: the conversions must leave them out")
	}
	if strings.Count(convert, "out.IntegrationParameters = ") != 2 {
		t.Error("Create and Update must build the object that holds the parameters")
	}
}

// withComponentField adds an optional string probeNote to the named components of the contract.
func withComponentField(t *testing.T, spec []byte, components ...string) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range components {
		props, ok := schemas[name].(map[string]any)["properties"].(map[string]any)
		if !ok {
			t.Fatalf("the contract has no component %s with properties", name)
		}
		props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note."}
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A field that a later contract adds to Scope reaches the CRD type, both conversions, the sample,
// and the e2e test with no new behavior-overrides line. The released chart upgrade check stays as
// it is. The getByIDs, name, required, and description lines of the kind do not stop it.
func TestScopeNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "Scope", Resource: "v1.Scope"}
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
	after := render(withNewScopeField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/scope_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/scope_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/scope.yaml", "probeNote:"},
		{"tests/e2e/scope_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/scope_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update request, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-scope.sh"]) != string(before["scripts/upgrade-check-scope.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewScopeField adds the optional field probeNote to every schema with the Scope properties:
// the Create body, the Update body, and the resource.
func withNewScopeField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasDefault := props["defaultExpression"]
				_, hasFilters := props["filters"]
				if hasDefault && hasFilters {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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
	if added != 3 {
		t.Fatalf("want probeNote in the Create body, the Update body, and the resource, added it %d times", added)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A field that a later contract adds to ApiKey reaches the CRD type, both conversions, the sample,
// and the e2e test with no new behavior-overrides line. The released chart upgrade check stays as
// it is. The inline, requestAs, updateAs, updateDefault, itemName, and hook lines of the kind do not
// stop it.
func TestApiKeyNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "ApiKey", Resource: "KeyInfo"}
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
	after := render(withNewAPIKeyField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/apikey_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/apikey_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/apikey.yaml", "probeNote:"},
		{"tests/e2e/apikey_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/apikey_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update request, got %d conversions", n)
	}
	if string(after["scripts/upgrade-check-apikey.sh"]) != string(before["scripts/upgrade-check-apikey.sh"]) {
		t.Error("a new field changed the released chart upgrade check")
	}
}

// withNewAPIKeyField adds the optional field probeNote to the Create body, the Update body, and the
// resource KeyInfo.
func withNewAPIKeyField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasPermissions := props["keyPermissions"]
				_, hasOwner := props["owner"]
				_, hasHashed := props["hashed"]
				_, hasNewName := props["newName"]
				_, hasIsActive := props["isActive"]
				if (hasPermissions && (hasOwner || hasHashed)) || (hasNewName && hasIsActive) {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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
	if added != 3 {
		t.Fatalf("want probeNote in the Create body, the Update body, and the resource, added it %d times", added)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A field that a later contract adds to the custom evaluation config reaches the CRD type, both
// request conversions, the sample, and the e2e test with no new behavior-overrides line. The
// response nests it in config, which the kind inlines, and the hooks and getFromList do not stop it.
func TestAICustomEvaluationNewFieldWithoutOverrides(t *testing.T) {
	input, err := source.Resolve(".")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal(err)
		}
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "AICustomEvaluation", Resource: "CustomEvaluation"}
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
	after := render(withNewAICustomEvaluationField(t, input.OpenAPI))
	for name, content := range after {
		if removed := removedLines(string(before[name]), string(content)); len(removed) != 0 {
			t.Errorf("%s: a new field changed existing lines:\n%s", name, strings.Join(removed, "\n"))
		}
	}
	for _, want := range []struct{ file, text string }{
		{"api/coralogix/v1alpha1/aicustomevaluation_types.go", "ProbeNote *string `json:\"probeNote,omitempty\"`"},
		{"api/coralogix/v1alpha1/aicustomevaluation_types.go", "+kubebuilder:validation:MaxLength=20"},
		{"config/samples/v1alpha1/generated/aicustomevaluation.yaml", "probeNote:"},
		{"tests/e2e/aicustomevaluation_generated_test.go", `"probeNote":`},
	} {
		if !strings.Contains(normalizeSpace(string(after[want.file])), normalizeSpace(want.text)) {
			t.Errorf("%s has no %q", want.file, want.text)
		}
	}
	if n := strings.Count(string(after["api/coralogix/v1alpha1/aicustomevaluation_convert.go"]), "out.ProbeNote = in.ProbeNote"); n != 2 {
		t.Errorf("want probeNote in the Create and the Update body, got %d conversions", n)
	}
}

// withNewAICustomEvaluationField adds the optional field probeNote to every schema with the config
// properties: the Create body, the Update body, and the config of the response. The bodies are
// inline in the contract.
func withNewAICustomEvaluationField(t *testing.T, spec []byte) []byte {
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
			if props, ok := v["properties"].(map[string]any); ok {
				_, hasInstructions := props["instructions"]
				_, hasSafe := props["safe"]
				_, hasViolates := props["violates"]
				if hasInstructions && hasSafe && hasViolates {
					props["probeNote"] = map[string]any{"type": "string", "x-coralogix-presence": true, "description": "A note.", "maxLength": 20}
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
	if added != 3 {
		t.Fatalf("want probeNote in the Create body, the Update body, and the config, added it %d times", added)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
