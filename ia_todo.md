- [x] Parse bytes from hex string (const data = h"ffccf1c2" // typeof data == bytes)
- [x] Parse bytes from string/raw string/heredoc/rawheredoc (const data = b"Hello" // typeof data == bytes)
- [x] recreate user documentation in ./doc with various examples and variations for all gad features. if necessary,
  split doc into multiples files.
- [ ] change gad cmd replace flags to subcommands use github.com/moisespsena-go/command-context.
      add fmt subcommand for format input files (one or more) from args. the files args accepts dir paths.
      if dirs likes with `PATH/...` run it recursively, but ignore hidden files and skip hidden directories.
      fmt has flag "--exclude GLOB_PATTERN" (allow many separated by "," or more same flag) and flag "--include GLOB_PATTERN" 
      (like "--exclude") to include ignoring exclusion rules. fmt has flag "--backup" (bool, default false) and
      "--backup-format" (default is "BASE_NAME.backup.gad"). update user documentation. commit changes to main.
