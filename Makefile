.PHONY: help build test lint fmt clean run docker-build docker-run install-tools \
        version tag tag-delete tags release-check docker-multiplatform docker-push ghcr-login \
        docker-compose-up docker-compose-down docker-compose-logs docker-compose-pull \
        docker-compose-restart docker-compose-ps

# Variables
BINARY_NAME=proxy
DOCKER_IMAGE=proxy:latest
REGISTRY=ghcr.io
REPO_OWNER=$(shell git config --get remote.origin.url | sed -n 's/.*github.com[:/]\([^/]*\).*/\1/p')
REPO_NAME=$(shell basename `git rev-parse --show-toplevel`)
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .
	@echo "✓ Build complete: $(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

coverage: test ## Show test coverage
	@go tool cover -html=coverage.out

lint: ## Run linters (go vet + staticcheck)
	@echo "Running linters..."
	@go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "⚠ staticcheck not installed (run: make install-tools)"
	@echo "✓ Lint complete"

lint-golangci: ## Run golangci-lint (requires installation)
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null && golangci-lint run --timeout 5m || echo "⚠ golangci-lint not installed"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "⚠ goimports not installed (run: make install-tools)"
	@echo "✓ Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

check: fmt vet lint lint-golangci test ## Run all checks (format, vet, lint, test)
	@echo "✓ All checks passed"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out
	@go clean
	@echo "✓ Clean complete"

run: build ## Build and run
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .
	@echo "✓ Docker build complete"

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	@docker run --rm \
		-p 80:80 -p 443:443 -p 53:53/udp \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		-e PROXY_PORTS=80,443,53 \
		-e RATE_LIMIT_ERRORS=10 \
		-e RATE_LIMIT_ERROR_WINDOW=5m \
		-e RATE_LIMIT_BLOCK_DURATION=15m \
		-e UDP_PORTS=53 \
		-e LOG_LEVEL=DEBUG \
		$(DOCKER_IMAGE)

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "✓ Tools installed"
	@echo ""
	@echo "Optional: Install golangci-lint for comprehensive linting:"
	@echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

mod-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated"

# CI/CD targets
ci-lint: ## CI: Run linters (strict)
	@go vet ./...
	@staticcheck ./...

ci-test: ## CI: Run tests with coverage
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

ci-build: ## CI: Build binary
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_NAME) .

ci: ci-lint ci-test ci-build ## CI: Run all CI checks

# Development helpers
dev: ## Run in development mode with auto-reload (requires entr)
	@echo "Starting development mode..."
	@find . -name '*.go' | entr -r make run

quick: fmt build ## Quick build (format + build)
	@echo "✓ Quick build complete"

.DEFAULT_GOAL := help
