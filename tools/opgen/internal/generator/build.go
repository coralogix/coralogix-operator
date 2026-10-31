package generator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// Issue codes of the operator checks.
const (
	codeNameCollision    = "OPERATOR_NAME_COLLISION"
	codeShapeUnsupported = "RENDERER_SHAPE_UNSUPPORTED"
	codeOverrideInvalid  = "OVERRIDE_INVALID"
	codeOverrideUnused   = "OVERRIDE_UNUSED"
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
	Header    string
	Kind      string // ViewFolder
	Lower     string // viewfolder
	Plural    string // viewfolders
	Finalizer string
	Module    string // the operator module path

	SDKPath, SDKAlias string
	CxsdkPath         string
	Client            string // the SDK client type
	Accessor          string // the method of the SDK client set that returns the client

	Spec    *crType
	Types   []*crType   // nested types, in order of first use
	Scalars []*crScalar // map value types, in order of first use
	Enums   []*enumMap
	Convs   []string // conversion methods, as Go code

	CreateBody, UpdateBody     string // SDK request body types
	CreateConv, UpdateConv     string // spec conversion methods
	CreateMethod, CreateSetter string
	UpdateMethod, UpdateSetter string
	GetMethod, DeleteMethod    string
	Replace                    bool
	Mask, MaskSetter           string // the update mask and its setter, "" for a full replace
	IDField                    string // the Go field of the id in the resource response
	IDValue                    bool   // the id field is a value, not a pointer
	CreateResponseField        string // the field that wraps the resource in the Create response
	CreateResponseValue        bool
	GetResponseField           string
	GetResponseValue           bool
	CreateWrapperField         string // the request field that wraps the resource
	CreateWrapperValue         bool
	UpdateWrapperField         string
	UpdateWrapperValue         bool
	ResourceBody               string // the SDK resource type inside a request wrapper
	MaskInBody                 bool
	MaskBodyField              string
	MaskBodyValue              bool
	CreateValues, UpdateValues []requestValueData
	HasUUID                    bool
	// UpdateIDInBody: Update has no id in its path (api.updateIDInBody). UpdateRequest copies the
	// status id into the body field BodyIDField, a value when BodyIDValue is true.
	UpdateIDInBody  bool
	BodyIDInWrapper bool
	BodyIDField     string
	BodyIDValue     bool

	E2E *e2eData
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

// crScalar is a named Go type of the values of a map. A map value has no marker of its own, so
// the limits of the contract sit on its type, as they do on the items of a list.
type crScalar struct {
	Name    string
	GoType  string
	Doc     []string
	Markers []string
	path    string
}

// crField is a field of a Go struct of the custom resource.
type crField struct {
	API       string // the API property name
	JSON      string // the custom resource name: the API name, or the name of a reference
	GoName    string
	GoType    string
	Optional  bool
	Doc       []string
	Markers   []string
	typ       *model.Type // the type in the resource response
	object    *crType     // the struct of an object, or of the items of a list or map
	enum      *enumMap    // the enum, or the enum of the items of a list
	immutable bool
	ref       string // the kind that a reference names, "" for a field that is not one
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

type requestValueData struct {
	Field string
	Expr  string
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
	kind             string
	file             *overrides.File
	used             map[overrides.Line]bool // the lines of the file that changed the output
	ix               *refIndex
	alias            string
	types            map[string]*crType
	order            []*crType
	scalars          []*crScalar
	enums            map[string]*enumMap
	enumSeq          []*enumMap
	convs            map[string]string // conversion method by "crType->sdkType"
	code             []string
	tmp              int
	conversionSuffix string // separates Create and Update conversions of a wrapped resource
	// droppedLimits: validators.inferred: false dropped at least one contract limit.
	droppedLimits bool
}

// buildKind maps the resource and its SDK names to the data of the templates. It is also the
// renderer check of check: an error is a contract that the renderer cannot write.
func buildKind(r *model.Resource, refs []sdkcheck.Ref, entry kindEntry, module string, file *overrides.File, updateIDOutsideWrapper bool) (*kindData, error) {
	ix, err := indexRefs(refs)
	if err != nil {
		return nil, err
	}
	b := &builder{kind: entry.Kind, file: file, used: map[overrides.Line]bool{}, ix: ix, alias: sdkAlias(ix.pkg.Name), types: map[string]*crType{},
		enums: map[string]*enumMap{}, convs: map[string]string{}}
	d := &kindData{
		Kind: entry.Kind, Lower: strings.ToLower(entry.Kind), Plural: plural(entry.Kind),
		Finalizer: kebab(entry.Kind) + ".coralogix.com/finalizer", Module: module,
		SDKPath: ix.pkg.Pkg, SDKAlias: b.alias, CxsdkPath: ix.cxsdk, Replace: r.Replace,
	}
	d.BodyIDInWrapper = r.Policy.RequestWrapper != "" && !updateIDOutsideWrapper
	if err := checkLifecycle(r); err != nil {
		return nil, err
	}
	if d.Spec, err = b.spec(r); err != nil {
		return nil, err
	}
	if err := b.operations(r, d); err != nil {
		return nil, err
	}
	if err := b.requests(r, d); err != nil {
		return nil, err
	}
	d.Types, d.Scalars, d.Enums, d.Convs = b.order, b.scalars, b.enumSeq, b.code
	if err := b.checkNames(d); err != nil {
		return nil, err
	}
	for _, line := range file.Lines() {
		if file.Types[line.Component].Fields[line.Field].ReadOnly && resourceHasField(r, line) {
			// The shared model removes a read-only field from the spec and requests.
			b.used[line] = true
		}
		if !b.used[line] {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String(), msg: "the line changes no generated code"}
		}
	}
	if !file.Inferred() && !b.droppedLimits {
		return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":validators.inferred",
			msg: "the contract has no limit that validators.inferred: false drops"}
	}
	return d, nil
}

func resourceHasField(r *model.Resource, line overrides.Line) bool {
	seen := map[*model.Type]bool{}
	var hasTypeField func(*model.Type) bool
	hasTypeField = func(t *model.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		if t.Schema == line.Component {
			for _, f := range t.Fields {
				if f.Name == line.Field {
					return true
				}
			}
		}
		if hasTypeField(t.Elem) {
			return true
		}
		for _, f := range t.Fields {
			if hasTypeField(f.Type) {
				return true
			}
		}
		return false
	}
	for _, f := range r.Fields {
		if r.Name == line.Component && f.Name == line.Field {
			return true
		}
		if hasTypeField(f.Type) {
			return true
		}
	}
	return false
}

// checkLifecycle rejects the API shapes that only a released resource has. Their keys land with
// the first kind that needs them.
func checkLifecycle(r *model.Resource) error {
	switch {
	case r.Singleton:
		return unsupported("resource", "a singleton is not supported yet")
	case r.IDType == nil || r.IDType.Kind != model.String:
		return unsupported("resource", "the id must be a string")
	case !r.Create.Response.Direct && r.Create.Response.Field == "":
		return unsupported("create.response", "Create must return the resource directly or in a field")
	}
	return nil
}

// isShapeError reports whether err is an operator check, and returns it.
func isShapeError(err error) (*shapeError, bool) {
	var s *shapeError
	ok := errors.As(err, &s)
	return s, ok
}
