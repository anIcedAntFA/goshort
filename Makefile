VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

BIN_DIR       := bin
BINARY_SERVER := $(BIN_DIR)/goshort
BINARY_CLI    := $(BIN_DIR)/goshort-cli
LDFLAGS       := -ldflags="-s -w -X main.version=$(VERSION)"

CMD_SERVER := ./cmd/server
CMD_CLI    := ./cmd/cli

$(shell mkdir -p $(BIN_DIR))

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────

.PHONY: build build/server build/cli
build: build/server build/cli ## Build all binaries
build/server: ## Build the server binary (goshort)
	go build $(LDFLAGS) -o $(BINARY_SERVER) $(CMD_SERVER)
build/cli: ## Build the CLI binary (goshort-cli)
	go build $(LDFLAGS) -o $(BINARY_CLI) $(CMD_CLI)

# ── Run ───────────────────────────────────────────

CONFIG ?=
.PHONY: run
run: ## Run the server (CONFIG=goshort.toml to load a file; Ctrl+C exit is expected)
	go run $(CMD_SERVER) $(if $(CONFIG),--config $(CONFIG),)

# ── Test ──────────────────────────────────────────

.PHONY: test test/unit test/race test/cover test/redis test/all test/bench
test: ## Run all tests
	go test ./...
test/unit: ## Run unit tests only (fast, no Redis)
	go test -count=1 -timeout=30s ./...
test/race: ## Run tests with race detector
	go test -race ./...
test/cover: ## Run tests with HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
test/redis: ## Run all tests including Redis integration tests
	go test -race -tags redis -count=1 ./...
test/all: ## Run all tests (unit + Redis integration, auto-detects Redis)
	@if docker compose -f docker-compose.dev.yml ps --status running 2>/dev/null | grep -q redis; then \
		echo "Redis detected — running with -tags redis"; \
		go test -race -tags redis -count=1 ./...; \
	else \
		echo "Redis not running — skipping Redis tests"; \
		echo "Start Redis with: make dev/redis"; \
		go test -race -count=1 ./...; \
	fi
test/bench: ## Run benchmarks with memory stats (3s per bench)
	go test -bench=. -benchmem -benchtime=3s ./...

# ── Fuzz ──────────────────────────────────────────

.PHONY: test/fuzz/url test/fuzz/alias test/fuzz/encoder
test/fuzz/url: ## Fuzz URL validator for 30s
	go test -fuzz=FuzzValidateURL -fuzztime=30s ./internal/shortener/
test/fuzz/alias: ## Fuzz alias validator for 30s
	go test -fuzz=FuzzValidateAlias -fuzztime=30s ./internal/shortener/
test/fuzz/encoder: ## Fuzz encoder for 30s
	go test -fuzz=FuzzSqidsEncoder_Encode -fuzztime=30s ./internal/encoder/

# ── Lint ──────────────────────────────────────────

.PHONY: lint lint/fix
lint: ## Run golangci-lint
	golangci-lint run ./...
lint/fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

# ── Database ──────────────────────────────────────

.PHONY: sqlc migrate migrate/status migrate/down tidy
sqlc: ## Generate type-safe Go from SQL
	sqlc generate
migrate: ## Run pending database migrations
	goose -dir db/migrations sqlite3 ./data/goshort.db up
migrate/status: ## Show applied migration status
	goose -dir db/migrations sqlite3 ./data/goshort.db status
migrate/down: ## Roll back the last migration
	goose -dir db/migrations sqlite3 ./data/goshort.db down
tidy: ## Tidy go.mod and go.sum
	go mod tidy

# ── Docker / Dev Services ─────────────────────────

.PHONY: dev/redis dev/redis/stop docker/up docker/down
dev/redis: ## Start Redis for local development
	docker compose -f docker-compose.dev.yml up -d
	@echo "Redis running at localhost:6379"
dev/redis/stop: ## Stop local Redis
	docker compose -f docker-compose.dev.yml down
docker/up: ## Start services with Docker Compose
	docker compose up -d
docker/down: ## Stop Docker Compose services
	docker compose down

# ── CI ────────────────────────────────────────────

.PHONY: ci
ci: lint test build ## Run full CI locally (lint + test + build)

# ── Website (Astro + Bun) ─────────────────────────

WEBSITE_DIR := website

.PHONY: website/dev website/build website/check website/worker/dev website/worker/deploy
website/dev: ## Start Astro dev server
	cd $(WEBSITE_DIR) && bun run dev
website/build: ## Build Astro static site to website/dist/
	cd $(WEBSITE_DIR) && bun run build
website/check: ## Lint + format website with Biome
	cd $(WEBSITE_DIR) && bun run check
website/worker/dev: website/build ## Build site then run Cloudflare Worker locally
	cd $(WEBSITE_DIR) && wrangler dev
website/worker/deploy: website/build ## Build site then deploy Worker to Cloudflare
	cd $(WEBSITE_DIR) && wrangler deploy

# ── Housekeeping ──────────────────────────────────

.PHONY: clean
clean: ## Remove bin/, dist/, coverage.out
	rm -rf $(BIN_DIR)/ dist/ coverage.out
