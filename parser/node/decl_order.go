package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/token"
)

// mergeableDecl reports whether s is a declaration statement that can be merged
// into a paren group, returning its kind token and the specs it contributes. A
// simple short var decl (`name := value`, single ident) counts as a `var`; a
// var/const/global/param declaration contributes its own specs.
func mergeableDecl(s Stmt) (tok token.Token, specs []Spec, ok bool) {
	switch v := s.(type) {
	case *AssignStmt:
		if v.Token == token.Define && len(v.LHS) == 1 && len(v.RHS) == 1 {
			if id, isID := v.LHS[0].(*IdentExpr); isID {
				return token.Var, []Spec{&ValueSpec{Idents: []*IdentExpr{id}, Values: []Expr{v.RHS[0]}}}, true
			}
		}
	case *DeclStmt:
		if gd, isGD := v.Decl.(*GenDecl); isGD {
			switch gd.Tok {
			case token.Var, token.Const, token.Global, token.Param:
				return gd.Tok, gd.Specs, true
			}
		}
	}
	return 0, nil, false
}

// newDeclGroup wraps the first statement of a mergeable run in a fresh paren
// GenDecl so later same-kind statements can be appended without mutating the
// original AST. Positions come from s (for the comment/blank-line machinery), and
// a var/const/... group's own lead doc is preserved.
func newDeclGroup(tok token.Token, s Stmt, specs []Spec) *DeclStmt {
	gd := &GenDecl{Tok: tok, TokPos: s.Pos(), Lparen: s.Pos()}
	gd.Specs = append(gd.Specs, specs...)
	if ds, ok := s.(*DeclStmt); ok {
		if og, ok := ds.Decl.(*GenDecl); ok {
			gd.Doc = og.Doc
		}
	}
	gd.setRparen()
	return &DeclStmt{Decl: gd}
}

// setRparen keeps Rparen (and thus End) at the last spec, so surrounding comment
// and blank-line handling sees a sensible span as specs are appended.
func (d *GenDecl) setRparen() {
	if n := len(d.Specs); n > 0 {
		d.Rparen = d.Specs[n-1].End()
	}
}

// stmtLeadDoc returns the lead doc comment that precedes a declaration statement
// (and travels with it into a merged group), or nil.
func stmtLeadDoc(s Stmt) *ast.CommentGroup {
	ds, ok := s.(*DeclStmt)
	if !ok {
		return nil
	}
	gd, ok := ds.Decl.(*GenDecl)
	if !ok {
		return nil
	}
	if gd.Doc != nil {
		return gd.Doc
	}
	if len(gd.Specs) == 1 {
		if vs, ok := gd.Specs[0].(*ValueSpec); ok {
			return vs.Doc
		}
	}
	return nil
}

// declGapBreaks reports whether a blank line or a floating comment between prev
// and cur should stop them from merging into one declaration group. cur's own
// lead doc is not a break — it travels with cur into the group.
func (ctx *CodeWriteContext) declGapBreaks(prev, cur Stmt) bool {
	if ctx.srcFile == nil {
		return false
	}
	doc := stmtLeadDoc(cur)

	// A blank line before cur (measuring from cur's own doc, which is the content
	// immediately above the declaration) breaks the run.
	curStart := cur.Pos()
	if doc != nil {
		curStart = doc.Pos()
	}
	if ctx.lineOf(curStart) > ctx.lineOf(prev.End()-1)+1 {
		return true
	}

	// Any comment between them that is not cur's lead doc breaks the run.
	for _, c := range ctx.comments {
		if c.Pos() < prev.End() || c.Pos() >= cur.Pos() {
			continue
		}
		if doc != nil && c.Pos() >= doc.Pos() && c.End() <= doc.End() {
			continue // part of cur's lead doc
		}
		return true
	}
	return false
}

// declValueRank ranks a value spec's value for declaration ordering (see
// TASK.md): 1 no value; 3 plain (literal or bare ident); 4 expression;
// 5 ComputedExpr `(= …)`; 6 closure `=>`; 7 func. (Rank 2 — typed valueless —
// is param-only and never reached here.)
func declValueRank(v Expr) int {
	switch v.(type) {
	case nil:
		return 1
	case *IntLit, *UintLit, *FloatLit, *DecimalLit, *BoolLit, *FlagLit, *CharLit,
		*NilLit, *StrLit, *RawStrLit, *BytesLit, *HeredocLit, *RawHeredocLit,
		*DurationLit, *DateTimeLit, *SymbolLit, *IdentExpr:
		return 3
	case *ComputedExpr:
		return 5
	case *ClosureExpr:
		return 6
	case *FuncExpr:
		return 7
	default:
		return 4 // expression: operators, calls, cond, index, selector, collections, …
	}
}

