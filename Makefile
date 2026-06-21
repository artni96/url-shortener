include .env
export

build_dir = ./cmd/shortener
dev_log_level=debug

PACKAGES ?= ./...
VERSION ?= N/A
BINARY=shorten-darwin

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_REMOTE := $(shell git rev-parse --short @{u} 2>/dev/null || echo "none")
GIT_PUSHED := $(if $(filter $(GIT_COMMIT),$(GIT_REMOTE)),true,false)

BINARY_COMMIT := $(shell ./$(BINARY) --version 2>/dev/null | grep Commit | cut -d':' -f2 | xargs || echo "none")
BINARY_PUSHED := $(shell ./$(BINARY) --version 2>/dev/null | grep Pushed | cut -d':' -f2 | xargs || echo "none")

.PHONY: help
help:
	@echo "Commands list:"
	@sed -n "s/^##//p" $(MAKEFILE_LIST) | column -t -s ":" | sed -e "s/^/ /"

.PHONY: check-darwin
check-darwin:
	@echo "Running custom static analysis for macOS"
	GOARCH=arm64 GOOS=darwin go vet -vettool=./cmd/staticlint/multicheck $(PACKAGES)

.PHONY: check-linux
check-linux:
	@echo "Running custom static analysis for linux"
	GOARCH=amd64 GOOS=linux go vet -vettool=./cmd/staticlint/multicheck $(PACKAGES)

.PHONY: build-darwin
build-darwin:
	@echo "Building the app for darwin"
	GOARCH=arm64 GOOS=darwin go build -ldflags="-X main.buildVersion=$(VERSION) -X 'main.buildDate=$$(date +'%Y/%m/%d %H:%M:%S')' -X main.buildCommit=$(COMMIT)" -o ./shortener-darwin ./cmd/shortener