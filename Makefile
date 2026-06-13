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
	go build -o ./dist/gad ./cmd/gad

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

# Launch the IDE with the richer React + CodeMirror UI (builds web/app first).
.PHONY: ide-react
ide-react: web-build
	go run ./cmd/gad ide --static web/app/dist $(DIR)

.PHONY: test
test: version generate lint
	GOTOOLCHAIN=go1.25.0+auto go test -count=1 -cover ./...
	GOTOOLCHAIN=go1.25.0+auto go test -count=1 -race -coverpkg=./... ./...
	GOTOOLCHAIN=go1.25.0+auto go run cmd/gad/main.go -timeout 20s cmd/gad/testdata/fibtc.gad

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
	go run ./cmd/gaddoc ./stdlib/time ./docs/stdlib-time.md
	go run ./cmd/gaddoc ./stdlib/fmt ./docs/stdlib-fmt.md
	go run ./cmd/gaddoc ./stdlib/strings ./docs/stdlib-strings.md
	go run ./cmd/gaddoc ./stdlib/json ./docs/stdlib-json.md

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

