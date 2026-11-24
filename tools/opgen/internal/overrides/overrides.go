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
	// ExactlyOneOf lists sets of spec fields of which a CR sets exactly one, when the contract
	// states the choice only in text. A name is an API property, or a group name. Each field must be
	// optional.
	ExactlyOneOf [][]string `yaml:"exactlyOneOf"`
	// Enums keeps released custom resource spellings of API enum values.
	Enums map[string]Enum `yaml:"enums"`
	// Finalizer keeps the released finalizer name of an existing kind. It is an operator key; the
	// shared iac module does not know it. The generated name is kebab(Kind).coralogix.com/finalizer.
	Finalizer string `yaml:"finalizer"`
	// HideBodiesInLogs keeps the Create and Update bodies and responses out of the operator logs,
	// for a kind whose spec can hold secrets, such as a collector configuration with credentials.
	// The logs name the kind and the id only. It is an operator key; the shared iac module does not
	// know it.
	HideBodiesInLogs bool `yaml:"hideBodiesInLogs"`
	// Hooks names the handwritten steps that the generated lifecycle calls. Each hook is an
	// unexported method of the reconciler, defined in a handwritten file of the same package. Hooks
	// change the generated body; they never replace it. Mode existing only.
	Hooks []string `yaml:"hooks"`
	// Status declares status fields beside id, conditions, and printableStatus. A hook sets them.
	// Mode existing only.
	Status map[string]StatusField `yaml:"status"`
	// Content makes the CR hold the API resource as JSON text, not as fields. The CRD has no
	// schema for the resource: opgen models only its id. Mode existing only.
	Content *Content `yaml:"content"`
	// Spec declares spec fields of handwritten Go types that a hook reads. The generated
	// conversions do not use them. Mode existing only, and the kind needs a hook.
	Spec map[string]SpecField `yaml:"spec"`
	// UpdateMask "omit" sends no update mask in the PATCH body (api.updateMaskInBody), as a released
	// kind that never sent one. The server then updates each field in the body, and it replaces
	// the whole value of a field, nested oneOf arms included: a config of another arm replaces the
	// stored one. That lets Update change a nested oneOf through a mask that names only top-level
	// fields, which the shared module cannot check. Mode existing only.
	UpdateMask string `yaml:"updateMask"`

	// SingletonID is the status id of a singleton: the API has one resource per team and no id.
	// It is an operator key. Mode existing only.
	SingletonID *SingletonID `yaml:"singletonID"` //nolint:tagliatelle // the key of the Terraform generator and of serverSetID, clientSetID

	// Inlined is set by the generator when it reshapes the contract: the object of the API
	// response that holds each field of an inline line on the resource.
	Inlined map[string]InlineParent `yaml:"-"`
	// PrunedArms is set by the generator: the skipped arms of the oneOf group of the resource that
	// it removed from the response, because no request has them.
	PrunedArms []string `yaml:"-"`

	// hooksNode is the hooks key as the file writes it, nil without one. A written empty list is an
	// error, and a Go slice cannot tell it from a missing key.
	hooksNode *yaml.Node
	// apiNode is the api section as the file writes it, nil without one. A Go bool cannot tell a
	// written false from a missing key, so the checks of the api keys read the node.
	apiNode *yaml.Node
	// validatorsNode is the validators section as the file writes it, nil without one. A written
	// validators: null decodes as a missing section.
	validatorsNode *yaml.Node
}

// SingletonID is the singletonID key.
type SingletonID struct {
	// Fixed is the status id that Create writes. A released kind keeps the id that its objects
	// hold.
	Fixed string `yaml:"fixed"`
	// None keeps the released status of a singleton without an id.
	None bool `yaml:"none"`
}

// InlineParent is the object of the API response that holds an inlined field.
type InlineParent struct {
	// Field is the property of the resource that holds the object, Schema is its component.
	Field, Schema string
}

// UpsertDelete is the value of api.operations.delete for a singleton that the API cannot remove.
const UpsertDelete = "none"

// The operationIds of the PUT and the DELETE that opgen adds to the contract of an upsert, so that
// the shared module sees every lifecycle step. The generated code calls the Set operation for both
// Create and Update, and makes no Delete call.
const (
	UpsertPutSuffix    = "_UpsertPut"
	UpsertDeleteSuffix = "_UpsertDelete"
)

// Upserts reports whether the kind is a singleton that one POST creates and replaces, and that the
// API cannot remove: api.operations names the POST as create and update, and delete is none.
func (f *File) Upserts() bool {
	return f != nil && f.API.Operations != nil && f.API.Operations.Delete == UpsertDelete
}

func (f *File) StatuslessSingleton() bool {
	return f != nil && f.SingletonID != nil && f.SingletonID.None
}

// UpsertOperations returns the operation ids that the shared module reads for an upsert: the ids of
// the PUT and the DELETE that opgen adds to the contract.
func (f *File) UpsertOperations() model.OperationIDs {
	o := f.API.Operations
	return model.OperationIDs{Create: o.Create, Get: o.Get, Update: o.Create + UpsertPutSuffix, Delete: o.Get + UpsertDeleteSuffix}
}

