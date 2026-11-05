package generator

import (
	"fmt"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
	"github.com/coralogix/coralogix-management-sdk/iac/sdkcheck"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// operations reads the SDK names of the calls.
func (b *builder) operations(r *model.Resource, d *kindData) error {
	client, err := b.ix.typeRef("resource")
	if err != nil {
		return err
	}
	d.Client = client.Name
	accessor, err := b.ix.method("clientset", "ClientSet")
	if err != nil {
		return err
	}
	d.Accessor = accessor.Name
	for _, op := range []struct {
		path   string
		method *string
		setter *string
		body   *string
	}{
		{"create", &d.CreateMethod, &d.CreateSetter, &d.CreateBody},
		{"get", &d.GetMethod, nil, nil},
		{"update", &d.UpdateMethod, &d.UpdateSetter, &d.UpdateBody},
		{"delete", &d.DeleteMethod, nil, nil},
	} {
		m, err := b.ix.method(op.path, client.Name)
		if err != nil {
			return err
		}
		*op.method = m.Name
		if op.setter == nil {
			continue
		}
		body, err := b.ix.typeRef(op.path + ".body")
		if err != nil {
			return unsupported(op.path, "the request has no body")
		}
		builderType, err := b.ix.typeRef(op.path)
		if err != nil {
			return err
		}
		setter, err := b.ix.method(op.path+".body", builderType.Name)
		if err != nil {
			return err
		}
		*op.body, *op.setter = body.Name, setter.Name
	}
	resource, err := b.ix.typeRef("fields")
	if err != nil {
		return err
	}
	id, err := b.ix.field(resource.Name, r.ID())
	if err != nil {
		return err
	}
	d.IDField, d.IDValue = id.Name, !strings.HasPrefix(id.Want, "*")
	if r.IDType.Kind == model.Integer {
		d.IDGoType = r.IDType.Format
		d.IDZeroValid = r.IDType.ZeroValid
		if d.IDGoType == "int32" {
			d.IDBits = 32
		} else {
			d.IDBits = 64
		}
		if strings.TrimPrefix(id.Want, "*") != d.IDGoType {
			return unsupported("resource.id", "the SDK id type %s does not match the API %s", id.Want, d.IDGoType)
		}
	} else if strings.TrimPrefix(id.Want, "*") != "string" {
		return unsupported("resource.id", "the SDK id type %s does not match the API string", id.Want)
	}
	if r.Create.Response.Field != "" {
		response, err := b.ix.typeRef("create.response")
		if err != nil {
			return err
		}
		field, err := b.ix.field(response.Name, r.Create.Response.Field)
		if err != nil {
			return err
		}
		d.CreateResponseField, d.CreateResponseValue = field.Name, field.Want == resource.Name
	}
	if r.Get.Response.Field != "" {
		response, err := b.ix.typeRef("get.response")
		if err != nil {
			return err
		}
		field, err := b.ix.field(response.Name, r.Get.Response.Field)
		if err != nil {
			return err
		}
		d.GetResponseField, d.GetResponseValue = field.Name, field.Want == resource.Name
	}
	if r.Policy.RequestWrapper != "" {
		d.ResourceBody = resource.Name
		for _, request := range []struct {
			body  string
			field *string
			value *bool
		}{{d.CreateBody, &d.CreateWrapperField, &d.CreateWrapperValue}, {d.UpdateBody, &d.UpdateWrapperField, &d.UpdateWrapperValue}} {
			ref, err := b.ix.field(request.body, r.Policy.RequestWrapper)
			if err != nil {
				return err
			}
			*request.field, *request.value = ref.Name, ref.Want == resource.Name
		}
	}
	if !r.Replace && !r.MaskInBody {
		builderType, err := b.ix.typeRef("update")
		if err != nil {
			return err
		}
		mask, err := b.ix.method("update.mask", builderType.Name)
		if err != nil {
			return err
		}
		d.MaskSetter = mask.Name
	}
	if r.MaskInBody {
		field, err := b.ix.field(d.UpdateBody, r.UpdateMask)
		if err != nil {
			return err
		}
		d.MaskInBody, d.MaskBodyField, d.MaskBodyValue = true, field.Name, field.Want == "string"
	}
	return nil
}

// requests builds the conversions of the spec to the Create and Update bodies, and the update mask.
func (b *builder) requests(r *model.Resource, d *kindData) error {
	var create, update []*model.Field
	var mask []string
	var err error
	bodyID := r.Policy.UpdateIDInBody
	for _, f := range r.Fields {
		if f.Create != nil {
			create = append(create, &model.Field{Name: f.Name, Type: f.Type.CreateType(), Create: f.Create, Attrs: *f.Create})
		}
		if bodyID && f.Name == r.ID() {
			continue // UpdateRequest copies the status id
		}
		if f.Update != nil {
			update = append(update, &model.Field{Name: f.Name, Type: f.Type.UpdateType(), Update: f.Update, Attrs: *f.Update})
			mask = append(mask, f.Name)
		}
	}
	createType, updateType := d.CreateBody, d.UpdateBody
	createSuffix, updateSuffix := "", ""
	if d.ResourceBody != "" {
		createType, updateType = d.ResourceBody, d.ResourceBody
		createSuffix, updateSuffix = "Create", "Update"
	}
	if d.CreateConv, err = b.conversionAs(d.Spec, createType, create, "spec", createSuffix); err != nil {
		return err
	}
	if d.UpdateConv, err = b.conversionAs(d.Spec, updateType, update, "spec", updateSuffix); err != nil {
		return err
	}
	if bodyID {
		owner := d.UpdateBody
		if d.BodyIDInWrapper {
			owner = d.ResourceBody
		}
		ref, err := b.ix.field(owner, r.ID())
		if err != nil {
			return err
		}
		if d.IDBits != 0 && strings.TrimPrefix(ref.Want, "*") != d.IDGoType {
			return unsupported("update.id", "the SDK id type %s does not match the API %s", ref.Want, d.IDGoType)
		}
		d.UpdateIDInBody, d.BodyIDField, d.BodyIDValue = true, ref.Name, !strings.HasPrefix(ref.Want, "*")
	}
	if r.Replace {
		return b.requestValues(r, d)
	}
	if r.UpdateMaskPattern != "" {
		valid, _, err := model.UpdateMaskRule(r.UpdateMaskPattern)
		if err != nil {
			return unsupported("update.mask", "%s", err)
		}
		for _, path := range mask {
			if !valid(path) {
				return unsupported("update.mask", "the update mask pattern rejects the field %s", path)
			}
		}
	}
	d.Mask = strings.Join(mask, ",")
	return b.requestValues(r, d)
}

func (b *builder) requestValues(r *model.Resource, d *kindData) error {
	if b.file == nil {
		return nil
	}
	for _, request := range []struct {
		name, body string
		values     *[]requestValueData
	}{{"create", d.CreateBody, &d.CreateValues}, {"update", d.UpdateBody, &d.UpdateValues}} {
		for _, name := range sortedKeys(b.file.API.RequestValues[request.name]) {
			location := "api.requestValues." + request.name + "." + name
			if name == r.ID() && request.name == "update" && r.Policy.UpdateIDInBody {
				return unsupported(location, "the status id already sets this field")
			}
			if name == r.UpdateMask && request.name == "update" && r.MaskInBody {
				return unsupported(location, "the update mask already sets this field")
			}
			ref, err := b.ix.field(request.body, name)
			if err != nil {
				return unsupported(location, "%s", err)
			}
			source := b.file.API.RequestValues[request.name][name]
			expr, pointer, err := b.requestValueExpression(source, d, location)
			if err != nil {
				return err
			}
			switch ref.Want {
			case "string":
				if pointer {
					expr = "ptr.Deref(" + expr + ", \"\")"
				}
			case "*string":
				if !pointer {
					expr = "ptr.To(" + expr + ")"
				}
			default:
				return unsupported(location, "the sibling field must be a string")
			}
			*request.values = append(*request.values, requestValueData{Field: ref.Name, Expr: expr})
		}
	}
	return nil
}

func (b *builder) requestValueExpression(source overrides.RequestValue, d *kindData, location string) (string, bool, error) {
	switch source.Source {
	case "uuid":
		d.HasUUID = true
		return strconv.Quote(source.Value) + " + gouuid.NewString()", false, nil
	case "literal":
		return strconv.Quote(source.Value), false, nil
	}
	name := strings.TrimPrefix(source.Source, "spec.")
	for _, field := range d.Spec.Fields {
		if field.JSON == name && field.typ != nil && field.typ.Kind == model.String {
			return "in.Spec." + field.GoName, field.Optional, nil
		}
	}
	return "", false, unsupported(location, "the spec has no string field %s", name)
}

// conversion writes the method that converts a struct of the custom resource to an SDK request
// type, once per pair, and returns its name.
func (b *builder) conversion(cr *crType, sdkType string, fields []*model.Field, path string) (string, error) {
	return b.conversionAs(cr, sdkType, fields, path, b.conversionSuffix)
}

func (b *builder) conversionAs(cr *crType, sdkType string, fields []*model.Field, path, suffix string) (string, error) {
	key := cr.Name + "->" + sdkType + suffix
	if name, ok := b.convs[key]; ok {
		return name, nil
	}
	name := "to" + sdkType + suffix
	b.convs[key] = name
	previous := b.conversionSuffix
	b.conversionSuffix = suffix
	defer func() { b.conversionSuffix = previous }()
	var body strings.Builder
	for _, rf := range fields {
		field, src := sourceField(cr, rf.Name)
		if field == nil {
			component := cr.schema
			if component == "" && b.file != nil {
				component = b.file.Resource
			}
			if by := b.file.Field(component, rf.Name).RequestValueBy; len(by) != 0 {
				code, err := b.requestValueBy(cr, sdkType, rf, by, path)
				if err != nil {
					return "", err
				}
				b.used[overrides.Line{Component: component, Field: rf.Name}] = true
				body.WriteString(code)
				continue
			}
			if rf.Attrs.Required {
				return "", unsupported(path+"."+rf.Name, "the request requires a field that the custom resource does not have")
			}
			continue
		}
		code, err := b.assignFrom(field, rf.Type, sdkType, path+"."+field.JSON, src)
		if err != nil {
			return "", err
		}
		body.WriteString(code)
	}
	lookupParam := ""
	if b.hasNameLookup {
		lookupParam = ", lookup *NameLookupContext"
	}
	b.code = append(b.code, fmt.Sprintf(`
// %s converts the %s to the API type %s.
func (in *%s) %s(ctx context.Context, namespace string%s) (*%s.%s, error) {
	out := &%s.%s{}
%s	return out, nil
}
`, name, cr.Name, sdkType, cr.Name, name, lookupParam, b.alias, sdkType, b.alias, sdkType, body.String()))
	return name, nil
}

// sourceField finds an API field at its released CR path.
func sourceField(cr *crType, api string) (*crField, string) {
	for _, f := range cr.Fields {
		if f.group {
			for _, child := range f.object.Fields {
				if child.API == api {
					return child, "in." + f.GoName + "." + child.GoName
				}
			}
			continue
		}
		if f.API == api {
			return f, "in." + f.GoName
		}
	}
	return nil, ""
}

// requestValueBy sends a hidden string or enum value from the selected object arm.
func (b *builder) requestValueBy(cr *crType, sdkType string, field *model.Field, by map[string]string, path string) (string, error) {
	location := path + "." + field.Name
	if field.Type.Kind != model.String && field.Type.Kind != model.Enum {
		return "", unsupported(location, "requestValueBy needs a string or enum API field")
	}
	var choices []model.OneOfGroup
	choices = append(choices, cr.groups...)
	for _, f := range cr.Fields {
		if f.group {
			choices = append(choices, f.object.groups...)
		}
	}
	if !slices.ContainsFunc(choices, func(g model.OneOfGroup) bool {
		if field.Attrs.Required && len(by) != len(g.Arms) {
			return false
		}
		for source := range by {
			if !slices.Contains(g.Arms, source) {
				return false
			}
		}
		return true
	}) {
		return "", unsupported(location, "requestValueBy must name arms of one oneOf group; a required value needs every arm")
	}
	ref, err := b.ix.field(sdkType, field.Name)
	if err != nil {
		return "", err
	}
	var code strings.Builder
	for i, source := range sortedKeys(by) {
		arm, src := sourceField(cr, source)
		if arm == nil || (arm.typ.Kind != model.Object && arm.typ.Kind != model.OneOf) || !arm.Optional {
			return "", unsupported(location, "requestValueBy source %s must be an optional object arm", source)
		}
		value := by[source]
		expr := strconv.Quote(value)
		if field.Type.Kind == model.Enum {
			if value == field.Type.EnumZero && value != "" {
				component := cr.schema
				if component == "" {
					component = b.file.Resource
				}
				line := overrides.Line{Component: component, Field: field.Name}
				return "", &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".requestValueBy", msg: "the enum zero value cannot represent a selected arm"}
			}
			if !slices.Contains(field.Type.Values, value) {
				return "", unsupported(location, "requestValueBy value %s is not in the API enum", value)
			}
			enumType, err := b.ix.schemaType(field.Type.Schema)
			if err != nil {
				return "", err
			}
			constant, err := b.ix.enumConst(enumType, value)
			if err != nil {
				return "", err
			}
			expr = b.alias + "." + constant
		}
		if i > 0 {
			code.WriteString("else ")
		}
		code.WriteString("if " + src + " != nil {\n")
		code.WriteString(b.set("out."+ref.Name, ref.Want, expr, false))
		code.WriteString("}")
	}
	if field.Attrs.Required {
		code.WriteString(" else {\nreturn nil, fmt.Errorf(" + strconv.Quote(location+": exactly one source arm is required") + ")\n}")
	}
	return code.String() + "\n", nil
}

