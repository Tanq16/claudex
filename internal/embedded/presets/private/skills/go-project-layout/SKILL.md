---
name: go-project-layout
description: The canonical Go project taxonomy, directory layout, logging discipline, and config loading. Use when starting a Go project, adding a package or directory, deciding where a file belongs, choosing between zerolog and the standard log package, or wiring config. Triggers on go.mod, main.go, cmd/, internal/, pkg/, utils/, internal/server/static/, and on any question of whether a project is CLI Only, Web Only, CLI + Web, a Headless API Service, or a Library.
user-invocable: false
---

# Go Project Layout

**Which of the five Go project types you are in, what its tree looks like, and how it logs and loads config.**

Every other Go convention keys off the project type, so settling the type is the first step of any Go task.

## Project Taxonomy

| Type | What it is | Defining markers |
|---|---|---|
| CLI Only | Terminal tool for users | `cobra`, `utils/`, zerolog, lipgloss/bubbletea/bubbles v2; multi-platform binaries; no Docker |
| Web Only | Web app served from a Go binary, with no real CLI beyond `serve` | `cobra` with a lone `serve` command, embedded frontend at `internal/server/static/`, standard `log`, Docker, no `utils/` |
| CLI + Web | A real CLI tool that also serves a web app from one `serve` subcommand | The full CLI Only stack for the command surface, plus `internal/server/static/` and a `serve` command whose server uses standard `log`; Docker |
| Headless API Service | REST or gRPC backend with no frontend | `internal/server` handlers with no `static/`, standard `log`, Docker, no `utils/`; `cobra` only when it needs more than `serve` |
| Library / Module | Importable package with no entry point | No `main.go`, no `cobra`, no `utils/`; exported packages at the module root or under `pkg/`; consumed via `go get` |

Read the type off the tree before writing anything, because the same file is correct in one type and a defect in another: a `utils/` package belongs in CLI Only and is a defect in Web Only.

## Layout

### CLI Only

```
project-root/
├── main.go                 # calls cmd.Execute() and nothing else
├── go.mod / go.sum
├── Makefile                # build targets, no docker targets
├── README.md
├── .github/
│   ├── assets/logo.png
│   └── workflows/release.yaml   # binaries only
├── cmd/
│   ├── root.go             # zerolog, --debug, --for-ai, utils
│   ├── command.go          # simple commands
│   └── feature-cmd/        # grouped subcommands get their own package
│       ├── parent.go
│       └── child.go
├── internal/               # private packages, where 90% of the logic lives
│   ├── feature1/
│   └── feature2/
├── utils/                  # top-level, not inside internal/
│   ├── globals.go
│   ├── printer.go
│   ├── input.go
│   ├── table.go
│   └── config.go
└── pkg/                    # rare, only for genuinely reusable packages
```

### Web Only

```
project-root/
├── main.go
├── go.mod / go.sum
├── Makefile                # build, assets, and docker targets
├── Dockerfile
├── README.md
├── .github/
│   ├── assets/logo.png
│   └── workflows/release.yaml   # docker and binaries
├── cmd/
│   ├── root.go             # no debug/for-ai, no zerolog, no utils
│   └── serve.go            # uses log.Printf
└── internal/
    ├── feature1/
    └── server/
        ├── server.go
        └── static/         # embedded frontend
            ├── css/ fonts/ js/
            └── index.html
```

### CLI + Web

Structurally a CLI Only project with a Web Only server grafted on: the full `utils/` package, zerolog, and the CLI Only `cmd/root.go`, plus an `internal/server/` holding the embedded `static/` frontend reached through a single `serve` command. It ships Docker and multi-platform binaries.

```
project-root/
├── main.go
├── Makefile                # CLI Only targets plus docker targets and assets
├── Dockerfile
├── cmd/
│   ├── root.go             # zerolog, --debug, --for-ai, utils
│   ├── serve.go            # the one web command, server logs via log.Printf
│   └── operation.go        # CLI subcommands, full utils/zerolog/TUI stack
├── internal/
│   ├── feature1/
│   └── server/
│       ├── server.go
│       └── static/
└── utils/                  # present, the CLI surface uses it
```

