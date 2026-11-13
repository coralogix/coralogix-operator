package generator

import (
	"crypto/sha256"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// spec builds the spec from the top-level fields. A field that only the server sets is not in it.
func (b *builder) spec(r *model.Resource) (*crType, error) {
	spec := &crType{Name: b.kind + "Spec", Doc: []string{fmt.Sprintf("%sSpec defines the desired state of a %s.", b.kind, b.kind)}}
	inGroup := map[string]bool{}
	for _, g := range r.Groups {
		for _, arm := range g.Arms {
			inGroup[arm] = true
		}
	}
	for _, f := range r.Fields {
		if f.Behavior != model.Computed && f.Create == nil && f.Update != nil {
			return nil, &shapeError{code: codeUpdateOnlyField, path: "types." + r.Name + ".fields." + f.Name,
				msg: "the Update request has the field " + f.Name + ", but the Create request does not; the renderer cannot send a field in only one of them"}
		}
		if f.Behavior == model.Computed || f.Create == nil {
			continue
		}
		if len(b.file.Field(r.Name, f.Name).RequestValueBy) != 0 {
			continue
		}
		if inGroup[f.Name] && b.file.Field(r.Name, f.Name).Required {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":types." + r.Name + ".fields." + f.Name + ".required", msg: "a oneOf arm cannot be required on its own"}
		}
		if folds := r.CreateFolds(f); len(folds) != 0 {
			fields, err := b.foldedFields(r, f, folds)
			if err != nil {
				return nil, err
			}
			spec.Fields = append(spec.Fields, fields...)
			continue
		}
		required := f.Create.Required && !inGroup[f.Name]
		field, err := b.field(b.kind, r.Name, f.Name, f.Description, f.Type, required, requiredScalarWithoutPresence(f.Create, f.Type), "spec."+f.Name, false)
		if err != nil {
			return nil, err
		}
		if err := b.contractDefault(field, f.Create, field.validationPath); err != nil {
			return nil, err
		}
		if f.Behavior == model.Immutable {
			immutable(field, field.validationPath)
		}
		spec.Fields = append(spec.Fields, field)
		if field.refForm == "split" {
			spec.Fields = append(spec.Fields, splitReferenceField(field))
		}
	}
	b.specType = spec
	groups, err := b.overrideGroups(r.Name, r.Groups, spec.Fields)
	if err != nil {
		return nil, err
	}
	groups, err = b.groupFields(r.Name, spec, groups)
	if err != nil {
		return nil, err
	}
	presenceRules(spec, spec, nil, false)
	if err := groupRules(spec, groups, "spec"); err != nil {
		return nil, err
	}
	splitReferenceRules(spec)
	if err := b.applyValidation(spec, "spec", map[*crType]bool{}, true); err != nil {
		return nil, err
	}
	for _, scalar := range b.scalars {
		if err := b.applyValidationMarkers(&scalar.Markers, scalar.path+"{}"); err != nil {
			return nil, err
		}
	}
	return spec, nil
}

// foldedFields builds the CR fields of a createAs line: one field for each property of the object,
// under the name of the flat Create field. A CR that follows the released API holds the flat value.
func (b *builder) foldedFields(r *model.Resource, f *model.ResourceField, folds []model.CreateFold) ([]*crField, error) {
	if !r.Replace {
		return nil, unsupported("types."+r.Name+".fields."+f.Name+".createAs", "createAs needs a full-replace Update")
	}
	if len(folds) > 1 {
		// With two flat fields, a CR can set one and leave the other out. Update would then send the
		// object with the missing property at its zero value, and overwrite the stored value.
		return nil, unsupported("types."+r.Name+".fields."+f.Name+".createAs", "createAs supports one property per object; %d are named", len(folds))
	}
	if b.folds == nil {
		b.folds = map[string]*foldedField{}
	}
	b.folds[f.Name] = &foldedField{field: f, folds: folds}
	var out []*crField
	for _, fold := range folds {
		switch fold.Type.Kind {
		case model.String, model.Bool, model.Number, model.Integer, model.Enum:
		case model.List, model.Set:
			if fold.Type.Elem == nil || !slices.Contains([]model.Kind{model.String, model.Bool, model.Number, model.Integer, model.Enum}, fold.Type.Elem.Kind) {
				return nil, unsupported("spec."+fold.Field, "createAs supports a scalar, an enum, or a list of them")
			}
		case model.Object, model.OneOf, model.Map:
			return nil, unsupported("spec."+fold.Field, "createAs supports a scalar, an enum, or a list of them, got %s", fold.Type.Kind)
		}
		field, err := b.field(b.kind, r.Name, fold.Field, f.Description, fold.Type, fold.Required, false, "spec."+fold.Field, false)
		if err != nil {
			return nil, err
		}
		field.fold = &foldOrigin{Object: f.Name, Property: fold.Property}
		if f.Behavior == model.Immutable {
			immutable(field, field.validationPath)
		}
		out = append(out, field)
	}
	return out, nil
}

