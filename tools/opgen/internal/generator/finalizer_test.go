package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

func TestFinalizerKeyParse(t *testing.T) {
	const head = "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n"
	if _, err := overrides.Parse([]byte(head + "finalizer: thing.coralogix.com/finalizer\n")); err != nil {
		t.Fatalf("a qualified name in mode existing: %v", err)
	}
	if _, err := overrides.Parse([]byte("resource: Thing\nmode: new\nfinalizer: thing.coralogix.com/finalizer\n")); err == nil || !strings.Contains(err.Error(), "finalizer") {
		t.Fatalf("mode new must reject finalizer, got %v", err)
	}
	for _, bad := range []string{"no-slash", "UPPER/x", "thing.coralogix.com/", "/finalizer", "thing.coralogix.com/a b"} {
		if _, err := overrides.Parse([]byte(head + "finalizer: " + bad + "\n")); err == nil || !strings.Contains(err.Error(), "finalizer") {
			t.Errorf("finalizer %q must be rejected, got %v", bad, err)
		}
	}
}

func TestFinalizerKeyChangesTheGeneratedName(t *testing.T) {
	root, input := syntheticRoot(t)
	path := filepath.Join(root, kindsDir, "thing", "behavior-overrides.yaml")
	write := func(finalizer string) {
		t.Helper()
		data := "resource: Thing\nmode: existing\nvalidators: {inferred: true}\n" + finalizer
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The generated name is the default. A line that states it changes nothing.
	write("finalizer: thing.coralogix.com/finalizer\n")
	wantCode(t, generateFromInput(input, root, root), codeOverrideUnused)

	write("finalizer: released-thing.coralogix.com/finalizer\n")
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	controller, err := os.ReadFile(filepath.Join(root, controllerDir, "thing_controller.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(controller), `return "released-thing.coralogix.com/finalizer"`) {
		t.Fatalf("FinalizerName does not return the override:\n%s", controller)
	}
}
