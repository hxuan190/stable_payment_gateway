.PHONY: help migrate-up migrate-down migrate-down-one migrate-create migrate-version migrate-force db-backup test build run

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Default database URL (override with DATABASE_URL env var)
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
MIGRATIONS_PATH = ./migrations

# Colors for output
COLOR_RESET = \033[0m
COLOR_BOLD = \033[1m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m
COLOR_BLUE = \033[34m

help: ## Show this help message
	@echo "$(COLOR_BOLD)Available commands:$(COLOR_RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'

# Database Migrations

migrate-up: ## Run all pending database migrations
	@echo "$(COLOR_BLUE)Running database migrations...$(COLOR_RESET)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "$(COLOR_GREEN)✓ Migrations completed$(COLOR_RESET)"

migrate-down: ## Rollback all database migrations
	@echo "$(COLOR_YELLOW)Rolling back ALL migrations...$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down; \
		echo "$(COLOR_GREEN)✓ All migrations rolled back$(COLOR_RESET)"; \
	else \
		echo "Cancelled"; \
	fi

migrate-down-one: ## Rollback the last database migration
	@echo "$(COLOR_YELLOW)Rolling back last migration...$(COLOR_RESET)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1
	@echo "$(COLOR_GREEN)✓ Last migration rolled back$(COLOR_RESET)"

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=create_new_table)
	@if [ -z "$(NAME)" ]; then \
		echo "$(COLOR_YELLOW)Error: NAME is required$(COLOR_RESET)"; \
		echo "Usage: make migrate-create NAME=create_new_table"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)
	@echo "$(COLOR_GREEN)✓ Migration files created$(COLOR_RESET)"

migrate-version: ## Show current migration version
	@echo "$(COLOR_BLUE)Current migration version:$(COLOR_RESET)"
	@migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version || echo "No migrations applied yet"

migrate-force: ## Force set migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(COLOR_YELLOW)Error: VERSION is required$(COLOR_RESET)"; \
		echo "Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@echo "$(COLOR_YELLOW)⚠️  Forcing migration version to $(VERSION)...$(COLOR_RESET)"
	@read -p "Are you sure? This can corrupt your database! [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(VERSION); \
		echo "$(COLOR_GREEN)✓ Migration version forced to $(VERSION)$(COLOR_RESET)"; \
	else \
		echo "Cancelled"; \
	fi

# Database Operations

db-backup: ## Backup database to file
	@echo "$(COLOR_BLUE)Backing up database...$(COLOR_RESET)"
	@mkdir -p backups
	@BACKUP_FILE="backups/backup_$$(date +%Y%m%d_%H%M%S).sql"; \
	pg_dump -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME) > $$BACKUP_FILE; \
	echo "$(COLOR_GREEN)✓ Database backed up to $$BACKUP_FILE$(COLOR_RESET)"

db-restore: ## Restore database from backup (usage: make db-restore FILE=backups/backup_20231116_120000.sql)
	@if [ -z "$(FILE)" ]; then \
		echo "$(COLOR_YELLOW)Error: FILE is required$(COLOR_RESET)"; \
		echo "Usage: make db-restore FILE=backups/backup_20231116_120000.sql"; \
		exit 1; \
	fi
	@echo "$(COLOR_YELLOW)⚠️  Restoring database from $(FILE)...$(COLOR_RESET)"
	@read -p "This will overwrite the current database! Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME) < $(FILE); \
		echo "$(COLOR_GREEN)✓ Database restored from $(FILE)$(COLOR_RESET)"; \
	else \
		echo "Cancelled"; \
	fi

db-create: ## Create database
	@echo "$(COLOR_BLUE)Creating database $(DB_NAME)...$(COLOR_RESET)"
	@createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME) || echo "Database may already exist"
	@echo "$(COLOR_GREEN)✓ Database $(DB_NAME) ready$(COLOR_RESET)"

