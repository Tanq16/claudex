---
name: go-foundations
description: Use when writing, refactoring, reviewing, or testing ANY Go code. The canonical reference for Go conventions — project taxonomy (CLI Only / Web Only / CLI + Web / Headless API Service / Library), project layout, modern Go 1.26+ idioms (wg.Go, t.Context(), range-over-int, slices/maps/cmp, errors.AsType, min/max/clear, new(value)), edge-case-driven unit testing (add/write Go tests, table-driven, edge cases), core principles, dependency selection, logging, and the utils package. Load this for any Go work. The `develop` skill is the per-task entry point that selects and applies these conventions.
user-invocable: false
---

# Go Foundations

**Shared patterns and conventions for all Go projects.**

## When to Use

Use this skill when:
- Writing, refactoring, reviewing, or testing any Go code
- Checking the canonical project taxonomy and layout
- Implementing logging or the utils package
- Reviewing a project for convention compliance

This skill is the *reference* for how Go projects should look. The `develop` skill is the entry point that loads this and the other relevant skills for a given task and holds the work to them.

**Related skills:**
- `project-readme` - README structure and templates
- `project-ci-cd` - Makefile, GitHub Actions, releases

## Project Taxonomy

This is the **canonical project taxonomy** for all Go work. Other skills (`go-cli`, `go-backend`, `go-frontend`, `develop`, `review-code`) refer back to these type names rather than redefining them.

| Type | What it is | Defining markers |
|------|-----------|------------------|
| **CLI Only** | Terminal tool for users | `cobra`, `utils/` package, zerolog, lipgloss/bubbletea/bubbles v2; multi-platform binaries; NO Docker |
| **Web Only** | Web app/dashboard served from a Go binary, with no real CLI beyond `serve` | `cobra` (a lone `serve` command) + embedded frontend (`internal/server/static/`); standard `log`; Docker; NO `utils/` |
| **CLI + Web** | Hybrid — a real CLI tool (many subcommands) that *also* serves a web app from one `serve` subcommand | Full CLI Only stack (`utils/`, zerolog, lipgloss/bubbletea/bubbles v2, `--debug`/`--for-ai`) for the command surface, **plus** `internal/server/static/` and a `serve` command whose server uses standard `log`; Docker |
| **Headless API Service** | REST/gRPC backend with no frontend | `cobra` (optional) + `internal/server` handlers, no `static/`; standard `log`; Docker; NO `utils/` |
| **Library / Module** | Importable package, no entry point | NO `main.go`, NO `cobra`, NO `utils/`; exported packages at root or under `pkg/`; consumed via `go get` |

**CLI Only** and **Web Only** are the most common and have full layouts below; **CLI + Web** is a composition of the two (summarized after), followed by **Headless API Service** and **Library / Module**.

## Project Layout

### CLI Only Projects

Terminal tools for users. Includes `utils/` package with debug/for-ai flags, zerolog, lipgloss v2, bubbletea v2, bubbles v2. Multi-platform binaries only — NO Dockerfile, NO Docker in CI/CD.

```
project-root/
├── main.go                 # Entry point - calls cmd.Execute() only
├── go.mod
├── go.sum
├── Makefile                # Build targets (no docker targets)
├── README.md
├── .github/
│   ├── assets/
│   │   └── logo.png        # Project logo
│   └── workflows/
│       └── release.yaml    # Binaries only, no docker job
├── cmd/
│   ├── root.go             # Cobra root command (zerolog, --debug, --for-ai, utils)
│   ├── command.go          # Simple commands
│   └── subcommand-cmd/     # Grouped subcommands get their own package
│       ├── parent.go
│       └── child.go
├── internal/               # Primary - private packages (90% of projects)
│   ├── feature1/
│   └── feature2/
├── utils/                  # Shared utilities (top-level, not inside internal/)
│   ├── globals.go          # GlobalDebugFlag, GlobalForAIFlag
│   ├── printer.go          # CLI output abstractions (three-tier: debug/AI/human)
│   ├── input.go            # User input handling (TUI or piped stdin)
│   ├── config.go           # Config management (when needed)
│   └── table.go            # Table output (lipgloss or markdown)
└── pkg/                    # Rare - only for reusable packages
```

### Web Only Projects

