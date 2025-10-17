GOCMD             := go
GOCACHE_DIR      ?= $(CURDIR)/.cache/go-build
GOMODCACHE_DIR   ?= $(CURDIR)/.cache/go-mod
BINARY_NAME      ?= gitlab-mcp-server
PACKAGE_RUN       := ./cmd/server

GO_ENV            := CGO_ENABLED=0 GOCACHE=$(GOCACHE_DIR) GOMODCACHE=$(GOMODCACHE_DIR)
GOTEST            := $(GO_ENV) $(GOCMD) test ./...
GOBUILD           := $(GO_ENV) $(GOCMD) build
GORUN             := $(GO_ENV) $(GOCMD) run $(PACKAGE_RUN)

.PHONY: deps fmt lint test build run clean

deps:
	$(GO_ENV) $(GOCMD) mod tidy

fmt:
	$(GOCMD) fmt ./...

lint:
	$(GO_ENV) $(GOCMD) vet ./...

test:
	$(GOTEST)

build:
	$(GOBUILD) -o $(CURDIR)/$(BINARY_NAME) $(PACKAGE_RUN)

run:
	$(GORUN)

clean:
	rm -f $(CURDIR)/$(BINARY_NAME)
