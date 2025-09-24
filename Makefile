# Simple Makefile for github.com/szydell/go-sha256
# Builds standalone binaries (no cgo) into ./bin for:
# - Linux x86_64 (amd64)
# - AIX Power (ppc64)

SHELL := /bin/sh

# Binary output directory and name
BIN_DIR := bin
BINARY := go-sha256

# Common Go build flags
GO_BUILD := go build -trimpath -ldflags "-s -w"

.PHONY: all linux aix local test clean help

# Build both target platforms
all: linux aix

# Ensure bin directory exists
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Linux x86_64 (amd64)
linux: $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(BINARY)-linux-amd64 .
	@echo "Built $(BIN_DIR)/$(BINARY)-linux-amd64"

# AIX Power (ppc64) — suitable for Power9 on AIX
# Note: Cross-compiling to AIX requires Go toolchain support; cgo is disabled to keep it standalone.
aix: $(BIN_DIR)
	CGO_ENABLED=0 GOOS=aix GOARCH=ppc64 $(GO_BUILD) -o $(BIN_DIR)/$(BINARY)-aix-ppc64 .
	@echo "Built $(BIN_DIR)/$(BINARY)-aix-ppc64"

# Build for the current host OS/ARCH
local: $(BIN_DIR)
	CGO_ENABLED=0 $(GO_BUILD) -o $(BIN_DIR)/$(BINARY) .
	@echo "Built $(BIN_DIR)/$(BINARY) for local $(shell go env GOOS)/$(shell go env GOARCH)"

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)

help:
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  all     - build linux and aix binaries into ./bin"
	@echo "  linux   - build Linux amd64 binary into ./bin"
	@echo "  aix     - build AIX ppc64 binary into ./bin"
	@echo "  local   - build for the current host into ./bin"
	@echo "  test    - run 'go test ./...'"
	@echo "  clean   - remove ./bin directory"