// assign writes the statements that set one SDK field from the custom resource field.
func (b *builder) assignFrom(f *crField, rt *model.Type, owner, path, src string) (string, error) {
	ref, err := b.ix.field(owner, f.API)
	if err != nil {
		return "", err
	}
	target := "out." + ref.Name
	if f.refForm == "split" {
		sibling := strings.TrimSuffix(src, f.GoName) + goName(f.refSibling)
		resolved := b.temp("id")
		call := fmt.Sprintf("ResolveResourceID(ctx, namespace, ResourceIDRef{ResourceRef: %s}, %q)", sibling, f.ref)
		if f.refVersion != "" {
			call = fmt.Sprintf("ResolveResourceIDVersion(ctx, namespace, ResourceIDRef{ResourceRef: %s}, %q, %q)", sibling, f.ref, f.refVersion)
		}
		return fmt.Sprintf("if %s != nil {\n%s} else if %s != nil {\n%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n%s}\n", src, b.set(target, ref.Want, src, true), sibling, resolved, call, path, b.set(target, ref.Want, resolved, false)), nil
	}
	switch {
	case f.typ.Kind == model.List || f.typ.Kind == model.Set || f.typ.Kind == model.Map:
		stmts, expr, _, err := b.value(f, rt, src, ref.Want, path)
		if err != nil {
			return "", err
		}
		code := stmts + target + " = " + expr + "\n"
		if f.Optional || f.omitEmptyInRequest {
			return "if len(" + src + ") != 0 {\n" + code + "}\n", nil
		}
		return code, nil
	case f.Optional:
		stmts, expr, isPtr, err := b.value(f, rt, "(*"+src+")", strings.TrimPrefix(ref.Want, "*"), path)
		if err != nil {
			return "", err
		}
		if stmts == "" && strings.HasPrefix(ref.Want, "*") && expr == "(*"+src+")" {
			return target + " = " + src + "\n", nil // the same pointer type
		}
		return "if " + src + " != nil {\n" + stmts + b.set(target, ref.Want, expr, isPtr) + "}\n", nil
	}
	stmts, expr, isPtr, err := b.value(f, rt, src, strings.TrimPrefix(ref.Want, "*"), path)
	if err != nil {
		return "", err
	}
	return stmts + b.set(target, ref.Want, expr, isPtr), nil
}

