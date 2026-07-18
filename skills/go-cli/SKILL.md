---
name: go-cli
description: Use when implementing CLI commands with Cobra - covers root command setup, subcommand patterns, flags, output lifecycle patterns, and TUI output with bubbletea/lipgloss
user-invocable: false
---

# Go CLI

**Cobra command patterns and terminal UI for CLI tools.**

## When to Use

Use this skill when:
- Setting up Cobra root command
- Adding new commands or subcommands
- Implementing CLI flags
- Showing sequential progress with running/done lifecycle (phases, single-ops, checks)
- Creating terminal UI output
- Adding `--debug` or `--for-ai` flag behavior

**Requires:** `go-foundations` for project layout and utils package.

## Project Type Context

CLI Only and Web Only projects have different root command setups (CLI + Web hybrids use the CLI Only setup, plus a `serve` command):

| Aspect | CLI Only | Web Only |
|--------|----------|-----------|
| Root command | Full: zerolog, utils, --debug, --for-ai, setupLogs | Simplified: just rootCmd, Execute(), init() with AddCommand |
| Logging | zerolog via setupLogs() | Standard `log` package (no setupLogs) |
| Flags | --debug and --for-ai global flags | No debug/for-ai flags |
| Utils | Full utils/ package imported | NO utils import — use log.Printf/log.Fatalf |
| Output | Three-tier: debug/AI/human via utils.Print* | log.Printf with manual prefixes |

## Root Command Setup

`main.go` only calls `cmd.Execute()`. The root command differs by project type:

| | CLI Only | Web Only |
|---|----------|-----------|
| Imports | zerolog, utils, subcommand pkgs | cobra + subcommand pkgs only |
| Flags | `--debug`, `--for-ai` (mutually exclusive) | none |
| Logging | `setupLogs()` configures zerolog; `--debug`/`--for-ai` toggle levels and `utils.Global*Flag` | standard `log`, no setup |
| `init()` | hides help/completion, registers flags, `cobra.OnInitialize(setupLogs)`, adds commands | hides help/completion, adds commands |

Both set `Use`, `Short`, `Version` (from `AppVersion` ldflag), and `HiddenDefaultCmd: true`. Full `main.go` and both `cmd/root.go` templates are in `./references/command-templates.md`.

## Command Patterns

- **Simple command (no subcommands):** define directly in `cmd/` with its own flag struct and `init()`. CLI Only (and a hybrid's CLI subcommands) use `u.Print*`/`u.PrintFatal`; Web Only (and a hybrid's `serve`/server) uses `log.Printf`/`log.Fatalf`.
- **Subcommand group:** create a package under `cmd/` (e.g. `cmd/feature-cmd/`) with an exported parent command (no `Run`) and unexported subcommands (each with `Run`). Wire them in the package's `init()`.

Full `serve` and subcommand-package templates are in `./references/command-templates.md`.

## Flag Patterns

### Flag Types

```go
// String with short flag
cmd.Flags().StringVarP(&flags.name, "name", "n", "default", "Description")

// Int with short flag
cmd.Flags().IntVarP(&flags.count, "count", "c", 10, "Number of items")

// Bool with short flag
cmd.Flags().BoolVarP(&flags.all, "all", "a", false, "Include all items")

// String slice
cmd.Flags().StringSliceVarP(&flags.tags, "tag", "t", []string{}, "Tags (can specify multiple)")

// Duration
cmd.Flags().DurationVarP(&flags.timeout, "timeout", "T", 30*time.Second, "Request timeout")
```

### Required Flags

```go
cmd.Flags().StringVarP(&flags.input, "input", "i", "", "Input file (required)")
cmd.MarkFlagRequired("input")
```

### Environment Variable Defaults

```go
// Read env var as default value
defaultToken := os.Getenv("GITHUB_TOKEN")
cmd.Flags().StringVarP(&flags.token, "token", "t", defaultToken, "GitHub token (or GITHUB_TOKEN env)")
```

### Mutually Exclusive Flags

```go
cmd.MarkFlagsMutuallyExclusive("file", "stdin")
```

## Run Function Pattern (CLI Only)

```go
Run: func(cmd *cobra.Command, args []string) {
    // 1. Validate flags
    if flags.required == "" {
        u.PrintFatal("--required flag is required", nil)
    }

    // 2. Build config struct
    cfg := internal.Config{
        Field1: flags.field1,
        Field2: flags.field2,
    }

    // 3. Call internal package
    result, err := internal.DoThing(cfg)
    if err != nil {
        u.PrintFatal("Failed to do thing", err)
    }

    // 4. Output result
    u.PrintSuccess("Thing completed")
    u.PrintGeneric(result)
}
```

## Login Command Pattern (CLI Only)

For CLI tools with OAuth authentication, the login command uses mutually exclusive flags (`--device-login`, `--manual`) to select the login mode. The auth logic lives in `internal/auth/` (see `go-backend`); the command just maps flags to a mode string, calls `auth.Login(config, mode)`, and handles output.

