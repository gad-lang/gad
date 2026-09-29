package gad

import (
	"fmt"
	"sort"
)

// InterfaceSpreadFunc is the `InterfaceSpread(iface, spread…)` builtin an
// interface with `**Expr` body items compiles to: a copy of iface with the
// members each spread gives added — a dict (or a key-value array) of
//
//   - `fields`: a name to its type, a list of types, nil (untyped), or a spec
//     `(; types=[…], nullable=true, meta=(; …))`;
//   - `methods`: a name to a func header `<(…) <…>>`, or a list of them (its
//     overloads);
//   - `props`: a name to a type (or a list of them: `prop name T`, a getter and
//     a setter of it), a func header (the getter), or a spec `(; get=<…>,
//     set=[<…>], meta=(; …))`;
//   - `funcs`: a name to the context function (required to exist), or a spec
//     `(; fn=…, headers=[<(… @self)>])` (its signatures checked).
//
// A dict gives its members in the order of their names; a key-value array, in
// its own. A nil spread adds nothing.
func InterfaceSpreadFunc(c Call) (Object, error) {
	if err := c.Args.CheckMinLen(1); err != nil {
		return nil, err
	}
	src, ok := c.Args.Get(0).(*Interface)
	if !ok {
		return nil, NewArgumentTypeError("1st (interface)", "interface", c.Args.Get(0).Type().Name())
	}
	cp := *src
	cp.Fields = append([]*InterfaceField(nil), src.Fields...)
	cp.Props = append([]*InterfaceProp(nil), src.Props...)
	cp.Methods = append([]*InterfaceMethod(nil), src.Methods...)
	cp.ContextFuncs = append([]*InterfaceContextFunc(nil), src.ContextFuncs...)
	cp.flat, cp.flatErr, cp.flatBuilt = nil, nil, false

	for i := 1; i < c.Args.Length(); i++ {
		if err := cp.addSpread(c.Args.Get(i)); err != nil {
			return nil, err
		}
	}
	return &cp, nil
}

