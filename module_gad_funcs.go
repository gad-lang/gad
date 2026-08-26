package gad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// SourceKind selects how source is parsed and compiled: plain Gad, a Gad
// template (mixed / `.gadt`) or a Gadx (`.gadx`) template. It is carried by
// SourceCode (module imports) and by gad.parse / gad.parseFile.
type SourceKind uint8

const (
	// SourceKindGad parses the input as ordinary Gad source.
	SourceKindGad SourceKind = iota
	// SourceKindGadt parses the input as a Gad template (mixed mode).
	SourceKindGadt
	// SourceKindGadx parses the input with the Gadx front-end.
	SourceKindGadx
)

// String returns the canonical upper-case dialect name: "GAD", "GADT" or "GADX".
func (k SourceKind) String() string {
	switch k {
	case SourceKindGadt:
		return "GADT"
	case SourceKindGadx:
		return "GADX"
	default:
		return "GAD"
	}
}

// Ext returns the file extension conventionally associated with the source kind.
func (k SourceKind) Ext() string {
	switch k {
	case SourceKindGadt:
		return ".gadt"
	case SourceKindGadx:
		return ".gadx"
	default:
		return ".gad"
	}
}

// SourceKindForExt maps a file extension (or path) to a SourceKind: `.gadx` ->
// Gadx, `.gadt` -> template, anything else -> plain Gad.
func SourceKindForExt(pth string) SourceKind { return sourceKindForExt(pth) }

// SourceCode is a unit of module source paired with the dialect it must be
// compiled as. An ExtImporter (e.g. importers.FileImporter) returns it from
// Import so the compiler parses the bytes with the right front-end — plain Gad,
// a `.gadt` mixed template, or a `.gadx` Gadx template. A bare []byte returned by
// an importer is treated as SourceKindGad for backward compatibility.
type SourceCode struct {
	Data []byte
	Kind SourceKind
}

// SourceTypeEnum is the builtin `enum SourceType (GAD, TEMPLATE, GADX)` exposed
// as gad.SourceType. Its members select the parse mode of gad.parse.
var SourceTypeEnum = newSourceTypeEnum()

func newSourceTypeEnum() *Enum {
	e := NewEnum("SourceType", gadModuleSpec)
	e.AddValue("GAD", Int(SourceKindGad))
	e.AddValue("TEMPLATE", Int(SourceKindGadt))
	e.AddValue("GADX", Int(SourceKindGadx))
	return e
}

// sourceKindFromArg reads a gad.SourceType member (or a plain int) into a
// SourceKind, defaulting to SourceKindGad when the argument is nil.
func sourceKindFromArg(o Object) (SourceKind, error) {
	switch v := o.(type) {
	case nil:
		return SourceKindGad, nil
	case *EnumValue:
		if iv, ok := v.Value.(Int); ok {
			return SourceKind(iv), nil
		}
	case Int:
		return SourceKind(v), nil
	case Uint:
		return SourceKind(v), nil
	}
	return SourceKindGad, NewArgumentTypeError("type", "SourceType", o.Type().Name())
}

// sourceKindForExt maps a file extension to a SourceKind: `.gadx` -> Gadx,
// `.gadt` -> template, anything else -> Gad.
func sourceKindForExt(pth string) SourceKind {
	switch {
	case strings.HasSuffix(pth, ".gadx"):
		return SourceKindGadx
	case strings.HasSuffix(pth, ".gadt"):
		return SourceKindGadt
	default:
		return SourceKindGad
	}
}

// parseToStmts parses src with the given source kind, returning the parsed Gad
// statements. name is used for error positions.
func parseToStmts(src []byte, name string, kind SourceKind) (StmtsObject, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AppendFileData(name, src)

	if kind == SourceKindGadx {
		pf, err := parseGadxFile(srcFile)
		if err != nil {
			return nil, err
		}
		return StmtsObject(pf.Stmts), nil
	}

	po := parser.ParserOptions{}
	so := parser.ScannerOptions{}
	if kind == SourceKindGadt {
		po.Mode |= parser.ParseMixed
		so.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
	}
	p := parser.NewParserWithOptions(srcFile, &po, &so)
	pf, err := p.ParseFile()
	if err != nil {
		return nil, err
	}
	return StmtsObject(pf.Stmts), nil
}

