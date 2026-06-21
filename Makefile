include .env
export

build_dir = ./cmd/shortener
dev_log_level=debug

PACKAGES ?= ./...
VERSION ?= N/A
COMMIT := $(shell git rev-parse --short HEAD)

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

.PHONY: run-reset
run-reset:
	@echo "running reset"
	go run ./cmd/reset

.PHONY: build-darwin
build-darwin:
	@echo "Building app for darwin"
	GOARCH=arm64 GOOS=darwin go build -ldflags="-X main.buildVersion=$(VERSION) -X 'main.buildDate=$$(date +'%Y/%m/%d %H:%M:%S')' -X main.buildCommit=$(COMMIT)" -o ./shortener-darwin ./cmd/shortener

.PHONY: build-linux
build-linux:
	@echo "Building app for darwin"
	GOARCH=amd64 GOOS=linux go build -ldflags="-X main.buildVersion=$(VERSION) -X 'main.buildDate=$$(date +'%Y/%m/%d %H:%M:%S')' -X main.buildCommit=$(COMMIT)" -o ./shortener-darwin ./cmd/shortener