// applyValidation changes released markers after the contract and oneOf rules are built.
func (b *builder) applyValidation(t *crType, path string, seen map[*crType]bool, root bool) error {
	if seen[t] {
		return nil
	}
	seen[t] = true
	markerPath := path + "#type"
	if root {
		markerPath = path
	}
	if err := b.applyValidationMarkers(&t.Markers, markerPath); err != nil {
		return err
	}
	for _, field := range t.Fields {
		child := path + "." + field.JSON
		if err := b.applyValidationMarkers(&field.Markers, child); err != nil {
			return err
		}
		if field.object != nil {
			if strings.HasPrefix(field.GoType, "[]") {
				child += "[]"
			} else if strings.HasPrefix(field.GoType, "map[") {
				child += "{}"
			}
			if err := b.applyValidation(field.object, child, seen, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *builder) applyValidationMarkers(markers *[]string, path string) error {
	change, ok := b.file.ValidationAt(path)
	if !ok {
		return nil
	}
	for _, marker := range change.Remove {
		index := slices.Index(*markers, marker)
		if index < 0 {
			return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":validators.fields." + path, msg: fmt.Sprintf("marker %q is not generated", marker)}
		}
		*markers = slices.Delete(*markers, index, index+1)
		b.usedValidators[path] = true
	}
	for _, marker := range change.Add {
		if slices.Contains(*markers, marker) {
			return &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":validators.fields." + path, msg: fmt.Sprintf("marker %q is already generated", marker)}
		}
		*markers = append(*markers, marker)
		b.usedValidators[path] = true
	}
	return nil
}

// contractDefault writes a Kubernetes default for a field that follows the API contract.
func (b *builder) contractDefault(field *crField, attrs *model.Attrs, path string) error {
	if attrs == nil || attrs.Default == nil {
		return nil
	}
	if !b.file.InferredAt(path) {
		if b.file.Inferred() {
			b.usedValidators[path] = true
		} else {
			b.droppedLimits = true
		}
		return nil
	}
	value := *attrs.Default
	switch field.typ.Kind {
	case model.String:
		value = markerString(value)
	case model.Enum:
		matched := false
		for _, enum := range field.enum.Values {
			if enum.API == value {
				value, matched = markerString(enum.Name), true
				break
			}
		}
		if !matched {
			return unsupported(path, "the API default %q is not an accepted CR enum value", value)
		}
	case model.Bool, model.Integer:
		// The API default is already a scalar literal.
	case model.Number:
		// A Quantity is int-or-string in the CRD, and Kubernetes rejects a number literal as its
		// default. The default is written as a string in plain decimal form.
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return unsupported(path, "the API number default %q is not a finite number", value)
		}
		value = markerString(strconv.FormatFloat(number, 'f', -1, 64))
	case model.List, model.Set:
		if strings.TrimSpace(value) != "[]" {
			return unsupported(path, "the API collection default %q is not an empty list", value)
		}
		value = "{}" // controller-gen reads {} as an empty list for a slice field
	case model.Map, model.Object, model.OneOf:
		if strings.TrimSpace(value) != "{}" {
			return unsupported(path, "the API object default %q is not empty", value)
		}
		value = "{}"
	default:
		return unsupported(path, "the API default on a %s needs a supported CR marker", field.typ.Kind)
	}
	if !b.file.Inferred() {
		b.usedValidators[path] = true
	}
	field.Markers = append(field.Markers, "+kubebuilder:default="+value)
	return nil
}

// immutable adds the rule that rejects a change of the field. A rule on the field does not run
// when the field or an object that holds it is missing, so presenceRules adds the rest.
func immutable(f *crField, path string) {
	f.immutable = true
	f.Markers = append(f.Markers, celMarker("self == oldSelf", path+" is immutable"))
}

// presenceRules adds to the spec a rule for each immutable field that can be added or removed: an
// optional field, or a field inside an optional object. The spec always exists, so the rule runs
// on every update. It compares the presence of the whole path, for example
// (has(self.settings) && has(self.settings.region)) == (has(oldSelf.settings) && has(oldSelf.settings.region)).
// A rule on the object that holds the field would not run when the object itself is added or removed.
func presenceRules(spec, t *crType, prefix []string, optionalParent bool) {
	for _, f := range t.Fields {
		path := append(slices.Clone(prefix), f.JSON)
		if f.immutable && (f.Optional || optionalParent) {
			var self, old []string
			for i := range path {
				segment := strings.Join(path[:i+1], ".")
				self = append(self, "has(self."+segment+")")
				old = append(old, "has(oldSelf."+segment+")")
			}
			rule := fmt.Sprintf("(%s) == (%s)", strings.Join(self, " && "), strings.Join(old, " && "))
			if len(path) == 1 {
				rule = fmt.Sprintf("%s == %s", self[0], old[0])
			}
			spec.Markers = append(spec.Markers, celMarker(rule, "spec."+strings.Join(path, ".")+" is immutable"))
			continue
		}
		// A list item or a map value has no immutable field: the renderer rejects one.
		if f.object != nil && !strings.HasPrefix(f.GoType, "[]") && !strings.HasPrefix(f.GoType, "map[") {
			presenceRules(spec, f.object, path, optionalParent || f.Optional)
		}
	}
}

// immutableField returns the path of the first immutable field in t or in an object that it holds,
// or "" when it has none.
func immutableField(t *crType, path string, seen map[*crType]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	for _, f := range t.Fields {
		child := path + "." + f.API
		if f.immutable {
			return child
		}
		if f.object != nil {
			if found := immutableField(f.object, child, seen); found != "" {
				return found
			}
		}
	}
	return ""
}

