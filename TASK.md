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
- [x] parser operator AssignTo `obj :: Type`. compile to OpAssign. it calls assign like method resolution. return an error
      if not assignable. allows `obj::Type1::Type2::Type3`. add samples, docs and tests.
      DONE. New `DoubleColon` token (`::`, scanned; placed after GroupKeywordEnd so
      no token value shifts; precedence 11 — tighter than arithmetic). Parses via
      the normal binary-expr loop (Precedence-driven). New opcode OpAssign (0
      operands; registered in MakeInstruction and both VM loops; excluded from the
      optimizer's constant-folding so a failing cast throws at runtime instead of
      becoming a compile error). Runtime: AssignToType(vm, obj, to) returns obj
      when assignable (ObjectType incl. *Class parent-walk, or structural
      TypeAssigner via CanAssign/CanAssignVM), else ErrIncompatibleAssign; a
      non-type RHS gives ErrType. Fixed a latent *Class.CanAssign bug (walked
      parents but never set ok). Formatter writes `a::B` tightly, dropping
      redundant chain parens (`(a::B)::C`→`a::B::C`) but keeping needed ones
      (`(2 + 3)::int`). Evidence: `go test ./...` → ok; `go vet` → clean;
        `go test -run TestVMAssignOperator` → PASS (cast, chain, precedence,
          subclass, structural, reject str::int and 42::met, non-type RHS)
        `go test -run TestFormatDoubleColon ./parser` → PASS
        samples/25_method_resolution.gad extended; doc/operators.md documents it.
- [ ] check cmd/update-*-plugin to accept all language changes.
    update vscode plugin to allow single "run" and "debug".
    create example page for codemirror and prismjs plugins.
