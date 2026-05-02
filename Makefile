VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

BINARY_SERVER := goshort
BINARY_CLI    := goshort-cli
LDFLAGS       := -ldflags="-s -w -X main.version=$(VERSION)"

CMD_SERVER := ./cmd/server
CMD_CLI    := ./cmd/cli

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

.PHONY: docker/up
docker/up: ## Start services with Docker Compose
	docker compose up -d

.PHONY: docker/down
docker/down: ## Stop Docker Compose services
	docker compose down

.PHONY: clean
clean: ## Remove binaries, dist/, coverage.out
	rm -f $(BINARY_SERVER) $(BINARY_CLI) coverage.out
	rm -rf dist/

