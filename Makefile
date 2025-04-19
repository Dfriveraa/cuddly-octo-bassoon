# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

# Create DB container
docker-run:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Unit tests only (excludes integration tests)
test-unit:
	@echo "Running unit tests..."
	@go test ./internal/domain/... -v

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@go test ./internal/adapters/repository -v
	@go test ./internal/database -v
	@go test ./internal/test/integration/... -v

# Run all tests (both unit and integration)
test: test-unit test-integration
	@echo "All tests completed!"

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g internal/server/routes.go -o docs; \
	else \
		go install github.com/swaggo/swag/cmd/swag@latest && \
		swag init -g internal/server/routes.go -o docs; \
	fi
	@echo "Swagger documentation generated. Access at: http://localhost:8080/swagger/index.html"

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test test-unit test-integration clean watch docker-run docker-down swagger
