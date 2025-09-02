# AMG-QAD Makefile

BINARY_NAME=amg-qad
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt

# Linting parameters
GOLANGCI_LINT=$(shell which golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint")

# Build flags
LDFLAGS=-ldflags "-X 'github.com/weka/amg-utils/tools/amg-qad/cmd.version=$(VERSION)' -X 'main.buildTime=$(BUILD_TIME)' -X 'main.commit=$(COMMIT)'"

# Default target
.PHONY: all
all: clean deps fmt build

# Build the binary
.PHONY: build
build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) .

# Build for Linux AMD64
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .

# Build all architectures (currently only AMD64)
.PHONY: build-all
build-all: build-linux-amd64

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Format code (modifies files)
.PHONY: fmt
fmt:
	$(GOFMT) ./...

# Check code formatting (read-only)
.PHONY: fmt-check
fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		echo "Run 'make fmt' to fix formatting issues."; \
		exit 1; \
	fi

# Run go vet for common errors
.PHONY: vet
vet:
	$(GOCMD) vet ./...

# Run golangci-lint
.PHONY: lint
lint:
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found. Run 'make lint-install' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run --no-config

# Check if golangci-lint is installed
.PHONY: lint-install
lint-install:
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	else \
		echo "golangci-lint is already installed at $(GOLANGCI_LINT)"; \
	fi

# Run all linting and formatting checks
.PHONY: lint-all
lint-all: fmt-check vet lint
	@echo "All linting checks completed successfully!"

# Fix all formatting and linting issues
.PHONY: fix
fix: fmt
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found. Run 'make lint-install' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run --fix --no-config

# Install Git hooks
.PHONY: install-hooks
install-hooks:
	@echo "Installing Git hooks..."
	@./scripts/install-hooks.sh

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64
	rm -f $(BINARY_NAME)-linux-arm64
	rm -rf bin/
	rm -f coverage.out

# Install the binary
.PHONY: install
install: build
	cp bin/$(BINARY_NAME) /usr/local/bin/

# Create sample configuration
.PHONY: config
config:
	@echo "Creating sample configuration file..."
	@mkdir -p $(HOME)/.config
	@echo "# AMG-QAD Configuration" > $(HOME)/.config/amg-qad.yaml
	@echo "test_time: \"02:00\"      # Time to run tests (HH:MM format)" >> $(HOME)/.config/amg-qad.yaml
	@echo "web_port: 8080          # Port for web dashboard" >> $(HOME)/.config/amg-qad.yaml
	@echo "results_path: \"./results\"  # Path to store test results" >> $(HOME)/.config/amg-qad.yaml
	@echo "Configuration created at $(HOME)/.config/amg-qad.yaml"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all              - Clean, install deps, format, and build"
	@echo "  build            - Build the binary"
	@echo "  build-all        - Build for all supported architectures"
	@echo "  build-linux-amd64 - Build for Linux AMD64"
	@echo "  deps             - Download and verify dependencies"
	@echo "  fmt              - Format Go code (modifies files)"
	@echo "  fmt-check        - Check Go code formatting (read-only)"
	@echo "  vet              - Run go vet for common errors"
	@echo "  lint             - Run golangci-lint"
	@echo "  lint-install     - Install golangci-lint if not present"
	@echo "  lint-all         - Run all linting and formatting checks"
	@echo "  fix              - Fix all formatting and auto-fixable linting issues"
	@echo "  install-hooks    - Install Git hooks for automatic linting"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  clean            - Clean build artifacts"
	@echo "  install          - Install binary to /usr/local/bin"
	@echo "  config           - Create sample configuration file"
	@echo "  help             - Show this help message"
