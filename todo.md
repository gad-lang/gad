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

- [ ] parse binary user operators: 1) TripleLess `<<<`; 2) TripleGreater `>>>`; 3) DoubleMod `%%`; create assign
  operators version `<<<=`, `>>>=`, `%%=`. created samples (using `met @binaryOperator`), docs and parser/compiler/vm
  tests.
- [ ] parse binary operator `in` (`A in B`). it compiles to `OpIn` to call `Contains` method (implements
  `interface Container { Contains(v Object) (bool, error) }`),
  for fallback, calls binary operator handlers. take Dict (of key name), Array (of value), KeyValueArray (of key name),
  SyncDict (of key name),
  MethodInterfaceInstance (of FunctionHeader value, or FunctionHeaders), Bytes
  to implements `Container` interface. for loop isn't binary operator (`for x in y {}` - ` x in y` ins't operator;
  `for (x in y) {}` - ` x in y` is operator).
  create samples, docs and parser/compiler/vm tests.
- [ ] parse binary operator `ins`, like `in`, but accept `array` of values. (`A ins B`/`[...] ins B`). it compiles to
  `OpInOfArray` to call `ContainsArray` method (implements
  `interface ArrayContainer { ContainsArray(v Array) (bool, error) }`,
  `ContainsArray` return `true` only if all values exists on object),
  if `B` does not implements `ArrayContainer` interface, but implements `Container` takes
  `for v in A { if !B.Contains(v) { return false } }; return true`, fallback calls binary operator for this Token.
  create samples, docs and parser/compiler/vm tests.
- [ ] implements parser of doc comments, contents is in Markdown (allowing safe HTML code).
  the doc is linked to IDENT (set to Doc field) or STMT. take to accept doc idents of DECL, `func/met/meti/prop` stmts (
  set to Doc field). put formatted doc in `WriteCode`.

  syntaxe: 
  - SINGLE: `/? ...\nSTMT`. `const (\n\t/? the pi value\n\tpi = 3.14\n)` (linked to `pi`);
  - INLINE (no value): `IDENT /? ...`. `var pi = 3.14 /? the pi value` (linked to `pi`);
  - INLINE_VALUE (with value): `IDENT = EXPR /? ...`. `const pi = 3.14 /? the pi value` (linked to `pi`);
  - BLOCK: `??\n...\n??\nSTMT`. `const (\n\t/??\n\tthe pi value\n\t??\n\tpi = 3.14\n)` (linked to `pi`).
  - ROOT_BLOCK: like BLOCK, but use `???` insteadof `??`.

  examples:
  ```gad
  ???
  this is a root doc
  ???
  
  const pi = 3.14 /? the pi value
  
  ???
  this an anoter root doc
  ???
  
  /? this is the server addr
  const ServerAddr = ":0"
  
  var (
    /? the value of a
    a
    
    /? the value of b
    b
  
    c = 1
    d = 2 /? the value of d
    e = 3
  
    f, g /? f and g (this is bad, throw parser error)
  )
  
  /? sum values
  func sum(a, b) { return a + b )
  
  /? sum values
  sum(a, b) => a + b 
  
  ??
  this is a difference calculator.
  see all methods of here to check
  specific diffenrece handler.
  ??
  func diff {
    /? compute difference of b and a
    (a int, b int) => b - a
    
    ??
    compute difference of b and a
    values.
    ??
    (a int, b flaot) => b - a
  }
  ```
  **format rules** (save it on conventions):
  - if is SINGLE, but it's long (see `NEW_LINE_CALC`), take it to BLOCK.
  - if is BLOCK, but it's short (see `NEW_LINE_CALC`), take it to SINGLE.
  - auto format contents using Markdown formatter.
  - when SINGLE or BLOCK, put new line after target (if necessary):
    - no formatted source:
      ```
      var (
        /? the a value
        a
        b
        c
      )
      
      ??
      this is a difference calculator
      see all methods of here to check.
      
      specific diffenrece handler.
      ??
      func diff {
        /? compute difference of b and a sa ash dkas dahs daks kjahd kash dasdh asdahd a dh a dasdh ad ah a skdahsd as dkad as dhkahs da sd
        (a int, b int) => b - a

        ??
        compute difference of b and a values.
        ??
        (a int, b flaot) => b - a
      }
      ```
    
      formatted

      ```
      var (
        /? the a value
        a
      
        b
        c
      )
      
      ??
      this is a difference calculator see all methods of here to check.
      
      specific diffenrece handler.
      ??
      func diff {
        ?? compute difference of b and a sa ash dkas dahs daks kjahd kash dasdh asdahd a dh a dasdh
        ad ah a skdahsd as dkad as dhkahs da sd
        ??
        (a int, b int) => b - a

        /? compute difference of b and a values
        (a int, b flaot) => b - a
      }
      ```
    - formatted source (no change formatted result):
      ```
      var (
        /? the a value
        a
      
        b
        c
      )
      ```
    - create samples, docs and parser/compiler/vm tests.
    - create gad subcommand `doc` like `fmt`, to save result with `.md` extension (if no flag `--no-save`).
      the flag `--out` (default is `"doc"`)
      - generate output like godoc: 
        ```markdonw
        # MODULE_NAME
        
        TABLE OF CONTENTS
        
        ## Constants
        section of constants
        
        ### const **pi**
          
          const pi = 3.14
        
        this is a pi value rounded to two fracts.
        
        ## Types
        section of func/met/meti/closures
        
        ### func **sum** // no methods or one method
        
          sum(a int, b int) <int> // this a HEADER
        
        returns a + b.
        
        ### func **diff** // this is func with 2 methods
        
          (a int, b int) <int> // default method
        
        compute difference of a and b int values
        
        **other methods**
        
          (a float, b float) <int>
        
        compute difference of a and b float values.
        
          (a int, b float) <int>
        
        compute difference of a int and b float values.
        ```
      - add root config file key `doc: { dst: DIR, skip: false }` when `DIR` is absolute path or relative to WORKSPACE_DIR.
        default value of `doc.dst` is value of flag `--out`. if `doc.skip` is `true`, no parse/doc for non INPUT_DIR sources.
      - add flag `--no-skip`, to set `doc.skip` to `false`.
      - add `doc: { dst: DIR (default is root "doc.dst" only if isn't absolute path), skip: (default is root "doc.skip") }` per INPUT_DIR, its absolute path or relative to INPUT_DIR path.
        if no flag `--no-save` and `doc.dst` is set, write doc files preserving tree relative to WORKSPACE.
        for sources in INPUT_DIR:
        - if `INPUT_DIR.doc.skip`. if is `true`, no parse/doc for here.
        - if `INPUT_DIR.doc.dst`. if not set, must skip.
        - if skips because `skip` or `doc.dst` is empty, log coloured (if stderr is tty) message to stderr and skip doc 
          generator for here, but not exit program.
        - when absolute path of `INPUT_DIR.doc.dst` equals to root `doc.dst`, raise an error and exit program with error status.
      - create doc and samples for this subcommand.