Web apps in Docker containers. NO `utils/` package, NO `GlobalDebugFlag`/`ForAIFlag`, NO lipgloss/bubbletea/bubbles/zerolog (including their v2 variants). Uses standard `log` package with manual prefixes. Includes Dockerfile and Docker in CI/CD.

```
project-root/
├── main.go                 # Entry point - calls cmd.Execute() only
├── go.mod
├── go.sum
├── Makefile                # Build targets with docker targets and assets
├── Dockerfile              # Two-stage build for containerized deployment
├── README.md
├── .github/
│   ├── assets/
│   │   └── logo.png        # Project logo
│   └── workflows/
│       └── release.yaml    # Docker + binaries
├── cmd/
│   ├── root.go             # Simplified Cobra root (no debug/for-ai, no zerolog, no utils)
│   ├── serve.go            # Serve command using log.Printf
│   └── subcommand-cmd/     # Grouped subcommands get their own package
│       ├── parent.go
│       └── child.go
├── internal/               # Primary - private packages
│   ├── feature1/
│   ├── feature2/
│   └── server/             # Web server
│       ├── server.go
│       └── static/         # Embedded frontend assets
│           ├── css/
│           ├── fonts/
│           ├── js/
│           └── index.html
└── pkg/                    # Rare - only for reusable packages
```

**Key rules:**
- `main.go` is ONLY an entry point to Cobra
- Use `internal/` by default, `pkg/` only when explicitly creating importable packages
- Subcommand groups get their own package under `cmd/`
- Frontend assets live in `/internal/server/static/`
- CLI Only projects have `utils/`; Web Only projects do NOT — CLI + Web hybrids do, for their command surface

### CLI + Web Projects (hybrid)

A real CLI tool that *also* serves a web app. Structurally it is a **CLI Only** project — full `utils/` package, zerolog, lipgloss/bubbletea, `--debug`/`--for-ai`, and the CLI Only `cmd/root.go` — with a **Web Only** server grafted on: an `internal/server/` package holding the embedded `static/` frontend, reached through a single `serve` command. Ships Docker **and** multi-platform binaries.

```
project-root/
├── main.go                 # Entry point - calls cmd.Execute() only
├── Makefile                # CLI Only targets + docker targets and assets
├── Dockerfile              # Two-stage build for the served web app
├── cmd/
│   ├── root.go             # CLI Only root: zerolog, --debug, --for-ai, utils
│   ├── serve.go            # The one web command — server logs via log.Printf
│   └── operation.go        # CLI-operation subcommands — full utils/zerolog/TUI stack
├── internal/
│   ├── feature1/
│   └── server/             # Web server, only reached by `serve`
│       ├── server.go
│       └── static/         # Embedded frontend assets
└── utils/                  # Present — the CLI surface uses it
```

**Split on command boundaries:** the `serve` command's server layer uses standard `log` (`log.Printf`), exactly like a Web Only project; every other subcommand uses the CLI Only stack (zerolog behind `--debug`, `utils` printers otherwise, bubbletea/lipgloss for TUI). One binary, two output disciplines — divided by which command you are in, never mixed within one.

### Headless API Service

A REST/gRPC backend with **no frontend**. Structurally a Web Only project minus the frontend: standard `log` (no `utils/`, no zerolog/lipgloss), Dockerfile and Docker in CI/CD, `cobra` only if the service needs subcommands beyond `serve`.

```
project-root/
├── main.go                 # Entry point (cmd.Execute() or a direct serve())
├── go.mod / go.sum
├── Makefile                # Build + docker targets (no frontend assets)
├── Dockerfile              # Two-stage build
├── cmd/                    # Optional — only if more than `serve`
│   └── serve.go
└── internal/
    ├── server/             # HTTP/gRPC server + handlers — NO static/ subtree
    │   └── server.go
    ├── feature1/
    └── feature2/
```

- Use the `go-backend` HTTP server pattern (`../go-backend/references/http-server-template.md`) but **drop the `embed.FS`/`static/`/`handleIndex` parts** — there is no frontend to serve.
- No `utils/`, no `--for-ai`/`--debug`, no `go-frontend`. Logging is standard `log` with manual prefixes (same rules as Web Only).

### Library / Module

An importable package with **no entry point** — consumed by other projects via `go get`. No `main.go`, no `cmd/`, no `cobra`, no `utils/`.

```
module-root/
├── go.mod / go.sum
├── README.md               # Usage/API docs (consumers read this)
├── <package>.go            # Exported API at module root, or...
├── pkg/                    # ...grouped exported packages
│   └── <package>/
└── internal/               # Private helpers not part of the public API
```

