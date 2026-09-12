.DEFAULT_GOAL := build

GO ?= go
ARGS ?=

.PHONY: build run test vet check fmt clean help

build:
	mkdir -p bin
	$(GO) build -o bin/resume2tex ./cmd/app

run:
	$(GO) run ./cmd/app $(ARGS)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test vet

fmt:
	$(GO) fmt ./...

clean:
	rm -rf bin

help:
	@printf '%s\n' \
		'build  Build bin/resume2tex (default target)' \
		'run    Run from source; pass CLI flags with ARGS="..."' \
		'test   Run Go regression tests' \
		'vet    Run go vet' \
		'check  Run test and vet' \
		'fmt    Format Go sources' \
		'clean  Remove bin/ only' \
		'help   Show this target list' \
		'' \
		'Variables: GO (default: go), ARGS (default: empty)' \
		'' \
		'Build and check require Go; TeX is only required for PDF output.' \
		'make run defaults to PDF and requires pdflatex on PATH.' \
		'Without TeX: make run ARGS="-format tex" or ARGS="-validate"' \
		'Other engines: make run ARGS="-engine xelatex" or ARGS="-engine lualatex"' \
		'Tectonic: brew install tectonic; make run ARGS="-engine tectonic -timeout 10m"' \
		'Tectonic caches downloaded support files; initial setup may need a longer timeout.' \
		'Install the selected engine and template packages; see README.md.'
