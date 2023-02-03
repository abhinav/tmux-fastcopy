BIN = bin
GO_FILES = $(shell find . -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' ')' -print)

TOOLS_GO_FILES = $(shell find tools -type f -a -name '*.go')

TMUX_FASTCOPY = $(BIN)/tmux-fastcopy

REVIVE = bin/revive
MOCKGEN = $(BIN)/mockgen
STATICCHECK = $(BIN)/staticcheck
TOOLS = $(REVIVE) $(STATICCHECK) $(MOCKGEN)

PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
export GOBIN ?= $(PROJECT_ROOT)/$(BIN)

.PHONY: all
all: build lint test

.PHONY: build
build: $(TMUX_FASTCOPY)

$(TMUX_FASTCOPY): $(GO_FILES)
	go install github.com/abhinav/tmux-fastcopy

.PHONY: generate
generate: $(MOCKGEN)
	PATH=$(GOBIN):$$PATH go generate -x ./...

$(MOCKGEN): tools/go.mod
	cd tools && go install github.com/golang/mock/mockgen

.PHONY: tools
tools: $(TOOLS)

.PHONY: test
test: $(GO_FILES)
	go test -v -race ./...

.PHONY: cover
cover: $(GO_FILES)
	go test -v -race -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: lint
lint: gofmt revive staticcheck gomodtidy nogenerate

.PHONY: gofmt
gofmt:
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@gofmt -e -s -l $(GO_FILES) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || \
		(echo "gofmt failed. Please reformat the following files:" | \
		cat - $(FMT_LOG) && false)

.PHONY: revive
revive: $(REVIVE)
	$(REVIVE) -config revive.toml ./...
	cd tools && ../$(REVIVE) -config ../revive.toml ./...

$(REVIVE): tools/go.mod
	cd tools && go install github.com/mgechev/revive

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	$(STATICCHECK) ./...
	cd tools && ../$(STATICCHECK) ./...

$(STATICCHECK): tools/go.mod
	cd tools && go install honnef.co/go/tools/cmd/staticcheck

.PHONY: gomodtidy
gomodtidy: go.mod go.sum tools/go.mod tools/go.sum
	go mod tidy
	cd tools && go mod tidy
	@if ! git diff --quiet $^; then \
		echo "go mod tidy changed files:" && \
		git status --porcelain $^ && \
		false; \
	fi

.PHONY: nogenerate
nogenerate:
	make generate
	@if ! git diff --quiet; then \
		echo "working tree is dirty after generate:" && \
		git status --porcelain && \
		false; \
	fi
