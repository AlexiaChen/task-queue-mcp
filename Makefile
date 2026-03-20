.PHONY: build test run clean dev fmt lint help e2e e2e-quick

# Binary name
BINARY_NAME=task-queue-mcp
BINARY_PATH=./bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Main package
MAIN_PACKAGE=./cmd/server

# Default port and database path
PORT?=9292
DB_PATH?=./data/tasks.db
MCP_MODE?=http

## build: Build the binary
build:
	@echo "Building..."
	@mkdir -p bin
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-s -w" -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "Binary built at $(BINARY_PATH)"

## build-static: Build fully static binary (no dynamic dependencies)
build-static:
	@echo "Building static binary..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -ldflags="-s -w -extldflags '-static'" -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "Static binary built at $(BINARY_PATH)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

## run: Run the server (HTTP mode)
run: build
	@echo "Starting server on port $(PORT)..."
	./bin/$(BINARY_NAME) -port=$(PORT) -db=$(DB_PATH) -mcp=$(MCP_MODE)

## run-stdio: Run the server in STDIO mode (for MCP clients)
run-stdio: build
	@echo "Starting server in STDIO mode..."
	./bin/$(BINARY_NAME) -mcp=stdio -db=$(DB_PATH)

## run-both: Run the server with both STDIO and HTTP
run-both: build
	@echo "Starting server with both STDIO and HTTP..."
	./bin/$(BINARY_NAME) -port=$(PORT) -db=$(DB_PATH) -mcp=both

## dev: Run in development mode with auto-reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf data/

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w -s .

## lint: Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## db-init: Initialize the database directory
db-init:
	@echo "Initializing database directory..."
	@mkdir -p data
	@echo "Database will be created at $(DB_PATH) on first run"

## install: Install the binary to $GOPATH/bin
install: build
	@echo "Installing to $(GOPATH)/bin..."
	cp $(BINARY_PATH) $(GOPATH)/bin/

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p $(PORT):$(PORT) -v $(PWD)/data:/app/data $(BINARY_NAME):latest

## api-test: Test the API endpoints (requires server running)
api-test:
	@echo "Testing API endpoints..."
	@echo "Creating a queue..."
	curl -X POST http://localhost:$(PORT)/api/queues -H "Content-Type: application/json" -d '{"name":"Test Queue","description":"A test queue"}'
	@echo ""
	@echo "Listing queues..."
	curl http://localhost:$(PORT)/api/queues
	@echo ""

## e2e: Run end-to-end tests (starts server automatically)
e2e: build
	@echo "Running e2e tests..."
	@rm -f ./data/tasks.db
	@./bin/$(BINARY_NAME) -port=$(PORT) -db=./data/tasks.db -mcp=http &
	@sleep 2
	@E2E_SERVER_URL=http://localhost:$(PORT) go test -v ./test/e2e/; \
	EXIT_CODE=$$?; \
	pkill -f "$(BINARY_NAME)" 2>/dev/null; \
	rm -f ./data/tasks.db; \
	exit $$EXIT_CODE

## e2e-quick: Run e2e tests against already running server
e2e-quick:
	@echo "Running e2e tests against http://localhost:$(PORT)..."
	@E2E_SERVER_URL=http://localhost:$(PORT) go test -v ./test/e2e/

## example-client: Run the MCP client example (requires server running)
example-client:
	@echo "Running MCP client example..."
	@echo "Make sure the server is running: make run"
	go run ./examples/mcp-client/main.go

## example-stdio: Run the STDIO MCP client example
example-stdio:
	@echo "Running STDIO client example..."
	go run ./examples/stdio-client/main.go

## help: Show this help message
help:
	@echo "Task Queue MCP Server - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Variables:"
	@echo "  PORT     - Server port (default: 9292)"
	@echo "  DB_PATH  - Database path (default: ./data/tasks.db)"
	@echo "  MCP_MODE - MCP mode: stdio, http, or both (default: http)"
	@echo ""
	@echo "Examples:"
	@echo "  make run                    # Run server on port 9292"
	@echo "  make run PORT=3000          # Run server on port 3000"
	@echo "  make run-stdio              # Run in STDIO mode for MCP clients"
	@echo "  make test                   # Run tests"
	@echo "  make e2e                    # Run e2e tests"
	@echo "  make build                  # Build binary"

# Default target
.DEFAULT_GOAL := help
