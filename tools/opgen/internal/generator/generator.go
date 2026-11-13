// Package generator validates and generates operator kinds from the API contract.
package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"golang.org/x/tools/go/packages"

	"github.com/coralogix/coralogix-management-sdk/iac/issue"
	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-management-sdk/iac/source"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// EligibilityError reports every reason that a kind cannot be generated.
type EligibilityError struct {
	Report issue.Report
}

func (e *EligibilityError) Error() string {
	return e.Report.Text("Kind is not eligible for operator generation")
}

// CheckOptions selects one resource in a local candidate OpenAPI document.
type CheckOptions struct {
	Kind        string // the custom resource kind; "" uses the resource name
	Resource    string
	OpenAPIPath string
	// E2EPath is the e2e.yaml file of the kind, "" for none.
	E2EPath string
	// OverridesPath is the behavior-overrides.yaml file of the kind, "" for none.
	OverridesPath string
}

type docType = *v3.Document

// validated is one kind that passed every check before the SDK check.
type validated struct {
	entry                  kindEntry
	resource               *model.Resource
	refs                   []sdkcheck.Ref
	e2e                    *e2eFile
	file                   *overrides.File // nil for a kind without a behavior-overrides file
	updateIDOutsideWrapper bool
}

// Generate resolves the SDK that the operator pins and publishes every kind of the kinds file.
func Generate(start string) error {
	input, err := source.Resolve(start)
	if err != nil {
		return err
	}
	return generateFromInput(input, input.ModuleRoot, input.ModuleRoot)
}

// Check validates one resource of a candidate OpenAPI document. It renders the files, so it finds
// every problem that generate finds, but it writes nothing and it does not check the SDK.
func Check(options CheckOptions) error {
	if options.Resource == "" || options.OpenAPIPath == "" {
		return errors.New("the resource and the candidate OpenAPI file are required")
	}
	data, err := os.ReadFile(options.OpenAPIPath)
	if err != nil {
		return fmt.Errorf("read candidate OpenAPI %s: %w", options.OpenAPIPath, err)
	}
	entry := kindEntry{Kind: options.Kind, Resource: options.Resource}
	if entry.Kind == "" {
		entry.Kind = options.Resource
	}
	e2e := &e2eFile{}
	if options.E2EPath != "" {
		if e2e, err = readE2E(options.E2EPath); err != nil {
			return asEligibility(entry, err)
		}
	}
	file, err := readOverrides(options.OverridesPath)
	if err != nil {
		return asEligibility(entry, err)
	}
	doc, err := model.Load(data)
	if err != nil {
		return &EligibilityError{Report: issue.Report{{Code: "OPENAPI_INVALID", Location: "openapi", Message: err.Error(), Remediation: "Correct the OpenAPI document."}}}
	}
	v, err := validate(doc, entry, source.SDKModule, "candidate.invalid/coralogix-operator", e2e, file)
	if err != nil {
		return err
	}
	if _, err := renderAll([]*validated{v}, source.SDKModule, "candidate.invalid/coralogix-operator"); err != nil {
		return err
	}
	return nil
}

