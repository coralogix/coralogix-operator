package generator

import (
	"errors"
	"fmt"
	"strconv"
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
	// HideBodiesInLogs keeps request and response bodies out of the logs (hideBodiesInLogs).
	HideBodiesInLogs bool

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
	// MaskArms are the update mask paths of oneOf arms of a nested object (api.updateMask.nested).
	// The mask names an arm only when the spec sets it, so UpdateMask is a method, not a const.
	MaskArms            []maskArm
	IDField             string // the Go field of the id in the resource response
	IDValue             bool   // the id field is a value, not a pointer
	IDBits              int    // 32 or 64 for a numeric API id; status still stores a string
	IDGoType            string // int32 or int64 for a numeric API id
	IDZeroValid         bool
	CreateResponseField string // the field that wraps the resource in the Create response
	CreateResponseValue bool
	// CreateResponseIDField is the field of a Create response that holds only the id
	// (api.createResponseOnlyIDProperty). CreateResponseIDValue: it is a value, not a pointer.
	CreateResponseIDField      string
	CreateResponseIDValue      bool
	GetResponseField           string
	GetResponseValue           bool
	CreateWrapperField         string // the request field that wraps the resource
	CreateWrapperValue         bool
	UpdateWrapperField         string
	UpdateWrapperValue         bool
	CreateResourceBody         string // the SDK type inside the Create wrapper: the resource, or a request component
	UpdateResourceBody         string // the SDK type inside the Update wrapper
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

	// Overwrite: one write replaces the whole list (api.overwrite). The status has no id, Create and
	// Update send the spec, and Delete sends WriteFields empty.
	Overwrite   bool
	WriteFields []writeField
	ItemGetType string // the SDK type of an item of the list that Get returns

	// Singleton: the API has one resource per team and no id (api.operations with delete: none).
	// Create and Update call the Set operation, Create writes SingletonID to status.id, and the
	// generated Delete makes no call.
	Singleton   bool
	SingletonID string
	// inlined is the object of the response that holds each field of an inline line on the
	// resource.
	inlined map[string]overrides.InlineParent

	// Content is set for a kind with the content key: the CR holds the resource as JSON text.
	Content *contentData

	// Hooks and Status come from the hooks and status keys of the overrides file.
	Hooks  hookData
	Status []statusField

	E2E *e2eData
}

// maskArm is an update mask path that the mask holds when Check, a Go expression on the CR, is true.
type maskArm struct {
	Path, Check string
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

	// inlineParent is the API object field that holds this field in the API, when an inline line
	// shows the fields of the object beside the fields of its component. "" otherwise.
	inlineParent string
	// joinedBy is the separator of a CR list of strings that the API holds as one string.
	joinedBy string
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
	schema  string // the contract component of the enum
}

type enumValue struct {
	API, Name, Const string
	// Send is the API value that the conversion sends for this CR spelling, when it differs from
	// API: an alias maps a spelling to another business value.
	Send string
}

// sent returns the API value that the conversion sends for the spelling.
func (v enumValue) sent() string {
	if v.Send != "" {
		return v.Send
	}
	return v.API
}

// writeField is a field of the write of an overwrite resource, with its empty value.
type writeField struct {
	Field, Empty string
}

type requestValueData struct {
	Field string
	Expr  string
}

// Names returns the custom resource values, for the Enum marker.
// Names returns the value list of the Enum marker. controller-gen reads a bare value that starts
// with a digit, such as 5m, as a number, so a list with such a value is written as quoted strings.
func (e *enumMap) Names() string {
	names := make([]string, len(e.Values))
	quoted := make([]string, len(e.Values))
	quote := false
	for i, v := range e.Values {
		names[i] = v.Name
		quoted[i] = strconv.Quote(v.Name)
		if !bareMarkerString(v.Name) {
			quote = true
		}
	}
	if quote {
		return "{" + strings.Join(quoted, ",") + "}"
	}
	return strings.Join(names, ";")
}

// bareMarkerString reports whether controller-gen reads the unquoted value as the same string.
func bareMarkerString(value string) bool {
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return value != "" && value != "true" && value != "false"
}

