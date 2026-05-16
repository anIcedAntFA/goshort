<h1 align="center">🔗 GoShort</h1>

<p align="center">
  Self-hosted URL shortener — single binary, SQLite-backed, zero config to start.
</p>

<p align="center">
  <sub>Turn long URLs into short, shareable links with click tracking, custom aliases, and AI agent integration.</sub>
</p>

<div align="center">

[![CI](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml/badge.svg)](https://github.com/anIcedAntFA/goshort/actions/workflows/ci.yml)
[![Best Practices](https://www.bestpractices.dev/projects/12838/badge)](https://www.bestpractices.dev/projects/12838)
[![govulncheck](https://github.com/anIcedAntFA/goshort/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/anIcedAntFA/goshort/actions/workflows/govulncheck.yml)
[![codecov](https://codecov.io/gh/anIcedAntFA/goshort/graph/badge.svg)](https://codecov.io/gh/anIcedAntFA/goshort)
[![Go Report Card](https://goreportcard.com/badge/github.com/anIcedAntFA/goshort)](https://goreportcard.com/report/github.com/anIcedAntFA/goshort)
[![Release](https://img.shields.io/github/v/release/anIcedAntFA/goshort)](https://github.com/anIcedAntFA/goshort/releases/latest)

</div>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](go.mod)
[![Website](https://img.shields.io/badge/website-visit-0066CC?logo=cloudflare&logoColor=white)](https://goshort.ngockhoi96.dev)
[![API Docs](https://img.shields.io/badge/API_docs-swagger-FF6C37?logo=openapiinitiative&logoColor=white)](https://goshort.app/docs)
[![Ko-fi](https://img.shields.io/badge/Ko--fi-support-FF5E5B?logo=kofi&logoColor=white)](https://ko-fi.com/anIcedAntFA)

</div>

---

## ✨ Features

- **Zero-collision codes** — atomic SQLite counter + [Sqids](https://sqids.org): non-sequential, bijective, no retry loops
- **Custom aliases** — bring your own slug (`/my-link`); charset isolation prevents collision with generated codes
- **URL expiration** — configurable TTL with lazy expiry on read + hourly background cleanup; update expiry via `PATCH`
- **Batch creation** — `POST /api/v1/urls/batch` accepts up to 100 URLs; per-item success/failure
- **QR codes** — `GET /api/v1/urls/{code}/qr` returns a PNG QR code for any short link
- **Link previews** — `title` and `description` auto-fetched on create (fail-open, private-IP blocked)
- **Spam detection** — optional Google Safe Browsing v4 integration; unsafe URLs rejected with 422 (fail-open)
- **Switchable cache** — `none | memory | redis` at config time; cache-aside with TTL capped to remaining expiry
- **API key auth** — constant-time comparison; per-IP token bucket rate limiting
- **CLI client** — `goshort-cli` for shorten, list, stats, delete from the terminal
- **Public shorten endpoint** — `POST /api/v1/urls/public`: no auth, 5 req/min rate limit, 30-day expiry; powers the landing page widget
- **Landing page** — [goshort.ngockhoi96.dev](https://goshort.ngockhoi96.dev): Astro static site on Cloudflare Workers with live shorten widget
- **MCP server** — AI agents (Claude Code, Cursor) can shorten, list, and manage URLs via [Model Context Protocol](https://modelcontextprotocol.io)
- **Prometheus metrics + structured logs** — `/metrics` endpoint, `slog` throughout, no extra dependencies
- **Self-documenting API** — OpenAPI 3.1 spec + interactive Scalar UI at `/docs`

---

## 🏗️ Architecture

![GoShort Architecture](docs/sys-arch.png)

Full architecture diagrams: [high-level](docs/sys-arch.png), [request flow](docs/request-flow.excalidraw), [layer boundaries](docs/layers.excalidraw).

---

## 🛠️ Tech Stack

| Component      | Technology                                                    |
|----------------|---------------------------------------------------------------|
| Language       | Go 1.26                                                       |
| HTTP           | [Chi](https://go-chi.io) v5                                   |
| Database       | SQLite via [sqlc](https://sqlc.dev) (pure Go, no CGO)         |
| Encoding       | [Sqids](https://sqids.org) (zero-collision, non-sequential)   |
| CLI            | [Cobra](https://cobra.dev)                                    |
| Config         | [Koanf](https://github.com/knadh/koanf) v2 (TOML + env vars) |
| Cache          | [go-redis](https://github.com/redis/go-redis) v9              |
| Metrics        | [Prometheus](https://github.com/prometheus/prometheus)        |
| Rate Limit     | [rate](https://pkg.go.dev/golang.org/x/time/rate) (token bucket) |
| MCP            | [go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.6 (official) |
| Migrations     | [goose](https://github.com/pressly/goose) v3 (embedded FS, versioned SQL) |
| QR codes       | [go-qrcode](https://github.com/skip2/go-qrcode) |
| Spam detection | Google Safe Browsing Lookup API v4 (optional, fail-open) |
| Reverse Proxy  | [Caddy](https://github.com/caddyserver/caddy) (Docker Compose) |
| Release        | [GoReleaser](https://github.com/goreleaser/goreleaser) + [GitHub Actions](https://github.com/features/actions) |
| Website        | [Astro](https://astro.build) v6, Tailwind CSS v4, [Biome](https://biomejs.dev) v2, Bun, Cloudflare Workers |

---

## 🚀 Quick Start

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

## 📡 Usage

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

## 💻 CLI

```bash
# Install
go install github.com/anIcedAntFA/goshort/cmd/cli@latest

# Shorten a URL
goshort-cli shorten https://example.com/long --alias my-link --expires 7d

# List all URLs (paginated)
goshort-cli list --page 1 --per-page 20

# Inspect a short code or alias
goshort-cli stats k7Xm2p

# Delete a short URL
goshort-cli delete k7Xm2p
```

Config file (`~/.goshort.toml`):

```toml
server_url = "http://localhost:8080"
api_key    = "your-api-key"
```

Per-command overrides (precedence: flag > env > config):

| Flag         | Env var              |
|--------------|----------------------|
| `--server`   | `GOSHORT_SERVER_URL` |
| `--api-key`  | `GOSHORT_API_KEY`    |
| `--json`     | —                    |

---

## 🤖 MCP (AI Agent Integration)

GoShort ships an [MCP](https://modelcontextprotocol.io) server so AI agents like Claude Code and Cursor can shorten, list, and manage URLs directly.

### Local — stdio

```bash
make build
```

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "goshort": {
      "command": "./bin/goshort",
      "args": ["--mcp"],
      "env": { "GOSHORT_STORAGE_SQLITE_PATH": "./data/goshort.db" }
    }
  }
}
```

### Remote — Streamable HTTP

The `/mcp` endpoint is served on the main port alongside the REST API. No separate server needed.

```bash
# Connect Claude Code to deployed instance
claude mcp add goshort-remote \
  --transport http \
  https://goshort.app/mcp \
  --header "X-API-Key: your-api-key"
```

### Tools

| Tool | Description |
|------|-------------|
| `shorten_url` | Create a short URL (alias + expiry optional) |
| `batch_shorten_urls` | Create up to 100 short URLs in one call |
| `list_urls` | List URLs with pagination |
| `get_url_stats` | Click count and full details for a URL |
| `delete_url` | Delete a short URL |
| `lookup_url` | Resolve a short code to its original URL |
| `update_url` | Update the expiry of an existing short URL |

### Resources

| URI | Description |
|-----|-------------|
| `goshort://stats/summary` | Total URL count and top URLs by clicks |
| `goshort://urls/{code}` | Full details for a specific short code |
| `goshort://urls/{code}/qr` | PNG QR code blob for a short URL |

### Prompts

| Prompt | Description |
|--------|-------------|
| `shorten_and_share` | Shorten + format for sharing (platform-aware) |
| `batch_shorten` | Shorten multiple URLs and return a table |

---

## ⚙️ Configuration

```toml
[server]
port     = 8080
base_url = "https://short.yourdomain.com"

[auth]
api_key = "your-secret-key"   # empty = no auth

[cache]
driver = "none"               # none | memory | redis

[shortener]
default_expiry = "0"          # "0" = no expiry; or "7d", "30d", "365d"
```

Every key maps to `GOSHORT_<SECTION>_<KEY>` env var — e.g., `GOSHORT_AUTH_API_KEY=secret`. Full reference: [`goshort.toml`](goshort.toml).

---

## 📋 API

| Method   | Path                       | Auth | Description              |
|----------|----------------------------|------|--------------------------|
| `POST`   | `/api/v1/urls`             | Yes  | Create short URL         |
| `POST`   | `/api/v1/urls/batch`       | Yes  | Create up to 100 URLs    |
| `GET`    | `/api/v1/urls`             | Yes  | List URLs (paginated)    |
| `GET`    | `/api/v1/urls/:code`       | Yes  | Get URL details          |
| `PATCH`  | `/api/v1/urls/:code`       | Yes  | Update URL expiry        |
| `DELETE` | `/api/v1/urls/:code`       | Yes  | Delete short URL         |
| `POST`   | `/api/v1/urls/public`      | No   | Public shorten (30d TTL) |
| `GET`    | `/api/v1/urls/:code/qr`    | No   | QR code PNG              |
| `GET`    | `/:code`                   | No   | Redirect (302)           |
| `GET`    | `/health`                  | No   | Health check             |
| `GET`    | `/metrics`                 | No   | Prometheus metrics       |
| `GET`    | `/docs`                    | No   | Interactive API docs     |
| `POST`   | `/mcp`                     | Yes  | MCP Streamable HTTP      |

**Auth:** `X-API-Key: <key>` header. **Redirect codes:** `302 Found`, `404 Not Found`, `410 Gone` (expired).

**POST `/api/v1/urls` body:**

| Field          | Type   | Required | Notes                                      |
|----------------|--------|----------|--------------------------------------------|
| `url`          | string | Yes      | Max 2048 chars                             |
| `custom_alias` | string | No       | `^[a-zA-Z0-9-]{3,30}$`                     |
| `expires_in`   | string | No       | `1h`, `7d`, `30d`, `90d`, `365d`, `never`  |

Interactive docs at [goshort.app/docs](https://goshort.app/docs). Landing page at [goshort.ngockhoi96.dev](https://goshort.ngockhoi96.dev).

---

## 📁 Project Structure

See [`docs/DESIGN.md`](docs/DESIGN.md) for the full architecture, layer boundaries, and design rationale.

---

## 🧑‍💻 Development

```bash
git clone https://github.com/anIcedAntFA/goshort
cd goshort
lefthook install        # install git hooks (once after clone)
make help               # list all targets
```

Run `make help` for the full list of targets. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development workflow including Redis integration tests and git hooks.

---

## 🚢 Deployment

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for comprehensive guides.

**Docker Compose** — GoShort + Caddy (auto-TLS):

```bash
docker compose up -d
```

**Fly.io** — API + redirects + MCP at [goshort.app](https://goshort.app):

```bash
fly launch && fly deploy
```

**Landing page** — [goshort.ngockhoi96.dev](https://goshort.ngockhoi96.dev) (Cloudflare Workers, static Astro build):

```bash
make website/build && cd website && wrangler deploy
```

**Bare VPS** — Nginx + systemd + Certbot (see DEPLOYMENT.md)

---

## 🗺️ Roadmap

| Phase | Focus                                           | Status             |
|-------|-------------------------------------------------|--------------------|
| 1     | Core library — SQLite, sqlc, Sqids, TDD         | ✅ v0.1.0          |
| 2     | HTTP API, caching, config, Prometheus           | ✅ v0.2.0          |
| 3     | Auth, rate limiting, CLI, Docker, release infra | ✅ v0.3.0          |
| 3.5   | Deploy — Fly.io + Cloudflare CDN                | ✅ [goshort.app](https://goshort.app) |
| 4     | MCP server — Claude / Cursor integration        | ✅ v0.4.0          |
| 5     | Batch, QR codes, link previews, spam detection  | ✅ v0.5.0          |
| 6     | Landing page (Cloudflare Workers), public endpoint | ✅ v0.6.0       |
| 7+    | Analytics, PostgreSQL, Redis counter            | 🔲                 |

Each phase ships a working, deployable product.

---

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=anIcedAntFA/goshort&type=Date)](https://star-history.com/#anIcedAntFA/goshort&Date)

---

## 📄 License

MIT — see [LICENSE](LICENSE).
