package gad

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/zeroer"
	"github.com/xlab/treeprint"
)

type IndexMethodAdder interface {
	AddMethodIndex(c Call) (ret Object, err error)
}

type IndexMethodGetter interface {
	GetIndexMethod(vm *VM, index Object) (ret Object, err error)
}

type ObjectTypeNode struct {
	Var  bool
	Type ObjectType
	VarChildren,
	Children []*ObjectTypeNode
}

func (n *ObjectTypeNode) Append(o ParamsTypes) (err error) {
	if len(o) > 0 {
		if o[0].IsZero() {
			var child = &ObjectTypeNode{}
			n.Children = append(n.Children, child)
			return child.Append(o[1:])
		} else {
			_, isvar := o[0].(VarParamTypes)

			for _, ot := range o[0].Items() {
				var child = &ObjectTypeNode{
					Var:  isvar,
					Type: ot,
				}

				if isvar {
					n.VarChildren = append(n.VarChildren, child)
				} else {
					n.Children = append(n.Children, child)
				}
				if !isvar {
					if err = child.Append(o[1:]); err != nil {
						return
					}
				} else if len(o) > 1 {
					return errors.New("more than one type for variadic parameter")
				}
			}
		}
	}
	return
}

func (n *ObjectTypeNode) Walk(cb func(types ObjectTypeArray) any) any {
	return n.walk(nil, cb)
}

func (n *ObjectTypeNode) WalkE(cb func(types ObjectTypeArray) any) error {
	if v := n.walk(nil, cb); v != nil {
		return v.(error)
	}
	return nil
}

func (n *ObjectTypeNode) walk(path ObjectTypeArray, cb func(types ObjectTypeArray) any) (v any) {
	if len(n.Children) == 0 {
		if len(n.VarChildren) > 0 {
			for _, child := range n.VarChildren {
				dot := path
				if child.Var {
					dot = append(dot, &VarObjectType{child.Type})
				} else {
					dot = append(dot, child.Type)
				}
				if v = child.walk(dot, cb); v != nil {
					return
				}
			}
			return
		} else {
			return cb(path)
		}
	}
	for _, child := range n.Children {
		dot := path
		if child.Var {
			dot = append(dot, &VarObjectType{child.Type})
		} else {
			dot = append(dot, child.Type)
		}
		if v = child.walk(dot, cb); v != nil {
			return
		}
	}
	return
}

type ObjectTypes []ObjectType

var _ ParamTypes = ObjectTypes{}

func (t ObjectTypes) Multi() (m ParamsTypes) {
	m = make(ParamsTypes, len(t))
	for i := range t {
		m[i] = t[i : i+1]
	}
	return
}

func (t ObjectTypes) String() string {
	var s = make([]string, len(t))
	for i, ot := range t {
		s[i] = ot.Name()
	}
	return strings.Join(s, "|")
}

func (t ObjectTypes) Items() ObjectTypes {
	return t
}

func (t ObjectTypes) IsZero() bool {
	return len(t) == 0
}

func (t ObjectTypes) Len() int {
	return len(t)
}

func (t ObjectTypes) Get(i int) ObjectType {
	return t[i]
}

func (t ObjectTypes) Last() ObjectType {
	return t[len(t)-1]
}

func (t ObjectTypes) HasVar() (ok bool) {
	if len(t) > 0 {
		_, ok = t.Last().(*VarObjectType)
	}
	return
}

func (t ObjectTypes) VarSplit() (nonVar ObjectTypes, varType ObjectType) {
	nonVar = t
	if len(t) > 0 {
		last := t[len(t)-1]
		if vart, _ := last.(*VarObjectType); vart != nil {
			varType = vart.ObjectType
			nonVar = nonVar[:len(nonVar)-1]
		}
	}
	return
}

func (t ObjectTypes) Var() (_ ObjectType) {
	if len(t) > 0 {
		if v, _ := t.Last().(*VarObjectType); v != nil {
			return v.ObjectType
		}
	}
	return
}

