package generator

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
)

// resolveSDKNames lists the SDK names that the generated code uses: the names of the resource,
// and the accessor of the SDK client set that the generated registry calls.
func resolveSDKNames(r *model.Resource, tag, module string) ([]sdkcheck.Ref, error) {
	refs, err := sdkcheck.Names(r, tag, module, nil)
	if err != nil {
		return nil, err
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

func (ix *refIndex) schemaType(schema string) (string, error) {
	ref, ok := ix.bySchema[schema]
	if !ok {
		return "", fmt.Errorf("no SDK type for component %s", schema)
	}
	return ref.Name, nil
}

func (ix *refIndex) enumConst(typeName, value string) (string, error) {
	name := enumConstName(typeName, value)
	if !ix.consts[name] {
		return "", fmt.Errorf("no SDK constant %s", name)
	}
	return name, nil
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
