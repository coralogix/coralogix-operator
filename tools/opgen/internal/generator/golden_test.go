package generator

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/coralogix/coralogix-management-sdk/iac/source"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

var updateGolden = flag.Bool("update", false, "replace the committed golden output")

const operatorModule = "github.com/coralogix/coralogix-operator/v2"

// syntheticRoot copies the seed operator root of testdata into a new directory. Its go.mod loads
// the stand-in SDK, so generation needs no network and no real SDK.
func syntheticRoot(t *testing.T) (string, source.Input) {
	t.Helper()
	root := t.TempDir()
	if err := copyTree(filepath.Join("testdata", "root"), root); err != nil {
		t.Fatal(err)
	}
	sdkDir, err := filepath.Abs(filepath.Join("testdata", "sdk"))
	if err != nil {
		t.Fatal(err)
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.26.0\n\nrequire %s v0.0.0\n\nreplace %s => %s\n", operatorModule, source.SDKModule, source.SDKModule, sdkDir)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	openAPI, err := os.ReadFile(filepath.Join("testdata", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return root, source.Input{ModuleRoot: root, Module: operatorModule, SDKDir: sdkDir, SDKModule: source.SDKModule, OpenAPI: openAPI}
}

// The golden output of the synthetic kinds: every generated file and every marked block. A second
// run writes the same files.
func TestGoldenOutput(t *testing.T) {
	root, input := syntheticRoot(t)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	first := readTree(t, root)
	if err := generateFromInput(input, root, root); err != nil {
		t.Fatal(err)
	}
	if diff := compareTrees(first, readTree(t, root)); diff != "" {
		t.Fatalf("a second run changed the output:\n%s", diff)
	}
	golden := filepath.Join("testdata", "golden")
	if *updateGolden {
		if err := os.RemoveAll(golden); err != nil {
			t.Fatal(err)
		}
		if err := writeTree(golden, first); err != nil {
			t.Fatal(err)
		}
	}
	if diff := compareTrees(readTree(t, golden), first); diff != "" {
		t.Fatalf("golden output differs; run go test ./internal/generator -run TestGoldenOutput -update:\n%s", diff)
	}
	for path, data := range first {
		if strings.HasSuffix(path, ".go") && !bytes.HasPrefix(data, []byte(version.GoHeader+"\n")) {
			t.Errorf("%s lacks the exact generator header", path)
		}
	}
}

// readTree reads every file under dir except go.mod, by relative path.
func readTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "go.mod" {
			return err
		}
		data, err := os.ReadFile(path)
		out[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeTree(dir string, files map[string][]byte) error {
	for rel, data := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func compareTrees(want, got map[string][]byte) string {
	var out []string
	for path, data := range got {
		prior, ok := want[path]
		switch {
		case !ok:
			out = append(out, "added "+path)
		case !bytes.Equal(prior, data):
			out = append(out, "changed "+path+":\n"+firstDifference(string(prior), string(data)))
		}
	}
	for path := range want {
		if _, ok := got[path]; !ok {
			out = append(out, "removed "+path)
		}
	}
	slices.Sort(out)
	return strings.Join(out, "\n")
}

func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("  line %d\n  want: %s\n  got:  %s", i+1, wl, gl)
		}
	}
	return ""
}
