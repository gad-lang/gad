package node

import (
	"fmt"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Convert recursively converts giom-specific AST nodes to GAD AST nodes,
// returning pure GAD statements suitable for Format.
// Consecutive const/var declarations are merged into grouped declarations.
func Convert(stmts gnode.Stmts) gnode.Stmts {
	var out gnode.Stmts
	for _, s := range stmts {
		out = append(out, convertStmt(s)...)
	}
	return mergeDecls(out)
}

// ConvertFile converts a whole giom file's top-level statements, wrapping them
// so all render content (whether under @main or written bare at the top level)
// builds into a single root tag that the program returns. Declarations, imports
// and exports remain top-level statements between the root binding and the
// return.
func ConvertFile(stmts gnode.Stmts) gnode.Stmts {
	return fragmentStmts(gnode.LNil(0), Convert(stmts), 0, 0)
}

// mergeDecls merges consecutive const/var GenDecl statements into grouped declarations.
func mergeDecls(stmts gnode.Stmts) gnode.Stmts {
	var out gnode.Stmts
	var pending *gnode.GenDecl
	for i, s := range stmts {
		ds, ok := s.(*gnode.DeclStmt)
		if !ok {
			out = appendPending(out, &pending)
			out = append(out, s)
			continue
		}
		gd, ok := ds.Decl.(*gnode.GenDecl)
		if !ok || (gd.Tok != token.Const && gd.Tok != token.Var) {
			out = appendPending(out, &pending)
			out = append(out, s)
			continue
		}
		if pending != nil && pending.Tok == gd.Tok {
			if !pending.Lparen.IsValid() {
				pending.Lparen = stmts[i-1].Pos()
			}
			pending.Specs = append(pending.Specs, gd.Specs...)
			if gd.Rparen.IsValid() {
				pending.Rparen = gd.Rparen
			}
			if i == len(stmts)-1 {
				out = append(out, gnode.SDecl(pending))
			}
		} else {
			out = appendPending(out, &pending)
			pending = gd
			if i == len(stmts)-1 {
				out = append(out, gnode.SDecl(pending))
			}
		}
	}
	return out
}

// appendPending flushes a pending GenDecl to out if it has multiple specs,
// setting Lparen/Rparen for grouped syntax.
func appendPending(out gnode.Stmts, pending **gnode.GenDecl) gnode.Stmts {
	if *pending == nil {
		return out
	}
	gd := *pending
	if len(gd.Specs) > 1 && !gd.Lparen.IsValid() {
		gd.Lparen = 1
	}
	if len(gd.Specs) > 1 && !gd.Rparen.IsValid() {
		gd.Rparen = 1
	}
	out = append(out, gnode.SDecl(gd))
	*pending = nil
	return out
}

func funcType(params *gnode.FuncParams) *gnode.FuncType {
	return &gnode.FuncType{
		FuncPos:    1,
		FuncHeader: gnode.FuncHeader{Params: *params},
	}
}

func funcExpr(params *gnode.FuncParams, body gnode.Stmts, pos, end source.Pos) *gnode.FuncExpr {
	return &gnode.FuncExpr{
		Type: funcType(params),
		Body: gnode.SBlock(pos, end, body...),
	}
}

// tagVar is the identifier that always names the current parent tag in scope.
// Each tag/fragment opens a block that rebinds `tag` (via `:=`) to itself, so
// nested content appends to it while sibling content sees the outer tag.
const tagVar = "tag"

func tagIdent(pos source.Pos) *gnode.IdentExpr { return gnode.EIdent(tagVar, pos) }

// giomNew builds a `giom.<ctor>(args…)` constructor call.
func giomNew(ctor string, pos, end source.Pos, args ...gnode.Expr) *gnode.CallExpr {
	call := giomCallExpr(ctor, pos)
	setParens(call, pos, end)
	call.Args.Values = args
	return call
}

// defineTag builds `tag := <rhs>` (a new block-scoped binding of the current tag).
func defineTag(rhs gnode.Expr, pos source.Pos) gnode.Stmt {
	return &gnode.AssignStmt{LHS: []gnode.Expr{tagIdent(pos)}, RHS: []gnode.Expr{rhs}, Token: token.Define, TokenPos: pos}
}

// appendToTag builds `tag += <expr>`, appending a rendered value (a component or
// slot fragment) to the current tag.
func appendToTag(expr gnode.Expr, pos source.Pos) gnode.Stmt {
	return &gnode.AssignStmt{LHS: []gnode.Expr{tagIdent(pos)}, RHS: []gnode.Expr{expr}, Token: token.AddAssign, TokenPos: pos}
}

// fragmentStmts wraps body so it builds into a fresh anonymous root tag and
// returns it: `tag := giom.Tag(<parent>); <body>; return tag`. A name-less
// giom.Tag call yields an anonymous fragment (renders only its children). Used
// for @comp / @func / @slot / slot-pass bodies and the top-level @main.
func fragmentStmts(parent gnode.Expr, body gnode.Stmts, pos, end source.Pos) gnode.Stmts {
	var out gnode.Stmts
	out.Append(defineTag(giomNew("Tag", pos, end, parent), pos))
	out.Append(body...)
	out.Append(gnode.SReturn(end, tagIdent(end)))
	return out
}

func convertStmt(s gnode.Stmt) gnode.Stmts {
	switch st := s.(type) {
	case *FuncDecl:
		return convertFuncDecl(st)
	case *CompDecl:
		return convertCompDecl(st)
	case *CompCallStmt:
		return convertCompCall(st)
	case *MatchStmt:
		return convertMatch(st)
	case *VarStmt:
		return convertVar(st)
	case *ConstStmt:
		return convertConst(st)
	case *GlobalStmt:
		return convertGlobal(st)
	case *EnumStmt:
		return convertEnum(st)
	case *ExportStmt:
		return convertExport(st)
	case *SlotDecl:
		return convertSlot(st)
	case *SlotPassStmt:
		return convertSlotPass(st)
	case *CodeStmt:
		return st.Stmts
	case *AssignStmt:
		return convertAssign(st)
	case *ForStmt:
		return convertFor(st)
	case *IfStmt:
		return convertIf(st)
	case *DoctypeStmt:
		return convertDoctype(st)
	case *TextStmt:
		return convertText(st)
	case *TagStmt:
		return convertTag(st)
	case *HtmlStmt:
		return convertHtml(st)
	default:
		return gnode.Stmts{s}
	}
}

func convertAssign(s *AssignStmt) gnode.Stmts {
	return gnode.Stmts{
		&gnode.AssignStmt{
			LHS:      []gnode.Expr{s.LHS},
			RHS:      []gnode.Expr{s.RHS},
			Token:    assignToken(s.Op),
			TokenPos: s.NodePos,
		},
	}
}

func assignToken(op string) token.Token {
	switch op {
	case ":=", ":":
		return token.Define
	case "=":
		return token.Assign
	case "+=":
		return token.AddAssign
	case "-=":
		return token.SubAssign
	case "*=":
		return token.MulAssign
	case "/=":
		return token.QuoAssign
	case "%=":
		return token.RemAssign
	case "??=":
		return token.NullichAssign
	default:
		return token.Assign
	}
}

func convertBody(stmts gnode.Stmts) gnode.Stmts {
	return Convert(stmts)
}

func convertFuncDecl(f *FuncDecl) gnode.Stmts {
	params := addSlotsParam(f.Params)
	body := fragmentStmts(gnode.LNil(f.Pos()), convertBody(f.Body), f.Pos(), f.End())
	fn := funcExpr(params, body, f.Pos(), f.End())
	stmts := recursiveFuncStmts(f.Name, fn, f.Pos())
	if f.Exported {
		stmts = append(stmts, &gnode.ExportStmt{
			TokenPos: f.Pos(),
			KeyExpr:  gnode.EIdent(f.Name, f.Pos()),
		})
	}
	return stmts
}

func addSlotsParam(params *gnode.FuncParams) *gnode.FuncParams {
	if params == nil {
		return nil
	}
	for _, n := range params.NamedArgs.Names {
		if n != nil && n.Ident != nil && n.Ident.Name == "slots" {
			return params
		}
	}
	out := *params
	out.NamedArgs.Names = append(out.NamedArgs.Names, &gnode.TypedIdentExpr{Ident: gnode.EIdent("slots", 0)})
	out.NamedArgs.Values = append(out.NamedArgs.Values, &gnode.DictExpr{})
	return &out
}

func convertCompDecl(c *CompDecl) gnode.Stmts {
	var body gnode.Stmts
	for _, comp := range c.Comps {
		body = append(body, convertStmt(comp)...)
	}
	body = append(body, convertBody(c.Body)...)

	if c.Main {
		// @main content builds into the file's root tag (created by ConvertFile),
		// so it is emitted inline; the surrounding wrapper returns the root.
		return body
	}
	fnBody := fragmentStmts(gnode.LNil(c.Pos()), body, c.Pos(), c.End())
	fn := funcExpr(addSlotsParam(c.Params), fnBody, c.Pos(), c.End())

	stmts := recursiveFuncStmts(c.ID, fn, c.Pos())
	if c.Exported {
		stmts = append(stmts, &gnode.ExportStmt{
			TokenPos: c.Pos(),
			KeyExpr:  gnode.EIdent(c.ID, c.Pos()),
		})
	}
	return stmts
}

func recursiveFuncStmts(name string, fn *gnode.FuncExpr, pos source.Pos) gnode.Stmts {
	ident := gnode.EIdent(name, pos)
	return gnode.Stmts{
		gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Var,
			TokPos: pos,
			Specs: []gnode.Spec{
				&gnode.ValueSpec{Idents: []*gnode.IdentExpr{ident}, Values: []gnode.Expr{gnode.LNil(pos)}},
			},
		}),
		&gnode.AssignStmt{
			LHS:      []gnode.Expr{gnode.EIdent(name, pos)},
			RHS:      []gnode.Expr{fn},
			Token:    token.Assign,
			TokenPos: pos,
		},
	}
}