// generateFromInput generates every listed kind into the operator at root. loadDir is the module
// that loads the SDK packages. It is the test seam for a synthetic contract and SDK.
func generateFromInput(input source.Input, root, loadDir string) error {
	kinds, err := readKinds(root)
	if err != nil {
		return err
	}
	var all []*validated
	var report issue.Report
	if len(kinds) != 0 {
		doc, err := model.Load(input.OpenAPI)
		if err != nil {
			return fmt.Errorf("read the OpenAPI of the SDK: %w", err)
		}
		for _, entry := range kinds {
			e2e, err := readE2E(filepath.Join(root, entry.dir(), e2eFileName))
			if err != nil {
				report = append(report, eligibility(entry, err).Report...)
				continue
			}
			file, err := readOverrides(filepath.Join(root, entry.dir(), overrides.FileName))
			if err == nil {
				err = refKindIssues(root, kinds, file)
			}
			if err != nil {
				report = append(report, eligibility(entry, err).Report...)
				continue
			}
			v, err := validate(doc, entry, input.SDKModule, input.Module, e2e, file)
			if err != nil {
				report = append(report, eligibility(entry, err).Report...)
				continue
			}
			all = append(all, v)
		}
	}
	if len(report) != 0 {
		return &EligibilityError{Report: report.Normalize()}
	}
	if err := checkSDK(all, input, loadDir); err != nil {
		return err
	}
	out, err := renderAll(all, input.SDKModule, input.Module)
	if err != nil {
		return err
	}
	if report := nameCollisions(root, out.files); len(report) != 0 {
		return &EligibilityError{Report: report.Normalize()}
	}
	if report := duplicateEntries(root, out); len(report) != 0 {
		return &EligibilityError{Report: report.Normalize()}
	}
	return publish(root, out)
}

// validate runs the contract checks of iac and the operator checks on one kind.
func validate(doc docType, entry kindEntry, sdkModule, module string, e2e *e2eFile, file *overrides.File) (*validated, error) {
	if err := overrideIssues(doc, entry, file); err != nil {
		return nil, asEligibility(entry, err)
	}
	policy := withFoldKept(doc, entry, file, file.Policy())
	if report := model.ValidateWithPolicy(doc, entry.Resource, model.OperationIDs{}, policy); len(report) != 0 {
		return nil, &EligibilityError{Report: located(entry, report)}
	}
	r, err := model.BuildWithPolicy(doc, entry.Resource, model.OperationIDs{}, policy)
	if err != nil {
		return nil, fmt.Errorf("%s: build the validated resource: %w", entry.Kind, err)
	}
	if err := nestedSkipIssues(doc, entry, file, policy); err != nil {
		return nil, asEligibility(entry, err)
	}
	if err := checkRequestValues(doc, r, file); err != nil {
		return nil, asEligibility(entry, err)
	}
	tag, err := sdkcheck.ResourceTag(doc, r)
	if err != nil {
		return nil, asEligibility(entry, &shapeError{code: "OPERATION_TAG_INCOMPATIBLE", path: "paths", msg: err.Error()})
	}
	outside := updateIDOutsideWrapper(doc, r)
	refs, err := resolveSDKNames(r, tag, sdkModule, outside, file)
	if err != nil {
		return nil, asEligibility(entry, &shapeError{code: "SDK_SHAPE_UNSUPPORTED", path: "resource", msg: err.Error()})
	}
	v := &validated{entry: entry, resource: r, refs: refs, e2e: e2e, file: file, updateIDOutsideWrapper: outside}
	if _, err := renderAll([]*validated{v}, sdkModule, module); err != nil {
		return nil, err
	}
	return v, nil
}

// withFoldKept exempts the folded properties of a createAs line from the requiredness check of the
// shared module. Create takes the value flat and optional, and Update requires it inside the
// object, so the module cannot tell what a client sends when it leaves the value out. The
// generated Update builds the object only when a flat field is set, and the API keeps its stored
// value otherwise. That is what the module accepts for a property that the resource response
// requires and that a client keeps from state: computed and keepState. The property stays
// required in Update, so the SDK type is the one of the contract.
func withFoldKept(doc docType, entry kindEntry, file *overrides.File, policy model.Policy) model.Policy {
	if file == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return policy
	}
	resource := doc.Components.Schemas.GetOrZero(entry.Resource)
	if resource == nil || resource.Schema() == nil || resource.Schema().Properties == nil {
		return policy
	}
	policy.Computed, policy.KeepState = slices.Clone(policy.Computed), slices.Clone(policy.KeepState)
	for _, line := range file.Lines() {
		fold := file.Field(line.Component, line.Field).CreateAs
		if len(fold) == 0 || line.Component != entry.Resource {
			continue
		}
		property := resource.Schema().Properties.GetOrZero(line.Field)
		if property == nil || componentName(property) == "" {
			continue // an inline object has no component name for the policy
		}
		for nested := range fold {
			key := componentName(property) + "." + nested
			policy.Computed = append(policy.Computed, key)
			policy.KeepState = append(policy.KeepState, key)
		}
	}
	return policy
}

