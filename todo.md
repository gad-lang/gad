- [ ] gad codemirror plugin isn't working on ide. change ide files tree to allow rename file/dir with F2 or RIGHT CLICK MENU (with options: `run`, `format`, `transpile` 
      (format with TranspileOptions for `.gad` and `.gadt` exts. add fields of config file key `transpile` (add to settings dialog)), and `remove` 
      (with confirmation dialog - if is nom empty dir, add check field RECURSIVELY)). put run/debug options dialog into new
      separated dialog "run/debug" settings and put run/debu as tabs. split field "Save stdout+stderr to file", allowing
      set file for stdout and stderr, add new flag field for combine stdout and stderr. change ide to on click over breackpoint
      on line number panel, open brackpoint dialog with fields "disabled" (if is set, ignore debug this breackpoint), "condition"
      (with gad codemirror plugin, to specify and expression and pause no debug only `!value.IsFalsy()`) and cancel/save buttons; change brackpoints
      panel, to add right button to remove here, and when click over brackpoint entry open the brackpoint dialog, when click
      here with double click, goto location here. change ide to support tooltip for ident of builtin values.
      change ide to add panel "evaluate". this panel haves list of evaluated expressions. on fixed top of this panel, puts form
      for add new expression to evalue, with expression field, flag field "repr" and "+" button; when add, include here
      to list and evaluate returning result of "str(EXPRESSION)" if flag "repr" is set, replace "str" to "repr". each
      list entry, add right buttons "edit" (open into top form and change button to save icon). trash icon, to remove then;
      "output" button to open new dialog with result value as codemirror editor for plain text in readonly mode and "copy"
      button (must icon) to copy to clipboard. update evaluated expressions when debugger step changing.
      change ide file editor controls add button "reload" to reload file from disk. add header to explore three to add flag field to show/hide hidden files/dirs.
      add file editor support for JSON, YAML, HTML, CSS, SCSS, JS (with types script e JSX) and open other types to plain/text editor.
      change id to alert error in dialog when fail to request to backend. change ide explorer add button to open and dialog to get file from web
      and allow to change your output name and choose directory to save then (default is current selected directory on tree).
      add buttons to history REDO and UNDO on file editor control header. change local variables panel to add copy to clipboard button (must icon) per entry.
      on gad editor, add copy to clipboard button (must icon) on tooltip. change codemirror plugin to add code editor features (auto complete etc) on
      edit code/expression in template strings.
- [x] change gad fmt to write report, write file goroutine (per file) is done (success or error) as single JSON inline string:
      `{ "input_dir": (only if file in INPUT_DIR), "file": (the file name, if in INPUT_DIR, relative to here), "error": (if failt) }\n`.
      default result example:
      ```json
      { "input_dir": "src", "file": "a.gad", err: "bad format" }
      { "input_dir": "src_2", "file": "b.gad" }
      ```
      replace flag `--to-stdout` to new flag `--report-stream` and allow `--report -` (`-` is to stdout).
      add flag `--report-contents` to put formatted result to report to file key "result".
      add flag `--no-save` to no save formatted result, must readonly (not backup, no write, no create ...).
      remove flag `--boundary`.
      the report output with contents flag example:
      ```
      BOUNDARY
      { "input_dir": "src", "file": "a.gad", err: "bad format" }
      { input_dir: "src_2", "file": "b.gad", "result": "FORMATTED RESULT" }
      ```
- [ ] add parser of `\d{4}\d{2}\d{2}D` as new type time.Date (alias of go uint) ("time" is gad module, not go time);
      parse Time syntaxe `(\d{4}\d{2}\d{2})?(_?\d{2}\d{2}\d{2})(.(\d{3}|\d{6}|\d{9}))(Z(-?\d{2}\d{2})|[A-Z]{3})T`(grop1) the date; group2) the time; 
      group3 the seconds fraction (\.d{3} as mili; \.d{6} as micro; \.d{9} as nano); group4) location offset or name), as type time.time; 
      examples: (must time date: `20260131T` (year: 2026, month: 01, day: 31); must time `235955T` (hour: 23, minute: 59, second: 55);
      `20260131_235955T` (year: 2026, month: 01, day: 31, our: 23, minute: 59, second: 55);
      time with seconds fractions `235955.001T`, `235955.001300T`, `235955.001300200T`;
      time with location: `235955.001ZGRUT` (location `GRU`), `235955.001Z-03:15T` (location `-03:15`);
      unix time seconds `1781609136U`; unix time fraction `1781609136.123U` (micro), `1781609136.123456U` (mili), `1781609136.123456789U` (nano)).
      implements encoder for time.Date and time.time. add parser for go time.Duration string (to new type time.Duration alias of go time.Duration). 
      compile values to time.Date, time.time, time.Duration (create constructor for this type). generate samples and docs.
      add method "strToTime" of time.time constructor to parse time from str (with parser syntaxe with/without `U` sufix).
      add method "strToDate" of time.Date constructor to parse Date from str (with parser syntaxe with/without `T` sufix).
      add method "strToDuration" of time.Duration constructor to parse Duration from str (with parser syntaxe).
      add method "strToLocation" of time.Location constructor to parse Location from str. 
      takes time.time, time.Date and time.Duration as primitive types.
      update docs and samples.