db-drop: ## Drop database
	@echo "$(COLOR_YELLOW)⚠️  Dropping database $(DB_NAME)...$(COLOR_RESET)"
	@read -p "Are you sure? All data will be lost! [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		dropdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME); \
		echo "$(COLOR_GREEN)✓ Database $(DB_NAME) dropped$(COLOR_RESET)"; \
	else \
		echo "Cancelled"; \
	fi

db-reset: db-drop db-create migrate-up ## Drop, create, and migrate database

db-shell: ## Open PostgreSQL shell
	@psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)

# Application Build & Run

build: ## Build all services
	@echo "$(COLOR_BLUE)Building services...$(COLOR_RESET)"
	@go build -o bin/api ./cmd/api
	@go build -o bin/listener ./cmd/listener
	@go build -o bin/worker ./cmd/worker
	@go build -o bin/admin ./cmd/admin
	@echo "$(COLOR_GREEN)✓ Build completed$(COLOR_RESET)"

run-api: ## Run API server
	@echo "$(COLOR_BLUE)Starting API server...$(COLOR_RESET)"
	@go run ./cmd/api/main.go

run-listener: ## Run blockchain listener
	@echo "$(COLOR_BLUE)Starting blockchain listener...$(COLOR_RESET)"
	@go run ./cmd/listener/main.go

run-worker: ## Run background worker
	@echo "$(COLOR_BLUE)Starting background worker...$(COLOR_RESET)"
	@go run ./cmd/worker/main.go

run-admin: ## Run admin server
	@echo "$(COLOR_BLUE)Starting admin server...$(COLOR_RESET)"
	@go run ./cmd/admin/main.go

# Testing

test: ## Run all tests
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)✓ Tests completed$(COLOR_RESET)"

test-coverage: test ## Run tests with coverage report
	@go tool cover -html=coverage.out

test-unit: ## Run unit tests only
	@go test -v -short ./...

test-integration: ## Run integration tests only
	@go test -v -run Integration ./...

# Development

install-tools: ## Install development tools
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(COLOR_GREEN)✓ Tools installed$(COLOR_RESET)"

lint: ## Run linter
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@golangci-lint run ./...

fmt: ## Format code
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@gofmt -s -w .

tidy: ## Tidy go.mod
	@echo "$(COLOR_BLUE)Tidying go.mod...$(COLOR_RESET)"
	@go mod tidy

clean: ## Clean build artifacts
	@echo "$(COLOR_BLUE)Cleaning...$(COLOR_RESET)"
	@rm -rf bin/
	@rm -f coverage.out
	@echo "$(COLOR_GREEN)✓ Cleaned$(COLOR_RESET)"

# Docker

docker-up: ## Start Docker Compose services
	@echo "$(COLOR_BLUE)Starting Docker services...$(COLOR_RESET)"
	@docker-compose up -d
	@echo "$(COLOR_GREEN)✓ Docker services started$(COLOR_RESET)"

docker-down: ## Stop Docker Compose services
	@echo "$(COLOR_BLUE)Stopping Docker services...$(COLOR_RESET)"
	@docker-compose down
	@echo "$(COLOR_GREEN)✓ Docker services stopped$(COLOR_RESET)"

docker-logs: ## View Docker Compose logs
	@docker-compose logs -f

docker-build: ## Build Docker images
	@echo "$(COLOR_BLUE)Building Docker images...$(COLOR_RESET)"
	@docker-compose build
	@echo "$(COLOR_GREEN)✓ Docker images built$(COLOR_RESET)"

# Environment Setup

setup: install-tools db-create migrate-up ## Initial project setup
	@echo "$(COLOR_GREEN)✓ Project setup completed$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Next steps:$(COLOR_RESET)"
	@echo "  1. Copy .env.example to .env and configure your settings"
	@echo "  2. Run 'make run-api' to start the API server"
	@echo "  3. Run 'make run-listener' to start the blockchain listener"

.DEFAULT_GOAL := help
