package generator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
)

// Issue codes of the operator checks.
const (
	codeNameCollision    = "OPERATOR_NAME_COLLISION"
	codeShapeUnsupported = "RENDERER_SHAPE_UNSUPPORTED"
)

// shapeError is a contract shape that the operator renderer cannot write. code is the issue code.
type shapeError struct {
	code, path, msg string
}

func (e *shapeError) Error() string { return e.path + ": " + e.msg }

func unsupported(path, format string, args ...any) error {
	return &shapeError{code: codeShapeUnsupported, path: path, msg: fmt.Sprintf(format, args...)}
}

func collision(path, format string, args ...any) error {
	return &shapeError{code: codeNameCollision, path: path, msg: fmt.Sprintf(format, args...)}
}

// kindData is everything that the templates of one kind need.
type kindData struct {
	Header string
	Kind   string // ViewFolder
	Lower  string // viewfolder
	Plural string // viewfolders

	Spec  *crType
	Types []*crType // nested types, in order of first use
	Enums []*enumMap
}

// crType is a Go struct of the custom resource.
type crType struct {
	Name    string
	Doc     []string
	Markers []string
	Fields  []*crField
	schema  string // the response component, "" for the spec
	groups  []model.OneOfGroup
}

// crField is a field of a Go struct of the custom resource.
type crField struct {
	API       string // the API property name
	JSON      string // the custom resource name
	GoName    string
	GoType    string
	Optional  bool
	Doc       []string
	Markers   []string
	typ       *model.Type // the type in the resource response
	object    *crType     // the struct of an object, or of the items of a list or map
	enum      *enumMap    // the enum, or the enum of the items of a list
	immutable bool
}

// JSONTag is the json struct tag of the field.
func (f *crField) JSONTag() string {
	if f.Optional {
		return f.JSON + ",omitempty"
	}
	return f.JSON
}

// enumMap maps the custom resource spelling of an enum to the SDK constants.
type enumMap struct {
	Func    string // the function that maps a spec value to the SDK value
	Var     string // the map
	SDKType string
	Values  []enumValue
}

type enumValue struct {
	Name, Const string
}

// Names returns the custom resource values, for the Enum marker.
func (e *enumMap) Names() string {
	names := make([]string, len(e.Values))
	for i, v := range e.Values {
		names[i] = v.Name
	}
	return strings.Join(names, ";")
}

type builder struct {
	kind    string
	ix      *refIndex
	alias   string
	types   map[string]*crType
	order   []*crType
	enums   map[string]*enumMap
	enumSeq []*enumMap
}

// buildKind maps the resource and its SDK names to the data of the templates. It is also the
// renderer check of check: an error is a contract that the renderer cannot write.
func buildKind(r *model.Resource, refs []sdkcheck.Ref, entry kindEntry) (*kindData, error) {
	ix, err := indexRefs(refs)
	if err != nil {
		return nil, err
	}
	b := &builder{kind: entry.Kind, ix: ix, alias: sdkAlias(ix.pkg.Name), types: map[string]*crType{}, enums: map[string]*enumMap{}}
	d := &kindData{Kind: entry.Kind, Lower: strings.ToLower(entry.Kind), Plural: plural(entry.Kind)}
	if err := checkLifecycle(r); err != nil {
		return nil, err
	}
	if d.Spec, err = b.spec(r); err != nil {
		return nil, err
	}
	d.Types, d.Enums = b.order, b.enumSeq
	if err := b.checkNames(d); err != nil {
		return nil, err
	}
	return d, nil
}

// checkLifecycle rejects the API shapes that only a released resource has. Their keys land with
// the first kind that needs them.
func checkLifecycle(r *model.Resource) error {
	switch {
	case r.Singleton:
		return unsupported("resource", "a singleton is not supported yet")
	case r.IDType == nil || r.IDType.Kind != model.String:
		return unsupported("resource", "the id must be a string")
	case r.MaskInBody:
		return unsupported("update", "an update mask in the body is not supported yet")
	case r.Policy.UpdateIDInBody, r.Policy.RequestWrapper != "", r.Policy.CustomMethods:
		return unsupported("api", "api keys are not supported yet")
	case !r.Create.Response.Direct || r.Create.Response.ID != "":
		return unsupported("create.response", "Create must return the resource")
	}
	return nil
}

// isShapeError reports whether err is an operator check, and returns it.
func isShapeError(err error) (*shapeError, bool) {
	var s *shapeError
	ok := errors.As(err, &s)
	return s, ok
}