- [x] updated doc to add examples for "~" and "~~" regexp operators and POSIX `/.../p` (`p` sufix), add examples using
      captured groups and regexp flags.
      `raw EXPR`, produces `rawStr` type (`raw "a"` is in compiler time, but `raw str(100)` is in execution time) - update doc for here.
      add examples for The `or` Fallback Operator using `$err` variable.
- [ ] on MatchExpr, take as just syntaxe: `x := match Expr { ArmExpr: ValueExpr, ArmExpr, ArmExpr: ValueExpr, else: ValueExpr }`; 
      allowing multiples Exprs per arm (separated by comma or new line or both). arms or `else` is optional, if not
      set both the value is `nil`. this is valid: `x := match Expr {}` (`x` is nil); `x := match 1 { 2: "ok"}` (`x` is nil).
      this is bad: `x := match Expr { else: 2 }`.
      on WriteCode, put arms to new line when `ctx.Flags.Has(CodeWriteContextFlagFormatMatchStmtArmsInNewLine)` (for all to new lines),
      or when `NEW_LINE_CALC` (put to new lines only cases when `writed_line_columns + formatted_inline_writed_arm_expr_columns + current_current_arm_expr_columns > ctx.MaxColumns`),
      include `CodeWriteContextFlagFormatMatchExprArmsInNewLine` to `CodeWriteContextFlagFormat`.
      example:
      ```gad
      x := match Expr { 
          ArmExpr: ValueExpr
          ArmExpr, ArmExpr, ArmExpr, ArmExpr, ArmExpr,
          ArmExpr, ExpArmExprr: ValueExpr
          else: ValueExpr
      }
      ```
      no new line split example: `x := match Expr { ArmExpr: ValueExpr, ArmExpr, ArmExpr: ValueExpr, else: ValueExpr }` (small columns count).
      create doc, parser/compiler/vm tests for match expr (including multiples Exprs per arm).
- [ ] on MatchStmt with syntaxe:
      ```gad
      match Expr { 
          Expr {
            // Stmt...
          }
          Expr, Expr {
              // Stmt...
          }
          else {
              // Stmt...
          }
      }
      ```
      no idented example:
      ```gad
      match Expr { Expr { // Stmt... } Expr, Expr { // Stmt... } else { // Stmt... }
      ```
      this is valid: `match Expr {}`.
      on WriteCode, split arms to new line when `ctx.Flags.Has(CodeWriteContextFlagFormatMatchStmtArmsInNewLine)` (for all to new lines),
      when `NEW_LINE_CALC`, put to new lines only cases when `writed_line_columns + formatted_inline_writed_arm_expr_columns + current_current_arm_expr_columns > ctx.MaxColumns` like:
      ```gad
      match Expr { 
          Expr {
            // Stmt...
          }
          Expr, Expr, Expr, // big columns count
          Expr {
              // Stmt...
          }
          else {
              // Stmt...
          }
      }
      ```
      allowing multiples Exprs per arm.
      create doc, parser/compiler/vm tests for match expr (including multiples Exprs per arm).
      
- [x] create doc of func/closure/method/ComputedValue syntax and add examples.
- [ ] add doc for gad code conventions:
      - single Decl without params. `var x` insteadof `var (x)`. (apply this rule for related CodeWriter).
      - split args/dict keys/named params keys etc to new lines for all or when `NEW_LINE_CALC`.
        good: `var (x, y)`, `var (x = 1, y = 2)`
        good:
        ```gad
        var (
            // group declarations without value as first
            a, b, c // big
            d, e
            f = 1
            g = 2
            r = 1, s = 2
            t, u = 3, 4
            v, x, y, x = Expr // destructuring
            (a1, a2; a3, **r) = Expr
        ) 
        ```
        bad:
        ```gad
        var ( a, b, c
            d, e
            f = 1,  g = 2
        ) 
        ``` 
        apply this rule for related CodeWriter when `ctx.Flags.Has(CodeWriteContextFlagFormat*InNewLine)` (force all to new lines) or `NEW_LINE_CALC`