// randomSourceName returns a generated file name with the extension of kind,
// used as the SourceFileObject path when gad.parse is called without a name.
func randomSourceName(kind SourceKind) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "source" + kind.Ext()
	}
	return "source-" + hex.EncodeToString(b[:]) + kind.Ext()
}

// parseResult parses src and returns the SourceFileObject carrying both the raw
// source and its parsed statements (in the .Stmts field / `stmts` attribute).
func parseResult(path string, kind SourceKind, src []byte) (Object, error) {
	stmts, err := parseToStmts(src, path, kind)
	if err != nil {
		return nil, err
	}
	return &SourceFileObject{
		Path:   path,
		Kind:   kind,
		Source: append([]byte(nil), src...),
		Stmts:  stmts,
	}, nil
}

// gadParse implements
// gad.parse(script str; type gad.SourceType=gad.SourceType.GAD, name str="")
// -> SourceFileObject. The parse mode is the explicit `type` when given;
// otherwise it is inferred from `name`'s extension; otherwise Gad. The returned
// source file's path is `name` when given, else a generated name carrying the
// selected mode's extension. Its statements are available as `.stmts`.
func gadParse(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	name := ""
	if v := call.NamedArgs.GetValueOrNil("name"); v != nil {
		name = v.ToString()
	}

	var kind SourceKind
	if typeArg := call.NamedArgs.GetValueOrNil("type"); typeArg != nil {
		// Explicit type wins.
		k, err := sourceKindFromArg(typeArg)
		if err != nil {
			return nil, err
		}
		kind = k
	} else if name != "" {
		// Otherwise infer from the given name's extension.
		kind = sourceKindForExt(name)
	}

	if name == "" {
		name = randomSourceName(kind)
	}
	src := call.Args.GetOnly(0).ToString()
	return parseResult(name, kind, []byte(src))
}

// gadParseFile implements gad.parseFile(pth str) -> SourceFileObject: it reads
// the file at pth and parses it, selecting the source mode from the extension
// (`.gadx` -> Gadx, `.gadt` -> template, otherwise Gad). Its statements are
// available as `.stmts`. It returns an error when the file cannot be read or
// parsed.
func gadParseFile(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	pth := call.Args.GetOnly(0).ToString()
	data, err := os.ReadFile(pth)
	if err != nil {
		return nil, err
	}
	return parseResult(pth, sourceKindForExt(pth), data)
}

// evalStmts compiles the given statements against the running VM's builtins and
// executes them like Eval.Run, returning the last value produced. Statements
// parsed from Gadx reference the `gadx` builtins, which must be registered in
// the running VM. The result is returned as-is (a Gadx element is not rendered).
func evalStmts(vm *VM, stmts node.Stmts) (Object, error) {
	fileSet := source.NewFileSet()
	pf := &parser.File{InputFile: fileSet.AppendFileData(MainName, nil), Stmts: stmts}
	st := NewSymbolTable(vm.Builtins.builtins.NameSet)
	module := &ModuleSpec{ModuleInfo: ModuleInfo{Name: MainName}, Flags: ModuleMain}
	res, err := CompileFile(st, module, pf, CompileOptions{})
	if err != nil {
		return nil, err
	}
	ev := &Eval{
		VM:      NewVM(vm.Builtins, nil).SetRecover(true),
		RunOpts: &RunOpts{StdOut: vm.StdOut, StdErr: vm.StdErr},
	}
	return ev.Run(context.Background(), res.Bytecode)
}

// gadEvalStr implements gad.eval(source str; type gad.SourceType=GAD): it parses
// source in the selected mode and evaluates the resulting statements.
func gadEvalStr(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	kind, err := sourceKindFromArg(call.NamedArgs.GetValueOrNil("type"))
	if err != nil {
		return nil, err
	}
	stmts, err := parseToStmts([]byte(call.Args.GetOnly(0).ToString()), "(gad.eval)", kind)
	if err != nil {
		return nil, err
	}
	return evalStmts(call.VM, node.Stmts(stmts))
}

// gadEvalSourceFile implements gad.eval(sourceFile gad.SourceFileObject): it
// evaluates the source file's already-parsed statements.
func gadEvalSourceFile(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	sf := call.Args.GetOnly(0).(*SourceFileObject)
	return evalStmts(call.VM, node.Stmts(sf.Stmts))
}

