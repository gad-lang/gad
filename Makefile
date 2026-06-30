SHELL       := bash
.SHELLFLAGS := -e -o pipefail -c
MAKEFLAGS   += --warn-nil-variables
GOTOOLCHAIN=go1.25.0+auto

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
# all under ./dist. Requires Node/pnpm.
.PHONY: dist
dist: web-build build-vscode-plugin
	go build -tags prod -o ./dist/gad ./cmd/gad
	@echo "dist artifacts:" && ls -1 dist

# Build the VS Code extension: regenerate the TextMate grammar from the language
# vocabulary, compile and package the .vsix, then move it into ./dist.
.PHONY: build-vscode-plugin
build-vscode-plugin:
	go run ./cmd/update-vscode-plugin -w
	cd editors/vscode-gad && $(NVM_USE) && pnpm install && pnpm run package
	mkdir -p dist
	mv editors/vscode-gad/vscode-gad.vsix dist/

# Build the Gad WASM module (and copy Go's wasm_exec.js) into web/app/public.
.PHONY: build-wasm
build-wasm:
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
# Use Node v26.3.0 via nvm when available; always use pnpm.
NVM_USE := { [ -s "$$HOME/.nvm/nvm.sh" ] && . "$$HOME/.nvm/nvm.sh" && nvm use v26.3.0 >/dev/null; } || true

.PHONY: web-install
web-install:
	cd web && $(NVM_USE) && pnpm install

# Build and run the Vite dev server (right: editor, left: formatted/output).
# The WASM example works standalone; for the "Go server" example also run
# `make web-server` in another terminal.
.PHONY: web
web: web-install
	cd web/app && $(NVM_USE) && pnpm run dev

# Run the Go backend (API at /api/*, also serves web/app/dist when built).
.PHONY: web-server
web-server:
	go run ./web/server -addr :8080 -static web/app/dist

# Production build of the React app (outputs web/app/dist).
.PHONY: web-build
web-build: web-install
	cd web/app && $(NVM_USE) && pnpm run build

# Launch the bundled web IDE for the samples workspace (override with DIR=path).
DIR ?= samples
.PHONY: ide
ide:
	go run ./cmd/gad ide $(DIR)

# Generate Markdown docs for the samples workspace (writes $(DIR)/doc).
.PHONY: samples-doc
samples-doc:
	cd $(DIR) && go run ../cmd/gad doc

# Launch the IDE with the richer React + CodeMirror UI (builds web/app first).
.PHONY: ide-react
ide-react: web-build
	go run ./cmd/gad ide --static web/app/dist $(DIR)

.PHONY: test
test: version generate lint
	GOTOOLCHAIN=go1.25.0+auto go test -count=1 -cover ./...
	GOTOOLCHAIN=go1.25.0+auto go test -count=1 -race -coverpkg=./... ./...
	GOTOOLCHAIN=go1.25.0+auto go run ./cmd/gad -timeout 20s cmd/gad/testdata/fibtc.gad

.PHONY: generate-all
generate-all: generate generate-docs

.PHONY: generate
generate: version
	go generate ./...

.PHONY: lint
lint: version check-delve
	staticcheck -checks all,-SA1019,-ST1000 ./...
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