func celMarker(rule, message string) string {
	return fmt.Sprintf("+kubebuilder:validation:XValidation:rule=%s,message=%s", strconv.Quote(rule), strconv.Quote(message))
}

// groupRules adds the CEL rule of each oneOf group: exactly one arm, or at most one.
func groupRules(t *crType, groups []model.OneOfGroup, path string) error {
	t.groups = groups
	for _, g := range groups {
		var terms, names []string
		for _, arm := range g.Arms {
			i := slices.IndexFunc(t.Fields, func(f *crField) bool { return f.API == arm })
			if i < 0 {
				return unsupported(path+"."+arm, "a oneOf arm that the custom resource cannot set is not supported")
			}
			terms = append(terms, fmt.Sprintf("(has(self.%s) ? 1 : 0)", t.Fields[i].JSON))
			names = append(names, t.Fields[i].JSON)
		}
		rule, msg := " == 1", "Exactly one of %s must be set"
		if g.AllowNone {
			rule, msg = " <= 1", "At most one of %s can be set"
		}
		t.Markers = append(t.Markers, celMarker(strings.Join(terms, " + ")+rule, fmt.Sprintf(msg, strings.Join(names, ", "))))
	}
	return nil
}

// overrideGroups keeps a released exactly-one rule when the contract permits zero arms.
func (b *builder) overrideGroups(component string, groups []model.OneOfGroup, fields []*crField) ([]model.OneOfGroup, error) {
	out := slices.Clone(groups)
	for i, g := range groups {
		out[i] = g
		out[i].Arms = slices.Clone(g.Arms)
		for _, arm := range g.Arms {
			line := overrides.Line{Component: component, Field: arm}
			if !b.file.Field(component, arm).RequireOne {
				continue
			}
			if !g.AllowNone || !out[i].AllowNone {
				return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".requireOne", msg: "the group already requires one arm"}
			}
			out[i].AllowNone = false
			b.used[line] = true
		}
	}
	for _, f := range fields {
		line := overrides.Line{Component: component, Field: f.API}
		if !b.file.Field(component, f.API).RequireOne || b.used[line] {
			continue
		}
		return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".requireOne", msg: "requireOne needs a oneOf field or an arm of a oneOf group"}
	}
	return out, nil
}

// groupFields places released CR fields under named objects and moves their oneOf rules.
func (b *builder) groupFields(component string, owner *crType, groups []model.OneOfGroup) ([]model.OneOfGroup, error) {
	configured := b.file.GroupsFor(component)
	if len(configured) == 0 {
		return groups, nil
	}
	selected := map[string]string{}
	for name, group := range configured {
		if slices.ContainsFunc(owner.Fields, func(f *crField) bool { return f.JSON == name }) {
			return nil, collision("groups."+name, "the group name is already a CR field")
		}
		for _, field := range group.Fields {
			selected[field] = name
		}
	}
	remainingGroups := make([]model.OneOfGroup, 0, len(groups))
	groupRulesByName := map[string][]model.OneOfGroup{}
	for _, rule := range groups {
		name := ""
		for _, arm := range rule.Arms {
			if selected[arm] != "" {
				if name != "" && name != selected[arm] {
					return nil, unsupported("groups", "oneOf arms must stay in one group")
				}
				name = selected[arm]
			}
		}
		if name == "" {
			remainingGroups = append(remainingGroups, rule)
			continue
		}
		if slices.ContainsFunc(rule.Arms, func(arm string) bool { return selected[arm] != name }) {
			return nil, unsupported("groups."+name, "all oneOf arms must be in the same group")
		}
		groupRulesByName[name] = append(groupRulesByName[name], rule)
	}
	remainingFields := slices.DeleteFunc(slices.Clone(owner.Fields), func(f *crField) bool { return selected[f.API] != "" })
	for _, name := range sortedKeys(configured) {
		group := configured[name]
		typeName := owner.Name + goName(name)
		if b.types[typeName] != nil {
			return nil, collision("groups."+name, "the group type %s already exists", typeName)
		}
		child := &crType{Name: typeName, Doc: []string{fmt.Sprintf("%s holds the %s fields of the released CR.", typeName, name)}}
		for _, api := range group.Fields {
			i := slices.IndexFunc(owner.Fields, func(f *crField) bool { return f.API == api })
			if i < 0 {
				return nil, unsupported("groups."+name, "the API component has no managed field %s", api)
			}
			if owner.Fields[i].immutable {
				return nil, unsupported("groups."+name, "an immutable field cannot move into a required group")
			}
			child.Fields = append(child.Fields, owner.Fields[i])
		}
		rules := groupRulesByName[name]
		if group.RequireOne {
			if len(rules) != 1 || !rules[0].AllowNone {
				return nil, unsupported("groups."+name+".requireOne", "requireOne needs one complete oneOf group that permits zero arms")
			}
			rules[0].AllowNone = false
		}
		if err := groupRules(child, rules, "groups."+name); err != nil {
			return nil, err
		}
		b.types[typeName] = child
		b.order = append(b.order, child)
		remainingFields = append(remainingFields, &crField{API: name, JSON: name, GoName: goName(name), GoType: typeName, typ: &model.Type{Kind: model.Object}, object: child, group: true})
		b.usedGroups[component+"."+name] = true
	}
	owner.Fields = remainingFields
	return remainingGroups, nil
}

