# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build parameters
BINARY_NAME=loki
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/loki

# Version information
VERSION ?= v1.0.0-dev
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_BY := $(shell whoami)

# Build flags
LDFLAGS=-ldflags "-X github.com/Blazingkevin/loki/internal/cli.buildTime=$(BUILD_TIME) \
                  -X github.com/Blazingkevin/loki/internal/cli.gitCommit=$(GIT_COMMIT) \
                  -X github.com/Blazingkevin/loki/internal/cli.buildBy=$(BUILD_BY)"

.PHONY: all build clean test test-verbose test-cover help run install deps

all: test build

# Build the binary
build:
	@echo "🔨 Building Loki..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "✅ Build complete: $(BINARY_PATH)"

# Run the application
run: build
	$(BINARY_PATH)

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf bin/

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running tests (verbose)..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-cover:
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# Install the binary
install:
	@echo "📦 Installing Loki..."
	$(GOCMD) install $(LDFLAGS) $(MAIN_PATH)

# Build for multiple platforms
build-all:
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✅ Cross-compilation complete"

# Format code
fmt:
	@echo "💅 Formatting code..."
	$(GOCMD) fmt ./...

# Run linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo "  build        Build the binary"
	@echo "  run          Build and run the application"
	@echo "  test         Run tests"
	@echo "  test-verbose Run tests with verbose output"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Install dependencies"
	@echo "  install      Install the binary"
	@echo "  build-all    Build for multiple platforms"
	@echo "  fmt          Format code"
	@echo "  lint         Run linter"
	@echo "  help         Show this help message"

# Example usage commands
examples:
	@echo "🚀 Example usage:"
	@echo "  make build && ./bin/loki version"
	@echo "  make build && ./bin/loki serve test/petstore.yaml"
	@echo "  make build && ./bin/loki validate test/petstore.yaml"
