package node

import "github.com/gad-lang/gad/token"

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