// requiredScalarWithoutPresence reports a required zero-valued scalar whose released behavior
// needs an explicit required line because the API contract cannot distinguish zero from omission.
func requiredScalarWithoutPresence(create *model.Attrs, t *model.Type) bool {
	if create == nil || !create.Required || create.Presence || !t.ZeroValid {
		return false
	}
	switch t.Kind {
	case model.Bool, model.Number, model.Integer:
		return true
	case model.String, model.Enum, model.Object, model.OneOf, model.List, model.Set, model.Map:
		return false
	}
	return false
}

// validationPath maps an API field to its final CR path before markers are selected.
func (b *builder) validationPath(component, path, apiName, crName string) string {
	parent := strings.TrimSuffix(path, "."+apiName)
	for groupName, group := range b.file.GroupsFor(component) {
		if slices.Contains(group.Fields, apiName) {
			return parent + "." + groupName + "." + crName
		}
	}
	return parent + "." + crName
}

// field builds one field of the Go type owner, which holds component. inItem is true inside a
// list item or a map value, where kubectl cannot compare a value with its old value.
func (b *builder) field(owner, component, name, description string, t *model.Type, required, requiredPresenceMissing bool, path string, inItem bool) (*crField, error) {
	if err := wrapped(t, path); err != nil {
		return nil, err
	}
	line := overrides.Line{Component: component, Field: name}
	change := b.file.Field(component, name)
	if change.RequireOne && t.Kind == model.OneOf {
		if !t.AllowNone {
			return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".requireOne", msg: "the oneOf already requires one arm"}
		}
		requiredType := *t
		requiredType.AllowNone = false
		t = &requiredType
		b.used[line] = true
	}
	if change.Required && (!required || requiredPresenceMissing) {
		required = true
		b.used[line] = true
	}
	if change.Optional {
		// The policy relaxes a required request field before the model is built.
		b.used[line] = true
	}
	jsonName := name
	if change.Name != "" && change.Name != name {
		jsonName = change.Name
		b.used[line] = true
	}
	f := &crField{API: name, JSON: jsonName, GoName: goName(jsonName), Optional: !required, Doc: docLines(description), typ: t}
	path = b.validationPath(component, path, name, jsonName)
	f.validationPath = path
	if change.OmitEmptyInRequest {
		if !required || (t.Kind != model.List && t.Kind != model.Set && t.Kind != model.Map) {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".omitEmptyInRequest", msg: "omitEmptyInRequest needs a required collection"}
		}
		f.omitEmptyInRequest = true
		b.used[line] = true
	}
	if change.SendNullAsEmpty {
		isString := t.Kind == model.String && t.Format != "date-time" && len(t.Wrappers) == 0
		if required || b.file.Ref(component, name) != nil || (!isString && t.Kind != model.List && t.Kind != model.Set) {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".sendNullAsEmpty", msg: "sendNullAsEmpty needs an optional string or an optional list"}
		}
		f.sendNullAsEmpty = true
		b.used[line] = true
	}
	if !required {
		f.Markers = append(f.Markers, "+optional")
	}
	if ref := b.file.Ref(component, name); ref != nil {
		return b.refField(f, component, ref, path)
	}
	switch t.Kind {
	case model.List, model.Set, model.Map:
		if change.Collection != "" {
			if t.Kind != model.Set {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".collection", msg: "collection: list needs an API set"}
			}
			b.used[line] = true
		}
		elem, err := b.elemType(t.Elem, path, f)
		if err != nil {
			return nil, err
		}
		if t.Kind == model.Map {
			if markers := b.scalarMarkers(t.Elem, f.enum, path+"{}"); f.object == nil && len(markers) != 0 {
				elem = b.scalar(owner+f.GoName+"Value", elem, markers, path)
			}
			f.GoType = "map[string]" + elem
			f.Markers = append(f.Markers, b.limits(boundMarkers("MinProperties", "MaxProperties", t.MinItems, t.MaxItems), path)...)
			return f, nil
		}
		f.GoType = "[]" + elem
		if change.ListMapKey != "" {
			if f.object == nil {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".listMapKey", msg: "a list-map needs object items"}
			}
			key := slices.IndexFunc(f.object.Fields, func(item *crField) bool { return item.JSON == change.ListMapKey })
			if key < 0 || f.object.Fields[key].Optional || f.object.Fields[key].GoType != "string" {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".listMapKey", msg: "the key must be a required string item field"}
			}
			f.Markers = append(f.Markers, "+listType=map", "+listMapKey="+change.ListMapKey)
		} else if t.Kind == model.Set && change.Collection == "" {
			if f.object != nil {
				return nil, unsupported(path, "a set of objects is not supported yet")
			}
			f.Markers = append(f.Markers, "+listType=set")
		}
		f.Markers = append(f.Markers, b.limits(boundMarkers("MinItems", "MaxItems", t.MinItems, t.MaxItems), path)...)
		f.Markers = append(f.Markers, prefixed("items:", b.scalarMarkers(t.Elem, f.enum, path+"[]"))...)
		return f, nil
	case model.Object, model.OneOf:
		if change.EmptyObject != "" {
			if t.Kind != model.Object || len(t.Fields) != 0 {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".emptyObject", msg: "emptyObject: map needs an object with no fields"}
			}
			f.object = &crType{}
			f.GoType = "map[string]string"
			f.Markers = append(f.Markers, "+kubebuilder:validation:MaxProperties=0")
			b.used[line] = true
			break
		}
		obj, err := b.object(t, path, inItem)
		if err != nil {
			return nil, err
		}
		f.object, f.GoType = obj, obj.Name
	case model.String, model.Bool, model.Number, model.Integer, model.Enum:
		goType, err := scalarType(t, path)
		if err != nil {
			return nil, err
		}
		if t.Kind == model.Enum {
			if f.enum, err = b.enum(t, path); err != nil {
				return nil, err
			}
		}
		f.GoType = goType
		f.Markers = append(f.Markers, b.scalarMarkers(t, f.enum, path)...)
	}
	if !required {
		f.GoType = "*" + f.GoType
	}
	return f, nil
}

// wrapped rejects a keyed value. Wrapper objects are restored by request conversion.
func wrapped(t *model.Type, path string) error {
	if t.Keyed != nil {
		return unsupported(path, "keyedBy values are not supported yet")
	}
	return nil
}

// refField makes a string id of another resource a reference (way 2). Without a name override,
// it replaces a trailing Id with Ref: thingId becomes thingRef.
func (b *builder) refField(f *crField, component string, ref *overrides.Ref, path string) (*crField, error) {
	line := overrides.Line{Component: component, Field: f.API}
	value := f.typ.Unwrapped()
	collection := value.Kind == model.List || value.Kind == model.Set
	if collection {
		value = value.Elem
	}
	if value == nil || value.Kind != model.String && value.Kind != model.Integer || value.Kind == model.String && value.Format != "" || value.Kind == model.Integer && value.Format != "int32" && value.Format != "int64" {
		return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref",
			msg: fmt.Sprintf("ref needs a string or int32/int64 id, or a list of ids, the field is a %s", f.typ.Kind)}
	}
	if value.Kind == model.String && ref.BackendIDType != "" {
		return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref.backendIdType", msg: "a string id has no numeric backend id type"}
	}
	b.used[line] = true
	base, _ := strings.CutSuffix(strings.TrimSuffix(f.API, "Id"), "ID")
	defaultName := base + "Ref"
	if name := b.file.Field(component, f.API).Name; name != "" && name == defaultName {
		return nil, &shapeError{code: codeOverrideUnused, path: overrides.FileName + ":" + line.String() + ".name",
			msg: "the name is already the default reference name"}
	}
	if b.file.Field(component, f.API).Name == "" {
		f.JSON = defaultName
	}
	f.GoName, f.GoType, f.ref = goName(f.JSON), "ResourceIDRef", ref.Kind
	path = path[:strings.LastIndex(path, ".")+1] + f.JSON
	f.validationPath = path
	f.refForm, f.refVersion = ref.Form, ref.Version
	f.refIDField = ref.IDField
	if ref.Form == "named" || ref.Form == "namedBackendOnly" {
		if ref.BackendIDType != "" {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref.backendIdType", msg: "named uses the API id type"}
		}
		if ref.Form == "namedBackendOnly" && value.Kind != model.String {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref.form", msg: "namedBackendOnly needs a string id"}
		}
		f.refLookup = ref.Lookup
		switch value.Kind {
		case model.String:
			if ref.Form == "namedBackendOnly" {
				f.GoType = "NamedBackendOnlyRef"
			} else {
				f.GoType = "NamedResourceIDRef"
			}
		case model.Integer:
			if value.Format == "int32" {
				f.GoType = "NamedInt32ResourceIDRef"
			} else {
				f.GoType = "NamedInt64ResourceIDRef"
			}
		case model.Bool, model.Number, model.Enum, model.Object, model.OneOf, model.List, model.Set, model.Map:
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref", msg: "a named reference needs a string or integer id"}
		}
		b.hasNameLookup = true
	}
	if ref.Form == "split" {
		if collection || !f.Optional || value.Kind != model.String {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".ref.form", msg: "split needs one optional string id"}
		}
		f.refSibling = ref.ResourceField
		f.GoType = "*string"
		f.Doc = append(f.Doc, fmt.Sprintf("It holds the backend id of a %s.", ref.Kind))
		return f, nil
	}
	if value.Kind == model.Integer {
		f.refIDType = value.Format
		if ref.Form != "named" && ref.Form != "namedBackendOnly" {
			backendType := value.Format
			if ref.BackendIDType != "" {
				backendType = ref.BackendIDType
			}
			switch backendType {
			case "int32":
				f.GoType = "Int32ResourceIDRef"
			case "int64":
				f.GoType = "Int64ResourceIDRef"
			case "uint32":
				f.GoType = "UInt32ResourceIDRef"
			}
		}
	}
	if ref.Form == "resourceRefOnly" {
		f.GoType = "ResourceOnlyRef"
	}
	if ref.Type != "" {
		f.GoType, f.refType = ref.Type, ref.Type
	}
	if collection {
		f.GoType = "[]" + f.GoType
		f.Markers = append(f.Markers, b.limits(boundMarkers("MinItems", "MaxItems", f.typ.MinItems, f.typ.MaxItems), path)...)
		if f.typ.Kind == model.Set && b.file.Field(component, f.API).Collection != "list" {
			f.Markers = append(f.Markers, "+listType=set")
		}
	} else if f.Optional {
		f.GoType = "*" + f.GoType
	}
	switch ref.Form {
	case "resourceRefOnly":
		f.Doc = append(f.Doc, fmt.Sprintf("It refers to a %s custom resource through resourceRef.", ref.Kind))
	case "named", "namedBackendOnly":
		f.Doc = append(f.Doc, fmt.Sprintf("It refers to a %s by backend id, backend name, or custom resource.", ref.Kind))
	default:
		f.Doc = append(f.Doc, fmt.Sprintf("It refers to a %s: by backendRef.id, or by resourceRef, a %s custom resource.", ref.Kind, ref.Kind))
	}
	return f, nil
}

