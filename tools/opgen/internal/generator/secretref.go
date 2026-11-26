package generator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/coralogix/coralogix-management-sdk/iac/model"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/overrides"
)

// secretField lets the CR read the value of a string field, or the values of a map of strings, from
// a Secret (secretRef). The literal field stays and is optional. A string gets the sibling
// <name>SecretKeyRef; a map gets <name>FromSecret. When the API needs the string, the CR sets
// exactly one of the two fields.
func (b *builder) secretField(f *crField, line overrides.Line, t *model.Type) (*crField, error) {
	invalid := func(msg string) error {
		return &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".secretRef", msg: msg}
	}
	switch {
	case f.ref != "" || f.joinedBy != "" || f.intNames != nil || f.sendNullAsEmpty:
		return nil, invalid("secretRef cannot change a reference, a joined list, integer names, or sendNullAsEmpty")
	case t.Kind == model.String && t.Format == "" && len(t.Wrappers) == 0 && f.enum == nil:
		f.secret = &secretSource{sibling: f.JSON + "SecretKeyRef", required: !f.Optional}
		if !f.Optional {
			f.Optional = true
			f.GoType = "*" + f.GoType
			f.Markers = append([]string{"+optional"}, f.Markers...)
		}
		f.Doc = append(f.Doc, fmt.Sprintf("Use %s when the value is a credential.", f.secret.sibling))
	case t.Kind == model.Map && f.GoType == "map[string]string":
		if !f.Optional {
			return nil, invalid("secretRef needs an optional map: a key can come from a Secret instead")
		}
		f.secret = &secretSource{sibling: f.JSON + "FromSecret", values: true}
		f.Doc = append(f.Doc, fmt.Sprintf("Use %s for the values that are credentials.", f.secret.sibling))
	default:
		return nil, invalid("secretRef needs a string or a map of strings with no limits on its values")
	}
	b.used[line] = true
	b.secretRefs = true
	return f, nil
}

// secretSibling is the CR field that names the Secret of a secretRef field.
func secretSibling(f *crField) *crField {
	s := f.secret
	sibling := &crField{API: s.sibling, JSON: s.sibling, GoName: goName(s.sibling), Optional: true, Markers: []string{"+optional"}, synthetic: true, secretOf: f}
	if s.values {
		sibling.GoType = "map[string]corev1.SecretKeySelector"
		sibling.Doc = []string{fmt.Sprintf("%s maps a key of %s to a key of a Secret in the namespace of the custom resource. A key can have only one source.", sibling.GoName, f.JSON)}
		return sibling
	}
	sibling.GoType = "*corev1.SecretKeySelector"
	sibling.Doc = []string{fmt.Sprintf("%s reads %s from a key of a Secret in the namespace of the custom resource.", sibling.GoName, f.JSON)}
	return sibling
}

// secretDefault rejects a secretRef field with a Kubernetes default, from the contract or from a
// validators line. Kubernetes would fill the literal before it checks the rule, so a CR that names
// only the Secret would have both sources.
func secretDefault(f *crField, path string) error {
	if f.secret == nil || !slices.ContainsFunc(f.Markers, func(m string) bool { return strings.HasPrefix(m, "+kubebuilder:default") }) {
		return nil
	}
	return &shapeError{code: codeOverrideInvalid, path: path, msg: "secretRef cannot read a field with a default: Kubernetes would set the literal next to the Secret reference"}
}

// secretRules adds the CEL rule of each secretRef field of t.
func secretRules(t *crType) {
	for _, f := range t.Fields {
		if f.secret != nil {
			t.Markers = append(t.Markers, secretRule(f))
		}
	}
}

func secretRule(f *crField) string {
	s := f.secret
	switch {
	case s.values:
		rule := fmt.Sprintf("!has(self.%[1]s) || !has(self.%[2]s) || self.%[2]s.all(k, !(k in self.%[1]s))", f.JSON, s.sibling)
		return celMarker(rule, "A key of "+f.JSON+" can have only one source")
	case s.required:
		return celMarker(fmt.Sprintf("has(self.%s) != has(self.%s)", f.JSON, s.sibling), fmt.Sprintf("Exactly one of %s or %s must be set", f.JSON, s.sibling))
	}
	return celMarker(fmt.Sprintf("!(has(self.%s) && has(self.%s))", f.JSON, s.sibling), fmt.Sprintf("Only one of %s or %s can be set", f.JSON, s.sibling))
}

