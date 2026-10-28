package generator

import (
	"fmt"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"
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
	d.IDField, d.IDValue = id.Name, id.Want == "string"
	if !r.Replace {
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
	return nil
}

// requests builds the conversions of the spec to the Create and Update bodies, and the update mask.
func (b *builder) requests(r *model.Resource, d *kindData) error {
	var create, update []*model.Field
	var mask []string
	for _, f := range r.Fields {
		if f.Create != nil {
			create = append(create, &model.Field{Name: f.Name, Type: f.Type.CreateType(), Create: f.Create, Attrs: *f.Create})
		}
		if f.Update != nil {
			update = append(update, &model.Field{Name: f.Name, Type: f.Type.UpdateType(), Update: f.Update, Attrs: *f.Update})
			mask = append(mask, f.Name)
		}
	}
	if _, err := b.conversion(d.Spec, d.CreateBody, create, "spec"); err != nil {
		return err
	}
	if _, err := b.conversion(d.Spec, d.UpdateBody, update, "spec"); err != nil {
		return err
	}
	if r.Replace {
		return nil
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
	return nil
}

// conversion writes the method that converts a struct of the custom resource to an SDK request
// type, once per pair, and returns its name.
func (b *builder) conversion(cr *crType, sdkType string, fields []*model.Field, path string) (string, error) {
	key := cr.Name + "->" + sdkType
	if name, ok := b.convs[key]; ok {
		return name, nil
	}
	name := "to" + sdkType
	b.convs[key] = name
	var body strings.Builder
	for _, rf := range fields {
		i := slices.IndexFunc(cr.Fields, func(f *crField) bool { return f.API == rf.Name })
		if i < 0 {
			if rf.Attrs.Required {
				return "", unsupported(path+"."+rf.Name, "the request requires a field that the custom resource does not have")
			}
			continue
		}
		code, err := b.assign(cr.Fields[i], rf.Type, sdkType, path+"."+rf.Name)
		if err != nil {
			return "", err
		}
		body.WriteString(code)
	}
	b.code = append(b.code, fmt.Sprintf(`
// %s converts the %s to the API type %s.
func (in *%s) %s(ctx context.Context, namespace string) (*%s.%s, error) {
	out := &%s.%s{}
%s	return out, nil
}
`, name, cr.Name, sdkType, cr.Name, name, b.alias, sdkType, b.alias, sdkType, body.String()))
	return name, nil
}

// assign writes the statements that set one SDK field from the custom resource field.
func (b *builder) assign(f *crField, rt *model.Type, owner, path string) (string, error) {
	ref, err := b.ix.field(owner, f.API)
	if err != nil {
		return "", err
	}
	target := "out." + ref.Name
	src := "in." + f.GoName
	switch {
	case f.typ.Kind == model.List || f.typ.Kind == model.Set || f.typ.Kind == model.Map:
		stmts, expr, _, err := b.value(f, rt, src, ref.Want, path)
		if err != nil {
			return "", err
		}
		code := stmts + target + " = " + expr + "\n"
		if f.Optional {
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
	switch rt.Kind {
	case model.List, model.Set, model.Map:
		elemWant, ok := strings.CutPrefix(want, "[]")
		if rt.Kind == model.Map {
			elemWant, ok = strings.CutPrefix(want, "map[string]")
		}
		if !ok {
			return "", "", false, fmt.Errorf("%s: the SDK type %s is not a collection", path, want)
		}
		elem := &crField{API: f.API, typ: f.typ.Elem, object: f.object, enum: f.enum}
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
		if isPtr {
			expr = "*" + expr
		}
		goType := qualify(want, b.alias)
		if rt.Kind == model.Map {
			return fmt.Sprintf("%s := make(%s, len(%s))\nfor %s, %s := range %s {\n%s%s[%s] = %s\n}\n", v, goType, src, k, e, src, stmts, v, k, expr), v, false, nil
		}
		return fmt.Sprintf("%s := make(%s, 0, len(%s))\nfor %s, %s := range %s {\n%s%s = append(%s, %s)\n}\n", v, goType, src, k, e, src, stmts, v, v, expr), v, false, nil
	case model.Object, model.OneOf:
		fields := rt.Fields
		name, err := b.conversion(f.object, strings.TrimPrefix(want, "*"), fields, path)
		if err != nil {
			return "", "", false, err
		}
		v := b.temp("v")
		return fmt.Sprintf("%s, err := %s.%s(ctx, namespace)\nif err != nil {\nreturn nil, err\n}\n", v, deref(src), name), v, true, nil
	case model.Enum:
		v := b.temp("v")
		return fmt.Sprintf("%s, err := %s(%s)\nif err != nil {\nreturn nil, fmt.Errorf(\"%s: %%w\", err)\n}\n", v, f.enum.Func, arg(src), path), v, false, nil
	case model.Integer:
		if rt.WireString {
			return "", "strconv.FormatInt(" + arg(src) + ", 10)", false, nil
		}
	case model.String, model.Bool, model.Number:
		// A scalar needs no conversion.
	}
	return "", src, false, nil
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