// declItem is one flattened declaration item for ordering.
type declItem struct {
	spec *ValueSpec
	name string
	rank int
	refs map[string]struct{} // names the value references (over-approx)
	orig int
}

// orderedSpecs returns d's specs reordered per the declaration-ordering rules
// (grouped by rank, then by name), or nil when the group is not eligible and must
// be left exactly as written. Only `var`/`const` paren groups of single-ident,
// pattern-free value specs are reordered, and never in a way that could change
// name resolution: a value that references a name declared in the same group
// keeps its position relative to that declaration (so shadowing is preserved).
func (d *GenDecl) orderedSpecs() []Spec {
	if (d.Tok != token.Var && d.Tok != token.Const) || !d.Lparen.IsValid() || len(d.Specs) < 2 {
		return nil
	}

	items := make([]declItem, len(d.Specs))
	declared := make(map[string]int, len(d.Specs)) // name -> item index
	for i, sp := range d.Specs {
		vs, ok := sp.(*ValueSpec)
		if !ok || vs.Pattern != nil || len(vs.Idents) != 1 {
			return nil // not a simple single-ident spec: leave the whole group intact
		}
		var val Expr
		if len(vs.Values) > 0 {
			val = vs.Values[0]
		}
		var refs map[string]struct{}
		if val != nil {
			refs = IdentNames(val)
			if _, iota := refs["iota"]; iota {
				return nil // const iota is position-sensitive: leave intact
			}
		}
		items[i] = declItem{spec: vs, name: vs.Idents[0].Name, rank: declValueRank(val), refs: refs, orig: i}
		declared[items[i].name] = i
	}

	// Resolution-preserving constraints: for each value reference to a
	// group-declared name, keep the referencing item and the declaration in their
	// original relative order. Every edge points forward in the original order, so
	// the graph is acyclic and the source order is a valid topological order.
	n := len(items)
	adj := make([][]int, n)
	indeg := make([]int, n)
	for i := range items {
		for ref := range items[i].refs {
			j, ok := declared[ref]
			if !ok || j == i {
				continue
			}
			u, v := j, i // originally j before i
			if j > i {
				u, v = i, j // originally i before j
			}
			adj[u] = append(adj[u], v)
			indeg[v]++
		}
	}

	// Greedy topological sort: repeatedly place the available item (all
	// predecessors placed) with the smallest (rank, name).
	placed := make([]bool, n)
	out := make([]Spec, 0, n)
	for len(out) < n {
		best := -1
		for k := 0; k < n; k++ {
			if placed[k] || indeg[k] != 0 {
				continue
			}
			if best == -1 || declLess(items[k], items[best]) {
				best = k
			}
		}
		if best == -1 {
			return nil // unexpected cycle: leave intact rather than risk breakage
		}
		placed[best] = true
		out = append(out, items[best].spec)
		for _, w := range adj[best] {
			indeg[w]--
		}
	}
	return out
}

// writeShortVar renders a single `var name = value` as the short form
// `name := value` (equivalent semantics) and returns true when it applied. Only
// a var with exactly one identifier and a value collapses; valueless vars,
// const/global, destructuring patterns and multi-item groups take the normal
// path.
func (d *GenDecl) writeShortVar(ctx *CodeWriteContext) bool {
	if d.Tok != token.Var || len(d.Specs) != 1 {
		return false
	}
	vs, ok := d.Specs[0].(*ValueSpec)
	if !ok || vs.Pattern != nil || len(vs.Idents) != 1 || len(vs.Values) != 1 || vs.Values[0] == nil {
		return false
	}
	lead := isLeadDoc(vs.Doc, vs)
	if lead {
		ctx.WriteLeadDoc(vs.Doc)
	}
	vs.Idents[0].WriteCode(ctx)
	ctx.WriteString(" := ")
	vs.Values[0].WriteCode(ctx)
	if !lead {
		ctx.WriteTrailingDoc(vs.Doc)
	}
	return true
}

// specsForWrite returns the specs in the order they should be emitted: the
// reordered order when the group is eligible, otherwise the original specs.
func (d *GenDecl) specsForWrite() []Spec {
	if o := d.orderedSpecs(); o != nil {
		return o
	}
	return d.Specs
}

// declLess orders items by rank, then by identifier name.
func declLess(a, b declItem) bool {
	if a.rank != b.rank {
		return a.rank < b.rank
	}
	return a.name < b.name
}