// armFields moves each top-level field with an inArms line from the spec into the listed oneOf arms.
// Every arm gets its own copy of the field, with its Secret sibling and rule. The conversion of the
// spec sends the value of the arm that the CR sets.
func (b *builder) armFields(r *model.Resource, spec *crType) error {
	inGroup := map[string]bool{}
	for _, g := range r.Groups {
		for _, arm := range g.Arms {
			inGroup[arm] = true
		}
	}
	for _, rf := range r.Fields {
		arms := b.file.Field(r.Name, rf.Name).InArms
		if len(arms) == 0 {
			continue
		}
		line := overrides.Line{Component: r.Name, Field: rf.Name}
		invalid := func(msg string, args ...any) error {
			return &shapeError{code: codeOverrideInvalid, path: overrides.FileName + ":" + line.String() + ".inArms", msg: fmt.Sprintf(msg, args...)}
		}
		i := slices.IndexFunc(spec.Fields, func(f *crField) bool { return f.API == rf.Name && f.secretOf == nil && !f.synthetic })
		switch {
		case i < 0:
			return invalid("the Create request has no field %s", rf.Name)
		case rf.Update == nil:
			return invalid("the Update request has no field %s; the arm value would reach only Create", rf.Name)
		case spec.Fields[i].typ.Kind != model.String || spec.Fields[i].object != nil:
			return invalid("inArms needs a string field")
		case inGroup[rf.Name]:
			return invalid("a oneOf arm cannot be shown in other arms")
		}
		field := spec.Fields[i]
		spec.Fields = slices.DeleteFunc(spec.Fields, func(f *crField) bool { return f == field || f.secretOf == field })
		for _, arm := range arms {
			j := slices.IndexFunc(spec.Fields, func(f *crField) bool { return f.API == arm && !f.synthetic })
			if j < 0 || !inGroup[arm] || spec.Fields[j].object == nil || strings.HasPrefix(spec.Fields[j].GoType, "[]") || strings.HasPrefix(spec.Fields[j].GoType, "map[") {
				return invalid("%s is not an object arm of a oneOf of %s", arm, r.Name)
			}
			holder := spec.Fields[j].object
			if slices.ContainsFunc(holder.Fields, func(f *crField) bool { return f.JSON == field.JSON || field.secret != nil && f.JSON == field.secret.sibling }) {
				return invalid("the arm %s has a field named %s already", arm, field.JSON)
			}
			copied := *field
			copied.armOf = rf.Name
			copied.Doc = slices.Clone(field.Doc)
			copied.Markers = slices.Clone(field.Markers)
			holder.Fields = append(holder.Fields, &copied)
			if copied.secret != nil {
				holder.Fields = append(holder.Fields, secretSibling(&copied))
				holder.Markers = append(holder.Markers, secretRule(&copied))
			}
		}
		b.used[line] = true
	}
	return nil
}

// apiFieldCount counts the fields of t that hold an API property of its own component: not the
// copies of an inArms line, and not synthetic fields.
func apiFieldCount(t *crType) int {
	n := 0
	for _, f := range t.Fields {
		if f.armOf == "" && !f.synthetic {
			n++
		}
	}
	return n
}

