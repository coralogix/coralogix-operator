package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/issue"
)

// nameCollisions reports each top-level name of a generated Go file that a handwritten file of the
// same package declares, or that another generated file of the same package declares, for example
// two kinds that both make a type ViewFolder. The operator would not compile, so the run stops
// before it writes.
func nameCollisions(root string, files map[string][]byte) issue.Report {
	var report issue.Report
	handwritten := map[string]map[string]string{} // directory → name → file
	generated := map[string]map[string]string{}   // directory and package → name → file
	for _, path := range sortedKeys(files) {
		if filepath.Ext(path) != ".go" {
			continue
		}
		dir := filepath.Dir(path)
		if _, ok := handwritten[dir]; !ok {
			handwritten[dir] = declaredNames(filepath.Join(root, dir))
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, files[path], 0)
		if err != nil {
			continue // rendering already parsed every file
		}
		pkg := dir + " " + f.Name.Name
		if _, ok := generated[pkg]; !ok {
			generated[pkg] = map[string]string{}
		}
		for _, name := range topLevelNames(f) {
			if other, ok := handwritten[dir][name]; ok {
				report = append(report, issue.Issue{Code: codeNameCollision, Location: path,
					Message:     fmt.Sprintf("the generated %s is declared in the handwritten file %s", name, other),
					Remediation: "Rename the handwritten declaration, or the component or field in the API contract."})
			}
			if other, ok := generated[pkg][name]; ok {
				report = append(report, issue.Issue{Code: codeNameCollision, Location: path,
					Message:     fmt.Sprintf("the generated %s is also declared in the generated file %s", name, other),
					Remediation: "Rename the component or field in the API contract of one of the two kinds."})
				continue
			}
			generated[pkg][name] = path
		}
	}
	return report
}

// declaredNames returns the file of each top-level name of the handwritten Go files in dir. A file
// of an external test package does not count.
func declaredNames(dir string) map[string]string {
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		rel := filepath.Join(filepath.Base(dir), e.Name())
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil || ownedData(data) {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), data, parser.SkipObjectResolution)
		if err != nil || strings.HasSuffix(f.Name.Name, "_test") {
			continue
		}
		for _, name := range topLevelNames(f) {
			out[name] = rel
		}
	}
	return out
}

// topLevelNames lists the package-level names of a file: types, functions, variables, and
// constants. Methods and init functions are not package-level names.
func topLevelNames(f *ast.File) []string {
	var out []string
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name != "init" {
				out = append(out, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					out = append(out, s.Name.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.Name != "_" {
							out = append(out, n.Name)
						}
					}
				}
			}
		}
	}
	return out
}
