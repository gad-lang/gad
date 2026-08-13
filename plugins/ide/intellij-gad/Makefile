# Makefile for the Gad IntelliJ Platform plugin.
#
# Thin wrapper over the Gradle build (the IntelliJ Platform Gradle plugin).
# The first Gradle build downloads the target IDE SDK (~1 GB).

GRADLE ?= ./gradlew
# Prefer a JDK 21 for the toolchain; override with `make JAVA_HOME=/path build`.
JAVA_HOME ?= $(firstword $(wildcard /usr/lib/jvm/java-21-openjdk-amd64 /usr/lib/jvm/temurin-21-jdk-amd64 $(JAVA_HOME)))
export JAVA_HOME
GRADLE_FLAGS ?= --console=plain

.DEFAULT_GOAL := help

## help: list the available targets
.PHONY: help
help:
	@echo "Gad IntelliJ plugin — make targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

## compile: compile the Kotlin sources only (fast sanity check)
.PHONY: compile
compile:
	$(GRADLE) $(GRADLE_FLAGS) compileKotlin

## build: assemble the distributable plugin .zip (build/distributions)
.PHONY: build
build:
	$(GRADLE) $(GRADLE_FLAGS) buildPlugin

## test: run the unit tests
.PHONY: test
test:
	$(GRADLE) $(GRADLE_FLAGS) test

## verify: run the JetBrains Plugin Verifier (API compatibility)
.PHONY: verify
verify:
	$(GRADLE) $(GRADLE_FLAGS) verifyPlugin

## check: compile + test + verify (the full validation pipeline)
.PHONY: check
check:
	$(GRADLE) $(GRADLE_FLAGS) check verifyPlugin

## run: launch a sandbox IDE with the plugin installed
.PHONY: run
run:
	$(GRADLE) $(GRADLE_FLAGS) runIde

## sign: sign the plugin .zip (needs CERTIFICATE_CHAIN / PRIVATE_KEY / PRIVATE_KEY_PASSWORD)
.PHONY: sign
sign:
	$(GRADLE) $(GRADLE_FLAGS) signPlugin

## publish: publish to the JetBrains Marketplace (needs PUBLISH_TOKEN)
.PHONY: publish
publish:
	$(GRADLE) $(GRADLE_FLAGS) publishPlugin

## clean: remove build outputs
.PHONY: clean
clean:
	$(GRADLE) $(GRADLE_FLAGS) clean
