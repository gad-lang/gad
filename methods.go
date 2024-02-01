package gad

import (
	"fmt"
	"sort"
	"strings"
)

type ObjectTypeNode struct {
	Type     ObjectType
	Children []*ObjectTypeNode
}

func (n *ObjectTypeNode) Append(o MultipleObjectTypes) {
	if len(o) > 0 {
		if len(o[0]) == 0 {
			var child = &ObjectTypeNode{}
			n.Children = append(n.Children, child)
			child.Append(o[1:])
		} else {
			for _, ot := range o[0] {
				var child = &ObjectTypeNode{
					Type: ot,
				}
				n.Children = append(n.Children, child)
				child.Append(o[1:])
			}
		}
	}
}

func (n *ObjectTypeNode) Walk(cb func(types ObjectTypes) any) any {
	return n.walk(nil, cb)
}

func (n *ObjectTypeNode) WalkE(cb func(types ObjectTypes) any) error {
	if v := n.walk(nil, cb); v != nil {
		return v.(error)
	}
	return nil
}

func (n *ObjectTypeNode) walk(path ObjectTypes, cb func(types ObjectTypes) any) (v any) {
	if len(n.Children) == 0 {
		return cb(path)
	}
	for _, child := range n.Children {
		if v = child.walk(append(path, child.Type), cb); v != nil {
			return
		}
	}
	return
}

type ObjectTypes []ObjectType

func (t ObjectTypes) String() string {
	var s = make([]string, len(t))
	for i, ot := range t {
		s[i] = ot.Name()
	}
	return strings.Join(s, ", ")
}

type MultipleObjectTypes []ObjectTypes

func (t MultipleObjectTypes) Tree() *ObjectTypeNode {
	var root ObjectTypeNode
	root.Append(t)
	return &root
}

type CallerMethod struct {
	Default bool
	CallerObject
	Types []ObjectType
	arg   *MethodArgType
	index int
}

func (o *CallerMethod) Caller() CallerObject {
	if o == nil {
		return nil
	}
	return o.CallerObject
}

func (o *CallerMethod) Remove() {
	for _, method := range o.arg.Methods[o.index+1:] {
		method.index--
	}
	o.arg.Methods = append(o.arg.Methods[:o.index], o.arg.Methods[o.index+1:]...)
	o.arg = nil
}

func (o *CallerMethod) String() string {
	var ts = make([]string, len(o.Types))
	for i := range ts {
		if o.Types[i] != nil {
			ts[i] = " " + o.Types[i].Name()
		}
	}

	return "(" + strings.Join(ts, ", ") + ") => " + o.CallerObject.ToString()
}

type CallerObjectWithMethods struct {
	CallerObject
	Methods    MethodArgType
	registered bool
}

func NewCallerObjectWithMethods(callerObject CallerObject) *CallerObjectWithMethods {
	return &CallerObjectWithMethods{CallerObject: callerObject}
}

func (o *CallerObjectWithMethods) HasCallerMethods() bool {
	if o.registered {
		return !o.Methods.IsZero()
	}
	return false
}

func (o *CallerObjectWithMethods) RegisterDefaultWithTypes(types MultipleObjectTypes) *CallerObjectWithMethods {
	o.registered = true
	o.Methods.Add(types, &CallerMethod{
		Default:      true,
		CallerObject: o.CallerObject,
	}, false)
	return o
}

func (o *CallerObjectWithMethods) AddCallerMethod(vm *VM, types MultipleObjectTypes, handler CallerObject, override bool) error {
	if !o.registered {
		o.registered = true
		if cot, _ := o.CallerObject.(CallerObjectWithParamTypes); cot != nil {
			types, err := o.CallerObject.(CallerObjectWithParamTypes).ParamTypes(vm)
			if err != nil {
				return err
			}
			o.RegisterDefaultWithTypes(types)
		}
	}

	return o.Methods.Add(types, &CallerMethod{
		CallerObject: handler,
	}, override)
}

func (o *CallerObjectWithMethods) ToString() string {
	var (
		s strings.Builder
		i int
	)

	if !o.registered {
		return o.CallerObject.ToString()
	}

	o.MethodWalkSorted(func(m *CallerMethod) any {
		if !m.Default {
			s.WriteString(fmt.Sprintf("  %d. ", i+1))
			s.WriteString(m.CallerObject.ToString())
			s.WriteByte('\n')
			i++
		}
		return nil
	})

	return o.CallerObject.ToString() + strings.TrimRight(fmt.Sprintf(" with %d methods:\n", i)+s.String(), "\n")
}

func (o *CallerObjectWithMethods) String() string {
	return o.ToString()
}

func (o *CallerObjectWithMethods) Caller() CallerObject {
	return o.CallerObject
}

func (o *CallerObjectWithMethods) Call(c Call) (Object, error) {
	caller, validate := o.CallerOf(c.Args)
	c.SafeArgs = !validate
	return YieldCall(caller, &c), nil
}

func (o *CallerObjectWithMethods) CallerOf(args Args) (CallerObject, bool) {
	if !o.registered {
		if cof, _ := o.CallerObject.(CanCallerObjectTypesValidation); cof != nil {
			return o.CallerObject, cof.CanValidateParamTypes()
		}
		return o.CallerObject, false
	}
	var types []ObjectType
	args.Walk(func(i int, arg Object) any {
		if t, ok := arg.(ObjectType); ok {
			types = append(types, t)
		} else {
			types = append(types, arg.Type())
		}
		return nil
	})
	return o.CallerOfTypes(types)
}

func (o *CallerObjectWithMethods) GetMethod(types []ObjectType) (co CallerObject) {
	return o.Methods.GetMethod(types).Caller()
}