- Public API lives at the module root or under `pkg/`; keep `internal/` for non-exported helpers.
- No logging framework imposed — a library should not configure global logging; return errors and let the caller decide. Avoid `log.Fatal`/`os.Exit` in library code.
- The unit-testing philosophy below applies fully; everything else (utils, Cobra, frontend, Docker) does not.

## Core Principles

### KISS (Keep It Stupid Simple)
- Use standard library packages when possible
- Pre-approved packages (always allowed):
  - `gorilla/websocket` for WebSocket
  - `goccy/go-yaml` for YAML (standard yaml.v2 deprecated)
  - `rs/zerolog` for structured logging **(CLI Only)**
  - `spf13/cobra` for CLI
  - `charm.land/bubbletea/v2` + `charm.land/lipgloss/v2` + `charm.land/bubbles/v2` for TUI **(CLI Only)**
- **Web Only projects** use the standard `log` package only — zerolog, bubbletea (v2), lipgloss (v2), and bubbles (v2) are NOT used
- Other third-party packages are acceptable when they fill a genuine need that the standard library cannot reasonably cover (e.g., `google/uuid` for UUID generation, database drivers, cloud SDK clients, `golang.org/x/` packages). Evaluate whether the dependency is justified, not whether it appears on the pre-approved list.

### YAGNI (You Ain't Gonna Need It)
- Don't build for all future needs
- Make code extendable, not comprehensive
- Focus on current requirements

### DRY (Don't Repeat Yourself)
- Abstract utilities: HTTP clients, config loading, common operations
- DON'T abstract: goroutine patterns, semaphores (keep vanilla, explicit)
- Prefer explicit parallelization over generic abstractions

## Comments and Code Style

Default to zero comments. Code should read as self-documenting; a comment earns its place only by saying what the code cannot.

The sole test for a comment is whether its *why* is **load-bearing** — would a competent reader misread the intent, a trade-off, or a non-obvious constraint without it? If not, delete it. That it reads as a reasonable explanation, or that the code is intricate, does not qualify it. Judge every comment on its own merit against this test.

- **Comment the *why*, not the *what*** — the code already states what it does. Reserve comments for intent, trade-offs, and non-obvious constraints (why this order, why this bound, what a subtle edge case guards against).
- **No redundant comments** — never restate the code or narrate obvious control flow (`// increment i`, `// loop over items`). A comment that mirrors the line below it is noise. This holds even when the task says "add comments" or "explain what it does": tidy the code and add a why only where one is warranted, never what-narration. Never embed example or scaffolding code behind `//` — that is documentation, not a comment.
- **One line by default** — a single line is usually enough to state the why, and often none is needed. Only span multiple lines when there is genuinely more to explain. Internal, unexported helpers rarely need a doc comment at all — don't add a paragraph that just restates the signature.
- **Keep them current** — a stale comment is worse than none. Update or delete comments when the code they describe changes.

## Modern Go (target 1.26+)

All Go in these projects targets **Go 1.26 or newer** — set the `go.mod` `go` directive to `go 1.26` (or later). Write to that baseline: prefer modern built-ins and standard-library helpers (`any`, `min`/`max`/`clear`, `slices`, `maps`, `cmp`, `sync.WaitGroup.Go`, `errors.AsType`, range-over-int and iterators, `new(value)`, etc.) over legacy patterns, and never use an outdated idiom when a current one exists.

See `./references/modern-go.md` for the full curated set of idioms with examples.

## Dependency Selection

Choosing dependencies is a KISS decision. Apply this judgment when adding, auditing, or upgrading any dependency:

- **Prefer the standard library** whenever it can reasonably do the job.
- **Pre-approved packages** (always fine — see the KISS list above): `gorilla/websocket`, `goccy/go-yaml`, `rs/zerolog` (CLI Only), `spf13/cobra`, `charm.land/bubbletea`+`lipgloss`+ `bubbles` v2 (CLI Only TUI).
- **Other third-party packages** are acceptable when they fill a genuine need the standard library cannot reasonably cover (e.g. `google/uuid`, database drivers, cloud SDKs, `golang.org/x/...`). Evaluate whether the dependency is *justified* — do not gate on a fixed allow-list.
- **Prefer well-maintained, standard choices** over niche alternatives (e.g. `zerolog` over `logrus`, `cobra` over `urfave/cli`).
- **Keep dependencies current** — track latest stable versions, prefer stable over pre-release, and check release notes for breaking changes on major bumps.