// componentName is the component that a schema proxy references, or "" for an inline schema.
func componentName(proxy *base.SchemaProxy) string {
	return strings.TrimPrefix(proxy.GetReference(), "#/components/schemas/")
}

// nestedSkipIssues checks the skip lines of nested components. A skipped field is never sent, so
// the resource must have a type of the component with that field, a request must send the field,
// and no request may require it. The checks read a model without the nested skips: the shared
// model prunes a skipped field, and a request can name its own component for the same object, so
// the component name of the line is the one of the merged type.
func nestedSkipIssues(doc docType, entry kindEntry, file *overrides.File, policy model.Policy) error {
	if file == nil {
		return nil
	}
	var lines []overrides.Line
	for _, line := range file.Lines() {
		if line.Component != entry.Resource && file.Field(line.Component, line.Field).Skip {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	policy.Skip = slices.DeleteFunc(slices.Clone(policy.Skip), func(skipped string) bool {
		return !strings.HasPrefix(skipped, entry.Resource+".")
	})
	full, err := model.BuildWithPolicy(doc, entry.Resource, model.OperationIDs{}, policy)
	if err != nil {
		return fmt.Errorf("%s: build the resource without nested skips: %w", entry.Kind, err)
	}
	for _, line := range lines {
		path := overrides.FileName + ":" + line.String() + ".skip"
		fields := componentFields(full, line.Component, line.Field)
		if len(fields) == 0 {
			return &shapeError{code: codeOverrideUnused, path: path, msg: "the resource has no type of this component with this field"}
		}
		sent := false
		for _, field := range fields {
			for _, attrs := range []*model.Attrs{field.Create, field.Update} {
				if attrs == nil || attrs.ReadOnly {
					continue
				}
				sent = true
				if attrs.Required {
					return &shapeError{code: codeOverrideInvalid, path: path, msg: "a request requires this field, so the CR must send it"}
				}
			}
		}
		if !sent {
			return &shapeError{code: codeOverrideUnused, path: path, msg: "Create and Update do not send this field"}
		}
	}
	return nil
}

// componentFields returns the fields with the given name of the types that come from the component.
func componentFields(r *model.Resource, component, name string) []*model.Field {
	var out []*model.Field
	seen := map[*model.Type]bool{}
	var walk func(*model.Type)
	walk = func(t *model.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		walk(t.Elem)
		if t.Schema == component {
			for _, f := range t.Fields {
				if f.Name == name {
					out = append(out, f)
				}
			}
		}
		for _, f := range t.Fields {
			walk(f.Type)
		}
	}
	for _, f := range r.Fields {
		walk(f.Type)
	}
	return out
}

// updateIDOutsideWrapper checks whether Update has the id beside its resource wrapper.
// The shared model describes the resource inside the wrapper but not sibling fields.
func updateIDOutsideWrapper(doc docType, r *model.Resource) bool {
	if !r.Policy.UpdateIDInBody || r.Policy.RequestWrapper == "" || doc.Paths == nil || doc.Paths.PathItems == nil {
		return false
	}
	return hasRequestProperty(requestBodySchema(doc, r.Update), r.ID())
}

func requestBodySchema(doc docType, operation model.Operation) *base.Schema {
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return nil
	}
	item, ok := doc.Paths.PathItems.Get(operation.Path)
	if !ok || item == nil {
		return nil
	}
	op, ok := item.GetOperations().Get(strings.ToLower(operation.Method))
	if !ok || op == nil || op.RequestBody == nil || op.RequestBody.Content == nil {
		return nil
	}
	media := op.RequestBody.Content.GetOrZero("application/json")
	if media == nil || media.Schema == nil {
		return nil
	}
	return media.Schema.Schema()
}

func checkRequestValues(doc docType, r *model.Resource, file *overrides.File) error {
	if r.Policy.RequestWrapper == "" {
		return nil
	}
	for _, request := range []struct {
		name string
		op   model.Operation
	}{{"create", r.Create}, {"update", r.Update}} {
		properties := requestProperties(requestBodySchema(doc, request.op))
		values := map[string]overrides.RequestValue{}
		if file != nil {
			values = file.API.RequestValues[request.name]
		}
		for _, name := range sortedKeys(values) {
			if _, ok := properties[name]; !ok {
				return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":api.requestValues." + request.name + "." + name,
					msg: "the request body has no such sibling field"}
			}
		}
		for _, name := range sortedKeys(properties) {
			property := properties[name]
			if name == r.Policy.RequestWrapper || (request.name == "update" && r.Policy.UpdateIDInBody && name == r.ID()) ||
				(request.name == "update" && r.MaskInBody && name == r.UpdateMask) ||
				(property.Schema() != nil && property.Schema().ReadOnly != nil && *property.Schema().ReadOnly) {
				continue
			}
			if _, ok := values[name]; !ok {
				return unsupported("api."+request.name+"."+name, "the request field beside the wrapper needs api.requestValues.%s.%s", request.name, name)
			}
		}
	}
	return nil
}

