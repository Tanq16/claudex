---
name: go-backend
description: Use when implementing Go backend logic - covers internal package architecture, error handling, HTTP servers, storage patterns, and OAuth authentication for CLI clients (browser/device/manual flows, not server-side web OAuth)
user-invocable: false
---

# Go Backend

**Architecture and implementation patterns for Go internal packages.**

## When to Use

Use this skill when:
- Structuring `internal/` packages
- Implementing business logic
- Setting up HTTP servers
- Designing storage/persistence layer
- Handling errors across packages
- Adding OAuth authentication to CLI tools

**Requires:** `go-foundations` for project layout and principles.

**CLI + Web constraint:** Internal packages in CLI + Web projects must NOT import `utils/`. Use `log.Printf` with manual level prefixes (e.g., `log.Printf("ERROR ...")`) and `log.Fatal()`/`log.Fatalf()` for fatal errors. Keep messages generic — no package-name prefix.

## Package Architecture

### By Domain/Feature (Default)

```
internal/
├── auth/           # Authentication logic
├── download/       # Download functionality
└── server/         # HTTP server + frontend
```

### By Layer (When Many Features)

When project has many features across multiple layers:

```
internal/
├── handlers/       # HTTP handlers by feature
│   ├── auth.go
│   └── download.go
├── services/       # Business logic by feature
│   ├── auth.go
│   └── download.go
├── models/         # Data structures
└── server/         # Server setup
```

**Decision rule:** Start with domain/feature. Switch to layered when you have 5+ features that each need handlers, services, and models.

## Error Handling

### Two Package Types

**Task Packages** (internal logic, potentially portable to `pkg/`):
- Return errors as-is
- Small, focused methods
- No wrapping, no logging

```go
// internal/download/client.go
func (c *Client) FetchFile(url string) ([]byte, error) {
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, err  // Return as-is
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    return io.ReadAll(resp.Body)
}
```

**Interaction Packages** (Cobra commands, HTTP handlers):
- Wrap errors with context
- Handle logging
- User-facing error messages

CLI Only pattern (uses utils):
```go
// cmd/download.go
Run: func(cmd *cobra.Command, args []string) {
    client := download.NewClient()
    data, err := client.FetchFile(url)
    if err != nil {
        u.PrintFatal(fmt.Sprintf("Failed to download from %s", url), err)
    }
    u.PrintSuccess("Download complete")
}
```

CLI + Web pattern (the interaction layer is HTTP handlers, shown in the HTTP Server Pattern section, which use `log.Printf`). For Cobra commands in CLI + Web:
```go
// cmd/download.go
Run: func(cmd *cobra.Command, args []string) {
    client := download.NewClient()
    data, err := client.FetchFile(url)
    if err != nil {
        log.Fatalf("ERROR Failed to download from %s: %v", url, err)
    }
    log.Printf("INFO Download complete")
}
```

### Rationale

Task packages stay portable—if moved to `pkg/` later, no changes needed. Context and logging happen at boundaries (commands, handlers).

## HTTP Server Pattern

Use standard `net/http` (KISS principle) — no third-party routers (gin, chi, echo). A `Server` struct holds `host`, `port`, and an `*http.ServeMux`; `Setup()` mounts embedded static files and routes; `Run()` calls `http.ListenAndServe`.

```go
//go:embed static
var staticFiles embed.FS

func (s *Server) Setup() error {
    staticFS, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return err
    }
    s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
    s.mux.HandleFunc("/api/health", s.handleHealth)
    s.mux.HandleFunc("/", s.handleIndex) // serves static/index.html
    return nil
}
```

**`go-backend` owns the canonical embedded-static server** — the full `server.go` (struct, `New`, `Run`, `handleIndex`, health handler), the skeleton variant, and the middleware wrapper pattern all live in `./references/http-server-template.md`. `go-frontend` references that file rather than redefining the boilerplate.

## Storage Pattern

Most projects: JSON/file-based, stateless.

When persistence needed, use interface abstraction:

```go
// internal/storage/storage.go
package storage

type Store interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    Delete(key string) error
    List() ([]string, error)
}

// internal/storage/json.go
type JSONStore struct {
    path string
}

func NewJSONStore(path string) *JSONStore {
    return &JSONStore{path: path}
}

func (s *JSONStore) Get(key string) ([]byte, error) {
    // Read from JSON file
}

// internal/storage/postgres.go
type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
    // Connect to Postgres
}

func (s *PostgresStore) Get(key string) ([]byte, error) {
    // Query Postgres
}
```

### Usage

```go
// In command or server setup
var store storage.Store

if usePostgres {
    store, err = storage.NewPostgresStore(connStr)
} else {
    store = storage.NewJSONStore(dataPath)
}

// Pass store to handlers/services
svc := myservice.New(store)
```

## Authentication Pattern (CLI Only)