func convertCompCall(c *CompCallStmt) gnode.Stmts {
	fn := c.Func
	if fn == nil {
		fn = gnode.EIdent(c.Name, c.Pos())
	}
	call := &gnode.CallExpr{
		Func: fn,
	}
	if !c.Args.LParen.IsValid() {
		call.LParen = c.Pos()
	}
	if !c.Args.RParen.IsValid() {
		call.RParen = c.End()
	}
	call.Args = c.Args.Args
	call.NamedArgs = c.Args.NamedArgs

	// A call to the auto-injected `super` forwards super's own super (an empty
	// function) as its first positional argument, so the invoked default/override
	// function — which also declares `super` first — receives a safe fallback and
	// may itself call `super(…)` without failing.
	if c.Name == "super" {
		call.Args.Values = append([]gnode.Expr{emptySuperFunc(c.Pos(), c.End())}, call.Args.Values...)
	}

	if len(c.SlotPass) == 0 && len(c.InitStmts) == 0 {
		return gnode.Stmts{appendToTag(call, c.Pos())}
	}

	// Call-scope init code (`~` / `~~ … ~~`) comes first, so interpolated slot
	// names and slot bodies can reference the values it declares.
	var stmts gnode.Stmts
	for _, st := range c.InitStmts {
		stmts = append(stmts, convertStmt(st)...)
	}

	if len(c.SlotPass) == 0 {
		stmts.Append(appendToTag(call, c.Pos()))
		return stmts
	}

	// With slot passes, wrap in a block:
	//   const $slot0 = func(...) { ... }
	//   var $$slots = {}
	//   $$slots["main"] = $slot0
	//   page_wrapper(args; slots=$$slots)
	slotPrefix := fmt.Sprintf("$slot%d", c.Pos())
	slotsName := fmt.Sprintf("$$slots%d", c.Pos())
	for i, sp := range c.SlotPass {
		slotName := fmt.Sprintf("%s_%d", slotPrefix, i)
		ft := sp.FuncType
		if ft == nil {
			ft = &gnode.FuncType{}
		}
		if !ft.FuncPos.IsValid() {
			ft.FuncPos = sp.Pos()
		}
		// Auto-inject `super` as the override's first positional parameter (the
		// enclosing component passes the slot's default as this argument), so
		// overriding content can render the default by calling `super(…)`.
		withSuperParam(&ft.FuncHeader.Params)
		stmts.Append(gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Const,
			TokPos: sp.Pos(),
			Specs: []gnode.Spec{
				&gnode.ValueSpec{
					Idents: []*gnode.IdentExpr{gnode.EIdent(slotName, sp.Pos())},
					Values: []gnode.Expr{
						&gnode.FuncExpr{
							Type: ft,
							Body: gnode.SBlock(sp.Pos(), sp.End(),
								fragmentStmts(gnode.LNil(sp.Pos()), convertBody(sp.Body), sp.Pos(), sp.End())...),
						},
					},
				},
			},
		}))
	}
	stmts.Append(gnode.SDecl(&gnode.GenDecl{
		Tok:    token.Var,
		TokPos: c.Pos(),
		Specs: []gnode.Spec{
			&gnode.ValueSpec{
				Idents: []*gnode.IdentExpr{gnode.EIdent(slotsName, c.Pos())},
				Values: []gnode.Expr{gnode.EDict(c.Pos(), c.End())},
			},
		},
	}))
	for i, sp := range c.SlotPass {
		slotName := fmt.Sprintf("%s_%d", slotPrefix, i)
		stmts = append(stmts, &gnode.AssignStmt{
			LHS: []gnode.Expr{
				&gnode.IndexExpr{
					X:     gnode.EIdent(slotsName, 0),
					Index: slotPassIndex(sp),
				},
			},
			RHS:      []gnode.Expr{gnode.EIdent(slotName, 0)},
			Token:    token.Assign,
			TokenPos: sp.Pos(),
		})
	}
	call.NamedArgs.AppendS("slots", gnode.EIdent(slotsName, 0))
	stmts.Append(appendToTag(call, c.Pos()))
	return stmts
}

