---
name: go-http-server
description: The canonical net/http server for Go web services, with an embedded static frontend. Use when creating internal/server/server.go, serving a frontend out of the binary, adding a route or an API handler, adding middleware, or streaming live updates to a page. Triggers on net/http, http.ServeMux, go:embed static, embed.FS, fs.Sub, http.StripPrefix, http.FileServer, ListenAndServe, text/event-stream, EventSource, Last-Event-ID, http.NewResponseController, and any mention of gin, chi, or echo.
user-invocable: false
---

# Go HTTP Server

**One `Server` struct, the standard library's `ServeMux`, and a frontend served straight out of the binary.**

Third-party routers (gin, chi, echo) are not used. `net/http` has had method-and-pattern routing since Go 1.22, so a router dependency now buys syntax rather than capability, and it costs a version to track and an API to relearn.

The server owns the `embed.FS` boilerplate for the whole project. A frontend skill covers what goes inside `static/`; this covers how those bytes reach a browser.

## The Server

```go
package server

import (
    "embed"
    "io/fs"
    "net"
    "net/http"
    "strconv"

    "github.com/rs/zerolog/log"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
    host string
    port int
    mux  *http.ServeMux
}

func New(host string, port int) *Server {
    return &Server{host: host, port: port, mux: http.NewServeMux()}
}

func (s *Server) Setup() error {
    staticFS, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return err
    }
    s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

    s.mux.HandleFunc("/api/health", s.handleHealth)
    s.mux.HandleFunc("/api/data", s.handleData)

    s.mux.HandleFunc("/", s.handleIndex)
    return nil
}

func (s *Server) Run() error {
    addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))
    log.Info().Str("addr", addr).Msg("starting")
    return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
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

`Setup` is separate from `Run` so a caller can report a mount failure before the process commits to listening on a port. `Run` logs the address it binds, so the command calling it logs nothing about the start.

The template carries no dependencies. A handler that needs one, such as a store, gets it as a field on `Server` that `New` takes after `host` and `port`. The entry point chooses it once and passes it down, so nothing in the server knows which backend it has.

`net.JoinHostPort` brackets an IPv6 host, which a plain `host:port` format leaves ambiguous.

`fs.Sub` strips the `static` prefix from the embedded tree, and `http.StripPrefix` strips it from the request path. Both are needed: without the first, `/static/css/app.css` resolves to `static/static/css/app.css` inside the FS.

A PWA service worker needs its own exact route rather than the `/static/` mount, because a worker's scope is the directory it is served from. One served under `/static/` cannot control the app at `/`.

`handleIndex` is registered on `/` rather than on an exact pattern, so a deep link into a client-routed single-page app returns the page instead of a 404.

The server layer logs through zerolog, the same as every other part of every project type. A CLI + Web hybrid keeps its `utils` printers for the command surface and uses zerolog here, so the binary has one logger rather than one per command.

## Skeleton

A fresh project drops the concrete API handlers and keeps the embed wiring exactly as it is, because that wiring is the part that fails silently when retyped from memory.

```go
func (s *Server) Setup() error {
    staticFS, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return err
    }
    s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
    s.mux.HandleFunc("/", s.handleIndex)
    return nil
}
```

## Headless API Service

A service with no frontend keeps the `Server` struct, `New`, `Setup`, and `Run`, and drops `//go:embed static`, `fs.Sub`, the `/static/` mount, and `handleIndex`. Its `Setup` registers API routes only, and an unmatched path falls through to the mux's own 404.

## Middleware

Middleware is added case by case rather than by default, since a wrapper applied to every route runs on the health check too.

```go
func withLogging(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("request")
        next(w, r)
    }
}

s.mux.HandleFunc("/api/data", withLogging(s.handleData))
```

Wrapping at the registration site keeps the applied middleware visible in `Setup`, where the route list already is.

## Streaming Events

A live view reads one Server-Sent Events stream rather than polling. `EventSource` reconnects on its own and sends the last `id:` it received as `Last-Event-ID`, so a dropped stream resumes where it stopped with no client code.

```go
type Event struct {
    ID   string
    Name string
    Data []byte
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
    after := cmp.Or(r.Header.Get("Last-Event-ID"), r.URL.Query().Get("after"))
    backlog, live, stop := s.events.Subscribe(after)
    defer stop()

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    rc := http.NewResponseController(w)
    write := func(ev Event) {
        fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", ev.ID, ev.Name, ev.Data)
    }
    for _, ev := range backlog {
        write(ev)
    }
    if rc.Flush() != nil {
        return
    }

    heartbeat := time.Tick(15 * time.Second)
    for {
        select {
        case <-r.Context().Done():
            return
        case <-heartbeat:
            fmt.Fprint(w, ": ping\n\n")
        case ev, ok := <-live:
            if !ok {
                return
            }
            write(ev)
        }
        if rc.Flush() != nil {
            return
        }
    }
}
```

`?after=` carries the cursor when the client opens a new `EventSource` itself, such as when a tab becomes visible again, because the constructor cannot set a request header.

`Subscribe` takes the backlog and the live channel under one lock, so no event falls between the replay and the stream. An empty, unknown, or stale cursor gets a single `reset` event carrying a full snapshot and the id it is current to, in place of a replay.

An id carries the log's epoch as well as its position, so an id from before a restart reads as stale rather than matching a different event.

A subscriber that falls behind has its channel closed rather than blocking the publisher. Its client reconnects from the last id it received and loses nothing.

`Data` is one line of compact JSON, because a newline inside it ends the `data:` field.

The comment line every 15 seconds keeps a proxy from closing an idle connection, the interval the WHATWG spec suggests. A write to a client that has gone away fails at the next flush, which ends the handler.

The handler returns when the request context ends, and the deferred `stop` releases the subscription, so a closed tab leaves nothing behind.
