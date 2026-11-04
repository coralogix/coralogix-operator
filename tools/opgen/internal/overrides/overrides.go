// Package overrides reads behavior-overrides.yaml of a kind. The file states where a kind differs
// from what the API contract alone gives: a field with a line keeps its released behavior, and a
// field without a line follows the contract.
//
// The reader is strict. An unknown key is an error. A line that matches nothing is an error too
// (OVERRIDE_UNUSED), so a stale line cannot stay.
package overrides

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	iacoverrides "github.com/coralogix/coralogix-management-sdk/iac/overrides"
)

// FileName is the name of the file in the directory of the kind.
const FileName = "behavior-overrides.yaml"

const (
	// ModeExisting is a kind that users already have. The contract checks of a released resource
	// apply.
	ModeExisting = "existing"
	// ModeNew is a new kind that uses an operator key. The contract checks of a new resource apply.
	ModeNew = "new"
)

// File is the content of behavior-overrides.yaml.
type File struct {
	// Resource is the OpenAPI component of the resource. It must match the resource of the kind.
	Resource string `yaml:"resource"`
	// Mode is "existing" or "new".
	Mode string `yaml:"mode"`
	// Validators says which validation markers the generated kind has. A kind that users already
	// have must state it.
	Validators *Validators `yaml:"validators"`
	// API describes the parts of the HTTP contract that differ from a new resource. These are the
	// API keys of the shared iac module that the operator renderer supports.
	API API `yaml:"api"`
	// Types has one entry per OpenAPI component.
	Types map[string]Type `yaml:"types"`
	// Unwrap names one-property components whose value is shown without its wrapper.
	Unwrap []string `yaml:"unwrap"`
	// Groups place API fields under released CR objects.
	Groups map[string]Group `yaml:"groups"`
	// Enums keeps released custom resource spellings of API enum values.
	Enums map[string]Enum `yaml:"enums"`

	// apiNode is the api section as the file writes it, nil without one. A Go bool cannot tell a
	// written false from a missing key, so the checks of the api keys read the node.
	apiNode *yaml.Node
	// validatorsNode is the validators section as the file writes it, nil without one. A written
	// validators: null decodes as a missing section.
	validatorsNode *yaml.Node
}

// Validators is the validators section.
type Validators struct {
	// Inferred: false drops the contract limits (lengths, bounds, patterns, item counts) from the
	// markers, as the released kind has none. A required field stays required, and an enum keeps
	// its values. true keeps the limits.
	Inferred *bool `yaml:"inferred"`
	// Fields changes released validation at one CR spec path. Unlisted fields follow the contract.
	Fields map[string]Validation `yaml:"fields"`
}

// Validation changes the validation markers of one released CR spec path.
type Validation struct {
	// Inferred overrides validators.inferred for this path's contract limits and default.
	Inferred *bool `yaml:"inferred"`
	// Remove names exact generated markers. Add supplies released markers.
	Remove []string `yaml:"remove"`
	Add    []string `yaml:"add"`
}

// API is the api section: the API keys of the iac module.
type API struct {
	iacoverrides.API `yaml:",inline"`
	// RequestValues sets fields beside a request wrapper from the CR or a generated value.
	RequestValues map[string]map[string]RequestValue `yaml:"requestValues"`
}

// RequestValue names the source of a string field beside a request wrapper.
type RequestValue struct {
	// Source is spec.<field>, uuid, or literal.
	Source string `yaml:"source"`
	// Value is the value of a literal source, or the prefix of a uuid source.
	Value string `yaml:"value"`
}

func (a API) Check() error {
	if err := a.API.Check(); err != nil {
		return err
	}
	if len(a.RequestValues) == 0 {
		return nil
	}
	if a.RequestWrapper == "" {
		return errors.New("api.requestValues needs api.requestWrapper")
	}
	for _, operation := range slices.Sorted(maps.Keys(a.RequestValues)) {
		fields := a.RequestValues[operation]
		if operation != "create" && operation != "update" {
			return fmt.Errorf("api.requestValues.%s: want create or update", operation)
		}
		if len(fields) == 0 {
			return fmt.Errorf("api.requestValues.%s: no fields", operation)
		}
		for _, name := range slices.Sorted(maps.Keys(fields)) {
			source := fields[name]
			if name == "" || name == a.RequestWrapper {
				return fmt.Errorf("api.requestValues.%s.%s: name a field beside the wrapper", operation, name)
			}
			switch {
			case source.Source == "uuid", source.Source == "literal":
				if source.Source == "literal" && source.Value == "" {
					return fmt.Errorf("api.requestValues.%s.%s: literal needs a value", operation, name)
				}
			case strings.HasPrefix(source.Source, "spec.") && len(source.Source) > len("spec."):
				if source.Value != "" {
					return fmt.Errorf("api.requestValues.%s.%s: spec source cannot set value", operation, name)
				}
			default:
				return fmt.Errorf("api.requestValues.%s.%s: source is not spec.<field>, uuid, or literal", operation, name)
			}
		}
	}
	return nil
}

