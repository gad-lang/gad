- [x] Parse bytes from hex string (const data = h"ffccf1c2" // typeof data == bytes)
- [x] Parse bytes from string/raw string/heredoc/rawheredoc (co[zeroer](zeroer)nst data = b"Hello" // typeof data == bytes)
- [x] recreate user documentation in ./doc with various examples and variations for all gad features. if necessary,
  split doc into multiples files.
- [x] change gad cmd replace flags to subcommands use github.com/moisespsena-go/command-context.
      add fmt subcommand for format input files (one or more) from args. the files args accepts dir paths.
      if dirs likes with `PATH/...` run it recursively, but ignore hidden files and skip hidden directories.
      fmt has flag "--exclude GLOB_PATTERN" (allow many separated by "," or more same flag) and flag "--include GLOB_PATTERN" 
      (like "--exclude") to include ignoring exclusion rules. fmt has flag "--backup" (bool, default false) and
      "--backup-format" (default is "BASE_NAME.backup.gad"). use logic like parse/test/Parser.FormattedCode, but create boolean flags to 
      remove command context flags for all flags (example: "--no-call-params-in-new-line" (for remove flag CodeWriteContextFlagFormatCallParamsInNewLine).
      update user documentation. commit changes to main.
- [x] create codemirror 6 plugin with auto complete and line/colunm warn or error messages. create example web app with Golang server and React to proccess
      source file from (`gad` reading from stdin writing to stdout and report errors to editor per line/colum) and put result to
      left viewer. create on example for interactive execution, like notebook. taks one example from backend server proccess and other example using WASM.
- [x] create prismjs plugin and create page on web app for usage example.
- [x] create cmd/delve as a debugger for gad language like github.com/go-delve/delve and create Visual Studio Code plugin like https://github.com/golang/vscode-go
      for execute delve (create script on your package.json for export as VSCode plugin file to friendly installation must importing her). 
      create React pugin using gad-codemirror for execute and debug source file. create new page on web app for run and debug.
- [x] create cmd/build-website for build full gad lang static website (language API, user documentation with term searcher) 
      with dark/light theme and examples with WASM compatible for github web page, create github action for auto rebuild website and publish then.
      the website generator per commit `/COMMIT-ID` and must publish to github release version on RELEASE.
- [x] create subcommand "ide" in cmd/gad to start web app with dark/light theme, in React, providing best ide with tabs
      for multiples files editing (on CWD or first command arg PATH - if PATH is single file, must edit here), 
      format, run and debug buttons. formatting settings (reading and save to .gad.yaml), using codemirron plugin.
      allowing open/hide panels and move to left/right/buttom. saving panel positions to .gad.yaml "ide" key.
      with dialogs for per file configure run/debug with field for set params, enable/disable builtins modules, and
      save STDOUT/STDERR to file. allow panels resizing and grouping with tabs layout.
- [x] in the mixed/template mode, the CodeBeginStmt haves optional `-` suffix (example: `{%-`); if
      this sufix is set, add flag RemoveRightSpaces to MixedTextStmt (previous stmt). 
      the CodeEndStmt haves optional `-` prefix (example: `-%}`); if this suffix is set, add flag RemoveLeftSpaces to 
      MixedTextStmt (next stmt). when format (call CodeWriter.WriteCode), preserves MixedTextStmt Literal value,
      buf format gad code between CodeBeginStmt to CodeEndStmt. then MixedValueStmt likes (joined form if CodeBeginStmt and CodeEndStmt).
      when format `(\{%-?)\s*end|done\s*(-?%\})` (regexp syntaxe), format to `$1 end $2`.
- [x] generate godoc strings for CodeBeginStmt, CodeEndStmt, MixedValueStmt. update gad program when run from STDIN buffer or run PATH,
      to run without interactive mode. change gad program to add flag `--template` to run template files (ParserOptions.Mode |= ParseMixed and
      ScannerOptions.mode |= ScanMixed | ScanConfigDisabled); flags `--template-start-delimiter DELIMITER` and `--template-end-delimiter DELIMITER`.
      put template-start/end-delimiter to config `tempate`; if demilimiters is not set, use default. change parser of mixed/template mode
      to add new Code Begin/End suffix `--` (`{%--` and `--%}`) resulting in two prefix/sufix forms: 1) `-` remove all black chars at `\n` (preserve it);
      2) `--` remove all blank chars. generate parser/vm tests. generate docs and samples. update README and doc.
      change gad program when run `.gadt` files, run as template mode.
- [x] add go build tags to skip build gad program subcommands `ide` and `debug`. set `run` subcommand as default command.
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
- [x] change gad fmt to write report per file, not grouped (current version) as single line JSON (remove YAML support) string with keys:
      `{ input_dir: (only if file in INPUT_DIR), file: (the file name, if in INPUT_DIR, relative to here), error: (if failt) }\n`.
      add flag `--to-stdout`. in this case if report file isn't set, print report to stdout, when format files print 
      result to stdout stream. add flag `--boundary BOUNDARY` if not set, generate new UUID as BOUNDARY and print it to first line of stdout `>> BOUNDARY`, the stream syntaxe:
      ```
      -- BOUNDARY #FILE_INDEX [INPUT_DIR] (with brackets, if in INPUT_DIR) FILE_NAME (if in INPUT_DIR, relative to here)
      FORMATTED_FILE_RESULT
      -- BOUNDARY #FILE_INDEX
      ```
- [ ] add parser for new token for CodeStrLit, like strheredoc with `code\s|\n` (start quote) and `\s|\nend` (end quote, \n only if start `code\n`).
      compiles to Str. no parses contents like Template. change codemirror plugin and prism plugin to recognize contents of this as gad source code. 
      add tests for scanner/parser/compiler/vm.
- [ ] takes stdlib/time and strings modules as builtin module, mapping all functions as BuiltinFunction and BuiltinType.
- [ ] add parser of `\d{4}\d{2}\d{2}D` as new type time.Date ("time" is gad module, not go time); `\d{4}\d{2}\d{2}_\d{2}\d{2}\d{2}(.(\d{3}|\d{6}|\d{9}))(Z\d{2}\d{2})T` as type time.Time.
      implements encoder for time.Date and time.Time. add parser for go time.Duration string (to new type time.Duration alias og go time.Duration). 
      compile values to time.Date, time.Time, time.Duration (create constructor for this type). generate samples and docs.
- [ ] updated doc to add examples for "~" and "~~" regexp operators and POSIX `/.../p` (`p` sufix), add examples using
      captured groups and regexp flags.
      `raw EXPR`, produces `rawStr` type (`raw "a"` is in compiler time, but `raw str(100)` is in execution time) - update doc for here.
      add examples for The `or` Fallback Operator using `$err` variable.
      replace match else `else:` must to `else`, update doc with examples.
      add godoc and docs with examples for ComputedValue
      create dog of func/closure/method syntax and add examples.
      