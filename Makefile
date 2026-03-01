.PHONY: help build test lint docker tidy run clean

help:
	@echo "oci-prometheus-sd-proxy - Available targets:"
	@echo ""
	@echo "  make build           Build binary (outputs to ./bin/oci-sd-proxy)"
	@echo "  make test            Run tests with race detector"
	@echo "  make lint            Lint code with golangci-lint"
	@echo "  make docker          Build Docker image"
	@echo "  make tidy            Download and tidy dependencies"
	@echo "  make run             Run locally (requires SERVER_TOKEN env var)"
	@echo "  make clean           Remove build artifacts"
	@echo ""

# Build binary
build:
	@mkdir -p bin
	@echo "Building binary..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags "-s -w -X main.version=$$(git describe --tags --abbrev=0 2>/dev/null || echo 'dev')" \
		-o bin/oci-sd-proxy \
		./cmd/server
	@echo "Binary ready: bin/oci-sd-proxy"

# Run tests
test:
	@echo "Running tests..."
	@go test -race -cover ./...

# Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t oci-prometheus-sd-proxy:latest .
	@echo "Image built: oci-prometheus-sd-proxy:latest"

# Download and tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod download
	@go mod tidy

# Run locally
run:
	@if [ -z "$$SERVER_TOKEN" ]; then \
		echo "Error: SERVER_TOKEN environment variable not set"; \
		echo "Usage: SERVER_TOKEN=your-token make run"; \
		exit 1; \
	fi
	@echo "Running locally (token: $$SERVER_TOKEN)..."
	@go run ./cmd/server

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Default target
.DEFAULT_GOAL := help
