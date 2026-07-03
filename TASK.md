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
  NOTES:
  - change ParamTypes interface methods `Items() ObjectTypes` to  `Items() TypeAssigners`, `Get(int) ObjectType` to `Get(int) TypeAssigner`
  - take all ObjectType to implements TypeAssigner method `CanAssign(obj Object) (bool, error) { return obj.Type() == this }` (default, if not implemented)
  - replace ParamType.Accept param `ot` to `obj Object`
  - replace MethodArgType.GetMethod param `types` to `types TypeAssignerArray`
- [x] change parser of `met<...>` to allow multiples headers `met<(int), (float)<str> [, ...]>`, when format,
    if has muliples headers, put it int new indented line without comma. parses allow multiples itens separated by new line without `,` (its optional, no required in this case).
      Done (commit 983d018): parseMetShortcut parses 1+ bracket-less headers
      between `<…>`, separated by commas or newlines (either optional, ExprLevel
      makes newlines skippable). WriteCode formats several headers one per indented
      line without commas (idempotent); a single header stays inline `met<(_ v)>`;
      String() keeps the compact comma form. Parser test extended; `go test ./...`
      -> 0 failures.
- [~] parser binary operator castTo `obj :: Type`. compile to OpCast
- [ ] check cmd/update-*-plugin to accept all language changes.
    update vscode plugin to allow single "run" and "debug".
    create example page for codemirror and prismjs plugins.
