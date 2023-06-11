SHELL = /bin/bash

PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Setting GOBIN and PATH ensures two things:
# - All 'go install' commands we run
#   only affect the current directory.
# - All installed tools are available on PATH
#   for commands like go generate.
export GOBIN = $(PROJECT_ROOT)/bin
export PATH := $(GOBIN):$(PATH)

GO_MODULES ?= $(shell find . \
	-path '*/.*' -prune -o \
	-type f -a -name 'go.mod' -printf '%h\n')

TEST_FLAGS ?= -v -race

# Non-test Go files.
GO_SRC_FILES = $(shell find . \
	   -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' -a -not -name '*_test.go' ')' -print)

TMUX_FASTCOPY = bin/tmux-fastcopy
MOCKGEN = bin/mockgen

.PHONY: all
all: build lint test

.PHONY: build
build: $(TMUX_FASTCOPY)

$(TMUX_FASTCOPY): $(GO_SRC_FILES)
	go install github.com/abhinav/tmux-fastcopy

.PHONY: generate
generate: $(MOCKGEN)
	go generate -x ./...

$(MOCKGEN): go.mod
	go install github.com/golang/mock/mockgen

.PHONY: test
test:
	go test $(TEST_FLAGS) ./...

.PHONY: test-integration
test-integration: $(TMUX_FASTCOPY)
	go test -C integration $(TEST_FLAGS) ./...

.PHONY: cover
cover: export GOEXPERIMENT = coverageredesign
cover:
	go test $(TEST_FLAGS) -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: cover-integration
cover-integration: export GOEXPERIMENT = coverageredesign
cover-integration:
	$(eval BIN := $(shell mktemp -d))
	$(eval COVERDIR := $(shell mktemp -d))
	GOBIN=$(BIN) \
	      go install -cover -coverpkg=./... github.com/abhinav/tmux-fastcopy
	GOCOVERDIR=$(COVERDIR) PATH=$(BIN):$$PATH \
		   go test -C integration $(TEST_FLAGS) ./...
	go tool covdata textfmt -i=$(COVERDIR) -o=cover.integration.out
	go tool cover -html=cover.integration.out -o cover.integration.html

.PHONY: lint
lint: golangci-lint tidy-lint generate-lint

.PHONY: fmt
fmt:
	gofumpt -w .

.PHONY: golangci-lint
golangci-lint:
	$(foreach mod,$(GO_MODULES), \
		(cd $(mod) && golangci-lint run --path-prefix $(mod)) &&) true

.PHONY: tidy
tidy:
	$(foreach mod,$(GO_MODULES),(cd $(mod) && go mod tidy) &&) true

.PHONY: tidy-lint
tidy-lint:
	$(foreach mod,$(GO_MODULES), \
		(cd $(mod) && go mod tidy && \
			git diff --exit-code -- go.mod go.sum || \
			(echo "[$(mod)] go mod tidy changed files" && false)) &&) true

.PHONY: generate-lint
generate-lint:
	make generate
	@if ! git diff --quiet; then \
		echo "working tree is dirty after generate:" && \
		git status --porcelain && \
		false; \
	fi