// addSpread adds the members one `**Expr` gives (see InterfaceSpreadFunc).
func (i *Interface) addSpread(spread Object) error {
	if spread == nil || spread == Nil {
		return nil
	}
	items := spreadItems(spread)
	if items == nil {
		return i.spreadErr("**Expr expects a dict of fields, methods, props and funcs, got %s", spread.Type().Name())
	}
	for _, kv := range items {
		var add func(name string, v Object) error
		switch kv.K.ToString() {
		case "fields":
			add = i.spreadField
		case "methods":
			add = i.spreadMethod
		case "props":
			add = i.spreadProp
		case "funcs":
			add = i.spreadFunc
		default:
			return i.spreadErr("**Expr has no %q: fields, methods, props or funcs", kv.K.ToString())
		}
		if kv.V == Nil {
			continue
		}
		members := spreadItems(kv.V)
		if members == nil {
			return i.spreadErr("**Expr %s: a dict, not %s", kv.K.ToString(), kv.V.Type().Name())
		}
		for _, m := range members {
			if err := add(m.K.ToString(), m.V); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Interface) spreadErr(format string, args ...any) error {
	return ErrType.NewError(fmt.Sprintf("interface %s: ", i.IName) + fmt.Sprintf(format, args...))
}

// spreadItems are the entries of a dict (by the order of the names) or a
// key-value array (in its own); nil for anything else.
func spreadItems(o Object) KeyValueArray {
	switch v := o.(type) {
	case KeyValueArray:
		return v
	case Dict:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(KeyValueArray, len(keys))
		for n, k := range keys {
			out[n] = &KeyValue{K: Str(k), V: v[k]}
		}
		return out
	}
	return nil
}

// spreadTypes reads a type, or a list of types.
func (i *Interface) spreadTypes(what string, v Object) (ObjectTypes, error) {
	switch t := v.(type) {
	case ObjectType:
		return ObjectTypes{t}, nil
	case Array:
		out := make(ObjectTypes, 0, len(t))
		for _, e := range t {
			ot, ok := e.(ObjectType)
			if !ok {
				return nil, i.spreadErr("%s: %s is not a type", what, e.Type().Name())
			}
			out = append(out, ot)
		}
		return out, nil
	}
	return nil, i.spreadErr("%s: a type or a list of types, not %s", what, v.Type().Name())
}

func (i *Interface) spreadField(name string, v Object) error {
	f := &InterfaceField{Iface: i, Name: name}
	switch t := v.(type) {
	case *NilType:
	case KeyValueArray:
		for _, kv := range t {
			switch kv.K.ToString() {
			case "types":
				types, err := i.spreadTypes("field "+name, kv.V)
				if err != nil {
					return err
				}
				f.Types = types
			case "nullable":
				f.Nullable = !kv.V.IsFalsy()
			case "meta":
				f.Meta, _ = kv.V.(KeyValueArray)
			default:
				return i.spreadErr("field %s: no %q in its spec (types, nullable, meta)", name, kv.K.ToString())
			}
		}
	default:
		types, err := i.spreadTypes("field "+name, v)
		if err != nil {
			return err
		}
		f.Types = types
	}
	i.Fields = append(i.Fields, f)
	return nil
}

// spreadHeaders reads a func header, or a list of them.
func (i *Interface) spreadHeaders(what string, v Object) ([]*FuncHeaderObject, error) {
	switch t := v.(type) {
	case *FuncHeaderObject:
		return []*FuncHeaderObject{t}, nil
	case Array:
		out := make([]*FuncHeaderObject, 0, len(t))
		for _, e := range t {
			h, ok := e.(*FuncHeaderObject)
			if !ok {
				return nil, i.spreadErr("%s: %s is not a func header", what, e.Type().Name())
			}
			out = append(out, h)
		}
		return out, nil
	}
	return nil, i.spreadErr("%s: a func header or a list of them, not %s", what, v.Type().Name())
}

func (i *Interface) spreadMethod(name string, v Object) error {
	headers, err := i.spreadHeaders("method "+name, v)
	if err != nil {
		return err
	}
	i.Methods = append(i.Methods, &InterfaceMethod{Iface: i, Name: name, Headers: headers})
	return nil
}

func (i *Interface) spreadProp(name string, v Object) error {
	p := &InterfaceProp{Iface: i, Name: name}
	switch t := v.(type) {
	case *FuncHeaderObject:
		p.Getter = t
	case KeyValueArray:
		for _, kv := range t {
			switch kv.K.ToString() {
			case "get":
				h, ok := kv.V.(*FuncHeaderObject)
				if !ok {
					return i.spreadErr("prop %s: get is a func header, not %s", name, kv.V.Type().Name())
				}
				p.Getter = h
			case "set":
				hs, err := i.spreadHeaders("prop "+name+" set", kv.V)
				if err != nil {
					return err
				}
				p.Setters = hs
			case "meta":
				p.Meta, _ = kv.V.(KeyValueArray)
			default:
				return i.spreadErr("prop %s: no %q in its spec (get, set, meta)", name, kv.K.ToString())
			}
		}
	default:
		// `prop name T`: a getter returning it, a setter taking it
		types, err := i.spreadTypes("prop "+name, v)
		if err != nil {
			return err
		}
		arr := make(Array, len(types))
		for n, t := range types {
			arr[n] = t
		}
		p.Getter = &FuncHeaderObject{FuncName: name, Return: Array{&TypedIdent{Name: "_", Types: arr}}}
		p.Setters = []*FuncHeaderObject{{FuncName: name, Params: Array{&TypedIdent{Name: "_", Types: arr}}}}
	}
	i.Props = append(i.Props, p)
	return nil
}

func (i *Interface) spreadFunc(name string, v Object) error {
	cf := &InterfaceContextFunc{FnName: name}
	switch t := v.(type) {
	case KeyValueArray:
		for _, kv := range t {
			switch kv.K.ToString() {
			case "fn":
				fn, ok := kv.V.(CallerObject)
				if !ok {
					return i.spreadErr("func %s: fn is a function, not %s", name, kv.V.Type().Name())
				}
				cf.Fn = fn
			case "headers":
				hs, err := i.spreadHeaders("func "+name, kv.V)
				if err != nil {
					return err
				}
				cf.Headers = hs
			default:
				return i.spreadErr("func %s: no %q in its spec (fn, headers)", name, kv.K.ToString())
			}
		}
	case CallerObject:
		cf.Fn = t
	default:
		return i.spreadErr("func %s: a function or a spec (; fn, headers), not %s", name, v.Type().Name())
	}
	i.ContextFuncs = append(i.ContextFuncs, cf)
	return nil
}
