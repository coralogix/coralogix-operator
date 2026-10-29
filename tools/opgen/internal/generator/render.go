package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"text/template"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{"lowerFirst": lowerFirst}).ParseFS(templateFS, "templates/*.tmpl"))

// apiDir holds the generated types, relative to the operator root.
const apiDir = "api/coralogix/v1alpha1"

// renderKind returns the files of one kind by path.
func renderKind(d *kindData) (map[string][]byte, error) {
	d.Header = version.GoHeader
	src, err := execute("types.go.tmpl", d)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{apiDir + "/" + d.Lower + "_types.go": src}, nil
}

func execute(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return formatSource(name, buf.Bytes())
}

// formatSource removes the unused imports and formats the file. A template lists every import
// that the file can use.
func formatSource(name string, src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse generated %s: %w\n%s", name, err, src)
	}
	type namedImport struct{ name, path string }
	var unused []namedImport
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name != "." && !astutil.UsesImport(f, p) {
			unused = append(unused, namedImport{name, p})
		}
	}
	for _, imp := range unused {
		astutil.DeleteNamedImport(fset, f, imp.name, imp.path)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return nil, fmt.Errorf("format generated %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
