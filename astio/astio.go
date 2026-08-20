// Package astio serializes a Gad AST to and from a generic tree (JSON / YAML).
//
// Export is reflection-based and works for any node — the built-in parser/node
// types, the gadx AST, and user-defined Expr/Stmt — with no registration: each
// node becomes an object tagged with "$type" (its package-qualified Go type),
// its exported fields recorded by name. Import reverses this: it looks "$type"
// up in a registry (built-ins and gadx are auto-registered; register custom
// types with Register) and reconstructs the concrete node, decoding fields by
// their target Go type. An unregistered "$type" decodes to a *RawNode fallback
// that preserves the tree so nothing is lost.
//
// Limitations: only exported fields are carried (unexported cache fields are
// recomputed on use); token and position fields round-trip as their integer
// values.
package astio

//go:generate go run ./internal/gen

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"gopkg.in/yaml.v3"
)

// RawNode is the import fallback for a "$type" that is not registered: it keeps
// the original tree so the data is not lost and re-exports verbatim. It is not
// renderable (WriteCode is a no-op).
type RawNode struct {
	Type string         // the unresolved "$type"
	Tree map[string]any // the original node object
}

func (r *RawNode) Pos() source.Pos { return 0 }
func (r *RawNode) End() source.Pos { return 0 }
func (r *RawNode) String() string  { return "«" + r.Type + "»" }

// WriteCode makes RawNode a full node.Node (so it can stand in for Expr/Stmt
// fields); it renders nothing — a fallback preserves data, not rendering.
func (r *RawNode) WriteCode(*node.CodeWriteContext) {}

// ExprNode/StmtNode let a fallback stand in for an Expr or Stmt field (the Spec
// and Decl markers are unexported in parser/node, so those positions cannot fall
// back). This keeps import working when a custom expression/statement is missing.
func (r *RawNode) ExprNode() {}
func (r *RawNode) StmtNode() {}

const typeKeyField = "$type"

var astNodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()

// registry maps a "$type" key to the concrete struct type of a node.
var registry = map[string]reflect.Type{}

// typeKey is the package-qualified name used as a node's "$type" (unambiguous
// across packages that share a short name, e.g. parser/node vs gadx/node).
func typeKey(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// Register makes a node type importable. sample must be a (typically nil) typed
// pointer to the node, e.g. Register((*node.IdentExpr)(nil)). Built-in and gadx
// types are registered automatically; call this for user-defined nodes.
func Register(sample any) {
	t := reflect.TypeOf(sample)
	if t == nil || t.Kind() != reflect.Ptr {
		panic("astio: Register wants a typed pointer, e.g. (*T)(nil)")
	}
	registry[typeKey(t)] = t.Elem()
}

// Marshal encodes n into a generic tree of maps/slices/scalars.
func Marshal(n ast.Node) any { return encode(reflect.ValueOf(n)) }

// MarshalJSON / MarshalYAML encode n to JSON / YAML bytes.
func MarshalJSON(n ast.Node) ([]byte, error) { return json.MarshalIndent(Marshal(n), "", "  ") }
func MarshalYAML(n ast.Node) ([]byte, error) { return yaml.Marshal(Marshal(n)) }

// Unmarshal reconstructs a node from a generic tree.
func Unmarshal(tree any) (ast.Node, error) {
	v, err := decode(tree, astNodeType)
	if err != nil {
		return nil, err
	}
	n, _ := v.Interface().(ast.Node)
	return n, nil
}

// UnmarshalJSON / UnmarshalYAML reconstruct a node from JSON / YAML bytes.
func UnmarshalJSON(data []byte) (ast.Node, error) {
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return Unmarshal(tree)
}

func UnmarshalYAML(data []byte) (ast.Node, error) {
	var tree any
	if err := yaml.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return Unmarshal(normalizeYAML(tree))
}

// encode turns a reflect.Value into a generic tree.
func encode(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return encode(v.Elem()) // encode the concrete value the interface holds
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		if raw, ok := v.Interface().(*RawNode); ok {
			return raw.Tree // re-export a fallback verbatim
		}
		if v.Type().Implements(astNodeType) {
			return encodeNode(v)
		}
		return encode(v.Elem())
	case reflect.Struct:
		if v.CanAddr() && v.Addr().Type().Implements(astNodeType) {
			return encodeNode(v.Addr())
		}
		return encodeFields(v)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil
		}
		out := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = encode(v.Index(i))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := map[string]any{}
		for _, k := range v.MapKeys() {
			out[fmt.Sprint(k.Interface())] = encode(v.MapIndex(k))
		}
		return out
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprint(v.Interface())
	}
}

