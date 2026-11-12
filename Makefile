.PHONY: build test test-db test-verbose lint run run-local docker-build clean help fmt deps mod-tidy db-up db-down db-logs db-shell dev-setup ci

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
	@echo "  make test          - Run all tests (PostgreSQL tests are skipped)"
	@echo "  make test-db       - Run all tests including PostgreSQL (starts the db container)"
	@echo "  make test-verbose  - Run tests with coverage report"
	@echo "  make lint          - Run gofmt check and go vet"
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
	@$(GO) test -race ./...
	@echo "✓ Tests passed"

# PostgreSQL 実装にも同じテストを当てる。テーブルを空にしてから始めるので、開発用の DB にだけ向けること
test-db:
	@echo "Starting database with Docker Compose..."
	@docker compose up -d --wait db
	@echo "Running tests with PostgreSQL..."
	@TEST_DATABASE_URL="postgres://sfmeter:sfmeter@localhost:5432/sfmeter_db?sslmode=disable" $(GO) test -race -count=1 ./...
	@echo "✓ Tests passed (PostgreSQL included)"

test-verbose:
	@echo "Running tests with coverage..."
	@$(GO) test -race -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

lint:
	@echo "Checking gofmt..."
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)
	@echo "Running go vet..."
	@$(GO) vet ./...
	@echo "✓ Lint check passed"

run:
	@echo "Starting sf-meter-api with Docker Compose..."
	@docker compose up --build
	@echo "✓ API running at http://localhost:8080"

run-local:
	@echo "Starting database with Docker Compose..."
	@docker compose up -d --wait db
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
