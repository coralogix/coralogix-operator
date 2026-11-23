package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
)

// This test keeps the behavior-overrides file of the released Connector true to the pinned
// contract: it generates the Connector from the real SDK, in a root that has no handwritten files,
// and checks the shapes of the released CRD.
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
	overrides, err := os.ReadFile(filepath.Join(operatorRoot, kindsDir, "connector", "behavior-overrides.yaml"))
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

	// The released type enum has every API value except IBM_EVENT_NOTIFICATIONS, in the released
	// spelling and the released order.
	enums := regexp.MustCompile(`\+kubebuilder:validation:Enum=([^\n]+)`).FindAllStringSubmatch(types, -1)
	var got []string
	for _, m := range enums {
		got = append(got, m[1])
	}
	want := []string{
		"slack;genericHttps;pagerDuty;pagerDutyIncidents;email;serviceNow;microsoftTeams;eventBridge;incidentIo",
		"alerts;cases",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the CRD enums are %q, the released CRD has %q", got, want)
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
