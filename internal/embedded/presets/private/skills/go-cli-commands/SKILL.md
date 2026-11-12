---
name: go-cli-commands
description: The surface a Go CLI command exposes - the baseline every tool starts at, flag conventions and precedence, positional arguments, enum flags, and the one-line stdin prompt a tool may buy for an absent flag. Use when adding or changing a flag, deciding where a value enters a command, registering a flag at the right level, declaring Args, validating a fixed set of allowed values, or asking the user for a value or a choice. Triggers on PersistentFlags, BoolVar, StringVarP, SortFlags, cobra.NoArgs, ExactArgs, RangeArgs, pflag.Value, MarkFlagRequired, MarkFlagsOneRequired, MarkFlagsRequiredTogether, MarkFlagsMutuallyExclusive, utils/input.go, PromptInput, PromptSelect, PromptPassword, PromptMultiSelect, bufio.NewReader, and os.Stdin. Not for the shape of the command tree itself.
user-invocable: false
---

# Go CLI Commands

**What a command offers its caller: the arguments it takes, the flags it reads, and, where a tool buys it, the one line it reads from stdin when a flag is absent.**

## The Command Surface

Every tool starts at one baseline and buys anything past it deliberately. The baseline is `tool <command> <args> --flags`: arguments name what is acted on, flags change how, and a value the command needs and did not get is an error naming the flag.

One purchase sits on top: asking for a value whose flag is absent. It is decided per tool while the surface is designed rather than discovered halfway through an implementation, because it adds a path every later change has to keep working. `serve` never buys it, since a server runs under a supervisor or in a container with nobody at its stdin.

Two channels carry a value into a command, and which one a given value uses is a decision rather than a preference.

| Channel | Carries | Where it lives |
|---|---|---|
| Config and environment | credentials, endpoints, anything set once and reused | `~/.config/[APP_NAME]/`, and `[APP_NAME]_*` variables |
| Flags | everything else a single run needs | `init()` on the command that reads them |

Precedence, highest first: an explicit flag, then the environment, then the config file, then the built-in default. The flag wins because it is the most specific thing the caller said in this invocation.

## Flags

Flags are registered in `init()`, one call per flag, next to the command they belong to.

A boolean flag takes a long name and no shorthand. A single letter standing for a switch abbreviates nothing the reader can recover, and it collides with the next switch somebody adds. A flag that takes a value may have a shorthand, because the value sitting beside it already says what it is.

```go
cmd.Flags().BoolVar(&flags.all, "all", false, "Include all items")
cmd.Flags().BoolVar(&flags.force, "force", false, "Overwrite an existing file")
cmd.Flags().StringVarP(&flags.name, "name", "n", "default", "Description")
cmd.Flags().IntVarP(&flags.count, "count", "c", 10, "Number of items")
cmd.Flags().StringSliceVarP(&flags.tags, "tag", "t", []string{}, "Tags (repeatable)")
cmd.Flags().DurationVarP(&flags.timeout, "timeout", "T", 30*time.Second, "Request timeout")
```

A flag registers at the level that reads it. `PersistentFlags` on the root is for what the whole tree honors, which in practice is `--debug` and nothing else; `PersistentFlags` on a parent command covers that group; everything else is `Flags()` on the command itself. A flag registered a level too high appears in the help of every command that ignores it.

Flag names stay unique across the whole tree, and Cobra does not catch a collision. `AddFlagSet` skips any flag whose name already exists (`pflag@v1.0.9 flag.go:914`), so a subcommand's local `--workers` shadows the root's persistent `--workers` with no error at registration and none at parse, and the subcommand reads a value the caller never set.

`SortFlags` stays at its default of `true` (`pflag@v1.0.9 flag.go:1270`), so `--help` lists flags alphabetically. A reader hunting one flag in the help output finds it faster than a reader reconstructing the order they were declared in.

A value that can span lines takes a flag naming a file to read it from rather than carrying the text itself, because a multi-line value passed as a flag argument survives one shell's quoting rules and not the next one's.