type VarParamTypes []ObjectType

type VarObjectType struct {
	ObjectType
}

func (v *VarObjectType) String() string {
	return "*" + v.ObjectType.String()
}

func (v *VarObjectType) Name() string {
	return "*" + v.ObjectType.Name()
}

func (v *VarObjectType) FullName() string {
	return "*" + v.ObjectType.FullName()
}

var _ ParamTypes = VarParamTypes{}

func (t VarParamTypes) String() string {
	return "*" + ObjectTypes(t).String()
}

func (t VarParamTypes) Items() ObjectTypes {
	return ObjectTypes(t)
}

func (t VarParamTypes) IsZero() bool {
	return len(t) == 0
}

func (t VarParamTypes) Len() int {
	return len(t)
}

func (t VarParamTypes) Get(i int) ObjectType {
	return t[i]
}

type ParamTypes interface {
	fmt.Stringer
	zeroer.Zeroer
	Items() ObjectTypes
	Len() int
	Get(int) ObjectType
}

type ParamsTypes []ParamTypes

func (t ParamsTypes) String() string {
	s := make([]string, len(t))
	for i, types := range t {
		s[i] = types.String()
	}
	return "(" + strings.Join(s, ", ") + ")"
}

func (t ParamsTypes) Tree() (r *ObjectTypeNode, err error) {
	var root ObjectTypeNode
	err = root.Append(t)
	r = &root
	return
}

type CallerMethod struct {
	target Object
	CallerObject
	ToStringDetailFunc func(m *CallerMethod) string
}

func NewCallerMethod(target Object, callerObject CallerObject) *CallerMethod {
	return &CallerMethod{target: target, CallerObject: callerObject}
}

func (o *CallerMethod) Caller() CallerObject {
	if o == nil {
		return nil
	}
	return o.CallerObject
}

func (o *CallerMethod) Target() Object {
	return o.target
}

func (o *CallerMethod) String() string {
	return o.StringTarget(true)
}

func (o *CallerMethod) StringTarget(targets bool) string {
	var target string
	if targets && o.target != nil {
		if t, _ := o.target.(ObjectType); t != nil {
			target = "[target " + t.String() + "]"
		} else {
			target = "[target " + repr.QuoteTyped(o.target.Type().Name(), o.target.ToString()) + "]"
		}
	}

	var detail string
	if o.ToStringDetailFunc != nil {
		detail = o.ToStringDetailFunc(o)
		if len(detail) > 0 {
			detail = " " + detail
		}
	}

	var sep string
	if len(target) > 0 || len(detail) > 0 {
		sep = " 🠆 "
	}

	return fmt.Sprintf("%s%s%s%s", o.CallerObject.ToString(), sep, target, detail)
}

func (o *CallerMethod) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "target":
		return o.target, nil
	case "caller":
		return o.CallerObject, nil
	default:
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}

type TypedCallerMethods []*TypedCallerMethod

var (
	_ Object       = (*TypedCallerMethod)(nil)
	_ CallerObject = (*TypedCallerMethod)(nil)
	_ IndexGetter  = (*TypedCallerMethod)(nil)
)

type TypedCallerMethod struct {
	*CallerMethod
	types ObjectTypeArray
	isVar bool
}

func (o *TypedCallerMethod) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "caller":
		return o.CallerMethod, nil
	case "isVar":
		return Bool(o.isVar), nil
	case "types":
		return o.types, nil
	default:
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}

func (o *TypedCallerMethod) Types() ObjectTypeArray {
	return o.types
}

func (o *TypedCallerMethod) IsVar() bool {
	return o.isVar
}

func (o *TypedCallerMethod) String() string {
	return o.StringTarget(true)
}

