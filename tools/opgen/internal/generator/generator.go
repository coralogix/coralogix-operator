// Package generator validates and generates operator kinds from the API contract.
package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	entry    kindEntry
	resource *model.Resource
	refs     []sdkcheck.Ref
	e2e      *e2eFile
	file     *overrides.File // nil for a kind without a behavior-overrides file
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
	return publish(root, out)
}

// validate runs the contract checks of iac and the operator checks on one kind.
func validate(doc docType, entry kindEntry, sdkModule, module string, e2e *e2eFile, file *overrides.File) (*validated, error) {
	if err := overrideIssues(doc, entry, file); err != nil {
		return nil, asEligibility(entry, err)
	}
	policy := file.Policy()
	if report := model.ValidateWithPolicy(doc, entry.Resource, model.OperationIDs{}, policy); len(report) != 0 {
		return nil, &EligibilityError{Report: located(entry, report)}
	}
	r, err := model.BuildWithPolicy(doc, entry.Resource, model.OperationIDs{}, policy)
	if err != nil {
		return nil, fmt.Errorf("%s: build the validated resource: %w", entry.Kind, err)
	}
	tag, err := sdkcheck.ResourceTag(doc, r)
	if err != nil {
		return nil, asEligibility(entry, &shapeError{code: "OPERATION_TAG_INCOMPATIBLE", path: "paths", msg: err.Error()})
	}
	refs, err := resolveSDKNames(r, tag, sdkModule)
	if err != nil {
		return nil, asEligibility(entry, &shapeError{code: "SDK_SHAPE_UNSUPPORTED", path: "resource", msg: err.Error()})
	}
	v := &validated{entry: entry, resource: r, refs: refs, e2e: e2e, file: file}
	if _, err := renderAll([]*validated{v}, sdkModule, module); err != nil {
		return nil, err
	}
	return v, nil
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
	return out, nil
}

func renderOne(v *validated, module string) (map[string][]byte, *kindData, error) {
	d, err := buildKind(v.resource, v.refs, v.entry, module, v.file)
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
				_, ok = proxy.Schema().Properties.Get(line.Field)
			}
		}
		if !ok {
			return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String(), msg: "the contract has no such property"}
		}
	}
	return nil
}

// refKindIssues checks that each ref line names a kind of the operator: a generated kind, or a kind
// whose types file exists.
func refKindIssues(root string, kinds []kindEntry, file *overrides.File) error {
	for _, line := range file.Lines() {
		kind := file.Ref(line.Component, line.Field).Kind
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