// slotPassIndex is the `$$slots[…]` key for a slot pass: the interpolated name
// expression when present, otherwise the static name as a string literal.
func slotPassIndex(sp *SlotPassStmt) gnode.Expr {
	if sp.NameExpr != nil {
		return sp.NameExpr
	}
	return gnode.Str(slotPassName(sp), 0)
}

func slotPassName(sp *SlotPassStmt) string {
	if sp.Name != nil {
		if s, ok := sp.Name.(*gnode.StrLit); ok {
			return s.Value()
		}
		if s, ok := sp.Name.(*gnode.IdentExpr); ok {
			return s.Name
		}
	}
	return "default"
}

func convertMatch(s *MatchStmt) gnode.Stmts {
	return gnode.Stmts{gnode.SExpr(switchMatchExpr(s))}
}

func switchMatchExpr(s *MatchStmt) *gnode.MatchExpr {
	match := &gnode.MatchExpr{
		MatchPos: s.Pos(),
		Expr:     s.Tag,
		LBrace:   s.Pos(),
		RBrace:   s.End(),
	}
	for _, c := range s.Cases {
		match.Arms = append(match.Arms, &gnode.MatchArm{
			Conds: []gnode.Expr{c.Expr},
			Body:  gnode.SBlock(s.Pos(), s.End(), convertBody(c.Body)...),
		})
	}
	if len(s.Default) > 0 {
		match.Arms = append(match.Arms, &gnode.MatchArm{
			Body: gnode.SBlock(s.Pos(), s.End(), convertBody(s.Default)...),
		})
	}
	return match
}

