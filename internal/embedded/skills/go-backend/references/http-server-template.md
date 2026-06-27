# HTTP Server Template

Canonical embedded-static `net/http` server for CLI + Web projects. This is the single source of
truth for the `embed.FS` + `fs.Sub` + `http.StripPrefix` + `handleIndex` boilerplate — `go-frontend`
references this file instead of re-defining it.

Use standard `net/http` (KISS principle). No third-party routers (gin, chi, echo).

## internal/server/server.go

```go
package server

import (
    "embed"
    "fmt"
    "io/fs"
    "log"
    "net/http"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
    host string
    port int
    mux  *http.ServeMux
}

func New(host string, port int) *Server {
    return &Server{
        host: host,
        port: port,
        mux:  http.NewServeMux(),
    }
}

func (s *Server) Setup() error {
    // Serve embedded static files
    staticFS, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return err
    }
    s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

    // API routes
    s.mux.HandleFunc("/api/health", s.handleHealth)
    s.mux.HandleFunc("/api/data", s.handleData)

    // Serve index.html for root
    s.mux.HandleFunc("/", s.handleIndex)

    return nil
}

func (s *Server) Run() error {
    addr := fmt.Sprintf("%s:%d", s.host, s.port)
    log.Printf("INFO Starting on %s", addr)
    return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
    data, err := staticFiles.ReadFile("static/index.html")
    if err != nil {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "text/html")
    w.Write(data)
}
```

## Skeleton variant

For a fresh skeleton, drop the concrete API handlers and leave a TODO — keep the `embed.FS`,
`fs.Sub`, `StripPrefix`, and `handleIndex` exactly as above:

```go
func (s *Server) Setup() error {
    staticFS, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return err
    }
    s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
    s.mux.HandleFunc("/", s.handleIndex)

    // TODO: Add API routes

    return nil
}
```

## Middleware (when needed)

Handle case-by-case, not by default. When needed, use the wrapper pattern:

```go
func withLogging(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Printf("INFO %s %s", r.Method, r.URL.Path)
        next(w, r)
    }
}

// Usage
s.mux.HandleFunc("/api/data", withLogging(s.handleData))
```