// InlinedFields returns the fields of the resource that inline lines name, in name order.
func (f *File) InlinedFields() []string {
	if f == nil {
		return nil
	}
	var out []string
	for _, name := range slices.Sorted(maps.Keys(f.Types[f.Resource].Fields)) {
		if f.Types[f.Resource].Fields[name].Inline {
			out = append(out, name)
		}
	}
	return out
}

// Content is the content key: the CR holds the resource as JSON from one of three sources, json,
// gzipJson, or configMapRef, as the released Dashboard does. Create and Update decode the JSON
// into the SDK type of the resource with Decode.
type Content struct {
	// Decode is the function that decodes the JSON text into the SDK type of the resource, as
	// <import path>.<Func>, with the signature func([]byte, any) error.
	Decode string `yaml:"decode"`
}

// DecodePackage and DecodeFunc split Decode into its import path and its function.
func (c *Content) DecodePackage() string {
	i := strings.LastIndex(c.Decode, ".")
	return c.Decode[:i]
}

func (c *Content) DecodeFunc() string {
	return c.Decode[strings.LastIndex(c.Decode, ".")+1:]
}

// SpecField is one handwritten spec field.
type SpecField struct {
	// Type is a Go type of the API package that a handwritten file defines. The field is a
	// pointer to it, optional.
	Type string `yaml:"type"`
}

// Hook names. The generated controller calls each one in a fixed place of the lifecycle.
const (
	// HookValidate runs first in Create and Update. An error stops before any API call.
	HookValidate = "validate"
	// HookBeforeCreate and HookBeforeUpdate change the request body before the API call.
	HookBeforeCreate = "beforeCreate"
	HookBeforeUpdate = "beforeUpdate"
	// HookAfterCreate runs after status.id and the finalizer are saved.
	HookAfterCreate = "afterCreate"
	// HookAfterUpdate runs after every successful Update, also when nothing changed.
	HookAfterUpdate = "afterUpdate"
	// HookBeforeDelete runs before the generated DELETE.
	HookBeforeDelete = "beforeDelete"
	// HookDelete runs in place of the generated DELETE.
	HookDelete = "delete"
	// HookAdopt runs in Create after beforeCreate. An id that it returns becomes status.id, and
	// the generated code does not call the API Create.
	HookAdopt = "adopt"
)

// HookNames lists every hook in the order of the lifecycle.
var HookNames = []string{HookValidate, HookBeforeCreate, HookAdopt, HookBeforeUpdate, HookAfterCreate, HookAfterUpdate, HookBeforeDelete, HookDelete}

// HasHook reports whether the file declares the hook.
func (f *File) HasHook(name string) bool {
	return f != nil && slices.Contains(f.Hooks, name)
}

// StatusField is one extra status field.
type StatusField struct {
	// Type is string, int32, int64, or bool. The Go field is a pointer with omitempty.
	Type string `yaml:"type"`
	// KeepOnCleanup keeps the field when a selector mismatch clears the remote status, as Dashboard
	// status.imported does.
	KeepOnCleanup bool `yaml:"keepOnCleanup"`
}

// StatusTypes are the allowed types of an extra status field.
var StatusTypes = []string{"string", "int32", "int64", "bool"}

// reservedStatusFields are the status fields that every generated kind has.
var reservedStatusFields = []string{"id", "conditions", "printableStatus"}

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
	// UpdateMask changes the paths of the update mask of a PATCH Update.
	UpdateMask *UpdateMask `yaml:"updateMask"`
}

// UpdateMask changes the paths of the update mask of a PATCH Update.
type UpdateMask struct {
	// Nested names object fields of the resource that the mask names by the paths of their fields,
	// such as family.active, when the API rejects the bare object path. A oneOf in the object adds
	// the path of the arm that the CR sets, and no other arm.
	Nested []string `yaml:"nested"`
}

// RequestValue names the source of a string field beside a request wrapper.
type RequestValue struct {
	// Source is spec.<field>, uuid, literal, field, or omit. field makes the sibling a spec field of
	// its own name: an optional string with the description of the contract, sent when it is set.
	// omit never sends the sibling, which must be optional.
	Source string `yaml:"source"`
	// Value is the value of a literal source, or the prefix of a uuid source.
	Value string `yaml:"value"`
}

func (a API) Check() error {
	if err := a.API.Check(); err != nil {
		return err
	}
	if a.UpdateMask != nil {
		if len(a.UpdateMask.Nested) == 0 {
			return errors.New("api.updateMask: name an object field in nested")
		}
		seen := map[string]bool{}
		for _, name := range a.UpdateMask.Nested {
			if name == "" || strings.Contains(name, ".") {
				return fmt.Errorf("api.updateMask.nested: %q must be a top-level field of the resource", name)
			}
			if seen[name] {
				return fmt.Errorf("api.updateMask.nested: %s is listed twice", name)
			}
			seen[name] = true
		}
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
			case source.Source == "field", source.Source == "omit":
				if source.Value != "" {
					return fmt.Errorf("api.requestValues.%s.%s: %s cannot set value", operation, name, source.Source)
				}
				if !validJSONName(name) {
					return fmt.Errorf("api.requestValues.%s.%s: %q is not a CR property name", operation, name, name)
				}
			case source.Source == "uuid", source.Source == "literal":
				if source.Source == "literal" && source.Value == "" {
					return fmt.Errorf("api.requestValues.%s.%s: literal needs a value", operation, name)
				}
			case strings.HasPrefix(source.Source, "spec.") && len(source.Source) > len("spec."):
				if source.Value != "" {
					return fmt.Errorf("api.requestValues.%s.%s: spec source cannot set value", operation, name)
				}
			default:
				return fmt.Errorf("api.requestValues.%s.%s: source is not spec.<field>, uuid, literal, field, or omit", operation, name)
			}
		}
	}
	return nil
}