// secretNamesCode writes SecretNames of the kind, and addSecretNames of each struct that holds a
// secretRef field directly or below it. The controller indexes the CRs by these names, so a change
// of a Secret reconciles the CRs that read it.
func secretNamesCode(kind string, spec *crType) string {
	holds := map[*crType]bool{}
	var visit func(t *crType, seen map[*crType]bool) bool
	visit = func(t *crType, seen map[*crType]bool) bool {
		if done, ok := holds[t]; ok {
			return done
		}
		if seen[t] {
			return false
		}
		seen[t] = true
		found := false
		for _, f := range t.Fields {
			if f.secretOf != nil || (f.object != nil && visit(f.object, seen)) {
				found = true
			}
		}
		holds[t] = found
		return found
	}
	visit(spec, map[*crType]bool{})
	var out strings.Builder
	fmt.Fprintf(&out, `
// SecretNames returns the names of the Secrets that the %[1]s reads, sorted and without repeats.
func (in *%[1]s) SecretNames() []string {
	names := map[string]bool{}
	in.Spec.addSecretNames(names)
	return slices.Sorted(maps.Keys(names))
}
`, kind)
	var order []*crType
	var collect func(t *crType)
	collect = func(t *crType) {
		if !holds[t] || slices.Contains(order, t) {
			return
		}
		order = append(order, t)
		for _, f := range t.Fields {
			if f.object != nil {
				collect(f.object)
			}
		}
	}
	collect(spec)
	for _, t := range order {
		fmt.Fprintf(&out, "\n// addSecretNames adds the names of the Secrets that the %s reads.\nfunc (in *%s) addSecretNames(names map[string]bool) {\n", t.Name, t.Name)
		for _, f := range t.Fields {
			src := "in." + f.GoName
			switch {
			case f.secretOf != nil && f.secretOf.secret.values:
				fmt.Fprintf(&out, "for _, ref := range %s {\nnames[ref.Name] = true\n}\n", src)
			case f.secretOf != nil:
				fmt.Fprintf(&out, "if %s != nil {\nnames[%s.Name] = true\n}\n", src, src)
			case f.object != nil && holds[f.object]:
				elem := f.GoType
				var loop string
				switch {
				case strings.HasPrefix(elem, "[]"):
					elem, loop = strings.TrimPrefix(elem, "[]"), "for i := range "+src+" {\n"
					src += "[i]"
				case strings.HasPrefix(elem, "map["):
					elem, loop = elem[strings.Index(elem, "]")+1:], "for key := range "+src+" {\n"
					src += "[key]"
				}
				call := src + ".addSecretNames(names)\n"
				if strings.HasPrefix(elem, "*") {
					call = "if " + src + " != nil {\n" + call + "}\n"
				} else if strings.HasPrefix(f.GoType, "map[") {
					// A map value is not addressable: copy it.
					call = "item := " + src + "\nitem.addSecretNames(names)\n"
				}
				if loop != "" {
					call = loop + call + "}\n"
				}
				out.WriteString(call)
			}
		}
		out.WriteString("}\n")
	}
	return out.String()
}

// secretValue writes the code that sets target from a secretRef field: the literal, or the value of
// the Secret that the sibling names. The API package resolves both, and rejects a value with two
// sources.
func (b *builder) secretValue(f *crField, want, target, path, src string) (string, error) {
	sibling := strings.TrimSuffix(src, f.GoName) + goName(f.secret.sibling)
	v := b.temp("secret")
	if f.secret.values {
		if strings.TrimPrefix(want, "*") != "map[string]string" {
			return "", unsupported(path, "secretRef on a map needs map[string]string in the SDK, got %s", want)
		}
		return fmt.Sprintf("%s, err := ResolveSecretValues(ctx, namespace, %q, %s, %s)\nif err != nil {\nreturn nil, err\n}\nif len(%s) != 0 {\n%s}\n",
			v, path, src, sibling, v, b.set(target, want, v, false)), nil
	}
	if strings.TrimPrefix(want, "*") != "string" {
		return "", unsupported(path, "secretRef needs a string in the SDK, got %s", want)
	}
	return fmt.Sprintf("%s, err := ResolveSecretValue(ctx, namespace, %q, %s, %s)\nif err != nil {\nreturn nil, err\n}\nif %s != nil {\n%s}\n",
		v, path, src, sibling, v, b.set(target, want, v, true)), nil
}

// armValue writes the code that sets a top-level API field of an inArms line from the arm that the
// CR sets. An arm that is not listed sends no value.
func (b *builder) armValue(spec *crType, sdkType string, rf *model.Field, arms []string, path string) (string, error) {
	var out strings.Builder
	for _, arm := range arms {
		holder, src, guard := sourceField(spec, arm)
		if holder == nil || holder.object == nil {
			return "", unsupported(path+"."+arm, "the arm of an inArms line is not in the custom resource")
		}
		i := slices.IndexFunc(holder.object.Fields, func(f *crField) bool { return f.armOf == rf.Name })
		if i < 0 {
			return "", unsupported(path+"."+arm, "the arm has no copy of %s", rf.Name)
		}
		copied := holder.object.Fields[i]
		code, err := b.assignFrom(copied, rf.Type, sdkType, crPath(spec, path, holder)+"."+copied.JSON, src+"."+copied.GoName)
		if err != nil {
			return "", err
		}
		code = "if " + src + " != nil {\n" + code + "}\n"
		if guard != "" {
			code = "if " + guard + " != nil {\n" + code + "}\n"
		}
		out.WriteString(code)
	}
	return out.String(), nil
}

// crPath returns the CR path of a field of the spec, also of an arm inside a group of the spec.
func crPath(spec *crType, path string, field *crField) string {
	for _, f := range spec.Fields {
		if f == field {
			return path + "." + f.JSON
		}
		if f.group && slices.Contains(f.object.Fields, field) {
			return path + "." + f.JSON + "." + field.JSON
		}
	}
	return path + "." + field.JSON
}
