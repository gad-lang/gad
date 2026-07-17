# CLAUDE.md - Development Context (giom)

Style guide, commands, and architectural guidelines for developing the `giom` template engine.

## 🛠️ Build and Test Commands
Always use the native Go 1.25 toolchain commands:

* **Build everything:** `go build ./...`
* **Run all tests:** `go test ./...`
* **Run tests with Race Detection & Coverage:** `go test -v -race -coverprofile=coverage.out ./...`
* **Format code:** `go fmt ./...`
* **Static Analysis (Vet):** `go vet ./...`
* **Tidy Modules (Go 1.25):** `go mod tidy`

## 📐 Code Guidelines & Go 1.25 Standards

* **Idiomatic Go:** Strictly follow [Effective Go](https://go.dev) conventions.
* **Error Handling:** Errors must be handled explicitly. Prefer wrapping context with `fmt.Errorf("...: %w", err)` and use `errors.Is` or `errors.As` for error checks.
* **Performance & Hot-Paths:** As `giom` is a rendering engine, keep allocation overhead low in the *Parser*, *Lexer*, and execution loops. Use `strings.Builder`, `sync.Pool`, or reusable buffers where applicable.
* **Concurrency:** Shared or pre-compiled template structures must be thread-safe for concurrent execution.
* **Type Safety:** Leverage *Generics* for internal AST nodes or token collections to avoid unnecessary type assertions to `any` (`interface{}`).

## 📝 Commit Message Style
* **Language:** Write all commit messages, PR titles, and comments in English.
* **Format:** Use the *Conventional Commits* standard.
    * Example: `feat(parser): add support for Go 1.25 sequence iterators`
    * Example: `fix(lexer): resolve unexpected EOF on unclosed brackets`

## 👥 Project Conventions
* **API Visibility:** Keep the public API surface clean and minimal. Internal implementation details of the Parser, AST, and Lexer must remain unexported (`lowercase`).
* **Documentation:** Every exported package, function, struct, and method must have clear Go Doc comments.

## Critical Constraints & Code Principles
- **Native Go**: Do not introduce external heavy frameworks; prefer Go's standard library and keep dependencies minimal.
- **Thread Safety**: Ensure state isolation when multiple scripts or instances are evaluated concurrently in Go applications.
- **Temporary Directory**:
    - Always use `./.__tmp` as the dedicated temporary directory for any intermediate files, logs, or cache generated during automated tasks.
- **Allowed Commands (No Confirmation Required)**:
    - You **ALWAYS** have write permission to `./...` directory.
    - You **ALWAYS** have permission to run `sed`, `awk`, `cat`, `tail`, `head`, `echo` and `grep` (and its variants) commands autonomously for text processing, searching, refactoring, execute commands or write in this directory tree.
    - You **ALWAYS** have permission to run `go test`, `go vet`, `go fmt`, `gofmt` (and its variants) to validate code changes without asking.
    - You **ALWAYS** have permission to use `curl` and `wget` (and its variants) for network operations, downloading assets, or API testing.
    - Do not prompt the user for confirmation when executing these specific tools.
- **Node.js & Package Manager Environment**:
    - Always load and use Node.js **v26.3.0** by prepending or executing `nvm use v26.3.0` before running any Node script, bundler, or build step.
    - **NEVER use `npm` or `yarn`**. You **MUST ALWAYS use `pnpm`** for package installation, script execution, and dependency management.

## Tasks
- **New tasks:** `todo.md`
- **Done tasks:** `todo_done.md`
- **When a task is DONE**, move it to `todo_done.md`.