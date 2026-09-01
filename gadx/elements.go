package gadx

import (
	"io"

	gad "github.com/gad-lang/gad"
)

// Elements is a FRAGMENT: an ordered list of child elements with no wrapping
// element. It is what a component, slot or `@main` body builds and returns — on
// render it writes its children inline, in order, with nothing around them.
//
// It replaces the old anonymous (empty-name) tag. Being a pointer type it is the
// cheap, mutable object the compiled template threads through the tree: a child
// tag/text created with an Elements parent appends itself into it, and the same
// `el += child` / `el ++= children` operators a tag supports build it up.
type Elements struct {
	Items []Element
}

// ElementsType is the Gad object type of *Elements. Calling it builds a fragment:
//
//	gadx.Elements(*children)
//
// A fragment has NO parent — it is the root a component/slot/@main body builds and
// returns; nested tags take it as THEIR parent and append into it.
var ElementsType = gad.NewBuiltinObjType("Elements").WithNew(elementsCtor)

func init() { ElementsType.SetModule(ModuleSpec) }

// elementsCtor implements gadx.Elements(*children).
func elementsCtor(c gad.Call) (gad.Object, error) {
	e := &Elements{}
	for i := 0; i < c.Args.Length(); i++ {
		e.append(c.Args.Get(i))
	}
	return e, nil
}

var (
	_ Element                             = (*Elements)(nil)
	_ gad.Object                          = (*Elements)(nil)
	_ gad.ToWriter                        = (*Elements)(nil)
	_ gad.ObjectWithAddBinOperator        = (*Elements)(nil)
	_ gad.ObjectWithAddSelfAssignOperator = (*Elements)(nil)
	_ gad.ObjectWithIncSelfAssignOperator = (*Elements)(nil)
	_ gad.Iterabler                       = (*Elements)(nil)
)

func (e *Elements) ElType() ElementType  { return ElementFragment }
func (e *Elements) Type() gad.ObjectType { return ElementsType }
func (e *Elements) ToString() string     { return gad.ReprQuote("elements") }
func (e *Elements) IsFalsy() bool        { return len(e.Items) == 0 }

func (e *Elements) Equal(right gad.Object) bool {
	o, _ := right.(*Elements)
	return o == e
}

// append adds one child (nil/Nil are skipped; a non-element value becomes text).
// An *Elements child is spliced in — its items are appended individually (as if
// `parent ++= child.Items`), never nested as a single fragment node — so building
// a fragment from other fragments yields one flat list.
func (e *Elements) append(child gad.Object) {
	if child == nil || child == gad.Nil {
		return
	}
	if frag, ok := child.(*Elements); ok {
		e.Items = append(e.Items, frag.Items...)
		return
	}
	e.Items = append(e.Items, toElement(child))
}

// appendMany adds each element of an iterable value as a child.
func (e *Elements) appendMany(vm *gad.VM, values gad.Object) error {
	if arr, ok := gad.ToArray(values); ok {
		for _, v := range arr {
			e.append(v)
		}
		return nil
	}
	vals, err := gad.ValuesOf(vm, values, &gad.NamedArgs{})
	if err != nil {
		return err
	}
	for _, v := range vals {
		e.append(v)
	}
	return nil
}

// WriteTo renders the fragment: just its children, in order.
func (e *Elements) WriteTo(vm *gad.VM, w io.Writer) (n int64, err error) {
	for _, c := range e.Items {
		var cn int64
		if cn, err = c.WriteTo(vm, w); err != nil {
			return n + cn, err
		}
		n += cn
	}
	return n, nil
}

// BinOpAdd implements `el + child` (and backs `el += child` via the fallback),
// appending the child and yielding the fragment.
func (e *Elements) BinOpAdd(_ *gad.VM, right gad.Object) (gad.Object, error) {
	e.append(right)
	return e, nil
}

// SelfAssignOpAdd implements `el += child`, appending one child.
func (e *Elements) SelfAssignOpAdd(_ *gad.VM, value gad.Object) (gad.Object, error) {
	e.append(value)
	return e, nil
}

// SelfAssignOpInc implements `el ++= children`, appending each element of the
// iterable value.
func (e *Elements) SelfAssignOpInc(vm *gad.VM, value gad.Object) (gad.Object, error) {
	if err := e.appendMany(vm, value); err != nil {
		return nil, err
	}
	return e, nil
}

// Iterate yields the fragment's children, so `for x in el` and `el ++= other`
// (via ValuesOf) work.
func (e *Elements) Iterate(vm *gad.VM, na *gad.NamedArgs) gad.Iterator {
	arr := make(gad.Array, len(e.Items))
	for i, c := range e.Items {
		arr[i] = c
	}
	return arr.Iterate(vm, na)
}
