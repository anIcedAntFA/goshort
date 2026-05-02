# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoShort is a self-hosted URL shortener built in Go — a dual-purpose project: a practical tool and a learning vehicle for Go, system design, caching, and MCP/AI integration.

**Current state:** Design/planning phase. The architecture is fully documented in `docs/DESIGN.md`; no Go source files exist yet.

## Commands

These commands are planned but not yet implemented. Update this section as the project progresses.

```bash
# Build
go build ./cmd/server/...
go build ./cmd/cli/...

# Test (table-driven, TDD approach)
go test ./...
go test ./internal/shortener/...     # Run a single package
go test -run TestServiceCreate ./... # Run a single test
go test -fuzz FuzzValidator ./...    # Fuzz testing

# Lint (when .golangci.yml is added)
golangci-lint run

# sqlc code generation
sqlc generate

# Docker
docker compose up -d
```

## Architecture

Three-layer pragmatic structure — avoid over-engineering:

```
cmd/
├── server/main.go     # HTTP server entry point
└── cli/main.go        # CLI entry point (Phase 3)

internal/
├── shortener/         # Core business logic (service, encoder, validator)
├── storage/           # Storage interface + SQLite (sqlc) implementation
├── cache/             # Cache interface + noop/memory/redis implementations
├── api/               # Chi HTTP handlers, router, middleware, error types
├── mcp/               # MCP server tools (Phase 4)
└── config/            # Koanf config loading (goshort.toml + env vars)

db/
├── schema.sql
├── queries.sql
└── sqlc.yaml

api-tests/             # Bruno API tests (.bru files, not Postman)
```

**Layer boundaries:** `api/` and `mcp/` call `shortener/`; `shortener/` calls `storage/` and `cache/` interfaces only — never concrete types.

## Key Design Decisions

Consult `docs/DESIGN.md` for full rationale. Critical decisions:

- **Short code generation:** Counter + Sqids library (zero collision, non-sequential). Generated codes use `[a-zA-Z0-9]` only; custom aliases allow hyphens (`my-link`) — this distinction avoids collision between generated and custom codes.
- **Database:** SQLite via sqlc (Phases 1–4), PostgreSQL (Phase 5+). Two tables: `urls` and `counter`.
- **Caching:** Cache-aside pattern. `cache.driver` config: `none | memory | redis`. Interface must be satisfied by all three; never bypass the interface.
- **Redirect strategy:** 302 (temporary) not 301 — 301 is browser-cached permanently and kills click analytics.
- **HTTP router:** Chi v5 (stdlib-compatible, no lock-in).
- **Logging:** `slog` stdlib only — zero dependencies.
- **Auth:** API key (Phase 3). MCP tools require auth to be implemented first (Phase 3 before Phase 4).
- **Scaling:** All three layers are intentional learning exercises — DB index (always on) → in-process cache → CDN edge cache.

## Phased Roadmap

| Phase | Focus | Entry Point |
|-------|-------|-------------|
| 1 | Core library: SQLite, sqlc, Sqids, TDD | `internal/shortener/` |
| 2 | HTTP API (Chi), caching, slog, Prometheus | `internal/api/` |
| 3 | Auth, CLI (Cobra), rate limiting, Docker, Fly.io | `cmd/cli/`, `internal/config/` |
| 3.5 | Bare VPS ops: Nginx, systemd, Certbot | Infrastructure only |
| 4 | MCP server (mcp-go), Claude/Cursor integration | `internal/mcp/` |
| 5+ | Analytics, spam detection, PostgreSQL, Redis counter | Incremental |

Implement phases in order — each phase is a deployable milestone.

## Technology Stack

| Concern | Library |
|---------|---------|
| HTTP | `go-chi/chi/v5` |
| DB access | `sqlc-dev/sqlc` (generates type-safe Go from SQL) |
| Short codes | `sqids-org/sqids-go` |
| CLI | `spf13/cobra` |
| Config | `knadh/koanf/v2` + TOML parser + env provider |
| Cache (Redis) | `redis/go-redis/v9` |
| Metrics | `prometheus/client_golang` |
| Rate limiting | `golang.org/x/time/rate` |
| MCP server | `mark3labs/mcp-go` |
| API testing | Bruno (`.bru` files in `api-tests/`) |

## Configuration

App reads `goshort.toml` at runtime (loaded via Koanf):

```toml
[server]
port = 8080
base_url = "https://short.yourdomain.com"

[storage]
driver = "sqlite"
sqlite_path = "./data/goshort.db"

[cache]
driver = "none"   # none | memory | redis

[auth]
api_key = ""

[rate_limit]
enabled = false
requests_per_minute = 60

[logging]
level = "info"
format = "json"
```

## Code Conventions

- Tabs for Go files, 2-space spaces for YAML/JSON/Markdown (enforced by `.editorconfig`)
- Error handling: wrap with `%w` for `errors.Is/As`; handle errors once, don't log-then-return
- Interfaces defined in the consumer package (`storage.Storage`, `cache.Cache`) — not in the implementation package
- Table-driven tests with `t.Run` subtests; use `httptest` for HTTP handler tests
- All SQL lives in `db/queries.sql`; never write raw SQL strings in Go code
- Config struct tags use `koanf:"field_name"` (not `mapstructure`)

## Reference Documentation

- `docs/DESIGN.md` — Full system design, API spec, data model, scaling rationale, risk table
- `docs/LEARNING.md` — Go philosophy, patterns, and GoShort-specific knowledge map with code location annotations

## Current State (update after each milestone)

Phase 2 in progress.

- ✅ 2.1: Chi router + handlers
- ✅ 2.2: HTTP tests (httptest)  
- ✅ 2.3: Server wiring (main.go, graceful shutdown, cleanup job)
- ✅ 2.4: Logging + Metrics (slog + Prometheus)
- ✅ 2.5: Caching layers (cache-aside, Redis, TTL alignment)
- 🔲 2.6: Configuration (Koanf) (next)
- 🔲 2.7: API Documentation
- 🔲 2.8: Bruno test collection
- 🔲 2.9: CI + Coverage
