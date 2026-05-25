BINARY     := oci-sd-proxy
IMAGE      := oci-prometheus-sd-proxy
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help
.PHONY: help build build-linux test vet lint fmt docker tidy run clean \
        docker-up docker-down docker-test docker-logs

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build binary for current OS/arch
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/server

build-linux: ## Cross-compile for linux/amd64
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/server

test: ## Run tests with race detector
	go test -race -cover ./...

vet: ## Run go vet
	go vet ./...

lint: vet ## Lint code with golangci-lint
	@command -v golangci-lint > /dev/null || { \
		echo "golangci-lint not installed"; \
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	}
	golangci-lint run ./...

fmt: ## Format code
	gofmt -w .

docker: ## Build Docker image
	docker build -t $(IMAGE):latest .

tidy: ## Download and tidy dependencies
	go mod download
	go mod tidy

run: ## Run locally (needs SERVER_TOKEN in env or .env)
	@set -a; [ -f .env ] && . .env; set +a; \
	if [ -z "$$SERVER_TOKEN" ]; then \
		echo "SERVER_TOKEN not set. Export it or add to .env"; \
		exit 1; \
	fi; \
	go run ./cmd/server

docker-up: ## Rebuild image (no cache) and start local Docker stack
	docker-compose -f docker-compose-local.yml build --no-cache
	docker-compose -f docker-compose-local.yml up -d

docker-down: ## Stop local Docker stack and remove containers, volumes, and orphans
	docker-compose -f docker-compose-local.yml down -v --remove-orphans

docker-logs: ## Tail logs from local Docker stack
	docker-compose -f docker-compose-local.yml logs -f

docker-test: ## Hit /v1/targets against the local Docker stack (reads SERVER_TOKEN from .env)
	@if [ ! -f .env ]; then echo "No .env file found"; exit 1; fi
	@TOKEN=$$(grep -E '^SERVER_TOKEN=' .env | cut -d= -f2 | tr -d '"'"'"' '); \
	if [ -z "$$TOKEN" ]; then echo "SERVER_TOKEN not found in .env"; exit 1; fi; \
	echo "GET http://localhost:8080/v1/targets"; \
	curl -sf -H "Authorization: Bearer $$TOKEN" http://localhost:8080/v1/targets | jq .

clean: ## Remove build artifacts
	rm -rf bin/
	go clean
