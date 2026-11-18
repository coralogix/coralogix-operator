package generator

import (
	"bytes"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// The content key makes the CR hold the resource as JSON text. opgen then models only the id of
// the resource, and the properties that have a field line: the CRD has no schema for the rest.
// The shared module still checks the lifecycle, the requests, and the id. The Go SDK type keeps
// every field, and the generated conversion decodes the JSON text into it.

// opaqueKeys are the keys of the resource component that an opaque resource keeps.
var opaqueKeys = []string{"title", "description", "type", "externalDocs"}

// opaqueDocument returns the OpenAPI document with the resource component cut down to the
// properties that opgen models for a kind with the content key.
func opaqueDocument(raw []byte, resource string, file *overrides.File) (docType, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("read the OpenAPI document: %w", err)
	}
	component := mappingPath(&root, "components", "schemas", resource)
	if component == nil {
		return nil, &shapeError{code: "OPERATOR_SHAPE_UNSUPPORTED", path: "components.schemas." + resource, msg: "the content key needs the resource component"}
	}
	keep := map[string]bool{}
	if id := file.API.IDProperty; id != "" {
		keep[id] = true
	}
	for field := range file.Types[resource].Fields {
		keep[field] = true
	}
	var content []*yaml.Node
	for i := 0; i+1 < len(component.Content); i += 2 {
		key, value := component.Content[i], component.Content[i+1]
		switch {
		case slices.Contains(opaqueKeys, key.Value):
			content = append(content, key, value)
		case key.Value == "properties" && value.Kind == yaml.MappingNode:
			kept := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			for j := 0; j+1 < len(value.Content); j += 2 {
				if keep[value.Content[j].Value] {
					kept.Content = append(kept.Content, value.Content[j], value.Content[j+1])
				}
			}
			content = append(content, key, kept)
		}
		// required, allOf, oneOf, and anyOf name the properties that are cut.
	}
	component.Content = content
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("write the OpenAPI document: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return model.Load(out.Bytes())
}

// mappingPath returns the mapping value at the keys, or nil.
func mappingPath(n *yaml.Node, keys ...string) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) != 0 {
		n = n.Content[0]
	}
	for _, key := range keys {
		if n.Kind != yaml.MappingNode {
			return nil
		}
		var next *yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				next = n.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	return n
}

// documents loads the OpenAPI document once, and once more for each kind with the content key.
type documents struct {
	raw    []byte
	shared docType
}

func (d *documents) forKind(entry kindEntry, file *overrides.File) (docType, error) {
	if file == nil || file.Content == nil {
		if d.shared == nil {
			doc, err := model.Load(d.raw)
			if err != nil {
				return nil, err
			}
			d.shared = doc
		}
		return d.shared, nil
	}
	return opaqueDocument(d.raw, entry.Resource, file)
}
