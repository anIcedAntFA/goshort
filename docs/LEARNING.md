# 📚 LEARNING.md — Learn Go & System Design by Building GoShort

> **By:** A Principal Software Engineer who believes the best way to learn is to build something real.
>
> **For:** Developers who know at least one other language (JS/TS/Python/Java) and want to learn Go properly — not just syntax, but *thinking in Go* — while simultaneously learning backend system design.
>
> **How to use:** Read linearly on first pass. Then use as reference while building GoShort. Each section maps to a concept you'll encounter in the codebase. "🎯 In GoShort:" annotations tell you exactly WHERE in the codebase each concept appears.

---

## Table of Contents

- [Part 0: Mindset — How Go Thinks](#part-0-mindset--how-go-thinks)
- [Part 1: Foundations — The Language](#part-1-foundations--the-language)
- [Part 2: Intermediate — Real Patterns](#part-2-intermediate--real-patterns)
- [Part 3: System Design Concepts](#part-3-system-design-concepts)
- [Part 4: Infrastructure & Deployment](#part-4-infrastructure--deployment)
- [Part 5: MCP & AI Integration](#part-5-mcp--ai-integration)
- [Part 6: GoShort Knowledge Map](#part-6-goshort-knowledge-map)
- [Part 7: Testing Deep Dive](#part-7-testing-deep-dive)
- [Part 8: Resources & Learning Path](#part-8-resources--learning-path)
- [Appendix A: Go Proverbs Applied to GoShort](#appendix-a-go-proverbs-applied-to-goshort)
- [Appendix B: Anti-Pattern Field Guide](#appendix-b-anti-pattern-field-guide)
- [Appendix C: Glossary](#appendix-c-glossary)

---

## Part 0: Mindset — How Go Thinks

### 🧘 The Go Philosophy

```
╭──────────────────────────────────────────────────────╮
│                                                      │
│   "Clear is better than clever."                     │
│                        — Rob Pike, Go Proverbs       │
│                                                      │
│   Go is BORING by design.                            │
│   And that's its superpower.                         │
│                                                      │
╰──────────────────────────────────────────────────────╯
```

Go was created at Google in 2009 by Rob Pike, Ken Thompson, and Robert Griesemer — the people who built Unix, UTF-8, and C. They were frustrated with C++ compile times and Java's complexity. They wanted a language that:

1. **Compiles fast** — seconds, not minutes
2. **Reads easy** — you spend 10x more time reading code than writing it
3. **Deploys simple** — single binary, no runtime, no dependencies
4. **Scales well** — concurrency built into the language, not bolted on

### 🧠 Mental Model Shift

If you come from another language, you need to *unlearn* some habits:

```
┌──────────────────┬─────────────────────────────────────────┐
│ Coming from...   │ What to unlearn in Go                   │
├──────────────────┼─────────────────────────────────────────┤
│ JavaScript/TS    │ No classes. No inheritance. No try/catch│
│                  │ No generics-heavy patterns (usually).   │
│                  │ Errors are VALUES, not exceptions.      │
│                  │ No event loop — goroutines instead.     │
├──────────────────┼─────────────────────────────────────────┤
│ Python           │ Static types. Explicit error handling.  │
│                  │ No duck typing (well, sort of...).      │
│                  │ Compilation required. No REPL.          │
├──────────────────┼─────────────────────────────────────────┤
│ Java/C#          │ No class hierarchy. No abstract class.  │
│                  │ Interfaces are IMPLICIT (structural).   │
│                  │ No annotations. Package = folder.       │
│                  │ No DI framework — just constructors.    │
├──────────────────┼─────────────────────────────────────────┤
│ Rust             │ No borrow checker. GC handles memory.   │
│                  │ Less type-level magic. Simpler generics.│
│                  │ More runtime, less compile-time safety. │
├──────────────────┼─────────────────────────────────────────┤
│ C/C++            │ GC (no manual memory management).       │
│                  │ No header files. Package system.        │
│                  │ Goroutines instead of pthreads.         │
└──────────────────┴─────────────────────────────────────────┘
```

### 🏔️ The Go Proverbs — Your North Star

Rob Pike delivered these at Gopherfest 2015. They're the *design principles* of the language itself.

```
 1. Don't communicate by sharing memory;
    share memory by communicating.          → Use channels, not shared vars

 2. Concurrency is not parallelism.         → Concurrency = structure.
                                              Parallelism = execution.

 3. Channels orchestrate;
    mutexes serialize.                      → Channels for data flow.
                                              Mutexes for protecting state.

 4. The bigger the interface,
    the weaker the abstraction.             → io.Reader has 1 method.
                                              That's why it's everywhere.

 5. Make the zero value useful.             → var buf bytes.Buffer
                                              // works immediately, no New()

 6. interface{} says nothing.               → Use concrete types or
                                              small, meaningful interfaces.

 7. Gofmt's style is no one's favorite,
    yet gofmt is everyone's favorite.       → Don't argue about formatting.
                                              gofmt decides. Move on.

 8. A little copying is better
    than a little dependency.               → Don't import a library for
                                              3 lines of code.

 9. Clear is better than clever.            → If you're proud of how
                                              tricky your code is, rewrite it.

10. Errors are values.                      → You can program with them.
                                              Store them. Pass them. Wrap them.

11. Don't just check errors,
    handle them gracefully.                 → Don't just `if err != nil { return err }`.
                                              Add context. Help the debugger.

12. Design the architecture,
    name the components,
    document the details.                   → Architecture > naming > docs.
                                              In that order.

13. Don't panic.                            → Return errors. Always.
                                              panic() is for truly impossible states.
```

### 🎯 The Priority Stack

When writing Go, always optimize in this order:

```
  ┌─────────────┐
  │  1. Clarity  │  ← Can someone read this without context?
  ├─────────────┤
  │ 2. Simplicity│  ← Is this the simplest way?
  ├─────────────┤
  │ 3. Concision │  ← Does every line earn its place?
  ├─────────────┤
  │4. Maintainab.│  ← Will this be easy to change in 6 months?
  ├─────────────┤
  │5. Performance│  ← Optimize LAST, with benchmarks.
  └─────────────┘
```

### 🌟 Why Go for GoShort?

```
┌──────────────────────────────────────────────────────────┐
│  GoShort needs:              Go provides:                │
├──────────────────────────────────────────────────────────┤
│  Single binary deployment  → Cross-compile, no runtime   │
│  Low-latency redirects     → Compiled, fast stdlib HTTP  │
│  CLI + Server same repo    → One language, shared code   │
│  MCP integration           → mcp-go library available    │
│  Opensource distribution   → go install, easy for users  │
│  Background cleanup jobs   → Goroutines, trivial         │
│  SQLite embedded           → modernc.org/sqlite, no CGO  │
└──────────────────────────────────────────────────────────┘
```

---

## Part 1: Foundations — The Language

### 1.1 Types & Variables

```go
// Go is statically typed. Every variable has a type at compile time.

// Declaration styles:
var name string              // explicit type, zero value ""
var age int = 25             // explicit type + value
count := 42                  // short declaration (type inferred)
// `:=` only works inside functions. `var` works everywhere.

// Zero values — Go initializes everything. No "undefined".
var i int       // 0
var s string    // ""
var b bool      // false
var p *int      // nil
var sl []int    // nil (but usable! len=0, cap=0)
```

> 💡 **ELI5:** Think of `var` as putting a labeled empty box on a shelf. Go always puts *something* in the box (zero value). `:=` is shorthand for "make a box and put this thing in it right now."

**🎯 In GoShort:** `internal/config/config.go` uses zero values as defaults. A `Config{}` with no fields set still has sensible defaults (`port = 0` means "pick default 8080 in code").

### 1.2 Structs — Go's "Objects"

```go
// Go doesn't have classes. It has structs + methods.

type URL struct {
    ID          int64
    ShortCode   string
    OriginalURL string
    IsCustom    bool
    CreatedAt   time.Time
    ExpiresAt   *time.Time  // pointer = nullable (nil means "no expiry")
    ClickCount  int64
}

// Method = function attached to a type via a "receiver"
func (u *URL) IsExpired() bool {
    if u.ExpiresAt == nil {
        return false  // no expiry set
    }
    return time.Now().After(*u.ExpiresAt)
}

// Constructor pattern (Go convention: NewXxx)
func NewURL(shortCode, originalURL string) *URL {
    return &URL{
        ShortCode:   shortCode,
        OriginalURL: originalURL,
        CreatedAt:   time.Now(),
        // Note: always use field names. Never positional.
    }
}
```

```
┌─────────────────────────────────────────────────┐
│ 🏗️ Composition over Inheritance                 │
│                                                  │
│ Java/C#:   AbstractStore → SQLStore → PGStore    │
│            (deep hierarchy, fragile)             │
│                                                  │
│ Go:        type SQLiteStorage struct {            │
│                db *sql.DB                        │
│            }                                     │
│            // implements Storage interface       │
│            // by having the right methods        │
│            (flat, flexible, no diamond problem)  │
└─────────────────────────────────────────────────┘
```

> 💡 **ELI5:** A struct is a named bag of related data. Methods are just functions that "belong to" that bag. No inheritance — you compose bags inside other bags.

**🎯 In GoShort:** `internal/shortener/service.go` defines the `URL` struct. `internal/storage/sqlite.go` has a `SQLiteStorage` struct with a `db` field. No inheritance — just interfaces.

### 1.3 Interfaces — Go's Superpower

This is the single most important concept in Go. Master this and you master Go.

```go
// An interface defines BEHAVIOR, not data.
type Storage interface {
    Create(ctx context.Context, url *URL) error
    GetByCode(ctx context.Context, code string) (*URL, error)
    Delete(ctx context.Context, code string) error
}

// ANY type that has these methods with matching signatures
// AUTOMATICALLY satisfies the interface.
// No "implements" keyword. No registration. Just... does it.

type SQLiteStorage struct { db *sql.DB }

func (s *SQLiteStorage) Create(ctx context.Context, url *URL) error { ... }
func (s *SQLiteStorage) GetByCode(ctx context.Context, code string) (*URL, error) { ... }
func (s *SQLiteStorage) Delete(ctx context.Context, code string) error { ... }
// SQLiteStorage satisfies Storage. Implicitly. No declaration needed.
```

```
╭─────────────────────────────────────────────────────╮
│ 🧩 Interface Satisfaction in Go                     │
│                                                      │
│    ┌─────────────┐       ┌──────────────────┐       │
│    │  Storage     │       │ SQLiteStorage    │       │
│    │ interface    │◄──────│ struct           │       │
│    │             │       │                  │       │
│    │ Create(...)  │       │ Create(...) ✓    │       │
│    │ GetByCode() │       │ GetByCode() ✓   │       │
│    │ Delete(...)  │       │ Delete(...) ✓    │       │
│    └─────────────┘       └──────────────────┘       │
│          ▲                                           │
│          │               ┌──────────────────┐       │
│          └───────────────│ PostgresStorage  │       │
│                          │ struct           │       │
│       implicit!          │ Create(...) ✓    │       │
│       no "implements"    │ GetByCode() ✓   │       │
│                          │ Delete(...) ✓    │       │
│                          └──────────────────┘       │
╰─────────────────────────────────────────────────────╯
```

**Key rules:**

| Rule | Why | GoShort Example |
|------|-----|-----------------|
| Define interfaces where they're USED (consumer) | The service defines Storage, not the storage package | `shortener/service.go` defines `Storage` |
| Keep interfaces SMALL (1-3 methods) | `io.Reader` has 1 method — that's why it works everywhere | `cache.Cache` has 3: `Get`, `Set`, `Delete` |
| Accept interfaces, return structs | Functions take `Storage`, return `*SQLiteStorage` | `NewService(s Storage)` not `NewService(s *SQLiteStorage)` |

**Compile-time interface check:**

```go
// This line doesn't run any code. It just ensures at compile time
// that SQLiteStorage satisfies the Storage interface.
var _ Storage = (*SQLiteStorage)(nil)

// If SQLiteStorage is missing a method, you get a compile error.
// Not a runtime error. Not a test failure. A COMPILE error.
```

> 💡 **ELI5:** An interface is like a job description. "I need someone who can `Create`, `GetByCode`, and `Delete` URLs." Anyone who can do that gets the job — no resume (explicit `implements`) needed.

**🎯 In GoShort:** Three interfaces drive the architecture:
- `storage.Storage` — database access
- `cache.Cache` — caching layer (Get/Set/Delete)
- `shortener.Encoder` — short code encoding (Encode/Decode)

### 1.4 Error Handling — Go's Most Controversial Feature

```go
// Go has NO exceptions. No try/catch. No throw.
// Functions that can fail return an error as the LAST return value.

func (s *Service) CreateURL(ctx context.Context, req CreateRequest) (*URL, error) {
    if err := validateURL(req.URL); err != nil {
        return nil, fmt.Errorf("validate: %w", err)  // wrap with context
    }

    code, err := s.encoder.Encode(s.nextID())
    if err != nil {
        return nil, fmt.Errorf("encode: %w", err)
    }

    url := &URL{ShortCode: code, OriginalURL: req.URL}
    if err := s.store.Create(ctx, url); err != nil {
        return nil, fmt.Errorf("store: %w", err)
    }

    return url, nil  // success
}
```

**The Error Decision Tree:**

```
Error encountered?
│
├─ Does the caller need to match it?
│  ├─ Yes, static message  → var ErrNotFound = errors.New("not found")
│  ├─ Yes, with data       → type ValidationError struct { Field, Msg string }
│  └─ No                   → fmt.Errorf("context: %w", err)
│
├─ At system boundary? (API handler, returning JSON)
│  └─ Use %v (not %w) to hide internal errors from clients
│
└─ How to handle:
   ├─ Return to caller (with context via %w)  ← most common
   ├─ Log and handle (top of call stack only)
   └─ NEVER: log AND return (duplicate noise)
```

**Three types of errors in GoShort:**

```go
// 1. Sentinel errors — for errors.Is() matching
var (
    ErrNotFound    = errors.New("url not found")
    ErrExpired     = errors.New("url has expired")
    ErrAliasTaken  = errors.New("custom alias already in use")
)

// Usage:
if errors.Is(err, ErrNotFound) {
    http.Error(w, "Not Found", http.StatusNotFound)
}

// 2. Custom error types — for errors.As() matching with data
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

// Usage:
var ve *ValidationError
if errors.As(err, &ve) {
    respondJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
        Code:    "invalid_" + ve.Field,
        Message: ve.Message,
    })
}

// 3. Wrapped errors — for adding context up the call stack
return fmt.Errorf("shortener.CreateURL: %w", err)
```

**Anti-patterns:**

```go
// ❌ BAD: Ignoring errors
data, _ := json.Marshal(url)

// ❌ BAD: Logging AND returning (double-handling)
if err != nil {
    slog.Error("failed", "error", err)
    return err
}

// ❌ BAD: No context (useless when debugging)
return err

// ✅ GOOD: Wrap with context
return fmt.Errorf("shortener: create url %q: %w", req.URL, err)

// ✅ GOOD: Handle at the top (API handler)
if err != nil {
    slog.Error("request failed", "error", err, "method", r.Method, "path", r.URL.Path)
    respondError(w, err)  // translates error → HTTP status
    return
}
```

> 💡 **ELI5:** Errors in Go are like receipts. Every function hands you back a receipt that says "done" or "failed because X." You must check every receipt. If it says "failed," you add a note explaining what YOU were doing, then pass it up.

**🎯 In GoShort:** Error handling follows a strict pattern:
- `internal/shortener/` defines sentinel errors (`ErrNotFound`, `ErrExpired`, `ErrAliasTaken`)
- `internal/storage/` wraps DB errors with context: `fmt.Errorf("sqlite.GetByCode: %w", err)`
- `internal/api/handler.go` is the ONLY place errors are logged — it maps errors to HTTP status codes
- `internal/api/errors.go` has a helper: `respondError(w, err)` that does `errors.Is` matching

### 1.5 Slices & Maps

```go
// Slices = dynamic arrays (most used data structure in Go)
codes := []string{"k7Xm2p", "Qp3nR", "mR8v4L"}
codes = append(codes, "new-code")  // MUST reassign!

// Pre-allocate when you know the size
urls := make([]URL, 0, 100)  // len=0, cap=100 — avoids reallocations

// Maps = hash tables
reservedPaths := map[string]bool{
    "api":     true,
    "health":  true,
    "metrics": true,
    "docs":    true,
}

// Check existence (the "comma ok" idiom):
if reserved, ok := reservedPaths[alias]; ok && reserved {
    return ErrReservedPath
}
```

**⚠️ Gotcha:** Nil slice vs empty slice:

```go
var s []int      // nil — preferred for declarations
s := []int{}     // empty, non-nil — use when JSON needs [] not null
// Both have len=0, cap=0. Both are safe to append to.
// json.Marshal: nil → null, []int{} → []
```

**🎯 In GoShort:** `internal/shortener/validator.go` uses a `map[string]bool` for reserved paths. `internal/api/handler.go` uses slices for paginated URL lists.

### 1.6 Control Flow

```go
// Go has ONE loop keyword: for
for i := 0; i < 10; i++ {}     // C-style
for condition {}                 // while
for {}                           // infinite (used in server loops)
for i, v := range slice {}      // iterate slice
for k, v := range myMap {}      // iterate map (random order!)

// if with initialization (limits scope of variables)
if err := doSomething(); err != nil {
    return err
}
// `err` doesn't exist outside this block — clean!

// switch (no fallthrough by default — unlike C!)
switch statusCode {
case http.StatusNotFound:
    respondNotFound(w)
case http.StatusGone:
    respondGone(w)
default:
    respondInternalError(w)
}

// Type switch (powerful pattern for error handling)
switch e := err.(type) {
case *ValidationError:
    respondJSON(w, 422, e)
case *NotFoundError:
    respondJSON(w, 404, e)
default:
    respondJSON(w, 500, "internal error")
}
```

**Guard Clauses — The Go Way:**

```go
// ❌ BAD: deeply nested
func (h *Handler) CreateURL(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
        if err := validate(req); err == nil {
            url, err := h.service.Create(r.Context(), req)
            if err == nil {
                respondJSON(w, 201, url)
            } else {
                respondError(w, err)
            }
        } else {
            respondError(w, err)
        }
    } else {
        respondError(w, err)
    }
}

// ✅ GOOD: flat with early returns (guard clauses)
func (h *Handler) CreateURL(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid JSON")
        return
    }

    if err := validate(req); err != nil {
        respondError(w, http.StatusUnprocessableEntity, err)
        return
    }

    url, err := h.service.Create(r.Context(), req)
    if err != nil {
        respondError(w, err)
        return
    }

    respondJSON(w, http.StatusCreated, url)
}
```

### 1.7 Pointers

```go
// A pointer holds the MEMORY ADDRESS of a value.
x := 42
p := &x    // p is *int, holds address of x
*p = 100   // dereference: change x through p
fmt.Println(x)  // 100

// Why pointers matter in Go:
// 1. Modify the original value (not a copy)
// 2. Avoid copying large structs
// 3. nil means "not set" (like null, but typed)

// Pointer receiver: modifies the struct
func (u *URL) IncrementClicks() {
    u.ClickCount++
}

// Value receiver: works on a copy (read-only)
func (u URL) IsExpired() bool {
    return u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt)
}
```

**Rule of thumb:**

```
Use POINTER receiver (*T) when:
  - Method modifies the struct
  - Struct is large (>5 fields)
  - Consistency: if ANY method needs pointer, use pointer for ALL

Use VALUE receiver (T) when:
  - Method only reads data
  - Struct is small and immutable (like time.Time)

🎯 In GoShort: ALL methods use pointer receivers
   (consistency — the struct has 7 fields)
```

> 💡 **ELI5:** A pointer is a sticky note that says "look in Box #42." Instead of giving someone a copy of what's in the box, you give them the sticky note. They can go look at the real thing — or change it.

**Nullable fields with pointers:**

```go
type URL struct {
    ExpiresAt *time.Time  // pointer = nullable
    // nil means "no expiry"
    // non-nil means "expires at this time"
}

// Check:
if url.ExpiresAt != nil {
    fmt.Println("Expires:", *url.ExpiresAt)
}
```

**🎯 In GoShort:** `ExpiresAt *time.Time` in the URL struct. In SQLite (via sqlc), this maps to `sql.NullTime`. The pointer approach is cleaner in Go code; sqlc handles the conversion.

### 1.8 Defer — Cleanup Guaranteed

```go
// defer schedules a function to run when the enclosing function returns.
// LIFO order (last defer runs first).

func (s *SQLiteStorage) GetByCode(ctx context.Context, code string) (*URL, error) {
    rows, err := s.db.QueryContext(ctx, "SELECT ... WHERE short_code = ?", code)
    if err != nil {
        return nil, fmt.Errorf("query: %w", err)
    }
    defer rows.Close()  // guaranteed cleanup, even if code below panics

    // ... process rows ...
}

// Common defer patterns:
f, err := os.Open("file.txt")
if err != nil { return err }
defer f.Close()                    // close file when function returns

mu.Lock()
defer mu.Unlock()                  // unlock mutex when function returns

ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()                     // cancel context when function returns
```

> 💡 **ELI5:** `defer` is like saying "before I leave, remind me to turn off the lights." No matter HOW you leave the room (normal exit, error, panic), the lights get turned off.

---

## Part 2: Intermediate — Real Patterns

### 2.1 Goroutines & Channels — Concurrency

Goroutines are Go's flagship feature. GoShort uses them for background cleanup.

```go
// A goroutine is a lightweight "thread" managed by Go's runtime.
// Costs ~2KB of stack (vs ~1MB for OS threads).
// You can run millions of them.
go doSomething()  // that's it. 3 characters: "go "

// But HOW DOES IT STOP? ← The most important question.
// Answer: context.Context
```

**GoShort's background cleanup goroutine:**

```go
func (s *Server) startCleanupJob(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():              // parent said "stop"
            slog.Info("cleanup stopped")
            return
        case <-ticker.C:                // timer fired
            deleted, _ := s.store.DeleteExpired(ctx, 1000)
            slog.Info("cleanup done", "deleted", deleted)
        }
    }
}

// Started in main:
go server.startCleanupJob(ctx)
```

```
╭─────────────────────────────────────────────────────╮
│ 🔄 Goroutine Lifecycle in GoShort                   │
│                                                      │
│  main() ──► server.Start(ctx)                        │
│              │                                       │
│              ├──► go startCleanupJob(ctx) ◄── hourly │
│              │        │                     cleanup  │
│              │        ├── select {                    │
│              │        │     case <-ctx.Done(): return │
│              │        │     case <-ticker.C:  clean() │
│              │        │   }                           │
│              │        │                              │
│              ├──► http.ListenAndServe(...)  ◄── HTTP  │
│              │                                server │
│              │                                       │
│              └──► signal.NotifyContext(SIGINT)        │
│                   └── Ctrl+C → cancels EVERYTHING ↑  │
╰─────────────────────────────────────────────────────╯
```

**Channels:**

```go
// Channels are typed pipes for goroutine communication.
events := make(chan Event)      // unbuffered (blocks until received)
events := make(chan Event, 10)  // buffered (10 slots before blocking)

// Send and receive
events <- Event{Type: "click"}   // send (blocks if full)
e := <-events                     // receive (blocks if empty)

// Direction in function signatures (self-documenting):
func produce(out chan<- Event) {}  // can only send
func consume(in <-chan Event) {}   // can only receive
```

**When to use what:**

```
┌─────────────────────────────────────────────────────┐
│ Need to pass data between goroutines?  → Channel    │
│ Need to protect shared state?          → sync.Mutex │
│ Need to wait for N goroutines?         → errgroup   │
│ Need to run something exactly once?    → sync.Once  │
│ Need an atomic counter?                → atomic.Int64│
│ Need to signal "stop"?                 → context    │
└─────────────────────────────────────────────────────┘
```

**🎯 In GoShort:** GoShort is simpler than a chat server — we don't need complex channel patterns. Main concurrency uses:
- Background cleanup goroutine (select + context)
- Graceful shutdown via `signal.NotifyContext`
- `go h.incrementClicks(code)` — fire-and-forget click counter

### 2.2 Context — The Cancellation Tree

```go
// context.Context carries deadlines, cancellation signals,
// and request-scoped values.
// ALWAYS the first parameter of functions that do I/O.

// Creating contexts:
ctx := context.Background()                                // root, never cancelled
ctx, cancel := context.WithCancel(parent)                   // manual cancel
ctx, cancel := context.WithTimeout(parent, 5*time.Second)   // auto-cancel after 5s
defer cancel()  // ALWAYS defer cancel immediately!

// Checking cancellation:
select {
case <-ctx.Done():
    return ctx.Err()  // context.Canceled or context.DeadlineExceeded
default:
    // keep working
}
```

```
╭─────────────────────────────────────────────────────╮
│ 🌳 Context Tree in GoShort                          │
│                                                      │
│  context.Background()                                │
│       │                                              │
│       ├── signal.NotifyContext(SIGINT, SIGTERM)       │
│       │       │                                      │
│       │       ├── HTTP server context                │
│       │       │       │                              │
│       │       │       ├── GET /:code (per-request)   │
│       │       │       ├── POST /api/v1/urls          │
│       │       │       └── DELETE /api/v1/urls/:code  │
│       │       │                                      │
│       │       ├── Cleanup goroutine context           │
│       │       │                                      │
│       │       └── MCP server context (Phase 4)       │
│       │                                              │
│       └── Ctrl+C → cancels EVERYTHING ↑              │
╰─────────────────────────────────────────────────────╯
```

**Why context matters for GoShort:**

```go
// Every HTTP request gets its own context.
// If client disconnects mid-request, ctx is cancelled.
// DB queries that use ctx will stop immediately — no wasted work.

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // per-request context

    url, err := h.store.GetByCode(ctx, code)
    // If client closed the connection, ctx.Done() fires,
    // and the DB query aborts. No zombie queries.
}
```

**🎯 In GoShort:** Context flows: main → HTTP server → per-request → storage/cache. Ctrl+C cancels the root context → all DB queries, cache calls, and the cleanup goroutine stop cleanly.

### 2.3 Packages & Modules

```
// A PACKAGE is a folder of .go files with the same `package xxx` declaration.
// A MODULE is a collection of packages with a go.mod file.

go.mod:
  module github.com/yourname/goshort
  go 1.22

// Import paths = module path + package path:
import "github.com/yourname/goshort/internal/shortener"

// internal/ = PRIVATE. Only importable within the module.
// This is enforced by the Go toolchain — not a convention, a RULE.
```

**GoShort's package dependency graph:**

```
cmd/server/main.go
├── imports: config, api, shortener, storage, cache
│
├── internal/api/              (HTTP handlers — delivery layer)
│   └── imports: shortener     (service interface only)
│
├── internal/shortener/        (business logic — service layer)
│   └── imports: nothing external (only stdlib)
│   └── DEFINES: Storage, Cache, Encoder interfaces
│
├── internal/storage/          (database — infrastructure layer)
│   └── imports: shortener     (implements Storage interface)
│
├── internal/cache/            (caching — infrastructure layer)
│   └── imports: shortener     (implements Cache interface)
│
└── internal/config/           (configuration)
    └── imports: nothing external
```

**Golden Rule:** Dependencies point INWARD. `shortener/` knows nothing about `storage/`, `cache/`, or `api/`. It defines interfaces; others implement them.

**Package naming rules:**

```
✅ shortener, storage, cache, config, api
❌ repository (too long — just "storage")
❌ helpers (too vague — name by what it DOES)
❌ utils (meaningless — spread logic into real packages)
❌ url_shortener (underscores forbidden in Go package names)
```

### 2.4 Middleware Pattern (Chi)

Middleware is the backbone of GoShort's HTTP layer. It's a chain of functions that wrap the handler.

```go
// A middleware is a function that takes a handler and returns a handler.
// type Middleware = func(http.Handler) http.Handler

func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Wrap response writer to capture status code
            ww := &responseWriter{ResponseWriter: w, statusCode: 200}

            next.ServeHTTP(ww, r)  // call the next handler

            logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.statusCode,
                "duration_ms", time.Since(start).Milliseconds(),
            )
        })
    }
}
```

**The middleware chain in GoShort:**

```
Request → [Logging] → [Metrics] → [RateLimit] → [Auth] → Handler → Response
             ↓           ↓            ↓           ↓          ↓
          log req     record      check rate   check      process
          + resp      histogram   limit       API key    request

// In Chi:
r := chi.NewRouter()
r.Use(LoggingMiddleware(logger))       // all routes
r.Use(MetricsMiddleware(prometheus))   // all routes

r.Get("/{code}", handler.Redirect)     // public — no auth

r.Route("/api/v1", func(r chi.Router) {
    r.Use(AuthMiddleware(apiKey))       // only /api/v1 routes
    r.Use(RateLimitMiddleware(limiter)) // only /api/v1 routes
    r.Post("/urls", handler.CreateURL)
    r.Get("/urls", handler.ListURLs)
    r.Delete("/urls/{code}", handler.DeleteURL)
})
```

```
╭─────────────────────────────────────────────────────╮
│ 🧅 Middleware = Onion Model                          │
│                                                      │
│  ┌────────────────────────────────────────┐          │
│  │ Logging                                │          │
│  │  ┌────────────────────────────────┐    │          │
│  │  │ Metrics                        │    │          │
│  │  │  ┌────────────────────────┐    │    │          │
│  │  │  │ Rate Limit             │    │    │          │
│  │  │  │  ┌────────────────┐    │    │    │          │
│  │  │  │  │ Auth           │    │    │    │          │
│  │  │  │  │  ┌────────┐    │    │    │    │          │
│  │  │  │  │  │ Handler│    │    │    │    │          │
│  │  │  │  │  └────────┘    │    │    │    │          │
│  │  │  │  └────────────────┘    │    │    │          │
│  │  │  └────────────────────────┘    │    │          │
│  │  └────────────────────────────────┘    │          │
│  └────────────────────────────────────────┘          │
│                                                      │
│  Request goes IN (left to right, top to bottom)      │
│  Response comes OUT (reverse order)                  │
╰─────────────────────────────────────────────────────╯
```

> 💡 **ELI5:** Middleware is like security checkpoints at an airport. Each checkpoint (logging, rate limit, auth) inspects you. If you pass all checkpoints, you reach the gate (handler). On the way out, each checkpoint can add things to your luggage (response headers, timing data).

### 2.5 Dependency Injection in Go

```go
// Go DI is simple: pass dependencies via constructors.
// No DI framework. No annotations. No magic. Just function parameters.

// Service depends on INTERFACES (not concrete types)
type Service struct {
    store   Storage   // interface
    cache   Cache     // interface
    encoder Encoder   // interface
    logger  *slog.Logger
}

func NewService(store Storage, cache Cache, enc Encoder, logger *slog.Logger) *Service {
    return &Service{
        store:   store,
        cache:   cache,
        encoder: enc,
        logger:  logger,
    }
}

// cmd/server/main.go — THE ONLY FILE that knows all concrete types
func main() {
    db, _ := storage.NewSQLite("./data/goshort.db")
    cache := cache.NewNoop()  // or cache.NewRedis(redisURL)
    encoder := shortener.NewSqidsEncoder()
    logger := slog.Default()

    svc := shortener.NewService(db, cache, encoder, logger)
    handler := api.NewHandler(svc, logger)

    // wire routes...
}
```

> 💡 **ELI5:** Imagine building a robot (Service). You don't solder the database directly onto the motherboard. You make a database port (Storage interface). At assembly time (main.go), you plug in whatever database you want — SQLite, Postgres, or a mock for tests. The robot doesn't care.

### 2.6 The Functional Options Pattern

```go
// When a constructor has optional parameters:

type ServerOption func(*Server)

func WithPort(port int) ServerOption {
    return func(s *Server) { s.port = port }
}

func WithLogger(logger *slog.Logger) ServerOption {
    return func(s *Server) { s.logger = logger }
}

func WithCache(c cache.Cache) ServerOption {
    return func(s *Server) { s.cache = c }
}

func NewServer(store storage.Storage, opts ...ServerOption) *Server {
    s := &Server{
        store:  store,
        port:   8080,          // default
        logger: slog.Default(), // default
        cache:  cache.NewNoop(), // default: no cache
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage — only specify what differs from defaults:
s := NewServer(db)                                    // all defaults
s := NewServer(db, WithPort(9090))                     // custom port
s := NewServer(db, WithPort(9090), WithCache(redis))   // custom port + cache
```

**🎯 In GoShort:** Used for `NewServer` and potentially `NewService`. Allows GoShort users to customize behavior without breaking the constructor signature.

---

## Part 3: System Design Concepts

> These aren't Go-specific — they're backend engineering concepts that GoShort teaches you by building.

### 3.1 Hashing & Encoding — The Core Algorithm

**The problem:** Turn a counter value (like 1000) into a short URL code (like "k7Xm2p").

**Base62 encoding:**

```
Charset: 0-9 (10) + a-z (26) + A-Z (26) = 62 characters

Base10:  1000
Base62:  "g8"

How? Same as converting decimal to hex, but with base 62:
  1000 ÷ 62 = 16 remainder 8  → charset[8]  = '8'
  16   ÷ 62 = 0  remainder 16 → charset[16] = 'g'
  Result: "g8"

Space: 62^6 = 56,800,235,584 possible 6-char codes
       That's ~56.8 BILLION — plenty for any URL shortener.
```

**The Birthday Paradox — Why Random Codes Fail:**

```
╭──────────────────────────────────────────────────────────╮
│ 🎂 Birthday Paradox                                      │
│                                                           │
│ Question: How many people in a room before there's a      │
│           50% chance two share a birthday?                 │
│                                                           │
│ Intuition says: ~183 (half of 365)                        │
│ Math says:      23 !!!                                    │
│                                                           │
│ Same applies to random short codes:                       │
│                                                           │
│ With 62^6 = 56.8B possible codes:                         │
│   At 280K codes: 50% chance of collision                  │
│   At 1M codes:   99.6% chance of collision                │
│                                                           │
│ Formula: P(collision) ≈ 1 - e^(-n² / 2m)                 │
│   n = number of codes generated                           │
│   m = total code space                                    │
│                                                           │
│ This is why GoShort uses a COUNTER, not random codes.     │
╰──────────────────────────────────────────────────────────╯
```

**Sqids — Making Counters Non-Predictable:**

```
Problem: Counter 1000 → Base62 "g8", 1001 → "g9"
         Users can guess the next code! Enumeration attack.

Solution: Sqids adds a permutation layer:
         Counter 1000 → Sqids → "k7Xm"
         Counter 1001 → Sqids → "Qp3n"
         Counter 1002 → Sqids → "mR8v"

Properties:
  - Bijective: every input → exactly one output, reversible
  - Non-sequential: adjacent inputs → very different outputs
  - Deterministic: same input always → same output
  - O(1): constant time, pure math
```

**🎯 In GoShort:** `internal/shortener/encoder.go` wraps Sqids:

```go
type SqidsEncoder struct {
    sqid *sqids.Sqids
}

func (e *SqidsEncoder) Encode(id int64) (string, error) {
    code, err := e.sqid.Encode([]uint64{uint64(id)})
    if err != nil {
        return "", fmt.Errorf("sqids encode: %w", err)
    }
    return code, nil
}

func (e *SqidsEncoder) Decode(code string) (int64, error) {
    ids := e.sqid.Decode(code)
    if len(ids) == 0 {
        return 0, ErrInvalidCode
    }
    return int64(ids[0]), nil
}
```

### 3.2 Caching Patterns

**Cache-Aside (Lazy Loading) — GoShort's Pattern:**

```
┌─────────┐  1. GET short:k7Xm2p  ┌─────────┐
│         │ ─────────────────────▶ │         │
│ Handler │                        │  Cache  │
│         │ ◀───────────────────── │         │
│         │  2a. HIT → return      └─────────┘
│         │
│         │  2b. MISS → query DB
│         │                        ┌─────────┐
│         │ ─────────────────────▶ │   DB    │
│         │ ◀───────────────────── │         │
│         │  3. Populate cache     └─────────┘
└─────────┘  4. Return result
```

**Cache stampede (thundering herd) — The Problem:**

```
Scenario: A hot URL's cache entry expires.
          100 simultaneous requests arrive.
          ALL get cache miss.
          ALL query the database.
          Database gets hammered.

╭────────────────────────────────────────────╮
│     Time ──►                               │
│                                            │
│  Req 1: cache MISS → DB query ─────────▶  │
│  Req 2: cache MISS → DB query ─────────▶  │
│  Req 3: cache MISS → DB query ─────────▶  │
│  ...                                       │
│  Req 100: cache MISS → DB query ────────▶  │
│                                            │
│  100 identical queries! 99 are wasted.     │
╰────────────────────────────────────────────╯
```

**Solution: singleflight**

```go
import "golang.org/x/sync/singleflight"

var group singleflight.Group

func (h *Handler) getURL(ctx context.Context, code string) (*URL, error) {
    // If 100 requests come in for the same code simultaneously,
    // only ONE actually queries the DB. The other 99 wait and
    // get the same result.
    v, err, _ := group.Do(code, func() (interface{}, error) {
        return h.store.GetByCode(ctx, code)
    })
    if err != nil {
        return nil, err
    }
    return v.(*URL), nil
}
```

```
╭────────────────────────────────────────────╮
│  With singleflight:                        │
│                                            │
│  Req 1: cache MISS → DB query ─────────▶  │
│  Req 2: cache MISS → wait for Req 1...    │
│  Req 3: cache MISS → wait for Req 1...    │
│  ...                                       │
│  Req 100: cache MISS → wait for Req 1...  │
│                                            │
│  Req 1 completes → ALL 100 get the result  │
│  1 query instead of 100!                   │
╰────────────────────────────────────────────╯
```

**TTL Alignment — Preventing Stale Cache:**

```go
func cacheTTL(url *URL) time.Duration {
    if url.ExpiresAt == nil {
        return 24 * time.Hour   // permanent URL → cache 24h
    }
    remaining := time.Until(*url.ExpiresAt)
    if remaining <= 0 {
        return 0                // expired → don't cache
    }
    if remaining > 24*time.Hour {
        return 24 * time.Hour   // cap at 24h
    }
    return remaining            // cache exactly until expiry
}
```

> 💡 **ELI5:** Imagine a library book. The library says "return by June 1." If you photocopy it (cache it), your photocopy should also have "valid until June 1" written on it. Otherwise you might keep reading an expired copy while the library already pulled the book.

### 3.3 Database Indexing — Why B-Trees Matter

```
Without index: Find short_code = "k7Xm2p"
  → Scan every row: O(n)
  → At 1M rows: ~100ms
  → At 1B rows: minutes

With B-tree index: Find short_code = "k7Xm2p"
  → Binary search through balanced tree: O(log n)
  → At 1M rows: ~0.1ms
  → At 1B rows: ~1ms
```

**How a B-tree works (simplified):**

```
                    ┌─────────┐
                    │  g-m-s  │         ← root node
                    └─┬─┬─┬──┘
                   ╱   │   ╲
          ┌──────┐ ┌──────┐ ┌──────┐
          │ a-c-e│ │ h-j-k│ │ n-p-r│   ← internal nodes
          └──┬───┘ └──┬───┘ └──┬───┘
            ...      ...      ...       ← leaf nodes (actual data)

Looking for "k7Xm2p":
  1. Root: k > g, k < m → go middle
  2. Internal: k > j, k = k → FOUND!
  3. Total: 2-3 lookups, regardless of table size
```

**EXPLAIN QUERY PLAN — Verify Your Index Works:**

```sql
-- Always check that your queries use the index!
EXPLAIN QUERY PLAN
SELECT * FROM urls WHERE short_code = 'k7Xm2p';

-- Good output:
-- SEARCH TABLE urls USING INDEX idx_short_code (short_code=?)

-- Bad output (full scan!):
-- SCAN TABLE urls
```

**Partial index for expiration cleanup:**

```sql
-- Only index rows that HAVE an expiration
-- (most URLs don't expire → smaller, faster index)
CREATE INDEX idx_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
```

**🎯 In GoShort:** `db/schema.sql` defines both indexes. Run `EXPLAIN QUERY PLAN` on your sqlc-generated queries to verify they use indexes.

### 3.4 Rate Limiting Algorithms

**Token Bucket — GoShort's Choice:**

```
╭─────────────────────────────────────────────╮
│ 🪣 Token Bucket Algorithm                    │
│                                              │
│  Bucket capacity: 60 tokens                  │
│  Refill rate: 1 token per second             │
│                                              │
│  ┌──────────────────────────────┐            │
│  │ ●●●●●●●●●●●●●●●●●●●●●●●●●● │ 60 tokens  │
│  └──────────────────────────────┘            │
│                                              │
│  Request arrives:                            │
│    → tokens > 0? Take one. Process request.  │
│    → tokens = 0? Reject: 429 Too Many Req.   │
│                                              │
│  Every second: add 1 token (up to max 60)    │
│                                              │
│  Burst: allows 60 rapid requests,            │
│         then throttles to 1/sec              │
╰─────────────────────────────────────────────╯
```

**Implementation using `golang.org/x/time/rate`:**

```go
import "golang.org/x/time/rate"

func RateLimitMiddleware(rps int) func(http.Handler) http.Handler {
    // Per-IP rate limiters
    limiters := make(map[string]*rate.Limiter)
    var mu sync.Mutex

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr

            mu.Lock()
            if _, ok := limiters[ip]; !ok {
                limiters[ip] = rate.NewLimiter(rate.Limit(rps), rps) // rps tokens/sec, burst = rps
            }
            limiter := limiters[ip]
            mu.Unlock()

            if !limiter.Allow() {
                w.Header().Set("Retry-After", "1")
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Token Bucket vs Sliding Window:**

```
┌────────────────────┬─────────────────────────────────┐
│ Token Bucket       │ Sliding Window                  │
├────────────────────┼─────────────────────────────────┤
│ Allows bursts      │ Strictly even rate              │
│ Simpler to impl    │ More complex                    │
│ stdlib available   │ Custom implementation           │
│ Memory: O(1)/user  │ Memory: O(window_size)/user     │
│ ✅ GoShort uses     │ Good for API billing            │
└────────────────────┴─────────────────────────────────┘
```

### 3.5 HTTP Redirect Mechanics

```
┌───────────────────────────────────────────────────────┐
│ What actually happens when you visit a short URL:     │
│                                                       │
│  1. Browser: GET https://goshort.dev/k7Xm2p           │
│     ↓                                                 │
│  2. DNS: goshort.dev → Cloudflare edge (CDN check)    │
│     ↓                                                 │
│  3. CDN: Cache HIT? → redirect from edge (<10ms)      │
│          Cache MISS? → forward to origin               │
│     ↓                                                 │
│  4. Origin: Go server receives request                 │
│     → Chi router matches /{code}                       │
│     → handler.Redirect(w, r)                           │
│     → cache.Get("short:k7Xm2p")                       │
│     → MISS? → store.GetByCode("k7Xm2p")               │
│     → check expiry                                     │
│     → cache.Set("short:k7Xm2p", url, ttl)             │
│     → w.Header().Set("Location", url)                  │
│     → w.WriteHeader(302)                               │
│     ↓                                                 │
│  5. Browser receives 302 + Location header             │
│     → automatically navigates to the long URL          │
│     ↓                                                 │
│  6. User sees the destination page.                    │
│     Total time: 10-50ms                                │
└───────────────────────────────────────────────────────┘
```

**Why 302 (Temporary), not 301 (Permanent)?**

```
301: Browser caches the redirect FOREVER.
     Next time user visits → browser goes directly to long URL.
     Your server NEVER sees the request again.
     → Can't track clicks
     → Can't expire URLs
     → Can't change destination
     → Can't rate limit repeat visitors

302: Browser asks your server EVERY time.
     → You see every click
     → You can expire, update, delete
     → Slightly slower (one extra hop) but negligible with caching
```

### 3.6 SQL with sqlc — Type-Safe Database Access

**The workflow:**

```
1. Write SQL (schema + queries)    ← you write this
2. Run: sqlc generate              ← sqlc reads SQL
3. Get: type-safe Go code          ← sqlc writes this
4. Use: generated functions        ← you call this
```

**Example:**

```sql
-- db/queries.sql

-- name: GetByCode :one
SELECT id, short_code, original_url, is_custom, created_at, expires_at, click_count
FROM urls
WHERE short_code = ?;
```

sqlc generates:

```go
// db/query.sql.go (auto-generated — DO NOT EDIT)

type GetByCodeRow struct {
    ID          int64
    ShortCode   string
    OriginalURL string
    IsCustom    int64
    CreatedAt   string
    ExpiresAt   sql.NullString
    ClickCount  int64
}

func (q *Queries) GetByCode(ctx context.Context, shortCode string) (GetByCodeRow, error) {
    row := q.db.QueryRowContext(ctx, getByCode, shortCode)
    var i GetByCodeRow
    err := row.Scan(&i.ID, &i.ShortCode, &i.OriginalURL, ...)
    return i, err
}
```

**Why sqlc over GORM/raw database/sql?**

```
┌──────────────┬──────────────────────────────────────┐
│ Raw database/sql │ Manual row scanning — error-prone │
│                  │ No compile-time type checking     │
│                  │ Typos in SQL found at RUNTIME     │
├──────────────────┼──────────────────────────────────┤
│ GORM (ORM)       │ Hides SQL — hard to optimize     │
│                  │ Reflection-heavy — 2x slower     │
│                  │ Magic behaviors (auto-migrate)   │
├──────────────────┼──────────────────────────────────┤
│ sqlc             │ You write REAL SQL                │
│                  │ sqlc checks SQL at COMPILE time   │
│                  │ Generated Go is type-safe         │
│                  │ Performance = raw database/sql    │
│                  │ No magic, no reflection           │
│                  │ ✅ GoShort uses this               │
└──────────────────┴──────────────────────────────────┘
```

### 3.7 API Design Principles

**REST Conventions GoShort Follows:**

```
Nouns for resources, not verbs:
  ✅ POST /api/v1/urls          (create a URL)
  ❌ POST /api/v1/shorten       (verb — anti-pattern)

HTTP methods carry the verb:
  POST   = Create
  GET    = Read
  PUT    = Full update
  PATCH  = Partial update
  DELETE = Delete

Status codes mean something:
  201 Created        → POST success
  204 No Content     → DELETE success
  400 Bad Request    → client sent garbage
  404 Not Found      → resource doesn't exist
  409 Conflict       → alias already taken
  410 Gone           → URL expired
  422 Unprocessable  → valid JSON, invalid data
  429 Too Many Reqs  → rate limited

Consistent error format:
  {
    "error": {
      "code": "alias_taken",           ← machine-readable
      "message": "The alias is taken", ← human-readable
      "details": { "alias": "my-link" }← contextual data
    }
  }
```

**API Versioning:**

```
/api/v1/urls  ← version in URL path
              ← simple, explicit, cacheable
              ← when v2 breaks changes, old clients still work on v1

Alternative (not used): Accept header versioning
  Accept: application/vnd.goshort.v1+json
  ← more "RESTful" but harder to test with curl
```

---

## Part 4: Infrastructure & Deployment

### 4.1 Docker — Containerization

**What is Docker?**

```
Without Docker:
  "It works on my machine" → "Well, your machine isn't production"

With Docker:
  Your app + its environment = a portable box (container)
  Same box runs the same way everywhere.
```

**GoShort's Dockerfile (multi-stage build):**

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download              # cache dependencies
COPY . .
RUN CGO_ENABLED=0 go build -o goshort ./cmd/server

# Stage 2: Run (tiny image — no Go compiler, no source code)
FROM alpine:3.19
COPY --from=builder /app/goshort /usr/local/bin/goshort
EXPOSE 8080
CMD ["goshort", "serve"]
```

**Why multi-stage?**

```
Stage 1 image: ~1.2 GB (Go compiler + source + deps)
Stage 2 image: ~15 MB  (just the binary + Alpine Linux)

Production doesn't need the compiler!
```

**Layer ordering matters:**

```dockerfile
# ❌ BAD: COPY source before go mod download
COPY . .
RUN go mod download
RUN go build ...
# Any source change → re-download ALL dependencies

# ✅ GOOD: Download deps first, then copy source
COPY go.mod go.sum ./
RUN go mod download          # cached unless go.mod changes
COPY . .
RUN go build ...
# Source changes → only rebuild, deps are cached
```

**🎯 In GoShort:** The Dockerfile is in the project root. `docker build -t goshort .` builds the image. `docker-compose up` runs GoShort + Caddy (reverse proxy) + optional Redis.

### 4.2 Reverse Proxy — Why Nginx/Caddy Sits in Front

```
Without reverse proxy:
  Client → :8080 → Go app
  Problems: no TLS, no buffering, no rate limiting at network level

With reverse proxy:
  Client → :443 → Nginx/Caddy → :8080 → Go app
  Benefits: TLS termination, request buffering, static files, rate limiting
```

```
┌──────────┐     HTTPS     ┌──────────────┐     HTTP      ┌──────────┐
│  Client  │ ────────────▶ │ Nginx/Caddy  │ ────────────▶ │ Go App   │
│ (browser)│ ◀──────────── │ (port 443)   │ ◀──────────── │ (port    │
└──────────┘               │              │               │  8080)   │
                           │ • TLS (HTTPS)│               └──────────┘
                           │ • Buffering  │
                           │ • Access logs│
                           │ • Static files│
                           └──────────────┘
```

**TLS Termination — What It Means:**

```
TLS (HTTPS) = encryption between client and server.
"Termination" = decryption happens at the proxy.

Client ──── encrypted ────▶ Nginx ──── plain HTTP ────▶ Go App

Why? Go CAN do TLS itself, but:
  - Nginx/Caddy handle certificate renewal automatically
  - Caddy does it with zero config (Let's Encrypt built-in)
  - Go app stays simple — just HTTP
```

**Caddy vs Nginx:**

```
Caddy config for GoShort:                Nginx config for GoShort:
┌────────────────────────┐               ┌──────────────────────────────┐
│ goshort.dev {          │               │ server {                     │
│   reverse_proxy :8080  │               │   listen 443 ssl;            │
│ }                      │               │   server_name goshort.dev;   │
│                        │               │   ssl_certificate /etc/...;  │
│ # That's it.           │               │   ssl_certificate_key /...;  │
│ # Auto HTTPS.          │               │   location / {               │
│ # Auto cert renewal.   │               │     proxy_pass :8080;        │
└────────────────────────┘               │     proxy_set_header ...;    │
                                         │   }                          │
2 lines.                                 │ }                            │
                                         └──────────────────────────────┘
                                         ~15 lines + Certbot setup.
```

### 4.3 DNS & CDN — How Cloudflare Works

```
User types: goshort.dev/k7Xm2p

Step 1: DNS Resolution
  Browser → DNS resolver → Cloudflare DNS
  "What IP is goshort.dev?"
  Without Cloudflare proxy: → your server IP (168.119.23.45)
  With Cloudflare proxy:    → Cloudflare edge IP (104.21.x.x)

Step 2: CDN Edge (with Cloudflare proxy enabled)
  Browser → Cloudflare edge server (Singapore, closest to user)
  Edge checks: is /k7Xm2p cached?
    → Cache HIT:  return cached 302 redirect (<10ms!)
    → Cache MISS: forward request to your origin server

Step 3: Origin Server (your Go app)
  Cloudflare → your server → Go app processes → returns 302
  Cloudflare caches the response at the edge for next time
```

```
╭─────────────────────────────────────────────────────╮
│ 🌍 CDN Edge Caching                                 │
│                                                      │
│  User (Saigon)                                       │
│       │                                              │
│       ▼                                              │
│  Cloudflare Edge (Singapore)  ← closest edge server  │
│       │                                              │
│       ├── Cache HIT?  → return cached redirect       │
│       │                  (< 10ms, server never hit)  │
│       │                                              │
│       └── Cache MISS? → forward to origin            │
│               │                                      │
│               ▼                                      │
│          Your Server (Frankfurt)                     │
│          Go app processes, returns 302               │
│          Cloudflare caches response at edge          │
╰─────────────────────────────────────────────────────╯
```

### 4.4 Networking Basics for Deployment

```
Concepts you need for VPS deployment:

PORT:
  A number (0-65535) that identifies a service on a machine.
  HTTP = 80, HTTPS = 443, SSH = 22.
  Your Go app listens on 8080 (convention for non-root).
  Nginx/Caddy listens on 80/443 (requires root or capabilities).

FIREWALL (UFW):
  Controls which ports accept incoming connections.
  ufw allow 22   → SSH (so you can connect)
  ufw allow 80   → HTTP
  ufw allow 443  → HTTPS
  ufw deny 8080  → block direct access to Go app (must go through proxy)

PUBLIC IP:
  Your VPS has a public IP (like 168.119.23.45).
  Anyone on the internet can reach it on allowed ports.
  DNS maps your domain to this IP.

NAT (Network Address Translation):
  Docker uses NAT to map container ports to host ports.
  docker -p 8080:8080 means: host:8080 → container:8080
```

### 4.5 systemd — Process Management

```
systemd keeps your Go app running. If it crashes, systemd restarts it.

/etc/systemd/system/goshort.service:
┌─────────────────────────────────────────────────┐
│ [Unit]                                          │
│ Description=GoShort URL Shortener               │
│ After=network.target                            │
│                                                 │
│ [Service]                                       │
│ Type=simple                                     │
│ User=goshort                                    │
│ ExecStart=/usr/local/bin/goshort serve           │
│ Restart=on-failure                              │
│ RestartSec=5                                    │
│ Environment=GOSHORT_DATA_DIR=/var/lib/goshort    │
│                                                 │
│ [Install]                                       │
│ WantedBy=multi-user.target                      │
└─────────────────────────────────────────────────┘

Commands:
  systemctl start goshort     # start
  systemctl stop goshort      # stop
  systemctl restart goshort   # restart
  systemctl status goshort    # check status
  systemctl enable goshort    # start on boot
  journalctl -u goshort -f    # follow logs
```

---

## Part 5: MCP & AI Integration

### 5.1 What is MCP?

```
MCP = Model Context Protocol
A standard for AI models to interact with external tools.

Think of it as: USB for AI.
  - USB defines how peripherals (keyboard, mouse) talk to computers.
  - MCP defines how AI models (Claude, Cursor) talk to tools (GoShort).

Without MCP:
  User → Claude: "shorten this URL"
  Claude: "I can't do that. Here's how you might..."

With MCP:
  User → Claude: "shorten this URL"
  Claude → GoShort MCP: tool_call("shorten_url", {url: "..."})
  GoShort → Claude: {short_url: "https://goshort.dev/k7Xm2p"}
  Claude → User: "Here's your short URL: https://goshort.dev/k7Xm2p"
```

### 5.2 MCP Architecture

```
┌──────────────┐     stdio (JSON-RPC)     ┌──────────────┐
│  Claude Code │ ◄──────────────────────▶ │  GoShort     │
│  or Cursor   │                          │  MCP Server  │
│  (AI Client) │                          │  (Go binary) │
└──────────────┘                          └──────┬───────┘
                                                 │
                                          calls GoShort API
                                          (same binary, in-process)
                                                 │
                                          ┌──────▼───────┐
                                          │   SQLite DB  │
                                          └──────────────┘
```

**Transport:** stdio (standard input/output). The AI client spawns the GoShort binary and communicates via JSON-RPC over stdin/stdout. No network, no HTTP, no auth needed — it's a local pipe.

### 5.3 Tool Definitions

```go
// Using github.com/mark3labs/mcp-go

import "github.com/mark3labs/mcp-go/server"

func registerTools(s *server.Server, svc *shortener.Service) {
    s.AddTool("shorten_url", "Create a short URL", shortenHandler(svc),
        server.WithString("url", "The URL to shorten", true),
        server.WithString("alias", "Custom alias (optional)", false),
        server.WithString("expires_in", "Expiration (e.g., '7d', '30d')", false),
    )

    s.AddTool("list_urls", "List all short URLs", listHandler(svc),
        server.WithNumber("page", "Page number", false),
        server.WithNumber("per_page", "Results per page", false),
    )

    s.AddTool("get_stats", "Get URL click stats", statsHandler(svc),
        server.WithString("code", "The short code to look up", true),
    )

    s.AddTool("delete_url", "Delete a short URL", deleteHandler(svc),
        server.WithString("code", "The short code to delete", true),
    )
}
```

**🎯 In GoShort:** `internal/mcp/server.go` registers tools. The MCP server reuses the same `shortener.Service` as the HTTP API — same business logic, different delivery mechanism.

### 5.4 Claude Code / Cursor Integration

```json
// .mcp.json (in project root)
{
  "mcpServers": {
    "goshort": {
      "command": "./goshort",
      "args": ["mcp"],
      "env": {
        "GOSHORT_DATA_DIR": "./data"
      }
    }
  }
}
```

After this, Claude Code can: "Shorten this URL: https://example.com/very/long/path" → automatically calls `shorten_url` tool → returns short URL.

---

## Part 6: GoShort Knowledge Map

Every file in GoShort teaches specific concepts. Use this to know what you'll learn.

```
╭──────────────────────────────────────────────────────────────────╮
│  File / Package                │ Concepts You'll Learn           │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/shortener/            │                                 │
│   service.go                   │ Interfaces, DI, error handling  │
│   encoder.go                   │ Sqids, bijective functions      │
│   validator.go                 │ Regex, guard clauses, sentinel  │
│                                │ errors, custom error types      │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/storage/              │                                 │
│   storage.go                   │ Interface definition            │
│   sqlite.go                    │ sqlc, context, SQL indexing,    │
│                                │ WAL mode, connection pooling    │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/cache/                │                                 │
│   cache.go                     │ Interface (Get/Set/Delete)      │
│   noop.go                      │ Null object pattern             │
│   memory.go                    │ sync.Map, TTL, in-process cache │
│   redis.go                     │ Redis client, serialization     │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/api/                  │                                 │
│   router.go                    │ Chi, route groups, versioning   │
│   handler.go                   │ HTTP handlers, JSON, status     │
│   middleware.go                │ Auth, rate limit, logging chain │
│   errors.go                    │ Error → HTTP status mapping     │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/config/               │                                 │
│   config.go                    │ TOML, env vars, koanf, defaults │
├────────────────────────────────┼─────────────────────────────────┤
│ internal/mcp/                  │                                 │
│   server.go                    │ MCP protocol, JSON-RPC, stdio   │
│   tools.go                     │ Tool definitions, AI integration│
├────────────────────────────────┼─────────────────────────────────┤
│ cmd/server/main.go             │ Wiring, DI, graceful shutdown,  │
│                                │ signal handling, context tree   │
├────────────────────────────────┼─────────────────────────────────┤
│ cmd/cli/main.go                │ Cobra, CLI UX, HTTP client      │
├────────────────────────────────┼─────────────────────────────────┤
│ db/schema.sql                  │ SQL DDL, indexes, constraints   │
│ db/queries.sql                 │ SQL DML, joins, aggregation     │
│ db/sqlc.yaml                   │ Code generation config          │
├────────────────────────────────┼─────────────────────────────────┤
│ Dockerfile                     │ Multi-stage builds, layers      │
│ docker-compose.yml             │ Multi-service orchestration     │
│ Caddyfile                      │ Reverse proxy, auto TLS         │
├────────────────────────────────┼─────────────────────────────────┤
│ docs/openapi.yaml              │ OpenAPI spec, API documentation │
│ docs/design.md                 │ System design, trade-offs       │
╰────────────────────────────────┴─────────────────────────────────╯
```

---

## Part 7: Testing Deep Dive

### 7.1 TDD Workflow

```
╭──────────────────────────────────────────╮
│         TDD: Red → Green → Refactor      │
│                                          │
│  1. RED:    Write a failing test          │
│             (function doesn't exist yet)  │
│                                          │
│  2. GREEN:  Write minimum code to pass    │
│             (ugly is OK at this step)     │
│                                          │
│  3. REFACTOR: Clean up, keep tests green  │
│             (make it beautiful)           │
│                                          │
│  Repeat for next behavior.               │
╰──────────────────────────────────────────╯
```

**Example: TDD for ValidateAlias**

```go
// Step 1: RED — write the test FIRST
func TestValidateAlias(t *testing.T) {
    tests := []struct {
        name    string
        alias   string
        wantErr bool
    }{
        {"valid simple",         "my-link",      false},
        {"valid alphanumeric",   "abc123",       false},
        {"too short",            "ab",           true},
        {"too long",             strings.Repeat("a", 31), true},
        {"has spaces",           "my link",      true},
        {"starts with hyphen",   "-foo",         true},
        {"ends with hyphen",     "foo-",         true},
        {"reserved path",        "api",          true},
        {"reserved health",      "health",       true},
        {"has special chars",    "my@link",      true},
        {"empty",                "",             true},
        {"single char",          "a",            true},
        {"exactly 3 chars",      "abc",          false},  // boundary
        {"exactly 30 chars",     strings.Repeat("a", 30), false},  // boundary
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateAlias(tt.alias)  // doesn't exist yet → RED!
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateAlias(%q) error = %v, wantErr %v",
                    tt.alias, err, tt.wantErr)
            }
        })
    }
}

// Step 2: GREEN — write minimum code to pass
func ValidateAlias(alias string) error {
    if len(alias) < 3 || len(alias) > 30 {
        return &ValidationError{Field: "alias", Message: "must be 3-30 characters"}
    }
    if !aliasRegexp.MatchString(alias) {
        return &ValidationError{Field: "alias", Message: "invalid characters"}
    }
    if reservedPaths[alias] {
        return &ValidationError{Field: "alias", Message: "reserved path"}
    }
    return nil
}

// Step 3: REFACTOR — extract regex, improve error messages, etc.
```

### 7.2 Test Types in GoShort

```
┌─────────────────┬───────────────────────────────────────────────┐
│ Test Type       │ What / When / How                             │
├─────────────────┼───────────────────────────────────────────────┤
│ Unit Tests      │ Test 1 function in isolation.                 │
│                 │ Fast. No I/O. Mock dependencies.              │
│                 │ Example: ValidateAlias, Encode, Decode        │
│                 │ Run: go test ./internal/shortener/            │
├─────────────────┼───────────────────────────────────────────────┤
│ Table-Driven    │ Multiple inputs for 1 function.               │
│                 │ Go idiom: []struct + t.Run.                   │
│                 │ All unit tests should be table-driven.        │
├─────────────────┼───────────────────────────────────────────────┤
│ Integration     │ Test service + real SQLite DB together.        │
│                 │ Uses t.TempDir() for throwaway DB.             │
│                 │ Example: CreateURL → GetByCode → DeleteByCode │
│                 │ Run: go test ./internal/storage/              │
├─────────────────┼───────────────────────────────────────────────┤
│ HTTP Tests      │ Test API endpoints with httptest.              │
│                 │ Real HTTP, fake dependencies.                 │
│                 │ Example: POST /api/v1/urls → 201              │
│                 │ Run: go test ./internal/api/                  │
├─────────────────┼───────────────────────────────────────────────┤
│ Fuzz Tests      │ Random inputs to find edge cases.              │
│                 │ Go 1.18+ built-in.                            │
│                 │ Example: FuzzValidateURL, FuzzEncode          │
│                 │ Run: go test -fuzz=FuzzValidateURL ./...      │
├─────────────────┼───────────────────────────────────────────────┤
│ Benchmark       │ Measure performance.                          │
│                 │ Example: BenchmarkRedirect, BenchmarkEncode   │
│                 │ Run: go test -bench=. -benchmem ./...         │
└─────────────────┴───────────────────────────────────────────────┘
```

### 7.3 HTTP Testing with httptest

```go
func TestCreateURL(t *testing.T) {
    // Setup
    store := storage.NewSQLite(t.TempDir() + "/test.db")
    svc := shortener.NewService(store, cache.NewNoop(), shortener.NewSqidsEncoder())
    handler := api.NewHandler(svc, slog.Default())
    router := api.NewRouter(handler)

    // Create request
    body := `{"url": "https://example.com/long", "custom_alias": "test-link"}`
    req := httptest.NewRequest("POST", "/api/v1/urls", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    // Execute
    router.ServeHTTP(rec, req)

    // Assert
    if rec.Code != http.StatusCreated {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
    }

    var resp CreateURLResponse
    json.NewDecoder(rec.Body).Decode(&resp)
    if resp.ShortCode != "test-link" {
        t.Errorf("short_code = %q, want %q", resp.ShortCode, "test-link")
    }
}
```

### 7.4 Fuzz Testing

```go
func FuzzValidateURL(f *testing.F) {
    // Seed corpus: known inputs
    f.Add("https://example.com")
    f.Add("")
    f.Add("not-a-url")
    f.Add("ftp://files.example.com")
    f.Add("javascript:alert(1)")  // XSS attempt

    f.Fuzz(func(t *testing.T, input string) {
        // Should NEVER panic, regardless of input
        err := ValidateURL(input)
        if err == nil {
            // If valid, it must start with http:// or https://
            if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
                t.Errorf("accepted non-HTTP URL: %q", input)
            }
        }
    })
}
```

### 7.5 What to Test, What NOT to Test

```
✅ TEST:
  - Business logic (validation, encoding, expiration checks)
  - Error paths (what happens when DB fails, cache misses)
  - Edge cases (empty input, max length, boundary values)
  - HTTP handlers (status codes, response format, error responses)
  - Integration (service + real DB in t.TempDir())

❌ DON'T TEST:
  - Go stdlib (json.Marshal works — Go already tested it)
  - sqlc-generated code (sqlc already validates SQL)
  - Third-party libraries (Sqids encodes correctly — they tested it)
  - Trivial getters/setters (no logic = no bugs = no test needed)
  - Private implementation details (test behavior, not structure)
```

---

## Part 8: Resources & Learning Path

### 📅 Learning Path Aligned with GoShort Phases

```
Phase 1 (Weeks 1-2): Core Library + Storage
├── go.dev/tour                      ← interactive, free, 2 hours
├── Go by Example (gobyexample.com)  ← quick syntax reference
├── Effective Go (go.dev/doc)        ← idiomatic patterns
├── sqlc docs (docs.sqlc.dev)        ← learn sqlc
└── 🔨 Build: shortener/, storage/, tests (TDD)

Phase 2 (Weeks 3-5): HTTP API + Scaling Reads
├── "Let's Go" — Alex Edwards       ← best Go web book, period
├── Chi docs (go-chi.io)            ← router patterns
├── Redis University (university.redis.io) ← free, interactive
├── Cloudflare docs                  ← CDN setup
└── 🔨 Build: api/, cache/, middleware, monitoring

Phase 3 (Weeks 6-7): Auth + CLI + Deploy
├── Cobra docs (cobra.dev)           ← CLI framework
├── Docker getting started           ← containerization
├── DigitalOcean tutorials           ← VPS, Nginx, Certbot
└── 🔨 Build: CLI, Docker, deploy to Fly.io + bare VPS

Phase 4 (Week 8): MCP Server
├── MCP spec (modelcontextprotocol.io)    ← protocol reference
├── mcp-go examples                       ← Go implementation
└── 🔨 Build: MCP server, connect to Claude Code

Phase 5+ (Ongoing): Advanced
├── "100 Go Mistakes" — Teiva Harsanyi   ← avoid footguns
├── "Concurrency in Go" — Cox-Buday      ← deep dive goroutines
├── "The Go Programming Language"        ← the Go bible
├── Go blog (go.dev/blog)                ← official deep-dives
└── 🔨 Build: analytics, AI agent, Postgres, replication
```

### 📖 Top Resources (Ranked by Impact)

**Books:**

| # | Book | Level | Why |
|---|------|-------|-----|
| 1 | *Let's Go* — Alex Edwards | Beginner | Best practical intro. Build a real web app. |
| 2 | *100 Go Mistakes* — Teiva Harsanyi | Intermediate | Learn from others' bugs. Every chapter = 1 mistake. |
| 3 | *The Go Programming Language* — Donovan & Kernighan | All | The Go bible. Dense but complete. |
| 4 | *Concurrency in Go* — Katherine Cox-Buday | Intermediate | Master goroutines and channels. |
| 5 | *Learning Go* — Jon Bodner | Beginner | Modern, covers generics. O'Reilly. |

**Videos:**

| Talk | Speaker | Why Watch |
|------|---------|-----------|
| Go Proverbs | Rob Pike | Design philosophy in 20 minutes |
| Simplicity is Complicated | Rob Pike | Why Go chose boring |
| Concurrency is not Parallelism | Rob Pike | Core mental model |
| The Zen of Go | Dave Cheney | 10 engineering values |
| GopherCon talks (yearly) | Various | Deep-dives on specific topics |

**Websites:**

| Resource | URL | Use For |
|----------|-----|---------|
| Go Tour | go.dev/tour | First 2 hours of Go |
| Go by Example | gobyexample.com | Quick syntax reference |
| Effective Go | go.dev/doc/effective_go | Idiomatic patterns |
| Go Style Guide | google.github.io/styleguide/go | Google's conventions |
| Go Playground | go.dev/play | Test snippets online |
| pkg.go.dev | pkg.go.dev | Package documentation |

**System Design:**

| Resource | URL | Use For |
|----------|-----|---------|
| System Design Primer | github.com/donnemartin/system-design-primer | Comprehensive overview |
| Designing Data-Intensive Apps | Book by Martin Kleppmann | Deep theory |
| ByteByteGo | bytebytego.com | Visual system design |
| Hello Interview | hellointerview.com | Practice problems |

---

## Appendix A: Go Proverbs Applied to GoShort

| Proverb | How GoShort Applies It |
|---------|----------------------|
| Don't communicate by sharing memory | Background cleanup uses context, not shared flags |
| The bigger the interface, the weaker the abstraction | `Cache` has 3 methods. `Storage` has 6. `Encoder` has 2. |
| Make the zero value useful | `Config{}` works with defaults. `URL{}` has sensible zeros. |
| Errors are values | `ErrNotFound`, `ErrExpired` — matched with `errors.Is()` |
| A little copying is better than a little dependency | Validation logic is hand-written, not imported from a library |
| Clear is better than clever | Guard clauses in handlers. No nested ternaries. No magic. |
| Don't panic | All functions return `error`. `panic` never used in library code. |
| interface{} says nothing | No `any` in GoShort. Concrete types or small interfaces. |
| Gofmt's style is no one's favorite | gofmt on save. No debates. |

---

## Appendix B: Anti-Pattern Field Guide

```
╭──────────────────────────────────────────────────────────────╮
│  🚫 ANTI-PATTERN FIELD GUIDE                                 │
│                                                               │
│  If you see yourself doing any of these, stop and rethink.   │
╰──────────────────────────────────────────────────────────────╯

❌ "util" or "helper" packages
   → Name packages by what they DO: "shortener", "storage", "cache"

❌ Global variables for dependencies
   → Pass via constructors. Always. Wire in main.go.

❌ Returning error AND logging it
   → One or the other. Never both. Log at the top (handler).

❌ Bare `return err` without context
   → Always: fmt.Errorf("what you were doing: %w", err)

❌ interface{} / any everywhere
   → Use concrete types or small, meaningful interfaces.

❌ Huge interfaces (5+ methods)
   → Split. Cache has 3 methods. Encoder has 2.

❌ init() functions with side effects
   → Explicit initialization in main().

❌ Goroutine without stop mechanism
   → Every `go` must have a way to exit (context, done channel).

❌ Mixing receiver types (value + pointer on same struct)
   → If one method needs pointer, ALL methods use pointer.

❌ Premature optimization
   → Write clear code first. Benchmark. Then optimize the hot path.

❌ Ignoring errors with _
   → Handle it. Or comment WHY it's safe to ignore.

❌ fmt.Sprintf in log messages
   → slog.Info("msg", "key", val)  // structured, not interpolated

❌ ALL_CAPS constants
   → MixedCaps always: MaxRetries, not MAX_RETRIES

❌ Using GORM "because it's popular"
   → sqlc: write real SQL, get type-safe Go code. No magic.

❌ Dockerfile that copies source before downloading deps
   → COPY go.mod first → go mod download → then COPY source

❌ No index on frequently-queried columns
   → EXPLAIN QUERY PLAN every query. Always.
```

---

## Appendix C: Glossary

### Go Terms

| Term | Meaning |
|------|---------|
| **goroutine** | Lightweight concurrent function execution (~2KB stack) |
| **channel** | Typed pipe for goroutine communication |
| **interface** | Set of method signatures, satisfied implicitly |
| **struct** | Named collection of fields (Go's "class" without inheritance) |
| **receiver** | The type a method is attached to: `func (t *Timer) Fire()` |
| **pointer** | Variable holding a memory address: `*T` is pointer to T |
| **slice** | Dynamic array, reference type |
| **map** | Hash table: `map[K]V`, reference type |
| **package** | A folder of .go files with the same `package` declaration |
| **module** | Collection of packages with a `go.mod` file |
| **context** | Carries deadlines, cancellation, and request-scoped values |
| **defer** | Schedule cleanup to run when function returns |
| **iota** | Auto-incrementing constant generator (for enums) |
| **zero value** | Default for uninitialized vars: `0`, `""`, `false`, `nil` |
| **sentinel error** | Package-level `var ErrXxx = errors.New(...)` |
| **guard clause** | Early return to keep happy path unindented |
| **table-driven test** | Test pattern: `[]struct` of cases iterated with `t.Run` |

### System Design Terms

| Term | Meaning |
|------|---------|
| **Base62** | Encoding using 62 chars: 0-9, a-z, A-Z |
| **Bijective** | One-to-one mapping: every input → exactly one output, reversible |
| **Birthday Paradox** | Collision probability grows faster than intuition suggests |
| **B-tree** | Balanced tree data structure used for database indexes |
| **Cache-aside** | Read cache first, on miss read DB and populate cache |
| **Cache stampede** | Many simultaneous cache misses hammering the database |
| **CDN** | Content Delivery Network — edge servers worldwide |
| **DNS** | Domain Name System — maps domain names to IP addresses |
| **Rate limiting** | Restricting request frequency per client |
| **Singleflight** | Deduplicate concurrent identical requests |
| **TLS** | Transport Layer Security — encryption for HTTPS |
| **Token bucket** | Rate limiting algorithm allowing bursts |
| **TTL** | Time To Live — how long a cached value is valid |
| **WAL** | Write-Ahead Logging — SQLite mode for crash safety |
| **301 redirect** | Permanent redirect — browser caches forever |
| **302 redirect** | Temporary redirect — browser asks server every time |

### Infrastructure Terms

| Term | Meaning |
|------|---------|
| **Container** | Lightweight, isolated process (Docker) |
| **Multi-stage build** | Dockerfile with separate build and runtime stages |
| **Reverse proxy** | Server in front of your app (Nginx/Caddy) |
| **TLS termination** | Decrypting HTTPS at the proxy, forwarding plain HTTP |
| **systemd** | Linux process manager — keeps services running |
| **UFW** | Uncomplicated Firewall — controls port access |
| **MCP** | Model Context Protocol — AI tool integration standard |
| **stdio transport** | MCP communication via stdin/stdout (local process) |

---

```
╭──────────────────────────────────────────────────────╮
│                                                      │
│    "The best way to learn is to build something      │
│     real, break it, fix it, and repeat."             │
│                                                      │
│    Now go build GoShort.                             │
│                                                      │
│    Start: go mod init github.com/yourname/goshort    │
│    Then:  Write a failing test.                      │
│    Then:  Make it pass.                              │
│    Then:  Refactor.                                  │
│    Repeat.                                           │
│                                                      │
╰──────────────────────────────────────────────────────╯
```

---

*Last updated: 2025-06-01*
*Companion to: [design.md](./design.md) · [cc-workflow-guide.md](./cc-workflow-guide.md)*
*Status: Final — Covers Go basics through MCP integration*