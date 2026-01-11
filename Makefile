# Makefile for astroladb
# Usage: make [target]

# Variables
BINARY_NAME := alab
BUILD_DIR := bin
CMD_DIR := ./cmd/alab
MODULE := github.com/hlop3z/astroladb

# Version - can be overridden: make build VERSION=v1.0.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Detect OS for binary extension
ifeq ($(OS),Windows_NT)
    BINARY_EXT := .exe
    USER_BIN := $(USERPROFILE)/bin
else
    BINARY_EXT :=
    USER_BIN := $(HOME)/bin
endif

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod
GOFMT := gofmt
GOVET := $(GO) vet

# Build flags
LDFLAGS := -s -w -X main.version=$(VERSION)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Dev build flags (always use "dev" version)
DEV_LDFLAGS := -s -w -X main.version=dev
DEV_BUILD_FLAGS := -ldflags "$(DEV_LDFLAGS)"

# Test flags
TEST_FLAGS := -v
INTEGRATION_FLAGS := -tags=integration

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all build build-dev release clean test test-unit test-integration test-sqlite test-postgres test-e2e test-all fmt vet lint deps tidy help install docs-format docs-format-check

# Default target
all: build

# Build the binary (uses git tag for version)
build:
	@echo "$(GREEN)Building $(BINARY_NAME) $(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) $(CMD_DIR)
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) ($(VERSION))$(NC)"

# Build release binaries for all platforms
release: clean
	@echo "$(GREEN)Building release $(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "$(GREEN)Release $(VERSION) built:$(NC)"
	@ls -la $(BUILD_DIR)/

# Build for all platforms (alias for release)
build-all: release

build-linux:
	@echo "$(GREEN)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

build-darwin:
	@echo "$(GREEN)Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows:
	@echo "$(GREEN)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Install the binary to GOPATH/bin
install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	$(GO) install $(CMD_DIR)
	@echo "$(GREEN)Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(NC)"

# Build and install to ~/bin for local development testing (version=dev)
build-dev:
	@echo "$(GREEN)Building $(BINARY_NAME) for local development...$(NC)"
	@mkdir -p "$(USER_BIN)"
	$(GOBUILD) $(DEV_BUILD_FLAGS) -o "$(USER_BIN)/$(BINARY_NAME)$(BINARY_EXT)" $(CMD_DIR)
	@echo "$(GREEN)Installed to $(USER_BIN)/$(BINARY_NAME)$(BINARY_EXT)$(NC)"
	@echo "$(YELLOW)Make sure $(USER_BIN) is in your PATH$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@$(GO) clean
	@echo "$(GREEN)Clean complete$(NC)"

# Run unit tests (no database required)
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) ./...

# Run unit tests with short flag
test-short:
	@echo "$(GREEN)Running short tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) -short ./...

# Run integration tests (requires database)
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) $(INTEGRATION_FLAGS) ./...

# Run SQLite-specific E2E tests
test-sqlite:
	@echo "$(GREEN)Running SQLite E2E tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) $(INTEGRATION_FLAGS) -run "TestE2E_SQLite" ./cmd/alab/...

# Run PostgreSQL/CockroachDB E2E tests (both databases)
test-postgres:
	@echo "$(GREEN)Running PostgreSQL/CockroachDB E2E tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) $(INTEGRATION_FLAGS) -run "TestE2E_Postgres" ./cmd/alab/...

# Run all E2E tests
test-e2e:
	@echo "$(GREEN)Running all E2E tests...$(NC)"
	$(GOTEST) $(TEST_FLAGS) $(INTEGRATION_FLAGS) -run "TestE2E" ./cmd/alab/...

# Run all tests (unit + integration)
test-all: test-unit test-integration

# Default test target (unit tests only)
test: test-unit

# Run tests with coverage
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) $(TEST_FLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

# Run tests with race detector
test-race:
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	$(GOTEST) $(TEST_FLAGS) -race ./...

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .

# Check formatting (returns error if not formatted)
fmt-check:
	@echo "$(GREEN)Checking formatting...$(NC)"
	@test -z "$$($(GOFMT) -l .)" || (echo "$(RED)Code is not formatted. Run 'make fmt'$(NC)" && exit 1)

# Format documentation files
docs-format:
	@echo "$(GREEN)Formatting documentation...$(NC)"
	@cd docs && npm run format

# Check documentation formatting
docs-format-check:
	@echo "$(GREEN)Checking documentation formatting...$(NC)"
	@cd docs && npm run format:check

# Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)" && exit 1)
	golangci-lint run ./...

# Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download

# Tidy dependencies
tidy:
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	$(GOMOD) tidy

# Verify dependencies
verify:
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	$(GOMOD) verify

# Run all checks (fmt, vet, test)
check: fmt-check vet test-unit

# Run CI pipeline (all checks + integration tests)
ci: deps fmt-check vet test-all

# Development workflow: format, build, test
dev: fmt build test-unit

# Quick rebuild and test
quick: build test-short

# Generate (placeholder for code generation)
generate:
	@echo "$(GREEN)Running go generate...$(NC)"
	$(GO) generate ./...

# Show help
help:
	@echo "$(GREEN)astroladb Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Build targets:$(NC)"
	@echo "  make build         - Build the binary to ./bin/ (uses git tag version)"
	@echo "  make build-dev     - Build and install to ~/bin/ (version=dev)"
	@echo "  make release       - Build release binaries for all platforms"
	@echo "  make build-all     - Alias for release"
	@echo "  make install       - Install binary to GOPATH/bin"
	@echo "  make clean         - Remove build artifacts"
	@echo ""
	@echo "  VERSION can be overridden: make build VERSION=v1.0.0"
	@echo ""
	@echo "$(YELLOW)Test targets:$(NC)"
	@echo "  make test          - Run unit tests (default)"
	@echo "  make test-unit     - Run unit tests"
	@echo "  make test-short    - Run short tests only"
	@echo "  make test-integration - Run integration tests (requires DB)"
	@echo "  make test-sqlite   - Run SQLite E2E tests"
	@echo "  make test-postgres - Run PostgreSQL/CockroachDB E2E tests"
	@echo "  make test-e2e      - Run all E2E tests"
	@echo "  make test-all      - Run all tests (unit + integration)"
	@echo "  make test-cover    - Run tests with coverage report"
	@echo "  make test-race     - Run tests with race detector"
	@echo ""
	@echo "$(YELLOW)Code quality:$(NC)"
	@echo "  make fmt           - Format code"
	@echo "  make fmt-check     - Check code formatting"
	@echo "  make docs-format   - Format documentation files"
	@echo "  make docs-format-check - Check documentation formatting"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make check         - Run fmt-check, vet, test-unit"
	@echo ""
	@echo "$(YELLOW)Dependencies:$(NC)"
	@echo "  make deps          - Download dependencies"
	@echo "  make tidy          - Tidy go.mod"
	@echo "  make verify        - Verify dependencies"
	@echo ""
	@echo "$(YELLOW)Workflows:$(NC)"
	@echo "  make dev           - Format, build, and test"
	@echo "  make quick         - Quick build and short tests"
	@echo "  make ci            - Full CI pipeline"
	@echo ""
