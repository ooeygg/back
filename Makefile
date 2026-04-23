GO ?= go
PKG ?= ./...

.DEFAULT_GOAL := help

.PHONY: help tidy fmt lint test build run-itemwatcher generate clean

help:
	@echo Available targets:
	@echo "  make tidy            - Run go mod tidy"
	@echo "  make fmt             - Format all Go files"
	@echo "  make lint            - List files that are not gofmt-formatted"
	@echo "  make test            - Run the full test suite"
	@echo "  make build           - Build all packages"
	@echo "  make run-itemwatcher - Run the item watcher tool"
	@echo "  make generate        - Regenerate static game data code"
	@echo "  make clean           - Clean Go build and test caches"

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w .

lint:
	gofmt -l .

test:
	$(GO) test $(PKG)

build:
	$(GO) build $(PKG)

run-itemwatcher:
	$(GO) run ./cmd/itemwatcher

generate:
	$(GO) run ./cmd/txttocode

clean:
	$(GO) clean -cache -testcache
