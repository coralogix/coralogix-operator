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
}

// Field is one field line.
type Field struct {
	// Ref makes a string id of another resource a reference (way 2): the custom resource sets
	// backendRef.id, or resourceRef, the custom resource that manages the other resource.
	Ref *Ref `yaml:"ref"`
	// ReadOnly states that the server owns this field, even when the contract does not.
	ReadOnly bool `yaml:"readOnly"`
}

// Ref is the ref key of a field line.
type Ref struct {
	// Kind is the kind of the custom resource that resourceRef names. Its status id is sent.
	Kind string `yaml:"kind"`
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
		if field.Ref == nil && !field.ReadOnly {
			return fmt.Errorf("%s: the line sets no key", line)
		}
		if field.Ref != nil && field.ReadOnly {
			return fmt.Errorf("%s: ref and readOnly cannot be set together", line)
		}
		if f.Mode == ModeNew && field.ReadOnly {
			return fmt.Errorf("%s.readOnly: a new kind follows the contract", line)
		}
		if field.Ref != nil && (!token.IsIdentifier(field.Ref.Kind) || !token.IsExported(field.Ref.Kind)) {
			return fmt.Errorf("%s.ref.kind: %q is not a kind", line, field.Ref.Kind)
		}
	}
	return nil
}

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
	if f.Mode == ModeExisting {
		if f.Validators == nil || f.Validators.Inferred == nil {
			return errors.New("validators.inferred is required: false for a kind whose released validation has no contract limits, and true otherwise")
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
	exceptions := iacoverrides.Exceptions{API: f.API.API, Fields: map[string]map[string]iacoverrides.Field{}}
	for _, line := range f.Lines() {
		if !f.Types[line.Component].Fields[line.Field].ReadOnly {
			continue
		}
		if exceptions.Fields[line.Component] == nil {
			exceptions.Fields[line.Component] = map[string]iacoverrides.Field{}
		}
		exceptions.Fields[line.Component][line.Field] = iacoverrides.Field{ReadOnly: true}
	}
	return exceptions.Policy()
}

// Inferred reports whether the markers have the contract limits: true unless the file says
// validators.inferred: false.
func (f *File) Inferred() bool {
	return f == nil || f.Validators == nil || f.Validators.Inferred == nil || *f.Validators.Inferred
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
