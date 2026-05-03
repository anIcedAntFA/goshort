# GoShort — Deployment Guide

Production deployment and operations reference.

## Table of Contents

1. [Deployment Options](#1-deployment-options)
2. [Fly.io (Current Production)](#2-flyio-current-production)
3. [Cloudflare DNS + CDN](#3-cloudflare-dns--cdn)
4. [Docker Compose (Self-Host)](#4-docker-compose-self-host)
5. [Operations](#5-operations)
6. [Troubleshooting](#6-troubleshooting)

---

## 1. Deployment Options

| Method | Best for | Cost | Complexity |
|--------|----------|------|-----------|
| Fly.io + Cloudflare | Production, low-ops | ~$3-5/mo | Low |
| Docker Compose + Caddy | Self-host, full control | VPS cost | Medium |
| Bare VPS + Nginx | Learning infra deeply | VPS cost | High |

GoShort runs as a single binary with embedded SQLite — no external
database required. Any method that can run a Docker container or
a Linux binary works.

---

## 2. Fly.io (Current Production)

GoShort is deployed at **https://goshort.app** on Fly.io Singapore region.

### Architecture

```
User → Cloudflare CDN (SIN edge) → Fly.io (SIN) → GoShort → SQLite
```

### Prerequisites

- Fly.io account with credit card
- `flyctl` CLI installed: `brew install flyctl`
- Domain (optional — `*.fly.dev` works without one)

### Configuration

The app is configured via `fly.toml` (committed to repo) and Fly secrets
(not committed — contains API key).

**fly.toml** defines:
- Region: `sin` (Singapore)
- Machine: `shared-cpu-1x`, 256MB RAM
- Volume: 1GB for SQLite database at `/app/data`
- Auto-stop: VM stops when idle (cost saving)
- Health check: `GET /health` every 30s

**Fly secrets** (set via `fly secrets set`):
- `GOSHORT_AUTH_API_KEY` — API key for write endpoints
- `GOSHORT_SERVER_BASE_URL` — Public URL (overrides fly.toml)

### Deploy

```bash
fly deploy                    # Deploy from Dockerfile
fly status                    # Check machine status
fly logs                      # Tail logs
```

### Useful commands

```bash
fly ssh console               # SSH into running machine
fly ssh sftp get /app/data/goshort.db ./backup.db  # Backup SQLite
fly machine restart            # Restart machine
fly secrets list               # List configured secrets
fly scale count 1              # Ensure single instance (SQLite)
```

### Cost

| Resource | Cost |
|----------|------|
| VM (shared-cpu-1x, idle ~80%) | ~$0.40/mo |
| VM (always-on) | ~$1.94/mo |
| Volume (1GB) | $0.15/mo |
| Dedicated IPv4 (if needed) | $2.00/mo |
| **Typical total** | **~$2-4/mo** |

---

## 3. Cloudflare DNS + CDN

Domain `goshort.app` is managed via Cloudflare (free plan).

### DNS Records

| Type | Name | Value | Proxy |
|------|------|-------|-------|
| A | `@` | `66.241.125.200` | Proxied (orange) |
| AAAA | `@` | `2a09:8280:1::110:ee2:0` | Proxied (orange) |
| CNAME | `www` | `93krgm3.goshort-api.fly.dev` | Proxied (orange) |

**Important:** When first configuring, set proxy to DNS-only (grey cloud)
until Fly.io cert is verified. Then switch to Proxied.

### SSL/TLS

- Mode: **Full** (not Full Strict)
- Fly.io uses Let's Encrypt certs; Cloudflare Full validates them correctly

### Cache Rules (2 rules, order matters)

**Rule 1 (top): Cache short URL redirects**
- Match: All incoming requests
- Action: Eligible for cache
- Edge TTL: 5 minutes (ignore origin cache-control)
- Status Code TTL: 302 → 5 min, 404 → 1 min
- Browser TTL: Bypass (no browser caching of redirects)

**Rule 2 (bottom): Bypass API and admin paths**
- Match: URI Path starts with `/api` OR equals `/health` OR
  equals `/metrics` OR starts with `/docs`
- Action: Bypass cache

Rule 2 is evaluated last and overrides Rule 1 for matching paths.

### Cache behavior

| Path | cf-cache-status | Explanation |
|------|----------------|-------------|
| `GET /abc123` (1st) | MISS | First request, cached for 5 min |
| `GET /abc123` (2nd) | HIT | Served from Cloudflare edge |
| `POST /api/v1/urls` | DYNAMIC | Bypassed by Rule 2 |
| `GET /health` | DYNAMIC | Bypassed by Rule 2 |

### Cache + Delete tradeoff

When a URL is deleted via API, Cloudflare may still serve the cached
redirect for up to 5 minutes. This is an intentional tradeoff — CDN
caching reduces origin load at the cost of slightly delayed deletion
visibility. For immediate purge, use Cloudflare API or dashboard.

### Cert management

```bash
fly certs list                 # List certificates
fly certs check goshort.app    # Check cert status
fly certs add <domain>         # Add new domain
```

---

## 4. Docker Compose (Self-Host)

For users who want to run GoShort on their own server.

### Quick start

```bash
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/goshort.toml
curl -O https://raw.githubusercontent.com/anIcedAntFA/goshort/main/Caddyfile

# Edit goshort.toml — set base_url and api_key
# Edit Caddyfile — replace localhost with your domain

docker compose up -d
```

### What's included

- **GoShort** — app container (~49MB Alpine image)
- **Caddy** — reverse proxy with automatic HTTPS (Let's Encrypt)
- **Redis** — optional, uncomment in docker-compose.yml

### Volumes

| Volume | Purpose |
|--------|---------|
| `goshort-data` | SQLite database persistence |
| `caddy-data` | TLS certificates |
| `caddy-config` | Caddy configuration |

### Enable Redis (optional)

Uncomment the `redis` service in `docker-compose.yml` and set:

```bash
CACHE_DRIVER=redis docker compose up -d
```

---

## 5. Operations

### Backup

```bash
# Fly.io
fly ssh sftp get /app/data/goshort.db ./backup-$(date +%Y%m%d).db

# Docker Compose
docker cp goshort:/app/data/goshort.db ./backup-$(date +%Y%m%d).db
```

### Monitoring

- **Health:** `GET /health` → `{"status":"ok"}`
- **Metrics:** `GET /metrics` → Prometheus format
- **Logs:** `fly logs` (Fly.io) or `docker compose logs -f` (self-host)

### Key metrics to watch

```
goshort_redirects_total{status="302"}     # successful redirects
goshort_redirects_total{status="404"}     # not found
goshort_redirects_total{status="410"}     # expired
goshort_urls_created_total{type}          # new URLs
goshort_http_request_duration_seconds     # latency
```

---

## 6. Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| `Could not resolve` | DNS not configured | Add A/AAAA/CNAME records |
| Cert "Not verified" | Cloudflare proxy blocking ACME | Set DNS-only, wait, then re-enable proxy |
| 502 after deploy | App crash or cold start | `fly logs` to check error |
| SQLite locked | Multiple instances | `fly scale count 1` |
| Cache serving stale redirect | CDN cache TTL | Wait 5 min or purge via Cloudflare |
| Health returning EXPIRED | Cache rule order wrong | Bypass rule must be BELOW cache rule |
