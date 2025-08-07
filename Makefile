.PHONY: help build run test clean docker-build docker-run docker-stop dev deps lint format

# Variables
APP_NAME := creative-studio-server
DOCKER_IMAGE := $(APP_NAME):latest
GO_VERSION := 1.21

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(APP_NAME) .

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@./bin/$(APP_NAME)

dev: ## Run the application in development mode with hot reload
	@echo "Starting development server with hot reload..."
	@go run . || (echo "Installing air..." && go install github.com/cosmtrek/air@latest && air)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && golangci-lint run)

format: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run with Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

docker-stop: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose down

docker-logs: ## View Docker logs
	@docker-compose logs -f

docker-restart: ## Restart Docker services
	@docker-compose restart

migrate-up: ## Run database migrations up
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go up

migrate-down: ## Run database migrations down
	@echo "Rolling back database migrations..."
	@go run cmd/migrate/main.go down

migrate-create: ## Create new migration file
	@echo "Creating new migration file..."
	@read -p "Enter migration name: " name; \
	go run cmd/migrate/main.go create $$name

seed: ## Seed database with sample data
	@echo "Seeding database..."
	@go run cmd/seed/main.go

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest

generate-docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@swag init

setup: deps install-tools ## Setup development environment
	@echo "Development environment setup complete!"

deploy-dev: docker-build docker-run ## Deploy to development environment
	@echo "Deployed to development environment"

deploy-prod: ## Deploy to production (placeholder)
	@echo "Production deployment not implemented yet"

backup-db: ## Backup database
	@echo "Creating database backup..."
	@docker exec creative-studio-postgres pg_dump -U postgres creative_studio > backup_$(shell date +%Y%m%d_%H%M%S).sql

restore-db: ## Restore database from backup
	@echo "Enter backup file path:"
	@read -p "Backup file: " file; \
	docker exec -i creative-studio-postgres psql -U postgres creative_studio < $$file

monitor: ## Monitor application logs
	@echo "Monitoring application logs..."
	@docker-compose logs -f app

status: ## Check service status
	@echo "Service Status:"
	@docker-compose ps

update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy