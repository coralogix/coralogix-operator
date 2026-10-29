package generator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
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
		if f.Behavior == model.Computed || f.Create == nil {
			continue
		}
		required := f.Create.Required && !inGroup[f.Name]
		field, err := b.field(b.kind, f.Name, f.Description, f.Type, required, "spec."+f.Name, false)
		if err != nil {
			return nil, err
		}
		if f.Behavior == model.Immutable {
			immutable(field, "spec."+f.Name)
		}
		spec.Fields = append(spec.Fields, field)
	}
	presenceRules(spec, spec, nil, false)
	if err := groupRules(spec, r.Groups, "spec"); err != nil {
		return nil, err
	}
	return spec, nil
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

// field builds one field of the Go type owner. inItem is true inside a list item or a map value,
// where kubectl cannot compare a value with its old value.
func (b *builder) field(owner, name, description string, t *model.Type, required bool, path string, inItem bool) (*crField, error) {
	if err := wrapped(t, path); err != nil {
		return nil, err
	}
	f := &crField{API: name, JSON: name, GoName: goName(name), Optional: !required, Doc: docLines(description), typ: t}
	if !required {
		f.Markers = append(f.Markers, "+optional")
	}
	switch t.Kind {
	case model.List, model.Set, model.Map:
		elem, err := b.elemType(t.Elem, path, f)
		if err != nil {
			return nil, err
		}
		if t.Kind == model.Map {
			if markers := scalarMarkers(t.Elem, f.enum); f.object == nil && len(markers) != 0 {
				elem = b.scalar(owner+f.GoName+"Value", elem, markers, path)
			}
			f.GoType = "map[string]" + elem
			f.Markers = append(f.Markers, boundMarkers("MinProperties", "MaxProperties", t.MinItems, t.MaxItems)...)
			return f, nil
		}
		f.GoType = "[]" + elem
		if t.Kind == model.Set {
			if f.object != nil {
				return nil, unsupported(path, "a set of objects is not supported yet")
			}
			f.Markers = append(f.Markers, "+listType=set")
		}
		f.Markers = append(f.Markers, boundMarkers("MinItems", "MaxItems", t.MinItems, t.MaxItems)...)
		f.Markers = append(f.Markers, prefixed("items:", scalarMarkers(t.Elem, f.enum))...)
		return f, nil
	case model.Object, model.OneOf:
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
		f.Markers = append(f.Markers, scalarMarkers(t, f.enum)...)
	}
	if !required {
		f.GoType = "*" + f.GoType
	}
	return f, nil
}

// wrapped rejects a value that the API holds in a wrapper object or keys by a property: the
// renderer would write the bare value and lose the wrapper.
func wrapped(t *model.Type, path string) error {
	if len(t.Wrappers) != 0 || t.Keyed != nil {
		return unsupported(path, "wrapper and keyedBy values are not supported yet")
	}
	return nil
}

// elemType is the Go type of the items of a list or the values of a map.
func (b *builder) elemType(t *model.Type, path string, f *crField) (string, error) {
	if err := wrapped(t, path+"[]"); err != nil {
		return "", err
	}
	switch t.Kind {
	case model.Object, model.OneOf:
		obj, err := b.object(t, path, true)
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
	case model.String, model.Enum:
		if t.Format == "date-time" {
			return "", unsupported(path, "a date-time is not supported yet")
		}
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
		return "", unsupported(path, "a number is not supported yet: a custom resource holds it as a quantity")
	case model.Object, model.OneOf, model.List, model.Set, model.Map:
		// Not a scalar: the error below.
	}
	return "", unsupported(path, "a %s is not supported", t.Kind)
}

// scalarMarkers are the markers of the contract limits of a scalar. The limits are API limits, so
// kubectl rejects only a value that the API rejects too.
func scalarMarkers(t *model.Type, enum *enumMap) []string {
	var out []string
	if enum != nil {
		return []string{"+kubebuilder:validation:Enum=" + enum.Names()}
	}
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

// object builds the struct of an object or a oneOf, once per component.
func (b *builder) object(t *model.Type, path string, inItem bool) (*crType, error) {
	if t.Schema == "" {
		return nil, unsupported(path, "an inline object is not supported")
	}
	name := b.typeName(t.Schema)
	if existing, ok := b.types[name]; ok {
		if existing.schema != t.Schema {
			return nil, collision(path, "the components %s and %s both are the Go type %s", existing.schema, t.Schema, name)
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
	obj := &crType{Name: name, schema: t.Schema, Doc: []string{fmt.Sprintf("%s is the %s object of the API.", name, t.Schema)}}
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
		required := t.Kind != model.OneOf && f.Create.Required && !inGroup[f.Name]
		child := path + "." + f.Name
		field, err := b.field(obj.Name, f.Name, f.Description, f.Type, required, child, inItem)
		if err != nil {
			return nil, err
		}
		if f.Behavior == model.Immutable {
			if inItem {
				return nil, unsupported(child, "kubectl cannot reject a change of an immutable field inside a list item or a map value")
			}
			immutable(field, child)
		}
		obj.Fields = append(obj.Fields, field)
	}
	if len(obj.Fields) == 0 {
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
	return obj, groupRules(obj, groups, path)
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
	e := &enumMap{Func: base + "ToAPI", Var: base + "Values", SDKType: b.alias + "." + sdkType}
	seen := map[string]string{}
	for _, v := range t.Values {
		name := enumName(v, t.EnumZero)
		if prior, ok := seen[name]; ok || name == "" {
			return nil, collision(path, "the enum values %s and %s both are %q", prior, v, name)
		}
		seen[name] = v
		c, err := b.ix.enumConst(sdkType, v)
		if err != nil {
			return nil, err
		}
		e.Values = append(e.Values, enumValue{Name: name, Const: b.alias + "." + c})
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
		for _, f := range t.Fields {
			if prior, ok := seen[f.GoName]; ok {
				return collision(t.Name, "the fields %s and %s both are the Go field %s", prior, f.API, f.GoName)
			}
			seen[f.GoName] = f.API
		}
	}
	return nil
}
