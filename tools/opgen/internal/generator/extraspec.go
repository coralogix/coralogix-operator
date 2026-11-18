package generator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// wrapperField is a field beside the request wrapper that the CR holds (api.requestValues source
// field). It is an optional string in the spec, with the description of the contract.
type wrapperField struct {
	Name, Description string
}

// requestFields reads the request fields of api.requestValues: the description of each field
// source, and that each field and omit source is an optional string or an optional field.
func wrapperFields(doc docType, r *model.Resource, file *overrides.File) ([]wrapperField, error) {
	if file == nil || r.Policy.RequestWrapper == "" {
		return nil, nil
	}
	descriptions := map[string]string{}
	inRequest := map[string][]string{}
	for _, request := range []struct {
		name string
		op   model.Operation
	}{{"create", r.Create}, {"update", r.Update}} {
		body := requestBodySchema(doc, request.op)
		properties := requestProperties(body)
		for _, name := range sortedKeys(file.API.RequestValues[request.name]) {
			value := file.API.RequestValues[request.name][name]
			if value.Source != "field" && value.Source != "omit" {
				continue
			}
			location := overrides.FileName + ":api.requestValues." + request.name + "." + name
			property := properties[name]
			if property == nil {
				continue // checkRequestValues reports it
			}
			if schemaRequiresProperty(body, name) {
				return nil, &shapeError{code: codeOverrideInvalid, path: location, msg: "the request requires the field, so the CR must always set it"}
			}
			if value.Source == "omit" {
				continue
			}
			schema := property.Schema()
			if schema == nil || !slices.Equal(schema.Type, []string{"string"}) {
				return nil, &shapeError{code: codeOverrideInvalid, path: location, msg: "a field source needs a string field"}
			}
			inRequest[name] = append(inRequest[name], request.name)
			if _, ok := descriptions[name]; !ok && schema.Description != "" {
				descriptions[name] = schema.Description
			}
		}
	}
	var out []wrapperField
	for _, name := range sortedKeys(inRequest) {
		if len(inRequest[name]) != 2 {
			// The CR field is sent in both requests, or Update would clear a value that Create set.
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":api.requestValues." + inRequest[name][0] + "." + name,
				msg: "a field source must be in create and update"}
		}
		out = append(out, wrapperField{Name: name, Description: descriptions[name]})
	}
	return out, nil
}

// contentData is what the templates need for the content key.
type contentData struct {
	DecodePath, DecodeFunc string
}

// extraSpec adds the spec fields that do not come from the resource: the request fields, the
// fields of the content key, and the handwritten fields of the spec key.
func (b *builder) extraSpec(r *model.Resource, d *kindData, fields []wrapperField) error {
	if b.file == nil {
		return nil
	}
	for _, f := range fields {
		d.Spec.Fields = append(d.Spec.Fields, &crField{
			API: f.Name, JSON: f.Name, GoName: goName(f.Name), GoType: "*string", Optional: true,
			Doc: docLines(f.Description), Markers: []string{"+optional"}, typ: &model.Type{Kind: model.String}, synthetic: true,
		})
	}
	if content := b.file.Content; content != nil {
		if len(d.Spec.Fields) != len(fields) {
			return unsupported("content", "the content key holds the whole resource: every modelled field must be read-only")
		}
		if r.Policy.RequestWrapper == "" || d.CreateResourceBody != r.Name || d.UpdateResourceBody != r.Name {
			return unsupported("content", "the content key needs a request wrapper that holds the resource")
		}
		d.Content = &contentData{DecodePath: content.DecodePackage(), DecodeFunc: content.DecodeFunc()}
		d.Spec.Markers = append(d.Spec.Markers,
			`+kubebuilder:validation:XValidation:rule="!(has(self.json) && has(self.configMapRef))", message="Only one of json or configMapRef can be declared at the same time"`)
		d.Spec.Fields = append(d.Spec.Fields,
			&crField{API: "json", JSON: "json", GoName: "Json", GoType: "*string", Optional: true, synthetic: true,
				Doc:     []string{"Json is the " + r.Name + " as JSON text. It wins over gzipJson and configMapRef."},
				Markers: []string{"+optional"}},
			&crField{API: "gzipJson", JSON: "gzipJson", GoName: "GzipJson", GoType: "[]byte", Optional: true, synthetic: true,
				Doc:     []string{"GzipJson is the JSON text compressed with gzip. It is base64-encoded in YAML. It wins over configMapRef."},
				Markers: []string{"+optional"}},
			&crField{API: "configMapRef", JSON: "configMapRef", GoName: "ConfigMapRef", GoType: "*corev1.ConfigMapKeySelector", Optional: true, synthetic: true,
				Doc:     []string{"ConfigMapRef names the key of a ConfigMap in the namespace of the CR that holds the JSON text."},
				Markers: []string{"+optional"}},
		)
	}
	for _, name := range sortedKeys(b.file.Spec) {
		d.Spec.Fields = append(d.Spec.Fields, &crField{
			API: name, JSON: name, GoName: goName(name), GoType: "*" + b.file.Spec[name].Type, Optional: true, synthetic: true,
			Markers: []string{"+optional"},
		})
	}
	return nil
}

// contentMethod is the spec method that reads and decodes the content. The Create and Update
// requests call it in place of a conversion of the spec fields.
func (b *builder) contentMethod(d *kindData) string {
	var s strings.Builder
	fmt.Fprintf(&s, `
// DecodeContent reads the JSON text of the %[1]s from json, gzipJson, or configMapRef, in this
// order, and decodes it into the SDK type. A hook can call it to read the content.
func (in *%[1]sSpec) DecodeContent(ctx context.Context, namespace string) (*%[2]s.%[3]s, error) {
	var text []byte
	switch {
	case in.Json != nil:
		text = []byte(*in.Json)
	case in.GzipJson != nil:
		reader, err := gzip.NewReader(bytes.NewReader(in.GzipJson))
		if err != nil {
			return nil, fmt.Errorf("failed to gunzip gzipJson: %%w", err)
		}
		if text, err = io.ReadAll(reader); err != nil {
			return nil, fmt.Errorf("failed to gunzip gzipJson: %%w", err)
		}
	case in.ConfigMapRef != nil:
		configMap := &corev1.ConfigMap{}
		if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: in.ConfigMapRef.Name}, configMap); err != nil {
			return nil, err
		}
		value, ok := configMap.Data[in.ConfigMapRef.Key]
		if !ok {
			return nil, fmt.Errorf("cannot find key %%q in config map %%q", in.ConfigMapRef.Key, in.ConfigMapRef.Name)
		}
		text = []byte(value)
	default:
		return nil, errors.New("json, gzipJson, or configMapRef is required")
	}
	out := new(%[2]s.%[3]s)
	if err := contentdecode.%[4]s(text, out); err != nil {
		return nil, fmt.Errorf("failed to decode the JSON content: %%w", err)
	}
	return out, nil
}
`, d.Kind, d.SDKAlias, d.CreateResourceBody, d.Content.DecodeFunc)
	return s.String()
}
