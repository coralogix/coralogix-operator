package generator

import (
	"strings"
	"testing"
)

// A value that the API keeps unique across the account, such as a custom id, gets a new UUID for
// each CR that a test builds. A run that stops halfway then leaves no object that blocks the next
// run. The sample keeps its fixed value.
func TestE2EUUIDSubstitution(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "values:\n  destinations: [https://example.com]\n  description: 00000000-0000-4000-8000-000000000001\n"+
		"substitutions:\n  00000000-0000-4000-8000-000000000001: '@{uuid}'\n  sample-name: '@{run}-name'\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"tests/e2e/thing_generated_test.go", "internal/controller/coralogix/v1alpha1/thing_coverage_test.go"} {
		test := readGenerated(t, root, path)
		for _, want := range []string{
			"\t\"github.com/google/uuid\"\n",
			`strings.ReplaceAll(spec, "00000000-0000-4000-8000-000000000001", strings.ReplaceAll(strings.ReplaceAll("@{uuid}", "@{run}", name), "@{uuid}", uuid.NewString()))`,
			`strings.ReplaceAll(spec, "sample-name", strings.ReplaceAll("@{run}-name", "@{run}", name))`,
		} {
			if !strings.Contains(test, want) {
				t.Errorf("%s has no %q", path, want)
			}
		}
	}
	sample := readGenerated(t, root, "config/samples/v1alpha1/generated/thing.yaml")
	if !strings.Contains(sample, "00000000-0000-4000-8000-000000000001") || strings.Contains(sample, "@{uuid}") {
		t.Errorf("the sample does not keep its fixed value:\n%s", sample)
	}
}

// A test without @{uuid} does not import the uuid package.
func TestE2ENoUUIDNoImport(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "values:\n  destinations: [https://example.com]\nsubstitutions:\n  sample-name: '@{run}-name'\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readGenerated(t, root, "tests/e2e/thing_generated_test.go"), "google/uuid") {
		t.Error("the e2e test imports the uuid package without @{uuid}")
	}
}

// Only a substitution makes a new UUID. A value reaches the sample, and the fixtures and the upgrade
// check apply no substitution.
func TestE2EUUIDOutsideSubstitutions(t *testing.T) {
	for name, e2e := range map[string]string{
		"a value":         "values:\n  destinations: [https://example.com]\n  description: '@{uuid}'\n",
		"a fixture":       "values:\n  destinations: [https://example.com]\nfixtures:\n  - {kind: Thing, suffix: -other, statusField: ID, spec: {name: '@{uuid}'}}\n",
		"an upgrade spec": "values:\n  destinations: [https://example.com]\nupgrade:\n  from: '1.0'\n  spec: {name: '@{uuid}'}\n  field: description\n  value: updated\n",
	} {
		t.Run(name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			writeThingE2E(t, root, e2e)
			wantCode(t, generateFromInput(input, root, root), codeE2EInvalid)
		})
	}
}
