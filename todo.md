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
- [ ] in the mixed/template mode, the CodeBeginStmt haves optional `-` suffix (example: `{%-`); if
      this sufix is set, add flag RemoveRightSpaces to MixedTextStmt (previous stmt). 
      the CodeEndStmt haves optional `-` prefix (example: `-%}`); if this suffix is set, add flag RemoveLeftSpaces to 
      MixedTextStmt (next stmt). when format (call CodeWriter.WriteCode), preserves MixedTextStmt Literal value,
      buf format gad code between CodeBeginStmt to CodeEndStmt. then MixedValueStmt likes (joined form if CodeBeginStmt and CodeEndStmt).
      when format `(\{%-?)\s*end|done\s*(-?%\})` (regexp syntaxe), format to `$1 end $2`.
- [ ] generate godoc strings for CodeBeginStmt, CodeEndStmt, MixedValueStmt. update gad program when run from STDIN buffer or run PATH,
      to run without interactive mode. change gad program to add flag `--template` to run template files (ParserOptions.Mode |= ParseMixed and
      ScannerOptions.mode |= ScanMixed | ScanConfigDisabled); flags `--template-start-delimiter DELIMITER` and `--template-end-delimiter DELIMITER`.
      put template-start/end-delimiter to config `tempate`; if demilimiters is not set, use default. change parser of mixed/template mode
      to add new Code Begin/End suffix `--` (`{%--` and `--%}`) resulting in two prefix/sufix forms: 1) `-` remove all black chars at `\n` (preserve it);
      2) `--` remove all blank chars. generate parser/vm tests. generate docs and samples. update README and doc.
      change gad program when run `.gadt` files, run as template mode.
- [ ] add go build tags to skip build gad program subcommands `ide` and `debug`. set `run` subcommand as default command.
- [ ] gad codemirror plugin isn't working on ide. change ide files tree to allow rename file/dir with F2 or RIGHT CLICK MENU (with options: `run`, `format`, `transpile` 
      (format with TranspileOptions for `.gad` and `.gadt` exts. add fields of config file key `transpile` (add to settings dialog)), and `remove` 
      (with confirmation dialog - if is nom empty dir, add check field RECURSIVELY)). put run/debug options dialog into new
      separated dialog "run/debug" settings and put run/debu as tabs. split field "Save stdout+stderr to file", allowing
      set file for stdout and stderr, add new flag field for combine stdout and stderr. change ide to on click over breackpoint
      on line number panel, open brackpoint dialog with fields "disabled" (if is set, ignore debug this breackpoint), "condition"
      (with gad codemirror plugin, to specify and expression and pause no debug only `!value.IsFalsy()`) and cancel/save buttons; change brackpoints
      panel, to add right button to remove here, and when click over brackpoint entry open the brackpoint dialog, when click
      here with double click, goto location here.
- [ ] change gad fmt to write report per file, not grouped (current version) as single line JSON (remove YAML support) string with keys:
      `{ input_dir: (only if file in INPUT_DIR), file: (the file name, if in INPUT_DIR, relative to here), error: (if failt) }\n`.
      add flag `--to-stdout`. in this case if report file isn't set, print report to stdout, when format files print 
      result to stdout stream. generate new UUID as BOUNDARY and print it to stdout line `>> BOUNDARY` with this syntaxe:
      ```
      -- BOUNDARY #FILE_INDEX [INPUT_DIR] (with brackets, if in INPUT_DIR) FILE_NAME (if in INPUT_DIR, relative to here)
      FORMATTED_FILE_RESULT
      -- BOUNDARY #FILE_INDEX
      ```
