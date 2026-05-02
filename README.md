# GoShort

[![CI](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml/badge.svg)](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/anIcedAntFA/goshort/graph/badge.svg)](https://codecov.io/gh/anIcedAntFA/goshort)
[![Go Report Card](https://goreportcard.com/badge/github.com/anIcedAntFA/goshort)](https://goreportcard.com/report/github.com/anIcedAntFA/goshort)
[![Release](https://img.shields.io/github/v/release/anIcedAntFA/goshort)](https://github.com/anIcedAntFA/goshort/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](go.mod)

Self-hosted URL shortener — single binary, SQLite-backed, API-key auth, Redis-optional caching.

```
https://docs.google.com/spreadsheets/d/1a2b3c4d5e6f7g8h  →  https://short.example.com/k7Xm2p
```

---

## Features

- **Zero-collision codes** — atomic SQLite counter + [Sqids](https://sqids.org): non-sequential, bijective, no retry loops
- **Custom aliases** — bring your own slug (`/my-link`); charset isolation prevents collision with generated codes
- **URL expiration** — configurable TTL; lazy expiry on read + hourly background cleanup
- **Switchable cache** — `none | memory | redis` at config time; cache-aside with TTL capped to remaining expiry
- **API key auth** — constant-time comparison; per-IP token bucket rate limiting
- **CLI client** — `goshort-cli` for shorten, list, stats, delete from the terminal
- **Prometheus metrics + structured logs** — `/metrics`, `slog` throughout, no extra dependencies
- **Self-documenting API** — OpenAPI spec + interactive Scalar UI at `/docs`

---

## Quick Start

### a. Docker Compose (recommended)

```bash
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/goshort.toml
docker compose up -d
```

Caddy handles TLS automatically. Edit `goshort.toml` to set your `base_url` and `api_key`.

### b. Binary (GitHub Releases)

```bash
curl -L https://github.com/anIcedAntFA/goshort/releases/latest/download/goshort_linux_amd64.tar.gz | tar xz
./goshort
```

Grab `goshort_darwin_arm64`, `goshort_windows_amd64`, etc. from the [releases page](https://github.com/anIcedAntFA/goshort/releases).

### c. go install

```bash
go install github.com/anIcedAntFA/goshort/cmd/server@latest
goshort
```

---

## Usage

**Create a short URL:**

```bash
curl -s -X POST http://localhost:8080/api/v1/urls \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"url": "https://example.com/very/long/path", "expires_in": "30d"}'
```

```json
{
  "short_code": "k7Xm2p",
  "short_url":  "http://localhost:8080/k7Xm2p",
  "original_url": "https://example.com/very/long/path",
  "expires_at": "2025-08-01T00:00:00Z"
}
```

**Redirect:**

```bash
curl -L http://localhost:8080/k7Xm2p
# → 302 to https://example.com/very/long/path
```

---

## CLI

```bash
# Install
go install github.com/anIcedAntFA/goshort/cmd/cli@latest

# Shorten
goshort-cli shorten https://example.com/long --alias my-link --expires 7d

# Pipe input
echo "https://example.com/long" | goshort-cli shorten

# List all URLs
goshort-cli list

# Inspect a code
goshort-cli stats k7Xm2p

# Delete
goshort-cli delete k7Xm2p
```

Config (`~/.goshort.toml`):

```toml
server_url = "http://localhost:8080"
api_key    = "your-api-key"
```

Override per-command: `--server-url`, `--api-key`, or env vars `GOSHORT_SERVER_URL` / `GOSHORT_API_KEY`.

---

## Configuration

```toml
[server]
port     = 8080
base_url = "https://short.yourdomain.com"  # used in API responses

[storage]
driver      = "sqlite"              # sqlite | postgres (Phase 5+)
sqlite_path = "./data/goshort.db"

[cache]
driver    = "none"                  # none | memory | redis
redis_url = "redis://localhost:6379"

[auth]
api_key = ""                        # empty = no auth

[rate_limit]
enabled             = false
requests_per_minute = 60            # per IP, token bucket

[shortener]
code_length    = 6
default_expiry = "0"                # "0" = no expiry; or "30d", "1h"

[logging]
level  = "info"                     # debug | info | warn | error
format = "json"                     # json | text
```

Env var override: every key maps to `GOSHORT_<SECTION>_<KEY>` — e.g., `GOSHORT_SERVER_PORT=9090`, `GOSHORT_AUTH_API_KEY=secret`.

---

## API

| Method   | Path                  | Auth | Description           |
|----------|-----------------------|------|-----------------------|
| `POST`   | `/api/v1/urls`        | Yes  | Create short URL      |
| `GET`    | `/api/v1/urls`        | Yes  | List URLs (paginated) |
| `GET`    | `/api/v1/urls/:code`  | Yes  | Get URL details       |
| `DELETE` | `/api/v1/urls/:code`  | Yes  | Delete short URL      |
| `GET`    | `/:code`              | No   | Redirect (302)        |
| `GET`    | `/health`             | No   | Health check          |
| `GET`    | `/metrics`            | No   | Prometheus metrics    |
| `GET`    | `/docs`               | No   | Interactive API docs  |

**Auth:** `X-API-Key: <key>` header. **Redirect codes:** `302 Found`, `404 Not Found`, `410 Gone` (expired).

**POST `/api/v1/urls` body:**

| Field          | Type   | Required | Notes                                      |
|----------------|--------|----------|--------------------------------------------|
| `url`          | string | Yes      | Max 2048 chars                             |
| `custom_alias` | string | No       | `^[a-zA-Z0-9-]{3,30}$`                    |
| `expires_in`   | string | No       | `1h`, `7d`, `30d`, `90d`, `365d`, `never` |

Interactive docs at [http://localhost:8080/docs](http://localhost:8080/docs).

---

## Architecture

```
Request
  │
  ▼
┌─────────────────────────────────────┐
│  Delivery   api/     Chi handlers   │  ← auth, rate-limit, logging, metrics
│             mcp/     MCP tools      │    (Phase 4)
│             cli/     Cobra commands │
└────────────────────┬────────────────┘
                     │
┌────────────────────▼────────────────┐
│  Service    shortener/              │  ← validate URL, generate/decode code,
│                                     │    expiry, business rules
└──────────┬──────────────────────────┘
           │                 │
┌──────────▼──────┐  ┌───────▼───────┐
│  Storage        │  │  Cache        │
│  storage/       │  │  cache/       │  ← none | memory | redis
│  SQLite (sqlc)  │  │  cache-aside  │
└─────────────────┘  └───────────────┘
```

Layer boundaries: `api/` and `mcp/` call `shortener/`; `shortener/` calls `storage/` and `cache/` interfaces only — never concrete types. Full design rationale in [`docs/DESIGN.md`](docs/DESIGN.md).

---

## Development

```bash
lefthook install        # install git hooks (once after clone)
make help               # list all targets
```

| Target           | What it does                          |
|------------------|---------------------------------------|
| `make build`     | Build server + CLI to `bin/`          |
| `make test`      | `go test ./...`                       |
| `make test/race` | Race-detector test run                |
| `make test/cover`| Coverage report → `coverage.html`     |
| `make lint`      | `golangci-lint run`                   |
| `make lint/fix`  | Lint + auto-fix                       |
| `make sqlc`      | Regenerate type-safe Go from SQL      |
| `make docker/up` | `docker compose up -d`                |
| `make clean`     | Remove `bin/` and coverage artifacts  |

**Git hooks (lefthook):** `pre-commit` lints staged files + secrets scan; `pre-push` runs full test suite with race detector; `commit-msg` enforces [Conventional Commits](https://www.conventionalcommits.org).

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow.

---

## Deployment

**Docker Compose** — bundled `docker-compose.yml` includes GoShort + Caddy (auto-TLS). Mount `./data` for SQLite persistence.

**Fly.io** — 30 minutes:
```bash
fly launch && fly volumes create data --size 1 && fly deploy
```

**Bare VPS** — SSH → systemd service → Nginx → Certbot. See [`docs/DESIGN.md` §16](docs/DESIGN.md#16-deployment-strategy) for the step-by-step.

---

## Roadmap

| Phase | Focus | Status |
|-------|-------|--------|
| 1 | Core library — SQLite, sqlc, Sqids, TDD | ✅ v0.1.0 |
| 2 | HTTP API, caching, config, Prometheus | ✅ v0.2.0 |
| 3 | Auth, rate limiting, CLI, Docker, release infra | ✅ v0.3.0 ← current |
| 3.5 | Bare VPS ops — Nginx, systemd, Certbot | 🔲 |
| 4 | MCP server — Claude / Cursor integration | 🔲 |
| 5+ | Analytics, PostgreSQL, Redis counter, AI agent | 🔲 |

Each phase ships a working, deployable product.

---

## License

MIT — see [LICENSE](LICENSE).