// elemType is the Go type of the items of a list or the values of a map.
func (b *builder) elemType(t *model.Type, path string, f *crField) (string, error) {
	if err := wrapped(t, path+"[]"); err != nil {
		return "", err
	}
	switch t.Kind {
	case model.Object, model.OneOf:
		suffix := "[]"
		if f.typ.Kind == model.Map {
			suffix = "{}"
		}
		obj, err := b.object(t, path+suffix, true)
		if err != nil {
			return "", err
		}
		f.object = obj
		return obj.Name, nil
	case model.List, model.Set, model.Map:
		return "", unsupported(path, "a collection of collections is not supported")
	case model.String, model.Bool, model.Number, model.Integer, model.Enum:
		// A scalar is handled below.
	}
	if t.Kind == model.Number && f.typ.Kind == model.Map {
		// A defined value type over resource.Quantity would lose its JSON and DeepCopy methods.
		return "", unsupported(path, "a map of numbers is not supported yet")
	}
	goType, err := scalarType(t, path)
	if err != nil {
		return "", err
	}
	if t.Kind == model.Enum {
		if f.enum, err = b.enum(t, path); err != nil {
			return "", err
		}
	}
	return goType, nil
}

// scalarType is the Go type of a scalar in the custom resource.
func scalarType(t *model.Type, path string) (string, error) {
	switch t.Kind {
	case model.String:
		if t.Format == "date-time" {
			return "metav1.Time", nil
		}
		return "string", nil
	case model.Enum:
		return "string", nil
	case model.Bool:
		return "bool", nil
	case model.Integer:
		switch t.Format {
		case "int32":
			return "int32", nil
		case "int64":
			return "int64", nil
		}
		return "", unsupported(path, "an integer with format %q is not supported", t.Format)
	case model.Number:
		// A custom resource holds a number as a quantity. The SDK type is float32 or float64.
		if t.Format != "float" && t.Format != "double" {
			return "", unsupported(path, "a number needs format float or double, got %q", t.Format)
		}
		return "resource.Quantity", nil
	case model.Object, model.OneOf, model.List, model.Set, model.Map:
		// Not a scalar: the error below.
	}
	return "", unsupported(path, "a %s is not supported", t.Kind)
}

// scalarMarkers are the markers of a scalar: the values of an enum, or the contract limits that
// limits keeps. The limits are API limits, so kubectl rejects only a value that the API rejects
// too.
func (b *builder) scalarMarkers(t *model.Type, enum *enumMap, path string) []string {
	if enum != nil {
		return []string{"+kubebuilder:validation:Enum=" + enum.Names()}
	}
	return b.limits(limitMarkers(t), path)
}

// limits returns the markers of contract limits, or none when validators.inferred is false. It
// records a dropped limit, so a validators line that drops nothing is reported as unused.
func (b *builder) limits(markers []string, path string) []string {
	if b.file.InferredAt(path) {
		if !b.file.Inferred() && len(markers) != 0 {
			b.usedValidators[path] = true
		}
		return markers
	}
	if len(markers) != 0 {
		if b.file.Inferred() {
			b.usedValidators[path] = true
		} else {
			b.droppedLimits = true
		}
	}
	return nil
}

func limitMarkers(t *model.Type) []string {
	if t.Format == "date-time" {
		// metav1.Time provides the date-time schema; controller-gen rejects string limits on it.
		return nil
	}
	if t.Kind == model.Number {
		// A Quantity is int-or-string in the CRD. Minimum and Maximum do not apply to its schema.
		return nil
	}
	var out []string
	out = append(out, boundMarkers("MinLength", "MaxLength", t.MinLength, t.MaxLength)...)
	if t.Pattern != "" {
		out = append(out, "+kubebuilder:validation:Pattern="+markerString(t.Pattern))
	}
	for _, m := range []struct {
		name  string
		value *float64
	}{{"Minimum", t.Minimum}, {"Maximum", t.Maximum}} {
		if m.value != nil {
			out = append(out, "+kubebuilder:validation:"+m.name+"="+strconv.FormatFloat(*m.value, 'f', -1, 64))
		}
	}
	return out
}

func boundMarkers(minName, maxName string, low, high *int64) []string {
	var out []string
	if low != nil && *low > 0 {
		out = append(out, fmt.Sprintf("+kubebuilder:validation:%s=%d", minName, *low))
	}
	if high != nil {
		out = append(out, fmt.Sprintf("+kubebuilder:validation:%s=%d", maxName, *high))
	}
	return out
}

func prefixed(prefix string, markers []string) []string {
	out := make([]string, len(markers))
	for i, m := range markers {
		out[i] = strings.Replace(m, "+kubebuilder:validation:", "+kubebuilder:validation:"+prefix, 1)
	}
	return out
}

