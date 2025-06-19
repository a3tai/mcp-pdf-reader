# PDF MCP Server Makefile

# Variables
BINARY_NAME=mcp-pdf-reader
MAIN_FILE=cmd/mcp-pdf-reader/main.go
BUILD_DIR=build
INSTALL_DIR=$(shell go env GOPATH)/bin
DEFAULT_PDF_DIR=$(HOME)/Documents

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo 'dev')
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
VERSION_FLAGS = -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)
LDFLAGS = -ldflags "$(VERSION_FLAGS)"

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BINARY_NAME)"

# Build for production (optimized)
.PHONY: build-prod
build-prod:
	@echo "Building $(BINARY_NAME) for production..."
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "Production build complete: $(BINARY_NAME)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies installed"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Install binary using Go's standard install method
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME) using go install..."
	@go install -ldflags "$(VERSION_FLAGS)" ./cmd/mcp-pdf-reader
	@echo "Installation complete!"
	@echo ""
	@echo "$(BINARY_NAME) has been installed to $(INSTALL_DIR)"
	@echo "Make sure $(INSTALL_DIR) is in your PATH (usually it is by default)"
	@echo ""
	@echo "You can now use: $(BINARY_NAME) -pdfdir=/path/to/pdfs"

# Uninstall binary from Go bin directory
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstall complete"

# Run the server for development (stdio mode - default)
.PHONY: run
run: build
	@echo "Starting $(BINARY_NAME) in stdio mode with PDF directory: $(DEFAULT_PDF_DIR)"
	./$(BINARY_NAME) -pdfdir=$(DEFAULT_PDF_DIR)

# Run the server in HTTP server mode
.PHONY: run-server
run-server: build
	@echo "Starting $(BINARY_NAME) in server mode with PDF directory: $(DEFAULT_PDF_DIR)"
	./$(BINARY_NAME) -mode=server -pdfdir=$(DEFAULT_PDF_DIR)

# Run with custom PDF directory (stdio mode)
.PHONY: run-custom
run-custom: build
	@if [ -z "$(DIR)" ]; then \
		echo "Usage: make run-custom DIR=/path/to/pdf/directory"; \
		exit 1; \
	fi
	@echo "Starting $(BINARY_NAME) in stdio mode with PDF directory: $(DIR)"
	./$(BINARY_NAME) -pdfdir=$(DIR)

# Run with custom PDF directory (server mode)
.PHONY: run-server-custom
run-server-custom: build
	@if [ -z "$(DIR)" ]; then \
		echo "Usage: make run-server-custom DIR=/path/to/pdf/directory"; \
		exit 1; \
	fi
	@echo "Starting $(BINARY_NAME) in server mode with PDF directory: $(DIR)"
	./$(BINARY_NAME) -mode=server -pdfdir=$(DIR)

# Cross-compile for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILE)

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)

	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)

	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

	@echo "Cross-compilation complete. Binaries in $(BUILD_DIR)/"

# Create release package
.PHONY: package
package: build-all
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/releases

	# Create tar.gz for Unix systems
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64 README.md
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64 README.md
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64 README.md
	tar -czf $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64 README.md

	# Create zip for Windows
	@cd $(BUILD_DIR) && zip releases/$(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe ../README.md

	@echo "Release packages created in $(BUILD_DIR)/releases/"

# Development setup without scripts
.PHONY: setup-dev
setup-dev: deps build
	@echo "Development environment setup complete"
	@echo "Run 'make run' to start the server in stdio mode"
	@echo "Run 'make run-server' to start the server in HTTP mode"

# Development setup
.PHONY: dev-setup
dev-setup: setup-dev

# Check if Go is installed
.PHONY: check-go
check-go:
	@which go > /dev/null || (echo "Go is not installed. Please install Go first." && exit 1)
	@echo "Go version: $$(go version)"

# Check if binary is installed and in PATH
.PHONY: check-install
check-install:
	@if command -v $(BINARY_NAME) >/dev/null 2>&1; then \
		echo "$(BINARY_NAME) is installed and available in PATH"; \
		echo "Location: $$(which $(BINARY_NAME))"; \
		echo "Go binary directory: $(INSTALL_DIR)"; \
		echo "Version: $$( $(BINARY_NAME) --help | head -1 || echo 'Unable to get version')"; \
	else \
		echo "$(BINARY_NAME) is not installed or not in PATH"; \
		echo "Expected location: $(INSTALL_DIR)/$(BINARY_NAME)"; \
		echo "Run 'make install' to install it using go install"; \
		echo "Make sure $(INSTALL_DIR) is in your PATH"; \
		exit 1; \
	fi

# Show help
.PHONY: help
help:
	@echo "PDF MCP Server - Available Make targets:"
	@echo ""
	@echo "Building:"
	@echo "  build         Build the binary"
	@echo "  build-prod    Build optimized binary for production"
	@echo "  build-all     Cross-compile for multiple platforms"
	@echo ""
	@echo "Development:"
	@echo "  run           Run server with default PDF directory"
	@echo "  run-custom    Run server with custom directory (make run-custom DIR=/path)"
	@echo "  dev-setup     Set up development environment"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           Format code"
	@echo "  lint          Run linter (requires golangci-lint)"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps          Install/update dependencies"
	@echo ""
	@echo "Installation:"
	@echo "  install       Install binary to system (/usr/local/bin)"
	@echo "  uninstall     Remove binary from system"
	@echo ""
	@echo "Packaging:"
	@echo "  package       Create release packages for all platforms"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean         Clean build artifacts"
	@echo "  setup-launch  Make launch script executable"
	@echo "  check-go      Verify Go installation"
	@echo ""
	@echo "Examples:"
	@echo "  make install                          # Install binary using go install"
	@echo "  make run                              # Run in stdio mode with ~/Documents"
	@echo "  make run-server                       # Run in HTTP server mode"
	@echo "  make run-custom DIR=/path/to/pdfs     # Run stdio mode with custom dir"
	@echo "  make run-server-custom DIR=/path      # Run server mode with custom dir"
	@echo "  make test                             # Run all tests"
	@echo "  make test-coverage                    # Run tests with coverage report"
	@echo "  make check-install                    # Check if binary is installed"
