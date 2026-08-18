---
name: go-cli-output
description: The utils printer for Go CLI tools - the three output tiers behind --debug and --for-ai, the Print* API, table rendering, terminal colors, and error discipline. Use when writing anything a CLI prints, when building or changing utils/printer.go, utils/table.go, or utils/globals.go, or when a command reaches for fmt.Println. Triggers on PrintInfo, PrintSuccess, PrintError, PrintFatal, PrintWarn, PrintGeneric, PrintTable, GlobalDebugFlag, GlobalForAIFlag, lipgloss.ANSIColor, and --for-ai.
user-invocable: false
---

# Go CLI Output

**Every line a CLI Only tool prints goes through `utils`, which renders it three ways depending on who is reading.**

Web Only and Headless API Service projects have no `utils/` package and print with `log.Printf` instead.

## The Three Tiers

| Tier | Flag | Output | Input |
|---|---|---|---|
| Human (default) | none | styled ANSI via lipgloss | interactive bubbletea TUI |
| AI | `--for-ai` | plain text with `[OK]`, `[ERROR]`, `[WARN]`, `[INFO]` prefixes | piped stdin |
| Debug | `--debug` | structured zerolog with timestamps and full error detail | not applicable |

`--debug` and `--for-ai` are mutually exclusive, enforced by `MarkFlagsMutuallyExclusive`, because zerolog output interleaved with parseable plain text is neither.

`--for-ai` is the single gate for every AI-friendly behavior, so a caller enables the whole contract with one flag instead of discovering three. The tool carries no LLM SDK dependency: it is the thing an agent invokes, not a thing that invokes a model.

```
echo "my input" | toolname command --for-ai
```

Human mode is ephemeral, since transient lines get cleared. AI and debug modes are permanent, since everything printed has to survive being piped into a parser or a log.

## Globals

```go
package utils

// GlobalDebugFlag is set by the cobra root command when --debug is passed.
var GlobalDebugFlag bool

// GlobalForAIFlag is set by the cobra root command when --for-ai is passed.
var GlobalForAIFlag bool
```

## The Print Shape

Every printer branches the same way, debug first, then AI, then human. Writing them all to one shape means a new printer is a copy with a different glyph rather than a new decision.

```go
var (
    infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12)) // bright blue
    successStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10)) // bright green
    errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9))  // bright red
    warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(11)) // bright yellow
)

func PrintInfo(msg string) {
    if GlobalDebugFlag {
        log.Info().Msg(msg)
    } else if GlobalForAIFlag {
        fmt.Println("[INFO] " + msg)
    } else {
        fmt.Println(infoStyle.Render("→ " + msg))
    }
}

func PrintError(msg string, err error) {
    if GlobalDebugFlag {
        if err != nil {
            log.Error().Err(err).Msg(msg)
        } else {
            log.Error().Msg(msg)
        }
    } else if GlobalForAIFlag {
        fmt.Println("[ERROR] " + msg)
    } else {
        fmt.Println(errorStyle.Render("✗ " + msg))
    }
}
```

## The Print Surface

| Function | Human | AI | Debug |
|---|---|---|---|
| `PrintInfo(msg)` | `→ msg` blue | `[INFO] msg` | `log.Info()` |
| `PrintSuccess(msg)` | `✓ msg` green | `[OK] msg` | `log.Info()` |
| `PrintError(msg, err)` | `✗ msg` red | `[ERROR] msg` | `log.Error().Err(err)` |
| `PrintFatal(msg, err)` | `✗ msg` red, then `os.Exit(1)` | `[ERROR] msg`, then exit | `log.Error().Err(err)`, then exit |
| `PrintWarn(msg, err)` | `! msg` yellow | `[WARN] msg` | `log.Warn().Err(err)` |
| `PrintGeneric(msg)` | `msg` | `msg` | `msg` |
| `PrintRunning(msg)` | `↻ msg` blue | `[RUNNING] msg` | `log.Info()` |
| `PrintIndentedSuccess(msg)` | `  ✓ msg` green | `[OK] msg` | `log.Info()` |
| `PrintIndentedError(msg, err)` | `  ✗ msg` red | `[ERROR] msg` | `log.Error().Err(err)` |
| `PrintIndentedWarn(msg, err)` | `  ! msg` yellow | `[WARN] msg` | `log.Warn().Err(err)` |
| `PrintIndentedRunning(msg)` | `  ↻ msg` blue | `[RUNNING] msg` | `log.Info()` |

