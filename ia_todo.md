- [x] Parse bytes from hex string (const data = h"ffccf1c2" // typeof data == bytes)
- [x] Parse bytes from string/raw string/heredoc/rawheredoc (const data = b"Hello" // typeof data == bytes)
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
- [ ] create codemirror 6 plugin with auto complete and line/colunm warn or error messages. create example web app with Golang server and React to proccess
      source file from (`gad` reading from stdin writing to stdout and report errors to editor per line/colum) and put result to
      left viewer. create on example for interactive execution, like notebook.