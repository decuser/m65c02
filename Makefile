# m65c02 build and verification entry points. Go 1.16-compatible.
GO ?= go
PROJECT := m65c02
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
OUT ?= ..
INCLUDE_GIT ?= 1

.PHONY: all build test validate bench vet fmt baseline clean help

all: test

# Until Go sources exist, these targets explicitly report that there is
# nothing to run. They do not claim to have tested an implementation.
build:
	@if find . -name '*.go' -not -path './.git/*' -print -quit | grep -q .; then \
		$(GO) build ./...; \
	else echo 'build: no Go packages yet'; fi

test:
	@if find . -name '*.go' -not -path './.git/*' -print -quit | grep -q .; then \
		$(GO) test ./...; \
	else echo 'test: no Go tests or packages yet'; fi

vet:
	@if find . -name '*.go' -not -path './.git/*' -print -quit | grep -q .; then \
		$(GO) vet ./...; \
	else echo 'vet: no Go packages yet'; fi

fmt:
	@if find . -name '*.go' -not -path './.git/*' -print -quit | grep -q .; then \
		$(GO) fmt ./...; \
	else echo 'fmt: no Go packages yet'; fi

validate: build vet test
	@echo 'validate: complete (external conformance corpus not yet installed)'

bench:
	@if find . -name '*_test.go' -not -path './.git/*' -print -quit | grep -q .; then \
		$(GO) test -run '^$$' -bench . -benchmem ./...; \
	else echo 'bench: no Go benchmarks yet'; fi

# Build a nested, normalized archive. Output goes outside the project root.
# INCLUDE_GIT=0 omits Git metadata when a source-only snapshot is desired.
baseline:
	@mkdir -p '$(OUT)'
	@set -eu; \
	name='$(PROJECT)-baseline-$(VERSION)'; \
	out=$$(cd '$(OUT)' && pwd)/$$name.tar.gz; \
	git_exclude=''; \
	if [ '$(INCLUDE_GIT)' = 0 ]; then git_exclude='--exclude=$(PROJECT)/.git'; fi; \
	cd ..; \
	tar --sort=name --mtime='@0' --owner=0 --group=0 --numeric-owner \
		--exclude='$(PROJECT)/*.tar.gz' \
		--exclude='$(PROJECT)/.cache' --exclude='$(PROJECT)/coverage.out' \
		--exclude='$(PROJECT)/*.prof' --exclude='$(PROJECT)/*.test' \
		--exclude='$(PROJECT)/dist' \
		$$git_exclude -cf - '$(PROJECT)' | gzip -n > "$$out"; \
	echo "baseline: $$out"

clean:
	@rm -f coverage.out cpu.prof mem.prof

help:
	@printf '%s\n' 'build: compile Go packages' 'test: quick Go tests' \
		'validate: build, vet, and test (add conformance checks when available)' \
		'bench: run Go benchmarks' 'baseline: nested project source archive' \
		'fmt: go fmt' 'clean: remove generated profiling files'

.PHONY: run
run:
	$(GO) run ./cmd/m65c02-monitor
