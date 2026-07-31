package gad

import (
	"bytes"

	"github.com/gad-lang/gad/parser/node"
)

// StmtsType is the object type of StmtsObject (gad.StmtsObject): an ordered,
// indexable, iterable collection of Gad statements produced by gad.parse /
// gad.parseFile and consumed by gad.eval.
var StmtsType = registerGadNamespaceType(BuiltinStmts, "StmtsObject", StmtsObject(nil))

// StmtType is the object type of a single Gad statement (gad.StmtObject), the
// element type yielded when iterating or indexing a StmtsObject.
var StmtType = registerGadNamespaceType(BuiltinStmt, "StmtObject", (*StmtObject)(nil))

// registerGadNamespaceType registers a builtin object type that lives in the
// `gad` namespace (FullName gad.<name>).
func registerGadNamespaceType(typ BuiltinType, name string, sample any) *BuiltinObjType {
	t := RegisterBuiltinType(typ, name, sample, nil)
	t.Module = gadModuleSpec
	return t
}

// StmtObject wraps a single Gad AST statement as a Gad value. It is the element
// type of StmtsObject.
type StmtObject struct {
	Stmt node.Stmt
}

func (*StmtObject) Type() ObjectType { return StmtType }

func (o *StmtObject) ToString() string {
	if o.Stmt == nil {
		return ""
	}
	var buf bytes.Buffer
	node.CodeW(&buf, o.Stmt)
	return buf.String()
}

// Equal reports identity of the wrapped statement node.
func (o *StmtObject) Equal(right Object) bool {
	r, ok := right.(*StmtObject)
	return ok && r.Stmt == o.Stmt
}

func (o *StmtObject) IsFalsy() bool { return o.Stmt == nil }

// StmtsObject is a sequence of Gad statements (an AST fragment) exposed to Gad
// code. It supports len(), indexing (get/set) and iteration; each element is a
// StmtObject. gad.eval compiles and runs it.
type StmtsObject node.Stmts

func (StmtsObject) Type() ObjectType { return StmtsType }

func (o StmtsObject) ToString() string {
	var buf bytes.Buffer
	node.CodeW(&buf, node.Stmts(o))
	return buf.String()
}

// Equal reports element-wise identity of the wrapped statement nodes.
func (o StmtsObject) Equal(right Object) bool {
	r, ok := right.(StmtsObject)
	if !ok || len(r) != len(o) {
		return false
	}
	for i := range o {
		if o[i] != r[i] {
			return false
		}
	}
	return true
}

func (o StmtsObject) IsFalsy() bool { return len(o) == 0 }

// Length implements the LengthGetter interface (len(stmts)).
func (o StmtsObject) Length() int { return len(o) }

// Copy implements the Copier interface.
func (o StmtsObject) Copy() Object {
	cp := make(StmtsObject, len(o))
	copy(cp, o)
	return cp
}

// IndexGet returns the statement at index i wrapped in a StmtObject.
func (o StmtsObject) IndexGet(_ *VM, index Object) (Object, error) {
	i, err := stmtsIndex(index, len(o))
	if err != nil {
		return nil, err
	}
	return &StmtObject{Stmt: o[i]}, nil
}

// IndexSet replaces the statement at index i; value must be a StmtObject.
func (o StmtsObject) IndexSet(_ *VM, index, value Object) error {
	i, err := stmtsIndex(index, len(o))
	if err != nil {
		return err
	}
	st, ok := value.(*StmtObject)
	if !ok {
		return NewIndexValueTypeError(index.ToString(), "StmtObject", value.Type().Name())
	}
	o[i] = st.Stmt
	return nil
}

// Iterate yields each statement as a StmtObject, reusing the Array iterator.
func (o StmtsObject) Iterate(vm *VM, na *NamedArgs) Iterator {
	arr := make(Array, len(o))
	for i, s := range o {
		arr[i] = &StmtObject{Stmt: s}
	}
	return arr.Iterate(vm, na)
}

func stmtsIndex(index Object, n int) (int, error) {
	switch v := index.(type) {
	case Int:
		i := int(v)
		if i < 0 {
			i = n + i
		}
		if i >= 0 && i < n {
			return i, nil
		}
		return 0, ErrIndexOutOfBounds
	case Uint:
		i := int(v)
		if i >= 0 && i < n {
			return i, nil
		}
		return 0, ErrIndexOutOfBounds
	}
	return 0, NewIndexTypeError("int|uint", index.Type().Name())
}
