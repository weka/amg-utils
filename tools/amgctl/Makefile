# Makefile for amgctl
BINARY_NAME=amgctl
MAIN_PACKAGE=.

# Default build
.PHONY: build
build:
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Install dependencies
.PHONY: deps
deps:
	go mod tidy
	go mod download

# Build for all platforms
.PHONY: build-all
build-all: clean
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

# Build release binaries with optimizations (for GitHub releases)
.PHONY: build-release
build-release: clean
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Note: darwin/arm64 build skipped due to Go version compatibility"
	sha256sum $(BINARY_NAME)-* > checksums.txt

# Run tests
.PHONY: test
test:
	go test -v ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Check formatting (without modifying files)
.PHONY: fmt-check
fmt-check:
	@echo "Checking Go formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files are not properly formatted:"; \
		gofmt -l .; \
		echo "Run 'make fmt' to fix formatting issues."; \
		exit 1; \
	fi
	@echo "All files are properly formatted."

# Vet code for common errors
.PHONY: vet
vet:
	go vet ./...

# Get golangci-lint path
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo "$$(go env GOPATH)/bin/golangci-lint")

# Lint code with golangci-lint
.PHONY: lint
lint:
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found. Run 'make lint-install' first."; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run

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
	$(GOLANGCI_LINT) run --fix

# Install Git hooks
.PHONY: install-hooks
install-hooks:
	@echo "Installing Git hooks..."
	@./scripts/install-hooks.sh

# Install locally
.PHONY: install
install: build
	cp $(BINARY_NAME) /usr/local/bin/

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary for current platform"
	@echo "  build-all    - Build binaries for all platforms"
	@echo "  build-release - Build optimized release binaries with checksums"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Install and tidy dependencies"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format Go code (modifies files)"
	@echo "  fmt-check    - Check Go code formatting (read-only)"
	@echo "  vet          - Run go vet for common errors"
	@echo "  lint         - Run golangci-lint"
	@echo "  lint-install - Install golangci-lint if not present"
	@echo "  lint-all     - Run all linting and formatting checks"
	@echo "  fix          - Fix all formatting and auto-fixable linting issues"
	@echo "  install-hooks - Install Git hooks for automatic linting"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  help         - Show this help message"