`PrintGeneric` prints raw text with no branch at all, because data the caller wants verbatim (a URL, a token, a rendered table) must not gain a prefix or a color that a consumer then has to strip.

`PrintFatal` exits with status 1 after printing, so a caller never has to remember the `os.Exit` that a fatal message implies.

## Error Discipline

The `msg` parameter is the human-readable label, and `err` is the Go error. Passing the actual `err` in the error parameter is what makes `--debug` useful, since zerolog's `.Err(err)` records it as a structured field that a baked-in string cannot become.

Human and AI modes show only `msg`. The error detail is exclusively for debug introspection, which keeps a stack of wrapped errors out of a user's face while leaving it one flag away.

```go
utils.PrintFatal("git not found in PATH", err)
utils.PrintIndentedError(toolName, result.Err)
```

Passing `nil` for `err` is correct only when there genuinely is no underlying error: a validation failure, a summary line, an informational warning.

## Subprocess Errors

A direct `exec.Command` that fails returns "exit status 1" and nothing else, so capture stderr into the error before printing it, or the debug tier records a message with no cause.

```go
cmd := exec.Command("sudo", "cp", src, dst)
var stderr strings.Builder
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
    if detail := strings.TrimSpace(stderr.String()); detail != "" {
        err = fmt.Errorf("%s: %w", detail, err)
    }
    utils.PrintFatal("failed to copy binary", err)
}
```

A helper that already captures both streams into the returned error needs none of this.

## Terminal Colors

Colors are ANSI indices 0 through 15, never hex. An index is remapped by the user's terminal theme, so the same tool reads correctly under Dracula, Catppuccin, Solarized, or a scheme nobody has published; a hex value overrides that theme and fights it. Bright variants, 8 through 15, are preferred for foreground text.

```go
var (
    ColorBlue    = lipgloss.ANSIColor(12) // bright blue
    ColorGreen   = lipgloss.ANSIColor(10) // bright green
    ColorRed     = lipgloss.ANSIColor(9)  // bright red
    ColorYellow  = lipgloss.ANSIColor(11) // bright yellow
    ColorMagenta = lipgloss.ANSIColor(13) // bright magenta
    ColorCyan    = lipgloss.ANSIColor(14) // bright cyan
    ColorFg      = lipgloss.ANSIColor(15) // bright white, primary text
    ColorMuted   = lipgloss.ANSIColor(7)  // white, secondary text
    ColorChrome  = lipgloss.ANSIColor(8)  // bright black, borders and dim UI
)
```

## Tables

`PrintTable(headers, rows)` renders lipgloss box-drawing in human mode and a markdown table in AI mode, so the same call feeds a terminal and a parser without the caller branching.

```go
package utils

import (
    "fmt"
    "strings"

    "charm.land/lipgloss/v2"
    "charm.land/lipgloss/v2/table"
)

var (
    headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorFg).Padding(0, 1)
    cellStyle   = lipgloss.NewStyle().Foreground(ColorMuted).Padding(0, 1)
    borderStyle = lipgloss.NewStyle().Foreground(ColorChrome)
)

func PrintTable(headers []string, rows [][]string) {
    if GlobalForAIFlag {
        printMarkdownTable(headers, rows)
        return
    }
    t := table.New().
        Border(lipgloss.NormalBorder()).
        BorderStyle(borderStyle).
        Headers(headers...).
        Rows(rows...).
        StyleFunc(func(row, col int) lipgloss.Style {
            if row == table.HeaderRow {
                return headerStyle
            }
            return cellStyle
        })
    PrintGeneric(t.Render())
}

func printMarkdownTable(headers []string, rows [][]string) {
    if len(headers) == 0 {
        return
    }
    seps := make([]string, len(headers))
    for i := range seps {
        seps[i] = "---"
    }
    fmt.Println("| " + strings.Join(escapeCells(headers), " | ") + " |")
    fmt.Println("| " + strings.Join(seps, " | ") + " |")
    for _, row := range rows {
        fmt.Println("| " + strings.Join(escapeCells(row), " | ") + " |")
    }
}

func escapeCells(cells []string) []string {
    escaped := make([]string, len(cells))
    for i, cell := range cells {
        escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
    }
    return escaped
}
```

Pipe characters in cell values are escaped, because one unescaped `|` shifts every column after it and silently corrupts the parse.
