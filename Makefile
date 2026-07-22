.PHONY: all build clean fmt vet test lint help

BINARY_DIR := bin
DAEMON_BIN := $(BINARY_DIR)/whatsd
CLI_BIN := $(BINARY_DIR)/whatsctl

all: build

build:
	@mkdir -p $(BINARY_DIR)
	go build -o $(DAEMON_BIN) ./cmd/whatsd
	go build -o $(CLI_BIN) ./cmd/whatsctl

clean:
	rm -rf $(BINARY_DIR)

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

lint: fmt vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed in PATH; ran go fmt and go vet."; \
	fi

help:
	@echo "whatsd Makefile targets:"
	@echo "  make build   Build whatsd daemon and whatsctl CLI binaries into bin/"
	@echo "  make clean   Remove built binaries in bin/"
	@echo "  make fmt     Format Go source code using go fmt"
	@echo "  make vet     Run static analysis using go vet"
	@echo "  make test    Run package test suites"
	@echo "  make lint    Run fmt, vet, and golangci-lint (if available)"