func (o *CallerObjectWithMethods) CallerOfTypes(types []ObjectType) (co CallerObject, validate bool) {
	if method := o.Methods.GetMethod(types); method != nil {
		return method.CallerObject, false
	}
	if cof, _ := o.CallerObject.(CanCallerObjectTypesValidation); cof != nil {
		validate = cof.CanValidateParamTypes()
	}
	return o.CallerObject, validate
}

func (o *CallerObjectWithMethods) CallerMethods() *MethodArgType {
	return &o.Methods
}

func (o *CallerObjectWithMethods) MethodWalk(cb func(m *CallerMethod) any) (v any) {
	if o.registered {
		return o.Methods.Walk(cb)
	}
	return cb(&CallerMethod{
		CallerObject: o.CallerObject,
	})
}

func (o *CallerObjectWithMethods) MethodWalkSorted(cb func(m *CallerMethod) any) (v any) {
	if o.registered {
		return o.Methods.WalkSorted(cb)
	}
	return cb(&CallerMethod{
		CallerObject: o.CallerObject,
	})
}

func (o *CallerObjectWithMethods) Equal(right Object) bool {
	if cowm, _ := right.(*CallerObjectWithMethods); cowm != nil {
		right = cowm.CallerObject
	}
	return o.CallerObject.Equal(right)
}

type MethodDefinition struct {
	Args    []ObjectType
	Handler CallerObject
}

type MethodArgType struct {
	Type    ObjectType
	Methods []*CallerMethod
	Next    Methods
}

func (at *MethodArgType) Walk(cb func(m *CallerMethod) any) (v any) {
	for _, method := range at.Methods {
		if v = cb(method); v != nil {
			return
		}
	}
	return at.Next.Walk(cb)
}

func (at *MethodArgType) WalkSorted(cb func(m *CallerMethod) any) (v any) {
	for _, method := range at.Methods {
		if v = cb(method); v != nil {
			return
		}
	}
	return at.Next.WalkSorted(cb)
}

func (at *MethodArgType) Add(types MultipleObjectTypes, m *CallerMethod, override bool) error {
	return types.Tree().WalkE(func(types ObjectTypes) any {
		return at.add(nil, types, m, override)
	})
}

func (at *MethodArgType) add(pth, types ObjectTypes, m *CallerMethod, override bool) error {
	if len(types) == 0 {
		if len(at.Methods) > 0 && !override {
			return ErrMethodDuplication.NewError(m.String())
		}
		m2 := *m
		m2.Types = pth
		m2.index = len(at.Methods)
		at.Methods = append(at.Methods, &m2)
		return nil
	}

	if at.Next == nil {
		at.Next = map[ObjectType]*MethodArgType{}
	}

	return at.Next.Add(pth, types, m, override)
}

func (at *MethodArgType) GetMethod(types []ObjectType) *CallerMethod {
	if len(types) == 0 {
		if len(at.Methods) > 0 {
			return at.Methods[len(at.Methods)-1]
		}
		return nil
	}
	return at.Next.GetMethod(types)
}

func (at *MethodArgType) IsZero() (ok bool) {
	ok = true
	at.Walk(func(m *CallerMethod) any {
		if m.Default {
			return nil
		}
		ok = false
		return ok
	})
	return
}

type Methods map[ObjectType]*MethodArgType

func (args Methods) IsZero() (ok bool) {
	ok = true
	for _, v := range args {
		if len(v.Methods) > 0 {
			return false
		}
		if ok = v.Next.IsZero(); !ok {
			return
		}
	}
	return
}

func (args Methods) Walk(cb func(m *CallerMethod) any) (rv any) {
	for _, v := range args {
		for _, method := range v.Methods {
			if rv = cb(method); rv != nil {
				return
			}
		}
		if v.Next != nil {
			if rv = v.Next.Walk(cb); rv != nil {
				return
			}
		}
	}
	return
}

func (args Methods) WalkSorted(cb func(m *CallerMethod) any) (rv any) {
	type kv struct {
		k string
		v ObjectType
	}
	var (
		l      = len(args)
		values = make([]kv, l)
		i      int
	)

	for key := range args {
		if key == nil {
			values[i] = kv{"", nil}
		} else {
			values[i] = kv{key.Name(), key}
		}
		i++
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].k < values[j].k
	})

	for _, kv := range values {
		v := args[kv.v]
		for _, method := range v.Methods {
			if rv = cb(method); rv != nil {
				return
			}
		}
		if v.Next != nil {
			if rv = v.Next.WalkSorted(cb); rv != nil {
				return
			}
		}
	}
	return
}

func (args Methods) Add(pth, types ObjectTypes, cm *CallerMethod, override bool) (err error) {
	cur, ok := args[types[0]]
	if !ok {
		cur = &MethodArgType{
			Type: types[0],
			Next: map[ObjectType]*MethodArgType{},
		}
		args[types[0]] = cur
	}

	return cur.add(append(pth, types[0]), types[1:], cm, override)
}

func (args Methods) GetMethod(types []ObjectType) (cm *CallerMethod) {
	var at *MethodArgType

	for i := len(types); i > 0; i-- {
		at = args[types[0]]
		if at == nil {
			if at = args[nil]; at == nil {
				return nil
			}
		}
		args = at.Next
		types = types[1:]
	}

	if at != nil && at.Methods != nil {
		cm = at.Methods[len(at.Methods)-1]
	}
	return
}

func NewTypedFunction(fn *Function, types MultipleObjectTypes) *CallerObjectWithMethods {
	return NewCallerObjectWithMethods(fn).RegisterDefaultWithTypes(types)
}