func convertExport(e *ExportStmt) gnode.Stmts {
	return gnode.Stmts{
		&gnode.ExportStmt{
			TokenPos:  e.Pos(),
			KeyExpr:   gnode.EIdent(e.Name, e.Pos()),
			ValueExpr: e.Value,
		},
	}
}

func convertVar(s *VarStmt) gnode.Stmts {
	if s.Decl != nil {
		return gnode.Stmts{gnode.SDecl(s.Decl)}
	}
	var specs []gnode.Spec
	for _, d := range s.Decls {
		var vals []gnode.Expr
		if d.Init != nil {
			vals = append(vals, d.Init)
		}
		specs = append(specs, gnode.NewValueSpec(
			[]*gnode.IdentExpr{gnode.EIdent(d.Name, s.Pos())},
			vals,
		))
	}
	return gnode.Stmts{
		gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Var,
			TokPos: s.Pos(),
			Lparen: s.Pos(),
			Rparen: s.End(),
			Specs:  specs,
		}),
	}
}

func convertConst(s *ConstStmt) gnode.Stmts {
	if s.Decl != nil {
		return gnode.Stmts{gnode.SDecl(s.Decl)}
	}
	var specs []gnode.Spec
	for _, d := range s.Decls {
		var vals []gnode.Expr
		if d.Init != nil {
			vals = append(vals, d.Init)
		}
		specs = append(specs, gnode.NewValueSpec(
			[]*gnode.IdentExpr{gnode.EIdent(d.Name, s.Pos())},
			vals,
		))
	}
	return gnode.Stmts{
		gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Const,
			TokPos: s.Pos(),
			Lparen: s.Pos(),
			Rparen: s.End(),
			Specs:  specs,
		}),
	}
}

