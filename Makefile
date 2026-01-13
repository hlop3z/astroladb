# Makefile for astroladb
# Usage: make [target]

# -----------------------------
# Variables
# -----------------------------
BINARY_NAME := alab
BUILD_DIR := bin
CMD_DIR := ./cmd/alab

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# -----------------------------
# OS detection
# -----------------------------
# Detect if running in MINGW/Git Bash on Windows
ifdef MSYSTEM
	# Git Bash / MINGW on Windows - use Unix commands
	BINARY_EXT := .exe
	USER_BIN := $(HOME)/bin
	MKDIR := mkdir -p
	RM := rm -rf
	LS := ls -la
	GREEN := \033[0;32m
	YELLOW := \033[0;33m
	RED := \033[0;31m
	NC := \033[0m
else ifeq ($(OS),Windows_NT)
	# Native Windows CMD
	BINARY_EXT := .exe
	USER_BIN := $(USERPROFILE)\bin
	MKDIR := mkdir
	RM := rmdir /S /Q
	LS :=
	GREEN :=
	YELLOW :=
	RED :=
	NC :=
else
	# Unix/Linux/macOS
	BINARY_EXT :=
	USER_BIN := $(HOME)/bin
	MKDIR := mkdir -p
	RM := rm -rf
	LS := ls -la
	GREEN := \033[0;32m
	YELLOW := \033[0;33m
	RED := \033[0;31m
	NC := \033[0m
endif

# -----------------------------
# Go commands
# -----------------------------
GO := go
GOBUILD := $(GO) build

# -----------------------------
# Build flags
# -----------------------------
LDFLAGS := -s -w -X main.version=$(VERSION)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

DEV_LDFLAGS := -s -w -X main.version=dev
DEV_BUILD_FLAGS := -ldflags "$(DEV_LDFLAGS)"

# -----------------------------
# Targets
# -----------------------------
.PHONY: all build build-dev release clean lint install

all: build

build:
	@echo "$(GREEN)Building $(BINARY_NAME) $(VERSION)...$(NC)"
	@$(MKDIR) $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) $(CMD_DIR)
	@echo "$(GREEN)Build complete$(NC)"

build-dev:
	@echo "$(GREEN)Building $(BINARY_NAME) (dev)...$(NC)"
	@$(MKDIR) "$(USER_BIN)"
	$(GOBUILD) $(DEV_BUILD_FLAGS) -o "$(USER_BIN)/$(BINARY_NAME)$(BINARY_EXT)" $(CMD_DIR)
	@echo "$(GREEN)Installed to $(USER_BIN)$(NC)"

release: clean
	@echo "$(GREEN)Building release $(VERSION)...$(NC)"
	@$(MKDIR) $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=darwin  GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin  GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "$(GREEN)Release built$(NC)"
	@$(LS) $(BUILD_DIR)

install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	$(GO) install -ldflags "$(LDFLAGS)" $(CMD_DIR)
	@echo "$(GREEN)Installed to GOPATH/bin$(NC)"

lint:
	@echo "$(GREEN)Linting...$(NC)"
	golangci-lint run

clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	@$(RM) $(BUILD_DIR)
