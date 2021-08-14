BIN = bin
GO_FILES = $(shell find . -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' ')' -print)

TMUX_FASTCOPY = $(BIN)/tmux-fastcopy

GOLINT = $(BIN)/golint
MOCKGEN = $(BIN)/mockgen
STATICCHECK = $(BIN)/staticcheck
TOOLS = $(GOLINT) $(STATICCHECK) $(MOCKGEN)

export GOBIN ?= $(shell pwd)/$(BIN)

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
	go test -race ./...

.PHONY: cover
cover: $(GO_FILES)
	go test -race -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: lint
lint: gofmt golint staticcheck gomodtidy nogenerate

.PHONY: gofmt
gofmt:
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@gofmt -e -s -l $(GO_FILES) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || \
		(echo "gofmt failed. Please reformat the following files:" | \
		cat - $(FMT_LOG) && false)

.PHONY: golint
golint: $(GOLINT)
	$(GOLINT) ./...

$(GOLINT): tools/go.mod
	cd tools && go install golang.org/x/lint/golint

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	$(STATICCHECK) ./...

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
