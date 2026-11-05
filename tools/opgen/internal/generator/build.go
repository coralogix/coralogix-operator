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
	codeUpdateOnlyField  = "OPERATOR_UPDATE_ONLY_FIELD"
	codeDuplicateEntry   = "OPERATOR_DUPLICATE_ENTRY"
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
	IDBits                     int    // 32 or 64 for a numeric API id; status still stores a string
	IDGoType                   string // int32 or int64 for a numeric API id
	IDZeroValid                bool
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
	HasNameLookup              bool
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
	path    string // CR path used to build this component's validation
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
	API            string // the API property name
	JSON           string // the custom resource name: the API name, or the name of a reference
	GoName         string
	GoType         string
	Optional       bool
	Doc            []string
	Markers        []string
	typ            *model.Type // the type in the resource response
	object         *crType     // the struct of an object, or of the items of a list or map
	enum           *enumMap    // the enum, or the enum of the items of a list
	immutable      bool
	ref            string // the kind that a reference names, "" for a field that is not one
	refType        string // the CR reference type, when it differs from ResourceIDRef
	refForm        string // resourceRefOnly, or empty for backendRef and resourceRef
	refVersion     string // the referenced CR API version, empty for v1alpha1
	refIDType      string // int32 or int64 for a numeric API id; empty for a string id
	refSibling     string // second CR field of a split reference
	refLookup      string // Go function that resolves a backend name
	refIDField     string // Go id field of a handwritten namedBackendOnly type
	validationPath string // final CR spec path used for inferred validation and defaults
	synthetic      bool   // a CR field with no API property of its own
	group          bool   // this field groups API fields under a released CR object

	// fold: this flat CR field holds a property of an object that Update and Get nest (createAs).
	fold *foldOrigin

	omitEmptyInRequest bool
	sendNullAsEmpty    bool // Create and Update send "" or [] when the optional field is unset
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
	API, Name, Const string
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
	usedEnums        map[string]bool
	usedGroups       map[string]bool
	usedValidators   map[string]bool
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
	hasNameLookup    bool
	// droppedLimits: validators.inferred: false dropped at least one contract limit.
	droppedLimits bool
	// folds are the createAs folds of the top-level fields, by API field name. The spec holds the
	// flat Create fields; Update rebuilds the object that they came from.
	folds    map[string]*foldedField
	specType *crType
	// outVar names the SDK value that assignFrom sets. It is "out" when empty.
	outVar string
}

// foldOrigin names the object and the property that a flat createAs field comes from.
type foldOrigin struct {
	Object, Property string // for example permutations and limit
}

// foldedField is a top-level object field that a createAs line replaces in the CR by the flat
// Create fields of its properties.
type foldedField struct {
	field *model.ResourceField
	folds []model.CreateFold
}

func (b *builder) out() string {
	if b.outVar != "" {
		return b.outVar
	}
	return "out"
}

// buildKind maps the resource and its SDK names to the data of the templates. It is also the
// renderer check of check: an error is a contract that the renderer cannot write.
func buildKind(r *model.Resource, refs []sdkcheck.Ref, entry kindEntry, module string, file *overrides.File, updateIDOutsideWrapper bool) (*kindData, error) {
	ix, err := indexRefs(refs)
	if err != nil {
		return nil, err
	}
	b := &builder{kind: entry.Kind, file: file, used: map[overrides.Line]bool{}, usedEnums: map[string]bool{}, ix: ix, alias: sdkAlias(ix.pkg.Name), types: map[string]*crType{},
		enums: map[string]*enumMap{}, usedGroups: map[string]bool{}, usedValidators: map[string]bool{}, convs: map[string]string{}}
	d := &kindData{
		Kind: entry.Kind, Lower: strings.ToLower(entry.Kind), Plural: plural(entry.Kind),
		Finalizer: kebab(entry.Kind) + ".coralogix.com/finalizer", Module: module,
		SDKPath: ix.pkg.Pkg, SDKAlias: b.alias, CxsdkPath: ix.cxsdk, Replace: r.Replace,
	}
	d.BodyIDInWrapper = r.Policy.RequestWrapper != "" && !updateIDOutsideWrapper
	if file != nil && file.Finalizer != "" {
		if file.Finalizer == d.Finalizer {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":finalizer", msg: "the value is the generated finalizer"}
		}
		d.Finalizer = file.Finalizer
	}
	if err := checkLifecycle(r); err != nil {
		return nil, err
	}
	hasGroups := false
	if file != nil {
		hasGroups = len(file.Groups) != 0
		for _, typ := range file.Types {
			hasGroups = hasGroups || len(typ.Groups) != 0
		}
	}
	if !r.Replace && hasGroups {
		return nil, unsupported("groups", "groups need a full-replace Update; PATCH nested mask paths are not supported")
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
	d.HasNameLookup = b.hasNameLookup
	if err := b.checkNames(d); err != nil {
		return nil, err
	}
	for _, line := range file.Lines() {
		if file.Field(line.Component, line.Field).Unwrap != nil && r.UnwrapUsed[line.Component+"."+line.Field] {
			b.used[line] = true
		}
		if len(file.Field(line.Component, line.Field).CreateAs) != 0 && b.folds[line.Field] != nil && line.Component == r.Name {
			b.used[line] = true
		}
		if file.Field(line.Component, line.Field).Skip {
			// overrideIssues checked that this is a used top-level API property. The model
			// removes it before the renderer sees the resource.
			b.used[line] = true
		}
		if file.Types[line.Component].Fields[line.Field].ReadOnly && resourceHasField(r, line) {
			// The shared model removes a read-only field from the spec and requests.
			b.used[line] = true
		}
		if !b.used[line] {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String(), msg: "the line changes no generated code"}
		}
	}
	if file != nil {
		if file.Validators != nil {
			for path := range file.Validators.Fields {
				if !b.usedValidators[path] {
					return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":validators.fields." + path, msg: "the line changes no generated validation"}
				}
			}
		}
		for _, name := range file.Unwrap {
			if !r.UnwrapUsed["unwrap."+name] {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":unwrap." + name, msg: "the component has no wrapper in this resource"}
			}
		}
		for name := range file.Groups {
			if !b.usedGroups[r.Name+"."+name] {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":groups." + name, msg: "the group changes no generated CR field"}
			}
		}
		for component, typ := range file.Types {
			for name := range typ.Groups {
				if !b.usedGroups[component+"."+name] {
					return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":types." + component + ".groups." + name, msg: "the group changes no generated CR field"}
				}
			}
		}
		for name := range file.Enums {
			for _, field := range r.Fields {
				if field.Behavior == model.Computed && field.Type.Kind == model.Enum && field.Type.Schema == name {
					// The model needs this line even though no CR field is rendered.
					b.usedEnums[name] = true
				}
			}
			if !b.usedEnums[name] {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":enums." + name, msg: "the line changes no generated code"}
			}
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
	case r.IDType == nil:
		return unsupported("resource", "the id has no type")
	case r.IDType.Kind == model.Integer && r.IDType.Format != "int32" && r.IDType.Format != "int64":
		return unsupported("resource", "a numeric id must be int32 or int64")
	case r.IDType.Kind != model.String && r.IDType.Kind != model.Integer:
		return unsupported("resource", "the id must be a string, int32, or int64")
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
