# Makefile for @gad-lang/ide-react. Thin wrapper over the bun scripts.
BUN ?= bun
.DEFAULT_GOAL := help

## help: list the available targets
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

## install: install dependencies
.PHONY: install
install:
	$(BUN) install

## build: compile TypeScript into dist/
.PHONY: build
build: install
	$(BUN) run build

## typecheck: type-check without emitting
.PHONY: typecheck
typecheck: install
	$(BUN) run typecheck

## clean: remove build output
.PHONY: clean
clean:
	$(BUN) run clean
