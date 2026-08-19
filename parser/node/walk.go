package node

import (
	"reflect"

	"github.com/gad-lang/gad/parser/ast"
)

// astNodeIface is the reflect.Type of ast.Node — the minimal node interface
// (Pos/End). Keying traversal on it (rather than node.Node) means custom node
// implementations, e.g. the gadx AST which embeds ast.NodeData, are traversed
// too.
var astNodeIface = reflect.TypeOf((*ast.Node)(nil)).Elem()

// Walk traverses the AST rooted at root in depth-first pre-order, calling f
// for every ast.Node it reaches (starting with root). When f returns false the
// children of that node are skipped; returning true descends into them.
//
// Traversal is reflection-based, so it covers every node type — including custom
// implementations (gadx) — without a per-type switch: it follows exported fields
// that are ast.Nodes and descends into slices, arrays, maps, pointers,
// interfaces and plain structs that may contain them. Unexported fields are not
// followed (reflection cannot read them), and nil interfaces/pointers are
// handled.
func Walk(root ast.Node, f func(ast.Node) bool) {
	if isNilNode(reflect.ValueOf(root)) {
		return
	}
	if !f(root) {
		return
	}
	walkChildren(reflect.ValueOf(root), f)
}

// isNilNode reports whether v is a nil node value (a nil interface, or a non-nil
// interface wrapping a nil pointer), which must not be handed to the visitor.
func isNilNode(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// walkChildren descends into the composite value v (dereferencing pointers and
// interfaces), visiting each element/field via walkValue.
func walkChildren(v reflect.Value, f func(ast.Node) bool) {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			walkChildren(v.Elem(), f)
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).PkgPath != "" {
				continue // unexported: not readable via reflection
			}
			walkValue(v.Field(i), f)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkValue(v.Index(i), f)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			walkValue(v.MapIndex(k), f)
		}
	}
}

// walkValue routes a single field/element: an ast.Node restarts Walk on it;
// anything else that may transitively hold nodes is descended into.
func walkValue(v reflect.Value, f func(ast.Node) bool) {
	if v.Type().Implements(astNodeIface) {
		if !isNilNode(v) {
			Walk(v.Interface().(ast.Node), f)
		}
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Array, reflect.Struct, reflect.Map:
		walkChildren(v, f)
	}
}

// IdentNames returns the set of identifier names appearing anywhere in n. It is
// an over-approximation of the free variables of an expression (it also includes
// bound names, selector field names, etc.), which is what callers that must not
// miss a reference — e.g. the declaration reorderer's scope check — rely on.
func IdentNames(n ast.Node) map[string]struct{} {
	names := map[string]struct{}{}
	Walk(n, func(x ast.Node) bool {
		if id, ok := x.(*IdentExpr); ok && !id.Empty && id.Name != "" {
			names[id.Name] = struct{}{}
		}
		return true
	})
	return names
}
