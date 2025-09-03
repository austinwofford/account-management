.PHONY: help build test test-coverage lint fmt vet migrate-up migrate-down migrate-create db-up db-down clean

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build
build: ## Build the application
	go build -o bin/account-management ./cmd/account-management

# Testing
test: ## Run tests
	go test ./...

test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

test-coverage-func: ## Show test coverage by function
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Code quality
lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

# All quality checks
check: fmt vet lint test ## Run all code quality checks

# migrations
migrate-up:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/account_management?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/account_management?sslmode=disable" down

migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	migrate create -ext sql -dir migrations $(name)
