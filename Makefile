.PHONY: run run-with-config test lint build docker migrate mock-load review-demo-load \
        run-server run-vippay sync-apple-iap-env \
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
	@echo "   Apple IAP: run 'bash scripts/sync_apple_iap_env.sh' after gh auth login, or set vars in .env"
	@set -a; [ -f .env ] && . ./.env; set +a; \
	if [ -z "$${APPLE_PRIVATE_KEY:-}" ] && [ -n "$${APPLE_PRIVATE_KEY_PATH:-}" ] && [ -f "$${APPLE_PRIVATE_KEY_PATH}" ]; then \
		export APPLE_PRIVATE_KEY="$$(cat "$${APPLE_PRIVATE_KEY_PATH}")"; \
	fi; \
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
	@set -a; [ -f .env ] && . ./.env; set +a; \
	if [ -z "$${APPLE_PRIVATE_KEY:-}" ] && [ -n "$${APPLE_PRIVATE_KEY_PATH:-}" ] && [ -f "$${APPLE_PRIVATE_KEY_PATH}" ]; then \
		export APPLE_PRIVATE_KEY="$$(cat "$${APPLE_PRIVATE_KEY_PATH}")"; \
	fi; \
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

# 从 GitHub Actions Variables 拉取 APPLE_* 并写入 .env + certs/AuthKey_*.p8（需 gh auth login）
sync-apple-iap-env:
	bash scripts/sync_apple_iap_env.sh

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

mock-load:
	@echo "Loading mock data (base + King) into $(DB_DATABASE)..."
	@mysql -u $(DB_USERNAME) -p$(DB_PASSWORD) -h $(DB_ADDRESS) $(DB_DATABASE) < scripts/mock_data.sql
	@mysql -u $(DB_USERNAME) -p$(DB_PASSWORD) -h $(DB_ADDRESS) $(DB_DATABASE) < migrations/king_mock_data.sql
	@echo "Mock data loaded successfully!"

# App Review demo account (jingjing@grapery.xyz) — idempotent, does NOT wipe other users.
# Order: main user/membership/content, then VipPay VIP grant (requires iap_products).
# Regenerate bcrypt/FNV: go run ./cmd/gen-review-demo
review-demo-load:
	@echo "Loading App Review demo account into $(DB_DATABASE)..."
	@mysql -u $(DB_USERNAME) -p$(DB_PASSWORD) -h $(DB_ADDRESS) $(DB_DATABASE) < scripts/app_review_demo_jingjing.sql
	@mysql -u $(DB_USERNAME) -p$(DB_PASSWORD) -h $(DB_ADDRESS) $(DB_DATABASE) < scripts/app_review_demo_jingjing_vippay.sql
	@echo "Review demo loaded. Login: jingjing@grapery.xyz / Gr@p3ryIap2026! (use Email in app)"

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
	@echo "  make run-vippay             - Run VIP Payment Service (port $(VIPPAY_PORT), loads .env if present)"
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
	@echo "Mock Data:"
	@echo "  make mock-load             - Load mock data (base + King, uses DB_* config)"
	@echo "  make review-demo-load      - App Review demo user (jingjing@grapery.xyz, idempotent)"
	@echo "  go run ./cmd/gen-review-demo - Print bcrypt hash + VipPay FNV for demo user"
	@echo ""
	@echo "Example with custom settings:"
	@echo "  make run-vippay DB_PASSWORD=mypassword"
	@echo ""
	@echo "VIPPay (.env or env vars, see env.vippay.example):"
	@echo "  APPLE_BUNDLE_ID / APPLE_ISSUER_ID / APPLE_KEY_ID / APPLE_PRIVATE_KEY — StoreKit 2 IAP"
	@echo "  WECHAT_APP_ID / WECHAT_APP_SECRET — WeChat mobile login"
	@echo "  GOOGLE_CLIENT_ID — Google ID token verification"
