package node

import (
	"fmt"
	"strconv"
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Convert recursively converts gadx-specific AST nodes to GAD AST nodes,
// returning pure GAD statements suitable for Format.
// Consecutive const/var declarations are merged into grouped declarations.
func Convert(stmts gnode.Stmts) gnode.Stmts {
	var out gnode.Stmts
	for _, s := range stmts {
		out = append(out, convertStmt(s)...)
	}
	return mergeDecls(out)
}

// ConvertFile converts a whole gadx file's top-level statements, wrapping them
// so all render content (whether under @main or written bare at the top level)
// builds into a single root tag that the program returns. Declarations, imports
// and exports remain top-level statements between the root binding and the
// return.
func ConvertFile(stmts gnode.Stmts) gnode.Stmts {
	return fragmentStmts(Convert(stmts), 0, 0)
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

// tagVar is the identifier that always names the current parent node in scope
// (a tag or the root Elements fragment). Each tag/fragment opens a block that
// rebinds `$el` (via `:=`) to itself, so nested content appends to it while
// sibling content sees the outer node. The `$` prefix keeps it out of the user
// identifier space, so a component parameter named `tag` no longer shadows it
// (which previously sent rendering into infinite recursion).
const tagVar = "$el"

func tagIdent(pos source.Pos) *gnode.IdentExpr { return gnode.EIdent(tagVar, pos) }

// gadxNew builds a `gadx.<ctor>(args…)` constructor call.
func gadxNew(ctor string, pos, end source.Pos, args ...gnode.Expr) *gnode.CallExpr {
	call := gadxCallExpr(ctor, pos)
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

// fragmentStmts wraps body so it builds into a fresh Elements fragment and
// returns it: `tag := gadx.Elements(); <body>; return tag`. Elements is a
// wrapper-less list (renders only its children); it has no parent — nested tags
// take it as THEIR parent and append into it, and appending one fragment to
// another splices its items. Used for @comp / @func / @slot / slot-pass bodies,
// @main and the file root.
func fragmentStmts(body gnode.Stmts, pos, end source.Pos) gnode.Stmts {
	var out gnode.Stmts
	out.Append(defineTag(gadxNew("Elements", pos, end), pos))
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
	case *ParamStmt:
		return convertParam(st)
	case *EnumStmt:
		return convertEnum(st)
	case *ExportStmt:
		return convertExport(st)
	case *TestDecl:
		return convertTestDecl(st)
	case *CallLineStmt:
		return convertCallLine(st)
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
	case *HTMLCommentStmt:
		return convertHTMLComment(st)
	case *TextStmt:
		return convertText(st)
	case *TagStmt:
		return convertTag(st)
	case *HTMLStmt:
		return convertHTML(st)
	case *MdBlockStmt:
		// Lower `@md` here (in the Convert pass) rather than lazily in WriteCode,
		// so the resulting gadx.Tag/gadx.Text nodes — and the original
		// interpolation expressions swapped into them — are compiled directly and
		// keep their source positions, instead of going through the compiler's
		// serialize-and-reparse fallback (which would drop them).
		return convertMdBlock(st)
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
	body := fragmentStmts(convertBody(f.Body), f.Pos(), f.End())
	fn := funcExpr(params, body, f.Pos(), f.End())
	fn.Type.TypeParams = f.TypeParams
	fn.Type.Return = f.Return
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

	fnBody := fragmentStmts(body, c.Pos(), c.End())
	fn := funcExpr(addSlotsParam(c.Params), fnBody, c.Pos(), c.End())
	fn.Type.TypeParams = c.TypeParams
	fn.Type.Return = c.Return

	if c.Main {
		// `@main` is sugar for `@comp main`: it lowers to a `main` component
		// function whose parameters are its own (with defaults) — NOT module
		// globals or params (use `@global` / `@param` for those) — and is then
		// invoked so its Elements build into the enclosing root fragment.
		stmts := recursiveFuncStmts("main", fn, c.Pos())
		mainCall := gnode.ECall(gnode.EIdent("main", c.Pos()), c.Pos(), c.End(),
			gnode.NewCallExprArgs(nil))
		stmts = append(stmts, appendToTag(mainCall, c.Pos()))
		return stmts
	}

	stmts := recursiveFuncStmts(c.ID, fn, c.Pos())
	if c.Exported {
		stmts = append(stmts, &gnode.ExportStmt{
			TokenPos: c.Pos(),
			KeyExpr:  gnode.EIdent(c.ID, c.Pos()),
		})
	}
	return stmts
}

// convertTestDecl lowers a `@test NAME` block to a Gad `test NAME { … }`
// statement (discovered by the `gad test` runner, with an injected `t`). The
// body is lowered like a component fragment, so template content builds an
// implicit `tag` while `~` code can assert with `t`.
func convertTestDecl(t *TestDecl) gnode.Stmts {
	body := fragmentStmts(convertBody(t.Body), t.Pos(), t.End())
	return gnode.Stmts{&gnode.TestStmt{
		Kind:   gnode.TestKindTest,
		KwPos:  t.Pos(),
		Name:   t.Name,
		Quoted: t.Quoted,
		Body:   gnode.SBlock(t.Pos(), t.End(), body...),
	}}
}

// convertCallLine lowers `! callee arg1 arg2 …` to the call statement
// `callee(arg1, arg2, …)`.
func convertCallLine(s *CallLineStmt) gnode.Stmts {
	call := gnode.ECall(s.Callee, s.Pos(), s.End(), gnode.NewCallExprArgs(nil, s.Args...))
	return gnode.Stmts{gnode.SExpr(call)}
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
								fragmentStmts(convertBody(sp.Body), sp.Pos(), sp.End())...),
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
// statement (already parsed by the gadx parser).
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

// convertParam lowers @param to its Gad `param (…)` declaration.
func convertParam(s *ParamStmt) gnode.Stmts {
	if s.Decl == nil {
		return nil
	}
	return gnode.Stmts{gnode.SDecl(s.Decl)}
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
			fragmentStmts(convertBody(s.Body), s.Pos(), s.End())...),
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
								fragmentStmts(convertBody(s.Body), s.Pos(), s.End())...),
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

// convertTag lowers a tag to a block that binds `tag` to a new gadx.Tag linked
// to the enclosing tag, then builds its body into it:
//
//	{ tag := gadx.Tag(tag, "name"; **attrs); <body> }
//
// A `<>…</>` fragment (Fragment) lowers to a gadx.Elements() node instead (see
// convertFragmentTag).
func convertTag(t *TagStmt) gnode.Stmts {
	if t.Fragment {
		return convertFragmentTag(t)
	}
	ctor := gadxNew("Tag", t.NodePos, t.NodeEnd,
		tagIdent(t.NodePos), gnode.Str(t.Name, t.NodePos))
	applyTagAttrs(ctor, t.Attributes)

	inner := gnode.Stmts{defineTag(ctor, t.NodePos)}
	if !t.SelfClosing {
		inner = append(inner, convertBody(t.Body)...)
	}
	return gnode.Stmts{gnode.SBlock(t.Pos(), t.End(), inner...)}
}

// convertFragmentTag lowers a `<>…</>` fragment to a gadx.Elements() node built
// from its children, then spliced into the enclosing parent:
//
//	{ $frag := tag; tag := gadx.Elements(); <body>; $frag += tag }
//
// The enclosing `tag` is captured first (Elements has no parent argument), then
// `tag` is rebound to the fresh fragment so the body builds into it, and finally
// the fragment is appended to the captured parent — which splices its children.
func convertFragmentTag(t *TagStmt) gnode.Stmts {
	parentName := fmt.Sprintf("$frag%d", t.NodePos)
	parentIdent := func() *gnode.IdentExpr { return gnode.EIdent(parentName, t.NodePos) }

	var inner gnode.Stmts
	inner.Append(&gnode.AssignStmt{
		LHS: []gnode.Expr{parentIdent()}, RHS: []gnode.Expr{tagIdent(t.NodePos)},
		Token: token.Define, TokenPos: t.NodePos,
	})
	inner.Append(defineTag(gadxNew("Elements", t.NodePos, t.NodeEnd), t.NodePos))
	inner.Append(convertBody(t.Body)...)
	inner.Append(&gnode.AssignStmt{
		LHS: []gnode.Expr{parentIdent()}, RHS: []gnode.Expr{tagIdent(t.NodePos)},
		Token: token.AddAssign, TokenPos: t.NodePos,
	})
	return gnode.Stmts{gnode.SBlock(t.Pos(), t.End(), inner...)}
}

// applyTagAttrs adds a tag's attributes as named arguments of the gadx.Tag call,
// expanding `**attrs`-style groups into individual name=value pairs. gadx.Tag
// classifies them into regular attributes, class list and styles.
func applyTagAttrs(call *gnode.CallExpr, attrs []*TagAttribute) {
	for _, attr := range attrs {
		if attr == nil {
			continue
		}
		if attr.Spread != nil {
			// `**expr` attribute spread (a computed/interpolated-name attribute).
			call.NamedArgs.Append(&gnode.NamedArgExpr{Var: true, Exp: attr.Spread}, nil)
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
				// A valueless attribute `[x]` is the flag `yes` (like a named
				// param `fn(;x)`): it renders as a bare boolean attribute `x`
				// (no value). `[x=no]` omits it. Distinct from `[x=true]`,
				// which renders the literal value `x="true"`.
				value = gnode.Flag(true, 0)
			} else {
				value = gnode.Str("", 0)
			}
		}
		addNamedArg(call, attr.Name, value)
	}
}

// convertHTML lowers an HTML region by converting its parsed gadx child nodes
// (TagStmt / TextStmt / …) the same way as any other gadx body, so the HTML
// builds gadx.Tag / gadx.Text elements linked to the enclosing tag rather than
// writing raw markup.
func convertHTML(h *HTMLStmt) gnode.Stmts {
	if len(h.Children) == 0 {
		return nil
	}
	return convertBody(h.Children)
}

// textCall builds `gadx.Text(tag, values…)`.
func textCall(pos, end source.Pos, values ...gnode.Expr) *gnode.CallExpr {
	return gadxNew("Text", pos, end, append([]gnode.Expr{tagIdent(pos)}, values...)...)
}

// convertTextBlock lowers an `@text` block: every source line becomes a literal
// gadx.Text append, and consecutive lines are separated by a newline write so the
// original line breaks are preserved. Interpolation (`{ … }`) inside a line keeps
// its source position because each line reuses convertText.
// convertRawTextBlock lowers a `@raw_text` block: its body is already a run of
// raw-string literals and `#{= … }#` values, so it lowers as one text node,
// exactly like the content of a script or a stylesheet.
func convertRawTextBlock(t *RawTextBlockStmt) gnode.Stmts {
	if len(t.Body) == 0 {
		return nil
	}
	return convertText(&TextStmt{NodePos: t.NodePos, NodeEnd: t.NodeEnd, Stmts: t.Body})
}

func convertTextBlock(t *TextBlockStmt) gnode.Stmts {
	// The folded style (`|>`, YAML `>`) joins lines with a space; the literal
	// style (`|` / `@text`) preserves line breaks.
	sep := "\n"
	if t.Fold {
		sep = " "
	}
	var a textAccum
	for i, stmt := range t.Body {
		if i > 0 {
			a.sep(sep, t.NodePos)
		}
		if ts, ok := stmt.(*TextStmt); ok {
			a.addText(ts)
			continue
		}
		a.flush()
		a.out.Append(stmt)
	}
	a.flush()
	return a.out
}

// convertParaBlock lowers an `@p` block: runs of consecutive non-blank lines
// become a `<p>` element (their text joined by newlines), and blank lines break
// paragraphs. Interpolation inside a line keeps its source position via
// convertText.
func convertParaBlock(t *ParaBlockStmt) gnode.Stmts {
	var (
		out  gnode.Stmts
		para []*TextStmt
	)
	flush := func() {
		if len(para) == 0 {
			return
		}
		ctor := gadxNew("Tag", t.NodePos, t.NodeEnd, tagIdent(t.NodePos), gnode.Str("p", t.NodePos))
		var a textAccum
		a.out = gnode.Stmts{defineTag(ctor, t.NodePos)}
		for i, ts := range para {
			if i > 0 {
				a.sep("\n", t.NodePos)
			}
			a.addText(ts)
		}
		a.flush()
		out.Append(gnode.SBlock(t.NodePos, t.NodeEnd, a.out...))
		para = nil
	}
	for _, stmt := range t.Body {
		ts, ok := stmt.(*TextStmt)
		if !ok {
			continue
		}
		if len(ts.Stmts) == 0 { // blank line → paragraph break
			flush()
			continue
		}
		para = append(para, ts)
	}
	flush()
	return out
}

// convertMdBlock lowers an `@md` block. The preferred lowering renders the
// Markdown to HTML at compile/transpile time and parses that HTML into
// gadx.Tag/gadx.Text nodes (an HTMLStmt), so `@md` produces a real tag tree.
// Each run of Markdown text between nested `@` directives is a section:
// interpolations (`{= … }`) are protected through the Markdown conversion and
// re-emerge as HTML `{ … }` interpolations, so dynamic values are inserted into
// the fixed HTML structure. Nested `@` directives render inline as their own
// gadx nodes. When the renderer/HTML-parser hooks are not installed (gadx or its
// parser not imported), it falls back to the runtime gadx.Md container.
func convertMdBlock(m *MdBlockStmt) gnode.Stmts {
	if MarkdownRenderer != nil && HTMLToNodes != nil {
		if stmts, ok := convertMdViaHTML(m); ok {
			return stmts
		}
	}
	return convertMdBlockRuntime(m)
}

// convertMdBlockRuntime is the fallback lowering: an `@md` block becomes a
// `gadx.Md` container whose children are assembled as Markdown source and
// converted to HTML by goldmark at render time.
func convertMdBlockRuntime(m *MdBlockStmt) gnode.Stmts {
	ctor := gadxNew("Md", m.NodePos, m.NodeEnd, tagIdent(m.NodePos))
	inner := gnode.Stmts{defineTag(ctor, m.NodePos)}
	nl := func(s string) {
		inner.Append(gnode.SExpr(textCall(m.NodePos, m.NodePos, gnode.Str(s, m.NodePos))))
	}
	for i, stmt := range m.Body {
		if i > 0 {
			// A nested `@` directive renders to an HTML block; a raw-HTML block in
			// Markdown ends at a blank line, so surround directive output with blank
			// lines to keep it a standalone block (and not swallow following text).
			if _, prevText := m.Body[i-1].(*TextStmt); isMdText(stmt) && prevText {
				nl("\n")
			} else {
				nl("\n\n")
			}
		}
		inner = append(inner, convertStmt(stmt)...)
	}
	return gnode.Stmts{gnode.SBlock(m.Pos(), m.End(), inner...)}
}

// isMdText reports whether an `@md` body item is a plain Markdown text line (as
// opposed to a nested `@` directive that renders to an HTML block).
func isMdText(stmt gnode.Stmt) bool {
	_, ok := stmt.(*TextStmt)
	return ok
}

// MarkdownRenderer converts Markdown source to an HTML fragment. The gadx
// package installs it (goldmark, via the customizable gadx.Markdown) in an init;
// gadx/node cannot import gadx directly (that would be an import cycle).
var MarkdownRenderer func(src []byte) ([]byte, error)

// HTMLToNodes parses a raw HTML fragment into gadx Tag/Text nodes (with `{ … }`
// interpolations preserved), anchoring node positions at base. The gadx parser
// installs it (buildHTMLNodes) in an init; gadx/node cannot import gadx/parser
// directly (import cycle).
var HTMLToNodes func(html string, base source.Pos) (gnode.Stmts, error)

// An interpolation placeholder in the Markdown source fed to the renderer is
// built entirely from Unicode Private-Use characters: mdInterpOpen/mdInterpClose
// delimit it and mdInterpDigit0+d encodes each decimal digit of the index. The
// Markdown converter passes these through verbatim (they are neither Markdown nor
// HTML syntax) so the interpolation survives conversion, yet goldmark's
// auto-heading-id slugifier drops them - keeping generated ids clean - and they
// are restored to an HTML `{ ... }` interpolation afterwards.
const (
	mdInterpOpen   = "\uE000"
	mdInterpClose  = "\uE001"
	mdInterpDigit0 = 0xE010
)

func mdInterpSentinel(n int) string {
	var b strings.Builder
	b.WriteString(mdInterpOpen)
	for _, d := range strconv.Itoa(n) {
		b.WriteRune(rune(mdInterpDigit0 + (d - '0')))
	}
	b.WriteString(mdInterpClose)
	return b.String()
}

// convertMdViaHTML lowers an `@md` block by rendering its Markdown sections to
// HTML and parsing the HTML into gadx nodes (see convertMdBlock). It returns
// false to fall back to the runtime container when a section cannot be lowered
// (e.g. an embedded statement, or the renderer/HTML parser fails).
func convertMdViaHTML(m *MdBlockStmt) (gnode.Stmts, bool) {
	var out gnode.Stmts
	i := 0
	for i < len(m.Body) {
		if _, ok := m.Body[i].(*TextStmt); ok {
			// Collect a maximal run of consecutive Markdown text lines.
			j := i
			var section []*TextStmt
			for j < len(m.Body) {
				ts, ok := m.Body[j].(*TextStmt)
				if !ok {
					break
				}
				section = append(section, ts)
				j++
			}
			stmts, ok := convertMdTextSection(m, section)
			if !ok {
				return nil, false
			}
			out = append(out, stmts...)
			i = j
			continue
		}
		// A nested `@` directive renders inline as its own gadx nodes.
		out = append(out, convertStmt(m.Body[i])...)
		i++
	}
	return out, true
}

// convertMdTextSection renders one run of Markdown text lines to HTML and parses
// it into gadx nodes. Interpolations are replaced with placeholders before the
// conversion and restored as HTML `{ … }` interpolations afterwards, so dynamic
// values are inserted into the fixed HTML structure produced by the renderer.
func convertMdTextSection(m *MdBlockStmt, section []*TextStmt) (gnode.Stmts, bool) {
	var (
		src   strings.Builder
		exprs []gnode.Expr
	)
	for li, ts := range section {
		if li > 0 {
			src.WriteString("\n")
		}
		for _, inner := range ts.Stmts {
			switch s := inner.(type) {
			case *gnode.MixedTextStmt:
				src.WriteString(s.Value())
			case *gnode.MixedValueStmt:
				src.WriteString(mdInterpSentinel(len(exprs)))
				exprs = append(exprs, s.Expr)
			default:
				return nil, false // an embedded statement: fall back to the runtime lowering
			}
		}
	}

	html, err := MarkdownRenderer([]byte(src.String()))
	if err != nil {
		return nil, false
	}

	// Escape backslashes and literal braces in the rendered HTML so the HTML
	// parser treats content braces as literal text (not interpolation). Each
	// interpolation placeholder — which contains no braces — is then restored as
	// an HTML `{N}` interpolation whose index selects the original expression.
	out := string(html)
	out = strings.ReplaceAll(out, "\\", "\\\\")
	out = strings.ReplaceAll(out, "{", "\\{")
	out = strings.ReplaceAll(out, "}", "\\}")
	for n := len(exprs) - 1; n >= 0; n-- {
		// `{= N }` (an emitting interpolation): an `@md` interpolation always
		// outputs its value, and after the semantics change a bare `{ N }` would be
		// a no-output control statement.
		out = strings.ReplaceAll(out, mdInterpSentinel(n), "{="+strconv.Itoa(n)+"}")
	}

	// Anchor synthetic node positions at the `@md` block so diagnostics land in
	// the file near the block rather than at its start.
	nodes, err := HTMLToNodes(out, m.NodePos)
	if err != nil {
		return nil, false
	}
	// Replace each `{N}` placeholder expression with the original interpolation
	// expression, so runtime errors and debug stepping keep the source positions
	// of the `.gadx` file (the placeholder int literal carries only the index).
	replaceMdInterpPlaceholders(nodes, exprs)
	return convertHTML(&HTMLStmt{NodePos: m.NodePos, NodeEnd: m.NodeEnd, Children: nodes}), true
}

// replaceMdInterpPlaceholders walks HTML-parsed nodes and swaps each `{N}`
// placeholder interpolation (an IntLit index) for the original expression
// exprs[N], preserving its source positions in the AST and the bytecode source
// map. It covers the node shapes buildHTMLNodes emits: TextStmt interpolations,
// tag attribute values/conditions, and nested tag bodies.
func replaceMdInterpPlaceholders(stmts gnode.Stmts, exprs []gnode.Expr) {
	swap := func(e gnode.Expr) gnode.Expr {
		if lit, ok := e.(*gnode.IntLit); ok && lit.Value >= 0 && int(lit.Value) < len(exprs) {
			return exprs[lit.Value]
		}
		return e
	}
	for _, s := range stmts {
		switch n := s.(type) {
		case *TextStmt:
			for _, inner := range n.Stmts {
				if mv, ok := inner.(*gnode.MixedValueStmt); ok {
					mv.Expr = swap(mv.Expr)
				}
			}
		case *TagStmt:
			for _, a := range n.Attributes {
				if a.Value != nil {
					a.Value = swap(a.Value)
				}
				if a.Condition != nil {
					a.Condition = swap(a.Condition)
				}
			}
			replaceMdInterpPlaceholders(n.Body, exprs)
		}
	}
}

func convertDoctype(d *DoctypeStmt) gnode.Stmts {
	raw := gnode.EToRaw(0, gnode.Str(doctypeValue(d.Value), 0))
	return gnode.Stmts{gnode.SExpr(textCall(d.NodePos, d.NodeEnd, raw))}
}

// convertHTMLComment lowers an HTML comment to the text it writes. It goes out
// raw: escaped, `<!--` would reach the page as `&lt;!--` and the comment would
// be a line of visible text instead.
func convertHTMLComment(c *HTMLCommentStmt) gnode.Stmts {
	raw := gnode.EToRaw(0, gnode.Str(c.Source(), 0))
	return gnode.Stmts{gnode.SExpr(textCall(c.NodePos, c.NodeEnd, raw))}
}

// convertText lowers text content to gadx.Text appends: consecutive literal and
// interpolation segments coalesce into a single gadx.Text(tag, …) call, while
// any interleaved statement is emitted as-is.
func convertText(t *TextStmt) gnode.Stmts {
	var a textAccum
	a.addText(t)
	a.flush()
	return a.out
}

// textAccum coalesces consecutive text values (literal segments, `{= expr }`
// interpolations and inter-line separators) into a single `gadx.Text(tag, v1, v2,
// …)` call, so a block of text lines emits one call instead of one per segment or
// line. A non-text control statement (a bare `{ expr }`) flushes the pending
// values and is emitted on its own, then accumulation resumes.
type textAccum struct {
	out    gnode.Stmts
	values []gnode.Expr
	pos    source.Pos
	end    source.Pos
	hasPos bool
}

// sep appends a literal separator between text lines (e.g. "\n" or " ").
func (a *textAccum) sep(s string, pos source.Pos) {
	a.values = append(a.values, gnode.Str(s, pos))
}

// addText appends one text line's values, splitting the call around any embedded
// control statement.
func (a *textAccum) addText(t *TextStmt) {
	if !a.hasPos {
		a.pos, a.hasPos = t.NodePos, true
	}
	a.end = t.NodeEnd
	for _, stmt := range t.Stmts {
		switch s := stmt.(type) {
		case *gnode.MixedTextStmt:
			a.values = append(a.values, gnode.Str(s.Value(), s.Pos()))
		case *gnode.MixedValueStmt:
			a.values = append(a.values, s.Expr)
		case gnode.Stmt:
			a.flush()
			a.out.Append(s)
		}
	}
}

func (a *textAccum) flush() {
	if len(a.values) == 0 {
		return
	}
	a.out.Append(gnode.SExpr(textCall(a.pos, a.end, a.values...)))
	a.values = nil
}
