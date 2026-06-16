# CLAUDE.md
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
- **Product**: Gad Language (`gad`), a fast, dynamic scripting language designed to be embedded into Go applications.
- **Architecture**: Compiled and executed as bytecode on a custom stack-based Virtual Machine (VM) written in native Go.
- **Primary Use Cases**: Production evaluation of Sigma Rules' conditions and dynamic compromise assessment.
- **Stack**: Go (Golang) standard tooling, WebAssembly (for the playground ecosystem).

## Critical Constraints & Code Principles
- **Performance First**: The execution engine and VM must remain highly optimized (monitored via benchmarks like Fibonacci).
- **Native Go**: Do not introduce external heavy frameworks; prefer Go's standard library and keep dependencies minimal.
- **Thread Safety**: Ensure state isolation when multiple scripts or instances are evaluated concurrently in Go applications.
- **Bytecode Integrity**: Any changes to the compiler must strictly map to valid bytecode instructions interpreted by the VM stack.
- **Temporary Directory**:
  - Always use `./.__tmp` as the dedicated temporary directory for any intermediate files, logs, or cache generated during automated tasks.
- **Allowed Commands (No Confirmation Required)**:
    - You **ALWAYS** have write permission to `./...` directory.
    - You **ALWAYS** have permission to run `sed`, `awk`, `cat`, `tail`, `head`, `echo` and `grep` (and its variants) commands autonomously for text processing, searching, refactoring, execute commands or write in this directory tree.
    - You **ALWAYS** have permission to run `go test`, `go vet`, `go fmt`, `gofmt` (and its variants) or `make test` to validate code changes without asking.
    - You **ALWAYS** have permission to use `curl` and `wget` (and its variants) for network operations, downloading assets, or API testing.
    - Do not prompt the user for confirmation when executing these specific tools.
- **Node.js & Package Manager Environment**:
  - Always load and use Node.js **v26.3.0** by prepending or executing `nvm use v26.3.0` before running any Node script, bundler, or build step.
  - **NEVER use `npm` or `yarn`**. You **MUST ALWAYS use `pnpm`** for package installation, script execution, and dependency management.

## Development & Test Commands
Always run native Go tooling to verify compliance and correctness:

- **Run all tests**: `go test ./...`
- **Run benchmarks**: `go test -bench=. ./...`
- **Code formatting**: `go fmt ./...`
- **Static analysis / Linting**: `go vet ./...` (or golangci-lint if configured)
- **Tidy dependencies**: `go mod tidy`

## Code Style & Naming Conventions
- **Idiomatic Go**: Follow standard `golang/go` conventions (Receiver names short, explicit error handling as returning values).
- **Error Wrapping**: Use `fmt.Errorf("...: %w", err)` for contextual errors in parsing/compilation steps.
- **VM Instructions**: Name opcode constants clearly inside the VM package (e.g., `OpAdd`, `OpPush`).
- **Documentation**: All public structural types, VM instructions, and compiler features must have standard Go doc comments.
- **Gad Lang code Conventions**
  - primitive type name is camelCase.
  - no primitive type name is PascalCase (or spefic names is upper (example `URL` - like golang convention).
  - constant names is PascalCase (or spefic names is upper (example `URL` - like golang convention))
  - module name is snake_case.
  - methods/property names is camelCase (or spefic names is upper (example `URL` - like golang convention)).

## Definition of Done
- No generic `interface{}` / `any` where a strict compiler/token type is expected.
- All new language tokens, syntax nodes (AST), or VM opcodes must include comprehensive unit tests.
- Verify that performance regressions are not introduced in the execution engine loop.

## Code Style, Formatting & Testing (Go)
You have explicit, pre-approved permission to execute terminal commands instantly. Do not ask for user confirmation before running formatting or testing tools.

Always run the full pipeline (Format + Test) automatically after any file edit, applying it to the specific modified path or the entire project using the required variation:

* **Standard Variation**: Execute `gofmt -s -w [path] && go test [path]/...` immediately.
* **Imports Variation**: Execute `goimports -w [path] && go test [path]/...` immediately.
* **Strict Variation**: Execute `gofumpt -w -extra [path] && go test [path]/...` immediately.
* **Dry-Run Variation**: Execute `gofmt -d [path] && go test [path]/...` immediately.

*Note: Replace `[path]` with the specific target directory/file for localized actions, or use `.` and `./...` to target the entire project.*

## Verification & Build
* **Global Build Check**: `go build ./...`



