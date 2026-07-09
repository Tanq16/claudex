# Review Domain: Go Foundations

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web **Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../go-foundations/SKILL.md`

---

## Category 1: Project Layout (go-foundations)

Verify against the project layout defined in `go-foundations`:

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Entry point | `main.go` exists at root, ONLY calls `cmd.Execute()` | Read `main.go`, confirm no logic beyond import and `cmd.Execute()` |
| cmd/ directory | `cmd/root.go` exists with Cobra root command | Glob for `cmd/root.go` |
| internal/ preferred | Business logic in `internal/`, not `pkg/` (unless explicitly reusable) | Glob for `internal/` and `pkg/`, flag `pkg/` usage if not intentionally public |
| Subcommand packages | Grouped subcommands have their own package under `cmd/` | Glob for `cmd/*/`, verify parent commands have no `Run` function |
| Frontend assets location | If web project, static files in `internal/server/static/` | Glob for static file directories |
| Utils package (CLI Only / CLI + Web) | `utils/` exists at project root with expected files (globals.go, printer.go) | Glob for `utils/` |
| No utils package (Web Only) | `utils/` does NOT exist at project root | Glob for `utils/`, flag if it exists |

---

## Category 2: Core Principles (go-foundations)

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| KISS -- standard library preference | Standard library used where possible. Pre-approved packages always allowed (cobra, zerolog, charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2, gorilla/websocket, goccy/go-yaml). Other third-party packages are acceptable when they fill a genuine need the stdlib cannot reasonably cover (e.g., UUID generation, database drivers, cloud SDKs, `golang.org/x/` packages). Only flag a dependency when a clear, simpler stdlib alternative exists. | Read `go.mod`, for each non-pre-approved dependency evaluate whether it serves a need that the standard library cannot reasonably address; flag only those with obvious stdlib replacements |
| YAGNI -- no speculative code | No empty packages, no unused imports, no over-engineered abstractions | Grep for packages with only a package declaration and no meaningful code beyond TODOs |
| DRY -- utilities abstracted | Common operations (HTTP clients, config loading) abstracted in utils; goroutine patterns kept explicit (not abstracted) | Check for duplicated HTTP client setup, config loading across packages |
| Comments -- load-bearing why only | Default to zero comments. The sole test is whether a comment's *why* is load-bearing -- would a competent reader misread the intent, a trade-off, or a non-obvious constraint without it -- and each comment is judged on its own merit. Flag what-narration, comments that restate a signature or obvious logic, and comments kept only because they read as a reasonable explanation. | For each comment, ask whether removing it would let a competent reader misread intent/trade-off/constraint; flag any that would not |

---

## Category 2b: Modern Go (go-foundations)

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web

Verify the project is written to the Go 1.26+ baseline defined in `go-foundations` ("Modern Go" section + `../../go-foundations/references/modern-go.md`). **Only flag the high-signal patterns below.** Do NOT flag classic `for i := 0; i < n; i++` loops, manual slice/map operations that could use `slices`/`maps`, or any other modernization that is purely stylistic — those generate noise and fall outside this check.

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| go.mod version | `go` directive is `1.26` or newer | Read `go.mod`; flag if the `go` directive is below 1.26 |
| `any` over `interface{}` | Empty interface written as `any`, never `interface{}` | Grep `.go` files for `interface{}`; flag each occurrence (an empty interface should always be `any`) |
| `wg.Go` over manual Add/Done | WaitGroup goroutines spawned with `wg.Go(fn)`, not `wg.Add(1)` + `go func() { defer wg.Done(); ... }()` | Grep for `wg.Add(1)` immediately preceding a `go func`; flag the manual pattern |
| Typed atomics | `atomic.Bool` / `atomic.Int64` / `atomic.Pointer[T]` instead of the `atomic.AddInt32` / `LoadInt32` / `StoreInt32` free functions | Grep for `atomic.AddInt`, `atomic.LoadInt`, `atomic.StoreInt`, `atomic.SwapInt` calls; flag if found |

---

## Category 3: Logging (go-foundations)

**CLI Only checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Uses zerolog | `rs/zerolog` imported, `ConsoleWriter` configured | Grep for zerolog imports in `cmd/root.go` |
| Debug flag controls log level | `zerolog.SetGlobalLevel` toggled by `debugFlag` | Read `cmd/root.go`, check `setupLogs()` function |

**Web Only checks** (a CLI + Web hybrid applies these to its `internal/server/` layer only, and the zerolog checks above to `cmd/`):

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Uses standard log | `log.Printf` with manual level prefixes (`INFO`, `ERROR`, `DEBUG`) | Grep for `log.Printf` patterns |
| NO zerolog | `rs/zerolog` is NOT imported anywhere | Grep for zerolog imports, flag if present |
| NO utils print functions | No `u.PrintInfo`, `u.PrintFatal` etc. | Grep for `utils.Print` or `u.Print` patterns |

---

## Category 4: Utils Package (go-foundations)

**Applies to: CLI Only and CLI + Web (hybrid)**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| globals.go exists | Contains `GlobalDebugFlag` and `GlobalForAIFlag` variables | Read `utils/globals.go` |
| printer.go base functions | Contains `PrintInfo`, `PrintSuccess`, `PrintError`, `PrintFatal`, `PrintWarn` with three-way branch (debug → AI → human), plus `PrintGeneric` as plain pass-through with no branching. `PrintError`, `PrintFatal`, and `PrintWarn` must NOT include `err` in human or AI mode output — only the friendly `msg`; full error details are exclusively for `--debug` mode | Read `utils/printer.go`, check the five structured helpers for `GlobalDebugFlag` / `GlobalForAIFlag` / default branches, verify `PrintGeneric` prints raw output without mode-specific handling, and confirm `PrintError`/`PrintFatal`/`PrintWarn` human and AI branches use only `msg` (no `err`) |
| printer.go lifecycle functions | If output lifecycle patterns are used: `PrintRunning(msg)`, `PrintIndentedSuccess(msg)`, `PrintIndentedError(msg, err)`, `PrintIndentedWarn(msg, err)`, `PrintIndentedRunning(msg)` follow the same three-way branch. `ClearLines(n)` and `ClearPreviousLine()` are no-ops in debug/AI modes. `PrintProgress(label, percent)` uses braille-dot bar in human mode, `[PROGRESS]` prefix in AI mode, structured zerolog in debug mode | Read `utils/printer.go`, verify lifecycle functions follow same branching pattern as base functions. Verify `ClearLines`/`ClearPreviousLine` check `GlobalDebugFlag \|\| GlobalForAIFlag` before emitting ANSI escapes |
| AI mode in table.go | If table output used, renders markdown table when `GlobalForAIFlag` is set | Read `utils/table.go`, check for `GlobalForAIFlag` branch |
| AI mode in input.go | If TUI input used, all prompt functions branch on `GlobalForAIFlag`: `PromptInput`/`PromptPassword` use `ReadPipedLine()`, `PromptTextArea` uses `ReadPipedInput()` | Read `utils/input.go`, check each prompt function for `GlobalForAIFlag` branch |
| input.go exists (if TUI input used) | `PromptInput` (single-line textinput), `PromptPassword` (masked textinput), `PromptTextArea` (multi-line textarea, Ctrl+D submit) — all Bubbletea in human mode, piped stdin in AI mode | Glob for `utils/input.go` |
| table.go exists (if table output used) | Lipgloss table formatting | Glob for `utils/table.go` |
| config.go exists (if config management used) | Config struct loading from file/env/flags hierarchy | Glob for `utils/config.go` |

---

## Category 4b: No Utils Package (go-foundations)

**Applies to: Web Only**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| utils/ does not exist | No `utils/` directory at project root | Glob for `utils/`, flag if it exists |
| No utils imports | No Go files import the project's `utils` package | Grep for `utils` import paths across all `.go` files |
| log.Printf used for output | Commands and internal packages use `log.Printf` with prefixes | Grep for `log.Printf` patterns |
| No GlobalDebugFlag/ForAIFlag | No references to `GlobalDebugFlag` or `GlobalForAIFlag` anywhere | Grep for `GlobalDebugFlag` and `GlobalForAIFlag` |

---

## Output Format

Report findings in this exact format:

```
## Domain: Go Foundations

### [PASS] Category Name (go-foundations)

All checks passed.

### [ISSUES] Category Name (go-foundations)

1. **[Issue title]** (go-foundations: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (go-foundations)

Not applicable to this project type.
```

End your response with exactly:
```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

---

## Out of Scope (Hard Boundary)

Do NOT flag any of the following — they are not defined in any loaded skill:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | No golangci-lint, no gofmt, inconsistent formatting |
| Pre-commit | No pre-commit hooks, no husky |
| Code Quality CI | No lint/format CI steps |
| Documentation beyond README | No godoc, no changelogs, no contributing guide |
| Docker Compose | No docker-compose for development |
| Database | No migrations, no schema files |
| Dependency tooling | No dependabot, no renovate |
| Security scanning | No SAST, no container scanning |
| Code style opinions | Naming conventions not in skills, personal preferences |

**Rule:** If you cannot cite a specific section in a loaded skill for a finding, do not report it.