Apply this judgment whenever you add, audit, or upgrade a dependency — scan `go.mod` (and any HTML/Makefile CDN pins), check the latest stable versions, and read release notes for breaking changes before bumping a major.

## Logging Patterns

### CLI Only Projects (use zerolog)

```go
import (
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

// In root.go setupLogs()
zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
output := zerolog.ConsoleWriter{
    Out:        os.Stdout,
    TimeFormat: time.DateTime,
    NoColor:    false,
}
log.Logger = zerolog.New(output).With().Timestamp().Logger()
zerolog.SetGlobalLevel(zerolog.InfoLevel)
if debugFlag {
    zerolog.SetGlobalLevel(zerolog.DebugLevel)
    utils.GlobalDebugFlag = true
}
```

- Logs only shown when `--debug` flag used
- Without debug: use utils package print abstractions
- Keep log messages generic — no package-name attribution required (most logs originate from the shared `utils` package, so a package field adds noise without value)

### Web Only Projects (use standard log)

**Web Only projects have NO utils package, NO `GlobalDebugFlag`/`GlobalForAIFlag`.** Use standard `log` package only. Use `log.Fatal()` or `log.Fatalf()` for fatal errors.

```go
import "log"

log.Printf("INFO Starting on port %d", port)
log.Printf("ERROR Failed to validate token: %v", err)
log.Fatalf("ERROR Failed to bind: %v", err)
```

- Manual level prefixes: `INFO`, `ERROR`, `DEBUG`
- Keep messages generic — no package-name prefix required
- Timestamp with date/time in local timezone
- No color, straight sequential logs

### CLI + Web (hybrid) Projects (split logging)

One binary, two disciplines drawn on command boundaries: the `serve` command's server layer follows the Web Only rules above (standard `log`, `log.Printf`), while every other subcommand follows the CLI Only rules (zerolog behind `--debug`, `utils` printers otherwise). Never mix the two within one command.

## Config Management

**Default:** Cobra flags only (most projects)

**Extended (when needed):** Config hierarchy with priority:
1. Environment variables (highest)
2. CLI flags
3. YAML config file (path passed via `--config` flag)
4. Defaults (lowest)

**CLI Only:** Config handling lives in `utils/config.go`, returns Go struct passed to functions.

**Web Only:** Config handling uses Cobra flags + env vars directly, or a config package in `internal/`. No `utils/config.go` — Web Only projects do not have a `utils/` package.

**CLI + Web (hybrid):** follows CLI Only — config in `utils/config.go`; the `serve` command reads the same struct (or env) to configure the server.

## README Structure

Use the `project-readme` skill for README templates covering:
- CLI Only projects (command-line tools)
- Web Only projects (web apps, dashboards)
- CLI + Web hybrids (a CLI tool that also serves a web app)
- Chrome Extensions

## AI-Augmented CLI Pattern (CLI Only)

**This section applies to CLI Only projects and the command surface of CLI + Web hybrids.** Web Only projects do not use `--for-ai`, `--debug`, or the utils package.

All CLI tools support a `--for-ai` global flag (mutually exclusive with `--debug`) that makes them AI-agent-friendly. This creates a **three-tier output system**:

| Tier | Flag | Output | Input |
|------|------|--------|-------|
| Human (default) | none | Styled ANSI/lipgloss | Interactive bubbletea TUI |
| AI | `--for-ai` | Structured plain text with prefixes | Piped stdin |
| Debug | `--debug` | Structured zerolog | N/A (logging only) |

**When `--for-ai` is set:**
- Print functions emit prefixed plain text: `[OK]`, `[ERROR]`, `[WARN]`, `[INFO]`
- Tables render as markdown (`| col | col |`) instead of lipgloss box-drawing
- Input functions read from stdin pipe instead of launching bubbletea TUI
- No ANSI colors, no spinners, no interactive prompts

**Design philosophy:**
- Zero LLM SDK dependencies — the tool itself is what AI agents invoke
- The `--for-ai` flag is the single gate for all AI-friendly behavior
- Enables usage like: `echo "my input" | toolname command --for-ai`

## Utils Package (CLI Only)