func requestProperties(schema *base.Schema) map[string]*base.SchemaProxy {
	out := map[string]*base.SchemaProxy{}
	if schema == nil {
		return out
	}
	if schema.Properties != nil {
		for name, property := range schema.Properties.FromOldest() {
			out[name] = property
		}
	}
	for _, part := range schema.AllOf {
		for name, property := range requestProperties(part.Schema()) {
			out[name] = property
		}
	}
	return out
}

func hasRequestProperty(schema *base.Schema, name string) bool {
	if schema == nil {
		return false
	}
	if schema.Properties != nil {
		if _, ok := schema.Properties.Get(name); ok {
			return true
		}
	}
	for _, part := range schema.AllOf {
		if hasRequestProperty(part.Schema(), name) {
			return true
		}
	}
	return false
}

// checkSDK checks that the pinned SDK has every name that the generated code uses. It sets the
// SDK names that the code uses, for example a value type in place of a pointer.
func checkSDK(all []*validated, input source.Input, loadDir string) error {
	var refs []sdkcheck.Ref
	for _, v := range all {
		refs = append(refs, v.refs...)
	}
	if len(refs) == 0 {
		return nil
	}
	loaded, err := sdkcheck.Load(refs, loadDir)
	if err != nil {
		return err
	}
	var report issue.Report
	for _, v := range all {
		report = append(report, located(v.entry, sdkIssues(v.refs, loaded, input))...)
	}
	if len(report) != 0 {
		return &EligibilityError{Report: report.Normalize()}
	}
	return nil
}

func sdkIssues(refs []sdkcheck.Ref, loaded map[string]*packages.Package, input source.Input) issue.Report {
	return sdkcheck.Issues(refs, loaded, func(sdkcheck.Ref) sdkcheck.Source {
		return sdkcheck.Source{Module: input.SDKModule, Version: input.SDKVersion, Dir: input.SDKDir,
			Remediation: "Load the Go SDK and the OpenAPI of the version that the operator go.mod pins."}
	})
}

// renderAll renders every kind and the files that list them.
func renderAll(all []*validated, sdkModule, module string) (output, error) {
	out := output{files: map[string][]byte{}}
	var kinds []*kindData
	var report issue.Report
	for _, v := range all {
		files, d, err := renderOne(v, module)
		if err != nil {
			report = append(report, eligibility(v.entry, err).Report...)
			continue
		}
		kinds = append(kinds, d)
		for path, data := range files {
			out.files[path] = data
		}
	}
	if len(report) != 0 {
		return output{}, &EligibilityError{Report: report.Normalize()}
	}
	shared, err := renderShared(module, sdkModule+"/go/openapi/cxsdk", kinds)
	if err != nil {
		return output{}, err
	}
	for path, data := range shared.files {
		out.files[path] = data
	}
	out.blocks = shared.blocks
	out.entries = shared.entries
	return out, nil
}

