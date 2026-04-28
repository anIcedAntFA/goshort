# 🔗 GoShort — System Design Document (Final)

> **TL;DR:** A self-hosted URL shortener in Go. Personal tool → opensource. SQLite + Chi + sqlc + slog. Progressive scaling layers (Index → Redis → CDN) as learning exercises. Auth + rate limiting before MCP. Deploy to Fly.io first (ship), then redo on bare VPS (learn). Target: 10-1000 users, each self-hosting their own instance.

---

## 📖 Table of Contents

1. [Problem Statement](#1--problem-statement)
2. [Decisions Log](#2--decisions-log)
3. [Requirements](#3--requirements)
4. [Back-of-the-Envelope Estimation](#4--back-of-the-envelope-estimation)
5. [Core Entities & Data Model](#5--core-entities--data-model)
6. [API Design](#6--api-design)
7. [Short Code Generation — Deep Dive](#7--short-code-generation--deep-dive)
8. [Architecture & Project Structure](#8--architecture--project-structure)
9. [High-Level System Architecture](#9--high-level-system-architecture)
10. [Scaling Reads — Progressive Layers](#10--scaling-reads--progressive-layers)
11. [Redirect Strategy — 301 vs 302](#11--redirect-strategy--301-vs-302)
12. [Expiration & Cleanup](#12--expiration--cleanup)
13. [Error Handling & Edge Cases](#13--error-handling--edge-cases)
14. [Monitoring & Observability](#14--monitoring--observability)
15. [API Documentation & Testing](#15--api-documentation--testing)
16. [Deployment Strategy](#16--deployment-strategy)
17. [Phase Roadmap](#17--phase-roadmap)
18. [Future: Scaling Writes & DB Replication](#18--future-scaling-writes--db-replication)

---

## 1. 🎯 Problem Statement

### What are we solving?

Users have long, ugly URLs. They need short, shareable links that redirect to the original.

```
INPUT:  https://docs.google.com/document/d/1a2b3c4d5e6f...&ouid=123456789
OUTPUT: https://goshort.dev/k7Xm2p
```

### Why build this?

**Two goals, one project:**

1. **Learn** — Go, system design, caching, networking, MCP, AI agents. This project touches nearly every backend concept: hashing, indexing, caching, rate limiting, middleware, protocol design, deployment, infrastructure.

2. **Use** — A real personal tool. Self-hosted, opensource, no vendor lock-in. Users fork the repo, deploy their own instance, own their data.

> 💡 **ELI5:** Imagine a coat check at a restaurant. You hand over your bulky coat (long URL), get a small numbered ticket (short code), and when you come back with the ticket, you get your coat back. Our job is to design the coat check system.

### Product Vision

```
Phase 1-4:  Personal tool → share with friends → opensource
Phase 5+:   Optional hosted service (free tier + paid for heavy use)
            Self-host always remains the primary model
```

---

## 2. 📌 Decisions Log

> Record of key architectural decisions and rationale.

| # | Decision | Options Considered | Choice | Rationale |
|---|----------|-------------------|--------|-----------|
| D1 | Tech stack | Bun+Cloudflare, Go+infra, Mix | **Go full stack** | Max learning; CLI+Server+MCP same language; single binary for opensource |
| D2 | Architecture model | Multi-tenant, Self-host, Both | **Self-host first** | Simpler auth; users own data; opensource-friendly |
| D3 | Auth timing | Before MCP, After MCP, Same phase | **Before MCP** | API must be protected before exposing tools to agents |
| D4 | Code architecture | Clean, Hexagonal, Layered | **Pragmatic layered** | GoShort is CRUD; interfaces for swapping, no ceremony overhead |
| D5 | Database | Postgres, SQLite, MySQL | **SQLite** (Phase 1) → **Postgres** (Phase 5+) | Zero ops; embedded; `modernc.org/sqlite` pure Go; interface allows swap |
| D6 | DB access | database/sql, sqlx, GORM, Bun ORM | **sqlc** | Write SQL → generate type-safe Go code; compile-time safety; raw SQL performance |
| D7 | Cache | Redis, Memcached, In-process | **In-process → Redis** | Config flag: none/memory/redis; Redis also serves counter + rate limiter |
| D8 | Encoding | Random, Hash, Base62, Sqids | **Counter + Sqids** | Zero collision; non-predictable; O(1); bijective |
| D9 | HTTP router | stdlib, Chi, Echo, Gin | **Chi** | stdlib-compatible; route groups; zero lock-in; ~1000 LOC |
| D10 | Logging | slog, Zap, Zerolog | **slog (stdlib)** | Zero deps; swappable backend; good enough performance |
| D11 | Deploy (ship) | Fly.io, Railway, VPS | **Fly.io** | SQLite-friendly; persistent volumes; single command deploy |
| D12 | Deploy (learn) | Same Fly.io, Bare VPS | **Bare VPS (Phase 3.5)** | Learn SSH, Nginx, Certbot, systemd, DNS, firewall |
| D13 | Reverse proxy | Nginx, Caddy, Traefik | **Caddy (Docker Compose), Nginx (VPS learning)** | Caddy for self-host ease; Nginx for learning depth |
| D14 | CDN | Cloudflare, CloudFront, None | **Cloudflare (free tier)** | Already have account; DNS + CDN + DDoS protection |
| D15 | Scaling reads | Skip, Some layers, All 3 | **All 3 layers** | Learning exercise — Index → Redis → CDN |
| D16 | Scaling writes | Now, Later | **Later (Phase 5+)** | 1000 users ≈ <1 write/sec; trivial |
| D17 | DB replication | Now, Later, Never | **Later (Phase 5+)** | Learning exercise only; no practical need at this scale |
| D18 | MCP runtime | Cloudflare Workers, Go | **Go (mcp-go)** | Same language; `github.com/mark3labs/mcp-go` production-ready |
| D19 | Containers | Docker only, Compose, Swarm, K8s | **Docker + Compose** | K8s/Swarm = separate learning project |
| D20 | API docs | swaggo, Huma, Manual OpenAPI | **OpenAPI YAML + Scalar UI** | Learn OpenAPI standard; 5 endpoints = manageable; serve at /docs |
| D21 | API testing | Postman, Bruno, Hoppscotch, curl | **Bruno + curl + Go httptest** | Bruno = Git-native; httptest = automated TDD; curl = quick |
| D22 | Testing approach | Test after, TDD | **TDD + table-driven + fuzz** | Write failing tests first; table-driven for multiple cases; fuzz for validators |

---

## 3. 📋 Requirements

### Functional Requirements

| Priority | Requirement | Phase | Description |
|----------|------------|-------|-------------|
| **P0** | Shorten URL | 1 | Accept a long URL → return a short URL |
| **P0** | Redirect | 1 | Access short URL → redirect to original |
| **P0** | Custom alias | 1 | User specifies their own short code (e.g., `/my-link`) |
| **P0** | Expiration | 1 | Short URL auto-expires after a duration |
| **P1** | Delete URL | 2 | Delete a short URL via API |
| **P1** | List URLs | 2 | Paginated list of created URLs |
| **P1** | API key auth | 3 | Protect write endpoints with API keys |
| **P1** | Rate limiting | 3 | Token bucket on all endpoints |
| **P2** | MCP tools | 4 | Expose shorten/list/stats as MCP tools |
| **P3** | Analytics | 5 | Click counts, referrer, geo (basic) |
| **P3** | Spam detection | 5 | Block known malicious URLs |
| **P3** | AI Agent | 5 | Smart slug generation, bulk categorize |

### Non-Functional Requirements

| Priority | Requirement | Target |
|----------|------------|--------|
| **P0** | Redirect latency | < 50ms P99 (cached) |
| **P0** | Uniqueness | Zero collisions (guaranteed by counter + Sqids) |
| **P0** | Self-hostable | Single binary + SQLite, `./goshort serve` and done |
| **P1** | Availability | 99.9% for redirects |
| **P1** | Durability | No data loss — SQLite WAL mode + backups |
| **P2** | Observability | slog structured logs + Prometheus metrics |

### Explicit Non-Goals (Phase 1-4)

- ❌ Multi-tenant (shared hosting for multiple users)
- ❌ User registration/signup flow
- ❌ Real-time analytics dashboards
- ❌ Database replication
- ❌ Horizontal scaling of writes
- ❌ Kubernetes / Docker Swarm

---

## 4. 🧮 Back-of-the-Envelope Estimation

### Our Actual Scale (self-hosted)

```
Per instance: 1 user (or small team)
  URLs created:  ~10-50/day
  Redirects:     ~100-500/day
  Storage:       ~1,000-10,000 URLs total

Writes: < 1/sec     → SQLite handles trivially
Reads:  < 1/sec     → SQLite handles trivially
Storage: < 5 MB     → SQLite file, trivial
```

> 🔑 **At this scale, everything is trivial.** Redis and CDN are learning exercises, not necessities.

### System Design Scale (learning: bit.ly-like)

```
Read:Write ratio ≈ 100:1

Reads:  100M redirects/day  = ~1,150/sec avg, ~5,000/sec peak
Writes: 1M new URLs/day     = ~12/sec avg
```

### Short Code Space

```
Base62, length 6: 62^6 = 56.8 billion combinations
For 10K URLs:  5.68 million × headroom → length 6 is more than sufficient
```

### Summary

| Metric | Our Scale | Learning Scale |
|--------|----------|---------------|
| Write QPS | < 1/sec | ~12/sec |
| Read QPS | < 1/sec | ~5,000/sec peak |
| Storage | ~5 MB | ~500 GB |
| Database | SQLite ✅ | PostgreSQL |
| Cache needed? | No | Yes (Redis) |
| CDN needed? | No | Yes |

---

## 5. 🏗️ Core Entities & Data Model

### Schema (SQLite — managed by sqlc)

```sql
-- schema.sql (input for sqlc)
CREATE TABLE urls (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code   TEXT    UNIQUE NOT NULL,
    original_url TEXT    NOT NULL,
    is_custom    INTEGER DEFAULT 0,
    created_at   TEXT    DEFAULT (datetime('now')),
    expires_at   TEXT,
    click_count  INTEGER DEFAULT 0
);

CREATE TABLE counter (
    id    INTEGER PRIMARY KEY CHECK (id = 1),
    value INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_short_code ON urls(short_code);
CREATE INDEX idx_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
```

### sqlc Queries

```sql
-- queries.sql (input for sqlc → generates type-safe Go code)

-- name: CreateURL :one
INSERT INTO urls (short_code, original_url, is_custom, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetByCode :one
SELECT * FROM urls WHERE short_code = ?;

-- name: DeleteByCode :exec
DELETE FROM urls WHERE short_code = ?;

-- name: ListURLs :many
SELECT * FROM urls ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountURLs :one
SELECT COUNT(*) FROM urls;

-- name: IncrementClicks :exec
UPDATE urls SET click_count = click_count + 1 WHERE short_code = ?;

-- name: DeleteExpired :execrows
DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at < datetime('now')
LIMIT ?;

-- name: IncrementCounter :one
UPDATE counter SET value = value + 1 RETURNING value;
```

### Design Decisions

- **Auto-increment `id` + indexed `short_code`:** `id` feeds Sqids encoding. If encoding changes, existing codes stay valid.
- **Separate `counter` table:** Decouples counter from URL table, portable to Redis later.
- **`is_custom` flag:** Custom aliases and generated codes share the table but differ in validation rules.
- **`click_count`:** Near-zero cost atomic counter. Even without dashboards, "was this link clicked" is useful.

---

## 6. 🌐 API Design

### Endpoints

```
┌──────────┬─────────────────────┬────────────────────────┬─────────┐
│  Method  │  Endpoint           │  Description           │  Auth?  │
├──────────┼─────────────────────┼────────────────────────┼─────────┤
│  GET     │  /:code             │  Redirect to original  │  No     │
│  POST    │  /api/v1/urls       │  Create short URL      │  Yes*   │
│  GET     │  /api/v1/urls       │  List URLs (paginated) │  Yes*   │
│  GET     │  /api/v1/urls/:code │  Get URL details       │  Yes*   │
│  DELETE  │  /api/v1/urls/:code │  Delete a short URL    │  Yes*   │
│  GET     │  /health            │  Health check          │  No     │
│  GET     │  /metrics           │  Prometheus metrics    │  No**   │
│  GET     │  /docs              │  Scalar API docs       │  No     │
└──────────┴─────────────────────┴────────────────────────┴─────────┘
*  Auth added in Phase 3      ** Bind to localhost or internal network
```

### Reserved Paths

Short codes cannot match: `/api/*`, `/health`, `/metrics`, `/docs`, `/_/*`. Validate on creation.

### POST /api/v1/urls

```http
POST /api/v1/urls
Content-Type: application/json
X-API-Key: sk_live_abc123

{
  "url": "https://example.com/very/long/path?with=params",
  "custom_alias": "my-link",
  "expires_in": "30d"
}
```

**201 Created:**
```json
{
  "short_code": "my-link",
  "short_url": "https://goshort.dev/my-link",
  "original_url": "https://example.com/very/long/path?with=params",
  "expires_at": "2025-07-01T00:00:00Z",
  "created_at": "2025-06-01T12:00:00Z"
}
```

### Error Response Format

```json
{
  "error": {
    "code": "alias_taken",
    "message": "The alias 'my-link' is already in use",
    "details": { "alias": "my-link" }
  }
}
```

| Status | Code | When |
|--------|------|------|
| `400` | `invalid_url` | URL format invalid |
| `400` | `invalid_expires` | Bad duration format |
| `409` | `alias_taken` | Custom alias exists |
| `422` | `invalid_alias` | Alias fails validation |
| `422` | `reserved_path` | Alias matches reserved path |
| `429` | `rate_limited` | Too many requests (Phase 3+) |

### GET /:code — Redirect

```
302 Found → Location: https://example.com/...
404 Not Found → short code not found
410 Gone → URL has expired
```

### GET /api/v1/urls — List

```json
{
  "data": [...],
  "pagination": { "page": 1, "per_page": 20, "total": 156, "total_pages": 8 }
}
```

---

## 7. 🔑 Short Code Generation — Deep Dive

### Bad → Good → Great

**Random String ❌:** Birthday Paradox — 50% collision chance at ~280K URLs. Retry loops under concurrency = thundering herd.

> 💡 **ELI5 Birthday Paradox:** 23 people in a room, 50% chance two share a birthday. Same math applies to random codes — collisions happen far sooner than intuition suggests.

**Hash (MD5+truncate) 🟡:** Deterministic but still collisions from truncation. Can't have 2 short codes for same URL with different expiry.

**Counter + Base62 ✅:** Zero collisions, but predictable (`/4c92` → try `/4c93`). Enumeration attack risk.

**Counter + Sqids 🌟 (our choice):**

```
Plain Base62:  1000 → "g8"    1001 → "g9"    ← Sequential!
Sqids:         1000 → "k7Xm"  1001 → "Qp3n"  ← Shuffled!
```

Bijective (reversible), non-sequential, O(1), zero collisions. Library: `github.com/sqids/sqids-go`.

### Counter Source

Phase 1-4: Separate `counter` table in SQLite, atomic increment via `UPDATE ... RETURNING`.
Phase 5+: Redis `INCR` for distributed counter.

### Custom Alias Collision Prevention

**Charset differentiation:** Sqids output = `[a-zA-Z0-9]` only. Custom aliases allow hyphens `-`. If a code contains a hyphen → custom. No ambiguity.

Validation: `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`, length 3-30.

### Decision Matrix

```
                    Uniqueness  Predictable?  Complexity  Our Choice
                    ──────────  ────────────  ──────────  ──────────
Random String       ❌ Poor      ✅ No          Low        ❌
Hash (MD5+trunc)    🟡 Okay      ✅ No          Medium     ❌
Counter + Base62    ✅ Perfect    ❌ Yes         Low        ❌
Counter + Sqids     ✅ Perfect    ✅ No          Low        ✅ Winner
```

---

## 8. 🏛️ Architecture & Project Structure

### Pragmatic Layered Architecture

```
┌─────────────────────────────────────────────────┐
│  Delivery Layer (api/, mcp/, cli/)              │
│  HTTP handlers, MCP tools, CLI commands          │
│  Depends on: Service interfaces                  │
├─────────────────────────────────────────────────┤
│  Service Layer (shortener/)                      │
│  Business logic, validation, encoding            │
│  Depends on: Storage + Cache interfaces          │
├─────────────────────────────────────────────────┤
│  Infrastructure Layer (storage/, cache/)         │
│  SQLite (via sqlc), Postgres, Redis, in-memory   │
│  Implements: Storage + Cache interfaces          │
└─────────────────────────────────────────────────┘
```

### Project Structure

```
goshort/
├── cmd/
│   ├── server/main.go
│   └── cli/main.go
├── internal/
│   ├── shortener/          # Core business logic
│   │   ├── service.go
│   │   ├── encoder.go
│   │   ├── validator.go
│   │   └── service_test.go
│   ├── storage/            # Storage abstraction
│   │   ├── storage.go      # Interface
│   │   ├── sqlite.go       # SQLite via sqlc-generated code
│   │   └── sqlite_test.go
│   ├── cache/              # Cache abstraction
│   │   ├── cache.go        # Interface
│   │   ├── noop.go         # No-op (cache disabled)
│   │   ├── memory.go       # In-process (sync.Map)
│   │   └── redis.go        # Redis
│   ├── api/                # HTTP layer (Chi)
│   │   ├── router.go
│   │   ├── handler.go
│   │   ├── middleware.go
│   │   ├── errors.go
│   │   └── handler_test.go
│   ├── mcp/                # MCP server (Phase 4)
│   │   ├── server.go
│   │   └── tools.go
│   └── config/
│       └── config.go
├── db/                     # sqlc input files
│   ├── schema.sql
│   ├── queries.sql
│   └── sqlc.yaml
├── docs/
│   ├── design.md           # This document
│   ├── openapi.yaml        # API specification
│   └── cc-workflow-guide.md
├── api-tests/              # Bruno .bru files (Git-native)
│   ├── create-url.bru
│   ├── redirect.bru
│   └── environments/
│       ├── local.bru
│       └── production.bru
├── data/                   # SQLite database (gitignored)
├── docker-compose.yml
├── Dockerfile
├── Caddyfile
├── Makefile
├── sqlc.yaml
├── go.mod
├── go.sum
├── CLAUDE.md
└── README.md
```

### Key Go Libraries

| Library | Purpose | Phase |
|---------|---------|-------|
| `github.com/go-chi/chi/v5` | HTTP router (stdlib-compatible) | 2 |
| `github.com/sqids/sqids-go` | Short code encoding | 1 |
| `modernc.org/sqlite` | SQLite (pure Go, no CGO) | 1 |
| `github.com/sqlc-dev/sqlc` | Generate type-safe Go from SQL | 1 |
| `github.com/redis/go-redis/v9` | Redis client | 2 |
| `github.com/prometheus/client_golang` | Prometheus metrics | 2 |
| `github.com/spf13/cobra` | CLI framework | 3 |
| `github.com/spf13/viper` | Configuration | 3 |
| `github.com/mark3labs/mcp-go` | MCP server | 4 |
| `golang.org/x/time/rate` | Rate limiting | 3 |
| `log/slog` | Structured logging (stdlib) | 2 |

---

## 9. 🏗️ High-Level System Architecture

### What We Build (Phase 1-4)

```
                          ┌─────────────────────────────────────┐
                          │          GoShort Instance            │
                          │         (self-hosted by user)        │
                          │                                     │
┌──────────┐    HTTP      │  ┌───────────┐    ┌─────────────┐  │
│ Browser  │ ──────────▶  │  │  Chi      │    │             │  │
│          │ ◀──────────  │  │  Router   │───▶│   SQLite    │  │
└──────────┘   302        │  │           │    │ (via sqlc)  │  │
                          │  │ middleware │    └─────────────┘  │
┌──────────┐    HTTP      │  │ • auth    │                     │
│ CLI      │ ──────────▶  │  │ • rate    │    ┌─────────────┐  │
│(goshort) │ ◀──────────  │  │ • log     │    │   Sqids     │  │
└──────────┘              │  │ • metrics │    │  (encoder)  │  │
                          │  │           │    └─────────────┘  │
┌──────────┐   stdio      │  │           │                     │
│ Claude / │ ──────────▶  │  │ MCP Server│    ┌─────────────┐  │
│ Cursor   │ ◀──────────  │  │ (mcp-go)  │    │ Cache (opt) │  │
└──────────┘              │  └───────────┘    │ noop/mem/   │  │
                          │                   │ redis       │  │
                          └───────────────────┴─────────────┘  │
                                      │
                               Single binary
```

### What We Learn (System Design Scale)

```
                    ┌─────────────────┐
                    │  Cloudflare CDN │
                    │   (Layer 3)     │
                    └────────┬────────┘
                             │ cache miss
                    ┌────────▼────────┐
                    │  Redis (Layer 2)│
                    │  Cache-Aside    │
                    └────────┬────────┘
                             │ cache miss
                    ┌────────▼────────┐
                    │  PostgreSQL     │
                    │  + Index (L1)   │
                    └─────────────────┘
```

---

## 10. ⚡ Scaling Reads — Progressive Layers

> 🎓 **Learning goal:** Understand WHY each layer exists and WHEN to add it. We implement all 3 even though our scale doesn't need them.

### Layer 1: Database Index ✅ Always On

```sql
CREATE UNIQUE INDEX idx_short_code ON urls(short_code);
```

Without index: O(n) full table scan. With B-tree index: O(log n), <1ms even at 1B rows. Non-negotiable.

### Layer 2: In-Memory Cache ✅ Optional

**Pattern: Cache-Aside (Lazy Loading)**

```
┌─────────┐  1. GET short:k7Xm2p  ┌─────────┐
│         │ ─────────────────────▶ │         │
│ Server  │                        │  Cache  │
│         │ ◀───────────────────── │         │
│         │  2a. HIT → return      └─────────┘
│         │
│         │  2b. MISS:
│         │  3. Query DB           ┌─────────┐
│         │ ─────────────────────▶ │   DB    │
│         │ ◀───────────────────── │         │
│         │  4. Populate cache     └─────────┘
└─────────┘
```

**Why Cache-Aside over Write-Through?** URL access follows power-law: ~20% of URLs get ~80% of clicks. Why cache the other 80%?

**Cache invalidation:** TTL-based + explicit delete on mutation. Simple by design.

**Implementation progression:**
- Phase 1-2: `cache/noop.go` or `cache/memory.go`
- Phase 2+: `cache/redis.go` (learning)
- Config: `cache.driver = "none" | "memory" | "redis"`

### Layer 3: CDN Edge Cache ✅ Optional

Cloudflare free tier (already have account). Domain DNS through Cloudflare proxy mode.

```go
w.Header().Set("Cache-Control", "public, max-age=300")
w.Header().Set("CDN-Cache-Control", "max-age=3600")
```

**Trade-off:** CDN caching = lose instant deletion (stale for up to TTL). Use Cloudflare Cache-Tag + API purge on DELETE.

### When to Use Each Layer

```
Layer     Cost          Latency    When Needed         Our Phase
─────     ────          ───────    ───────────         ─────────
L1 Index  Free          ~1ms      Always              Phase 1
L2 Cache  Redis/$0 mem  ~0.5ms    > 100 req/sec       Phase 2 (learning)
L3 CDN    $0 (CF free)  ~10ms     Global users        Phase 2 (learning)
```

---

## 11. 🔀 Redirect Strategy — 301 vs 302

### The Problem

When a user accesses a short URL, the server must send an HTTP redirect response. The choice of status code has significant implications.

### Two Types of Redirects

**301 (Permanent Redirect):** Browser caches the redirect. Next time the user visits the same short URL, the browser goes directly to the long URL — **bypassing our server entirely.**

```http
HTTP/1.1 301 Moved Permanently
Location: https://www.original-long-url.com
```

**302 (Temporary Redirect / Found):** Browser does NOT cache. Every visit goes through our server, giving us full control.

```http
HTTP/1.1 302 Found
Location: https://www.original-long-url.com
```

### Trade-off Analysis

```
┌────────────────────────┬──────────────────────────────┐
│   301 Permanent        │    302 Temporary              │
├────────────────────────┼──────────────────────────────┤
│ Browser caches redirect│ Browser always asks server    │
│                        │                              │
│ ✅ Faster for repeat    │ ✅ We see every click          │
│   visits (no roundtrip)│ ✅ Can update destination      │
│                        │ ✅ Can expire/delete URLs      │
│ ❌ Can't track clicks   │ ✅ Analytics possible          │
│ ❌ Can't expire links   │ ✅ Rate limiting works         │
│ ❌ Can't update dest    │                              │
│ ❌ User stuck with old  │ 🟡 Slightly slower             │
│   destination forever  │   (one extra roundtrip)      │
└────────────────────────┴──────────────────────────────┘
```

### Why 302 Wins for GoShort

1. **Expiration support:** We support `expires_at`. If a URL expires, we need to return `410 Gone` — impossible if browser cached a 301 and never contacts our server again.

2. **Click counting:** Even our simple `click_count` field requires seeing every request. With 301, repeat visits are invisible to us.

3. **URL mutability:** If we ever add "update destination" (PUT endpoint), 302 allows it. 301 locks the destination in the browser cache — users would need to clear their browser cache to see changes.

4. **Rate limiting & abuse prevention:** 301 bypasses our server entirely on repeat visits — rate limiting, blocklists, and any server-side logic become useless for those requests.

5. **Performance cost is negligible:** The "extra roundtrip" of 302 is one HTTP request (~5-50ms). With Redis cache (<1ms lookup) and CDN edge caching, the total latency is negligible.

### Cache-Control Headers

Even with 302, we set explicit headers to prevent unwanted caching:

```go
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
    // ... lookup URL ...
    w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
    w.Header().Set("Location", url.OriginalURL)
    w.WriteHeader(http.StatusFound) // 302
}
```

For CDN layer (opt-in), we use separate CDN-specific headers:
```go
// Only when CDN caching is enabled via config
if config.CDN.Enabled {
    w.Header().Set("Cache-Control", "public, max-age=300")       // browser: 5 min
    w.Header().Set("CDN-Cache-Control", "max-age=3600")          // CDN edge: 1 hour
}
```

> 💡 **Advanced (future):** Could support both — `301` for URLs explicitly marked as permanent by the user, `302` for everything else. Over-engineering for now.

---

## 12. 🧹 Expiration & Cleanup

### How Expiration Works

URLs can optionally have an `expires_at` timestamp. When a user accesses an expired URL, we return `410 Gone` instead of redirecting.

### Read Path (Lazy Check)

Every redirect request checks expiration before redirecting:

```go
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
    code := chi.URLParam(r, "code")
    
    // 1. Check cache first (if enabled)
    if cached, ok := h.cache.Get(ctx, "short:"+code); ok {
        h.incrementClicks(code)
        http.Redirect(w, r, cached, http.StatusFound)
        return
    }
    
    // 2. Cache miss → query DB
    url, err := h.store.GetByCode(ctx, code)
    if err != nil {
        h.respondError(w, http.StatusNotFound, "not_found", "Short URL not found")
        return
    }
    
    // 3. Check if expired
    if url.ExpiresAt.Valid && url.ExpiresAt.Time.Before(time.Now()) {
        // Don't cache expired URLs
        h.respondError(w, http.StatusGone, "expired", "This short URL has expired")
        return
    }
    
    // 4. Cache the result for next time
    h.cache.Set(ctx, "short:"+code, url.OriginalURL, h.cacheTTL(url))
    
    // 5. Increment click count (fire-and-forget)
    go h.incrementClicks(code)
    
    // 6. Redirect
    w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
    http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}
```

### Cache TTL Alignment

When caching a URL that has an expiration, the cache TTL must be ≤ remaining time. Otherwise, we serve an expired URL from cache:

```go
func (h *Handler) cacheTTL(url *URL) time.Duration {
    defaultTTL := 24 * time.Hour
    
    if !url.ExpiresAt.Valid {
        return defaultTTL  // no expiry → cache for 24h
    }
    
    remaining := time.Until(url.ExpiresAt.Time)
    if remaining <= 0 {
        return 0  // already expired → don't cache
    }
    if remaining > defaultTTL {
        return defaultTTL  // cap at 24h
    }
    return remaining  // cache exactly until expiry
}
```

### Background Cleanup (Goroutine)

Expired URLs still occupy storage even after lazy check stops serving them. A background goroutine cleans them up periodically:

```go
func (s *Server) startCleanupJob(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            slog.Info("cleanup job stopped")
            return
        case <-ticker.C:
            deleted, err := s.store.DeleteExpired(ctx, 1000)  // batch of 1000
            if err != nil {
                slog.Error("cleanup failed", "error", err)
                continue
            }
            if deleted > 0 {
                slog.Info("cleanup completed", "deleted", deleted)
            }
        }
    }
}
```

### Strategy: Both Lazy + Background

| Strategy | How | Pros | Cons |
|----------|-----|------|------|
| **Lazy deletion** | Check on read, return 410 if expired | ✅ Immediate correctness | ❌ Dead rows accumulate |
| **Background job** | Goroutine deletes expired rows periodically | ✅ Clean storage | ❌ Expired URL served briefly between expiry and cleanup |
| **Both** | Lazy on read + periodic cleanup | ✅ Best of both | Minimal extra complexity |

→ **Choice: Both.** Lazy deletion handles the read path immediately. Background cleanup is storage hygiene.

### Cache Invalidation on Deletion

When a URL is explicitly deleted via API (not just expired):

```go
func (h *Handler) DeleteURL(w http.ResponseWriter, r *http.Request) {
    code := chi.URLParam(r, "code")
    
    // 1. Delete from DB
    err := h.store.DeleteByCode(ctx, code)
    if err != nil { ... }
    
    // 2. Explicitly invalidate cache
    h.cache.Delete(ctx, "short:"+code)
    
    w.WriteHeader(http.StatusNoContent)
}
```

### Summary

```
Read Path:
  GET /:code → check cache → cache miss? → query DB → expired? → 410 Gone
                                                      → valid?   → 302 Redirect + cache

Background:
  Every 1 hour → DELETE FROM urls WHERE expires_at < NOW() LIMIT 1000

Cache:
  TTL aligned to expiry → expired URLs never served from cache
  Explicit DEL on API delete → no stale cache entries
```

---

## 13. 🛡️ Error Handling & Edge Cases

| Edge Case | How We Handle |
|-----------|---------------|
| Same URL shortened twice | Different short codes (by design) |
| Custom alias = reserved path | Validate against list, return 422 |
| Custom alias = generated code | Impossible — charset differentiation (hyphens) |
| Very long URL (>2048 chars) | Validate max length, return 400 |
| URL without scheme | Auto-prepend `https://` or return 400 (configurable) |
| Expired URL accessed | Return 410 Gone with message |
| SQLite file locked | WAL mode; single-writer; connection pool max 1 writer |
| Redis down | Graceful fallback to DB; cache is optional |
| Malformed short code | Regex validation before DB lookup; return 404 |
| Rate limit exceeded | Return 429 with `Retry-After` header |

### Input Validation Rules

```
URL:    MaxLength 2048, AllowSchemes ["http","https"], BlockPrivate true
Alias:  MinLength 3, MaxLength 30, Pattern ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$
Expiry: Min 1h, Max 365d, formats: "1h", "7d", "30d", "90d", "365d", "never"
```

---

## 14. 📊 Monitoring & Observability

### Structured Logging (slog — stdlib)

```go
slog.Info("url created",
    "short_code", code,
    "is_custom", isCustom,
    "expires_in", expiresIn,
    "latency_ms", latency.Milliseconds(),
)
```

### Prometheus Metrics

```
goshort_urls_created_total{type="generated|custom"}
goshort_redirects_total{status="302|404|410"}
goshort_cache_hits_total / goshort_cache_misses_total
goshort_redirect_duration_seconds (histogram)
goshort_api_request_duration_seconds{method,endpoint} (histogram)
goshort_urls_active_count (gauge)
```

### Health Check

```json
GET /health → {
  "status": "ok", "version": "0.1.0", "uptime": "72h15m",
  "database": "ok", "cache": "ok|disabled", "urls_count": 1542
}
```

---

## 15. 📚 API Documentation & Testing

### API Documentation: OpenAPI + Scalar

Write `docs/openapi.yaml` by hand (5 endpoints = manageable). Serve interactive docs at `/docs` using Scalar UI — a single HTML page that embeds the spec.

```go
// Serve Scalar API docs at /docs
router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(`<!doctype html>
    <html><head><title>GoShort API</title></head><body>
    <script id="api-reference" data-url="/docs/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    </body></html>`))
})
```

### API Testing: Three Layers

| Layer | Tool | Purpose | Phase |
|-------|------|---------|-------|
| **Automated** | Go `httptest` package | TDD, CI/CD — tests live in codebase | 1+ |
| **Manual/Exploratory** | Bruno (`.bru` files in `api-tests/`) | Git-native, no cloud, no account | 2+ |
| **Quick one-off** | `curl + jq` | Terminal, fastest for debugging | All |

### Bruno Setup

```
api-tests/
├── create-url.bru       # POST /api/v1/urls
├── redirect.bru          # GET /:code
├── list-urls.bru         # GET /api/v1/urls
├── delete-url.bru        # DELETE /api/v1/urls/:code
├── health.bru            # GET /health
└── environments/
    ├── local.bru          # base_url=http://localhost:8080
    └── production.bru     # base_url=https://goshort.dev
```

Bruno `.bru` files commit alongside code — other contributors have test requests ready.

---

## 16. 🚀 Deployment Strategy

### Overview: Ship First, Learn Second

```
Phase 3:    Fly.io (ship fast, get feedback)        ← 30 min to live
Phase 3.5:  Bare VPS (learn infrastructure deeply)  ← 1 week to learn
Both:       Cloudflare DNS + CDN                    ← free tier
```

### Path A: Fly.io (Ship Fast)

```bash
fly launch
fly volumes create data --size 1
fly deploy
```

### Path B: Bare VPS (Learn Everything)

```
Day 1: VPS + SSH + firewall (UFW)
Day 2: Go binary + systemd service
Day 3: Domain + DNS (Cloudflare)
Day 4: Nginx reverse proxy
Day 5: Certbot + HTTPS
Day 6: Docker deploy (alternative)
Day 7: Caddy vs Nginx comparison
```

### Docker Compose (Self-Host Distribution)

```yaml
version: "3.8"
services:
  goshort:
    image: ghcr.io/yourname/goshort:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - GOSHORT_DATA_DIR=/data
      - GOSHORT_BASE_URL=https://short.yourdomain.com
      - GOSHORT_CACHE_DRIVER=none
      - GOSHORT_API_KEY=your-secret-key
    restart: unless-stopped

  caddy:
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
    restart: unless-stopped

volumes:
  caddy-data:
```

### Configuration

```toml
# goshort.toml
[server]
port = 8080
base_url = "https://short.yourdomain.com"

[storage]
driver = "sqlite"                    # sqlite | postgres
sqlite_path = "./data/goshort.db"

[cache]
driver = "none"                      # none | memory | redis
redis_url = "redis://localhost:6379"

[auth]
api_key = ""

[rate_limit]
enabled = false
requests_per_minute = 60

[shortener]
code_length = 6
default_expiry = "0"

[logging]
level = "info"
format = "json"
```

### Cross-Compilation

```makefile
build-all:
	GOOS=linux   GOARCH=amd64 go build -o dist/goshort-linux-amd64   ./cmd/server
	GOOS=linux   GOARCH=arm64 go build -o dist/goshort-linux-arm64   ./cmd/server
	GOOS=darwin  GOARCH=arm64 go build -o dist/goshort-darwin-arm64  ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o dist/goshort-windows.exe   ./cmd/server
```

---

## 17. 🗺️ Phase Roadmap

```
Phase 1 (2w)        Phase 2 (3w)         Phase 3 (2w)          Phase 3.5 (1w)
─────────────     ───────────────     ───────────────       ──────────────
Core Library       HTTP API (Chi)       Auth + CLI             Bare VPS
+ SQLite/sqlc      + Scaling Reads      + Rate Limit           + Nginx
+ Sqids            + slog + Prom        + Docker               + Certbot
+ TDD              + Error Handling     + Fly.io Deploy        + systemd
+ Bruno tests      + Scalar docs        + Bruno test suite
─────────────     ───────────────     ───────────────       ──────────────
Deliverable:       Deliverable:         Deliverable:          Deliverable:
go test passes     curl works           v1.0 release          infra knowledge

                                    ▲
                                    │ OPENSOURCE HERE

Phase 4 (1w)                    Phase 5+ (ongoing)
──────────────               ──────────────────────────
MCP Server                    Analytics, Spam Detection
(mcp-go, stdio)               AI Agent (smart slug)
+ Tool defs                   PostgreSQL + Redis counter
+ Claude/Cursor               DB Replication (learning)
──────────────               ──────────────────────────
Deliverable:                  Deliverable:
AI can shorten                v2.0 full features
```

### Key Principle

> 🏗️ **Each phase = working, deployable product.** Stop at any phase and you have something useful.

---

## 18. 🔮 Future: Scaling Writes & DB Replication

> Phase 5+. Learning exercise, not needed at our scale.

### Redis Atomic Counter (Multi-Instance)

```
Instance A: INCR url_counter → 1000001
Instance B: INCR url_counter → 1000002
```

### Counter Batching

```
Instance A: INCRBY url_counter 1000 → gets range [1000, 1999]
  → uses locally, crash loses unused values (gaps harmless)
```

### DB Replication (Docker Compose Learning)

```
PostgreSQL Primary (writes) → Streaming Replication → Replica (reads)
```

### Multi-Region (Interview Level)

```
Region US:    counter range [0, 1B)
Region EU:    counter range [1B, 2B)
Region Asia:  counter range [2B, 3B)
```

---

## Appendix A: Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Custom alias collides with generated code | Data inconsistency | Charset differentiation (hyphens) |
| SQLite corruption | Data loss | WAL mode + periodic backups + PRAGMA integrity_check |
| Redis down | Cache unavailable | Graceful fallback to DB; cache is optional |
| Malicious URLs shortened | Reputation damage | URL validation + future blocklist (Phase 5) |
| API key leaked | Unauthorized access | Key rotation support; rate limiting limits blast radius |
| Fly.io outage | App down | VPS as backup deploy target |

---

*Last updated: 2025-06-01*
*Status: Final — Ready for Phase 1 implementation*
*Next action: Install samber skills → `go mod init` → TDD Phase 1*