**Web Only projects must NOT create or import a `utils/` package** — they use `log.Printf` with manual prefixes instead. CLI + Web hybrids DO have `utils/`, for their command surface.

The `utils/` package provides:

| File | Purpose |
|------|---------|
| `printer.go` | **Base:** `PrintInfo(msg)`, `PrintSuccess(msg)`, `PrintError(msg, err)`, `PrintFatal(msg, err)`, `PrintWarn(msg, err)`, `PrintGeneric(msg)` — three-way branch: debug (zerolog) → AI (prefixed plain text) → human (styled lipgloss). **Lifecycle:** `PrintRunning(msg)`, `PrintIndentedSuccess(msg)`, `PrintIndentedError(msg, err)`, `PrintIndentedWarn(msg, err)`, `PrintIndentedRunning(msg)` — running indicators and indented sub-task output. **Clearing:** `ClearLines(n)`, `ClearPreviousLine()` — ANSI line clearing (no-op in debug/AI modes). **Progress:** `PrintProgress(label, percent)` — braille-dot progress bar with in-place overwrite |
| `input.go` | `PromptInput(prompt, placeholder)` — single-line Bubbletea textinput (human) / `ReadPipedLine` (AI). `PromptPassword(prompt)` — masked single-line. `PromptTextArea(prompt, placeholder)` — multi-line Bubbletea textarea with Ctrl+D submit (human) / `ReadPipedInput` bulk stdin (AI) |
| `table.go` | `PrintTable(headers, rows)` — lipgloss box-drawing tables (human) / markdown tables (AI mode) |
| `config.go` | Config struct loading (when needed) |
| `globals.go` | `GlobalDebugFlag` and `GlobalForAIFlag` variables |

See `./references/utils-templates.md` for implementation patterns.

## Unit Testing

**Tests exist to prove logical correctness and robustness — not to chase coverage.**

The goal of a unit test is to pin down edge cases, special conditions, and the inputs that are prone to breaking or panicking. A test that merely re-asserts the happy path so the suite "passes" adds noise, not safety. Every test should earn its place by encoding a scenario you actually reasoned about.

### Principles

- **Scenario-driven, not coverage-driven.** Before writing tests, think through how the code can break: boundary values, empty/nil inputs, zero-length and single-element slices, malformed data, concurrent access, integer/size overflow, and anything that could panic. Those scenarios are what become test cases.
- **Unit tests, not integration tests.** Test the package's own logic in isolation. Do not stand up servers, hit networks, or exercise multiple packages end-to-end — that is out of scope here.
- **Robustness focus.** A function that can panic should have a test proving it doesn't on the nasty inputs. Prefer table-driven tests to enumerate edge cases compactly.
- **Don't test the trivial.** Skip pedantic tests of getters, trivial wrappers, or code with no branching or failure modes. If there's no logic to break, there's nothing to test.
- **Fully implement the code first.** Write complete, working implementations; then add tests for the scenarios that matter. Tests are not a substitute for finishing the code.

### Placement

- Tests live **in the package they test** (same directory, `package foo` or `package foo_test`).
- **One `_test.go` file per package is enough.** Even when a package has several source files, collect that package's edge-case tests into a single test file rather than mirroring each source file. Split only if one file becomes genuinely unwieldy.

### Modern test idioms (1.26+)

- **`t.Context()`** for a test's context — not `context.WithCancel(context.Background())`.
- **`for b.Loop()`** for the main benchmark loop — not `for i := 0; i < b.N; i++`.
- **`omitzero`** (not `omitempty`) in JSON struct tags for `time.Time`, `time.Duration`, structs, slices, and maps, where zero-value omission matters.

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        in      string
        want    Result
        wantErr bool
    }{
        {"empty input", "", Result{}, true},
        {"only separator", ",", Result{}, true},
        {"trailing separator", "a,", Result{Items: []string{"a"}}, false},
        // ... the edge cases you reasoned about
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(t.Context(), tt.in)
            if (err != nil) != tt.wantErr {
                t.Fatalf("Parse(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
            }
            // assert got vs tt.want
        })
    }
}
```

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/utils-templates.md` | Template | Complete utils package implementations — printer (base + lifecycle + progress), input, table, config |
| `./references/modern-go.md` | Reference | Curated modern Go (1.26+) idioms — built-ins, slices/maps/cmp, sync, errors, iterators, strings, time, new(value) |
