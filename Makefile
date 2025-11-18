SHELL       := bash
.SHELLFLAGS := -e -o pipefail -c
MAKEFLAGS   += --warn-nil-variables

all: version depcheck generate lint test

depcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest

build-cli:
	go build -o ./dist/gad ./cmd/gad

.PHONY: test
test: version generate lint
	go test -count=1 -cover ./...
	go test -count=1 -race -coverpkg=./... ./...
	go run cmd/gad/main.go -timeout 20s cmd/gad/testdata/fibtc.gad

.PHONY: generate-all
generate-all: generate generate-docs

.PHONY: generate
generate: version
	go generate ./...

.PHONY: lint
lint: version
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