// Type is the lines of one component.
type Type struct {
	Fields map[string]Field `yaml:"fields"`
	Groups map[string]Group `yaml:"groups"`
}

// Group is a released CR object that holds fields from one API component.
type Group struct {
	Fields     []string `yaml:"fields"`
	RequireOne bool     `yaml:"requireOne"`
}

// Field is one field line.
type Field struct {
	// Collection keeps an API set as a released CR list.
	Collection string `yaml:"collection"`
	// ListMapKey identifies items when the released list is a Kubernetes list-map.
	ListMapKey string `yaml:"listMapKey"`
	// EmptyObject keeps the released map form of an object with no fields.
	EmptyObject string `yaml:"emptyObject"`
	// Skip omits a released top-level API field from the CR and requests.
	Skip bool `yaml:"skip"`
	// Ref makes a string id of another resource a reference (way 2): the custom resource sets
	// backendRef.id, or resourceRef, the custom resource that manages the other resource.
	Ref *Ref `yaml:"ref"`
	// ReadOnly states that the server owns this field, even when the contract does not.
	ReadOnly bool `yaml:"readOnly"`
	// Required and Optional keep the released CRD presence rule.
	Required bool `yaml:"required"`
	Optional bool `yaml:"optional"`
	// OmitEmptyInRequest keeps a required collection out of the API request when it is empty.
	OmitEmptyInRequest bool `yaml:"omitEmptyInRequest"`
	// Name is the released CR property name when it differs from the API property.
	Name string `yaml:"name"`
	// RequireOne keeps the released exactly-one rule for a oneOf that allows no arm in the CR.
	RequireOne bool `yaml:"requireOne"`
	// Unwrap overrides the component-wide wrapper choice at this field.
	Unwrap *bool `yaml:"unwrap"`
	// RequestValueBy sends a hidden API value chosen by the populated object arm.
	RequestValueBy map[string]string `yaml:"requestValueBy"`
}

// Enum records the accepted API values and their custom resource spellings.
type Enum struct {
	Zero       string            `yaml:"zero"`
	RejectZero bool              `yaml:"rejectZero"`
	Values     []string          `yaml:"values"`
	Verbatim   bool              `yaml:"verbatim"`
	TrimPrefix string            `yaml:"trimPrefix"`
	Names      map[string]string `yaml:"names"`
	Aliases    map[string]string `yaml:"aliases"`
	Rejected   []string          `yaml:"rejected"`
}

// Ref is the ref key of a field line.
type Ref struct {
	// Kind is the kind of the custom resource that resourceRef names. Its status id is sent.
	Kind string `yaml:"kind"`
	// Form keeps a released reference with only resourceRef. The default also accepts backendRef.
	Form string `yaml:"form"`
	// ResourceField is the second CR field of a split reference.
	ResourceField string `yaml:"resourceField"`
	// Lookup names a Go function that resolves a backend name through a second SDK client.
	Lookup string `yaml:"lookup"`
	// IDField is the Go name of a handwritten namedBackendOnly type's backend id field.
	IDField string `yaml:"idField"`
	// Version is the API version of the referenced custom resource; v1alpha1 is the default.
	Version string `yaml:"version"`
	// BackendIDType keeps a released numeric backendRef.id type, such as uint32.
	BackendIDType string `yaml:"backendIDType"`
	// Type is NCRef when a released CR uses its one-rule reference validation. Without it,
	// generated fields use ResourceIDRef and its four rules.
	Type string `yaml:"type"`
}

// Line is one field line: the component and the field that it names.
type Line struct {
	Component, Field string
}

func (l Line) String() string { return "types." + l.Component + ".fields." + l.Field }

