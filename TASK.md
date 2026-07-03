# IDE epic (`web/ide` backend + `web/app` React frontend)


# web/js projects

# Language

- [ ] change typed ident and param parser to parse param type of method interface. add exaplained tests and docs for it
  of funcs/closures/methods/funcHeaders/properties etc.... examples: `func x(cb meti{(int)<float>} ) {...}`, `met x(iOrCb int|meti{(int)<float>}) {...}`,
      STAGE 1 (parser) done (commit a95881c): the type parser accepts `meti{…}` /
      `interface{…}` / `met<…>` structural literals wherever a type is read (typed
      idents, params, func-headers, unions); isTypeStart + parseType handle the
      literal forms; parser tests (TestParseTypeMethodInterface). Compiling such a
      type is NOT yet supported — nameSymbolsOfTypedIdent returns a clear error
      (was a nil-deref panic).
      REMAINING (runtime, a central type-system refactor — agreed design):
      introduce `gad.TypeAssigner { Object; AssignTo(*VM, obj Object, to
      TypeAssigner) (Object, error) }` and `TypeAssigners []TypeAssigner`
      (rename Cast->Assign / CastTo->AssignTo). All ObjectType impls (~9: Type,
      *BuiltinObjType, BuiltinObjTypeKey, *Enum, Class, ReflectType, the operator
      types), *Interface and *MethodInterface implement AssignTo (returns obj when
      it assigns, else ErrIncompatibleCast). Add *Class.AssignTo checking
      the instance (obj) + parents. Change
      MethodArgType.add param `types` and TypedCallerMethod.types from
      ObjectTypeArray to TypeAssigners, and wire AssignTo into the dispatch match.
      IsTypeAssignableTo uses AssignTo. Then compile a meti/interface literal type
      to a constant (a ScopeConstant symbol) so a param can reference it. This
      touches the method-dispatch core — do it carefully in one focused pass.
      STAGE 2 (runtime dispatch) done: structural (meti) param types now dispatch
      by value. Design: the dispatch tree stays ObjectType-keyed (structural param
      keys as TAny → zero overhead on the hot path), but a method carrying a
      structural param sets TypedCallerMethod.forceValidate (computed once at
      registration via CompiledFunction.HasStructuralParamTypes, memoized with an
      atomic int32 on the shared function constant). On an exact tree match, if
      forceValidate is set, value-based validation runs (ParamType.Accept →
      vmCanAssigner.CanAssignVM → MethodInterfaceImplements). The tree key type is
      now TypeAssigner (foundation for structural keys). Evidence:
        `go test ./...` → all ok (exit 0)
        `go test -run TestVMMethodInterface` → PASS (new cases #12 accept callable,
          #13 reject `x(42)` with "invalid type for argument")
        `go vet ./...` → clean; `go test -race` on VM/Eval → ok
      REMAINING: *Interface (not meti) structural param CanAssign is still
      equal-only (fields/props/methods check TODO); the TASK NOTES refactor of the
      ParamTypes interface (Items/Get → TypeAssigner) is not done — dispatch works
      without it via the forceValidate path.
  NOTES:
  - change ParamTypes interface methods `Items() ObjectTypes` to  `Items() TypeAssigners`, `Get(int) ObjectType` to `Get(int) TypeAssigner`
  - take all ObjectType to implements TypeAssigner method `CanAssign(obj Object) (bool, error) { return obj.Type() == this }` (default, if not implemented)
  - replace ParamType.Accept param `ot` to `obj Object`
  - replace MethodArgType.GetMethod param `types` to `types TypeAssignerArray`
- [x] check if method has `$old` first param. in this case, create a scope variable `$old` to get current method before override it
  and must compile method without this param. like `x(i int) => i*10; met ~x($old, i int) { return $old($i*2) }`, compiles to
  `const $old = gad.methodFromArgs(x, *args); met ~x(i int) {return $old($i*2)}`. create tests, samples and docs.
      DONE. New builtin `gad.methodFromArgs(target, ...args)` (BuiltinMethodFromArgs)
      resolves the method a call would dispatch to (each arg is a value → its type,
      or an ObjectType → used directly); returns nil if none. Compiler
      (compileMethodExpr): a method whose first positional param is `$old` is
      desugared in its own block scope to
        `$old := gad.methodFromArgs(<target>, <one type per real param>)`
      then the method is added with `$old` stripped; its body closes over `$old`
      (captured before the override runs, so no self-recursion). Overrides chain;
      `$old` is nil when no prior method matches. Works for plain funcs, operator
      selector targets (`met ~gad.binOpMul($old,…)`), block form, and untyped
      params (uses `any`). The formatter preserves `$old` (it is a normal param in
      the AST; only stripped at compile, restored via defer). Evidence:
        `go test ./...` → ok (exit 0); `go vet ./...` → clean
        `go test -run TestVMOldOverrideParam` → PASS (wrap=31, chain=111,
          multi=14, untyped, nil-when-absent, methodFromArgs by value+type)
        samples/25_method_resolution.gad runs; doc/functions.md "Overriding and
          `$old`" section added.
- [ ] parser operator AssignTo `obj :: Type`. compile to OpAssign. it calls assign like method resolution. return an error
      if not assignable.
- [x] update 11_classes.gad to add `$old` examples rewriting methods, constructors
      and property setters (the concrete goal behind the constructor/resolver
      snippets). DONE:
      - `*Class` and `*ClassProperty` now implement MethodCaller by delegating to
        their FuncSpec (`Class.new.f` / `ClassProperty.f`); ClassMethod/
        ClassProperty/ClassConstructor expose GetFuncSpec. So `$old`
        (gad.methodFromArgs) resolves class methods, constructors and property
        setters.
      - Fixed Class.AddMethodIndex: `met Class.NAME(...)` now routes to an existing
        property's getter/setter (was always adding a shadowing method, so setter
        overrides silently did nothing).
      - Renamed the class index selector `@properties` → `@props`.
      - samples/11_classes.gad gains a "rewriting members with `met ~` and `$old`"
        section (method → "Rex barks loudly", constructor → 30 40, setter →
        "int:9 (checked)"). doc/classes.md documents it.
      Evidence: `go test ./...` → ok; `go test -run TestVMClassOldOverride` → PASS.
- [ ] check cmd/update-*-plugin to accept all language changes.
    update vscode plugin to allow single "run" and "debug".
    create example page for codemirror and prismjs plugins.
