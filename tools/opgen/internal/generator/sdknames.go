package generator

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// resolveSDKNames lists the SDK names that the generated code uses: the names of the resource,
// and the accessor of the SDK client set that the generated registry calls.
func resolveSDKNames(r *model.Resource, tag, module string, updateIDOutsideWrapper bool, file *overrides.File) ([]sdkcheck.Ref, error) {
	refs, err := sdkcheck.Names(r, tag, module, nil)
	if err != nil {
		return nil, err
	}
	if updateIDOutsideWrapper {
		for _, ref := range refs {
			if ref.Path != "update.body" || ref.Kind != sdkcheck.KindType {
				continue
			}
			refs = append(refs, sdkcheck.Ref{Path: "update.body." + r.ID(), Kind: sdkcheck.KindField,
				Pkg: ref.Pkg, Owner: ref.Name, Name: model.GoName(r.ID()), Want: "*string", WantValue: "string", Rule: sdkcheck.RuleProperty})
			break
		}
	}
	if file != nil {
		for _, operation := range []string{"create", "update"} {
			values := file.API.RequestValues[operation]
			for _, ref := range refs {
				if ref.Path != operation+".body" || ref.Kind != sdkcheck.KindType {
					continue
				}
				for _, name := range sortedKeys(values) {
					refs = append(refs, sdkcheck.Ref{Path: operation + ".body." + name, Kind: sdkcheck.KindField,
						Pkg: ref.Pkg, Owner: ref.Name, Name: model.GoName(name), Want: "*string", WantValue: "string", Rule: sdkcheck.RuleProperty})
				}
				break
			}
		}
	}
	// The client set accessor usually drops " Service" from the tag. When it does not, the check
	// finds the one accessor that returns the client type.
	return append(refs, sdkcheck.Ref{
		Path: "clientset", Kind: sdkcheck.KindMethod, Pkg: module + "/go/openapi/cxsdk", Owner: "ClientSet",
		Name: goName(strings.TrimSuffix(tag, " Service")), Want: "func() *" + sdkcheck.PackageName(tag) + "." + sdkcheck.ClientType(tag),
		ByType: true, Rule: sdkcheck.RuleClientSet,
	}), nil
}

// refIndex finds SDK names by model path, by owner type, and by component.
type refIndex struct {
	pkg      sdkcheck.Ref
	types    map[string]sdkcheck.Ref   // by path
	fields   map[string]sdkcheck.Ref   // by "Owner.Name"
	methods  map[string][]sdkcheck.Ref // by path
	consts   map[string]bool           // by name
	bySchema map[string]sdkcheck.Ref   // component types by schema name
	cxsdk    string                    // import path of the SDK client set package
}

func indexRefs(refs []sdkcheck.Ref) (*refIndex, error) {
	ix := &refIndex{types: map[string]sdkcheck.Ref{}, fields: map[string]sdkcheck.Ref{}, methods: map[string][]sdkcheck.Ref{},
		consts: map[string]bool{}, bySchema: map[string]sdkcheck.Ref{}}
	for _, ref := range refs {
		switch ref.Kind {
		case sdkcheck.KindPackage:
			switch ref.Path {
			case "resource":
				ix.pkg = ref
			case "cxsdk":
				ix.cxsdk = ref.Pkg
			}
		case sdkcheck.KindType:
			ix.types[ref.Path] = ref
			if ref.Schema != "" {
				ix.bySchema[ref.Schema] = ref
			}
		case sdkcheck.KindField:
			ix.fields[ref.Owner+"."+ref.Name] = ref
		case sdkcheck.KindMethod:
			ix.methods[ref.Path] = append(ix.methods[ref.Path], ref)
		case sdkcheck.KindConst:
			ix.consts[ref.Name] = true
		case sdkcheck.KindFunc:
			// The cxsdk functions have fixed names.
		}
	}
	if ix.pkg.Name == "" || ix.cxsdk == "" {
		return nil, fmt.Errorf("the SDK names have no resource package or no cxsdk package")
	}
	return ix, nil
}

func (ix *refIndex) typeRef(path string) (sdkcheck.Ref, error) {
	ref, ok := ix.types[path]
	if !ok {
		return sdkcheck.Ref{}, fmt.Errorf("no SDK type for %s", path)
	}
	return ref, nil
}

func (ix *refIndex) schemaType(schema string) (string, error) {
	ref, ok := ix.bySchema[schema]
	if !ok {
		return "", fmt.Errorf("no SDK type for component %s", schema)
	}
	return ref.Name, nil
}

// field returns the Go type of the field of owner for the API property name.
func (ix *refIndex) field(owner, property string) (sdkcheck.Ref, error) {
	ref, ok := ix.fields[owner+"."+goName(property)]
	if !ok {
		return sdkcheck.Ref{}, fmt.Errorf("no SDK field %s.%s", owner, goName(property))
	}
	return ref, nil
}

// method returns the method at path. With owner "", the path must have exactly one method that is
// not Execute.
func (ix *refIndex) method(path, owner string) (sdkcheck.Ref, error) {
	var found []sdkcheck.Ref
	for _, ref := range ix.methods[path] {
		if (owner == "" && ref.Name != "Execute") || ref.Owner == owner {
			found = append(found, ref)
		}
	}
	if len(found) != 1 {
		return sdkcheck.Ref{}, fmt.Errorf("want one SDK method for %s, found %d", path, len(found))
	}
	return found[0], nil
}

func (ix *refIndex) enumConst(typeName, value string) (string, error) {
	name := enumConstName(typeName, value)
	if !ix.consts[name] {
		return "", fmt.Errorf("no SDK constant %s", name)
	}
	return name, nil
}

// qualify writes an SDK type relative to its package as Go code in another package: "[]Item" is
// "[]alias.Item", and "*string" stays.
func qualify(goType, alias string) string {
	prefix := ""
	for _, p := range []string{"*", "[]", "map[string]"} {
		for strings.HasPrefix(goType, p) {
			prefix, goType = prefix+p, goType[len(p):]
		}
	}
	if builtinTypes[goType] || strings.Contains(goType, ".") {
		return prefix + goType
	}
	return prefix + alias + "." + goType
}

var builtinTypes = map[string]bool{
	"string": true, "bool": true, "int32": true, "int64": true, "float32": true, "float64": true,
	"interface{}": true, "any": true,
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