// convertEnum lowers an `@enum` directive to its Gad `enum IDENT { … }`
// statement (already parsed by the giom parser).
func convertEnum(s *EnumStmt) gnode.Stmts {
	if s.Decl == nil {
		return nil
	}
	return gnode.Stmts{s.Decl}
}

func convertGlobal(s *GlobalStmt) gnode.Stmts {
	if s.Decl != nil {
		return gnode.Stmts{gnode.SDecl(s.Decl)}
	}
	var specs []gnode.Spec
	for _, name := range s.Names {
		specs = append(specs, gnode.NewParamSpec(false,
			&gnode.TypedIdentExpr{Ident: gnode.EIdent(name, s.Pos())},
		))
	}
	return gnode.Stmts{
		gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Global,
			TokPos: s.Pos(),
			Specs:  specs,
		}),
	}
}

func slotVarName(id string) string     { return "$slot$" + id }
func slotDefaultName(id string) string { return "$slot$" + id + "$" }

// slotScopeArgs forwards a slot's scope parameters as call arguments, passing
// each by its own name so slot content receives the surrounding component's
// values (Vue-style scoped slots).
func slotScopeArgs(scope *gnode.FuncParams) (pos gnode.CallExprPositionalArgs, named gnode.CallExprNamedArgs) {
	if scope == nil {
		return
	}
	for _, a := range scope.Args.Values {
		if a != nil && a.Ident != nil {
			pos.Values = append(pos.Values, gnode.EIdent(a.Ident.Name, 0))
		}
	}
	for _, n := range scope.NamedArgs.Names {
		if n != nil && n.Ident != nil {
			named.AppendS(n.Ident.Name, gnode.EIdent(n.Ident.Name, 0))
		}
	}
	return
}

// slotDefaultParams returns the parameters for a slot's default function: a
// leading `super` positional parameter followed by its scope parameters.
func slotDefaultParams(scope *gnode.FuncParams) *gnode.FuncParams {
	out := &gnode.FuncParams{}
	if scope != nil {
		out.Args = scope.Args
		out.NamedArgs.Var = scope.NamedArgs.Var
		out.NamedArgs.Names = append([]*gnode.TypedIdentExpr{}, scope.NamedArgs.Names...)
		out.NamedArgs.Values = append([]gnode.Expr{}, scope.NamedArgs.Values...)
	}
	return withSuperParam(out)
}

// withSuperParam prepends a `super` positional parameter unless the first
// positional parameter is already named `super`. `super` is auto-injected so a
// slot override can render the slot's default content by calling `super(…)`.
func withSuperParam(params *gnode.FuncParams) *gnode.FuncParams {
	if params == nil {
		params = &gnode.FuncParams{}
	}
	if len(params.Args.Values) > 0 {
		if first := params.Args.Values[0]; first != nil && first.Ident != nil && first.Ident.Name == "super" {
			return params
		}
	}
	params.Args.PrependValue(&gnode.TypedIdentExpr{Ident: gnode.EIdent("super", 0)})
	return params
}