func (o *TypedCallerMethod) StringTarget(target bool) string {
	if o == nil {
		return ""
	}
	var types = make([]string, len(o.types))
	for i := range types {
		types[i] = o.types[i].Name()
	}

	return fmt.Sprintf("⨍(%s) 🠆 %s", strings.Join(types, ", "), o.CallerMethod.StringTarget(target))
}

func (o *TypedCallerMethod) ToString() string {
	return o.String()
}

func (o *TypedCallerMethod) Print(state *PrinterState) error {
	targets, _ := state.context.Value(typedCallerMethodContextKeyNoTarget).(bool)
	return state.WriteString(o.StringTarget(!targets))
}

type typedCallerMethodContextKey uint8

const (
	typedCallerMethodContextKeyNoTarget typedCallerMethodContextKey = iota + 1
	typedCallerMethodContextKeyNoMethods
)

type MethodDefinition struct {
	Args    []ObjectType
	Handler CallerObject
}

type MethodArgType struct {
	parent  *MethodArgType
	Type    ObjectType
	Method  *TypedCallerMethod
	Var     bool
	Next    Methods
	NextVar Methods
}

func (at *MethodArgType) Parents() (parents []*MethodArgType) {
	at = at.parent
	for at != nil {
		parents = append(parents, at)
		at = at.parent
	}
	slices.Reverse(parents)
	return
}

func (at *MethodArgType) Path() (path []*MethodArgType) {
	path = append(path, at)
	at = at.parent
	for at != nil {
		path = append(path, at)
		at = at.parent
	}
	slices.Reverse(path)
	return
}

func (at MethodArgType) Copy() *MethodArgType {
	if at.Next != nil {
		at.Next = at.Next.Copy()
	}
	if (at.NextVar) != nil {
		at.NextVar = at.Next.Copy()
	}
	return &at
}

func (at *MethodArgType) Walk(cb func(m *TypedCallerMethod) any) (v any) {
	if at.Method != nil {
		if v = cb(at.Method); v != nil {
			return
		}
	}

	if at.Next != nil {
		if v = at.Next.Walk(cb); v != nil {
			return
		}
	}

	if at.NextVar != nil {
		if v = at.NextVar.Walk(cb); v != nil {
			return
		}
	}
	return
}

func (at *MethodArgType) WalkSorted(cb func(m *TypedCallerMethod) any) (v any) {
	if at.Method != nil {
		if v = cb(at.Method); v != nil {
			return
		}
	}
	if at.Next != nil {
		if v = at.Next.WalkSorted(cb); v != nil {
			return
		}
	}
	if at.NextVar != nil {
		if v = at.NextVar.WalkSorted(cb); v != nil {
			return
		}
	}

	return
}

func (at *MethodArgType) Add(types ParamsTypes, m *CallerMethod, override bool, onAdd func(tcm *TypedCallerMethod) error) (err error) {
	if len(types) == 0 {
		return at.add(nil, m, override, onAdd)
	}

	var root *ObjectTypeNode
	if root, err = types.Tree(); err != nil {
		return
	}

	err = root.WalkE(func(types ObjectTypeArray) any {
		return at.add(types, m, override, onAdd)
	})
	return
}

