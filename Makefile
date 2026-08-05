SHELL       := bash
.SHELLFLAGS := -e -o pipefail -c
MAKEFLAGS   += --warn-nil-variables
GOTOOLCHAIN=go1.26.5+auto

all: version depcheck generate lint test

depcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

# Default build: the CLI plus the WebAssembly module.
.PHONY: build
build: build-cli build-wasm

.PHONY: build-cli
build-cli:
	go build -tags prod -o ./dist/gad ./cmd/gad

# Minimal CLI: exclude the `ide` and `debug` subcommands (and their web/DAP
# dependencies) via build tags. Useful for small, embeddable binaries.
.PHONY: build-min
build-min:
	go build -tags 'noide nodebug' -o ./dist/gad-min ./cmd/gad

# Distribution build: the React web app + the gad binary with the embedded UI
# (`gad ide` serves it without --static) and the packaged VS Code extension,
# all under ./dist. Requires Node/bun.
.PHONY: dist
dist: web-build build-vscode-plugin build-wasm
	go build -tags prod -o ./dist/gad ./cmd/gad
	@echo "dist artifacts:" && ls -1 dist

# Prerequisites for goreleaser: web app and VS Code plugin only (no binary —
# goreleaser builds the binary itself). Keeps output out of ./dist/ so
# goreleaser's own dist/ directory stays empty before its build step.
.PHONY: goreleaser-setup
goreleaser-setup: web-build
	go run ./cmd/update-vscode-plugin -w
	cd editors/vscode-gad && bun install && bun run package

# Build the VS Code extension: regenerate the TextMate grammar from the language
# vocabulary, compile and package the .vsix, then move it into ./dist.
.PHONY: build-vscode-plugin
build-vscode-plugin:
	go run ./cmd/update-vscode-plugin -w
	cd editors/vscode-gad && bun install && bun run package
	mkdir -p dist
	mv editors/vscode-gad/vscode-gad.vsix dist/

# Build both distributable Gad WASM modules into ./dist (and copy wasm_exec.js):
#   gad.wasm       — normal build, no debugger (smaller)
#   gad_debug.wasm — includes the gadDebug* stepping protocol
.PHONY: build-wasm
build-wasm:
	bash scripts/build-wasm.sh ./dist both

# Only the normal (no-debugger) WASM module into ./dist.
.PHONY: build-wasm-normal
build-wasm-normal:
	bash scripts/build-wasm.sh ./dist normal

# Only the debugger-enabled WASM module (gad_debug.wasm) into ./dist.
.PHONY: build-wasm-debug
build-wasm-debug:
	bash scripts/build-wasm.sh ./dist debug

# Build the Gad WASM module into the React app's public/ (debugger-enabled, so
# the in-browser IDE keeps working), under the plain gad.wasm name it loads.
.PHONY: build-wasm-app
build-wasm-app:
	bash web/app/scripts/build-wasm.sh

# Regenerate the VM debug loop (vm_loop_debug.go) from the production loop.
.PHONY: gen-delve
gen-delve:
	go run ./cmd/update-delve gen

# Fail if the VM debug loop is out of date with the production loop.
.PHONY: check-delve
check-delve:
	go run ./cmd/update-delve check

# --- Web example (CodeMirror plugin + React app) ---------------------------
# Uses bun (which does not need nvm).

.PHONY: web-install
web-install:
	cd web && bun install

# Build and run the Vite dev server (right: editor, left: formatted/output).
# The WASM example works standalone; for the "Go server" example also run
# `make web-server` in another terminal.
.PHONY: web
web: web-install
	cd web/app && bun run dev

# Run the Go backend (API at /api/*, also serves web/app/dist when built).
.PHONY: web-server
web-server:
	go run ./web/server -addr :8080 -static web/app/dist

# Production build of the React app (outputs web/app/dist). Emits two pages:
# index.html (the playground) and webide.html (the standalone embeddable IDE).
.PHONY: web-build
web-build: web-install
	cd web/app && bun run build