// markerString writes a marker string value: a raw string when it can, a quoted one otherwise.
func markerString(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// docLines splits a description into comment lines. A line that starts with "+" would be read as
// a marker, so it gets a backslash.
func docLines(description string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(description), "\n") {
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(strings.TrimSpace(line), "+") {
			line = `\` + strings.TrimSpace(line)
		}
		out = append(out, line)
	}
	if len(out) == 1 && out[0] == "" {
		return nil
	}
	return out
}

// scalar declares the named type of the values of a map, which holds the limits of the values.
func (b *builder) scalar(name, goType string, markers []string, path string) string {
	b.scalars = append(b.scalars, &crScalar{Name: name, GoType: goType, Markers: markers, path: path,
		Doc: []string{fmt.Sprintf("%s is a value of %s.", name, path)}})
	return name
}

// typeName is the Go type of a component in the custom resource. It starts with the kind, so two
// kinds in one package cannot declare the same type.
func (b *builder) typeName(component string) string {
	name := goName(component)
	if strings.HasPrefix(name, b.kind) {
		return name
	}
	return b.kind + name
}

// object builds the struct of an object or a oneOf. A path with different validation gets its own struct.
func (b *builder) object(t *model.Type, path string, inItem bool) (*crType, error) {
	if t.Schema == "" {
		return nil, unsupported(path, "an inline object is not supported")
	}
	name := b.typeName(t.Schema)
	if existing := b.types[name]; existing != nil && existing.schema != t.Schema {
		return nil, collision(path, "the components %s and %s both are the Go type %s", existing.schema, t.Schema, name)
	}
	if existing := b.types[name]; existing != nil && existing.path != path &&
		(b.hasNestedValidation(existing.path) || b.hasNestedValidation(path)) {
		hash := sha256.Sum256([]byte(path))
		name = fmt.Sprintf("%sAt%s%x", name, goName(path), hash[:6])
	}
	if existing, ok := b.types[name]; ok {
		if existing.schema != t.Schema {
			return nil, collision(path, "the components %s and %s both are the Go type %s", existing.schema, t.Schema, name)
		}
		if t.Kind == model.OneOf && len(existing.groups) != 0 && existing.groups[0].AllowNone != t.AllowNone {
			return nil, unsupported(path, "one component cannot have different requireOne rules at different CR fields")
		}
		// The type was built for another place. A list item or a map value still cannot hold an
		// immutable field, so check that again here.
		if inItem {
			if field := immutableField(existing, path, map[*crType]bool{}); field != "" {
				return nil, unsupported(field, "kubectl cannot reject a change of an immutable field inside a list item or a map value")
			}
		}
		return existing, nil
	}
	obj := &crType{Name: name, schema: t.Schema, path: path, Doc: []string{fmt.Sprintf("%s is the %s object of the API.", name, t.Schema)}}
	b.types[name] = obj
	b.order = append(b.order, obj)
	inGroup := map[string]bool{}
	for _, g := range t.Groups {
		for _, arm := range g.Arms {
			inGroup[arm] = true
		}
	}
	for _, f := range t.Fields {
		if f.Behavior == model.Computed || f.Behavior == "" || f.Create == nil {
			continue
		}
		if len(b.file.Field(t.Schema, f.Name).RequestValueBy) != 0 {
			continue
		}
		if (t.Kind == model.OneOf || inGroup[f.Name]) && b.file.Field(t.Schema, f.Name).Required {
			return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":types." + t.Schema + ".fields." + f.Name + ".required", msg: "a oneOf arm cannot be required on its own"}
		}
		required := t.Kind != model.OneOf && f.Create.Required && !inGroup[f.Name]
		child := path + "." + f.Name
		field, err := b.field(obj.Name, t.Schema, f.Name, f.Description, f.Type, required, requiredScalarWithoutPresence(f.Create, f.Type), child, inItem)
		if err != nil {
			return nil, err
		}
		if err := b.contractDefault(field, f.Create, field.validationPath); err != nil {
			return nil, err
		}
		if f.Behavior == model.Immutable {
			if inItem {
				return nil, unsupported(child, "kubectl cannot reject a change of an immutable field inside a list item or a map value")
			}
			immutable(field, field.validationPath)
		}
		obj.Fields = append(obj.Fields, field)
		if field.refForm == "split" {
			obj.Fields = append(obj.Fields, splitReferenceField(field))
		}
	}
	if len(obj.Fields) == 0 && (t.Kind != model.Object || len(t.Fields) != 0) {
		return nil, unsupported(path, "an object without fields that a request sends is not supported yet")
	}
	groups := t.Groups
	if t.Kind == model.OneOf {
		arms := make([]string, len(obj.Fields))
		for i, f := range obj.Fields {
			arms[i] = f.API
		}
		groups = append([]model.OneOfGroup{{Arms: arms, AllowNone: t.AllowNone}}, groups...)
	}
	groups, err := b.overrideGroups(t.Schema, groups, obj.Fields)
	if err != nil {
		return nil, err
	}
	groups, err = b.groupFields(t.Schema, obj, groups)
	if err != nil {
		return nil, err
	}
	if err := groupRules(obj, groups, path); err != nil {
		return nil, err
	}
	splitReferenceRules(obj)
	return obj, nil
}

// hasNestedValidation reports whether a path changes markers inside an object.
func (b *builder) hasNestedValidation(path string) bool {
	if b.file == nil || b.file.Validators == nil {
		return false
	}
	for key := range b.file.Validators.Fields {
		if key == path+"#type" || strings.HasPrefix(key, path+".") {
			return true
		}
	}
	return false
}

func splitReferenceRules(t *crType) {
	for _, f := range t.Fields {
		if f.refForm == "split" {
			rule := fmt.Sprintf("!(has(self.%s) && has(self.%s))", f.JSON, f.refSibling)
			t.Markers = append(t.Markers, celMarker(rule, "Only one of "+f.JSON+" or "+f.refSibling+" can be set"))
		}
	}
}

func splitReferenceField(f *crField) *crField {
	return &crField{
		JSON: f.refSibling, GoName: goName(f.refSibling), GoType: "*ResourceRef", Optional: true,
		Doc: []string{fmt.Sprintf("It refers to a %s custom resource.", f.ref)}, synthetic: true,
	}
}

// enum builds the map of an enum component, once per component.
func (b *builder) enum(t *model.Type, path string) (*enumMap, error) {
	if e, ok := b.enums[t.Schema]; ok {
		return e, nil
	}
	sdkType, err := b.ix.schemaType(t.Schema)
	if err != nil {
		return nil, err
	}
	base := lowerFirst(b.typeName(t.Schema))
	e := &enumMap{Func: base + "ToAPI", Var: base + "Values", SDKType: b.alias + "." + sdkType, schema: t.Schema}
	seen := map[string]string{}
	line, overridden := b.file.Enum(t.Schema)
	values := t.Values
	if overridden {
		values = nil
		for _, v := range t.Values {
			if v != t.EnumZero && !line.IsRejected(v) {
				values = append(values, v)
			}
		}
		if line.Zero != "" && !line.RejectZero {
			values = append([]string{t.EnumZero}, values...)
		}
	}
	constantByName := map[string]string{}
	for _, v := range values {
		name := enumName(v, t.EnumZero)
		if overridden {
			switch {
			case v == t.EnumZero && line.Zero != "":
				name = line.Zero
			case line.Names[v] != "":
				name = line.Names[v]
			case line.Verbatim:
				name = strings.TrimPrefix(v, line.TrimPrefix)
			case line.TrimPrefix != "":
				name = enumName(strings.TrimPrefix(v, line.TrimPrefix), "")
			default:
				name = enumName(v, t.EnumZero)
			}
		}
		if prior, ok := seen[name]; ok || name == "" {
			return nil, collision(path, "the enum values %s and %s both are %q", prior, v, name)
		}
		seen[name] = v
		c, err := b.ix.enumConst(sdkType, v)
		if err != nil {
			return nil, err
		}
		constantByName[name] = b.alias + "." + c
		e.Values = append(e.Values, enumValue{API: v, Name: name, Const: constantByName[name]})
	}
	if overridden {
		for alias, target := range line.Aliases {
			if _, ok := constantByName[alias]; !ok {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":enums." + t.Schema + ".aliases", msg: fmt.Sprintf("%q is not an accepted CR value", alias)}
			}
			if alias == target || line.Aliases[target] != "" || constantByName[target] == "" || target == line.Zero {
				return nil, &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":enums." + t.Schema + ".aliases", msg: fmt.Sprintf("%q must map to another accepted business value", alias)}
			}
			for i := range e.Values {
				if e.Values[i].Name == alias {
					e.Values[i].Const = constantByName[target]
				}
			}
		}
		if len(values) != len(t.Values) || len(line.Aliases) != 0 {
			b.usedEnums[t.Schema] = true
		} else {
			for i, v := range t.Values {
				if values[i] != v || e.Values[i].Name != enumName(v, t.EnumZero) {
					b.usedEnums[t.Schema] = true
					break
				}
			}
		}
	}
	if len(e.Values) == 0 {
		return nil, unsupported(path, "an enum without values")
	}
	b.enums[t.Schema] = e
	b.enumSeq = append(b.enumSeq, e)
	return e, nil
}

// checkNames rejects two Go types or two fields with one name.
func (b *builder) checkNames(d *kindData) error {
	reserved := map[string]string{}
	for _, name := range []string{d.Kind, d.Kind + "Spec", d.Kind + "Status", d.Kind + "List"} {
		reserved[name] = "the kind"
	}
	for _, t := range d.Types {
		if _, ok := reserved[t.Name]; ok {
			return collision(t.schema, "the component %s is the Go type %s, which the kind declares", t.schema, t.Name)
		}
		reserved[t.Name] = "the component " + t.schema
	}
	for _, s := range d.Scalars {
		if prior, ok := reserved[s.Name]; ok {
			return collision(s.path, "the values of %s are the Go type %s, which %s declares", s.path, s.Name, prior)
		}
		reserved[s.Name] = "the values of " + s.path
	}
	for _, t := range append([]*crType{d.Spec}, d.Types...) {
		seen := map[string]string{}
		jsonNames := map[string]string{}
		for _, f := range t.Fields {
			if prior, ok := seen[f.GoName]; ok {
				return collision(t.Name, "the fields %s and %s both are the Go field %s", prior, f.API, f.GoName)
			}
			seen[f.GoName] = f.API
			if prior, ok := jsonNames[f.JSON]; ok {
				return collision(t.Name, "the fields %s and %s both are the CR property %s", prior, f.API, f.JSON)
			}
			jsonNames[f.JSON] = f.API
		}
	}
	return nil
}
