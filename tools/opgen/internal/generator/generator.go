// Package generator validates and generates operator kinds from the API contract.
package generator

import (
	"errors"
	"fmt"
	"os"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"golang.org/x/tools/go/packages"

	"github.com/coralogix/coralogix-management-sdk/iac/issue"
	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-management-sdk/iac/source"
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
}

type docType = *v3.Document

// validated is one kind that passed every check before the SDK check.
type validated struct {
	entry    kindEntry
	resource *model.Resource
	refs     []sdkcheck.Ref
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
	doc, err := model.Load(data)
	if err != nil {
		return &EligibilityError{Report: issue.Report{{Code: "OPENAPI_INVALID", Location: "openapi", Message: err.Error(), Remediation: "Correct the OpenAPI document."}}}
	}
	v, err := validate(doc, entry, source.SDKModule)
	if err != nil {
		return err
	}
	_, err = renderAll([]*validated{v})
	return err
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
			v, err := validate(doc, entry, input.SDKModule)
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
	out, err := renderAll(all)
	if err != nil {
		return err
	}
	if report := nameCollisions(root, out.files); len(report) != 0 {
		return &EligibilityError{Report: report.Normalize()}
	}
	return publish(root, out)
}

// validate runs the contract checks of iac and the operator checks on one kind.
func validate(doc docType, entry kindEntry, sdkModule string) (*validated, error) {
	var policy model.Policy
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
	v := &validated{entry: entry, resource: r, refs: refs}
	if _, err := renderAll([]*validated{v}); err != nil {
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

// renderAll renders every kind.
func renderAll(all []*validated) (output, error) {
	out := output{files: map[string][]byte{}}
	var report issue.Report
	for _, v := range all {
		files, err := renderOne(v)
		if err != nil {
			report = append(report, eligibility(v.entry, err).Report...)
			continue
		}
		for path, data := range files {
			out.files[path] = data
		}
	}
	if len(report) != 0 {
		return output{}, &EligibilityError{Report: report.Normalize()}
	}
	return out, nil
}

func renderOne(v *validated) (map[string][]byte, error) {
	d, err := buildKind(v.resource, v.refs, v.entry)
	if err != nil {
		return nil, err
	}
	files, err := renderKind(d)
	if err != nil {
		return nil, &shapeError{code: "RENDERER_OUTPUT_INVALID", path: "renderer.output", msg: err.Error()}
	}
	return files, nil
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
