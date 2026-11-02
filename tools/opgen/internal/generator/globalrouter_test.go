package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"
)

// The pinned GlobalRouter contract accepts the G6 reference names and validation and the G14
// skip line. The kind stays handwritten until its released CRD is compared with generated output.
func TestGlobalRouterGapOverrides(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := source.Resolve(root)
	if err != nil {
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	openAPI := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(openAPI, pinned.OpenAPI, 0o644); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join("testdata", "globalrouter")
	if err := Check(CheckOptions{Resource: "GlobalRouter", OpenAPIPath: openAPI,
		OverridesPath: filepath.Join(fixture, "behavior-overrides.yaml"), E2EPath: filepath.Join(fixture, "e2e.yaml")}); err != nil {
		t.Fatal(err)
	}
}
