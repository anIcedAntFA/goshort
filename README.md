# GoShort

[![CI](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml/badge.svg)](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/anIcedAntFA/goshort/graph/badge.svg)](https://codecov.io/gh/anIcedAntFA/goshort)
[![Go Report Card](https://goreportcard.com/badge/github.com/anIcedAntFA/goshort)](https://goreportcard.com/report/github.com/anIcedAntFA/goshort)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A self-hosted URL shortener written in Go. Single binary, SQLite-backed, zero external dependencies required to run.

```
https://docs.google.com/document/d/1a2b3c4d5e6f...  →  https://your.domain/k7Xm2p
```

---

## Features

- **Zero-collision short codes** — Counter + [Sqids](https://sqids.org) encoding: non-sequential, bijective, O(1) generation with no retry loops
- **Custom aliases** — bring your own slug (`/my-link`); charset differentiation prevents collisions with generated codes
- **URL expiration** — configurable TTL with lazy expiry check on read + hourly background cleanup
- **Progressive caching** — three-layer read path: DB index → in-process cache → CDN edge; driver switchable at runtime (`none | memory | redis`)
- **MCP server** — expose shorten/list/stats as tools for Claude and Cursor (Phase 4)
- **Prometheus metrics + structured logs** — `/metrics` endpoint, `slog` throughout
- **Self-documenting API** — OpenAPI spec served as interactive Scalar UI at `/docs`
- **Single binary** — `./goshort serve` and done; SQLite embedded via pure-Go driver (no CGO)

---

## Quick Start

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/docker-compose.yml
docker compose up -d
```

GoShort listens on port `8080`. Caddy handles TLS automatically.

### Binary

```bash
go install github.com/anIcedAntFA/goshort/cmd/server@latest
goshort serve --config goshort.toml
```

### Shorten a URL

```bash
curl -X POST http://localhost:8080/api/v1/urls \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"url": "https://example.com/very/long/path", "expires_in": "30d"}'
```

```json
{
  "short_code": "k7Xm2p",
  "short_url": "http://localhost:8080/k7Xm2p",
  "original_url": "https://example.com/very/long/path",
  "expires_at": "2025-08-01T00:00:00Z",
  "created_at": "2025-07-01T12:00:00Z"
}
```

---

## Configuration

Copy `goshort.toml` and adjust:

```toml
[server]
port     = 8080
base_url = "https://short.yourdomain.com"

[storage]
driver      = "sqlite"              # sqlite | postgres
sqlite_path = "./data/goshort.db"

[cache]
driver    = "none"                  # none | memory | redis
redis_url = "redis://localhost:6379"

[auth]
api_key = "sk_live_changeme"

[rate_limit]
enabled             = false
requests_per_minute = 60

[shortener]
code_length    = 6
default_expiry = "0"                # "0" = no expiry

[logging]
level  = "info"
format = "json"
```

All values can be overridden via environment variables: `GOSHORT_SERVER_PORT`, `GOSHORT_AUTH_API_KEY`, etc.

---

## API

| Method   | Endpoint              | Auth | Description            |
|----------|-----------------------|------|------------------------|
| `POST`   | `/api/v1/urls`        | Yes  | Create short URL       |
| `GET`    | `/api/v1/urls`        | Yes  | List URLs (paginated)  |
| `GET`    | `/api/v1/urls/:code`  | Yes  | Get URL details        |
| `DELETE` | `/api/v1/urls/:code`  | Yes  | Delete short URL       |
| `GET`    | `/:code`              | No   | Redirect to original   |
| `GET`    | `/health`             | No   | Health check           |
| `GET`    | `/metrics`            | No   | Prometheus metrics     |
| `GET`    | `/docs`               | No   | Interactive API docs   |

**Authentication:** `X-API-Key: <key>` header on all write endpoints.

**Redirect codes:** `302 Found` (always), `404 Not Found`, `410 Gone` (expired).

**Error shape:**

```json
{
  "error": {
    "code": "alias_taken",
    "message": "The alias 'my-link' is already in use"
  }
}
```

**Request body (`POST /api/v1/urls`):**

| Field          | Type   | Required | Description                                |
|----------------|--------|----------|--------------------------------------------|
| `url`          | string | Yes      | Original URL (max 2048 chars)              |
| `custom_alias` | string | No       | Custom slug, `^[a-zA-Z0-9-]{3,30}$`       |
| `expires_in`   | string | No       | `1h`, `7d`, `30d`, `90d`, `365d`, `never`  |

---

## Architecture

Three-layer pragmatic structure:

```
Delivery     api/       HTTP handlers (Chi), middleware (auth, rate limit, logging, metrics)
             mcp/       MCP server tools (Phase 4)
             cli/       Cobra commands

Service      shortener/ Business logic: validate URL, generate/validate code, manage expiry

Storage      storage/   Interface + SQLite impl (sqlc-generated, pure-Go driver)
             cache/     Interface + noop / sync.Map / Redis implementations
```

**Short code generation:** an atomic SQLite counter feeds Sqids, producing shuffled non-sequential codes. Generated codes use `[a-zA-Z0-9]`; custom aliases allow hyphens — the charset difference makes collision between the two impossible by construction.

**Read path:** cache lookup → DB query → expiry check → 302 redirect. Cache TTL is capped to `min(24h, remaining_expiry)` so expired URLs are never served from cache.

**Redirect strategy:** always `302` (not `301`) — enables click counting, expiration enforcement, and destination updates. Browser and CDN caching controlled explicitly via `Cache-Control` headers.

---

## Development

```bash
# Install git hooks (run once after cloning)
lefthook install

make help          # list all targets

make build         # build server + CLI binaries
make test          # go test ./...
make test/race     # go test -race ./...
make test/cover    # test + open HTML coverage report
make lint          # golangci-lint run ./...
make lint/fix      # golangci-lint run --fix ./...
make sqlc          # regenerate type-safe Go from db/queries.sql
make tidy          # go mod tidy
make docker/up     # docker compose up -d
make docker/down   # docker compose down
make clean         # remove binaries and coverage.out
```

**Git hooks (via lefthook):**
- `pre-commit` — fast lint on staged Go files; auto-stages fixes; secrets scan via gitleaks
- `pre-push` — full test suite with race detector + `go vet`
- `commit-msg` — enforces [Conventional Commits](https://www.conventionalcommits.org) format

**Commit format:** `type(scope): description` — e.g., `feat: add URL expiration`, `fix(cache): align TTL to expiry`.

---

## Deployment

**Fly.io (recommended — 30 minutes):**

```bash
fly launch
fly volumes create data --size 1
fly deploy
```

**Bare VPS (learning path — 1 week):** SSH → systemd service → Nginx reverse proxy → Certbot TLS. See [`docs/DESIGN.md` §16](docs/DESIGN.md#16--deployment-strategy).

**Self-host with Docker Compose:** the included `docker-compose.yml` bundles GoShort + Caddy (auto-TLS). Mount `./data` for SQLite persistence.

---

## Roadmap

| Phase | Focus | Deliverable |
|-------|-------|-------------|
| **1** | Core: SQLite, sqlc, Sqids, TDD | `go test ./...` passes |
| **2** | HTTP API, Chi, caching, slog, Prometheus | `curl` works |
| **3** | Auth, CLI, rate limiting, Docker, Fly.io | v1.0 — public release |
| **3.5** | Bare VPS, Nginx, systemd, Certbot | Infrastructure depth |
| **4** | MCP server, Claude/Cursor integration | AI can shorten URLs |
| **5+** | Analytics, PostgreSQL, Redis, AI agent | v2.0 |

Each phase ships a working, deployable product.

---

## License

MIT — see [LICENSE](LICENSE).
