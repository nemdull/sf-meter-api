.PHONY: build test test-verbose lint run run-local docker-build clean help

# Go version
GO := go
GO_VERSION := 1.24

# Binary name
BINARY := sf-meter-api

# Build variables
BUILD_DIR := bin
MAIN_FILE := main.go

help:
	@echo "sf-meter-api Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Compile the binary"
	@echo "  make test          - Run all tests"
	@echo "  make test-verbose  - Run tests with coverage report"
	@echo "  make lint          - Run go vet for linting"
	@echo "  make run           - Run with Docker Compose (PostgreSQL + app)"
	@echo "  make run-local     - Run locally with Docker database only"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make clean         - Remove binary artifacts"
	@echo "  make help          - Show this help message"

build: clean
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/$(BINARY) -v $(MAIN_FILE)
	@echo "✓ Binary created at $(BUILD_DIR)/$(BINARY)"

test:
	@echo "Running tests..."
	@$(GO) test -v ./...
	@echo "✓ Tests passed"

test-verbose:
	@echo "Running tests with coverage..."
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

lint:
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "✓ Lint check passed"

run:
	@echo "Starting sf-meter-api with Docker Compose..."
	@docker compose up --build
	@echo "✓ API running at http://localhost:8080"

run-local:
	@echo "Starting database with Docker Compose..."
	@docker compose up -d db
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "Starting sf-meter-api locally..."
	@DATABASE_URL="postgres://sfmeter:sfmeter@localhost:5432/sfmeter_db?sslmode=disable" $(GO) run $(MAIN_FILE)

docker-build:
	@echo "Building Docker image..."
	@docker build -t sf-meter:latest .
	@echo "✓ Image built: sf-meter:latest"
	@docker images | grep sf-meter

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

# Development helpers
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@echo "✓ Dependencies downloaded"

mod-tidy:
	@echo "Tidying go.mod..."
	@$(GO) mod tidy
	@echo "✓ Module tidied"

# Docker Compose shortcuts
db-up:
	@docker compose up -d db

db-down:
	@docker compose down

db-logs:
	@docker compose logs -f db

db-shell:
	@docker compose exec db psql -U sfmeter -d sfmeter_db

# Shortcuts for common workflows
dev-setup: deps fmt mod-tidy
	@echo "✓ Development environment ready"

ci: lint test build
	@echo "✓ CI pipeline passed"
