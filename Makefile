# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building..."
	
	
	@go build -o main.exe cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go
# Create DB container
docker-run:
	@docker compose up --build

# Shutdown DB container
docker-down:
	@docker compose down

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# ---- Production / deploy ----

# Build the React frontend -> frontend/dist
frontend-build:
	@cd frontend && npm ci && npm run build

# Build the production web server binary (serves /api + frontend/dist)
build-web:
	@go build -trimpath -ldflags="-s -w" -o web.exe cmd/web/main.go

# Build frontend + binary, then run the single-artifact production server
serve: frontend-build build-web
	@./web.exe

# Build the Docker image
docker-build:
	@docker build -t financal-management:latest .

# Full stack (app + mysql) via docker compose
compose-up:
	@docker compose up --build -d

compose-down:
	@docker compose down

# Live Reload
watch:
	@powershell -ExecutionPolicy Bypass -Command "if (Get-Command air -ErrorAction SilentlyContinue) { \
		air; \
		Write-Output 'Watching...'; \
	} else { \
		Write-Output 'Installing air...'; \
		go install github.com/air-verse/air@latest; \
		air; \
		Write-Output 'Watching...'; \
	}"

.PHONY: all build run test clean watch docker-run docker-down itest \
	frontend-build build-web serve docker-build compose-up compose-down
