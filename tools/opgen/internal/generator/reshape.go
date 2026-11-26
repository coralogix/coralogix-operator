package generator

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// Three lines of behavior-overrides.yaml change the contract that the shared module reads, so opgen
// gives it the contract that they describe:
//
//   - api.operations with delete: none, and one POST as create and update: a singleton that a GET
//     reads and a POST on the same path sets. The path gets a PUT that is a copy of the POST, and a
//     DELETE with the responses of the GET. The shared module then sees a singleton with every
//     lifecycle step. upsertSDKNames maps the SDK names of the added operations to the POST and the
//     GET. The generated code calls the POST for Create and Update, and makes no Delete call.
//   - inline: true on a field of the resource: an object that only the API response nests, such as
//     {spec: {isActive}} for a request {isActive}. The resource component gets the fields of
//     the object in its place. inlineSDKNames moves their SDK names to the type of the object, and
//     the generated code that reads a response goes through the object.
//   - skip: true on an arm of a oneOf group of the resource that no Create or Update body has: the
//     group exists only in the response, where it names what the server can return. The arm and,
//     when one arm is left, the group go away, so the groups of the response and the requests agree.

// reshape returns the contract for the kind, and the object that holds each inlined field. A kind
// without an upsert or an inline line gets raw unchanged and no parents.
func reshape(raw []byte, resource string, file *overrides.File) ([]byte, map[string]overrides.InlineParent, error) {
	inlined := file.InlinedFields()
	file.PrunedArms = nil
	if !needsReshape(file) {
		return raw, nil, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, nil, fmt.Errorf("read the OpenAPI document: %w", err)
	}
	if len(root.Content) == 0 {
		return nil, nil, errors.New("read the OpenAPI document: it is empty")
	}
	doc := root.Content[0]
	schemas := mappingPath(doc, "components", "schemas")
	if file.Upserts() {
		if err := upsertOperations(doc, file.API.Operations.Get, file.API.Operations.Create); err != nil {
			return nil, nil, err
		}
	}
	if err := pruneSkippedArms(doc, schemas, resource, file); err != nil {
		return nil, nil, err
	}
	var parents map[string]overrides.InlineParent
	if len(inlined) != 0 {
		var err error
		if parents, err = inlineResponse(doc, schemas, resource, inlined, file); err != nil {
			return nil, nil, err
		}
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	if err := enc.Encode(&root); err != nil {
		return nil, nil, fmt.Errorf("write the OpenAPI document: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, nil, err
	}
	return out.Bytes(), parents, nil
}

// needsReshape reports whether the kind has a line that changes the contract.
func needsReshape(file *overrides.File) bool {
	if file.Upserts() || len(file.InlinedFields()) != 0 {
		return true
	}
	if file == nil || file.API.Operations == nil {
		return false
	}
	for _, field := range file.Types[file.Resource].Fields {
		if field.Skip {
			return true
		}
	}
	return false
}

// pruneSkippedArms removes the skipped arms of the oneOf group of the resource that no Create or
// Update body has. With one arm left, the group goes away: the arm is a plain field of the response.
// It needs api.operations, which name the requests. An arm that a request has keeps the rule of the
// shared module.
func pruneSkippedArms(doc, schemas *yaml.Node, resource string, file *overrides.File) error {
	ops := file.API.Operations
	s := mapValue(schemas, resource)
	group := mapValue(s, "oneOf")
	if ops == nil || group == nil {
		return nil
	}
	var bodies []*yaml.Node
	for _, id := range []string{ops.Create, ops.Update} {
		_, _, op := findOperation(doc, id)
		if op == nil {
			// Without the request, an arm cannot be known to be response-only.
			return &shapeError{code: "OPERATION_NOT_FOUND", path: "behavior-overrides.yaml:api.operations", msg: "the contract has no operation " + id}
		}
		bodies = append(bodies, bodySchema(schemas, op))
	}
	inRequest := func(arm string) bool {
		return slices.ContainsFunc(bodies, func(b *yaml.Node) bool { return mapValue(mapValue(b, "properties"), arm) != nil })
	}
	var kept []*yaml.Node
	pruned := false
	for _, entry := range group.Content {
		arms := stringList(mapValue(entry, "required"))
		if len(arms) == 1 && file.Field(resource, arms[0]).Skip && !inRequest(arms[0]) {
			pruned = true
			file.PrunedArms = append(file.PrunedArms, arms[0])
			continue
		}
		kept = append(kept, entry)
	}
	switch {
	case !pruned:
	case len(kept) <= 1:
		deleteMap(s, "oneOf")
	default:
		group.Content = kept
	}
	return nil
}

// upsertOperations checks the two calls of an upsert, a GET and a POST on one path without path
// parameters, and adds the PUT and the DELETE that the shared module needs.
func upsertOperations(doc *yaml.Node, getID, setID string) error {
	getPath, getMethod, get := findOperation(doc, getID)
	setPath, setMethod, set := findOperation(doc, setID)
	switch {
	case get == nil:
		return upsertIssue(".get", "the contract has no operation %s", getID)
	case getMethod != "get" || strings.Contains(getPath, "{"):
		return upsertIssue(".get", "%s is %s %s; a singleton that one call sets has a GET with no path parameters", getID, strings.ToUpper(getMethod), getPath)
	case set == nil:
		return upsertIssue(".create", "the contract has no operation %s", setID)
	case setMethod != "post" || setPath != getPath:
		return upsertIssue(".create", "%s is %s %s; the call that sets a singleton is a POST on the Get path %s, with no path parameters", setID, strings.ToUpper(setMethod), setPath, getPath)
	}
	item := mappingPath(doc, "paths", getPath)
	if mapValue(item, "delete") != nil {
		return upsertIssue("", "%s also has a DELETE operation, so the API can remove the singleton", getPath)
	}
	// The POST creates and replaces the singleton. Another write on its path would be a second way to
	// change it, which the upsert would hide.
	for _, method := range []string{"put", "patch"} {
		if mapValue(item, method) != nil {
			return upsertIssue("", "%s also has a %s operation; name the operations of an update, not an upsert", getPath, strings.ToUpper(method))
		}
	}
	put := cloneMapping(set)
	setMap(put, "operationId", scalar(setID+overrides.UpsertPutSuffix))
	setMap(item, "put", put)
	del := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMap(del, "operationId", scalar(getID+overrides.UpsertDeleteSuffix))
	for _, key := range []string{"responses", "tags"} {
		if v := mapValue(get, key); v != nil {
			setMap(del, key, v)
		}
	}
	setMap(item, "delete", del)
	return nil
}

func upsertIssue(suffix, format string, args ...any) error {
	return &shapeError{code: "UPSERT_SHAPE_UNSUPPORTED", path: "behavior-overrides.yaml:api.operations" + suffix, msg: fmt.Sprintf(format, args...)}
}

func inlineIssue(resource, format string, args ...any) error {
	return &shapeError{code: "INLINE_UNSUPPORTED", path: "components.schemas." + resource, msg: fmt.Sprintf(format, args...)}
}

// inlineResponse replaces each inlined object of the resource component by its fields, in place and
// in order. The object must be required, so that its fields are always in the response, by the
// contract or by required: true on the line, and no Create or Update body may hold it: the requests
// send its fields at the top level.
func inlineResponse(doc, schemas *yaml.Node, resource string, inlined []string, file *overrides.File) (map[string]overrides.InlineParent, error) {
	s := mapValue(schemas, resource)
	props := mapValue(s, "properties")
	if props == nil {
		return nil, inlineIssue(resource, "the contract has no object component %s", resource)
	}
	if err := checkRequestsFlat(doc, schemas, resource, inlined, file); err != nil {
		return nil, err
	}
	required := stringList(mapValue(s, "required"))
	var arms []string
	for _, entry := range nodeContent(mapValue(s, "oneOf")) {
		arms = append(arms, stringList(mapValue(entry, "required"))...)
	}
	parents := map[string]overrides.InlineParent{}
	flat := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	var flatRequired []string
	add := func(key, value *yaml.Node, isRequired bool) error {
		if mapValue(flat, key.Value) != nil {
			return inlineIssue(resource, "two fields have the name %q", key.Value)
		}
		flat.Content = append(flat.Content, key, value)
		if isRequired {
			flatRequired = append(flatRequired, key.Value)
		}
		return nil
	}
	for _, name := range inlined {
		if mapValue(props, name) == nil {
			return nil, inlineIssue(resource, "%s has no field %s", resource, name)
		}
	}
	for i := 0; i+1 < len(props.Content); i += 2 {
		key, value := props.Content[i], props.Content[i+1]
		name := key.Value
		if !slices.Contains(inlined, name) {
			if err := add(key, value, slices.Contains(required, name)); err != nil {
				return nil, err
			}
			continue
		}
		// A $ref with a description of its own is a one-item allOf.
		component := refName(value)
		if all := nodeContent(mapValue(value, "allOf")); component == "" && len(all) == 1 && len(nodeContent(mapValue(value, "properties"))) == 0 {
			component = refName(all[0])
		}
		object := mapValue(schemas, component)
		objectProps := mapValue(object, "properties")
		switch {
		case component == "" || object == nil:
			return nil, inlineIssue(resource, "inline: %s must be a $ref to an object component", name)
		case objectProps == nil || len(objectProps.Content) == 0:
			return nil, inlineIssue(resource, "inline: %s must be an object with fields", name)
		case mapValue(object, "oneOf") != nil || mapValue(object, "allOf") != nil || mapValue(object, "anyOf") != nil:
			return nil, inlineIssue(resource, "inline: %s must be an object without oneOf, allOf, or anyOf", name)
		case slices.Contains(arms, name):
			return nil, inlineIssue(resource, "inline: %s is an arm of a oneOf group, so the response does not always have it", name)
		case !slices.Contains(required, name) && !file.Types[resource].Fields[name].Required:
			return nil, inlineIssue(resource, "inline: %s must be required, so that its fields are always in the response; when the API always returns it, state that with required: true on the line", name)
		}
		objectRequired := stringList(mapValue(object, "required"))
		for j := 0; j+1 < len(objectProps.Content); j += 2 {
			child := objectProps.Content[j].Value
			if err := add(objectProps.Content[j], objectProps.Content[j+1], slices.Contains(objectRequired, child)); err != nil {
				return nil, err
			}
			parents[child] = overrides.InlineParent{Field: name, Schema: component}
		}
	}
	setMap(s, "properties", flat)
	if len(flatRequired) == 0 {
		deleteMap(s, "required")
	} else {
		setMap(s, "required", stringNode(flatRequired))
	}
	return parents, nil
}

// checkRequestsFlat reports an inlined object that a Create or Update body holds. Root inline is for
// an object that only the response nests: the requests send its fields at the top level, so the
// generated conversions need no change. An object that a request nests too needs the requests to
// rebuild it, which is not supported yet.
func checkRequestsFlat(doc, schemas *yaml.Node, resource string, inlined []string, file *overrides.File) error {
	ops := file.API.Operations
	if ops == nil || ops.Create == "" || ops.Update == "" {
		return inlineIssue(resource, "inline on the resource needs api.operations, which name the requests")
	}
	for _, id := range []string{ops.Create, ops.Update} {
		_, _, op := findOperation(doc, id)
		body := bodySchema(schemas, op)
		for _, name := range inlined {
			if mapValue(mapValue(body, "properties"), name) != nil {
				return inlineIssue(resource, "inline: the request of %s holds %s too; inline on the resource supports only an object that the response alone nests", id, name)
			}
		}
	}
	return nil
}

// bodySchema returns the schema of the request body of op: the component that it refers to, or the
// inline schema.
func bodySchema(schemas, op *yaml.Node) *yaml.Node {
	schema := mapValue(mapValue(mapValue(mapValue(op, "requestBody"), "content"), "application/json"), "schema")
	if ref := refName(schema); ref != "" {
		return mapValue(schemas, ref)
	}
	return schema
}

// findOperation returns the path, the lower-case method, and the node of the operation with the
// operationId id, or a nil node.
func findOperation(doc *yaml.Node, id string) (string, string, *yaml.Node) {
	paths := mapValue(doc, "paths")
	if paths == nil {
		return "", "", nil
	}
	for i := 0; i+1 < len(paths.Content); i += 2 {
		item := paths.Content[i+1]
		for j := 0; j+1 < len(item.Content); j += 2 {
			op := item.Content[j+1]
			if v := mapValue(op, "operationId"); v != nil && v.Value == id {
				return paths.Content[i].Value, item.Content[j].Value, op
			}
		}
	}
	return "", "", nil
}

func setMap(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content, scalar(key), value)
}

func deleteMap(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

func cloneMapping(m *yaml.Node) *yaml.Node {
	out := *m
	out.Content = slices.Clone(m.Content)
	return &out
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// refName returns the component that a {$ref: '#/components/schemas/X'} node names, or "".
func refName(n *yaml.Node) string {
	ref := mapValue(n, "$ref")
	if ref == nil {
		return ""
	}
	name, _ := strings.CutPrefix(ref.Value, "#/components/schemas/")
	if name == ref.Value {
		return ""
	}
	return name
}

func nodeContent(n *yaml.Node) []*yaml.Node {
	if n == nil {
		return nil
	}
	return n.Content
}

func stringList(n *yaml.Node) []string {
	var out []string
	for _, item := range nodeContent(n) {
		out = append(out, item.Value)
	}
	return out
}

func stringNode(values []string) *yaml.Node {
	out := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, v := range values {
		out.Content = append(out.Content, scalar(v))
	}
	return out
}
