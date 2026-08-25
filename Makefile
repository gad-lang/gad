SHELL       := bash
.SHELLFLAGS := -e -o pipefail -c
MAKEFLAGS   += --warn-nil-variables
GOTOOLCHAIN=go1.26.5+auto

all: version depcheck generate lint test

depcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

# --- Submodules (editor plugins live in their own repos) -------------------
# submodules-init: check out every submodule (run once after a plain clone).
.PHONY: submodules-init
submodules-init:
	git submodule update --init --recursive

# submodules-update: fast-forward every submodule pointer to the tip of its
# tracked remote branch, then stage the bumped pointers. Review with
# `git submodule status` / `git diff --cached`, then commit.
.PHONY: submodules-update
submodules-update:
	git submodule update --remote --recursive
	git submodule status
	git add $$(git config --file .gitmodules --get-regexp path | awk '{ print $$2 }')

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
	git submodule update --init --recursive plugins/ide/vscode-gad
	go run ./cmd/update-vscode-plugin -w
	cd plugins/ide/vscode-gad && bun install && bun run package

# Build the VS Code extension: refresh the gad-textmate bundle submodule,
# regenerate the TextMate grammar into it from the current language vocabulary,
# compile and package the .vsix, then move it into ./dist.
.PHONY: build-vscode-plugin
build-vscode-plugin:
	git submodule update --init --recursive plugins/ide/vscode-gad
	go run ./cmd/update-vscode-plugin -w
	cd plugins/ide/vscode-gad && bun install && bun run package
	mkdir -p dist
	mv plugins/ide/vscode-gad/vscode-gad.vsix dist/

# --- Shared TextMate bundle (gad-lang/gad-textmate) ------------------------
# The gad.tmLanguage.json grammar is generated here (source of truth) and shared
# by the editor plugins via the gad-textmate repo. `textmate-publish` regenerates
# it and pushes to gad-textmate when it changed; the plugins then bump their
# submodule pointer. Uses your local git credentials (no CI secret needed).
TEXTMATE_REPO ?= git@github.com:gad-lang/gad-textmate.git
TEXTMATE_DIR  ?= .__tmp/gad-textmate
.PHONY: textmate-publish
textmate-publish:
	@if [ -d $(TEXTMATE_DIR)/.git ]; then git -C $(TEXTMATE_DIR) pull --ff-only; \
		else git clone $(TEXTMATE_REPO) $(TEXTMATE_DIR); fi
	go run ./cmd/update-vscode-plugin -print > $(TEXTMATE_DIR)/syntaxes/gad.tmLanguage.json
	@if [ -n "$$(git -C $(TEXTMATE_DIR) status --porcelain)" ]; then \
		git -C $(TEXTMATE_DIR) add syntaxes/gad.tmLanguage.json; \
		git -C $(TEXTMATE_DIR) commit -m "chore: regenerate gad.tmLanguage.json from gad@$$(git rev-parse --short HEAD)"; \
		git -C $(TEXTMATE_DIR) push origin HEAD:main; \
		echo "==> published updated grammar to gad-textmate"; \
	else echo "==> gad-textmate grammar already up to date"; fi

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

# Tokenize the generated TextMate grammar with the real vscode-textmate engine
# (the same stack VS Code and the IntelliJ TextMate bundle use) and assert the
# highlighting contract — notably interpolated-string `{ … }` islands. Needs bun.
.PHONY: grammar-test
grammar-test:
	cd cmd/internal/pluginsync/tmtest && bun install && bun test

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

# Build the JS editor plugins (@gad-lang/codemirror-gad and @gad-lang/prism-gad).
# They live as git submodules under web/plugins/js and are members of the web bun
# workspace, so a single `bun install` in web/ links them; build each so their
# dist/ is consumed by web (ide-react, app). Run `git submodule update --init`
# first if the submodules are not checked out.
.PHONY: plugins-js
plugins-js:
	cd web && bun install
	cd web/plugins/js/codemirror-gad && bun run build
	cd web/plugins/js/prism-gad && bun run build

# Production build of the React app (outputs web/app/dist). Emits two pages:
# index.html (the playground) and webide.html (the standalone embeddable IDE).
# The app imports the editor packages (@gad-lang/codemirror-gad, prism-gad,
# ide-react), so build those first (their dist/ .d.ts are needed by the app's
# tsc); the app's own `prebuild` regenerates src/samples.gen.ts.
.PHONY: web-build
web-build: plugins-js web-install
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
	# `./...` recurses into sample subdirectories (e.g. samples/class/) so their
	# .md render alongside the flat samples; `.` alone would skip them.
	cd samples && go run ../cmd/gad doc --out ../doc/samples \
		--doc-template-md ../doc-templates/md.gadx ./...

# Run the web workspace's JS/TS tests. `bun test` auto-discovers every
# *.test.ts under web/** (across all workspace members), so new tests are picked
# up without wiring. It errors out when there are none, so only invoke it once at
# least one test file exists. Needs bun.
.PHONY: web-test
web-test:
	cd web && bun install
	@if find web \( -name '*.test.ts' -o -name '*.test.tsx' -o -name '*.spec.ts' \) \
		-not -path '*/node_modules/*' | grep -q .; then \
		echo "==> web: bun test"; cd web && bun test; \
	else \
		echo "==> web: no test files yet (bun test auto-discovers web/**/*.test.ts)"; \
	fi

.PHONY: test
test: version generate lint
	go test -count=1 -cover ./...
	go test -count=1 -race -coverpkg=./... ./...
	go run ./cmd/gad -timeout 20s cmd/gad/testdata/fibtc.gad
	# Front-end + grammar: web workspace tests and the TextMate tokenization tests.
	$(MAKE) web-test grammar-test

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
	go run ./cmd/gaddoc . ./doc/stdlib/time.md time
	go run ./cmd/gaddoc . ./doc/stdlib/base64.md base64
	go run ./cmd/gaddoc . ./doc/stdlib/fmt.md fmt
	go run ./cmd/gaddoc . ./doc/stdlib/strings.md strings
	go run ./cmd/gaddoc ./stdlib/json ./doc/stdlib/json.md

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