The two disciplines divide on command boundaries and never mix inside one command, because a user running `serve` reads a server log and a user running `sync` reads styled terminal output, and one binary emitting both formats from one command is unreadable.

### Headless API Service

A Web Only project minus the frontend: standard `log`, no `utils/`, Dockerfile and Docker in CI, and `cobra` only when the service needs subcommands beyond `serve`.

```
project-root/
├── main.go                 # cmd.Execute() or a direct serve()
├── go.mod / go.sum
├── Makefile                # build and docker targets, no frontend assets
├── Dockerfile
├── cmd/serve.go            # optional, only when more than serve exists
└── internal/
    ├── server/server.go    # handlers, no static/ subtree
    └── feature1/
```

Its HTTP server drops the `embed.FS`, `static/`, and `handleIndex` pieces, since there is no frontend to serve.

### Library / Module

```
module-root/
├── go.mod / go.sum
├── README.md               # usage and API docs, since consumers read this
├── <package>.go            # exported API at the module root, or
├── pkg/<package>/          # grouped exported packages
└── internal/               # private helpers outside the public API
```

A library configures no global logging and avoids `log.Fatal` and `os.Exit`, because those decisions belong to the program importing it rather than to the package.

## Layout Rules

`main.go` holds an import and a call to `cmd.Execute()`, so the entry point stays free of logic that tests cannot reach.

New packages go under `internal/` by default. `pkg/` is for packages you intend other repositories to import, and putting private code there commits you to an API you never meant to promise.

A group of related subcommands gets its own package under `cmd/` (`cmd/feature-cmd/`), which keeps the flag variables of one group from colliding with another's in the shared `cmd` package.

Frontend assets live at `internal/server/static/` so a single `//go:embed static` directive in the server package picks them all up.

## Logging

### CLI Only, and the command surface of CLI + Web

zerolog behind `--debug`, and the `utils` printers otherwise. Logs stay hidden in normal use because a user running a tool wants the result, not a trace of how it was produced.

```go
func setupLogs() {
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime, NoColor: false}
    log.Logger = zerolog.New(output).With().Timestamp().Logger()
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    if debugFlag {
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
        utils.GlobalDebugFlag = true
    }
}
```

Log messages stay generic and carry no package-name field, because most logs originate in the shared `utils` package where a package field would be the same value every time.

### Web Only, Headless API Service, and the server layer of CLI + Web

The standard `log` package with manual level prefixes, and `log.Fatalf` for errors that end the process. These projects have no `utils/` package and no `GlobalDebugFlag`.

```go
log.Printf("INFO Starting on port %d", port)
log.Printf("ERROR Failed to validate token: %v", err)
log.Fatalf("ERROR Failed to bind: %v", err)
```

Output is timestamped, sequential, and uncolored, so it survives being piped into a container log collector that strips nothing and interprets nothing.

## Config

Cobra flags alone cover most projects. Reach for the layered form only when a project genuinely needs a config file, because a hierarchy nobody populates is four lookups to answer one question.

The layered form resolves in this order, highest first: environment variables, then CLI flags, then a YAML file passed via `--config`, then built-in defaults. Environment variables win so a deployment can override a baked-in flag without rebuilding the image.

| Project type | Where config loading lives |
|---|---|
| CLI Only, CLI + Web | `utils/config.go`, returning a struct passed into functions |
| Web Only, Headless API Service | Cobra flags and environment variables directly, or a config package under `internal/` |
| Library / Module | Nowhere. A library takes its configuration as function arguments |

The loader returns a struct rather than exposing a global, so a caller can construct one in a test without touching the environment.

```go
func LoadConfig(path string) (*Config, error) {
    cfg := &Config{Server: ServerConfig{Port: 8080, Host: "0.0.0.0"}}
    if path != "" {
        if data, err := os.ReadFile(path); err == nil {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, err
            }
        }
    }
    if host := os.Getenv("APP_HOST"); host != "" {
        cfg.Server.Host = host
    }
    return cfg, nil
}
```

A missing config file falls back to defaults instead of erroring, because `--config` naming a path that does not exist yet is a normal first run.
