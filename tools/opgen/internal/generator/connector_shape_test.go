package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
)

// The released Connector is not a listed kind yet, because its handwritten files still exist and
// the generator would stop on them. Its behavior-overrides file waits in testdata/connector until
// the migration moves it to kinds/connector. This test keeps the file true to the pinned contract: it
// generates the Connector from the real SDK, in a root that has no handwritten files, and checks
// the shapes of the released CRD.
func TestReleasedConnectorShapes(t *testing.T) {
	if testing.Short() {
		t.Skip("loads the pinned SDK")
	}
	operatorRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	input, err := source.Resolve(operatorRoot)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := copyTree(filepath.Join("testdata", "root"), root); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, kindsDir)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, kindsDir, "connector"), 0o755); err != nil {
		t.Fatal(err)
	}
	overrides, err := os.ReadFile(filepath.Join("testdata", "connector", "behavior-overrides.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, kindsDir, "connector", "behavior-overrides.yaml"), overrides, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, kindsFile), []byte("kinds:\n  - kind: Connector\n    resource: Connector\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generateFromInput(input, root, input.ModuleRoot); err != nil {
		t.Fatal(err)
	}
	types := readGenerated(t, root, apiDir+"/connector_types.go")

	// The released type enum has every API value except IBM_EVENT_NOTIFICATIONS, in the released spelling.
	enum := regexp.MustCompile(`\+kubebuilder:validation:Enum=([^\n]+)`).FindAllStringSubmatch(types, -1)
	var got [][]string
	for _, m := range enum {
		got = append(got, strings.Split(m[1], ";"))
	}
	want := map[string]bool{"slack": true, "genericHttps": true, "pagerDuty": true, "pagerDutyIncidents": true, "email": true, "serviceNow": true, "microsoftTeams": true, "eventBridge": true, "incidentIo": true, "alerts": true, "cases": true}
	seen := 0
	for _, values := range got {
		for _, value := range values {
			if !want[value] {
				t.Errorf("the CRD accepts %q, which the released CRD does not", value)
			}
			seen++
		}
	}
	if seen != len(want) {
		t.Errorf("the CRD has %d enum values, the released CRD has %d: %v", seen, len(want), got)
	}

	// The released rule of a connector config field, word for word: the schema check hashes the message.
	const rule = `rule="(has(self.value) ? 1 : 0) + (has(self.secretKeyRef) ? 1 : 0) == 1",message="Exactly one of value or secretKeyRef must be set"`
	if !strings.Contains(types, rule) {
		t.Errorf("the released rule of the config field is missing:\n%s", types)
	}
	if !strings.Contains(types, "SecretKeyRef *corev1.SecretKeySelector `json:\"secretKeyRef,omitempty\"`") {
		t.Error("the config field has no secretKeyRef")
	}
	// The set fields of the API are the lists of the released CR.
	for _, list := range []string{"Fields []ConnectorNotificationCenterConnectorConfigField `json:\"fields\"`", "Fields []ConnectorTemplatedConnectorConfigField `json:\"fields,omitempty\"`"} {
		if !strings.Contains(types, list) {
			t.Errorf("missing the released list %s", list)
		}
	}
	if strings.Contains(types, "listType=set") {
		t.Error("a list of the released CR is a set")
	}
}