func renderOne(v *validated, module string) (map[string][]byte, *kindData, error) {
	d, err := buildKind(v.resource, v.refs, v.entry, module, v.file, v.updateIDOutsideWrapper)
	if err != nil {
		return nil, nil, err
	}
	ix, err := indexRefs(v.refs)
	if err != nil {
		return nil, nil, err
	}
	full, _, e2e, err := buildValues(d, ix, v.e2e)
	if err != nil {
		return nil, nil, err
	}
	d.E2E = e2e
	files, err := renderKind(d, full)
	if err != nil {
		return nil, nil, &shapeError{code: "RENDERER_OUTPUT_INVALID", path: "renderer.output", msg: err.Error()}
	}
	return files, d, nil
}

// readOverrides reads the behavior-overrides file of a kind. It returns nil when path is "" or the
// file does not exist: the kind follows the contract alone.
func readOverrides(path string) (*overrides.File, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	file, err := overrides.Parse(data)
	if err != nil {
		return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName, msg: err.Error()}
	}
	return file, nil
}

// overrideIssues checks that the file is for the resource of the kind, and that each line names a
// property of a component of the contract. The renderer then checks that each line changes the
// output.
func overrideIssues(doc docType, entry kindEntry, file *overrides.File) error {
	if file == nil {
		return nil
	}
	if file.Resource != entry.Resource {
		return &shapeError{code: "OVERRIDE_RESOURCE_MISMATCH", path: overrides.FileName,
			msg: fmt.Sprintf("the file is for resource %q, but the kind is resource %q", file.Resource, entry.Resource)}
	}
	for _, line := range file.Lines() {
		ok := false
		if doc.Components != nil && doc.Components.Schemas != nil {
			if proxy, found := doc.Components.Schemas.Get(line.Component); found && proxy.Schema() != nil && proxy.Schema().Properties != nil {
				property, found := proxy.Schema().Properties.Get(line.Field)
				ok = found
				change := file.Field(line.Component, line.Field)
				if found && change.Skip && property.Schema() != nil && property.Schema().ReadOnly != nil && *property.Schema().ReadOnly {
					return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".skip", msg: "the contract already marks this field read-only"}
				}
				if found && change.Skip && line.Component == entry.Resource && !requestHasField(doc, entry.Resource, file, line.Field) {
					return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".skip", msg: "Create and Update already omit this field"}
				}
				if found && change.ReadOnly && property.Schema() != nil && property.Schema().ReadOnly != nil && *property.Schema().ReadOnly {
					return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".readOnly", msg: "the contract already marks this field read-only"}
				}
				if found && change.Optional &&
					((line.Component == entry.Resource && !requestRequiresField(doc, entry.Resource, file, line.Field, change.RequestAs)) ||
						(line.Component != entry.Resource && !slices.Contains(proxy.Schema().Required, line.Field))) {
					return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".optional", msg: "the contract already makes this field optional"}
				}
			}
		}
		if !ok && line.Component == entry.Resource && file.Field(line.Component, line.Field).Skip {
			// A field that only the requests have has no property in the resource component.
			requestOnly, err := requestOnlyField(doc, entry, file, line.Field)
			if err != nil {
				return err
			}
			ok = requestOnly
		}
		if !ok {
			return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String(), msg: "the contract has no such property"}
		}
	}
	for name, enum := range file.Enums {
		location := overrides.FileName + ":enums." + name
		if doc.Components == nil || doc.Components.Schemas == nil {
			return &shapeError{code: codeOverrideUnused, path: location, msg: "the contract has no component schemas"}
		}
		proxy := doc.Components.Schemas.GetOrZero(name)
		if proxy == nil || proxy.Schema() == nil || len(proxy.Schema().Enum) == 0 {
			return &shapeError{code: codeOverrideUnused, path: location, msg: "the contract has no such enum"}
		}
		var contract []string
		for _, value := range proxy.Schema().Enum {
			contract = append(contract, value.Value)
		}
		for _, value := range append(slices.Clone(enum.Values), enum.Rejected...) {
			if !slices.Contains(contract, value) {
				return &shapeError{code: codeOverrideInvalid, path: location, msg: fmt.Sprintf("the API enum has no value %q", value)}
			}
		}
		hasZero := len(contract) != 0 && strings.HasSuffix(contract[0], "_UNSPECIFIED")
		if hasZero != (enum.Zero != "") {
			return &shapeError{code: codeOverrideInvalid, path: location + ".zero", msg: "zero must name the *_UNSPECIFIED value, and only that value"}
		}
		if hasZero && (slices.Contains(enum.Values, contract[0]) || slices.Contains(enum.Rejected, contract[0])) {
			return &shapeError{code: codeOverrideInvalid, path: location, msg: "the zero API value belongs only in zero"}
		}
		for i, value := range contract {
			if i == 0 && hasZero {
				continue
			}
			if !slices.Contains(enum.Values, value) && !slices.Contains(enum.Rejected, value) {
				return &shapeError{code: codeOverrideInvalid, path: location, msg: fmt.Sprintf("the API value %q is neither accepted nor rejected", value)}
			}
		}
	}
	return nil
}

