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

# Run tests
.PHONY: test
test:
	go test -v ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Install locally
.PHONY: install
install: build
	cp $(BINARY_NAME) /usr/local/bin/

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary for current platform"
	@echo "  build-all  - Build binaries for all platforms"
	@echo "  clean      - Remove build artifacts"
	@echo "  deps       - Install and tidy dependencies"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Run linter"
	@echo "  install    - Install binary to /usr/local/bin"
	@echo "  help       - Show this help message"