type builder struct {
	kind             string
	file             *overrides.File
	used             map[overrides.Line]bool // the lines of the file that changed the output
	usedEnums        map[string]bool
	usedGroups       map[string]bool
	usedExactlyOne   map[string]bool // components whose exactlyOneOf sets became oneOf groups
	usedIgnored      map[string]bool // components whose ignored fields are in a CR type
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
func buildKind(r *model.Resource, refs []sdkcheck.Ref, entry kindEntry, module string, file *overrides.File, updateIDOutsideWrapper bool, fields []wrapperField) (*kindData, error) {
	ix, err := indexRefs(refs)
	if err != nil {
		return nil, err
	}
	b := &builder{kind: entry.Kind, file: file, used: map[overrides.Line]bool{}, usedEnums: map[string]bool{}, ix: ix, alias: sdkAlias(ix.pkg.Name), types: map[string]*crType{},
		enums: map[string]*enumMap{}, usedGroups: map[string]bool{}, usedExactlyOne: map[string]bool{}, usedIgnored: map[string]bool{}, usedValidators: map[string]bool{}, convs: map[string]string{}}
	d := &kindData{
		Kind: entry.Kind, Lower: strings.ToLower(entry.Kind), Plural: plural(entry.Kind),
		Finalizer: kebab(entry.Kind) + ".coralogix.com/finalizer", Module: module,
		SDKPath: ix.pkg.Pkg, SDKAlias: b.alias, CxsdkPath: ix.cxsdk, Replace: r.Replace,
	}
	d.BodyIDInWrapper = r.Policy.RequestWrapper != "" && !updateIDOutsideWrapper
	d.HideBodiesInLogs = file != nil && file.HideBodiesInLogs
	if file != nil && file.Finalizer != "" {
		if file.Finalizer == d.Finalizer {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":finalizer", msg: "the value is the generated finalizer"}
		}
		d.Finalizer = file.Finalizer
	}
	if err := checkLifecycle(r, file); err != nil {
		return nil, err
	}
	if file != nil {
		d.inlined = file.Inlined
	}
	for _, field := range file.InlinedFields() {
		// reshape moved the fields of the object into the resource: the line made the contract.
		b.used[overrides.Line{Component: file.Resource, Field: field}] = true
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
	if err := b.extraSpec(r, d, fields); err != nil {
		return nil, err
	}
	if err := b.requests(r, d); err != nil {
		return nil, err
	}
	d.Types, d.Scalars, d.Enums, d.Convs = b.order, b.scalars, b.enumSeq, b.code
	d.HasNameLookup = b.hasNameLookup
	if err := b.hooks(d); err != nil {
		return nil, err
	}
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
		if file.Field(line.Component, line.Field).Immutable {
			// applyImmutable checked that Update had the field, and removed it.
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
		if len(file.ExactlyOneOf) != 0 && !b.usedExactlyOne[r.Name] {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":exactlyOneOf", msg: "the resource has no such set of fields"}
		}
		for component, typ := range file.Types {
			if len(typ.ExactlyOneOf) != 0 && !b.usedExactlyOne[component] {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":types." + component + ".exactlyOneOf", msg: "the component is not an object of the CR"}
			}
			for name := range typ.Groups {
				if !b.usedGroups[component+"."+name] {
					return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":types." + component + ".groups." + name, msg: "the group changes no generated CR field"}
				}
			}
		}
		for component, typ := range file.Types {
			if len(typ.Ignored) != 0 && !b.usedIgnored[component] {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":types." + component + ".ignored", msg: "the component is not an object of the CR"}
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
func checkLifecycle(r *model.Resource, file *overrides.File) error {
	if r.Overwrite {
		return checkOverwrite(r)
	}
	switch {
	case r.Singleton && file.Upserts():
		return nil // the status id is fixed, and no call takes an id
	case r.Singleton:
		return unsupported("resource", "a singleton is supported only when one POST sets it: api.operations with delete: none")
	case r.IDType == nil:
		return unsupported("resource", "the id has no type")
	case r.IDType.Kind == model.Integer && r.IDType.Format != "int32" && r.IDType.Format != "int64":
		return unsupported("resource", "a numeric id must be int32 or int64")
	case r.IDType.Kind != model.String && r.IDType.Kind != model.Integer:
		return unsupported("resource", "the id must be a string, int32, or int64")
	case r.Create.Response.ID != "" && r.IDType.Kind != model.String:
		return unsupported("create.response", "a Create response that holds only the id needs a string id")
	case !r.Create.Response.Direct && r.Create.Response.Field == "" && r.Create.Response.ID == "":
		return unsupported("create.response", "Create must return the resource directly, in a field, or its id (api.createResponseOnlyIDProperty)")
	}
	return nil
}

// checkOverwrite checks an overwrite resource (api.overwrite): one write replaces the whole list.
// The CR is the list, so its spec has the fields of the write, and the status has no id.
func checkOverwrite(r *model.Resource) error {
	if r.WriteReturnsResource {
		return unsupported("api.overwrite.writeReturnsResource", "the operator does not read the write response")
	}
	for _, f := range r.Fields {
		if f.Create != nil && f.Type.Kind != model.List && f.Type.Kind != model.Set && f.Type.Kind != model.Map {
			return unsupported("resource."+f.Name, "the write of an overwrite resource sends a list or a map")
		}
	}
	return nil
}

// isShapeError reports whether err is an operator check, and returns it.
func isShapeError(err error) (*shapeError, bool) {
	var s *shapeError
	ok := errors.As(err, &s)
	return s, ok
}
