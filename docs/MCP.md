# 🔌 MCP.md — Model Context Protocol Deep Dive

> **TL;DR:** MCP is "USB-C for AI" — a JSON-RPC protocol that lets AI models discover and call tools on external servers through a standardized interface. Instead of every AI app writing custom integrations for every tool (N×M problem), MCP makes it N+M: build a server once, any compliant client can use it.
>
> **For:** Developers who want to understand MCP from first principles, then see it applied in a real Go codebase (GoShort).
>
> **How to use:** Part 1–3 teaches MCP concepts. Part 4 maps everything to GoShort's `internal/mcp/` package. Part 5 covers advanced topics. "🎯 In GoShort:" annotations tell you exactly WHERE in the codebase each concept appears.

---

## 📖 Table of Contents

- [Part 1: Why MCP Exists](#part-1-why-mcp-exists)
- [Part 2: Core Concepts](#part-2-core-concepts)
- [Part 3: Protocol Mechanics](#part-3-protocol-mechanics)
- [Part 4: GoShort MCP Implementation](#part-4-goshort-mcp-implementation)
- [Part 5: Advanced Topics](#part-5-advanced-topics)
- [Part 6: Resources & References](#part-6-resources--references)
- [Appendix A: MCP Glossary](#appendix-a-mcp-glossary)
- [Appendix B: JSON-RPC Message Reference](#appendix-b-json-rpc-message-reference)

---

## Part 1: Why MCP Exists

### 🧩 The N×M Integration Problem

Before MCP, connecting AI models to external tools was chaos.

```
╭─────────────────────────────────────────────────────────╮
│  BEFORE MCP: The N×M Problem                            │
│                                                         │
│  AI Apps (M)          Tools (N)                         │
│  ┌──────────┐         ┌──────────┐                      │
│  │ ChatGPT  │─────────│ Slack    │                      │
│  │          │──┐  ┌───│          │                      │
│  └──────────┘  │  │   └──────────┘                      │
│  ┌──────────┐  │  │   ┌──────────┐                      │
│  │ Claude   │──┼──┼───│ GitHub   │                      │
│  │          │──┼──┼─┐ │          │                      │
│  └──────────┘  │  │ │ └──────────┘                      │
│  ┌──────────┐  │  │ │ ┌──────────┐                      │
│  │ Gemini   │──┘  └─┼─│ Database │                      │
│  │          │────────┘ │          │                      │
│  └──────────┘          └──────────┘                      │
│                                                         │
│  3 apps × 3 tools = 9 custom integrations               │
│  Each with its own auth, schema, error handling          │
│  10 apps × 20 tools = 200 integrations 💀               │
╰─────────────────────────────────────────────────────────╯
```

Every AI app had to write custom code for every tool. Different auth flows, different schemas, different error formats. Scale this to 10 apps × 20 tools and you have 200 bespoke integrations to maintain.

```
╭─────────────────────────────────────────────────────────╮
│  AFTER MCP: The N+M Solution                            │
│                                                         │
│  AI Apps (M)       MCP        Servers (N)               │
│  ┌──────────┐   ┌───────┐   ┌──────────┐               │
│  │ ChatGPT  │───│       │───│ Slack    │               │
│  └──────────┘   │       │   └──────────┘               │
│  ┌──────────┐   │  MCP  │   ┌──────────┐               │
│  │ Claude   │───│Protocol│───│ GitHub   │               │
│  └──────────┘   │       │   └──────────┘               │
│  ┌──────────┐   │       │   ┌──────────┐               │
│  │ Gemini   │───│       │───│ Database │               │
│  └──────────┘   └───────┘   └──────────┘               │
│                                                         │
│  3 apps + 3 servers = 6 implementations                 │
│  Standard auth, schema, error handling                   │
│  10 apps + 20 servers = 30 implementations ✅            │
╰─────────────────────────────────────────────────────────╯
```

> 💡 **ELI5:** Imagine every phone needed a different charger — one for Samsung, one for Apple, one for Xiaomi. That's the N×M problem. USB-C solved it: one standard port, every phone works with every charger. MCP is USB-C for AI.

### 📜 Brief History

| Date | Event |
|------|-------|
| Nov 2024 | Anthropic releases MCP as open standard |
| Early 2025 | Claude Desktop, Claude Code adopt MCP natively |
| Mar 2025 | OpenAI adopts MCP for ChatGPT Desktop |
| Mid 2025 | Google DeepMind, Microsoft integrate MCP |
| Nov 2025 | MCP 2025-11-25 spec: tasks, elicitation, OAuth 2.0 |
| 2026 | Linux Foundation governance, MCP Apps (UI), 97M+ monthly SDK downloads |

### 🏗️ What MCP Standardizes

MCP defines three things:

1. **What to say** — the data format (JSON-RPC 2.0 messages)
2. **How to say it** — the transport (stdio for local, HTTP for remote)
3. **What can be said** — the primitives (tools, resources, prompts)

It does NOT define:
- What the AI model does with the data (that's the model's job)
- How auth works at the application level (that's the host's job)
- What tools should exist (that's the server developer's job)

---

## Part 2: Core Concepts

### 🏛️ Architecture — Three Roles

```
╭─────────────────────────────────────────────────────────╮
│  MCP Architecture: Host → Client → Server               │
│                                                         │
│  ┌─────────────────────────────────────────────┐        │
│  │  HOST (Claude Desktop / Claude Code / IDE)   │        │
│  │                                              │        │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │        │
│  │  │ MCP      │  │ MCP      │  │ MCP      │  │        │
│  │  │ Client 1 │  │ Client 2 │  │ Client 3 │  │        │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  │        │
│  └───────┼──────────────┼──────────────┼───────┘        │
│          │              │              │                 │
│     ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐          │
│     │ MCP      │  │ MCP      │  │ MCP      │          │
│     │ Server A │  │ Server B │  │ Server C │          │
│     │ (GoShort)│  │ (GitHub) │  │ (Slack)  │          │
│     └──────────┘  └──────────┘  └──────────┘          │
╰─────────────────────────────────────────────────────────╯
```

| Role | What it does | Example |
|------|-------------|---------|
| **Host** | The AI application users interact with. Manages multiple clients. Owns security, consent, user trust. | Claude Desktop, Claude Code, Cursor, ChatGPT |
| **Client** | Maintains a 1:1 stateful connection to one server. Routes messages. Handles capability negotiation. | Built into the host — one client per server connection |
| **Server** | Exposes capabilities (tools, resources, prompts) via MCP primitives. Lightweight, focused process. | GoShort, filesystem server, GitHub MCP, Sentry MCP |

**Key insight:** The host spawns one client per server. Each client↔server pair is an isolated session. GoShort's MCP server doesn't know (or care) about the GitHub server running alongside it.

> 💡 **ELI5:** The Host is like an office manager. Each Client is a phone line to a different department. Each Server is a department (accounting, HR, IT). The office manager decides which calls to make, but each call is private — accounting doesn't hear the HR call.

🎯 **In GoShort:** `internal/mcp/server.go` — GoShort is an MCP **Server**. Claude Code is the Host+Client. When Claude Code runs GoShort with `--mcp`, it spawns a client that connects to GoShort's server.

### 🧱 The Three Primitives

MCP servers expose exactly three types of capabilities. No more, no less.

```
╭────────────────────────────────────────────────────────────╮
│  MCP Primitives                                            │
│                                                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   🔧 Tools   │  │  📄 Resources│  │  💬 Prompts  │       │
│  │             │  │             │  │             │       │
│  │ Actions the │  │ Data the    │  │ Templates   │       │
│  │ AI can      │  │ AI can      │  │ that guide  │       │
│  │ EXECUTE     │  │ READ        │  │ AI behavior │       │
│  │             │  │             │  │             │       │
│  │ Model-      │  │ App-        │  │ User-       │       │
│  │ controlled  │  │ controlled  │  │ controlled  │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
│                                                            │
│  "Do this"        "Read this"       "Say it like this"    │
╰────────────────────────────────────────────────────────────╯
```

#### 🔧 Tools — "Functions the AI can call"

Tools are **actions**. The model decides when to call them based on user intent. They have input schemas (what parameters they accept) and return results.

```json
{
  "name": "shorten_url",
  "description": "Create a short URL from a long URL",
  "inputSchema": {
    "type": "object",
    "properties": {
      "url": { "type": "string", "description": "The URL to shorten" },
      "alias": { "type": "string", "description": "Custom alias" }
    },
    "required": ["url"]
  }
}
```

The model reads the description + schema → decides whether to call the tool → sends a `tools/call` request → gets a result.

**Who controls tools?** The **model** decides when to call them (model-controlled). The host may require user confirmation before execution.

🎯 **In GoShort:** `internal/mcp/tools.go` — 5 tools: `shorten_url`, `list_urls`, `get_url_stats`, `delete_url`, `lookup_url`. Each has a typed Go struct for input with `jsonschema` tags that auto-generate the JSON Schema.

#### 📄 Resources — "Data the AI can read"

Resources are **read-only context**. They provide structured data the AI can access — files, database schemas, summaries, configurations.

```json
{
  "uri": "goshort://stats/summary",
  "name": "GoShort Stats Summary",
  "description": "Total URL count and top URLs by click count",
  "mimeType": "application/json"
}
```

Resources are identified by URIs. They can be static (always available) or templated (parameterized).

**Who controls resources?** The **application** decides when to read them (app-controlled). The AI can request a resource read, but the host mediates.

🎯 **In GoShort:** `internal/mcp/resources.go` — 2 resources:
- `goshort://stats/summary` — total URL count + top 5 URLs
- `goshort://urls/{code}` — full details for a specific short code (resource template)

#### 💬 Prompts — "Reusable instruction templates"

Prompts are predefined templates that guide how the AI should approach a task. The team that owns the domain (e.g., GoShort) writes the prompt — not the user.

```json
{
  "name": "shorten_and_share",
  "description": "Shorten a URL and format it for sharing on a platform",
  "arguments": [
    { "name": "url", "description": "URL to shorten", "required": true },
    { "name": "platform", "description": "slack, email, twitter", "required": false }
  ]
}
```

**Who controls prompts?** The **user** selects them (user-controlled). They appear in the UI as slash commands or templates.

🎯 **In GoShort:** `internal/mcp/prompts.go` — 2 prompts:
- `shorten_and_share` — shorten + format for a specific platform
- `batch_shorten` — shorten multiple URLs and return a table

### 🔄 How They Differ

| Aspect | Tools | Resources | Prompts |
|--------|-------|-----------|---------|
| Purpose | Execute actions | Provide context | Guide behavior |
| Control | Model-controlled | App-controlled | User-controlled |
| Analogy | Function call | File read | Template |
| Has side effects? | Yes (create, delete, mutate) | No (read-only) | No |
| Example | `shorten_url(url)` | `goshort://stats/summary` | "Shorten and format for Slack" |

> 💡 **ELI5:** Think of a restaurant. **Tools** are the kitchen — they cook food (actions). **Resources** are the menu — you read it to know what's available (data). **Prompts** are the waiter's suggestions — "May I recommend the chef's special?" (templates).

---

## Part 3: Protocol Mechanics

### 📨 JSON-RPC 2.0 — The Wire Format

All MCP communication uses JSON-RPC 2.0. Three message types:

```
┌──────────────────────────────────────────────────────┐
│  JSON-RPC 2.0 Message Types                          │
│                                                      │
│  Request (expects a response):                       │
│  {                                                   │
│    "jsonrpc": "2.0",                                 │
│    "id": 1,              ← unique, ties to response  │
│    "method": "tools/call",                           │
│    "params": { ... }                                 │
│  }                                                   │
│                                                      │
│  Response (answers a request):                       │
│  {                                                   │
│    "jsonrpc": "2.0",                                 │
│    "id": 1,              ← matches the request id    │
│    "result": { ... }     ← or "error": { ... }      │
│  }                                                   │
│                                                      │
│  Notification (fire-and-forget, no response):        │
│  {                                                   │
│    "jsonrpc": "2.0",                                 │
│    "method": "notifications/initialized"             │
│    ← NO "id" field = notification                    │
│  }                                                   │
└──────────────────────────────────────────────────────┘
```

**Key rule:** If a message has an `id` → it's a request, and expects a response with the same `id`. If no `id` → it's a notification, no response expected.

### 🔄 Session Lifecycle

Every MCP session follows this exact sequence:

```
Client                                          Server
  │                                                │
  │  1. initialize (request)                       │
  │  { method: "initialize",                       │
  │    params: { protocolVersion, capabilities,    │
  │              clientInfo } }                    │
  ├───────────────────────────────────────────────▶│
  │                                                │
  │  2. initialize response                        │
  │  { result: { protocolVersion, capabilities,    │
  │              serverInfo } }                    │
  │◀───────────────────────────────────────────────┤
  │                                                │
  │  3. notifications/initialized (notification)   │
  │  { method: "notifications/initialized" }       │
  │  ← NO id field, no response expected           │
  ├───────────────────────────────────────────────▶│
  │                                                │
  │  ════════ SESSION IS NOW READY ════════        │
  │                                                │
  │  4. tools/list (request)                       │
  ├───────────────────────────────────────────────▶│
  │◀───────────────────────────────────────────────┤
  │                                                │
  │  5. tools/call (request)                       │
  ├───────────────────────────────────────────────▶│
  │◀───────────────────────────────────────────────┤
  │                                                │
  │  6. resources/read (request)                   │
  ├───────────────────────────────────────────────▶│
  │◀───────────────────────────────────────────────┤
  │                                                │
  │  ... (more requests) ...                       │
  │                                                │
  │  N. Transport closes                           │
  │  ════════ SESSION ENDS ════════                │
```

**Step 1–3 is the handshake.** This is where the "invalid during session initialization" error comes from — if you send `tools/call` before step 3, the server rejects it because the session isn't ready.

**What gets negotiated:**
- Protocol version (must be compatible)
- Server capabilities: which primitives it supports (tools? resources? prompts?)
- Client capabilities: what features it supports (sampling? roots?)
- Server/client identity info (name, version)

🎯 **In GoShort:** The handshake is handled automatically by `modelcontextprotocol/go-sdk`. When `server.Run(ctx, transport)` is called, the SDK handles initialize → capabilities → ready. GoShort's `server.go` just registers tools/resources/prompts before calling `Run`.

### 🚚 Transports — How Messages Travel

MCP supports two transports. Same JSON-RPC messages, different delivery mechanism.

#### stdio — Local (same machine)

```
┌──────────────┐   stdin (JSON-RPC)    ┌──────────────┐
│  Host        │ ────────────────────▶ │  MCP Server  │
│  (Claude     │                       │  (GoShort    │
│   Code)      │ ◀──────────────────── │   binary)    │
│              │   stdout (JSON-RPC)   │              │
└──────────────┘                       └──────────────┘
```

The host spawns the server as a child process. Messages flow through stdin/stdout pipes. No network, no auth needed — OS process isolation provides security.

**When to use:** Local development, IDE integrations (Claude Code, Cursor), same-machine tools.

🎯 **In GoShort:** `./bin/goshort --mcp` starts stdio mode. `server.RunStdio(ctx)` calls `server.Run(ctx, &mcp.StdioTransport{})`.

#### Streamable HTTP — Remote (over network)

```
┌──────────────┐   HTTP POST + SSE     ┌──────────────┐
│  MCP Client  │ ────────────────────▶ │  MCP Server  │
│  (remote)    │                       │  (GoShort    │
│              │ ◀──────────────────── │   on Fly.io) │
│              │   Server-Sent Events  │              │
└──────────────┘                       └──────────────┘
```

Client sends JSON-RPC requests via HTTP POST. Server responds via Server-Sent Events (SSE) for streaming. Session state is tracked via `Mcp-Session-Id` header.

**When to use:** Remote servers, cloud deployments, multi-client scenarios.

**Session management:**
1. Client sends `initialize` request → server returns response with `Mcp-Session-Id` header
2. Client must include `Mcp-Session-Id` in ALL subsequent requests
3. Missing or invalid session ID → server rejects with error

🎯 **In GoShort:** `/mcp` endpoint on main port (8080). `server.RunHTTP(ctx, addr, apiKey)` or `server.HTTPHandler(apiKey)` for embedding in existing router. `APIKeyMiddleware` wraps the handler for auth.

### 🔑 Key MCP Methods

| Method | Direction | Purpose |
|--------|-----------|---------|
| `initialize` | Client → Server | Start session, negotiate capabilities |
| `notifications/initialized` | Client → Server | Signal handshake complete |
| `tools/list` | Client → Server | Discover available tools |
| `tools/call` | Client → Server | Execute a tool |
| `resources/list` | Client → Server | Discover available resources |
| `resources/read` | Client → Server | Read a resource's content |
| `prompts/list` | Client → Server | Discover available prompts |
| `prompts/get` | Client → Server | Get a prompt template with arguments |
| `ping` | Either direction | Keep-alive / health check |
| `notifications/tools/list_changed` | Server → Client | Tools were added/removed |

---

## Part 4: GoShort MCP Implementation

### 📁 File Structure

```
internal/mcp/
├── server.go       # Server struct, NewServer, RunStdio, RunHTTP, HTTPHandler
├── tools.go        # 5 tool handlers with typed input/output structs
├── resources.go    # 2 resource handlers (stats summary, URL by code)
├── prompts.go      # 2 prompt handlers (shorten_and_share, batch_shorten)
├── auth.go         # APIKeyMiddleware for HTTP transport
├── auth_test.go    # Auth middleware tests
├── tools_test.go   # 15 tool tests via in-process transport
└── server_test.go  # Resource, prompt, and registration tests
```

### 🔧 Tool Implementation Pattern

The official `go-sdk` uses **generics** (`AddTool[In, Out]`) to auto-generate JSON Schema from Go struct tags:

```go
// 1. Define typed input struct with jsonschema tags
type shortenInput struct {
    URL       string `json:"url"       jsonschema:"required,The URL to shorten"`
    Alias     string `json:"alias,omitempty"    jsonschema:"Custom alias (3-30 chars)"`
    ExpiresIn string `json:"expires_in,omitempty" jsonschema:"Expiration e.g. 1h 7d 30d"`
}

// 2. Define typed output struct
type shortenOutput struct {
    ShortCode   string  `json:"short_code"`
    ShortURL    string  `json:"short_url"`
    OriginalURL string  `json:"original_url"`
    ExpiresAt   *string `json:"expires_at,omitempty"`
    CreatedAt   string  `json:"created_at"`
}

// 3. Handler function — generic signature
func (s *Server) handleShortenURL(
    ctx context.Context,
    _ *mcp.CallToolRequest,   // raw request (rarely needed)
    in shortenInput,           // auto-deserialized from JSON
) (*mcp.CallToolResult, shortenOutput, error) {
    // Call the SAME service the HTTP handler uses — zero duplication
    url, err := s.svc.Create(ctx, shortener.CreateRequest{
        URL:         in.URL,
        CustomAlias: in.Alias,
        ExpiresIn:   in.ExpiresIn,
    })
    if err != nil {
        return nil, shortenOutput{}, err  // SDK wraps as error result
    }
    return nil, shortenOutput{
        ShortCode:   url.ShortCode,
        ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, url.ShortCode),
        OriginalURL: url.OriginalURL,
        CreatedAt:   url.CreatedAt.UTC().Format(time.RFC3339),
    }, nil
}

// 4. Register during server setup
mcp.AddTool(s.server, &mcp.Tool{
    Name:        "shorten_url",
    Description: "Create a short URL from a long URL",
}, s.handleShortenURL)
// SDK reads shortenInput struct tags → generates JSON Schema automatically
```

**Why this pattern is powerful:**
- `jsonschema:"required,description"` tags → SDK auto-generates the JSON Schema the AI model reads
- Typed input/output → compile-time safety, no `map[string]any` casting
- Handler calls `shortener.Service` — same business logic as HTTP API, zero duplication
- Error handling: return `error` → SDK wraps it as `isError: true` in the tool result

### 📄 Resource Implementation

```go
// Static resource — always available
s.server.AddResource(&mcp.Resource{
    URI:         "goshort://stats/summary",
    Name:        "GoShort Stats Summary",
    Description: "Total URL count and top URLs by click count",
    MIMEType:    "application/json",
}, s.handleStatsSummary)

// Resource template — parameterized by {code}
s.server.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "goshort://urls/{code}",
    Name:        "URL Details",
    Description: "Full details for a shortened URL by short code",
}, s.handleURLByCode)
```

Resources return `ReadResourceResult` with content:
```go
return &mcp.ReadResourceResult{
    Contents: []*mcp.ResourceContents{{
        URI:      req.Params.URI,
        MIMEType: "application/json",
        Text:     string(jsonBytes),
    }},
}, nil
```

### 🧪 Testing Pattern — In-Process Transport

No real stdio/HTTP needed. The SDK provides `NewInMemoryTransports()`:

```go
func newTestClient(t *testing.T) *mcp.ClientSession {
    // Real SQLite + real service — only transport is fake
    store, _ := storage.NewSQLiteStorage(ctx, ":memory:")
    enc, _ := encoder.NewSqidsEncoder(4)
    svc := shortener.NewService(store, enc)
    srv := mcpserver.NewServer(svc, "http://localhost:8080")

    // In-memory transport — no network, no stdio
    ct, st := mcp.NewInMemoryTransports()

    // Connect server side
    ss, _ := srv.MCPServer().Connect(ctx, st, nil)

    // Connect client side
    client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
    cs, _ := client.Connect(ctx, ct, nil)

    return cs  // use cs.CallTool(), cs.ListTools(), etc.
}
```

This pattern gives you:
- Real business logic (SQLite, encoder, validator — all real)
- Real MCP protocol (handshake, capability negotiation — all real)
- No I/O overhead (in-memory pipes, nanosecond latency)
- Parallel test safety (each test gets its own `:memory:` DB)

🎯 **In GoShort:** `internal/mcp/tools_test.go` uses this pattern for all 15 tool tests.

### 🔒 Auth — HTTP Transport Only

stdio doesn't need auth (OS process isolation). HTTP needs it:

```go
func APIKeyMiddleware(apiKey string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if apiKey == "" {
            next.ServeHTTP(w, r)   // auth disabled
            return
        }
        got := r.Header.Get("X-API-Key")
        if subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) != 1 {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

Constant-time comparison prevents timing attacks. Empty API key = auth disabled (backward compatible).

🎯 **In GoShort:** `internal/mcp/auth.go`. The `/mcp` endpoint bypasses Chi's rate limit middleware because MCP sessions are long-lived — per-request rate limiting would break them.

### 🔌 Dual Transport — Same Server, Two Entry Points

```
                    ┌─────────────────────────────────┐
                    │        mcp.Server (go-sdk)       │
                    │                                  │
                    │  5 tools │ 2 resources │ 2 prompts│
                    │                                  │
                    │  shortener.Service (shared)      │
                    └───────┬──────────┬───────────────┘
                            │          │
              ┌─────────────┘          └──────────────┐
              │                                       │
     ┌────────▼─────────┐               ┌─────────────▼──────┐
     │  RunStdio(ctx)   │               │  HTTPHandler(key)  │
     │  --mcp flag      │               │  /mcp on port 8080 │
     │  Claude Code     │               │  Remote agents     │
     │  Cursor          │               │  API Connector     │
     └──────────────────┘               └────────────────────┘
```

Both transports use the **same `mcp.Server` instance** with the same registered tools/resources/prompts. The transport is just the delivery mechanism.

🎯 **In GoShort:** `cmd/server/main.go` wires both:
- `--mcp` flag → `srv.RunStdio(ctx)` — dedicated process, stdio only
- Main HTTP server → `router.Handle("/mcp", mcpSrv.HTTPHandler(apiKey))` — shared port

### 📊 Request Flow — Tool Call End-to-End

Here's what happens when Claude Code says "shorten this URL":

```
User: "Shorten https://github.com/anIcedAntFA/goshort"
  │
  ▼
Claude Code (Host)
  │  Analyzes user intent
  │  Sees shorten_url tool available (from tools/list)
  │  Decides to call it
  │
  ├──── MCP Client ────▶ GoShort MCP Server
  │     {                     │
  │       "jsonrpc": "2.0",   │
  │       "id": 42,           │
  │       "method":           │  1. SDK deserializes JSON
  │         "tools/call",     │     into shortenInput struct
  │       "params": {         │
  │         "name":           │  2. handleShortenURL() called
  │           "shorten_url",  │     with typed input
  │         "arguments": {    │
  │           "url": "https://│  3. Calls s.svc.Create()
  │            github.com/    │     (same service as HTTP API)
  │            anIcedAntFA/   │
  │            goshort"       │  4. Service validates URL,
  │         }                 │     increments counter,
  │       }                   │     encodes with Sqids,
  │     }                     │     inserts into SQLite
  │                           │
  │     {                     │  5. Returns shortenOutput
  │       "jsonrpc": "2.0",   │     SDK serializes to JSON
  │       "id": 42,           │
  │       "result": {         │
  │         "content": [{     │
  │           "type": "text", │
  │           "text": "{      │
  │             \"short_code\":│
  │             \"k7Xm2p\",   │
  │             \"short_url\":│
  │             \"https://    │
  │              goshort.app/ │
  │              k7Xm2p\"    │
  │           }"              │
  │         }]                │
  │       }                   │
  │     }                     │
  │◀──── MCP Client ─────────┘
  │
  ▼
Claude Code formats response:
"Here's your short URL: https://goshort.app/k7Xm2p"
```

**Key insight:** The MCP server doesn't know or care about the AI model. It receives a `tools/call` request, executes business logic, returns a result. The model decides **what** to call and **how** to present the result to the user.

---

## Part 5: Advanced Topics

### 🔄 MCP vs REST API — Why Both?

GoShort has both a REST API and an MCP server. Why?

| Aspect | REST API (`/api/v1/urls`) | MCP Server (`/mcp`) |
|--------|--------------------------|---------------------|
| Consumer | Apps, scripts, other services | AI models (Claude, Cursor, agents) |
| Discovery | Read OpenAPI spec manually | Auto-discovery via `tools/list` |
| Auth | `X-API-Key` header | `X-API-Key` (HTTP) or OS process (stdio) |
| Session | Stateless (each request independent) | Stateful (handshake → session → calls) |
| Schema | OpenAPI 3.1 YAML | JSON Schema in tool definitions |
| Caching | Cache-aside in handler | No cache (not hot-path) |
| Use case | Programmatic CRUD | Conversational AI interaction |

Both call the **same `shortener.Service`** — zero business logic duplication.

### 📡 Anthropic API MCP Connector

Instead of running an MCP client yourself, Anthropic's API can connect to your MCP server directly:

```python
response = client.beta.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    betas=["mcp-client-2025-11-20"],
    mcp_servers=[{
        "type": "url",
        "url": "https://goshort.app/mcp",
        "name": "goshort",
        "authorization_token": "YOUR_KEY"
    }],
    messages=[{
        "role": "user",
        "content": "Shorten https://example.com"
    }]
)
```

Anthropic's infrastructure acts as the MCP client — it connects to `goshort.app/mcp`, performs the handshake, discovers tools, and Claude decides what to call. One API request from your app, everything else handled.

**Limitation:** Only tools are supported via this method — resources and prompts are not (yet).

### 🔐 Security Considerations

MCP's security model puts trust in the **host**, not the protocol:

```
╭────────────────────────────────────────────────────────╮
│  Security Responsibilities                              │
│                                                         │
│  HOST (Claude Code):                                    │
│  ✓ User consent before tool execution                   │
│  ✓ Credential management (API keys, OAuth tokens)       │
│  ✓ Per-tool allow/deny lists                            │
│  ✓ Sandboxing server processes                          │
│                                                         │
│  MCP SERVER (GoShort):                                  │
│  ✓ Input validation (don't trust tool arguments)        │
│  ✓ Auth on HTTP transport (API key, OAuth)              │
│  ✓ Rate limiting (if applicable)                        │
│  ✓ Principle of least privilege                         │
│                                                         │
│  MCP PROTOCOL:                                          │
│  ✓ Capability negotiation (don't use undeclared features)│
│  ✓ Session isolation (one client ↔ one server)          │
│  ✗ Does NOT define auth (that's transport-level)        │
│  ✗ Does NOT sandbox servers (that's the host's job)     │
╰────────────────────────────────────────────────────────╯
```

🎯 **In GoShort:**
- stdio: OS process isolation — Claude Code spawns GoShort, only that process can access its stdin/stdout
- HTTP: `APIKeyMiddleware` with constant-time comparison
- Input validation: MCP tool handlers call the same validators as the REST API (`ValidateURL`, `ValidateAlias`, `ValidateExpiresIn`)

### 🆚 MCP Protocol Versions

| Version | Date | Key Changes |
|---------|------|-------------|
| `2024-11-05` | Nov 2024 | Initial release. stdio + SSE transports. |
| `2025-03-26` | Mar 2025 | Streamable HTTP transport (replaces SSE). Session management. |
| `2025-06-18` | Jun 2025 | Elicitation, OAuth 2.1, tool annotations. |
| `2025-11-25` | Nov 2025 | Tasks (async), structured output, MCP Apps (UI). |

GoShort uses `protocolVersion: "2025-03-26"` which introduced Streamable HTTP — the transport GoShort uses for `/mcp`.

---

## Part 6: Resources & References

### 📚 Official Documentation

| Resource | URL | What you'll learn |
|----------|-----|-------------------|
| MCP Spec | [modelcontextprotocol.io/specification](https://modelcontextprotocol.io/specification/2025-11-25) | Full protocol spec — the source of truth |
| Architecture | [modelcontextprotocol.io/docs/learn/architecture](https://modelcontextprotocol.io/docs/learn/architecture) | Host/client/server roles, primitives, lifecycle |
| Getting Started | [modelcontextprotocol.io/docs/getting-started](https://modelcontextprotocol.io/docs/getting-started/intro) | First MCP server tutorial |
| Go SDK | [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) | Official Go SDK — what GoShort uses |
| TypeScript SDK | [github.com/modelcontextprotocol/typescript-sdk](https://github.com/modelcontextprotocol/typescript-sdk) | Reference implementation |
| MCP Blog | [blog.modelcontextprotocol.io](https://blog.modelcontextprotocol.io) | Spec updates, retrospectives |

### 📖 Learning Resources

| Resource | URL | Why read it |
|----------|-----|-------------|
| Anthropic MCP Announcement | [anthropic.com/news/model-context-protocol](https://www.anthropic.com/news/model-context-protocol) | Original vision and motivation |
| MCP Cheat Sheet | [webfuse.com/mcp-cheat-sheet](https://www.webfuse.com/mcp-cheat-sheet) | Quick reference for all concepts |
| Protocol Mechanics Deep Dive | [pradeepl.com/blog/mcp-protocol-mechanics](https://pradeepl.com/blog/model-context-protocol/mcp-protocol-mechanics-and-architecture/) | Detailed technical walkthrough |
| Wikipedia | [en.wikipedia.org/wiki/Model_Context_Protocol](https://en.wikipedia.org/wiki/Model_Context_Protocol) | History, adoption timeline |

### 🔧 GoShort-Specific References

| File | What it teaches |
|------|----------------|
| `internal/mcp/server.go` | Server setup, dual transport, HTTPHandler pattern |
| `internal/mcp/tools.go` | Generic tool handlers with typed I/O structs |
| `internal/mcp/resources.go` | Static + templated resources |
| `internal/mcp/prompts.go` | Prompt templates with arguments |
| `internal/mcp/auth.go` | HTTP auth middleware for MCP |
| `internal/mcp/tools_test.go` | In-process testing with NewInMemoryTransports |
| `.mcp.json` | Claude Code project config for stdio |
| `cmd/server/main.go` | Wiring: runMCPMode + HTTPHandler on Chi router |

---

## Appendix A: MCP Glossary

| Term | Definition |
|------|-----------|
| **Host** | The AI application (Claude Desktop, Claude Code, Cursor). Manages clients, owns security. |
| **Client** | Component inside the host that maintains a 1:1 connection to one server. |
| **Server** | Process that exposes tools, resources, and prompts via MCP. |
| **Primitive** | One of three capability types: tool, resource, or prompt. |
| **Tool** | Executable action (model-controlled). Has input schema + returns result. |
| **Resource** | Read-only data (app-controlled). Identified by URI. |
| **Prompt** | Reusable template (user-controlled). Guides AI behavior for a task. |
| **Transport** | How messages travel: stdio (local pipes) or Streamable HTTP (network). |
| **Capability** | Feature advertised during handshake (e.g., "I support tools"). |
| **Session** | Stateful connection from initialize → usage → close. |
| **JSON-RPC 2.0** | The wire format for all MCP messages. |
| **Notification** | JSON-RPC message without `id` — fire-and-forget, no response expected. |
| **Handshake** | The 3-step initialization: initialize → response → notifications/initialized. |
| **Mcp-Session-Id** | HTTP header tracking session state for Streamable HTTP transport. |
| **Resource Template** | A resource with URI parameters, e.g., `goshort://urls/{code}`. |
| **Sampling** | Server asking the client's LLM to generate a completion (advanced, recursive). |
| **Elicitation** | Server asking the user a question via the client (2025-06-18+). |

---

## Appendix B: JSON-RPC Message Reference

### Handshake

```json
// Client → Server: initialize
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {},
    "clientInfo": { "name": "claude-code", "version": "2.1" }
  }
}

// Server → Client: initialize response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": { "listChanged": true },
      "resources": { "listChanged": true },
      "prompts": { "listChanged": true }
    },
    "serverInfo": { "name": "goshort", "version": "0.4.0" }
  }
}

// Client → Server: initialized notification (NO id!)
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

### Tool Operations

```json
// Client → Server: list tools
{ "jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {} }

// Client → Server: call tool
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "shorten_url",
    "arguments": {
      "url": "https://github.com/anIcedAntFA/goshort",
      "alias": "my-repo"
    }
  }
}

// Server → Client: tool result (success)
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{
      "type": "text",
      "text": "{\"short_code\":\"my-repo\",\"short_url\":\"https://goshort.app/my-repo\"}"
    }]
  }
}

// Server → Client: tool result (error)
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "isError": true,
    "content": [{
      "type": "text",
      "text": "alias already taken"
    }]
  }
}
```

### Resource Operations

```json
// Client → Server: read resource
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "resources/read",
  "params": { "uri": "goshort://stats/summary" }
}

// Server → Client: resource content
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "contents": [{
      "uri": "goshort://stats/summary",
      "mimeType": "application/json",
      "text": "{\"total_urls\":25,\"top_urls\":[...]}"
    }]
  }
}
```

---

```
╭──────────────────────────────────────────────────────╮
│                                                      │
│  MCP is not just a protocol.                         │
│  It's the difference between every AI app            │
│  reinventing integrations from scratch,              │
│  and building once for all of them.                  │
│                                                      │
│  GoShort's MCP server is 5 files, ~400 lines.        │
│  It gives Claude Code the same power as the          │
│  REST API, the CLI, and the browser combined.        │
│                                                      │
│  That's the point.                                   │
│                                                      │
╰──────────────────────────────────────────────────────╯
```

---

*Last updated: 2026-05-05*
*Companion to: [DESIGN.md](./DESIGN.md) · [LEARNING.md](./LEARNING.md)*
*GoShort MCP: `internal/mcp/` · [Official MCP Spec](https://modelcontextprotocol.io/specification/2025-11-25)*