```go
cmd.Flags().StringVarP(&flags.bodyFile, "body-file", "f", "", "Read the body from this file")
```

`MarkFlagRequired` states a requirement Cobra enforces before `Run` is reached, which produces a usage message rather than a nil dereference:

```go
cmd.Flags().StringVarP(&flags.input, "input", "i", "", "Input file (required)")
cmd.MarkFlagRequired("input")
```

An environment variable supplies a default rather than being read inside `Run`, which is what puts the flag above it in precedence and makes `--help` show the value the command will actually use. A variable the tool owns is namespaced with the tool's name; one belonging to another tool keeps that tool's name.

```go
defaultRegion := os.Getenv("AWS_REGION")
cmd.Flags().StringVarP(&flags.region, "region", "r", defaultRegion, "AWS region (or AWS_REGION env)")
```

A secret is the exception and never becomes a flag default. pflag appends `(default %q)` to a string flag's usage line whenever the default is non-empty (`pflag@v1.0.9 flag.go:753-757`), so `--help` prints the live credential into screenshots, terminal logs, and pasted output. The flag registers empty and the variable is read inside `Run`, where the value reaches no help text.

```go
cmd.Flags().StringVarP(&flags.token, "token", "t", "", "GitHub token (or GITHUB_TOKEN env)")
```

```go
token := cmp.Or(flags.token, os.Getenv("GITHUB_TOKEN"))
if token == "" {
    u.PrintFatal("send needs --token, or GITHUB_TOKEN in the environment", nil)
}
```

`cmp.Or` returns the first argument that is not the zero value, so the precedence reads as the one line it is rather than an if-else chain.

Three markers state a relationship between flags that Cobra enforces at parse time, which is where the caller can still fix it (`cobra@v1.10.2 flag_groups.go`):

| The relationship | Marker |
|---|---|
| both flags only mean something together | `cmd.MarkFlagsRequiredTogether("cert", "key")` |
| at least one of the set is needed | `cmd.MarkFlagsOneRequired("file", "url")` |
| the flags contradict each other | `cmd.MarkFlagsMutuallyExclusive("file", "url")` |

The hand-rolled equivalent inside `Run` rejects the same combination several lines later, after the body has already done whatever it does before validating.

## Positional Arguments

Every command carrying a `Run` sets `Args`, including the ones taking none. Cobra falls back to `ArbitraryArgs` when `Args` is nil (`cobra@v1.10.2 command.go:1172-1177`), so a command silently swallowing a mistyped subcommand name as a positional and ignoring it is what a project gets by default.

| The command takes | `Args` |
|---|---|
| nothing | `cobra.NoArgs` |
| exactly n | `cobra.ExactArgs(n)` |
| at least n | `cobra.MinimumNArgs(n)` |
| between n and m | `cobra.RangeArgs(n, m)` |
| one value drawn from a known set | `cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)` |

A command that only groups subcommands leaves `Args` nil, because having no `Run` is exactly what puts `cobra.NoArgs` out of reach. `execute` returns `flag.ErrHelp` for a non-runnable command (`cobra@v1.10.2 command.go:955-957`) before it reaches `ValidateArgs` at `:968`, and `ExecuteC` renders that as help text with a nil error (`command.go:1152-1154`), so `appname feature bogus` prints help and exits zero either way.

On a root with subcommands, `Args` is worse than inert. `Find` calls `legacyArgs` only when `Args` is nil (`cobra@v1.10.2 command.go:775-777`), and that call is the only thing reporting an unknown command on a root with children, so `rootCmd.Args = cobra.NoArgs` turns `appname bogus` from an error into help text and exit zero.

## Enum Flags

A flag with a fixed set of allowed values validates through a `pflag.Value` implementation rather than inside `Run`, so a bad value is rejected before any work starts and `--help` states the set. `pflag.Value` is three methods (`pflag@v1.0.9 flag.go:208`).

