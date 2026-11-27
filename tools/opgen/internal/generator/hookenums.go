package generator

import (
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// withHookEnums lets the shared model read the enums below a field that a hook sets (setByHook) with
// any value prefix. The generator renders nothing for such a field, so no enums line can name the
// spelling of its values: the line would change no generated code, and the released CR keeps its
// own type. A field of the CR that is not below a hook field keeps the strict prefix rule.
func withHookEnums(doc docType, entry kindEntry, file *overrides.File, policy model.Policy) model.Policy {
	if file == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return policy
	}
	found := map[string]bool{}
	for _, line := range file.Lines() {
		if line.Component != entry.Resource || !file.Field(line.Component, line.Field).SetByHook {
			continue
		}
		seen := map[string]bool{}
		if proxy := doc.Components.Schemas.GetOrZero(entry.Resource); proxy != nil && proxy.Schema() != nil && proxy.Schema().Properties != nil {
			if property := proxy.Schema().Properties.GetOrZero(line.Field); property != nil {
				collectEnumComponents(property, seen, found)
			}
		}
		for _, body := range requestBodies(doc, entry.Resource, file) {
			if property := requestProperties(body)[line.Field]; property != nil {
				collectEnumComponents(property, seen, found)
			}
		}
	}
	for _, name := range sortedKeys(found) {
		if !slices.Contains(policy.EnumAnyPrefix, name) {
			policy.EnumAnyPrefix = append(policy.EnumAnyPrefix, name)
		}
	}
	return policy
}

// collectEnumComponents adds the name of each enum component that the schema reaches.
func collectEnumComponents(proxy *base.SchemaProxy, seen, out map[string]bool) {
	if proxy == nil {
		return
	}
	schema := proxy.Schema()
	if schema == nil {
		return
	}
	if reference := proxy.GetReference(); reference != "" {
		name := strings.TrimPrefix(reference, "#/components/schemas/")
		if seen[name] {
			return
		}
		seen[name] = true
		if len(schema.Enum) != 0 {
			out[name] = true
		}
	}
	if schema.Properties != nil {
		for _, property := range schema.Properties.FromOldest() {
			collectEnumComponents(property, seen, out)
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		collectEnumComponents(schema.AdditionalProperties.A, seen, out)
	}
	if schema.Items != nil && schema.Items.IsA() {
		collectEnumComponents(schema.Items.A, seen, out)
	}
	for _, group := range [][]*base.SchemaProxy{schema.AllOf, schema.OneOf, schema.AnyOf} {
		for _, part := range group {
			collectEnumComponents(part, seen, out)
		}
	}
}
