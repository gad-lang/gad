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

# Build the distributable Gad WASM module (gad.wasm, debugger-enabled) and
# wasm_exec.js into ./dist.
.PHONY: build-wasm
build-wasm:
	bash scripts/build-wasm.sh ./dist

# Build the Gad WASM module into the React app's public/ (the in-browser IDE).
.PHONY: build-wasm-app
build-wasm-app:
	bash web/app/scripts/build-wasm.sh

# --- Documentation website (cmd/build-website) -----------------------------
# Output directory and serve address (override on the command line if needed,
# e.g. `make website WEBSITE_ADDR=:9000`).
WEBSITE_OUT  ?= dist/website
WEBSITE_ADDR ?= :8090

# Build the docs website (with the WASM playground + the embedded ide-vuetify
# demo, which needs bun) and serve it locally. Open http://localhost:8090.
.PHONY: website
website: generate-docs
	go run ./cmd/build-website build --out $(WEBSITE_OUT)
	@echo "Serving on http://localhost$(WEBSITE_ADDR) (Ctrl-C to stop)"
	go run ./cmd/build-website serve --out $(WEBSITE_OUT) --addr $(WEBSITE_ADDR)

# Fast iteration: build the docs website WITHOUT the WASM module / embedded demo
# (the Playground menu falls back to the simple in-page playground), then serve
# it. Much faster for content/layout tweaks; re-run to pick up changes.
.PHONY: website-fast
website-fast:
	go run ./cmd/build-website build --out $(WEBSITE_OUT) --no-wasm
	@echo "Serving on http://localhost$(WEBSITE_ADDR) (Ctrl-C to stop)"
	go run ./cmd/build-website serve --out $(WEBSITE_OUT) --addr $(WEBSITE_ADDR)

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
# The app imports the workspace packages (@gad-lang/codemirror-gad, prism-gad,
# ide-react) as `workspace:*`, so build those first (their dist/ .d.ts are needed
# by the app's tsc); the app's own `prebuild` regenerates src/samples.gen.ts.
.PHONY: web-build
web-build: web-install
	cd web && bun run plugins:build
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

# Generate the official sample docs from samples/*.{gad,gadt,gadx} into
# doc/samples, using the repo's official Markdown template (doc-templates/md.gadx,
# identical to the embedded default). Snippets and their /**= … **/ / /**< … **/
# results are executed and verified during generation. `.` (non-recursive) keeps
# it to the top-level numbered samples, not the sub-workspaces.
.PHONY: samples-doc
.PHONY: generate-api
# Regenerate the documented public-API stub samples (samples/**/<module>.gad) from
# the gad:doc comments in the Go source. The stub-generation commands live as
# //go:generate directives in gaddoc_generate.go (single source of truth); this
# target runs just those (via `-run gaddoc`), skipping the heavier code
# generators (mkcallable/update-delve). samples-doc renders their .md afterwards.
generate-api: version
	go generate -run gaddoc ./...

samples-doc: generate-api
	cd samples && go run ../cmd/gad doc --out ../doc/samples \
		--doc-template-md ../doc-templates/md.gadx .

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
	# GOTOOLCHAIN pins the Go toolchain so staticcheck's analysis matches the
	# language version the code targets (a staticcheck built against an older Go
	# reports "requires newer Go version" instead of real findings).
	GOTOOLCHAIN=go1.26.5 staticcheck -checks all,-SA1019,-ST1000,-ST1020 ./...
	go vet ./...

.PHONY: generate-docs
generate-docs: version samples-doc
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
	rm -f cmd/gad/gad cmd/gad/gad.exe dist/*


.PHONY: ci
ci:
	./scripts/golangci.sh