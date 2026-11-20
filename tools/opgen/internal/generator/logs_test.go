package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hideBodiesInLogs keeps the Create and Update bodies and responses out of the logs. A kind without
// the key logs them, as before.
func TestHideBodiesInLogs(t *testing.T) {
	contract, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, resource, override, file string
		lines                          []string
	}{
		{
			name: "a PATCH resource", resource: "MaskThing", override: nestedMaskOverride + "hideBodiesInLogs: true\n",
			file: "internal/controller/coralogix/v1alpha1/maskthing_controller.go",
			lines: []string{
				`log.Info("Creating remote MaskThing")`,
				`log.Info("Updating remote MaskThing", "id", id)`,
				`log.Info("Remote MaskThing updated", "id", id)`,
				"_, httpResp, err := r.Client.",
			},
		},
		{
			name: "an overwrite resource", resource: "GetThingPoliciesResponse",
			override: readFile(t, filepath.Join("testdata", "root", "tools", "opgen", "kinds", "thingpolicies", "behavior-overrides.yaml")),
			file:     "internal/controller/coralogix/v1alpha1/getthingpoliciesresponse_controller.go",
			lines:    []string{`log.Info("Overwriting remote GetThingPoliciesResponse")`, `log.Info("Remote GetThingPoliciesResponse overwritten")`},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files := checkLifecycleFixture(t, tt.resource, string(contract), tt.override)
			controller := string(files[tt.file])
			if controller == "" {
				t.Fatalf("no %s; files: %v", tt.file, keys(files))
			}
			for _, banned := range []string{"FormatJSON", "/internal/utils\""} {
				if strings.Contains(controller, banned) {
					t.Errorf("the controller has %s", banned)
				}
			}
			for _, line := range tt.lines {
				if !strings.Contains(controller, line) {
					t.Errorf("the controller omits %s", line)
				}
			}
		})
	}
	t.Run("a new kind", func(t *testing.T) {
		if err := checkMask(t, "MaskThing", "resource: MaskThing\nmode: new\nhideBodiesInLogs: true\n", nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("the default", func(t *testing.T) {
		files := checkLifecycleFixture(t, "MaskThing", string(contract), nestedMaskOverride)
		controller := string(files["internal/controller/coralogix/v1alpha1/maskthing_controller.go"])
		for _, line := range []string{`"MaskThing", utils.FormatJSON(body)`, `"response", utils.FormatJSON(resp)`} {
			if !strings.Contains(controller, line) {
				t.Errorf("the controller omits %s", line)
			}
		}
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func keys(files map[string][]byte) []string {
	out := make([]string, 0, len(files))
	for name := range files {
		out = append(out, name)
	}
	return out
}
