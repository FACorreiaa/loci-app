# Database connection settings
DB_HOST ?= localhost
DB_PORT ?= 5454
DB_USER ?= loci
DB_PASSWORD ?= loci123
DB_NAME ?= loci-dev
DB_URL = "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"
MIGRATIONS_DIR = internal/db/migrations

# Migration commands using Goose
migrate-up: ## Run all pending migrations
	@echo "Running migrations..."
	@goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) up

migrate-down: ## Rollback the last migration
	@echo "Rolling back last migration..."
	@goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) down

migrate-status: ## Show migration status
	@goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) status

migrate-create: ## Create a new migration (usage: make migrate-create NAME=add_column)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=add_column"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	@goose -s -dir $(MIGRATIONS_DIR) create $(NAME) sql
	@echo "Migration created successfully!"

migrate-reset: ## Reset database (down all, then up all)
	@echo "Resetting database..."
	@goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) reset

migrate-version: ## Show current migration version
	@goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) version

testifylint:
	testifylint ./...

testifylint-fix:
	testifylint -fix ./...

static:
	staticcheck ./...

lint: ## Runs linter for .go files
	golangci-lint run --config .golangci.yml
	@echo "Go lint passed successfully"

# Run templ generation in watch mode
templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false

t-fmt:
	templ fmt .

# Run air for Go hot reload
server:
	air \
	--build.cmd "go build -o tmp/bin/main ./main.go" \
	--build.bin "tmp/bin/main" \
	--build.delay "100" \
	--build.exclude_dir "node_modules" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

# Watch Tailwind CSS changes
tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

tailwind-build:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --build

#db up
db-up:
	docker compose up --build

db-down:
	docker compose down

db-delete:
	docker compose down -v

# Start development server with all watchers
dev:
	make -j3 db-up tailwind templ server

# OTEL specific targets
otel-up: ## Start OpenTelemetry collector and observability stack
	@echo "Starting observability stack..."
	@docker compose up -d otel-collector prometheus alertmanager

otel-down: ## Stop observability stack
	@docker compose down otel-collector prometheus alertmanager

otel-logs: ## Show OTEL collector logs
	@docker compose logs -f otel-collector

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'