// Parse reads the file. An unknown key or a wrong value is an error.
func Parse(data []byte) (*File, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var f File
	if err := dec.Decode(&f); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s is empty", FileName)
		}
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	f.apiNode = section(&doc, "api")
	f.validatorsNode = section(&doc, "validators")
	if err := f.check(); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &f, nil
}

func (f *File) check() error {
	if f.Resource == "" {
		return errors.New("resource is required")
	}
	if f.Mode != ModeExisting && f.Mode != ModeNew {
		return fmt.Errorf("mode is %q, want %q or %q", f.Mode, ModeExisting, ModeNew)
	}
	if err := f.checkAPI(); err != nil {
		return err
	}
	if err := f.checkValidators(); err != nil {
		return err
	}
	for _, line := range f.Lines() {
		field := f.Types[line.Component].Fields[line.Field]
		if !field.Skip && field.Ref == nil && !field.ReadOnly && !field.Required && !field.Optional && !field.OmitEmptyInRequest && field.Name == "" && !field.RequireOne && field.Unwrap == nil && len(field.RequestValueBy) == 0 && field.Collection == "" && field.ListMapKey == "" && field.EmptyObject == "" {
			return fmt.Errorf("%s: the line sets no key", line)
		}
		if field.Skip && (field.Ref != nil || field.ReadOnly || field.Required || field.Optional || field.OmitEmptyInRequest || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0 || field.Collection != "" || field.ListMapKey != "" || field.EmptyObject != "") {
			return fmt.Errorf("%s.skip: skip cannot change another property of the CR field", line)
		}
		if field.Skip && f.Mode != ModeExisting {
			return fmt.Errorf("%s.skip: a new kind follows the contract", line)
		}
		if field.OmitEmptyInRequest && f.Mode != ModeExisting {
			return fmt.Errorf("%s.omitEmptyInRequest: a new kind follows the contract", line)
		}
		if field.Ref != nil && field.ReadOnly {
			return fmt.Errorf("%s: ref and readOnly cannot be set together", line)
		}
		if field.Collection != "" && field.Collection != "list" {
			return fmt.Errorf("%s.collection: want list", line)
		}
		if field.Collection != "" && f.Mode != ModeExisting {
			return fmt.Errorf("%s.collection: a new kind follows the contract", line)
		}
		if field.ListMapKey != "" && (field.Collection != "list" || !validJSONName(field.ListMapKey)) {
			return fmt.Errorf("%s.listMapKey: name an item field and set collection: list", line)
		}
		if field.EmptyObject != "" && (field.EmptyObject != "map" || f.Mode != ModeExisting) {
			return fmt.Errorf("%s.emptyObject: a released empty object can use map", line)
		}
		if field.Required && field.Optional {
			return fmt.Errorf("%s: required and optional cannot be set together", line)
		}
		if field.ReadOnly && (field.Required || field.Optional || field.Name != "") {
			return fmt.Errorf("%s: readOnly cannot change the CR field", line)
		}
		if f.Mode == ModeNew && field.ReadOnly {
			return fmt.Errorf("%s.readOnly: a new kind follows the contract", line)
		}
		if f.Mode == ModeNew && (field.Required || field.Optional || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0) {
			return fmt.Errorf("%s: a new kind follows the contract", line)
		}
		if field.Name != "" && !validJSONName(field.Name) {
			return fmt.Errorf("%s.name: %q is not a CR property name", line, field.Name)
		}
		if len(field.RequestValueBy) != 0 {
			if field.Ref != nil || field.ReadOnly || field.Required || field.Optional || field.Name != "" || field.RequireOne || field.Unwrap != nil {
				return fmt.Errorf("%s.requestValueBy: the hidden API field cannot set another field key", line)
			}
			for source, value := range field.RequestValueBy {
				if source == "" || value == "" {
					return fmt.Errorf("%s.requestValueBy: name an arm and a nonempty API value", line)
				}
			}
		}
		if field.Ref != nil && (!token.IsIdentifier(field.Ref.Kind) || !token.IsExported(field.Ref.Kind)) {
			return fmt.Errorf("%s.ref.kind: %q is not a kind", line, field.Ref.Kind)
		}
		if field.Ref != nil {
			if field.Ref.BackendIDType != "" && field.Ref.BackendIDType != "int32" && field.Ref.BackendIDType != "int64" && field.Ref.BackendIDType != "uint32" {
				return fmt.Errorf("%s.ref.backendIDType: use int32, int64, or uint32", line)
			}
			if field.Ref.Form == "resourceRefOnly" && field.Ref.BackendIDType != "" {
				return fmt.Errorf("%s.ref.backendIDType: resourceRefOnly has no backend id", line)
			}
			if field.Ref.Form != "" && field.Ref.Form != "resourceRefOnly" && field.Ref.Form != "split" && field.Ref.Form != "named" && field.Ref.Form != "namedBackendOnly" {
				return fmt.Errorf("%s.ref.form: use resourceRefOnly, split, named, namedBackendOnly, or omit form", line)
			}
			if (field.Ref.Form == "named" || field.Ref.Form == "namedBackendOnly") && (!token.IsIdentifier(field.Ref.Lookup) || !token.IsExported(field.Ref.Lookup)) {
				return fmt.Errorf("%s.ref.lookup: a named reference needs an exported lookup function", line)
			}
			if field.Ref.Form != "named" && field.Ref.Form != "namedBackendOnly" && field.Ref.Lookup != "" {
				return fmt.Errorf("%s.ref.lookup: only a named reference uses this key", line)
			}
			if field.Ref.Form == "split" && (!validJSONName(field.Ref.ResourceField) || field.Ref.ResourceField == field.Name) {
				return fmt.Errorf("%s.ref.resourceField: split needs a distinct CR property name", line)
			}
			if field.Ref.Form != "split" && field.Ref.ResourceField != "" {
				return fmt.Errorf("%s.ref.resourceField: only split uses this key", line)
			}
			if field.Ref.Version != "" && !regexp.MustCompile(`^v[0-9]+(?:(?:alpha|beta)[0-9]+)?$`).MatchString(field.Ref.Version) {
				return fmt.Errorf("%s.ref.version: %q is not a Kubernetes API version", line, field.Ref.Version)
			}
			if field.Ref.IDField != "" && (field.Ref.Form != "namedBackendOnly" || field.Ref.Type == "" || !token.IsIdentifier(field.Ref.IDField)) {
				return fmt.Errorf("%s.ref.idField: use it with a handwritten namedBackendOnly type", line)
			}
			if field.Ref.Form != "" && field.Ref.Form != "namedBackendOnly" && field.Ref.Type != "" {
				return fmt.Errorf("%s.ref.type: the selected form has its own CR type", line)
			}
			switch field.Ref.Type {
			case "":
			case "NCRef":
				if field.Ref.Form != "" {
					return fmt.Errorf("%s.ref.type: NCRef needs the default reference form", line)
				}
				if f.Mode != ModeExisting {
					return fmt.Errorf("%s.ref.type: NCRef keeps released validation and needs mode existing", line)
				}
			default:
				if field.Ref.Form != "namedBackendOnly" || f.Mode != ModeExisting || !token.IsIdentifier(field.Ref.Type) || !token.IsExported(field.Ref.Type) {
					return fmt.Errorf("%s.ref.type: %q is not a supported released reference type", line, field.Ref.Type)
				}
			}
		}
	}
	if f.Mode == ModeNew && len(f.Enums) != 0 {
		return errors.New("enums: a new kind follows the contract")
	}
	for name, enum := range f.Enums {
		if err := enum.check(name); err != nil {
			return err
		}
	}
	if f.Mode == ModeNew && len(f.Unwrap) != 0 {
		return errors.New("unwrap: a new kind follows the contract")
	}
	seenUnwrap := map[string]bool{}
	for _, name := range f.Unwrap {
		if name == "" || seenUnwrap[name] {
			return fmt.Errorf("unwrap: component %q is empty or listed twice", name)
		}
		seenUnwrap[name] = true
	}
	if f.Mode == ModeNew && len(f.Groups) != 0 {
		return errors.New("groups: a new kind follows the contract")
	}
	if err := checkGroups("groups", f.Groups); err != nil {
		return err
	}
	for name, typ := range f.Types {
		if f.Mode == ModeNew && len(typ.Groups) != 0 {
			return fmt.Errorf("types.%s.groups: a new kind follows the contract", name)
		}
		if err := checkGroups("types."+name+".groups", typ.Groups); err != nil {
			return err
		}
	}
	return nil
}