// encodeNode records a node's "$type" and exported fields.
func encodeNode(v reflect.Value) any {
	s := v.Elem()
	m := map[string]any{typeKeyField: typeKey(v.Type())}
	encodeInto(s, m)
	return m
}

func encodeFields(v reflect.Value) any {
	m := map[string]any{}
	encodeInto(v, m)
	return m
}

func encodeInto(s reflect.Value, m map[string]any) {
	if s.Kind() != reflect.Struct {
		return
	}
	t := s.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		if f.Anonymous {
			// Flatten embedded structs (e.g. ast.NodeData) into the same object.
			if s.Field(i).Kind() == reflect.Struct {
				encodeInto(s.Field(i), m)
				continue
			}
		}
		m[f.Name] = encode(s.Field(i))
	}
}

// decode reconstructs a reflect.Value of type target from a generic tree, using
// the target Go type to interpret each value (so token/position fields become
// their integer types, and interface fields dispatch on "$type").
func decode(tree any, target reflect.Type) (reflect.Value, error) {
	switch target.Kind() {
	case reflect.Interface:
		if tree == nil {
			return reflect.Zero(target), nil
		}
		m, ok := tree.(map[string]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("astio: expected a node object for %s, got %T", target, tree)
		}
		return decodeNode(m)

	case reflect.Ptr:
		if tree == nil {
			return reflect.Zero(target), nil
		}
		ptr := reflect.New(target.Elem())
		ev, err := decode(tree, target.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		ptr.Elem().Set(ev)
		return ptr, nil

	case reflect.Struct:
		v := reflect.New(target).Elem()
		if m, ok := tree.(map[string]any); ok {
			if err := decodeStruct(m, v); err != nil {
				return reflect.Value{}, err
			}
		}
		return v, nil

	case reflect.Slice:
		if tree == nil {
			return reflect.Zero(target), nil
		}
		arr, ok := tree.([]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("astio: expected a list for %s, got %T", target, tree)
		}
		out := reflect.MakeSlice(target, len(arr), len(arr))
		for i, e := range arr {
			ev, err := decode(e, target.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(ev)
		}
		return out, nil

	case reflect.Map:
		if tree == nil {
			return reflect.Zero(target), nil
		}
		m, ok := tree.(map[string]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("astio: expected a mapping for %s, got %T", target, tree)
		}
		out := reflect.MakeMapWithSize(target, len(m))
		for k, e := range m {
			ev, err := decode(e, target.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			kv := reflect.ValueOf(k)
			if kv.Type() != target.Key() {
				kv = kv.Convert(target.Key())
			}
			out.SetMapIndex(kv, ev)
		}
		return out, nil

	case reflect.Bool:
		b, _ := tree.(bool)
		return reflect.ValueOf(b).Convert(target), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(toInt(tree)).Convert(target), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(uint64(toInt(tree))).Convert(target), nil
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(toFloat(tree)).Convert(target), nil
	case reflect.String:
		s, _ := tree.(string)
		return reflect.ValueOf(s).Convert(target), nil
	default:
		return reflect.Zero(target), nil
	}
}

// decodeNode dispatches on "$type": a registered type is reconstructed, an
// unknown one becomes a *RawNode fallback.
func decodeNode(m map[string]any) (reflect.Value, error) {
	key, _ := m[typeKeyField].(string)
	st, ok := registry[key]
	if !ok {
		return reflect.ValueOf(&RawNode{Type: key, Tree: m}), nil
	}
	ptr := reflect.New(st)
	if err := decodeStruct(m, ptr.Elem()); err != nil {
		return reflect.Value{}, err
	}
	return ptr, nil
}

// decodeStruct sets each exported field of v from m (by field name), decoding by
// the field's Go type. Unexported fields are left zero (they are caches).
func decodeStruct(m map[string]any, v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		if f.Anonymous && v.Field(i).Kind() == reflect.Struct {
			if err := decodeStruct(m, v.Field(i)); err != nil {
				return err
			}
			continue
		}
		raw, ok := m[f.Name]
		if !ok || raw == nil {
			continue
		}
		fv, err := decode(raw, f.Type)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}
		if fv.IsValid() && v.Field(i).CanSet() {
			v.Field(i).Set(fv)
		}
	}
	return nil
}

func toInt(tree any) int64 {
	switch x := tree.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case uint64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

func toFloat(tree any) float64 {
	switch x := tree.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}

// normalizeYAML converts yaml.v3's map[string]interface{} decoding to the same
// shape json produces (map[string]any / []any), so decode is source-agnostic.
func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = normalizeYAML(e)
		}
		return x
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[fmt.Sprint(k)] = normalizeYAML(e)
		}
		return out
	case []any:
		for i, e := range x {
			x[i] = normalizeYAML(e)
		}
		return x
	default:
		return v
	}
}
