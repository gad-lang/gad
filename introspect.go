package gad

import "sort"

// Member is a completion member of an Object: a dict key, a class-instance
// field/property/method, a class static member, or a module export.
type Member struct {
	Name string // the member name
	Kind string // field | property | method | key | export | function
	Doc  string // documentation, when the member value carries it
}

// Members lists the members of obj for auto-completion. Names are precise —
// taken from the live value — so they cover dicts (incl. nested), class
// instances (fields, properties, methods, inherited), class static members and
// module exports. Doc is filled when the member value itself is documented (e.g.
// a builtin function); class member docs live in source and are attached by the
// caller from the AST.
func Members(obj Object) []Member {
	switch v := obj.(type) {
	case Dict:
		return dictMembers(v, "key")
	case *ClassInstance:
		return instanceMembers(v)
	case *Class:
		return classMembers(v)
	case *Module:
		if v.Data != nil {
			return dictMembers(v.Data.ToDict(), "export")
		}
	}
	// Generic fallback: any value that can list its keys (module data, proxies…).
	if d, ok := obj.(ToDictConverter); ok {
		return dictMembers(d.ToDict(), "export")
	}
	if kg, ok := obj.(KeysGetter); ok {
		return keyNameMembers(kg.Keys(), "key")
	}
	return nil
}

// dictMembers lists a dict's entries, labelling function-valued entries and
// pulling a builtin function's own documentation.
func dictMembers(d Dict, kind string) []Member {
	out := make([]Member, 0, len(d))
	for name, val := range d {
		m := Member{Name: name, Kind: kind}
		switch f := val.(type) {
		case *BuiltinFunction:
			m.Kind, m.Doc = "function", f.Doc()
		case *Function, *CompiledFunction:
			m.Kind = "function"
		}
		out = append(out, m)
	}
	sortMembers(out)
	return out
}

// instanceMembers lists a class instance's fields, properties and methods,
// including those inherited from parent classes.
func instanceMembers(o *ClassInstance) []Member {
	seen := map[string]bool{}
	var out []Member
	add := func(name, kind string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, Member{Name: name, Kind: kind})
	}
	// The instance's own fields carry the concrete values.
	for name := range o.Fields() {
		add(name, "field")
	}
	// Static shape (fields/properties/methods) from the class and its parents.
	if cls, ok := o.Type().(*Class); ok {
		for _, m := range classMembers(cls) {
			add(m.Name, m.Kind)
		}
	}
	for _, parent := range o.Parents() {
		if pi, ok := parent.(*ClassInstance); ok {
			for _, m := range instanceMembers(pi) {
				add(m.Name, m.Kind)
			}
		}
	}
	sortMembers(out)
	return out
}

// classMembers lists a class's declared fields, properties and methods.
func classMembers(t *Class) []Member {
	var out []Member
	for name := range t.Fields() {
		out = append(out, Member{Name: name, Kind: "field"})
	}
	for name := range t.Properties() {
		out = append(out, Member{Name: name, Kind: "property"})
	}
	for name := range t.Methods() {
		out = append(out, Member{Name: name, Kind: "method"})
	}
	sortMembers(out)
	return out
}

func keyNameMembers(keys Array, kind string) []Member {
	out := make([]Member, 0, len(keys))
	for _, k := range keys {
		out = append(out, Member{Name: k.ToString(), Kind: kind})
	}
	sortMembers(out)
	return out
}

func sortMembers(m []Member) {
	sort.Slice(m, func(i, j int) bool { return m[i].Name < m[j].Name })
}