// requestOnlyField reports whether Create or Update has a top-level field that the resource
// component does not have, such as a teamId that the API reads only on write. A skip line can name it.
func requestOnlyField(doc docType, entry kindEntry, file *overrides.File, field string) (bool, error) {
	fields, err := model.RequestOnlyFields(doc, entry.Resource, model.OperationIDs{}, file.Policy())
	if err != nil {
		return false, &shapeError{code: codeOverrideInvalid, path: overrides.FileName, msg: err.Error()}
	}
	return slices.Contains(fields, field), nil
}

// requestRequiresField reports whether Create or Update requires a top-level API property. The
// requests can hold the field at another path (requestAs): it is required when every segment of
// the path is required in its parent.
func requestRequiresField(doc docType, resource string, file *overrides.File, field, requestAs string) bool {
	path := []string{field}
	if requestAs != "" {
		path = strings.Split(requestAs, ".")
	}
	for _, body := range requestBodies(doc, resource, file) {
		if requiresPath(body, path) {
			return true
		}
	}
	return false
}

// requiresPath reports whether every segment of path is required in its parent object.
func requiresPath(schema *base.Schema, path []string) bool {
	for _, segment := range path {
		if !schemaRequiresProperty(schema, segment) {
			return false
		}
		proxy := requestProperties(schema)[segment]
		if proxy == nil {
			return true // a required property that the schema does not describe
		}
		schema = proxy.Schema()
	}
	return true
}

// requestHasField reports whether Create or Update can send a top-level API property.
func requestHasField(doc docType, resource string, file *overrides.File, field string) bool {
	return requestField(doc, resource, file, field, hasRequestProperty)
}

func requestField(doc docType, resource string, file *overrides.File, field string, matches func(*base.Schema, string) bool) bool {
	for _, body := range requestBodies(doc, resource, file) {
		if matches(body, field) {
			return true
		}
	}
	return false
}

