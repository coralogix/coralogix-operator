package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/issue"
	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/source"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// A kind without an overrides file follows the contract alone. These tests keep that true for the
// shapes of the Product Analytics kinds: a singleton with only Get and Update, a top-level update
// mask with a oneOf behind it, and a set of objects.
func TestNewKindShapesOfProductAnalytics(t *testing.T) {
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
	for _, resource := range []string{"ProductAnalyticsSettings", "SavedFunnel", "SavedPathway", "SavedHeatmap"} {
		t.Run(resource, func(t *testing.T) {
			if err := Check(CheckOptions{Resource: resource, OpenAPIPath: openAPI}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAlwaysExistsSingletonNeedsNoOverrides(t *testing.T) {
	if err := checkLifecycle(&model.Resource{Singleton: true, AlwaysExists: true}, nil); err != nil {
		t.Fatalf("a singleton with only Get and Update must be supported for a new kind: %v", err)
	}
}

func TestTopLevelMaskAcceptsAWholeOneOfOnlyForANewKind(t *testing.T) {
	report := issue.Report{
		{Code: "UPDATE_MASK_NESTED_ONEOF_UNSUPPORTED"},
		{Code: "UPDATE_MASK_CONTRACT_INVALID"},
	}
	got := maskIssues(report, nil)
	if len(got) != 1 || got[0].Code != "UPDATE_MASK_CONTRACT_INVALID" {
		t.Errorf("a new kind must drop only the nested oneOf issue: %+v", got)
	}
	existing, err := overrides.Parse([]byte("resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := maskIssues(report, existing); len(got) != 2 {
		t.Errorf("a released kind must keep both issues until it states updateMask: omit: %+v", got)
	}
}

func TestPluralKeepsAPluralNoun(t *testing.T) {
	for kind, want := range map[string]string{
		"ProductAnalyticsSettings": "productanalyticssettings",
		"TCOLogsPolicies":          "tcologspolicies",
		"SavedFunnel":              "savedfunnels",
		"Scope":                    "scopes",
	} {
		if got := plural(kind); got != want {
			t.Errorf("plural(%s) = %s, want %s", kind, got, want)
		}
	}
}

// The Update of the settings singleton is a PATCH, and the API requires update_mask. The generated
// controller must send it, or every reconcile fails with 400.
func TestAlwaysExistsSingletonSendsItsUpdateMask(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := source.Resolve(root)
	if err != nil {
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	entry := kindEntry{Kind: "ProductAnalyticsSettings", Resource: "ProductAnalyticsSettings"}
	doc, err := (&documents{raw: pinned.OpenAPI}).forKind(entry, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := validate(doc, entry, source.SDKModule, operatorModule, &e2eFile{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := renderOne(v, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	controller := string(files["internal/controller/coralogix/v1alpha1/productanalyticssettings_controller.go"])
	if want := ".UpdateMask(mask)"; !strings.Contains(controller, want) {
		t.Errorf("the controller must send the update mask: it misses %q", want)
	}
}

// The API of a saved view answers 412 "Product Analytics is not enabled for this team", and the
// settings singleton has the enabled field of that feature. The e2e tests of the saved views must
// turn it on first, and the test of the settings defines the helper. Nothing in a kind file says so.
func TestFeatureSwitchIsLinkedFromTheContract(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := source.Resolve(root)
	if err != nil {
		t.Skipf("the pinned SDK is not in the module cache: %v", err)
	}
	var all []*validated
	for _, resource := range []string{"SavedFunnel", "ProductAnalyticsSettings"} {
		entry := kindEntry{Kind: resource, Resource: resource}
		doc, err := (&documents{raw: pinned.OpenAPI}).forKind(entry, nil)
		if err != nil {
			t.Fatal(err)
		}
		v, err := validate(doc, entry, source.SDKModule, operatorModule, &e2eFile{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, v)
	}
	out, err := renderAll(all, source.SDKModule, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	funnel := string(out.files["tests/e2e/savedfunnel_generated_test.go"])
	settings := string(out.files["tests/e2e/productanalyticssettings_generated_test.go"])
	if !strings.Contains(funnel, "ensureProductAnalyticsSettingsEnabledGenerated(ctx)") {
		t.Error("the saved funnel test must turn Product Analytics on before its specs")
	}
	if !strings.Contains(settings, "func ensureProductAnalyticsSettingsEnabledGenerated(") {
		t.Error("the settings test must define the helper")
	}
	// A kind that nothing requires has no helper, or the unused linter fails the generated file.
	alone, err := renderAll(all[1:], source.SDKModule, operatorModule)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(alone.files["tests/e2e/productanalyticssettings_generated_test.go"]), "EnabledGenerated(") {
		t.Error("the settings test must not define a helper that no kind calls")
	}
}
