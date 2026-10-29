package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v4"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{"lowerFirst": lowerFirst}).ParseFS(templateFS, "templates/*.tmpl"))

// Paths of the generated files, relative to the operator root.
const (
	apiDir        = "api/coralogix/v1alpha1"
	controllerDir = "internal/controller/coralogix/v1alpha1"
	registryFile  = controllerDir + "/generated_registry.go"
	kindsGoFile   = "internal/utils/generated_kinds.go"
	samplesDir    = "config/samples/v1alpha1/generated"
	e2eDir        = "tests/e2e"
)

// Handwritten files with a block of generated lines, relative to the operator root.
const (
	projectFile      = "PROJECT"
	crdKustomization = "config/crd/kustomization.yaml"
	helmClusterRole  = "charts/coralogix-operator/templates/cluster_role.yaml"
)

// renderKind returns the files of one kind by path.
func renderKind(d *kindData, full *omap) (map[string][]byte, error) {
	d.Header = version.GoHeader
	files := map[string][]byte{}
	for _, f := range []struct{ path, tmpl string }{
		{apiDir + "/" + d.Lower + "_types.go", "types.go.tmpl"},
		{apiDir + "/" + d.Lower + "_convert.go", "convert.go.tmpl"},
		{controllerDir + "/" + d.Lower + "_controller.go", "controller.go.tmpl"},
		{e2eDir + "/" + d.Lower + "_generated_test.go", "e2e_test.go.tmpl"},
	} {
		src, err := execute(f.tmpl, d)
		if err != nil {
			return nil, err
		}
		files[f.path] = src
	}
	sample, err := renderSample(d, full)
	if err != nil {
		return nil, err
	}
	files[samplesDir+"/"+d.Lower+".yaml"] = sample
	return files, nil
}

// sharedData is the data of the files that list every kind.
type sharedData struct {
	Header    string
	Module    string
	CxsdkPath string
	Kinds     []*kindData
}

// renderShared returns the files and the blocks that list every kind.
func renderShared(module, cxsdk string, kinds []*kindData) (output, error) {
	data := &sharedData{Header: version.GoHeader, Module: module, CxsdkPath: cxsdk, Kinds: kinds}
	out := output{files: map[string][]byte{}, blocks: map[string]string{}}
	for path, tmpl := range map[string]string{registryFile: "registry.go.tmpl", kindsGoFile: "kinds.go.tmpl"} {
		src, err := execute(tmpl, data)
		if err != nil {
			return output{}, err
		}
		out.files[path] = src
	}
	var project, kustomization strings.Builder
	var resources []string
	for _, k := range kinds {
		fmt.Fprintf(&project, `- api:
    crdVersion: v1
    namespaced: true
  controller: true
  domain: coralogix.com
  group: coralogix
  kind: %s
  path: %s/api/coralogix/v1alpha1
  version: v1alpha1
`, k.Kind, module)
		fmt.Fprintf(&kustomization, "  - bases/coralogix.com_%s.yaml\n", k.Plural)
		resources = append(resources, k.Plural)
	}
	out.files[version.VersionsFile] = renderVersions(kinds)
	out.blocks[projectFile] = project.String()
	out.blocks[crdKustomization] = kustomization.String()
	out.blocks[helmClusterRole] = clusterRoleRules(resources)
	return out, nil
}

// renderVersions returns versions.yaml: the generator version of each kind. Every run generates
// every kind, so each kind has the current version, and a kind that is no longer listed drops out.
func renderVersions(kinds []*kindData) []byte {
	var b strings.Builder
	b.WriteString(version.YAMLHeaderLine + "\n# The generator version of each generated kind.\n")
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.Kind
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(&b, "%s: %s\n", name, version.Current)
	}
	return []byte(b.String())
}

// clusterRoleRules are the rules of the Helm cluster role for the generated kinds, as the
// kubebuilder RBAC markers of the controllers state them.
func clusterRoleRules(resources []string) string {
	if len(resources) == 0 {
		return ""
	}
	var b strings.Builder
	for _, rule := range []struct {
		suffix string
		verbs  []string
	}{
		{"", []string{"create", "delete", "get", "list", "patch", "update", "watch"}},
		{"/finalizers", []string{"update"}},
		{"/status", []string{"get", "patch", "update"}},
	} {
		b.WriteString("  - apiGroups:\n      - coralogix.com\n    resources:\n")
		for _, r := range resources {
			b.WriteString("      - " + r + rule.suffix + "\n")
		}
		b.WriteString("    verbs:\n")
		for _, v := range rule.verbs {
			b.WriteString("      - " + v + "\n")
		}
	}
	return b.String()
}

// renderSample writes the sample custom resource with every field.
func renderSample(d *kindData, full *omap) ([]byte, error) {
	spec, err := yamlNode(full)
	if err != nil {
		return nil, err
	}
	root := &yaml.Node{Kind: yaml.MappingNode}
	add := func(key string, value *yaml.Node) {
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
	}
	scalar := func(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: v} }
	add("apiVersion", scalar("coralogix.com/v1alpha1"))
	add("kind", scalar(d.Kind))
	add("metadata", &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{scalar("name"), scalar(kebab(d.Kind) + "-sample")}})
	add("spec", spec)
	var buf bytes.Buffer
	buf.WriteString(version.YAMLHeader + "\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
