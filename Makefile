# Trak - Development Environment Manager
# Makefile for building, testing, and installing

BINARY_NAME := trak
MODULE := github.com/laurent/trak
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Directories
CMD_DIR := ./cmd/trak
BUILD_DIR := ./build
INSTALL_DIR := $(HOME)/bin

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod

.PHONY: all build test test-verbose test-coverage install clean deps lint fmt check help

# Default target
all: deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for current platform (alias)
build-local: build

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -timeout 30s ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -timeout 30s -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -timeout 30s -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Run a specific package's tests
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./internal/db"; exit 1; fi
	$(GOTEST) -timeout 30s -v $(PKG)

# Install to ~/bin
install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(BINARY_NAME)"

# Uninstall from ~/bin
uninstall:
	@echo "Removing $(INSTALL_DIR)/$(BINARY_NAME)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

# Run all checks (format, lint, test)
check: fmt lint test

# Quick development build and install
dev: build install

# Run the TUI (for quick testing)
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Trak Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            - Download deps, run tests, build binary (default)"
	@echo "  build          - Build the binary to ./build/"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests and generate coverage report"
	@echo "  test-pkg       - Run tests for a specific package (PKG=./internal/db)"
	@echo "  install        - Build and install to ~/bin"
	@echo "  uninstall      - Remove from ~/bin"
	@echo "  clean          - Remove build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  fmt            - Format all Go code"
	@echo "  lint           - Run golangci-lint"
	@echo "  check          - Run fmt, lint, and tests"
	@echo "  dev            - Quick build and install for development"
	@echo "  run            - Build and run the TUI"
	@echo "  help           - Show this help message"