// Type is the lines of one component.
type Type struct {
	Fields map[string]Field `yaml:"fields"`
	Groups map[string]Group `yaml:"groups"`
	// ExactlyOneOf is the key of the resource, for the fields of this component.
	ExactlyOneOf [][]string `yaml:"exactlyOneOf"`
	// RequestWrapper is the property of the request object of a nested component that holds its
	// fields, for example policy in {"policy": {...}, "logRules": {...}}. The CR shows the fields
	// beside the other fields of the component, as Get does. Mode existing only.
	RequestWrapper string `yaml:"requestWrapper"`
	// Ignored lists released CR fields of the component that the API does not have, such as a
	// deprecated field that was never sent. The CR keeps each one, so stored objects and manifests
	// that set it stay valid, and Create and Update never send it. Existing kinds only.
	Ignored map[string]IgnoredField `yaml:"ignored"`
	// UnsupportedArms are oneOf arms of this component that the released CR never had, such as an
	// arm that another kind manages. They stay out of the CR and the requests. Existing kinds only.
	UnsupportedArms []UnsupportedArm `yaml:"unsupportedArms"`
}

// UnsupportedArm is one oneOf arm that the CR does not offer.
type UnsupportedArm struct {
	// Arm is the API property of the arm.
	Arm string `yaml:"arm"`
	// Reason says why the CR does not offer the arm.
	Reason string `yaml:"reason"`
}

// IgnoredField is one released CR field that the API does not have.
type IgnoredField struct {
	// Type is the Go type of the CR field: string, []string, or bool.
	Type string `yaml:"type"`
	// Doc is the description of the CR field. It says why the field has no effect.
	Doc string `yaml:"doc"`
}

// IgnoredTypes are the allowed types of an ignored field.
var IgnoredTypes = []string{"string", "[]string", "bool"}

// Group is a released CR object that holds fields from one API component.
type Group struct {
	Fields []string `yaml:"fields"`
	// RequireOne keeps the released exactly-one rule of the arms of the group.
	RequireOne bool `yaml:"requireOne"`
	// Optional lets a CR leave the group out. The Go field is a pointer, and a request skips the
	// fields of a missing group. Use it when a new API field replaces the fields of the group, and
	// ExactlyOneOf states the choice. The released CRD requires the group, so this loosens it.
	Optional bool `yaml:"optional"`
}

// Field is one field line.
type Field struct {
	// Collection keeps an API set as a released CR list.
	Collection string `yaml:"collection"`
	// ListMapKey identifies items when the released list is a Kubernetes list-map.
	ListMapKey string `yaml:"listMapKey"`
	// EmptyObject keeps the released map form of an object with no fields.
	EmptyObject string `yaml:"emptyObject"`
	// CreateAs names Create-body fields that hold the parts of an object that Update and Get nest,
	// such as {limit: permutationsLimit}. The CR keeps the flat Create fields; Update builds the
	// object when one of them is set. It applies to a field of the resource in an existing kind.
	CreateAs map[string]string `yaml:"createAs"`
	// SendNullAsEmpty sends "" for an optional string, or [] for an optional list, in Create and
	// Update when the CR leaves the field out. Use it when the API keeps the stored value of an
	// omitted field on a full replace, or needs [] instead of null.
	SendNullAsEmpty bool `yaml:"sendNullAsEmpty"`
	// Skip omits an API field from the CR and requests. The field can belong to the resource or to
	// a nested component. A nested skip applies to every CR path of the component. Skipping a oneOf
	// arm removes the arm. A root oneOf with fewer than two arms left is no group.
	Skip bool `yaml:"skip"`
	// Immutable keeps a released CR field that cannot change after Create, when the PATCH Update
	// of the API accepts it. The CRD rejects a change of the field, and Update leaves it out, as
	// for a field that only Create has. A top-level field of an existing kind with a PATCH Update.
	Immutable bool `yaml:"immutable"`
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
	// RequestAs is the path of the property that holds the field in the Create and Update bodies,
	// when it is not the name of the field. A dot goes into an object: "integrationParameters.parameters".
	// The CR keeps the name of the field. It applies to a field of the resource in an existing kind,
	// and, with one segment, to a field of a nested component.
	RequestAs string `yaml:"requestAs"`
	// UpdateAs names the Update-body field when it differs from Create and Get.
	UpdateAs string `yaml:"updateAs"`
	// Inline shows the fields of an optional object beside the other fields of its component, as the
	// released CR does. Create and Update rebuild the object when the CR sets one of them. Mode
	// existing only.
	Inline bool `yaml:"inline"`
	// JoinedBy shows a string that holds several values as a list. Create and Update join the items
	// with this separator. Mode existing only.
	JoinedBy string `yaml:"joinedBy"`
	// Description replaces the contract description of the CR field, when the contract text does
	// not fit the released CR, such as a deprecation that points at a field the CR does not have.
	// Mode existing only.
	Description string `yaml:"description"`
}

