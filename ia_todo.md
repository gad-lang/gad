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