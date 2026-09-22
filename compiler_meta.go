package gad

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
)

// evalMeta evaluates a `[k=v, …]` metadata block to a runtime KeyValueArray at
// compile time and returns it, so it can be attached to the compiled element and
// read through `@meta`. Metadata is a real KeyValueArray expression, but it must
// be constant-evaluable (literals, nested `[…]`/`(;…)`, arrays, dicts and
// constant operators) so the value can accompany the element in memory; a
// non-constant value is a compile error.
func (c *Compiler) evalMeta(nd ast.Node, kva *node.KeyValueArrayLit) (KeyValueArray, error) {
	if kva == nil || len(kva.Elements) == 0 {
		return nil, nil
	}
	obj, err := c.metaValue(nd, kva)
	if err != nil {
		return nil, err
	}
	arr, _ := obj.(KeyValueArray)
	return arr, nil
}

// metaKeyValueArray builds a KeyValueArray from a `[…]` / `(;…)` literal's
// elements: `k=v` pairs, bare-name flags (`k` == `k=true`) and `**spread` of a
// nested key-value array.
func (c *Compiler) metaKeyValueArray(nd ast.Node, kva *node.KeyValueArrayLit) (KeyValueArray, error) {
	out := make(KeyValueArray, 0, len(kva.Elements))
	for _, el := range kva.Elements {
		switch e := el.(type) {
		case *node.KeyValuePairLit:
			k, err := c.metaKey(nd, e.Key)
			if err != nil {
				return nil, err
			}
			var v Object = Flag(true) // a bare key is a flag `k` == `k=true`
			if e.Value != nil {
				if v, err = c.metaValue(nd, e.Value); err != nil {
					return nil, err
				}
			}
			out = append(out, &KeyValue{K: k, V: v})
		case *node.NamedArgVarLit:
			v, err := c.metaValue(nd, e.Value)
			if err != nil {
				return nil, err
			}
			if kv, ok := v.(KeyValueArray); ok {
				out = append(out, kv...)
			} else {
				return nil, c.Errorf(nd, "metadata `**` spread must be a key-value array, got %s", v.Type().Name())
			}
		default:
			return nil, c.Errorf(nd, "unsupported metadata element %T", el)
		}
	}
	return out, nil
}

// metaKey resolves a metadata key expression to its Object key (a bare name or a
// string literal becomes a Str; any other constant is used verbatim).
func (c *Compiler) metaKey(nd ast.Node, e node.Expr) (Object, error) {
	switch k := e.(type) {
	case *node.IdentExpr:
		return Str(k.Name), nil
	case *node.StrLit:
		return Str(k.Value()), nil
	case *node.RawStrLit:
		return Str(k.Value()), nil
	}
	return c.metaValue(nd, e)
}

// metaValue evaluates a constant metadata value expression to an Object.
func (c *Compiler) metaValue(nd ast.Node, e node.Expr) (Object, error) {
	switch v := e.(type) {
	case *node.StrLit:
		return Str(v.Value()), nil
	case *node.RawStrLit:
		return RawStr(v.Value()), nil
	case *node.IntLit:
		return Int(v.Value), nil
	case *node.UintLit:
		return Uint(v.Value), nil
	case *node.FloatLit:
		return Float(v.Value), nil
	case *node.DecimalLit:
		return Decimal(v.Value), nil
	case *node.CharLit:
		return Char(v.Value), nil
	case *node.BoolLit:
		return Bool(v.Value), nil
	case *node.FlagLit:
		return Flag(v.Value), nil
	case *node.NilLit:
		return Nil, nil
	case *node.KeyValueArrayLit:
		return c.metaKeyValueArray(nd, v)
	case *node.ArrayExpr:
		arr := make(Array, 0, len(v.Elements))
		for _, el := range v.Elements {
			o, err := c.metaValue(nd, el)
			if err != nil {
				return nil, err
			}
			arr = append(arr, o)
		}
		return arr, nil
	case *node.DictExpr:
		d := make(Dict, len(v.Elements))
		for _, el := range v.Elements {
			if el.Spread != nil {
				sv, err := c.metaValue(nd, el.Spread)
				if err != nil {
					return nil, err
				}
				if sd, ok := sv.(Dict); ok {
					for k, val := range sd {
						d[k] = val
					}
					continue
				}
				return nil, c.Errorf(nd, "metadata dict `*` spread must be a dict, got %s", sv.Type().Name())
			}
			k, err := c.metaKey(nd, el.Key)
			if err != nil {
				return nil, err
			}
			val, err := c.metaValue(nd, el.Value)
			if err != nil {
				return nil, err
			}
			d[k.ToString()] = val
		}
		return d, nil
	}
	return nil, c.Errorf(nd, "metadata value must be a constant expression, got %T", e)
}
