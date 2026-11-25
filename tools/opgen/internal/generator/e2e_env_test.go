package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeThingE2E(t *testing.T, root, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, kindsDir, "thing", e2eFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A value of the account that the public repository cannot hold comes from the environment. The
// live tests expand it before they decode a spec, skip without it, and fail without it in CI. The
// upgrade check requires it and expands it with sed.
func TestE2EEnvValues(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "env: [THING_DESCRIPTION]\nvalues:\n  destinations: [https://example.com]\n  description: e2e-description\n"+
		"substitutions:\n  e2e-description: '@{env.THING_DESCRIPTION}'\n"+
		"upgrade:\n  from: '1.0'\n  spec: {name: '@{run}', description: '@{env.THING_DESCRIPTION}'}\n  field: description\n  value: updated\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	test := readGenerated(t, root, "tests/e2e/thing_generated_test.go")
	for _, want := range []string{
		"\t\"os\"\n",
		"func requireThingEnv() {",
		`[]string{"THING_DESCRIPTION"}`,
		`quoted, err := json.Marshal(os.Getenv(name))`,
		`spec = strings.ReplaceAll(spec, "@{env."+name+"}", string(quoted[1:len(quoted)-1]))`,
		"spec = expandThingEnv(spec)",
		`strings.ReplaceAll(spec, "e2e-description", strings.ReplaceAll("@{env.THING_DESCRIPTION}", "@{run}", name))`,
	} {
		if !strings.Contains(test, want) {
			t.Errorf("the e2e test has no %q", want)
		}
	}
	if n := strings.Count(test, "\t\trequireThingEnv()\n"); n == 0 {
		t.Error("BeforeAll does not check the variables")
	}
	script := readGenerated(t, root, "scripts/upgrade-check-thing.sh")
	for _, want := range []string{`: "${THING_DESCRIPTION:?THING_DESCRIPTION is required}"`, `jq --arg value "${THING_DESCRIPTION}" 'walk(if type == "string" then gsub("@\\{env\\.THING_DESCRIPTION\\}"; $value) else . end)'`} {
		if !strings.Contains(script, want) {
			t.Errorf("the upgrade check has no %q", want)
		}
	}
	if strings.Contains(readGenerated(t, root, "config/samples/v1alpha1/generated/thing.yaml"), "@{env") {
		t.Error("the sample names an environment variable")
	}
}

func TestE2EEnvRejectsWrongNames(t *testing.T) {
	for name, tc := range map[string]struct{ e2e, code string }{
		"an unlisted variable": {"values:\n  destinations: [https://example.com]\nsubstitutions:\n  sample-description: '@{env.OTHER}'\n", codeE2EInvalid},
		"a value":              {"env: [A]\nvalues:\n  destinations: [https://example.com]\n  description: '@{env.A}'\n", codeE2EInvalid},
		"an unused variable":   {"env: [THING_DESCRIPTION]\nvalues:\n  destinations: [https://example.com]\n", codeE2EUnused},
		"a lower-case name":    {"env: [thing]\nvalues:\n  destinations: [https://example.com]\nsubstitutions:\n  sample-description: '@{env.thing}'\n", codeE2EInvalid},
		"a name listed twice":  {"env: [A, A]\nvalues:\n  destinations: [https://example.com]\nsubstitutions:\n  sample-description: '@{env.A}'\n", codeE2EInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			root, input := syntheticRoot(t)
			writeThingE2E(t, root, tc.e2e)
			wantCode(t, generateFromInput(input, root, root), tc.code)
		})
	}
}

// The upgrade check writes a variable into a string. A value that is exactly one variable and that
// an integer field holds becomes a number, since the CRD rejects a string there. A Quantity takes a
// string, and a string field keeps it.
func TestE2EEnvValueOfAnIntegerField(t *testing.T) {
	root, input := syntheticRoot(t)
	writeThingE2E(t, root, "env: [THING_NUMBER]\nvalues:\n  destinations: [https://example.com]\n"+
		"upgrade:\n  from: '1.0'\n  spec:\n    name: '@{run}'\n    quota: '@{env.THING_NUMBER}'\n    ratio: '@{env.THING_NUMBER}'\n"+
		"    description: 'n-@{env.THING_NUMBER}'\n    settings: {mode: fast, retries: '@{env.THING_NUMBER}'}\n  field: description\n  value: updated\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	script := readGenerated(t, root, "scripts/upgrade-check-thing.sh")
	var paths []string
	for _, line := range strings.Split(script, "\n") {
		if _, rest, ok := strings.Cut(line, "--argjson path '"); ok {
			path, _, _ := strings.Cut(rest, "'")
			paths = append(paths, path)
		}
	}
	if want := []string{`["quota"]`, `["settings","retries"]`}; strings.Join(paths, " ") != strings.Join(want, " ") {
		t.Fatalf("number paths = %v, want %v", paths, want)
	}
	if !strings.Contains(script, `'setpath($path; getpath($path) | tonumber)'`) {
		t.Error("the upgrade check does not turn the values into numbers")
	}
}