func (at *MethodArgType) add(types ObjectTypeArray, m *CallerMethod, override bool, onAdd func(tcm *TypedCallerMethod) error) (err error) {
	var (
		getOrAdd = func(dst *Methods, t ObjectType) (cur *MethodArgType, added bool) {
			if *dst == nil {
				*dst = make(map[ObjectType]*MethodArgType)
			}

			if bt, _ := t.(*BuiltinObjType); bt != nil {
				t = bt.TypeKey()
			}

			cur = (*dst)[t]
			if cur == nil {
				cur = &MethodArgType{
					parent: at,
					Type:   t,
				}
				(*dst)[t] = cur
				added = true
			}
			return
		}
		set = func(cur *MethodArgType, added bool, types ObjectTypeArray, raise bool) {
			if added || override || cur.Method == nil {
				cur.Method = &TypedCallerMethod{
					CallerMethod: m,
					types:        types,
				}

				if onAdd != nil {
					if err = onAdd(cur.Method); err != nil {
						return
					}
				}
			} else if raise {
				err = ErrMethodDuplication.NewErrorf("params %s: %s. Current method is %s", types.String(), m.StringTarget(false), cur.Method.CallerMethod.StringTarget(false))
			}
		}
	)

	if len(types) == 0 {
		set(at, at.Method == nil, nil, true)
		return
	}

	nonVarT, varT := types.VarSplit()

	if len(nonVarT) > 0 {
		for _, t := range nonVarT[:len(nonVarT)-1] {
			at, _ = getOrAdd(&at.Next, t)
		}

		var (
			t          = nonVarT[len(nonVarT)-1]
			cur, added = getOrAdd(&at.Next, t)
		)

		set(cur, added, nonVarT, true)
		if err != nil {
			return
		}

		at = cur
	}

	if varT != nil {
		cur, added := getOrAdd(&at.NextVar, varT)
		cur.Var = true
		set(cur, added, types, true)
	}
	return
}

func (at *MethodArgType) GetMethod(types ObjectTypeArray) *TypedCallerMethod {
	var (
		l   = len(types)
		i   int
		cur = at
		tmp *MethodArgType
	)

	if l == 0 {
		if cur.Method != nil {
			return cur.Method
		}

		if cur.NextVar != nil {
			if arg := cur.NextVar[TAny]; arg != nil {
				return arg.Method
			}
		}
		return nil
	}

	for ; i < l; i++ {
		if tmp = cur.Next.get(types[i]); tmp == nil {
			break
		}
		cur = tmp
	}

	if cur == nil {
		return nil
	}

	// has more or not have method, try *args
	if i < l || cur.Method == nil {
		atv := cur
		for ; atv != nil && i != -1; i-- {
			if atv.NextVar != nil {
				if tmp = atv.NextVar.get(types[i]); tmp != nil {
					if tmp.Type != TAny {
						for _, t := range types[i:] {
							if !IsAssignableTo(t, tmp.Type) {
								goto up
							}
						}
					}
					// check if *args is some type
					return tmp.Method
				}
			}
		up:
			atv = atv.parent
		}
	} else {
		return cur.Method
	}
	return nil
}

func (at *MethodArgType) IsZero() (ok bool) {
	ok = true
	at.Walk(func(m *TypedCallerMethod) any {
		ok = false
		return ok
	})
	return
}

func (at *MethodArgType) label() string {
	return at.labelIndex("")
}

func (at *MethodArgType) labelIndex(methodIndex string) string {
	var m, typ string

	if at.Method != nil {
		m = " " + at.Method.StringTarget(false)
		if len(methodIndex) > 0 {
			m = " " + methodIndex + m
		}
	}

	if at.Type != nil {
		typ = at.Type.Name()
	}

	if at.Var {
		typ = "*" + typ
	}

	if len(typ) > 0 && len(m) > 0 {
		typ += " 🠆"
	}

	return fmt.Sprintf("%s%s", typ, m)
}

func (at *MethodArgType) EachMethods() func(func(i int, m *MethodArgType) bool) {
	return func(yield func(i int, m *MethodArgType) bool) {
		var i int
		at.ArgWalk(func(m *MethodArgType) any {
			if m.Method != nil {
				if !yield(i, m) {
					return false
				}
				i++
			}
			return nil
		})
	}
}

func (at *MethodArgType) MethodsWalk(f func(m *MethodArgType) any) (r any) {
	if at.Method != nil {
		if r = f(at); r != nil {
			return
		}
	}
	if len(at.Next) > 0 {
		if r = at.Next.Sorted(func(m *MethodArgType) any {
			return m.MethodsWalk(f)
		}); r != nil {
			return
		}
	}

	if len(at.NextVar) > 0 {
		r = at.NextVar.Sorted(func(m *MethodArgType) any {
			return f(m)
		})
	}
	return
}

