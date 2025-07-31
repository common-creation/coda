# Makefile for CODA

# Variables
BINARY_NAME := coda
BUILD_DIR := ./bin
DIST_DIR := ./dist
GO_FILES := $(shell find . -name '*.go' -type f -not -path "./vendor/*")
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

.PHONY: all build test test-coverage lint fmt clean install run help

# Default target
all: lint test build

# Build the binary
build:
	@echo "$(COLOR_BLUE)Building $(BINARY_NAME)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/coda

# Run tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage.html$(COLOR_RESET)"

# Run linter
lint:
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "$(COLOR_YELLOW)golangci-lint not found. Installing...$(COLOR_RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

# Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	$(GOFMT) -w $(GO_FILES)

# Clean build artifacts
clean:
	@echo "$(COLOR_BLUE)Cleaning...$(COLOR_RESET)"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html

# Install binary
install: build
	@echo "$(COLOR_BLUE)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	$(GOCMD) install ./cmd/coda

# Run the application
run:
	@echo "$(COLOR_BLUE)Running $(BINARY_NAME)...$(COLOR_RESET)"
	$(GOCMD) run ./cmd/coda

# Download dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	$(GOMOD) download
	$(GOMOD) tidy

# Verify dependencies
verify:
	@echo "$(COLOR_BLUE)Verifying dependencies...$(COLOR_RESET)"
	$(GOMOD) verify

# Build for multiple platforms
build-all:
	@echo "$(COLOR_BLUE)Building for multiple platforms...$(COLOR_RESET)"
	@mkdir -p $(DIST_DIR)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/coda
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/coda
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/coda
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/coda
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/coda
	@echo "$(COLOR_GREEN)Build complete for all platforms$(COLOR_RESET)"

# Show help
help:
	@echo "$(COLOR_BOLD)CODA Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET)"
	@echo "  make $(COLOR_GREEN)[target]$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Targets:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)all$(COLOR_RESET)            - Run lint, test, and build"
	@echo "  $(COLOR_GREEN)build$(COLOR_RESET)          - Build the binary"
	@echo "  $(COLOR_GREEN)test$(COLOR_RESET)           - Run tests"
	@echo "  $(COLOR_GREEN)test-coverage$(COLOR_RESET)  - Run tests with coverage report"
	@echo "  $(COLOR_GREEN)lint$(COLOR_RESET)           - Run golangci-lint"
	@echo "  $(COLOR_GREEN)fmt$(COLOR_RESET)            - Format code with gofmt"
	@echo "  $(COLOR_GREEN)clean$(COLOR_RESET)          - Remove build artifacts"
	@echo "  $(COLOR_GREEN)install$(COLOR_RESET)        - Install the binary"
	@echo "  $(COLOR_GREEN)run$(COLOR_RESET)            - Run the application"
	@echo "  $(COLOR_GREEN)deps$(COLOR_RESET)           - Download dependencies"
	@echo "  $(COLOR_GREEN)verify$(COLOR_RESET)         - Verify dependencies"
	@echo "  $(COLOR_GREEN)build-all$(COLOR_RESET)      - Build for multiple platforms"
	@echo "  $(COLOR_GREEN)help$(COLOR_RESET)           - Show this help message"