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
		`spec = strings.ReplaceAll(spec, "@{env.THING_DESCRIPTION}", os.Getenv("THING_DESCRIPTION"))`,
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
	for _, want := range []string{`: "${THING_DESCRIPTION:?THING_DESCRIPTION is required}"`, `-e "s|@{env.THING_DESCRIPTION}|${THING_DESCRIPTION}|g"`} {
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
