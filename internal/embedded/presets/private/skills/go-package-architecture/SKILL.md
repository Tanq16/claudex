---
name: go-package-architecture
description: How internal/ packages are organized in a Go project and how errors travel across their boundaries. Use when creating a package under internal/, deciding between domain and layered organization, choosing where to wrap an error, designing a persistence layer, or shaping a config struct that a command fills from flags. Triggers on internal/ package creation, fmt.Errorf wrapping, Store interfaces, JSONStore, handlers/services/models directories, and http.Client construction.
user-invocable: false
---

# Go Package Architecture

**How `internal/` is divided, and which layer is allowed to wrap an error or print one.**

## Organization

Start with one package per domain or feature. The grouping matches how work arrives, so a change to downloads touches one directory.

```
internal/
├── auth/
├── download/
└── server/
```

Switch to layered organization once five or more features each need handlers, services, and models. Below that threshold the layers hold one file each and every change edits three directories to accomplish one thing.

```
internal/
├── handlers/       # HTTP handlers by feature
├── services/       # business logic by feature
├── models/         # data structures
└── server/         # server setup
```

Deviating from the structure a project already has is a design change: propose it rather than introducing it inside an unrelated diff.

## Error Handling

Packages fall into two kinds, and the kind decides what happens to an error passing through.

**Task packages** hold internal logic that could be lifted to `pkg/` unchanged. They return errors as they are, add no context for its own sake, and log nothing. Staying quiet is what makes them portable, since a package that logs has already decided how the program talks to its user.

```go
// internal/download/client.go
func (c *Client) FetchFile(url string) ([]byte, error) {
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    return io.ReadAll(resp.Body)
}
```

**Interaction packages** are the boundaries: Cobra commands and HTTP handlers. They add the context that names what the user was trying to do, they log, and they produce the user-facing message. Doing it here means the context is written once, where the intent is known, rather than accumulated as a stack of prefixes on the way up.

```go
// cmd/download.go, CLI Only
Run: func(cmd *cobra.Command, args []string) {
    data, err := download.NewClient().FetchFile(url)
    if err != nil {
        u.PrintFatal(fmt.Sprintf("Failed to download from %s", url), err)
    }
    u.PrintSuccess("Download complete")
}
```

A Web Only project writes the same boundary with `log.Fatalf("ERROR Failed to download from %s: %v", url, err)`, and its HTTP handlers log with `log.Printf` and return a status code.

Internal packages in a Web Only project, and the server layer of a CLI + Web hybrid, do not import `utils`. A hybrid's CLI-operation packages follow the CLI Only convention instead, so the import tells you which half of the binary you are in.

## Storage

Most projects are JSON-file backed and stateless, which needs no abstraction at all.

Introduce a `Store` interface only when a second implementation genuinely exists or is committed to. An interface with one implementation is indirection that costs a jump and buys nothing.

```go
// internal/storage/storage.go
type Store interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    Delete(key string) error
    List() ([]string, error)
}
```

```go
// internal/storage/json.go
type JSONStore struct{ path string }

func NewJSONStore(path string) *JSONStore { return &JSONStore{path: path} }
```

Selection happens once at startup and the chosen store is passed down, so nothing below the entry point knows which backend it has:

```go
var store storage.Store
if usePostgres {
    store, err = storage.NewPostgresStore(connStr)
} else {
    store = storage.NewJSONStore(dataPath)
}
svc := myservice.New(store)
```

A state file the program itself writes is created at mode `0600`, because it lives in the user's working directory and often carries paths or identifiers from their environment.

## Config Structs

An internal package takes a config struct rather than reading flags or the environment, which is what lets it be called from a test, a second command, or a server handler without any of them.

```go
// internal/download/config.go
type Config struct {
    URL         string
    OutputPath  string
    Concurrency int
    Timeout     time.Duration
}

func Download(cfg Config) error { /* ... */ }
```

The command fills the struct from its flags and calls in:

```go
cfg := download.Config{
    URL:         downloadFlags.url,
    OutputPath:  downloadFlags.output,
    Concurrency: downloadFlags.concurrency,
    Timeout:     time.Duration(downloadFlags.timeout) * time.Second,
}
if err := download.Download(cfg); err != nil {
    u.PrintFatal("Download failed", err)
}
```

## HTTP Clients

A shared client sets explicit timeouts and connection limits, because `http.DefaultClient` has no timeout at all and one unresponsive host hangs the program indefinitely.

```go
// internal/httpclient/client.go
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
