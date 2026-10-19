package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// ContractStmt is a contract declaration: what a type is (its target) and
// what its metadata may say (the body of an interface):
//
//	contract TSelectField {
//	    *TField                      // parent contracts
//	    target type<str|int|enum>    // the types the target may be (default any)
//	    meta {                       // the keys of the metadata
//	        options []str
//	        layout? str = "table"
//	    }
//	}
//
// Inside the contract `target` is also a type and a value: the target the
// contract is given (`options []target = target.@members`).
type ContractStmt struct {
	ContractPos source.Pos
	NameExpr    *IdentExpr
	// Parents are the `*Parent` contracts.
	Parents []Expr
	// Target are the types of the target (`target T|U`); nil: any.
	Target    []*TypeExpr
	TargetPos source.Pos
	// MetaIface is the `meta { … }` body (nil: no keys of its own).
	MetaIface *InterfaceExpr
	LBrace    source.Pos
	RBrace    source.Pos
	Doc       *ast.CommentGroup
	// Meta is the `[k=v, …]` metadata before the declaration.
	Meta *KeyValueArrayLit
}

func (s *ContractStmt) StmtNode() {}

func (s *ContractStmt) Pos() source.Pos { return s.ContractPos }

func (s *ContractStmt) End() source.Pos { return s.RBrace + 1 }

func (s *ContractStmt) String() string { return Code(s) }

// WriteCode renders the declaration: `contract NAME {*P; target T; meta {…}}`.
func (s *ContractStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(s.Doc)
	writeMeta(ctx, s.Meta)
	ctx.WriteString("contract ")
	s.NameExpr.WriteCode(ctx)
	ctx.WriteString(" {")
	var items []func()
	for _, p := range s.Parents {
		p := p
		items = append(items, func() {
			ctx.WriteString("*")
			ctx.WriteString(p.String())
		})
	}
	if len(s.Target) > 0 {
		items = append(items, func() {
			ctx.WriteString("target ")
			for i, t := range s.Target {
				if i > 0 {
					ctx.WriteString("|")
				}
				t.WriteCode(ctx)
			}
		})
	}
	if s.MetaIface != nil {
		items = append(items, func() {
			ctx.WriteString("meta {")
			writeInterfaceBody(ctx, s.MetaIface)
			ctx.WriteString("}")
		})
	}
	if ctx.HasPrefix() {
		writeBraceItems(ctx, len(items), func(i int) { items[i]() })
		return
	}
	for i, it := range items {
		if i > 0 {
			ctx.WriteString("; ")
		}
		it()
	}
	ctx.WriteString("}")
}