func (at *MethodArgType) ArgWalk(f func(m *MethodArgType) any) (r any) {
	if len(at.Next) > 0 {
		if r = at.Next.Sorted(func(m *MethodArgType) any {
			if r = f(m); r != nil {
				return r
			}
			return m.ArgWalk(f)
		}); r != nil {
			return
		}
	}

	if len(at.NextVar) > 0 {
		if r = at.NextVar.Sorted(func(m *MethodArgType) any {
			if r = f(m); r != nil {
				return r
			}
			return m.ArgWalk(f)
		}); r != nil {
			return
		}
	}
	return
}

func (at *MethodArgType) NumMethods() (i int) {
	at.MethodsWalk(func(m *MethodArgType) any {
		i++
		return nil
	})
	return
}

func (at *MethodArgType) ToString() string {
	var (
		t     = treeprint.NewWithRoot(at.label())
		nodes = map[string]treeprint.Tree{
			fmt.Sprintf("%p", at): t,
		}
		count int
	)

	at.MethodsWalk(func(m *MethodArgType) any {
		var (
			path   = m.Path()[1:]
			parent = t
		)

		for i, pmat := range path {
			pid := fmt.Sprintf("%p", pmat)
			p := nodes[pid]
			if p == nil {
				var methodIndex string
				if i == len(path)-1 {
					methodIndex = fmt.Sprintf("#%d", count)
					count++
				}
				p = parent.AddBranch(pmat.labelIndex(methodIndex))
				nodes[pid] = p
			}
			parent = p
		}
		return nil
	})

	return t.String()
}

type Methods map[ObjectType]*MethodArgType

func (m Methods) Copy() (cp Methods) {
	cp = make(Methods, len(m))
	for ot, at := range m {
		cp[ot] = at.Copy()
	}
	return
}

func (m Methods) IsZero() (ok bool) {
	ok = true
	for _, v := range m {
		if v.Method != nil {
			return false
		}
		if ok = v.Next.IsZero(); !ok {
			return
		}
	}
	return
}

func (m Methods) Walk(cb func(m *TypedCallerMethod) any) (v any) {
	for _, e := range m {
		if v = e.Walk(cb); v != nil {
			return
		}
	}
	return
}

func (m Methods) Sorted(cb func(m *MethodArgType) any) (err any) {
	type kv struct {
		k string
		v ObjectType
	}
	var (
		l      = len(m)
		values = make([]kv, l)
		i      int
	)

	for key := range m {
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
		if err = cb(m[kv.v]); err != nil {
			return
		}
	}
	return
}

func (m Methods) WalkSorted(cb func(m *TypedCallerMethod) any) (v any) {
	type kv struct {
		k string
		v ObjectType
	}

	var (
		l      = len(m)
		values = make([]kv, l)
		i      int
		hasAny bool
	)

	for key := range m {
		if key.Equal(TAny) {
			hasAny = true
		} else {
			values[i] = kv{key.FullName(), key}
			i++
		}
	}

	sort.Slice(values[:i], func(i, j int) bool {
		return values[i].k < values[j].k
	})

	if hasAny {
		values[i] = kv{TAny.FullName(), TAny}
	}

	for _, kv := range values {
		if v = m[kv.v].WalkSorted(cb); v != nil {
			return
		}
	}
	return
}

func (m Methods) get(t ObjectType) (at *MethodArgType) {
	TypeAssigners(t, func(t ObjectType) any {
		if bt, _ := t.(*BuiltinObjType); bt != nil {
			t = bt.TypeKey()
		}
		if at = m[t]; at != nil {
			return true
		}
		return nil
	})
	return
}

type CallerMethodDefinition struct {
	Handler  CallerObject
	Types    ParamsTypes
	Override bool
}
