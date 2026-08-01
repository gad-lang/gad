package gad

import "github.com/gad-lang/gad/parser/node"

// compileDeleteStmt compiles a `delete` statement. Two forms are supported:
//
//	delete Target.field        // Keys == nil: single key (the selector name)
//	delete Target [k1, k2, …]   // Keys is an array/expr of evaluated keys
//
// Both leave the target (`this`) and the key(s) on the stack for OpDelete, which
// pops both and deletes this[k] for each key (a single non-array key deletes one).
func (c *Compiler) compileDeleteStmt(nd *node.DeleteStmt) error {
	if nd.Keys != nil {
		if err := c.Compile(nd.Target); err != nil {
			return err
		}
		if err := c.Compile(nd.Keys); err != nil {
			return err
		}
		c.emit(nd, OpDelete)
		return nil
	}

	// Selector form: `delete base.field` -> this = base, key = "field".
	sel, ok := nd.Target.(*node.SelectorExpr)
	if !ok {
		return c.Errorf(nd, "delete: expected `Target.key` or `Target [keys]`")
	}
	if err := c.Compile(sel.X); err != nil {
		return err
	}
	if err := c.Compile(sel.Sel); err != nil {
		return err
	}
	c.emit(nd, OpDelete)
	return nil
}
