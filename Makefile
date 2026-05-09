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
	@grep -E '^[a-zA-Z_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build/server build/cli ## Build all binaries

.PHONY: build/server
build/server: ## Build the server binary (goshort)
	go build $(LDFLAGS) -o $(BINARY_SERVER) $(CMD_SERVER)

.PHONY: build/cli
build/cli: ## Build the CLI binary (goshort-cli)
	go build $(LDFLAGS) -o $(BINARY_CLI) $(CMD_CLI)

CONFIG ?=

.PHONY: run
run: ## Run the server (CONFIG=goshort.toml to load a file; Ctrl+C exit is expected)
	go run $(CMD_SERVER) $(if $(CONFIG),--config $(CONFIG),)

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test/cover
test/cover: ## Run tests with HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: test/race
test/race: ## Run tests with race detector
	go test -race ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: lint/fix
lint/fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

.PHONY: sqlc
sqlc: ## Generate type-safe Go from SQL
	sqlc generate

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

.PHONY: dev/redis
dev/redis: ## Start Redis for local development
	docker compose -f docker-compose.dev.yml up -d
	@echo "Redis running at localhost:6379"

.PHONY: dev/redis/stop
dev/redis/stop: ## Stop local Redis
	docker compose -f docker-compose.dev.yml down

.PHONY: test/redis
test/redis: ## Run all tests including Redis integration tests
	go test -race -tags redis -count=1 ./...

.PHONY: test/all
test/all: ## Run all tests (unit + Redis integration, auto-detects Redis)
	@if docker compose -f docker-compose.dev.yml ps --status running 2>/dev/null | grep -q redis; then \
		echo "Redis detected — running with -tags redis"; \
		go test -race -tags redis -count=1 ./...; \
	else \
		echo "Redis not running — skipping Redis tests"; \
		echo "Start Redis with: make dev/redis"; \
		go test -race -count=1 ./...; \
	fi

.PHONY: test/bench
test/bench: ## Run benchmarks with memory stats (3s per bench)
	go test -bench=. -benchmem -benchtime=3s ./...

.PHONY: test/fuzz/url
test/fuzz/url: ## Fuzz URL validator for 30s
	go test -fuzz=FuzzValidateURL -fuzztime=30s ./internal/shortener/

.PHONY: test/fuzz/alias
test/fuzz/alias: ## Fuzz alias validator for 30s
	go test -fuzz=FuzzValidateAlias -fuzztime=30s ./internal/shortener/

.PHONY: test/fuzz/encoder
test/fuzz/encoder: ## Fuzz encoder for 30s
	go test -fuzz=FuzzSqidsEncoder_Encode -fuzztime=30s ./internal/encoder/

.PHONY: migrate
migrate: ## Run pending database migrations (requires GOOSE_DB env or default path)
	goose -dir db/migrations sqlite3 ./data/goshort.db up

.PHONY: migrate/status
migrate/status: ## Show applied migration status
	goose -dir db/migrations sqlite3 ./data/goshort.db status

.PHONY: migrate/down
migrate/down: ## Roll back the last migration
	goose -dir db/migrations sqlite3 ./data/goshort.db down

.PHONY: docker/up
docker/up: ## Start services with Docker Compose
	docker compose up -d

.PHONY: docker/down
docker/down: ## Stop Docker Compose services
	docker compose down

.PHONY: clean
clean: ## Remove bin/, dist/, coverage.out
	rm -rf $(BIN_DIR)/ dist/ coverage.out