// set writes the assignment of expr to an SDK field of type want. isPtr: expr is a pointer.
func (b *builder) set(target, want, expr string, isPtr bool) string {
	switch {
	case strings.HasPrefix(want, "*") && isPtr:
		return target + " = " + expr + "\n"
	case strings.HasPrefix(want, "*") && token.IsIdentifier(expr):
		return target + " = &" + expr + "\n"
	case strings.HasPrefix(want, "*"):
		v := b.temp("v")
		return v + " := " + expr + "\n" + target + " = &" + v + "\n"
	case isPtr:
		return target + " = *" + expr + "\n"
	}
	return target + " = " + expr + "\n"
}

func (b *builder) temp(prefix string) string {
	b.tmp++
	return prefix + strconv.Itoa(b.tmp)
}

// value writes the statements that convert src, a value of the custom resource, to the SDK value
// type want. It returns the expression of the result and whether that expression is a pointer.
func (b *builder) value(f *crField, rt *model.Type, src, want, path string) (string, string, bool, error) {
	if len(rt.Wrappers) != 0 {
		return b.wrappedValue(f, rt, src, path)
	}
	if f.ref != "" {
		if rt.Kind == model.List || rt.Kind == model.Set {
			elemWant, ok := strings.CutPrefix(want, "[]")
			if !ok || elemWant != referenceWant(f) {
				return "", "", false, unsupported(path, "a reference list needs []%s in the SDK, got %s", referenceWant(f), want)
			}
			out, item := b.temp("v"), b.temp("e")
			statements, value := b.referenceValue(f, item, path+"[]")
			return fmt.Sprintf("%s := make([]%s, 0, len(%s))\nfor _, %s := range %s {\n%s%s = append(%s, %s)\n}\n", out, elemWant, src, item, src, statements, out, out, value), out, false, nil
		}
		if want != referenceWant(f) {
			return "", "", false, unsupported(path, "a reference needs %s in the SDK, got %s", referenceWant(f), want)
		}
		statements, value := b.referenceValue(f, src, path)
		return statements, value, false, nil
	}
	switch rt.Kind {
	case model.List, model.Set, model.Map:
		elemWant, ok := strings.CutPrefix(want, "[]")
		if rt.Kind == model.Map {
			elemWant, ok = strings.CutPrefix(want, "map[string]")
		}
		if !ok {
			return "", "", false, fmt.Errorf("%s: the SDK type %s is not a collection", path, want)
		}
		elem := &crField{API: f.API, JSON: f.JSON, typ: f.typ.Elem, object: f.object, enum: f.enum}
		v, e, k := b.temp("v"), b.temp("e"), "_"
		if rt.Kind == model.Map {
			k = b.temp("k")
		}
		item := e
		if value := b.mapValue(f); value != nil {
			item = value.GoType + "(" + e + ")" // the named value type of a map holds the limits
		}
		stmts, expr, isPtr, err := b.value(elem, rt.Elem, item, elemWant, path+"[]")
		if err != nil {
			return "", "", false, err
		}
		if stmts == "" && expr == e && builtinTypes[elemWant] {
			return "", src, false, nil // the same Go type: the value needs no conversion
		}
		if isPtr && !strings.HasPrefix(elemWant, "*") {
			expr = "*" + expr
		} else if !isPtr && strings.HasPrefix(elemWant, "*") {
			expr = "&" + expr
		}
		goType := qualify(want, b.alias)
		if rt.Kind == model.Map {
			return fmt.Sprintf("%s := make(%s, len(%s))\nfor %s, %s := range %s {\n%s%s[%s] = %s\n}\n", v, goType, src, k, e, src, stmts, v, k, expr), v, false, nil
		}
		return fmt.Sprintf("%s := make(%s, 0, len(%s))\nfor %s, %s := range %s {\n%s%s = append(%s, %s)\n}\n", v, goType, src, k, e, src, stmts, v, v, expr), v, false, nil
	case model.Object, model.OneOf:
		if f.object != nil && len(f.object.Fields) == 0 && len(rt.Fields) == 0 && want == "map[string]interface{}" {
			return "", "map[string]interface{}{}", false, nil
		}
		fields := rt.Fields
		name, err := b.conversion(f.object, strings.TrimPrefix(want, "*"), fields, path)
		if err != nil {
			return "", "", false, err
		}
		v := b.temp("v")
		args := "ctx, namespace"
		if b.hasNameLookup {
			args += ", lookup"
		}
		return fmt.Sprintf("%s, err := %s.%s(%s)\nif err != nil {\nreturn nil, err\n}\n", v, deref(src), name, args), v, true, nil
	case model.Enum:
		v := b.temp("v")
		return fmt.Sprintf("%s, err := %s(%s)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, f.enum.Func, arg(src), path), v, false, nil
	case model.Integer:
		if rt.WireString {
			return "", "strconv.FormatInt(" + arg(src) + ", 10)", false, nil
		}
	case model.String:
		if rt.Format == "date-time" {
			if want != "time.Time" {
				return "", "", false, unsupported(path, "the SDK date-time field must be time.Time, got %s", want)
			}
			return "", src + ".Time", false, nil
		}
	case model.Number:
		switch want {
		case "float32":
			return "", "float32(" + src + ".AsApproximateFloat64())", false, nil
		case "float64":
			return "", src + ".AsApproximateFloat64()", false, nil
		}
		return "", "", false, unsupported(path, "the SDK number field must be float32 or float64, got %s", want)
	case model.Bool:
		// A scalar needs no conversion.
	}
	return "", src, false, nil
}

func referenceWant(f *crField) string {
	if f.refIDType != "" {
		return f.refIDType
	}
	return "string"
}

func (b *builder) referenceValue(f *crField, src, path string) (string, string) {
	v := b.temp("v")
	if f.refForm == "named" || f.refForm == "namedBackendOnly" {
		if f.refIDType != "" {
			backend := b.temp("backend")
			name := b.temp("name")
			bits := 64
			if f.refIDType == "int32" {
				bits = 32
			}
			statements := fmt.Sprintf("var %s *int64\nvar %s *string\nif %s.BackendRef != nil {\n%s = %s.BackendRef.Name\nif %s.BackendRef.ID != nil {\nid := int64(*%s.BackendRef.ID)\n%s = &id\n}\n}\n", backend, name, src, name, src, src, src, backend)
			statements += fmt.Sprintf("%s, err := ResolveNamedNumericResourceID(ctx, namespace, %s, %s, %s.ResourceRef, %q, %q, %q, lookup, %s, %d)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, backend, name, src, f.ref, f.refVersion, f.refLookup, f.refLookup, bits, path)
			return statements, f.refIDType + "(" + v + ")"
		}
		value := arg(src)
		if f.refForm == "namedBackendOnly" {
			if f.refType == "" {
				value = "NamedResourceIDRef{BackendRef: &" + src + ".BackendRef}"
			} else {
				idField := f.refIDField
				if idField == "" {
					idField = "ID"
				}
				value = "NamedResourceIDRef{BackendRef: &NamedBackendRef{ID: " + src + ".BackendRef." + idField + ", Name: " + src + ".BackendRef.Name}}"
			}
		}
		call := fmt.Sprintf("ResolveNamedResourceID(ctx, namespace, %s, %q, %q, %q, lookup, %s)", value, f.ref, f.refVersion, f.refLookup, f.refLookup)
		return fmt.Sprintf("%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, call, path), v
	}
	if f.refIDType != "" {
		backend := b.temp("backend")
		resource := src + ".ResourceRef"
		statements := "var " + backend + " *int64\n"
		if f.refForm != "resourceRefOnly" {
			statements += fmt.Sprintf("if %s.BackendRef != nil {\nid := int64(%s.BackendRef.ID)\n%s = &id\n}\n", src, src, backend)
		}
		bits := 64
		if f.refIDType == "int32" {
			bits = 32
		}
		statements += fmt.Sprintf("%s, err := ResolveNumericResourceID(ctx, namespace, %s, %s, %q, %q, %d)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, backend, resource, f.ref, f.refVersion, bits, path)
		return statements, f.refIDType + "(" + v + ")"
	}
	value := arg(src)
	if f.refForm == "resourceRefOnly" {
		value = "ResourceIDRef{ResourceRef: " + src + ".ResourceRef}"
	} else if f.refType != "" {
		value = "ResourceIDRef(" + value + ")"
	}
	call := fmt.Sprintf("ResolveResourceID(ctx, namespace, %s, %q)", value, f.ref)
	if f.refVersion != "" {
		call = fmt.Sprintf("ResolveResourceIDVersion(ctx, namespace, %s, %q, %q)", value, f.ref, f.refVersion)
	}
	return fmt.Sprintf("%s, err := %s\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, call, path), v
}

// wrappedValue restores each one-property API wrapper around the CR value.
func (b *builder) wrappedValue(f *crField, rt *model.Type, src, path string) (string, string, bool, error) {
	types := make([]string, len(rt.Wrappers))
	fields := make([]sdkcheck.Ref, len(rt.Wrappers))
	for i, w := range rt.Wrappers {
		if w.Schema == "" {
			return "", "", false, unsupported(path, "the request wrapper has no component")
		}
		name, err := b.ix.schemaType(w.Schema)
		if err != nil {
			return "", "", false, err
		}
		field, err := b.ix.field(name, w.Field)
		if err != nil {
			return "", "", false, err
		}
		types[i], fields[i] = name, field
	}
	inner := *f
	inner.typ = f.typ.Unwrapped()
	last := len(fields) - 1
	stmts, expr, isPtr, err := b.value(&inner, rt.Unwrapped(), src, fields[last].Want, path)
	if err != nil {
		return "", "", false, err
	}
	for i := last; i >= 0; i-- {
		v := b.temp("w")
		stmts += fmt.Sprintf("%s := %s.%s{}\n", v, b.alias, types[i])
		stmts += b.set(v+"."+fields[i].Name, fields[i].Want, expr, isPtr)
		expr, isPtr = v, false
	}
	return stmts, expr, false, nil
}

// mapValue returns the named type of the values of the map field f, nil when they have none.
func (b *builder) mapValue(f *crField) *crScalar {
	name, ok := strings.CutPrefix(f.GoType, "map[string]")
	if !ok {
		return nil
	}
	for _, s := range b.scalars {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// deref turns "(*x)" into "x" for a method call: Go dereferences the pointer itself.
func deref(src string) string {
	if inner, ok := strings.CutPrefix(src, "(*"); ok {
		return strings.TrimSuffix(inner, ")")
	}
	return src
}

// arg turns "(*x)" into "*x" for a function argument.
func arg(src string) string {
	if inner, ok := strings.CutPrefix(src, "("); ok {
		return strings.TrimSuffix(inner, ")")
	}
	return src
}
