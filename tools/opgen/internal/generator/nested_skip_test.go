package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
)

const thingSDKFile = "go/openapi/gen/things_service/things.go"

// skipRoot is the synthetic root with an edited contract and an edited stand-in SDK.
func skipRoot(t *testing.T, editAPI, editSDK func(string) string) (string, source.Input) {
	t.Helper()
	root, input := syntheticRoot(t)
	if editAPI != nil {
		input.OpenAPI = []byte(editAPI(string(input.OpenAPI)))
	}
	if editSDK == nil {
		return root, input
	}
	sdkDir := filepath.Join(t.TempDir(), "sdk")
	if err := copyTree(filepath.Join("testdata", "sdk"), sdkDir); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sdkDir, thingSDKFile)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(editSDK(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(root, "go.mod")
	mod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modPath, []byte(strings.Replace(string(mod), input.SDKDir, sdkDir, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	input.SDKDir = sdkDir
	return root, input
}

func writeThingOverrides(t *testing.T, root, types string) {
	t.Helper()
	data := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n" + types
	if err := os.WriteFile(filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readGenerated(t *testing.T, root, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A nested optional scalar leaves the CR, the sample, and both requests.
func TestNestedSkipRemovesAnOptionalField(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  ThingItem:\n    fields:\n      weight: {skip: true}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if strings.Contains(types, "Weight") || !strings.Contains(types, "type ThingItem struct") {
		t.Fatalf("the skipped field is still in the CR, or the component is gone:\n%s", types)
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	if strings.Contains(convert, "Weight") {
		t.Fatal("a request still sends the skipped field")
	}
}

// The E2M shape: an empty object arm of an optional oneOf. The shared module drops the arm, and
// the CR keeps only the other arms.
func TestNestedSkipRemovesAnEmptyOneOfArm(t *testing.T) {
	editAPI := func(s string) string {
		s = replaceOnce(t, s, "        queue: {$ref: '#/components/schemas/QueueThingConfig'}\n      oneOf:\n        - required: [http]\n        - required: [queue]\n",
			"        queue: {$ref: '#/components/schemas/QueueThingConfig'}\n        none: {type: object}\n      oneOf:\n        - required: [http]\n        - required: [queue]\n        - required: [none]\n        - not:\n            anyOf:\n              - required: [http]\n              - required: [queue]\n              - required: [none]\n")
		return s
	}
	editSDK := func(s string) string {
		return replaceOnce(t, s, "	Queue *QueueThingConfig `json:\"queue,omitempty\"`\n", "	Queue *QueueThingConfig `json:\"queue,omitempty\"`\n	None  map[string]interface{} `json:\"none,omitempty\"`\n")
	}
	root, input := skipRoot(t, editAPI, editSDK)
	writeThingOverrides(t, root, "types:\n  ThingConfig:\n    fields:\n      none: {skip: true}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/thing_types.go")
	if strings.Contains(types, "None") {
		t.Fatalf("the skipped arm is still in the CR:\n%s", types)
	}
	if !regexp.MustCompile(`Http\s+\*ThingHttpThingConfig`).MatchString(types) || !regexp.MustCompile(`Queue\s+\*ThingQueueThingConfig`).MatchString(types) {
		t.Fatal("the other arms are gone")
	}
	if strings.Contains(readGenerated(t, root, apiDir+"/thing_convert.go"), "None") {
		t.Fatal("a request still sends the skipped arm")
	}
	// A line that names the skipped arm in another key stops the run. It must not change the output
	// without a word.
	for name, types := range map[string]string{
		"a group": "types:\n  ThingConfig:\n    groups:\n      endpoint: {fields: [http, queue, none]}\n    fields:\n      none: {skip: true}\n",
	} {
		writeThingOverrides(t, root, types)
		err := generateFromInput(input, root, root)
		if err == nil {
			t.Errorf("%s: a line names the skipped arm, but the run did not stop", name)
		}
		if err != nil && !strings.Contains(err.Error(), "none") {
			t.Errorf("%s: the error does not name the skipped arm: %v", name, err)
		}
	}
}

// A request that requires a nested field must send it, so the field cannot be skipped.
func TestNestedSkipRejectsARequiredField(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  ThingItem:\n    fields:\n      name: {skip: true}\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}

// A line stays only while a request sends the field. The server sets revision: it is in the
// resource and in no request.
func TestNestedSkipRejectsAFieldThatNoRequestSends(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  ThingSettings:\n    fields:\n      revision: {skip: true}\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideUnused)
	// A component that has no such property.
	writeThingOverrides(t, root, "types:\n  ThingItem:\n    fields:\n      missing: {skip: true}\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideUnused)
}

// Create and Update name their own components for an object that the resource names ThingSettings.
// The line names the component of the resource, and it changes both requests.
func TestNestedSkipNamesTheComponentOfTheResource(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  ThingSettings:\n    fields:\n      retries: {skip: true}\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	convert := readGenerated(t, root, apiDir+"/thing_convert.go")
	if strings.Contains(convert, "Retries") {
		t.Fatal("a request still sends the skipped field")
	}
	// The request components are merged into ThingSettings, so they are not names of a line.
	writeThingOverrides(t, root, "types:\n  ThingSettingsUpdate:\n    fields:\n      retries: {skip: true}\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideUnused)
}

// A request component can require the field under its own name. The merged type knows it.
func TestNestedSkipRejectsAFieldThatARequestComponentRequires(t *testing.T) {
	root, input := skipRoot(t, nil, nil)
	writeThingOverrides(t, root, "types:\n  ThingSettings:\n    fields:\n      mode: {skip: true}\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideInvalid)
}
