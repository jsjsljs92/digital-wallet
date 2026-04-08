.PHONY: help build test run migrate seed docker-up docker-down docker-clean clean fmt lint install-deps wire

# Variables
BINARY_SERVER=bin/server
BINARY_MIGRATE=bin/migrate
GO=go
DOCKER_COMPOSE=docker compose

help:
	@echo "╔════════════════════════════════════════════════════════════╗"
	@echo "║     Digital Wallet Microservice - Make Commands            ║"
	@echo "╚════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Build & Compile:"
	@echo "  make build              Build Go binaries (server & migrate)"
	@echo "  make clean              Remove build artifacts"
	@echo "  make wire               Generate wire dependency injection"
	@echo ""
	@echo "Testing:"
	@echo "  make test               Run all unit tests"
	@echo "  make test-verbose       Run tests with verbose output"
	@echo "  make test-coverage      Run tests with coverage report"
	@echo ""
	@echo "Local Development:"
	@echo "  make deps               Download dependencies"
	@echo "  make fmt                Format code"
	@echo "  make lint               Run linter (go vet)"
	@echo "  make run                Run server (requires MySQL & Redis running)"
	@echo "  make migrate            Run database migrations"
	@echo "  make seed               Seed sample data"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up          Start full stack (MySQL, Redis, App)"
	@echo "  make docker-down        Stop all services"
	@echo "  make docker-build       Build Docker image"
	@echo "  make docker-clean       Remove Docker containers & volumes"
	@echo ""
	@echo "Development:"
	@echo "  make dev-db             Start only MySQL & Redis"
	@echo "  make health             Check service health"
	@echo ""

# ============================================================================
# Build & Compile Targets
# ============================================================================

install-deps:
	@echo "📦 Installing Go dependencies..."
	$(GO) mod download
	$(GO) mod tidy

wire:
	@echo "🔌 Generating wire dependency injection..."
	@command -v wire >/dev/null 2>&1 || (echo "Installing wire..." && go install github.com/google/wire/cmd/wire@latest)
	wire

build: install-deps
	@echo "🔨 Building binaries..."
	@mkdir -p bin
	$(GO) build -o $(BINARY_SERVER) ./cmd/server
	$(GO) build -o $(BINARY_MIGRATE) ./cmd/migrate
	@echo "✅ Build complete: $(BINARY_SERVER), $(BINARY_MIGRATE)"

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	$(GO) clean
	@echo "✅ Clean complete"

# ============================================================================
# Testing Targets
# ============================================================================

test:
	@echo "🧪 Running tests..."
	$(GO) test ./tests/unit/... -v

test-verbose:
	@echo "🧪 Running tests (verbose)..."
	$(GO) test ./... -v -race

test-coverage:
	@echo "📊 Running tests with coverage..."
	$(GO) test ./... -v -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# ============================================================================
# Local Development Targets
# ============================================================================

deps:
	@echo "📥 Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies downloaded"

fmt:
	@echo "💅 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Code formatted"

lint:
	@echo "🔍 Running linter..."
	$(GO) vet ./...
	@echo "✅ Linting complete"

migrate:
	@echo "🗄️  Running database migrations..."
	$(GO) run cmd/migrate/main.go
	@echo "✅ Migrations complete"

seed:
	@echo "🌱 Seeding sample data..."
	$(GO) run scripts/seed.go
	@echo "✅ Sample data seeded"

run: build migrate
	@echo "🚀 Starting server..."
	./$(BINARY_SERVER)

# ============================================================================
# Docker Targets
# ============================================================================

docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t digital-wallet:latest .
	@echo "✅ Docker image built"

docker-up:
	@echo "🐳 Starting Docker Compose stack..."
	$(DOCKER_COMPOSE) up -d
	@sleep 3
	@echo "✅ Services started"
	@echo ""
	@echo "Available services:"
	@echo "  MySQL:   localhost:3306 (wallet_user / wallet_pass)"
	@echo "  Redis:   localhost:6379"
	@echo "  API:     http://localhost:8080"
	@echo ""

docker-down:
	@echo "🛑 Stopping Docker Compose stack..."
	$(DOCKER_COMPOSE) down
	@echo "✅ Services stopped"

docker-clean:
	@echo "🗑️  Cleaning Docker artifacts..."
	$(DOCKER_COMPOSE) down -v
	docker system prune -f
	@echo "✅ Docker cleaned"

docker-logs:
	@echo "📋 Showing Docker logs..."
	$(DOCKER_COMPOSE) logs -f app

# ============================================================================
# Development Targets
# ============================================================================

dev-db:
	@echo "🐳 Starting MySQL & Redis only..."
	$(DOCKER_COMPOSE) up -d mysql redis
	@sleep 2
	@echo "✅ Database services started"
	@echo ""
	@echo "Run migrations:"
	@echo "  make migrate"
	@echo ""
	@echo "Start server:"
	@echo "  make run"
	@echo ""

health:
	@echo "🏥 Checking service health..."
	@echo ""
	@echo "MySQL:"
	@docker exec digital-wallet-mysql mysqladmin ping -h localhost --quiet && echo "  ✅ MySQL is running" || echo "  ❌ MySQL is down"
	@echo ""
	@echo "Redis:"
	@docker exec digital-wallet-redis redis-cli ping > /dev/null 2>&1 && echo "  ✅ Redis is running" || echo "  ❌ Redis is down"
	@echo ""
	@echo "API:"
	@curl -s http://localhost:8080/v1/health > /dev/null && echo "  ✅ API is running" || echo "  ❌ API is down"
	@echo ""

# ============================================================================
# Combined Development Workflows
# ============================================================================

.PHONY: dev setup all

dev: docker-up migrate seed
	@echo "🎯 Development environment ready!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run: make run"
	@echo "  2. Test: curl -H 'X-User-ID: user1' http://localhost:8080/v1/wallets"
	@echo ""

setup: install-deps docker-up migrate seed build
	@echo "✅ Project setup complete!"
	@echo ""
	@echo "Start development with:"
	@echo "  make run"
	@echo ""

all: clean install-deps lint test build
	@echo "✅ Full build successful!"
	@echo ""
	@echo "Binaries ready:"
	@echo "  - $(BINARY_SERVER)"
	@echo "  - $(BINARY_MIGRATE)"
	@echo ""

# ============================================================================
# Default Target
# ============================================================================

.DEFAULT_GOAL := help