// Enum records the rejected API values and the custom resource spellings of an enum. The accepted
// values are the contract values, without the zero value and the rejected values. A new contract
// value is accepted without a change of this line.
type Enum struct {
	Zero       string            `yaml:"zero"`
	RejectZero bool              `yaml:"rejectZero"`
	Verbatim   bool              `yaml:"verbatim"`
	TrimPrefix string            `yaml:"trimPrefix"`
	Names      map[string]string `yaml:"names"`
	Aliases    map[string]string `yaml:"aliases"`
	Rejected   []RejectedValue   `yaml:"rejected"`
	// Values is the removed list of accepted values. The key is read only to tell the user to delete it.
	Values []string `yaml:"values"`
}

// RejectedValue is an API value that the operator does not accept, with the reason.
type RejectedValue struct {
	Value  string `yaml:"value"`
	Reason string `yaml:"reason"`
}

// UnmarshalYAML reads the mapping form. The old plain string form gets a message that shows the new form.
func (r *RejectedValue) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return fmt.Errorf("rejected: write {value: %s, reason: <why the operator cannot accept it>} instead of the plain value", node.Value)
	}
	type plain RejectedValue
	return node.Decode((*plain)(r))
}

// RejectedAPIValues returns the API values of the rejected list.
func (e Enum) RejectedAPIValues() []string {
	values := make([]string, len(e.Rejected))
	for i, rejected := range e.Rejected {
		values[i] = rejected.Value
	}
	return values
}

