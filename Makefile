SHELL := bash
.SHELLFLAGS := -e -o pipefail -c
MAKEFLAGS += --warn-undefined-variables

all: generate lint test

.PHONY: test
test: generate lint
	go test -count=1 ./...
	go test -count=1 -race -cover ./...
	go run cmd/ugo/main.go -timeout 20s cmd/ugo/testdata/fibtc.ugo

.PHONY: generate
generate:
	go generate ./...

.PHONY: lint
lint:
	golint -set_exit_status ./...
	staticcheck ./...

