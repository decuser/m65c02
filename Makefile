# m65c02 build and verification entry points. Go 1.16-compatible.
GO ?= go
PROJECT := m65c02
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
OUT ?= ..
INCLUDE_GIT ?= 1
TESTS_DIR ?= ../tests
SYNERTEK_DIR ?= $(TESTS_DIR)/65x02/synertek65c02/v1
DORMANN_DIR ?= $(TESTS_DIR)/6502_65C02_functional_tests/bin_files
DORMANN_PROFILE_DIR ?= $(TESTS_DIR)/dormann-profile

.PHONY: all build test validate validate-full bench vet fmt baseline clean help

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
	@if [ -n "$${M65C02_CORPUS:-}" ]; then \
		echo 'validate: selected Synertek corpus supplied; results above'; \
	else echo 'validate: selected Synertek corpus unavailable (set M65C02_CORPUS)'; fi
	@if [ -n "$${M65C02_DORMANN:-}" ]; then \
		echo 'validate: Dormann documented 6502 binary supplied; result above'; \
	else echo 'validate: Dormann functional binary unavailable (set M65C02_DORMANN)'; fi
	@if [ -n "$${M65C02_DORMANN_EXTENDED:-}" ]; then \
		echo 'validate: profile-configured Dormann 65C02 binary supplied; result above'; \
	else echo 'validate: Dormann 65C02 profile binary unavailable (set M65C02_DORMANN_EXTENDED)'; fi
	@if [ -n "$${M65C02_DORMANN_INTERRUPT:-}" ]; then \
		echo 'validate: CMOS Dormann interrupt binary supplied; result above'; \
	else echo 'validate: Dormann CMOS interrupt binary unavailable (set M65C02_DORMANN_INTERRUPT)'; fi

# Full M1 external gate. All three Dormann fixtures are required.
validate-full: build vet test
	@test -f '$(SYNERTEK_DIR)/00.json' -a -f '$(SYNERTEK_DIR)/ff.json' || { \
		echo 'validate-full: missing Synertek JSON files in $(SYNERTEK_DIR)'; exit 1; }
	@test -f '$(DORMANN_DIR)/6502_functional_test.bin' || { \
		echo 'validate-full: missing Dormann binary in $(DORMANN_DIR)'; exit 1; }
	@test -f '$(DORMANN_PROFILE_DIR)/65C02_profile.bin' || { \
		echo 'validate-full: missing profile 65C02 binary in $(DORMANN_PROFILE_DIR)'; exit 1; }
	@test -f '$(DORMANN_PROFILE_DIR)/6502_interrupt_cmos.bin' || { \
		echo 'validate-full: missing CMOS interrupt binary in $(DORMANN_PROFILE_DIR)'; exit 1; }
	@set -eu; corpus=$$(cd '$(SYNERTEK_DIR)' && pwd); \
	for group in '[0-3]' '[4-7]' '[8-b]' '[c-f]'; do \
		echo "validate-full: Synertek group $$group (640,000 vectors)"; \
		M65C02_CORPUS="$$corpus" M65C02_CORPUS_LIMIT=10000 \
			$(GO) test ./internal/cpu -run "^TestSynertekAllOpcodes$$/$$group[0-9a-f]$$" -count=1 -timeout=20m; \
	done
	@M65C02_DORMANN=$$(cd '$(DORMANN_DIR)' && pwd) \
		$(GO) test ./internal/cpu -run '^TestDormann$$' -count=1 -timeout=10m
	@M65C02_DORMANN_EXTENDED=$$(cd '$(DORMANN_PROFILE_DIR)' && pwd)/65C02_profile.bin \
		$(GO) test ./internal/cpu -run '^TestDormannProfile65C02$$' -count=1 -timeout=10m
	@M65C02_DORMANN_INTERRUPT=$$(cd '$(DORMANN_PROFILE_DIR)' && pwd)/6502_interrupt_cmos.bin \
		$(GO) test ./internal/cpu -run '^TestDormannInterrupt$$' -count=1 -timeout=10m
	@echo 'validate-full: required corpus runs completed'

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
		'validate: build, vet, and available conformance tests' \
		'validate-full: all Synertek and three Dormann tests (../tests)' \
		'bench: run Go benchmarks' 'baseline: nested project source archive' \
		'fmt: go fmt' 'clean: remove generated profiling files' \
        'run-prod: execute production CPU demonstration'

.PHONY: run
run:
	$(GO) run ./cmd/m65c02-monitor

.PHONY: run-prod
run-prod:
	$(GO) run ./cmd/m65c02-prod
