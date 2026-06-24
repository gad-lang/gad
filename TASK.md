- [ ] gad codemirror plugin isn't working on ide. change ide files tree to allow rename file/dir with F2 or RIGHT CLICK
  MENU (with options: `run`, `format`, `transpile`
  (format with TranspileOptions for `.gad` and `.gadt` exts. add fields of config file key `transpile` (add to settings
  dialog)), and `remove`
  (with confirmation dialog - if is nom empty dir, add check field RECURSIVELY)). put run/debug options dialog into new
  separated dialog "run/debug" settings and put run/debu as tabs. split field "Save stdout+stderr to file", allowing
  set file for stdout and stderr, add new flag field for combine stdout and stderr. change ide to on click over
  breackpoint
  on line number panel, open brackpoint dialog with fields "disabled" (if is set, ignore debug this breackpoint), "
  condition"
  (with gad codemirror plugin, to specify and expression and pause no debug only `!value.IsFalsy()`) and cancel/save
  buttons; change brackpoints
  panel, to add right button to remove here, and when click over brackpoint entry open the brackpoint dialog, when click
  here with double click, goto location here. change ide to support tooltip for ident of builtin values.
  change ide to add panel "evaluate". this panel haves list of evaluated expressions. on fixed top of this panel, puts
  form
  for add new expression to evalue, with expression field, flag field "repr" and "+" button; when add, include here
  to list and evaluate returning result of "str(EXPRESSION)" if flag "repr" is set, replace "str" to "repr". each
  list entry, add right buttons "edit" (open into top form and change button to save icon). trash icon, to remove then;
  "output" button to open new dialog with result value as codemirror editor for plain text in readonly mode and "copy"
  button (must icon) to copy to clipboard. update evaluated expressions when debugger step changing.
  change ide file editor controls add button "reload" to reload file from disk. add header to explore three to add flag
  field to show/hide hidden files/dirs.
  add file editor support for JSON, YAML, HTML, CSS, SCSS, JS (with types script e JSX) and open other types to
  plain/text editor.
  change id to alert error in dialog when fail to request to backend. change ide explorer add button to open and dialog
  to get file from web
  and allow to change your output name and choose directory to save then (default is current selected directory on
  tree).
  add buttons to history REDO and UNDO on file editor control header. change local variables panel to add copy to
  clipboard button (must icon) per entry.
  on gad editor, add copy to clipboard button (must icon) on tooltip. change codemirror plugin to add code editor
  features (auto complete etc) on
  edit code/expression in template strings.
  add right closable panel to render doc comments of current editing file. it reloads 5s after edit or now (with reload button). 
- [ ] update `mkoptypes` generator for `op_api.go` to generate interfaces of unary operators `--`/`++`
  with syntaxe `type ObjectWith[OPERATOR_NAME]UnaryOperator interface { UnOp[OPERATOR_NAME](vm *VM) (Object, error) }`.
  add builtin function "unOp" to module "core" for unary operators. change `VM.xOpUnary()` to call "core.unOp(&vm.stack[vm.sp-1])" and move all
  per type implementatios to use `AddMethod` API of `core.unOp` builtin function, calling `obj.UnOp(vm)`.
- [ ] update `mkoptypes` generator for `op_api.go` to generate interfaces of self assign operators.
  with syntaxe `type ObjectWith[OPERATOR_NAME]SelfAssigOperator interface { SelfAssignOp[OPERATOR_NAME](vm *VM, value Object) (Object, error) }`.
  change builtin function "core.selfAssignOp" methods (`AddMethod` API) to call `obj.SelfAssignOp(vm, value)`.
- [ ] parse binary operator `ain`, like `in`, but accept `array` of values. (`A ain B`/`[...] ain B`).
  update `mkoptypes` generator for `op_api.go` to add interface of `ain` operator (`ArrayIn`.
  if `B` does not implements `ObjectWithArrayInBinOp` interface, but implements `ObjectWithInBinOp` takes
  `for v in A { v, err := B.BinOpIn(v); // check error\nif v.IsFalsy() { return false } }; return true`.
  create samples, docs and parser/compiler/vm tests.
- [ ] `with` implementation. add new gad objects interface `type ObjectEnter interface { Enter(*VM) (error) }` and
  `type ObjectExit interface { Exit(*VM, err error) (Object, error) }`.
  add new builtin functions "enter" and "exit" (with empty body) to "core" module.
  parse `with` Stmt and Expr, like python, with syntaxe:
  - Stmt:
  
    ```gad
    with Expr [as IDENT] {
      // STMT
    }
    
    //////////////////////
    // with assign
    //////////////////////
    var x
    
    with x = Expr { ... }
    
    // is shortcut form of:
    x = Expr
    with x { ...}
    
    //////////////////////
    // with define
    //////////////////////
    with x := Expr { ... }
    
    // is shortcut form of:
    x := Expr
    with x { ...}
    
    //////////////////////
    // with "as"
    //////////////////////
    with open("file") as f {}
    
    // is shortcut form of (in new block)`
    {
      const f = open("file")
      with f { //.. }
    }
    
    ```
  
  - Expr: `x := with Expr [as IDENT]: ValueExpr`, examples:
  
    ```gad
    const f = open("file")
    contents := with f: f.read()
    
    // with "as"
    const f2 = with open("f2") as f: f.read()
    
    // joined values
    const data = "a" + (with open("f3") as f: f.read()) + "b"
    ```
  
  all forms compile like to `deferb`, no require new Op Code:
  ```gad
  {
    deferb { core.exit(x, $err) }
    core.enter(x)
    // Body or result set
  }
  ```

  create samples, docs and parser/compiler/vm tests.
- [ ] create command cmd/update-codemirror-plugin to update codemirror-gad plugin with language changes
      after last codemirror plugin updates by git log.
- [ ] create command cmd/update-prism-plugin to update prism-gad plugin with language changes
  after last prism-gad plugin updates by git log.