func checkGroups(path string, groups map[string]Group) error {
	used := map[string]bool{}
	for name, group := range groups {
		if !validJSONName(name) || len(group.Fields) == 0 {
			return fmt.Errorf("%s.%s: name a CR object and at least one API field", path, name)
		}
		for _, field := range group.Fields {
			if field == "" || used[field] {
				return fmt.Errorf("%s.%s: field %q is empty or belongs to more than one group", path, name, field)
			}
			used[field] = true
		}
	}
	return nil
}

func validJSONName(name string) bool {
	if name == "" || !asciiLetter(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !asciiLetter(c) && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

func asciiLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func (e Enum) check(name string) error {
	if name == "" || len(e.Values) == 0 {
		return fmt.Errorf("enums.%s.values: name an enum and at least one accepted value", name)
	}
	if !validEnumValue(e.Zero) && e.Zero != "" {
		return fmt.Errorf("enums.%s.zero: %q is not one CR enum value", name, e.Zero)
	}
	if e.RejectZero && e.Zero == "" {
		return fmt.Errorf("enums.%s.rejectZero: zero must name the omitted API value", name)
	}
	seen := map[string]bool{}
	for _, value := range append(slices.Clone(e.Values), e.Rejected...) {
		if value == "" || seen[value] {
			return fmt.Errorf("enums.%s: value %q is empty or listed twice", name, value)
		}
		seen[value] = true
	}
	for value, cr := range e.Names {
		if !slices.Contains(e.Values, value) || !validEnumValue(cr) {
			return fmt.Errorf("enums.%s.names.%s: name an accepted API value and a nonempty CR value", name, value)
		}
	}
	for alias, target := range e.Aliases {
		if !validEnumValue(alias) || !validEnumValue(target) {
			return fmt.Errorf("enums.%s.aliases: %q and %q must be single CR enum values", name, alias, target)
		}
	}
	if e.TrimPrefix != "" && !slices.ContainsFunc(e.Values, func(value string) bool { return strings.HasPrefix(value, e.TrimPrefix) }) {
		return fmt.Errorf("enums.%s.trimPrefix: %q is not a prefix of an accepted API value", name, e.TrimPrefix)
	}
	return nil
}

func validEnumValue(value string) bool { return value != "" && !strings.ContainsAny(value, ",;\r\n") }

// checkAPI accepts only the API keys that the operator renderer supports. A key may state its
// default (false), as validators.inferred: true does, but a new kind follows the contract, so it
// sets no key to another value. A written section must set a key.
func (f *File) checkAPI() error {
	if f.apiNode == nil {
		return nil
	}
	if f.apiNode.Kind != yaml.MappingNode || len(f.apiNode.Content) == 0 {
		return errors.New("api: the section sets no key; delete it")
	}
	for i := 0; i < len(f.apiNode.Content); i += 2 {
		key, value := f.apiNode.Content[i].Value, f.apiNode.Content[i+1]
		switch key {
		case "updateIDInBody", "operations", "customMethods", "serverSetID", "idProperty", "clientSetID", "delete", "updateMaskInBody", "requestWrapper", "requestValues":
		default:
			return fmt.Errorf("api.%s: opgen does not support this api key", key)
		}
		if f.Mode == ModeNew && !zeroValue(value) {
			return fmt.Errorf("api.%s: a new kind follows the contract, so mode new sets no api key", key)
		}
	}
	return f.API.Check()
}

// section returns the value of a top-level key of the document, nil when the file has no such key.
func section(doc *yaml.Node, key string) *yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

// checkValidators requires validators.inferred in mode existing, where it states the released
// validation. A new kind keeps the contract limits by default, so there the section is optional:
// inferred: false drops the limits of an API whose server does not apply them, and inferred: true
// states the default. A written section must set the key.
func (f *File) checkValidators() error {
	if f.Validators != nil {
		if f.Mode == ModeNew && len(f.Validators.Fields) != 0 {
			return errors.New("validators.fields: a new kind follows the contract")
		}
		for path, change := range f.Validators.Fields {
			if path != "spec" && !strings.HasPrefix(path, "spec.") {
				return fmt.Errorf("validators.fields.%s: use a spec path", path)
			}
			if change.Inferred == nil && len(change.Remove) == 0 && len(change.Add) == 0 {
				return fmt.Errorf("validators.fields.%s: set inferred, remove, or add", path)
			}
			for _, marker := range append(slices.Clone(change.Remove), change.Add...) {
				if !strings.HasPrefix(marker, "+kubebuilder:validation:") && !strings.HasPrefix(marker, "+kubebuilder:default=") {
					return fmt.Errorf("validators.fields.%s: %q is not a validation or default marker", path, marker)
				}
			}
		}
	}
	if f.Mode == ModeExisting {
		if f.Validators == nil || f.Validators.Inferred == nil {
			return errors.New("validators.inferred is required: use true and set validators.fields.<path>.inferred: false for released fields without contract limits")
		}
		return nil
	}
	if f.validatorsNode != nil && (f.Validators == nil || f.Validators.Inferred == nil) {
		return errors.New("validators: the section sets no key; delete it")
	}
	return nil
}

// zeroValue reports whether a node is false, 0, "", null, or empty: the value a missing key has.
func zeroValue(n *yaml.Node) bool {
	if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
		return len(n.Content) == 0
	}
	var v any
	if err := n.Decode(&v); err != nil {
		return false
	}
	return v == nil || reflect.ValueOf(v).IsZero()
}

// Policy returns the contract rules of the kind: the rules of a released resource with the API keys
// of the file in mode existing, and the rules of a new resource in mode new.
func (f *File) Policy() model.Policy {
	if f == nil || f.Mode == ModeNew {
		return model.Policy{}
	}
	exceptions := iacoverrides.Exceptions{API: f.API.API, Unwrap: f.Unwrap, Fields: map[string]map[string]iacoverrides.Field{}}
	for _, line := range f.Lines() {
		field := f.Types[line.Component].Fields[line.Field]
		if !field.Skip && !field.ReadOnly && !field.Optional && field.Unwrap == nil {
			continue
		}
		if exceptions.Fields[line.Component] == nil {
			exceptions.Fields[line.Component] = map[string]iacoverrides.Field{}
		}
		exceptions.Fields[line.Component][line.Field] = iacoverrides.Field{Skip: field.Skip, ReadOnly: field.ReadOnly, Optional: field.Optional, Unwrap: field.Unwrap}
	}
	p := exceptions.Policy()
	for _, line := range f.Lines() {
		if field := f.Field(line.Component, line.Field); field.Optional || field.Required {
			p.Released = append(p.Released, line.Component+"."+line.Field)
		}
		if len(f.Field(line.Component, line.Field).RequestValueBy) != 0 {
			p.FixedRequest = append(p.FixedRequest, line.Component+"."+line.Field)
		}
	}
	for name, enum := range f.Enums {
		if len(enum.Values) != 0 {
			p.EnumAnyPrefix = append(p.EnumAnyPrefix, name)
		}
	}
	return p
}

// Inferred reports whether the markers have the contract limits: true unless the file says
// validators.inferred: false.
func (f *File) Inferred() bool {
	return f == nil || f.Validators == nil || f.Validators.Inferred == nil || *f.Validators.Inferred
}

// InferredAt reports whether a CR spec path keeps contract validation and defaults.
func (f *File) InferredAt(path string) bool {
	if f != nil && f.Validators != nil {
		if change, ok := f.Validators.Fields[path]; ok && change.Inferred != nil {
			return *change.Inferred
		}
	}
	return f.Inferred()
}

// ValidationAt returns the released validation line at a CR spec path.
func (f *File) ValidationAt(path string) (Validation, bool) {
	if f == nil || f.Validators == nil {
		return Validation{}, false
	}
	change, ok := f.Validators.Fields[path]
	return change, ok
}

// Lines returns every field line in a stable order.
func (f *File) Lines() []Line {
	if f == nil {
		return nil
	}
	var out []Line
	for component, t := range f.Types {
		for field := range t.Fields {
			out = append(out, Line{Component: component, Field: field})
		}
	}
	slices.SortFunc(out, func(a, b Line) int {
		return strings.Compare(a.Component+"."+a.Field, b.Component+"."+b.Field)
	})
	return out
}

// Ref returns the ref key of component.field, or nil.
func (f *File) Ref(component, field string) *Ref {
	if f == nil {
		return nil
	}
	return f.Types[component].Fields[field].Ref
}

// Field returns the line for one API property.
func (f *File) Field(component, field string) Field {
	if f == nil {
		return Field{}
	}
	return f.Types[component].Fields[field]
}

// GroupsFor returns the groups of a resource or nested component.
func (f *File) GroupsFor(component string) map[string]Group {
	if f == nil {
		return nil
	}
	if component == f.Resource {
		return f.Groups
	}
	return f.Types[component].Groups
}

// Enum returns the line for an API enum component.
func (f *File) Enum(component string) (Enum, bool) {
	if f == nil {
		return Enum{}, false
	}
	e, ok := f.Enums[component]
	return e, ok
}
