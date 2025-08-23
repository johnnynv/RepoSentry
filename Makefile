# Makefile for RepoSentry

# Variables
APP_NAME := reposentry
BINARY_NAME := reposentry
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
.PHONY: help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "📖 Swagger UI (without full app):"
	@echo "  make swagger-ui     - Start static server at http://localhost:8081/swagger"

## build: Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(GOBIN)
	@go build -ldflags "$(LDFLAGS)" -o $(GOBIN)/$(BINARY_NAME) ./cmd/reposentry

## build-linux: Build Linux binary (for deployment)
.PHONY: build-linux
build-linux:
	@echo "Building Linux binary..."
	@mkdir -p $(GOBIN)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(GOBIN)/$(BINARY_NAME)-linux ./cmd/reposentry

## install: Install the binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/reposentry

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(GOBIN)
	@rm -rf dist/
	@rm -rf data/
	@rm -rf logs/
	@rm -f ./reposentry
	@rm -f coverage.txt *.coverprofile *.test *.prof *.out 2>/dev/null || true
	@find . -name "*.tmp" -o -name "*.temp" -o -name "*.log" -o -name "*.cache" | xargs rm -f 2>/dev/null || true
	@find . -name ".DS_Store" -o -name "Thumbs.db" -o -name "*.swp" -o -name "*.swo" | xargs rm -f 2>/dev/null || true
	@go clean

## test: Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

## test-all: Run all tests
.PHONY: test-all
test-all: test

## lint: Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	@golangci-lint run

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## mod-tidy: Tidy up module dependencies
.PHONY: mod-tidy
mod-tidy:
	@echo "Tidying module dependencies..."
	@go mod tidy

## mod-download: Download module dependencies
.PHONY: mod-download
mod-download:
	@echo "Downloading module dependencies..."
	@go mod download

## swagger: Generate Swagger documentation
.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	@go run github.com/swaggo/swag/cmd/swag init -g internal/api/docs.go -o docs --parseDependency

## swagger-ui: Start Swagger UI static server (without full app)
.PHONY: swagger-ui
swagger-ui:
	@echo "Starting Swagger UI static server..."
	@go run tools/swagger-static-server.go

## deps: Install development dependencies
.PHONY: deps
deps:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -f deployments/docker/Dockerfile -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

## docker-run: Run Docker container
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run --rm -v $(PWD)/configs:/app/configs $(APP_NAME):latest

## systemd-install: Install systemd service (requires sudo)
.PHONY: systemd-install
systemd-install: build-linux
	@echo "Installing systemd service..."
	@sudo cp deployments/systemd/reposentry.service /etc/systemd/system/
	@sudo cp $(GOBIN)/$(BINARY_NAME)-linux /usr/local/bin/$(BINARY_NAME)
	@sudo systemctl daemon-reload
	@sudo systemctl enable $(APP_NAME)

## systemd-uninstall: Uninstall systemd service (requires sudo)
.PHONY: systemd-uninstall
systemd-uninstall:
	@echo "Uninstalling systemd service..."
	@sudo systemctl stop $(APP_NAME) || true
	@sudo systemctl disable $(APP_NAME) || true
	@sudo rm -f /etc/systemd/system/$(APP_NAME).service
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo systemctl daemon-reload

## dev-setup: Set up development environment
.PHONY: dev-setup
dev-setup: deps mod-download
	@echo "Development environment setup complete!"

## release: Create a release build
.PHONY: release
release: clean fmt vet test swagger build-linux
	@echo "Release build complete!"

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

.PHONY: all
all: check build