**Modes:**
- `appname login` — default, opens browser with localhost callback
- `appname login --device-login` — shows URL + code for headless/SSH
- `appname login --manual` — paste authorization code (last resort when device flow unsupported)

If the provider does not support device authorization (RFC 8628), omit `--device-login`. Full `cmd/login.go` template is in `./references/command-templates.md`.

## Output Lifecycle Patterns (CLI Only)

For sequential operations that need running/done progress, line clearing, and phase grouping. This is the simple counterpart to the Highway pattern (used for concurrent multi-job pipelines).

**Four lifecycle types:**

| Lifecycle | When | Pattern |
|-----------|------|---------|
| Phase | Group of sequential sub-tasks | Running header → indented results → clear → summary |
| Single-operation | One task, no sub-steps | Running → clear → result |
| Multi-step | Multiple sequential steps | Running → clear → running → clear → ... → final result |
| Check | Read-only scan of multiple items | Running → clear → summary with indented findings |

**Key functions:** `PrintRunning`, `PrintIndented{Success,Error,Warn,Running}`, `ClearLines(n)`, `ClearPreviousLine()`, `PrintProgress(label, percent)`.

**ClearLines count rule:** Always `ClearLines(lineCount + 1)` — `+1` for the running header.

**Progress bar:** Goroutine-driven braille-dot bar that overwrites a single line. Use `atomic.Bool` to guard the final clear (prevents eating the wrong line if work completes before first tick).

See `./references/output-lifecycle-patterns.md` for full patterns and examples.

## TUI Patterns (Bubbletea + Lipgloss) (CLI Only)

### Terminal Colors (Lipgloss)

Use ANSI standard color indices (0-15) instead of hardcoded hex values. These indices are remapped by the user's terminal theme, so output adapts to Dracula, Catppuccin, Solarized, or any custom scheme automatically. Hardcoded hex colors (e.g. `#89b4fa`) bypass the theme and force a specific palette regardless of user preference. Prefer bright variants (8-15) for foreground text.

The full `lipgloss.ANSIColor` palette and common style definitions are in `./references/command-templates.md` (Terminal Colors section).

### Selection Prompts

For interactive choices, reuse the `utils` selector helpers rather than hand-rolling a bubbletea model per command:

- `PromptSelect(label, options) (int, error)` — single-choice list; arrow / `j` / `k` to move, `enter` to pick, `esc` / `ctrl+c` to cancel. Returns the chosen index, or `-1` when cancelled.
- `PromptMultiSelect(label, options) (map[int]bool, error)` — multi-choice list; `space` toggles, `enter` confirms. Returns the selected indices, or `nil` when cancelled.

Both branch on `--for-ai`: instead of a TUI they read a line from stdin — a 1-based index for `PromptSelect`, and a comma-separated list of indices (or `none`) for `PromptMultiSelect` — so choices stay scriptable (`echo "2" | tool cmd --for-ai`). Always treat the cancel path (`idx < 0` / `nil` map) as a clean no-op abort.

## Output Tier Behavior (CLI Only)

`--debug` and `--for-ai` are **mutually exclusive** (enforced via `MarkFlagsMutuallyExclusive`).

When **neither** flag is set (human mode):
- `utils.Print*` functions render styled ANSI output via lipgloss
- Tables render with box-drawing borders
- Input prompts launch bubbletea TUI

When `--debug` is set:
- `utils.GlobalDebugFlag = true`
- Print functions route to zerolog with timestamps and full error details
- zerolog level set to `DebugLevel`

When `--for-ai` is set:
- `utils.GlobalForAIFlag = true`
- zerolog disabled (`zerolog.Disabled`) to prevent stray log output
- See "AI Mode Behavior" below for details

## AI Mode Behavior (`--for-ai`) (CLI Only)

The `--for-ai` global flag enables AI-agent-friendly I/O. When set:

**Output changes:**
- Print functions emit prefixed plain text instead of styled ANSI:
  - `[OK]` (success), `[ERROR]` (error/fatal), `[WARN]` (warning), `[INFO]` (info)
- Tables render as markdown (`| col | col |`) instead of lipgloss box-drawing
- No spinners, no color codes, no interactive TUI elements

**Input changes:**
- `PromptInput()` and `PromptPassword()` read from stdin pipe via `ReadPipedInput()`
- No bubbletea TUI launched — enables `echo "data" | toolname cmd --for-ai`

**Note:** `--debug` and `--for-ai` are mutually exclusive — Cobra enforces this at the flag level.

See `go-foundations` for the three-tier utils implementation.

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/command-templates.md` | Template | Full code: `main.go`, both `cmd/root.go` variants, simple commands, subcommand package, `cmd/login.go`, and lipgloss colors/styles |
| `./references/output-lifecycle-patterns.md` | Pattern | Phase, single-op, multi-step, check, and progress indicator lifecycle patterns with code examples |
