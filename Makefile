include .env
export

build_dir = ./cmd/shortener
dev_log_level=debug

PACKAGES ?= ./...

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