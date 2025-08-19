#!/bin/bash

docker run --rm -t -v $(pwd):/app -w /app \
  --user $(id -u):$(id -g) \
  -v $(go env GOCACHE):/.cache/go-build -e GOCACHE=/.cache/go-build \
  -v $(go env GOMODCACHE):/.cache/mod -e GOMODCACHE=/.cache/mod \
  -v ~/.cache/goimports:/.cache/goimports \
  -v ~/.cache/golangci-lint:/.cache/golangci-lint -e GOLANGCI_LINT_CACHE=/.cache/golangci-lint \
  golangci/golangci-lint:v2.4.0 golangci-lint run