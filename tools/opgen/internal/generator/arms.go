package generator

import (
	"fmt"
	"maps"
	"slices"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// withoutUnsupportedArms returns the contract with the unsupportedArms of each component removed:
// the property of the arm, and its alternative in the oneOf. The CR, the requests, the sample, and
// the e2e coverage then never see the arm. The SDK type keeps the field, which stays unset.
func withoutUnsupportedArms(raw []byte, file *overrides.File) ([]byte, error) {
	arms := file.UnsupportedArms()
	if len(arms) == 0 {
		return raw, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("read the OpenAPI document: %w", err)
	}
	for _, component := range slices.Sorted(maps.Keys(arms)) {
		path := overrides.FileName + ":types." + component + ".unsupportedArms"
		schema := mappingPath(&root, "components", "schemas", component)
		if schema == nil {
			return nil, &shapeError{code: codeOverrideUnused, path: path, msg: "the contract has no component " + component}
		}
		for _, arm := range arms[component] {
			if err := removeArm(schema, arm, path); err != nil {
				return nil, err
			}
		}
		if countArms(mapValue(schema, "oneOf")) == 0 {
			return nil, &shapeError{code: codeOverrideInvalid, path: path, msg: "the oneOf has no arm left"}
		}
	}
	return yaml.Marshal(&root)
}

// removeArm removes the property arm and its oneOf alternative from the component schema.
func removeArm(schema *yaml.Node, arm, path string) error {
	properties := mapValue(schema, "properties")
	if properties == nil || !deleteKey(properties, arm) {
		return &shapeError{code: codeOverrideUnused, path: path, msg: "the component has no property " + arm}
	}
	oneOf := mapValue(schema, "oneOf")
	if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
		return &shapeError{code: codeOverrideInvalid, path: path, msg: arm + " is not a oneOf arm; use skip for a plain field"}
	}
	found := false
	kept := oneOf.Content[:0:0]
	for _, alternative := range oneOf.Content {
		if requiresOnly(alternative, arm) {
			found = true
			continue
		}
		// The "no arm" alternative lists every arm under not.anyOf.
		if anyOf := mapValue(mapValue(alternative, "not"), "anyOf"); anyOf != nil && anyOf.Kind == yaml.SequenceNode {
			anyOf.Content = slices.DeleteFunc(anyOf.Content, func(n *yaml.Node) bool { return requiresOnly(n, arm) })
		}
		kept = append(kept, alternative)
	}
	if !found {
		return &shapeError{code: codeOverrideInvalid, path: path, msg: arm + " is not a oneOf arm; use skip for a plain field"}
	}
	oneOf.Content = kept
	if required := mapValue(schema, "required"); required != nil && required.Kind == yaml.SequenceNode {
		required.Content = slices.DeleteFunc(required.Content, func(n *yaml.Node) bool { return n.Value == arm })
	}
	return nil
}

// requiresOnly reports whether the schema n is {required: [arm]}, with an optional type: object.
func requiresOnly(n *yaml.Node, arm string) bool {
	if n == nil || n.Kind != yaml.MappingNode {
		return false
	}
	required := mapValue(n, "required")
	if required == nil || required.Kind != yaml.SequenceNode || len(required.Content) != 1 || required.Content[0].Value != arm {
		return false
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if key := n.Content[i].Value; key != "required" && key != "type" {
			return false
		}
	}
	return true
}

// countArms counts the alternatives of a oneOf that select an arm.
func countArms(oneOf *yaml.Node) int {
	if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
		return 0
	}
	n := 0
	for _, alternative := range oneOf.Content {
		if mapValue(alternative, "not") == nil {
			n++
		}
	}
	return n
}

// deleteKey removes key from the mapping m and reports whether m had it.
func deleteKey(m *yaml.Node, key string) bool {
	if m.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return true
		}
	}
	return false
}