// emptySuperFunc builds a variadic no-op function used as the `super` value for
// optional slots (those without default content), so calling `super(…)` from an
// override is always safe and renders nothing.
func emptySuperFunc(pos, end source.Pos) *gnode.FuncExpr {
	params := &gnode.FuncParams{Args: gnode.ArgsList{Var: &gnode.TypedIdentExpr{Ident: gnode.EIdent("_", pos)}}}
	return funcExpr(params, nil, pos, end)
}

// convertSlot compiles an `@slot` declaration. `super` is always the resolved
// slot function's first positional argument, so an overriding slot may render
// the fallback by calling `super(…)`.
//
// A slot with default content compiles to a default function, a
// `var $slot$ID = (slots.ID ?? $slot$ID$)` binding and a call passing the
// default function `$slot$ID$` as `super`. A slot with no default content
// compiles to a nullish call `slots.ID?.(superEmpty, scope…)` (so it renders
// only when provided), passing an empty-body function as `super`.
func convertSlot(s *SlotDecl) gnode.Stmts {
	var slotsSel gnode.Expr
	if s.NameExpr != nil {
		// Interpolated name: `slots[<nameExpr>]`.
		slotsSel = &gnode.IndexExpr{X: gnode.EIdent("slots", s.Pos()), Index: s.NameExpr}
	} else {
		slotsSel = gnode.ESelector(gnode.EIdent("slots", s.Pos()), gnode.Str(s.ID, s.Pos()))
	}
	posArgs, namedArgs := slotScopeArgs(s.Scope)

	if len(s.Body) == 0 {
		call := &gnode.NullishCallExpr{Func: slotsSel}
		call.Args = posArgs
		call.Args.Values = append([]gnode.Expr{emptySuperFunc(s.Pos(), s.End())}, call.Args.Values...)
		call.NamedArgs = namedArgs
		return gnode.Stmts{appendToTag(call, s.Pos())}
	}

	defName := slotDefaultName(s.ID)
	varName := slotVarName(s.ID)

	defFunc := &gnode.FuncExpr{
		Type: funcType(slotDefaultParams(s.Scope)),
		Body: gnode.SBlock(s.Pos(), s.End(),
			fragmentStmts(gnode.LNil(s.Pos()), convertBody(s.Body), s.Pos(), s.End())...),
	}

	var stmts gnode.Stmts
	stmts.Append(gnode.SDecl(&gnode.GenDecl{
		Tok:    token.Const,
		TokPos: s.Pos(),
		Specs: []gnode.Spec{&gnode.ValueSpec{
			Idents: []*gnode.IdentExpr{gnode.EIdent(defName, s.Pos())},
			Values: []gnode.Expr{defFunc},
		}},
	}))
	stmts.Append(gnode.SDecl(&gnode.GenDecl{
		Tok:    token.Var,
		TokPos: s.Pos(),
		Specs: []gnode.Spec{&gnode.ValueSpec{
			Idents: []*gnode.IdentExpr{gnode.EIdent(varName, s.Pos())},
			Values: []gnode.Expr{gnode.EBinary(slotsSel, gnode.EIdent(defName, s.Pos()), token.Nullich, s.Pos())},
		}},
	}))
	call := &gnode.CallExpr{Func: gnode.EIdent(varName, s.Pos())}
	call.Args = posArgs
	call.Args.Values = append([]gnode.Expr{gnode.EIdent(defName, s.Pos())}, call.Args.Values...)
	call.NamedArgs = namedArgs
	stmts.Append(appendToTag(call, s.Pos()))
	return stmts
}

func convertSlotPass(s *SlotPassStmt) gnode.Stmts {
	return gnode.Stmts{
		gnode.SDecl(&gnode.GenDecl{
			Tok:    token.Const,
			TokPos: s.Pos(),
			Specs: []gnode.Spec{
				&gnode.ValueSpec{
					Idents: []*gnode.IdentExpr{gnode.EIdent("$slot", s.Pos())},
					Values: []gnode.Expr{
						&gnode.FuncExpr{
							Type: s.FuncType,
							Body: gnode.SBlock(s.Pos(), s.End(),
								fragmentStmts(gnode.LNil(s.Pos()), convertBody(s.Body), s.Pos(), s.End())...),
						},
					},
				},
			},
		}),
	}
}

