# RangeDB Makefile

# Project information
PROJECT_NAME := rangedb
MODULE_NAME := github.com/samintheshell/rangekey
BINARY_NAME := rangedb
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Build parameters
BUILD_FLAGS := -v
LDFLAGS := -X '$(MODULE_NAME)/internal/version.Version=$(VERSION)' \
           -X '$(MODULE_NAME)/internal/version.BuildTime=$(BUILD_TIME)' \
           -X '$(MODULE_NAME)/internal/version.Commit=$(COMMIT)'

# Directories
BUILD_DIR := build
DIST_DIR := dist
DATA_DIR := data

# Proto directories
PROTO_DIR := api
PROTO_GEN_DIR := $(PROTO_DIR)

# Default target
.PHONY: all
all: clean deps build test

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -rf $(DATA_DIR)
	@$(GOCLEAN)

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

# Build the project
.PHONY: build
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Build for Linux
.PHONY: build-linux
build-linux: deps
	@echo "Building for Linux..."
	@mkdir -p $(DIST_DIR)/linux
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/linux/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Build for macOS
.PHONY: build-darwin
build-darwin: deps
	@echo "Building for macOS..."
	@mkdir -p $(DIST_DIR)/darwin
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/darwin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Build for Windows
.PHONY: build-windows
build-windows: deps
	@echo "Building for Windows..."
	@mkdir -p $(DIST_DIR)/windows
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/windows/$(BINARY_NAME).exe ./cmd/$(BINARY_NAME)

# Run tests
.PHONY: test
test: deps
	@echo "Running tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: test
	@echo "Generating coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run benchmarks
.PHONY: bench
bench: deps
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@$(GOFMT) ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	@$(GOVET) ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run

# Generate code
.PHONY: generate
generate:
	@echo "Generating code..."
	@$(GOCMD) generate ./...

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Run the server
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) server..."
	@./$(BUILD_DIR)/$(BINARY_NAME) server --cluster-init

# Run multiple nodes for testing
.PHONY: run-cluster
run-cluster: build
	@echo "Starting 3-node cluster..."
	@./$(SCRIPTS_DIR)/start-cluster.sh

# Stop cluster
.PHONY: stop-cluster
stop-cluster:
	@echo "Stopping cluster..."
	@./$(SCRIPTS_DIR)/stop-cluster.sh

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/air-verse/air@latest

# Run with hot reload
.PHONY: dev
dev: dev-setup
	@echo "Starting development server with hot reload..."
	@air

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(PROJECT_NAME):$(VERSION) .
	@docker tag $(PROJECT_NAME):$(VERSION) $(PROJECT_NAME):latest

# Docker run
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -p 8081:8081 $(PROJECT_NAME):latest

# Create release
.PHONY: release
release: build-all
	@echo "Creating release..."
	@mkdir -p $(DIST_DIR)/release
	@tar -czf $(DIST_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR)/linux $(BINARY_NAME)
	@tar -czf $(DIST_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR)/darwin $(BINARY_NAME)
	@zip -j $(DIST_DIR)/release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(DIST_DIR)/windows/$(BINARY_NAME).exe

# Generate protobuf files
.PHONY: proto
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/rangedb/v1/rangedb.proto

# Clean generated proto files
.PHONY: clean-proto
clean-proto:
	@echo "Cleaning generated protobuf files..."
	find $(PROTO_GEN_DIR) -name "*.pb.go" -type f -delete

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Clean, deps, build, and test"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  build         - Build the project"
	@echo "  build-all     - Build for all platforms"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-darwin  - Build for macOS"
	@echo "  build-windows - Build for Windows"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  bench         - Run benchmarks"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Lint code"
	@echo "  generate      - Generate code"
	@echo "  install       - Install the binary"
	@echo "  run           - Run the server"
	@echo "  run-cluster   - Run 3-node cluster"
	@echo "  stop-cluster  - Stop cluster"
	@echo "  dev-setup     - Setup development environment"
	@echo "  dev           - Run with hot reload"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  release       - Create release packages"
	@echo "  help          - Show this help"