```go
type mcpMode string

func (m *mcpMode) String() string { return string(*m) }
func (m *mcpMode) Type() string   { return "mcps|connectors|none" }

func (m *mcpMode) Set(v string) error {
    switch v {
    case "mcps", "connectors", "none":
        *m = mcpMode(v)
        return nil
    }
    return fmt.Errorf("must be one of mcps, connectors, none")
}
```

```go
launchCmd.Flags().Var(&launchFlags.mcp, "mcp", "Which MCP sources to load")
```

`Type()` is what `--help` prints beside the flag, so naming the allowed set there documents the flag without a second sentence of help text.

## Input

A command that buys a prompt reads one line from stdin through two helpers in `utils/input.go`. There is no terminal UI behind them: what is typed echoes, a secret included, and a pipe feeds the same reader a terminal does, so `echo "$TOKEN" | tool login` needs nothing extra.

| Helper | Reads | Returns |
|---|---|---|
| `PromptInput(prompt)` | one line, trimmed | `(string, error)` |
| `PromptSelect(headers, rows)` | a serial number typed against a numbered table | `(int, error)`, the 0-based row |

```go
package utils

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

var stdin = bufio.NewReader(os.Stdin)

func PromptInput(prompt string) (string, error) {
    fmt.Print(prompt + " ")
    line, err := stdin.ReadString('\n')
    if line == "" && err != nil {
        return "", err
    }
    return strings.TrimSpace(line), nil
}

func PromptSelect(headers []string, rows [][]string) (int, error) {
    numbered := make([][]string, len(rows))
    for i, row := range rows {
        numbered[i] = append([]string{strconv.Itoa(i + 1)}, row...)
    }
    PrintTable(append([]string{"#"}, headers...), numbered)
    answer, err := PromptInput(fmt.Sprintf("Choose 1-%d:", len(rows)))
    if err != nil {
        return -1, err
    }
    n, err := strconv.Atoi(answer)
    if err != nil || n < 1 || n > len(rows) {
        return -1, fmt.Errorf("%q is not a number from 1 to %d", answer, len(rows))
    }
    return n - 1, nil
}
```

One `bufio.Reader` wraps stdin for the whole process. It reads ahead in 4096-byte chunks (`bufio.go:19`), so a second reader on `os.Stdin` starts after whatever the first one buffered, and the second of two piped answers disappears.

The reader is `ReadString` rather than `bufio.Scanner`, whose 64 KiB token limit (`scan.go:82`) rejects a long pasted assertion that `ReadString` returns whole. A last line with no trailing newline still counts as an answer, and only an empty read at the end of input is an error.

A terminal holds less than a pipe. macOS caps a line in canonical mode at 1024 bytes (`MAX_CANON` in `sys/syslimits.h`), and a longer paste never reaches the reader, so a value that can pass about a thousand characters, such as a SAML assertion, is piped in: `pbpaste | tool login`.

A prompted value resolves from its flag first and asks only when the flag is empty. A failed read, which is what an exhausted pipe or Ctrl+D produces, ends the command with a message naming the flag that would have supplied the value.

```go
account := launchFlags.account
if account == "" {
    idx, err := u.PromptSelect([]string{"Account", "Plan"}, rows)
    if err != nil {
        u.PrintFatal("launch needs --account", err)
    }
    account = accounts[idx].Name
}
```

A flag with a prompt behind it is never marked required. Cobra validates required flags at `command.go:1007` and reaches `c.Run` at `:1019` (`cobra@v1.10.2`), so the invocation is rejected before the prompt could run.

A value is validated once, after it resolves, whichever channel it came from. A check placed beside the prompt covers only the interactive path, and the flag then accepts what the prompt would have refused.

A choice is a numbered table and a typed serial number, with no cursor and no key handling. An answer outside the range is an error rather than a second question, which keeps the helper to one read.

A confirmation is `PromptInput("Proceed? [y/N]")`, where only `y` or `yes` proceeds, and a `--yes` flag skips it. A reflexive Enter then declines.

A secret read at a prompt never reaches a printer. The structured printers branch to the debug tier, which writes the string into a log that outlives the session.