func convertFor(f *ForStmt) gnode.Stmts {
	if arr, ok := f.Cond.(*gnode.ArrayExpr); ok && len(arr.Elements) == 2 {
		key, keyOK := arr.Elements[0].(*gnode.IdentExpr)
		bin, binOK := arr.Elements[1].(*gnode.BinaryExpr)
		if keyOK && binOK && bin.Token == token.In {
			if val, valOK := bin.LHS.(*gnode.IdentExpr); valOK {
				return gnode.Stmts{
					&gnode.ForInStmt{
						ForPos:   f.Pos(),
						Key:      key,
						Value:    val,
						Iterable: bin.RHS,
						Body:     gnode.SBlock(f.Pos(), f.End(), convertBody(f.Body)...),
					},
				}
			}
		}
	}
	if mp, ok := f.Cond.(*gnode.MultiParenExpr); ok && len(mp.PositionalElements) == 2 {
		key, keyOK := mp.PositionalElements[0].(*gnode.IdentExpr)
		bin, binOK := mp.PositionalElements[1].(*gnode.BinaryExpr)
		if keyOK && binOK && bin.Token == token.In {
			val, valOK := bin.LHS.(*gnode.IdentExpr)
			if !valOK {
				return gnode.Stmts{
					&gnode.ForStmt{
						ForPos: f.Pos(),
						Init:   f.Init,
						Cond:   f.Cond,
						Post:   f.Post,
						Body:   gnode.SBlock(f.Pos(), f.End(), convertBody(f.Body)...),
					},
				}
			}
			return gnode.Stmts{
				&gnode.ForInStmt{
					ForPos:   f.Pos(),
					Key:      key,
					Value:    val,
					Iterable: bin.RHS,
					Body:     gnode.SBlock(f.Pos(), f.End(), convertBody(f.Body)...),
				},
			}
		}
	}
	if bin, ok := f.Cond.(*gnode.BinaryExpr); ok && bin.Token == token.In {
		if val, ok := bin.LHS.(*gnode.IdentExpr); ok {
			return gnode.Stmts{
				&gnode.ForInStmt{
					ForPos:   f.Pos(),
					Key:      &gnode.IdentExpr{Name: "_", Empty: true},
					Value:    val,
					Iterable: bin.RHS,
					Body:     gnode.SBlock(f.Pos(), f.End(), convertBody(f.Body)...),
				},
			}
		}
	}
	return gnode.Stmts{
		&gnode.ForStmt{
			ForPos: f.Pos(),
			Init:   f.Init,
			Cond:   f.Cond,
			Post:   f.Post,
			Body:   gnode.SBlock(f.Pos(), f.End(), convertBody(f.Body)...),
		},
	}
}

func convertIf(s *IfStmt) gnode.Stmts {
	body := gnode.SBlock(s.Pos(), s.End(), convertBody(s.Body)...)
	var elseStmt gnode.Stmt
	if len(s.Else) > 0 {
		elseStmt = gnode.SBlock(s.Pos(), s.End(), convertBody(s.Else)...)
	}
	for i := len(s.ElseIfs) - 1; i >= 0; i-- {
		eif := s.ElseIfs[i]
		eifBody := gnode.SBlock(s.Pos(), s.End(), convertBody(eif.Body)...)
		elseStmt = &gnode.IfStmt{Cond: eif.Cond, Body: eifBody, Else: elseStmt}
	}
	ifStmt := &gnode.IfStmt{Cond: s.Cond, Body: body, Else: elseStmt}
	if s.Init != nil {
		ifStmt.Init = s.Init
	}
	return gnode.Stmts{ifStmt}
}

