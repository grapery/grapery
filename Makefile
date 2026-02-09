.PHONY: run run-with-config test lint build docker migrate \
        run-server run-vippay \
        build-server build-vippay build-all \
        clean help

# ========== Default Targets ==========

run: run-server

run-with-config:
	@if [ ! -f config.yaml ]; then \
		echo "⚠️  config.yaml not found, creating from example..."; \
		cp config.yaml.example config.yaml; \
		echo "✅ config.yaml created. Please edit it with your settings."; \
		echo ""; \
	fi
	go run ./cmd/server -config config.yaml

# ========== Port Configuration ==========
# server:  8080 (default)
# vippay:  8060

SERVER_PORT ?= 8080
VIPPAY_PORT ?= 8060

# ========== Database Configuration ==========
# Override these with environment variables or command line
DB_USERNAME ?= root
DB_PASSWORD ?= 12345678
DB_ADDRESS ?= localhost
DB_DATABASE ?= grapery

# ========== Individual Service Run Commands ==========

run-server:
	@echo "🚀 Starting Grapery Server on port $(SERVER_PORT)..."
	GRAPERY_HTTP_PORT=$(SERVER_PORT) \
	DB_USERNAME=$(DB_USERNAME) \
	DB_PASSWORD=$(DB_PASSWORD) \
	DB_ADDRESS=$(DB_ADDRESS) \
	DB_DATABASE=$(DB_DATABASE) \
	go run ./cmd/server

run-vippay:
	@echo "🚀 Starting VIP Payment Service on port $(VIPPAY_PORT)..."
	VIPPAY_PORT=$(VIPPAY_PORT) \
	DB_USERNAME=$(DB_USERNAME) \
	DB_PASSWORD=$(DB_PASSWORD) \
	DB_ADDRESS=$(DB_ADDRESS) \
	DB_DATABASE=$(DB_DATABASE) \
	go run ./cmd/vippay

# ========== Run with Config ==========

run-server-with-config:
	@if [ ! -f config.yaml ]; then \
		echo "⚠️  config.yaml not found, creating from example..."; \
		cp config.yaml.example config.yaml; \
		echo "✅ config.yaml created. Please edit it with your settings."; \
	fi
	@echo "🚀 Starting Grapery Server on port $(SERVER_PORT) with config..."
	GRAPERY_HTTP_PORT=$(SERVER_PORT) \
	DB_USERNAME=$(DB_USERNAME) \
	DB_PASSWORD=$(DB_PASSWORD) \
	DB_ADDRESS=$(DB_ADDRESS) \
	DB_DATABASE=$(DB_DATABASE) \
	go run ./cmd/server -config config.yaml

run-vippay-with-config:
	@echo "🚀 Starting VIP Payment Service on port $(VIPPAY_PORT) with config..."
	VIPPAY_PORT=$(VIPPAY_PORT) \
	DB_USERNAME=$(DB_USERNAME) \
	DB_PASSWORD=$(DB_PASSWORD) \
	DB_ADDRESS=$(DB_ADDRESS) \
	DB_DATABASE=$(DB_DATABASE) \
	go run ./cmd/vippay -config vippay.json

# ========== Build Commands ==========

build: build-all

build-server:
	@echo "🔨 Building Grapery Server..."
	go build -o bin/grapery-server ./cmd/server
	@echo "✅ Built: bin/grapery-server"

build-vippay:
	@echo "🔨 Building VIP Payment Service..."
	go build -o bin/grapery-vippay ./cmd/vippay
	@echo "✅ Built: bin/grapery-vippay"

build-all:
	@echo "🔨 Building all services..."
	@mkdir -p bin
	@$(MAKE) build-server
	@$(MAKE) build-vippay
	@echo ""
	@echo "✅ All services built successfully!"
	@ls -la bin/

# ========== Development Tools ==========

lint:
	gofmt -w .
	go vet ./...

test:
	go test -v ./...

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	@echo "✅ Clean complete"

# ========== Docker ==========

docker:
	docker build -t grapery -f Dockerfile ..

docker-server:
	docker build -t grapery-server -f Dockerfile --target server ..

docker-vippay:
	docker build -t grapery-vippay -f Dockerfile --target vippay ..

# ========== Database ==========

migrate:
	@echo "Database migrations run automatically on startup"

# ========== Help ==========

help:
	@echo "Grapery Makefile - Unified Service Management"
	@echo ""
	@echo "Services:"
	@echo "  server   - Main Grapery API Server  (port $(SERVER_PORT))"
	@echo "  vippay   - VIP Payment Service      (port $(VIPPAY_PORT))"
	@echo ""
	@echo "Run Commands:"
	@echo "  make run                    - Run server (default, port $(SERVER_PORT))"
	@echo "  make run-server             - Run Grapery Server (port $(SERVER_PORT))"
	@echo "  make run-vippay             - Run VIP Payment Service (port $(VIPPAY_PORT))"
	@echo "  make run-server-with-config - Run Server with config.yaml"
	@echo "  make run-vippay-with-config - Run VIP Pay with vippay.json"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build                  - Build all services"
	@echo "  make build-all              - Build all services"
	@echo "  make build-server           - Build Grapery Server"
	@echo "  make build-vippay           - Build VIP Payment Service"
	@echo ""
	@echo "Development:"
	@echo "  make lint                   - Format and vet code"
	@echo "  make test                   - Run tests"
	@echo "  make clean                  - Remove build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker                 - Build main Docker image"
	@echo "  make docker-server          - Build Server Docker image"
	@echo "  make docker-vippay          - Build VIP Pay Docker image"
	@echo ""
	@echo "Database Configuration (override with env vars):"
	@echo "  DB_USERNAME=$(DB_USERNAME)"
	@echo "  DB_PASSWORD=$(DB_PASSWORD)"
	@echo "  DB_ADDRESS=$(DB_ADDRESS)"
	@echo "  DB_DATABASE=$(DB_DATABASE)"
	@echo ""
	@echo "Example with custom settings:"
	@echo "  make run-vippay DB_PASSWORD=mypassword"