// IsRejected reports whether an API value is in the rejected list.
func (e Enum) IsRejected(value string) bool {
	return slices.ContainsFunc(e.Rejected, func(r RejectedValue) bool { return r.Value == value })
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
	BackendIDType string `yaml:"backendIdType"`
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
	f.hooksNode = section(&doc, "hooks")
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
	if err := f.checkSingleton(); err != nil {
		return err
	}
	if err := f.checkFinalizer(); err != nil {
		return err
	}
	if err := f.checkHooks(); err != nil {
		return err
	}
	if err := f.checkStatus(); err != nil {
		return err
	}
	if err := f.checkContent(); err != nil {
		return err
	}
	if err := f.checkSpec(); err != nil {
		return err
	}
	for _, component := range slices.Sorted(maps.Keys(f.Types)) {
		wrapper := f.Types[component].RequestWrapper
		switch {
		case wrapper == "":
		case f.Mode != ModeExisting:
			return fmt.Errorf("types.%s.requestWrapper: a new kind follows the contract", component)
		case component == f.Resource:
			return fmt.Errorf("types.%s.requestWrapper: use api.requestWrapper for the resource", component)
		case !validJSONName(wrapper):
			return fmt.Errorf("types.%s.requestWrapper: %q is not a property name", component, wrapper)
		}
	}
	if err := f.checkIgnored(); err != nil {
		return err
	}
	if err := f.checkUnsupportedArms(); err != nil {
		return err
	}
	if err := f.checkUpdateMask(); err != nil {
		return err
	}
	for _, line := range f.Lines() {
		field := f.Types[line.Component].Fields[line.Field]
		if !field.Skip && !field.SendNullAsEmpty && len(field.CreateAs) == 0 && field.Ref == nil && !field.ReadOnly && !field.Required && !field.Optional && !field.OmitEmptyInRequest && field.Name == "" && !field.RequireOne && field.Unwrap == nil && len(field.RequestValueBy) == 0 && field.Collection == "" && field.ListMapKey == "" && field.EmptyObject == "" && field.RequestAs == "" && field.UpdateAs == "" && !field.Inline && field.JoinedBy == "" && field.Description == "" && !field.Immutable {
			return fmt.Errorf("%s: the line sets no key", line)
		}
		if field.Skip && (field.SendNullAsEmpty || len(field.CreateAs) != 0 || field.Ref != nil || field.ReadOnly || field.Required || field.Optional || field.OmitEmptyInRequest || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0 || field.Collection != "" || field.ListMapKey != "" || field.EmptyObject != "" || field.RequestAs != "" || field.UpdateAs != "" || field.Inline || field.JoinedBy != "" || field.Description != "" || field.Immutable) {
			return fmt.Errorf("%s.skip: skip cannot change another property of the CR field", line)
		}
		if field.Immutable {
			switch {
			case f.Mode != ModeExisting:
				return fmt.Errorf("%s.immutable: a new kind follows the contract", line)
			case line.Component != f.Resource:
				return fmt.Errorf("%s.immutable: only a top-level field of the resource can be immutable", line)
			case field.ReadOnly || len(field.CreateAs) != 0 || field.RequestAs != "" || field.UpdateAs != "" || len(field.RequestValueBy) != 0:
				return fmt.Errorf("%s.immutable: the line cannot set readOnly, createAs, requestAs, updateAs, or requestValueBy", line)
			}
		}
		if field.Skip && f.Mode != ModeExisting {
			return fmt.Errorf("%s.skip: a new kind follows the contract", line)
		}
		if len(field.CreateAs) != 0 {
			if f.Mode != ModeExisting {
				return fmt.Errorf("%s.createAs: a new kind follows the contract", line)
			}
			if line.Component != f.Resource {
				return fmt.Errorf("%s.createAs: it applies only to a field of the resource", line)
			}
			if err := (iacoverrides.Field{CreateAs: field.CreateAs}).Check(); err != nil {
				return fmt.Errorf("%s.%w", line, err)
			}
			if field.SendNullAsEmpty || field.Ref != nil || field.ReadOnly || field.Required || field.Optional || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0 || field.Collection != "" || field.ListMapKey != "" || field.EmptyObject != "" || field.RequestAs != "" || field.UpdateAs != "" {
				return fmt.Errorf("%s.createAs: the line cannot set another key", line)
			}
		}
		if field.RequestAs != "" {
			if f.Mode != ModeExisting {
				return fmt.Errorf("%s.requestAs: a new kind follows the contract", line)
			}
			if line.Component != f.Resource && strings.Contains(field.RequestAs, ".") {
				return fmt.Errorf("%s.requestAs: a field of a nested component takes a name, not a path", line)
			}
			if err := (iacoverrides.Field{RequestAs: field.RequestAs}).Check(); err != nil {
				return fmt.Errorf("%s.%w", line, err)
			}
			if field.Ref != nil || field.ReadOnly || len(field.RequestValueBy) != 0 || field.Unwrap != nil {
				return fmt.Errorf("%s.requestAs: the line cannot set ref, readOnly, requestValueBy, or unwrap", line)
			}
		}
		if field.UpdateAs != "" {
			if f.Mode != ModeExisting || line.Component != f.Resource {
				return fmt.Errorf("%s.updateAs: use it on a released resource field", line)
			}
			if err := (iacoverrides.Field{UpdateAs: field.UpdateAs}).Check(); err != nil {
				return fmt.Errorf("%s.%w", line, err)
			}
			if field.Ref != nil || field.ReadOnly || len(field.RequestValueBy) != 0 || field.Unwrap != nil || len(field.CreateAs) != 0 {
				return fmt.Errorf("%s.updateAs: the line cannot set ref, readOnly, requestValueBy, unwrap, or createAs", line)
			}
		}
		if field.Inline {
			if f.Mode != ModeExisting {
				return fmt.Errorf("%s.inline: a new kind follows the contract", line)
			}
			if field.SendNullAsEmpty || field.Ref != nil || field.ReadOnly || field.Required || field.Optional || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0 || field.Collection != "" || field.ListMapKey != "" || field.EmptyObject != "" || field.RequestAs != "" || field.UpdateAs != "" || field.JoinedBy != "" || field.OmitEmptyInRequest {
				return fmt.Errorf("%s.inline: the line cannot set another key", line)
			}
		}
		if field.JoinedBy != "" {
			if f.Mode != ModeExisting {
				return fmt.Errorf("%s.joinedBy: a new kind follows the contract", line)
			}
			if field.Ref != nil || field.ReadOnly || field.SendNullAsEmpty || len(field.RequestValueBy) != 0 || field.Unwrap != nil || field.Collection != "" || field.ListMapKey != "" || field.EmptyObject != "" {
				return fmt.Errorf("%s.joinedBy: the line cannot set ref, readOnly, sendNullAsEmpty, requestValueBy, unwrap, collection, listMapKey, or emptyObject", line)
			}
		}
		if field.SendNullAsEmpty && f.Mode != ModeExisting {
			return fmt.Errorf("%s.sendNullAsEmpty: a new kind follows the contract", line)
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
		if f.Mode == ModeNew && (field.Required || field.Optional || field.Name != "" || field.RequireOne || field.Unwrap != nil || len(field.RequestValueBy) != 0 || field.Description != "") {
			return fmt.Errorf("%s: a new kind follows the contract", line)
		}
		if field.Description != "" && strings.TrimSpace(field.Description) == "" {
			return fmt.Errorf("%s.description: the description is blank", line)
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
				return fmt.Errorf("%s.ref.backendIdType: use int32, int64, or uint32", line)
			}
			if field.Ref.Form == "resourceRefOnly" && field.Ref.BackendIDType != "" {
				return fmt.Errorf("%s.ref.backendIdType: resourceRefOnly has no backend id", line)
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
	if err := checkExactlyOneOf("exactlyOneOf", f.ExactlyOneOf); err != nil {
		return err
	}
	for name, typ := range f.Types {
		if f.Mode == ModeNew && len(typ.Groups) != 0 {
			return fmt.Errorf("types.%s.groups: a new kind follows the contract", name)
		}
		if err := checkGroups("types."+name+".groups", typ.Groups); err != nil {
			return err
		}
		if err := checkExactlyOneOf("types."+name+".exactlyOneOf", typ.ExactlyOneOf); err != nil {
			return err
		}
	}
	return nil
}

// finalizerPattern is a Kubernetes qualified name with a DNS prefix: <prefix>/<name>.
var finalizerPattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/[A-Za-z0-9]([-A-Za-z0-9_.]*[A-Za-z0-9])?$`)

func (f *File) checkFinalizer() error {
	if f.Finalizer == "" {
		return nil
	}
	if f.Mode != ModeExisting {
		return errors.New("finalizer: a new kind uses the generated finalizer")
	}
	prefix, name, _ := strings.Cut(f.Finalizer, "/")
	if !finalizerPattern.MatchString(f.Finalizer) || len(prefix) > 253 || len(name) > 63 {
		return fmt.Errorf("finalizer: %q is not a qualified name such as example.coralogix.com/finalizer", f.Finalizer)
	}
	return nil
}

// checkSingleton checks an upsert and its singletonID. One POST creates and replaces the singleton,
// so create and update name it. The API cannot remove it, so delete is none. The status id is fixed.
func (f *File) checkSingleton() error {
	if !f.Upserts() {
		if f.SingletonID == nil {
			return nil
		}
		if !f.SingletonID.None || f.SingletonID.Fixed != "" || f.Mode != ModeExisting || f.API.Operations == nil || f.API.Operations.Delete == UpsertDelete || f.API.Overwrite != nil {
			return errors.New("singletonID.none: an existing singleton with separate Create, Get, Update, and Delete operations is required")
		}
		return nil
	}
	o, a := f.API.Operations, f.API
	switch {
	case f.Mode != ModeExisting:
		return errors.New("api.operations.delete: none: a new kind follows the contract")
	case o.Create != o.Update:
		return errors.New("api.operations.delete: none needs create and update to name the same POST, which sets the singleton")
	case strings.HasSuffix(o.Create, UpsertPutSuffix) || strings.HasSuffix(o.Get, UpsertDeleteSuffix):
		return fmt.Errorf("api.operations: an operation id cannot end in %s or %s", UpsertPutSuffix, UpsertDeleteSuffix)
	case a.Overwrite != nil || a.Delete != nil || a.RequestWrapper != "" || a.UpdateIDInBody || a.ClientSetID || a.ServerSetID || a.IDProperty != "" || a.CustomMethods || a.UpdateMaskInBody || a.CreateResponseOnlyIDProperty != "" || len(a.RequestValues) != 0:
		return errors.New("api.operations.delete: none describes a singleton without an id or a wrapper; it cannot go with the other api keys")
	case f.SingletonID == nil || f.SingletonID.Fixed == "" || f.SingletonID.None:
		return errors.New("singletonID.fixed is required: the API has no id, so Create writes this one to status.id")
	case f.Content != nil:
		return errors.New("api.operations.delete: none cannot go with content")
	}
	return nil
}

// checkHooks checks the hooks key. Hooks are for a kind that users already have: a new kind
// follows the contract alone.
func (f *File) checkHooks() error {
	if f.hooksNode == nil {
		return nil
	}
	if f.Mode != ModeExisting {
		return errors.New("hooks: a new kind follows the contract")
	}
	if len(f.Hooks) == 0 {
		return errors.New("hooks: the list is empty; delete the key or name a hook")
	}
	seen := map[string]bool{}
	for _, name := range f.Hooks {
		switch {
		case !slices.Contains(HookNames, name):
			return fmt.Errorf("hooks: %q is not a hook; use one of %s", name, strings.Join(HookNames, ", "))
		case seen[name]:
			return fmt.Errorf("hooks: %q is listed twice", name)
		}
		seen[name] = true
	}
	if seen[HookAfterCreate] && !seen[HookAfterUpdate] {
		// A failed afterCreate keeps status.id, so the next run takes the Update path. Only
		// afterUpdate runs there and repairs what afterCreate missed.
		return fmt.Errorf("hooks: %s needs %s: %s repairs what a failed %s missed", HookAfterCreate, HookAfterUpdate, HookAfterUpdate, HookAfterCreate)
	}
	if seen[HookAdopt] && seen[HookAfterCreate] {
		// An adopted resource has no Create response for afterCreate.
		return fmt.Errorf("hooks: %s skips the API Create, so %s has no response; declare one", HookAdopt, HookAfterCreate)
	}
	if seen[HookDelete] && seen[HookBeforeDelete] {
		return fmt.Errorf("hooks: %s replaces the generated DELETE and %s runs before it; declare one", HookDelete, HookBeforeDelete)
	}
	return nil
}

// decodePattern is <import path>.<Func>: a Go import path, then an exported function name.
var decodePattern = regexp.MustCompile(`^[a-z0-9.\-]+(/[A-Za-z0-9_.\-]+)+\.[A-Z][A-Za-z0-9_]*$`)

// checkContent checks the content key.
func (f *File) checkContent() error {
	if f.Content == nil {
		return nil
	}
	if f.Mode != ModeExisting {
		return errors.New("content: a new kind follows the contract")
	}
	if !decodePattern.MatchString(f.Content.Decode) {
		return fmt.Errorf("content.decode: %q is not <import path>.<Func>", f.Content.Decode)
	}
	for _, name := range ContentFields {
		if _, ok := f.Spec[name]; ok {
			return fmt.Errorf("spec.%s: the content key has this field", name)
		}
	}
	return nil
}

// ContentFields are the spec fields of the content key.
var ContentFields = []string{"json", "gzipJson", "configMapRef"}

// checkSpec checks the spec key.
func (f *File) checkSpec() error {
	if len(f.Spec) == 0 {
		return nil
	}
	if f.Mode != ModeExisting {
		return errors.New("spec: a new kind follows the contract")
	}
	if len(f.Hooks) == 0 {
		return errors.New("spec: a handwritten spec field needs a hook that reads it")
	}
	for _, name := range slices.Sorted(maps.Keys(f.Spec)) {
		field := f.Spec[name]
		if !validJSONName(name) {
			return fmt.Errorf("spec.%s: %q is not a spec property name", name, name)
		}
		if !token.IsIdentifier(field.Type) || !token.IsExported(field.Type) {
			return fmt.Errorf("spec.%s.type: %q is not an exported Go type name", name, field.Type)
		}
	}
	return nil
}

// checkStatus checks the status key.
func (f *File) checkStatus() error {
	if len(f.Status) == 0 {
		return nil
	}
	if f.Mode != ModeExisting {
		return errors.New("status: a new kind follows the contract")
	}
	goNames := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(f.Status)) {
		field := f.Status[name]
		switch {
		case !validJSONName(name):
			return fmt.Errorf("status.%s: %q is not a status property name", name, name)
		case slices.Contains(reservedStatusFields, name):
			return fmt.Errorf("status.%s: every kind has this field", name)
		case !slices.Contains(StatusTypes, field.Type):
			return fmt.Errorf("status.%s.type: %q is not one of %s", name, field.Type, strings.Join(StatusTypes, ", "))
		}
		goName := model.GoName(name)
		if slices.Contains([]string{"ID", "Conditions", "PrintableStatus"}, goName) {
			return fmt.Errorf("status.%s: the Go field %s is a field that every kind has", name, goName)
		}
		if other, ok := goNames[goName]; ok {
			return fmt.Errorf("status.%s: the Go field %s is the one of status.%s too", name, goName, other)
		}
		goNames[goName] = name
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

func checkExactlyOneOf(path string, sets [][]string) error {
	for i, set := range sets {
		if len(set) < 2 {
			return fmt.Errorf("%s[%d]: name at least two fields", path, i)
		}
		seen := map[string]bool{}
		for _, name := range set {
			if !validJSONName(name) || seen[name] {
				return fmt.Errorf("%s[%d]: %q is not a field name, or it is listed twice", path, i, name)
			}
			seen[name] = true
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
	if name == "" {
		return errors.New("enums: name an enum")
	}
	if len(e.Values) != 0 {
		return fmt.Errorf("enums.%s.values: the key is removed; every contract value is accepted except the zero value and the rejected values. Delete the key and list each unwanted value under rejected", name)
	}
	if !validEnumValue(e.Zero) && e.Zero != "" {
		return fmt.Errorf("enums.%s.zero: %q is not one CR enum value", name, e.Zero)
	}
	if e.RejectZero && e.Zero == "" {
		return fmt.Errorf("enums.%s.rejectZero: zero must name the omitted API value", name)
	}
	seen := map[string]bool{}
	for _, rejected := range e.Rejected {
		if rejected.Value == "" || seen[rejected.Value] {
			return fmt.Errorf("enums.%s.rejected: value %q is empty or listed twice", name, rejected.Value)
		}
		if strings.TrimSpace(rejected.Reason) == "" {
			return fmt.Errorf("enums.%s.rejected: value %q needs a reason", name, rejected.Value)
		}
		seen[rejected.Value] = true
	}
	for value, cr := range e.Names {
		if e.IsRejected(value) || !validEnumValue(cr) {
			return fmt.Errorf("enums.%s.names.%s: name an accepted API value and a nonempty CR value", name, value)
		}
	}
	for alias, target := range e.Aliases {
		if !validEnumValue(alias) || !validEnumValue(target) {
			return fmt.Errorf("enums.%s.aliases: %q and %q must be single CR enum values", name, alias, target)
		}
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
		case "updateIDInBody", "operations", "methods", "customMethods", "serverSetID", "idProperty", "clientSetID", "delete", "updateMaskInBody", "requestWrapper", "requestValues", "overwrite", "createResponseOnlyIDProperty", "updateMask":
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
	for _, component := range slices.Sorted(maps.Keys(f.Types)) {
		if wrapper := f.Types[component].RequestWrapper; wrapper != "" {
			if exceptions.RequestWrappers == nil {
				exceptions.RequestWrappers = map[string]string{}
			}
			exceptions.RequestWrappers[component] = wrapper
		}
	}
	for _, line := range f.Lines() {
		field := f.Types[line.Component].Fields[line.Field]
		if !field.Skip && !field.ReadOnly && !field.Optional && field.Unwrap == nil && len(field.CreateAs) == 0 && field.RequestAs == "" && field.UpdateAs == "" {
			continue
		}
		if exceptions.Fields[line.Component] == nil {
			exceptions.Fields[line.Component] = map[string]iacoverrides.Field{}
		}
		exceptions.Fields[line.Component][line.Field] = iacoverrides.Field{Skip: field.Skip, ReadOnly: field.ReadOnly, Optional: field.Optional, Unwrap: field.Unwrap, CreateAs: field.CreateAs, RequestAs: field.RequestAs, UpdateAs: field.UpdateAs}
	}
	p := exceptions.Policy()
	if f.Upserts() {
		p.Operations = f.UpsertOperations()
	}
	for _, line := range f.Lines() {
		if field := f.Field(line.Component, line.Field); field.Optional || field.Required {
			p.Released = append(p.Released, line.Component+"."+line.Field)
		}
		if len(f.Field(line.Component, line.Field).RequestValueBy) != 0 {
			p.FixedRequest = append(p.FixedRequest, line.Component+"."+line.Field)
		}
	}
	for name := range f.Enums {
		p.EnumAnyPrefix = append(p.EnumAnyPrefix, name)
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

// ExactlyOneOfFor returns the exactly-one sets of a resource or nested component.
func (f *File) ExactlyOneOfFor(component string) [][]string {
	if f == nil {
		return nil
	}
	if component == f.Resource {
		return f.ExactlyOneOf
	}
	return f.Types[component].ExactlyOneOf
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

// checkIgnored checks the ignored fields of each component.
func (f *File) checkIgnored() error {
	for _, component := range slices.Sorted(maps.Keys(f.Types)) {
		typ := f.Types[component]
		for _, name := range slices.Sorted(maps.Keys(typ.Ignored)) {
			field := typ.Ignored[name]
			location := "types." + component + ".ignored." + name
			if f.Mode != ModeExisting {
				return fmt.Errorf("%s: a new kind follows the contract", location)
			}
			if !validJSONName(name) {
				return fmt.Errorf("%s: %q is not a CR property name", location, name)
			}
			if !slices.Contains(IgnoredTypes, field.Type) {
				return fmt.Errorf("%s.type: use %s", location, strings.Join(IgnoredTypes, ", "))
			}
			if strings.TrimSpace(field.Doc) == "" {
				return fmt.Errorf("%s.doc: say why the field has no effect", location)
			}
			if _, ok := typ.Fields[name]; ok {
				return fmt.Errorf("%s: a field line names the same field", location)
			}
		}
	}
	return nil
}

// UpdateMaskOmit is the updateMask value that sends no update mask.
const UpdateMaskOmit = "omit"

// OmitsUpdateMask reports whether Update sends no update mask.
func (f *File) OmitsUpdateMask() bool {
	return f != nil && f.UpdateMask == UpdateMaskOmit
}

// checkUpdateMask checks the updateMask key. It accepts a PATCH body mask or a POST Update with
// no mask. A PATCH query mask is a required parameter of the SDK call.
func (f *File) checkUpdateMask() error {
	switch {
	case f.UpdateMask == "":
		if f.API.Methods != nil && f.API.Methods.Update == "POST" {
			return errors.New("updateMask: api.methods.update: POST needs updateMask: omit")
		}
		return nil
	case f.UpdateMask != UpdateMaskOmit:
		return fmt.Errorf("updateMask: want %q", UpdateMaskOmit)
	case f.Mode != ModeExisting:
		return errors.New("updateMask: a new kind follows the contract")
	case !f.API.UpdateMaskInBody && (f.API.Methods == nil || f.API.Methods.Update != "POST"):
		return errors.New("updateMask: omit needs api.updateMaskInBody: true or api.methods.update: POST")
	case f.API.UpdateMask != nil:
		return errors.New("updateMask: omit sends no mask, so api.updateMask has no paths to change")
	}
	return nil
}

// checkUnsupportedArms checks the unsupportedArms of each component. The generator checks that
// each one is a oneOf arm of its component.
func (f *File) checkUnsupportedArms() error {
	for _, component := range slices.Sorted(maps.Keys(f.Types)) {
		typ := f.Types[component]
		seen := map[string]bool{}
		for i, arm := range typ.UnsupportedArms {
			location := fmt.Sprintf("types.%s.unsupportedArms[%d]", component, i)
			switch {
			case f.Mode != ModeExisting:
				return fmt.Errorf("%s: a new kind follows the contract", location)
			case !validJSONName(arm.Arm):
				return fmt.Errorf("%s.arm: %q is not a property name", location, arm.Arm)
			case seen[arm.Arm]:
				return fmt.Errorf("%s.arm: %s is listed twice", location, arm.Arm)
			case strings.TrimSpace(arm.Reason) == "":
				return fmt.Errorf("%s.reason: say why the CR does not offer the arm", location)
			}
			seen[arm.Arm] = true
			if _, ok := typ.Fields[arm.Arm]; ok {
				return fmt.Errorf("%s: a field line names the arm %s, which the CR does not have", location, arm.Arm)
			}
			for name, group := range typ.Groups {
				if slices.Contains(group.Fields, arm.Arm) {
					return fmt.Errorf("%s: group %s names the arm %s, which the CR does not have", location, name, arm.Arm)
				}
			}
			if component == f.Resource {
				for name, group := range f.Groups {
					if slices.Contains(group.Fields, arm.Arm) {
						return fmt.Errorf("%s: group %s names the arm %s, which the CR does not have", location, name, arm.Arm)
					}
				}
			}
		}
	}
	return nil
}

// UnsupportedArms returns the unsupported arms of each component, by component.
func (f *File) UnsupportedArms() map[string][]string {
	if f == nil {
		return nil
	}
	out := map[string][]string{}
	for component, typ := range f.Types {
		for _, arm := range typ.UnsupportedArms {
			out[component] = append(out[component], arm.Arm)
		}
	}
	return out
}