// convertTag lowers a tag to a block that binds `tag` to a new giom.Tag linked
// to the enclosing tag, then builds its body into it:
//
//	{ tag := giom.Tag(tag, "name"; **attrs); <body> }
func convertTag(t *TagStmt) gnode.Stmts {
	ctor := giomNew("Tag", t.NodePos, t.NodeEnd,
		tagIdent(t.NodePos), gnode.Str(t.Name, t.NodePos))
	applyTagAttrs(ctor, t.Attributes)

	inner := gnode.Stmts{defineTag(ctor, t.NodePos)}
	if !t.SelfClosing {
		inner = append(inner, convertBody(t.Body)...)
	}
	return gnode.Stmts{gnode.SBlock(t.Pos(), t.End(), inner...)}
}

// applyTagAttrs adds a tag's attributes as named arguments of the giom.Tag call,
// expanding `**attrs`-style groups into individual name=value pairs. giom.Tag
// classifies them into regular attributes, class list and styles.
func applyTagAttrs(call *gnode.CallExpr, attrs []*TagAttribute) {
	for _, attr := range attrs {
		if attr == nil {
			continue
		}
		if attr.Elements != nil {
			for _, el := range attr.Elements.Elements {
				if kv, ok := el.(*gnode.KeyValuePairLit); ok {
					addNamedArg(call, exprKeyName(kv.Key), kv.Value)
				}
			}
			continue
		}
		value := attr.Value
		if value == nil {
			if attr.IsFlag {
				value = gnode.Str(attr.Name, 0)
			} else {
				value = gnode.Str("", 0)
			}
		}
		addNamedArg(call, attr.Name, value)
	}
}

// convertHtml lowers a raw HTML region to a giom.Text append of its literal and
// interpolated parts. The region's pre-lowered write statements are unwrapped
// into the values of a single giom.Text(tag, …) call.
func convertHtml(h *HtmlStmt) gnode.Stmts {
	if len(h.Stmts) == 0 {
		return nil
	}
	var (
		out    gnode.Stmts
		values []gnode.Expr
	)
	flush := func() {
		if len(values) == 0 {
			return
		}
		out.Append(gnode.SExpr(textCall(h.Pos(), h.End(), values...)))
		values = nil
	}
	for _, stmt := range h.Stmts {
		if arg := htmlWriteArg(stmt); arg != nil {
			values = append(values, arg)
			continue
		}
		flush()
		out.Append(stmt)
	}
	flush()
	return gnode.Stmts{gnode.SBlock(h.Pos(), h.End(), out...)}
}

// htmlWriteArg returns the written value of a pre-lowered html write statement
// (`write(x)` / `giom.write(x)`), or nil when stmt is not such a call.
func htmlWriteArg(stmt gnode.Stmt) gnode.Expr {
	es, ok := stmt.(*gnode.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := es.Expr.(*gnode.CallExpr)
	if !ok || len(call.Args.Values) == 0 {
		return nil
	}
	return call.Args.Values[0]
}

// textCall builds `giom.Text(tag, values…)`.
func textCall(pos, end source.Pos, values ...gnode.Expr) *gnode.CallExpr {
	return giomNew("Text", pos, end, append([]gnode.Expr{tagIdent(pos)}, values...)...)
}

func convertDoctype(d *DoctypeStmt) gnode.Stmts {
	raw := gnode.EToRaw(0, gnode.Str(doctypeValue(d.Value), 0))
	return gnode.Stmts{gnode.SExpr(textCall(d.NodePos, d.NodeEnd, raw))}
}

// convertText lowers text content to giom.Text appends: consecutive literal and
// interpolation segments coalesce into a single giom.Text(tag, …) call, while
// any interleaved statement is emitted as-is.
func convertText(t *TextStmt) gnode.Stmts {
	var (
		out    gnode.Stmts
		values []gnode.Expr
	)
	flush := func() {
		if len(values) == 0 {
			return
		}
		out.Append(gnode.SExpr(textCall(t.NodePos, t.NodeEnd, values...)))
		values = nil
	}
	for _, stmt := range t.Stmts {
		switch s := stmt.(type) {
		case *gnode.MixedTextStmt:
			values = append(values, gnode.Str(s.Value(), s.Pos()))
		case *gnode.MixedValueStmt:
			values = append(values, s.Expr)
		case gnode.Stmt:
			flush()
			out.Append(s)
		}
	}
	flush()
	return out
}