For CLI tools that authenticate with OAuth2 providers (Google, GitHub, Microsoft, etc.), use the three-mode login pattern in `internal/auth/`.

### Three Login Modes

| Mode | Flag | How It Works | Environment |
|------|------|-------------|-------------|
| Callback (default) | (none) | Opens browser, localhost server receives redirect | Interactive desktop |
| Device | `--device-login` | Shows URL + code, polls until authorized | Headless / SSH / server |
| Manual | `--manual` | Shows URL, user pastes authorization code | Last resort / no device flow support |

The user explicitly selects their mode via flags — no auto-fallback chain. Default opens browser; if it fails on headless, the error message directs them to `--device-login`.

### Key Design Decisions

- **`Login(config, mode)` takes a mode string** — the command layer maps flags to mode, auth package handles the flow
- **`openBrowser` returns error** — fast-fail on headless instead of hanging for 5 minutes
- **Device flow uses `oauth2.Config.DeviceAuth()` and `DeviceAccessToken()`** — built into `golang.org/x/oauth2`, handles polling and backoff automatically
- **Manual flow accepts both bare codes and full URLs** via `extractCode()` helper
- **Token persistence**: `~/.config/[APP_NAME]/token.json` with `0600` permissions
- **Auto-refresh on load**: `GetHTTPClient()` loads cached token, refreshes if expired, saves if refreshed

### Provider Support

Not all providers support device authorization (RFC 8628). When unsupported, omit `loginWithDevice` and the `--device-login` flag, keeping only callback and manual modes.

| Provider | Device Auth | Device Auth URL |
|----------|------------|-----------------|
| Google | Yes | `https://oauth2.googleapis.com/device/code` |
| Microsoft | Yes | `https://login.microsoftonline.com/common/oauth2/v2.0/devicecode` |
| GitHub | Yes | `https://github.com/login/device/code` |
| Box.com | No | — |

### Output Tier Usage

Login flows use `u.PrintInfo` for instructions/status and `u.PrintGeneric` for data (URLs, codes). `u.PromptInput` in manual mode reads from pipe in `--for-ai` mode. `u.PrintFatal` and `u.PrintSuccess` are used at the command layer only.

### Implementation

Use `./references/auth-patterns.md` for the complete `internal/auth/auth.go` and `cmd/login.go` templates.

## Config Struct Pattern

Define config structs that map to Cobra flags:

```go
// internal/download/config.go
package download

type Config struct {
    URL         string
    OutputPath  string
    Concurrency int
    Timeout     time.Duration
}

// internal/download/downloader.go
func Download(cfg Config) error {
    // Use cfg fields
}
```

CLI Only (uses `u.PrintFatal`):
```go
// cmd/download.go
Run: func(cmd *cobra.Command, args []string) {
    cfg := download.Config{
        URL:         downloadFlags.url,
        OutputPath:  downloadFlags.output,
        Concurrency: downloadFlags.concurrency,
        Timeout:     time.Duration(downloadFlags.timeout) * time.Second,
    }
    if err := download.Download(cfg); err != nil {
        u.PrintFatal("Download failed", err)
    }
}
```

CLI + Web (uses `log.Fatalf`):
```go
// cmd/download.go
Run: func(cmd *cobra.Command, args []string) {
    cfg := download.Config{
        URL:         downloadFlags.url,
        OutputPath:  downloadFlags.output,
        Concurrency: downloadFlags.concurrency,
        Timeout:     time.Duration(downloadFlags.timeout) * time.Second,
    }
    if err := download.Download(cfg); err != nil {
        log.Fatalf("ERROR Download failed: %v", err)
    }
}
```

## Common Patterns

### HTTP Client with Defaults

```go
// internal/httpclient/client.go
package httpclient

import (
    "net/http"
    "time"
)

func New() *http.Client {
    return &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }
}
```

### Parallelization (Vanilla Goroutines)

Don't abstract—keep explicit:

```go
func ProcessItems(items []string, concurrency int) error {
    sem := make(chan struct{}, concurrency)
    errCh := make(chan error, len(items))

    var wg sync.WaitGroup
    for _, item := range items {
        sem <- struct{}{} // Acquire (blocks if `concurrency` already running)
        wg.Go(func() {
            defer func() { <-sem }() // Release

            if err := processItem(item); err != nil {
                errCh <- err
            }
        })
    }

    wg.Wait()
    close(errCh)

    // Collect errors
    var errs []error
    for err := range errCh {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("%d items failed", len(errs))
    }
    return nil
}
```

## References

| File | Purpose |
|------|---------|
| `./references/http-server-template.md` | Canonical embedded-static `net/http` server (full `server.go`, skeleton variant, middleware) — referenced by `go-frontend` |
| `./references/auth-patterns.md` | Complete OAuth auth templates (`internal/auth/auth.go` and `cmd/login.go`) |