- [x] implement unary operators `--IDENT` and `++IDENT`. create samples, doc, and parser/compiler/vm tests for here.
- [x] FuncHeader feature:
  - create new node FuncHeader and use it as anonymous field of FuncType:
    ```go
    type FuncHeader struct {
        NameExpr Expr
        Params   FuncParams
        Return   []*TypedIdentExpr
    }
      
      // Pos returns the position of first character belonging to the node.
      func (e *FuncHeader) Pos() source.Pos {
		  // detect order
          // NameExpr.Pos if set
          // Params.Pos if set
          // Return[0] if set
          // NoPos
      }
      
      // End returns the position of first character immediately after the node.
      func (e *FuncHeader) End() source.Pos {
          // detect order
          // Return[len(Return)-1] if set
          // Params.End if set
          // NameExpr.Pos if set
          // NoPos
      }
      
      func (e *FuncHeader) NameIdent() *IdentExpr {
          if e.NameExpr == nil {
              return nil
          }
          return IdentOfSelector(e.NameExpr)
      }
      
      func (e *FuncHeader) Name() string {
          if e.NameExpr == nil {
              return ""
          }
          switch t := e.NameExpr.(type) {
          case *IdentExpr:
              return t.Name
          case *IndexExpr:
              switch it := t.Index.(type) {
              case *StrLit:
                  return it.Value()
              }
          case *SelectorExpr:
              switch it := t.Sel.(type) {
              case *IdentExpr:
                  return it.Name
              }
          }
      
          return ""
      }
      
      func (e *FuncHeader) String() string {
          var s string
          if e.NameExpr != nil {
              s = e.NameExpr.String()
          }
          s += e.Params.String()
          s += FormatFuncReturn(e.Return)
          return s
      }
    
    type FuncType struct {
        Token    TokenLit
        FuncPos  source.Pos
        FuncHeader
    }
    
    // Pos returns the position of first character belonging to the node.
      func (e *FuncType) Pos() source.Pos {
		  // detect order
          // Token.Pos if set
          // FuncPos.Pos if set
          // Header.Pos
      }
      
      // End returns the position of first character immediately after the node.
      func (e *FuncType) End() source.Pos {
          // Header.Pos
      }
    
    type FuncHeaderExpr struct {
        OpenPos    source.Pos // `<`
        ClosePos    source.Pos // `>`
        FuncHeader
    }

    // Pos returns the position of first character belonging to the node.
      func (e *FuncHeaderExpr) Pos() source.Pos {
		  // detect order
          // OpenPos.Pos if set
          // FuncHeader.Pos if set
      }
      
      // End returns the position of first character immediately after the node.
      func (e *FuncHeaderExpr) End() source.Pos {
          // detect order
          // Header.Pos
      }
    
      func (e *FuncHeaderExpr) String() string {
            return "<" + e.FuncHeader.String() + ">"
      }
    
    ```
    - create FuncHeaderExpr syntaxe like func type `<()>`,`<(v int)<x uint|int>>`
    - take gad.FunctionHeader as Object with `IndexGet` for `name`, `params` and `namedParams` (see gad.CompiledFunction) - builtin type is `TFunctionHeader` (name "FunctionHeader") 
    - compile FuncHeaderExpr to call builtin type FunctionHeader
    - create `MethodInterfaceExpr{ NameExpr Expr (optional), Headers []*FuncHeaderExpr // min 1 header }` parsed from:
      `meti { () }`/`meti { (), (v)<int> }`/`meti IDENT { () }` (IDENT as NameExpr, if is *Ident define `const IDENT = meti IDENT {...}`)
      allowing `x := meti { () }`/`x := meti Z { () }`, when headers was separated by command or new line. Take `WriteCode` here
      like MatchExpr with context format flag for here.
    - create object type `MethodInterface` (builtin type - global var `TMethodInterface`) and compile `MethodInterfaceExpr` to call `MethodInterface(name, *headers)`,
      return instance of `MethodInterfaceInstance` (`Type()` method return `TMethodInterface`).
    - allow `append` function to create new `MethodInterface` join multiples `MethodInterface` 
    - implement builtin operator `add` to merge `MethodInterfaceInstances` to new `MethodInterfaceInstance`
    - create builtin function `implements(fn CALLABLE, mi MethodInterfaceInstance, *otherMi MethodInterfaceInstance) <bool>` to return if
      fn has all defined headers of all items in the list `[mi, *otherMi]`.
    - create samples, doc, and parser/compiler/vm tests for all new features.
    
- [x] allow `--`;`++` as binary operator (preserves unary operator). if `++` not handled on Object, call `append` function as fallback. create samples, doc, and parser/compiler/vm tests for here. (append fallback skipped per request)