// requestBodies returns the schemas of the Create and Update request bodies. With a request
// wrapper, the schema of the wrapper property is a body too.
func requestBodies(doc docType, resource string, file *overrides.File) []*base.Schema {
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return nil
	}
	var bodies []*base.Schema
	for _, item := range doc.Paths.PathItems.FromOldest() {
		for _, operation := range item.GetOperations().FromOldest() {
			id := operation.OperationId
			selected := strings.HasSuffix(id, "_Create"+resource) || strings.HasSuffix(id, "_Update"+resource) || strings.HasSuffix(id, "_Replace"+resource)
			if file.API.Operations != nil {
				selected = selected || id == file.API.Operations.Create || id == file.API.Operations.Update
			}
			if !selected || operation.RequestBody == nil || operation.RequestBody.Content == nil {
				continue
			}
			media := operation.RequestBody.Content.GetOrZero("application/json")
			if media == nil || media.Schema == nil {
				continue
			}
			body := media.Schema.Schema()
			if body == nil {
				continue
			}
			bodies = append(bodies, body)
			if wrapper := file.API.RequestWrapper; wrapper != "" && body.Properties != nil {
				if property := body.Properties.GetOrZero(wrapper); property != nil && property.Schema() != nil {
					bodies = append(bodies, property.Schema())
				}
			}
		}
	}
	return bodies
}

func schemaRequiresProperty(schema *base.Schema, property string) bool {
	if schema == nil {
		return false
	}
	if slices.Contains(schema.Required, property) {
		return true
	}
	for _, part := range schema.AllOf {
		if schemaRequiresProperty(part.Schema(), property) {
			return true
		}
	}
	return false
}

// refKindIssues checks that each ref line names a kind of the operator: a generated kind, or a kind
// whose types file exists.
func refKindIssues(root string, kinds []kindEntry, file *overrides.File) error {
	for _, line := range file.Lines() {
		ref := file.Ref(line.Component, line.Field)
		if ref == nil || ref.Form == "namedBackendOnly" {
			// A backend-only reference has no resourceRef arm: its kind only names the API resource.
			continue
		}
		kind := ref.Kind
		if slices.ContainsFunc(kinds, func(k kindEntry) bool { return k.Kind == kind }) {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, apiDir, strings.ToLower(kind)+"_types.go")); err != nil {
			return &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref.kind",
				msg: fmt.Sprintf("the operator has no kind %s", kind)}
		}
	}
	return nil
}

// eligibility turns an error of one kind into a report.
func eligibility(entry kindEntry, err error) *EligibilityError {
	var e *EligibilityError
	if errors.As(err, &e) {
		return e
	}
	return asEligibility(entry, err)
}

func asEligibility(entry kindEntry, err error) *EligibilityError {
	code, location := codeShapeUnsupported, "renderer"
	if s, ok := isShapeError(err); ok {
		code, location = s.code, s.path
	}
	remediation := map[string]string{
		codeShapeUnsupported:         "Use a contract shape that the operator renderer supports, or extend the renderer.",
		codeNameCollision:            "Rename the component or the field in the API contract.",
		codeUpdateOnlyField:          "Add `skip: true` for the field to " + overrides.FileName + " to leave it out of the CR and the requests, or extend the renderer to send an Update-only field.",
		codeE2EInvalid:               "Fix " + e2eFileName + ".",
		codeE2ERequired:              "Set the value in " + e2eFileName + ".",
		codeE2EUnused:                "Delete the line from " + e2eFileName + ", or fix the field path.",
		"OPERATION_TAG_INCOMPATIBLE": "Give every lifecycle operation exactly one matching SDK package tag.",
		"SDK_SHAPE_UNSUPPORTED":      "Use OpenAPI shapes that have deterministic generated Go SDK names and types.",
		"RENDERER_OUTPUT_INVALID":    "Use names and shapes that render as valid formatted Go code.",
	}[code]
	return &EligibilityError{Report: located(entry, issue.Report{{Code: code, Location: location, Message: err.Error(), Remediation: remediation}})}
}

// located prefixes each location with the kind, so a report of several kinds stays readable.
func located(entry kindEntry, report issue.Report) issue.Report {
	out := make(issue.Report, len(report))
	for i, is := range report {
		if !strings.HasPrefix(is.Location, entry.Kind+":") {
			is.Location = entry.Kind + ":" + is.Location
		}
		out[i] = is
	}
	return out.Normalize()
}