# Dev server (hot reload) for the standalone, server-less IDE page. No Go backend
# is needed: the tree is read-only samples + a LocalStorage overlay, and
# run/doc/debug all run in-browser via the Gad WASM module in a Web Worker
# (webide.html never calls /api). Open the printed /webide.html URL.
.PHONY: webide
webide: web-install
	@echo "Open http://localhost:5173/webide.html"
	cd web/app && bun run dev

# Preview the production build of the standalone IDE page (also no Go backend).
# Builds web/app/dist once, then serves it; open the printed /webide.html URL.
.PHONY: webide-preview
webide-preview: web-build
	@echo "Open http://localhost:4173/webide.html"
	cd web/app && bun run preview

# Dev server for the server-less Vuetify IDE demo (@gad-lang/ide-vuetify/demo).
# Same server-less model as `webide`, with the reusable Vuetify <GadIde>. Builds
# the Gad WASM module into the demo's public/ first (it is git-ignored), then
# runs Vite (which regenerates the bundled samples via predev).
.PHONY: ide-vuetify-demo
ide-vuetify-demo: web-install
	cd web/ide-vuetify/demo && bun run wasm
	@echo "Open http://localhost:5173/"
	cd web/ide-vuetify/demo && bun run dev

# Preview the production build of the Vuetify IDE demo (also no Go backend).
.PHONY: ide-vuetify-demo-preview
ide-vuetify-demo-preview: web-install
	cd web/ide-vuetify/demo && bun run wasm && bun run build
	@echo "Open http://localhost:4173/"
	cd web/ide-vuetify/demo && bun run preview

# Dev server for the server-less React IDE demo (the reusable @gad-lang/ide-react
# <Ide> driven by the in-browser backend). It is the webide.html entry of the
# web/app project; `bun run dev` builds the Gad WASM module first.
.PHONY: ide-react-demo
ide-react-demo: web-install
	@echo "Open http://localhost:5173/webide.html"
	cd web/app && bun run dev

# Preview the production build of the React IDE demo (also no Go backend).
.PHONY: ide-react-demo-preview
ide-react-demo-preview: web-build
	@echo "Open http://localhost:4173/webide.html"
	cd web/app && bun run preview

# Launch the IDE with the React + CodeMirror UI (builds web/app first).
# Override the workspace with DIR=path (defaults to samples).
DIR ?= samples
.PHONY: ide
ide: web-build
	go run ./cmd/gad ide --static web/app/dist $(DIR)

# Generate Markdown docs for the samples workspace (writes $(DIR)/doc).
.PHONY: samples-doc
samples-doc:
	cd $(DIR) && go run ../cmd/gad doc

.PHONY: test
test: version generate lint
	go test -count=1 -cover ./...
	go test -count=1 -race -coverpkg=./... ./...
	go run ./cmd/gad -timeout 20s cmd/gad/testdata/fibtc.gad

.PHONY: generate-all
generate-all: generate generate-docs

.PHONY: generate
generate: version
	go generate ./...

.PHONY: lint
lint: version check-delve
	# -ST1000/-ST1020: the codebase uses intentional descriptive/group doc
	# comments (e.g. one comment heading a type's operator-method group) that do
	# not start with the symbol name.
	staticcheck -checks all,-SA1019,-ST1000,-ST1020 ./...
	go vet ./...

.PHONY: generate-docs
generate-docs: version
	# time, fmt and strings are builtin module namespaces in the root package;
	# the 3rd arg selects which module's gad:doc to emit.
	go run ./cmd/gaddoc . ./doc/stdlib-time.md time
	go run ./cmd/gaddoc . ./doc/stdlib-fmt.md fmt
	go run ./cmd/gaddoc . ./doc/stdlib-strings.md strings
	go run ./cmd/gaddoc ./stdlib/json ./doc/stdlib-json.md

.PHONY: version
version:
	@go version

.PHONY: clean
clean:
	find . -type f \( -name "cpu.out" -o -name "*.test" -o -name "mem.out" \) -delete
	rm -f cmd/gad/gad cmd/gad/gad.exe


.PHONY: ci
ci:
	./scripts/golangci.sh


xx:
	echo "param(*argv); println(repr(argv))" | go run ./cmd/gad -- x