// gadEvalStmts implements gad.eval(stmts gad.StmtsObject).
func gadEvalStmts(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	stmts := call.Args.GetOnly(0).(StmtsObject)
	return evalStmts(call.VM, node.Stmts(stmts))
}

// gadEvalStmt implements gad.eval(stmt gad.StmtObject): it evaluates the single
// statement.
func gadEvalStmt(call Call) (Object, error) {
	if err := call.Args.CheckLen(1); err != nil {
		return nil, err
	}
	stmt := call.Args.GetOnly(0).(*StmtObject)
	return evalStmts(call.VM, node.Stmts{stmt.Stmt})
}

// gadParseFn / gadParseFileFn / gadEvalFn are the callable objects registered in
// the `gad` namespace as gad.parse / gad.parseFile / gad.eval. They are built by
// buildGadNamespaceFuncs (from registerGadModule, i.e. package init time) rather
// than as plain var initializers, so the builtin type values they reference
// (TStr, SourceTypeEnum, StmtsType, …) are guaranteed to be initialized first.
var (
	gadParseFn     Object
	gadParseFileFn Object
	gadEvalFn      Object
	gadInvokerFn   Object
)

// gadInvoker implements gad.invoker(fn, args array; **nargs) -> invoker. It
// pre-resolves fn's overload for the parameter types held in args (each initial
// element is a type, or a sample value whose type is used) and returns a
// BoundInvoker bound to the SAME args array, so a hot loop can mutate the array
// in place and call the invoker with no per-call dispatch, allocation or
// validation. When fn has no overloads it is bound directly.
func gadInvoker(c Call) (Object, error) {
	fn := c.Args.Get(0)
	callee, ok := fn.(CallerObject)
	if !ok || !Callable(fn) {
		return nil, ErrNotCallable.NewError("1st argument (fn)")
	}
	arr, ok := c.Args.Get(1).(Array)
	if !ok {
		return nil, ErrType.NewError("2nd argument (args) must be an " + ReprQuote("array"))
	}
	if mc, ok := fn.(MethodCaller); ok && mc.HasCallerMethods() {
		types := make(ObjectTypeArray, len(arr))
		for i, e := range arr {
			if ot, isType := e.(ObjectType); isType {
				types[i] = ot
			} else {
				types[i] = e.Type()
			}
		}
		if m := mc.CallerMethodOfArgsTypes(types); m != nil {
			callee = m
		}
	}
	return NewBoundInvoker(callee, arr, c.NamedArgs.UnreadPairs()), nil
}

// buildGadNamespaceFuncs constructs gad.parse / gad.parseFile / gad.eval. gad.eval
// is a type-dispatched callable: its overloads accept a source string (with a
// `type` mode), a parsed SourceFileObject, a StmtsObject, or a single StmtObject.
func buildGadNamespaceFuncs() {
	gadParseFn = NewFunction("parse", gadParse,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("script").Type(TStr) }),
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
			np("type").Type(SourceTypeEnum)
			np("name").Type(TStr)
		}),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) {
			ret("_", SourceFileType)
		}),
	)
	gadParseFileFn = NewFunction("parseFile", gadParseFile,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("pth").Type(TStr) }),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) {
			ret("_", SourceFileType)
		}),
	)

	evalRet := FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) { ret("_", TAny) })
	evalStrFn := NewFunction("eval", gadEvalStr,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("source").Type(TStr) }),
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) { np("type").Type(SourceTypeEnum) }),
		evalRet,
	)
	evalSourceFileFn := NewFunction("eval", gadEvalSourceFile,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("sourceFile").Type(SourceFileType) }),
		evalRet,
	)
	evalStmtsFn := NewFunction("eval", gadEvalStmts,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("stmts").Type(StmtsType) }),
		evalRet,
	)
	evalStmtFn := NewFunction("eval", gadEvalStmt,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) { p("stmt").Type(StmtType) }),
		evalRet,
	)
	gadEvalFn = AddMethod(evalStrFn, evalSourceFileFn, evalStmtsFn, evalStmtFn)

	gadInvokerFn = NewFunction("invoker", gadInvoker,
		FunctionWithModule(gadModuleSpec),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("fn")
			p("args").Type(TArray)
		}),
		FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) { np("nargs").Var() }),
		FunctionWithReturnVars(func(ret func(name string, typ ...TypeAssigner)) {
			ret("ret", CallableInterface)
		}),
	)

	buildGadQuoteFuncs()
}
