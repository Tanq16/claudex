---
name: go-cli-commands
description: The surface a Go CLI command exposes - the baseline every tool starts at, flag conventions and precedence, positional arguments, and enum flags. Use when adding or changing a flag, deciding where a value enters a command, registering a flag at the right level, declaring Args, or validating a fixed set of allowed values. Triggers on PersistentFlags, BoolVar, StringVarP, SortFlags, cobra.NoArgs, ExactArgs, RangeArgs, pflag.Value, MarkFlagRequired, MarkFlagsOneRequired, MarkFlagsRequiredTogether, and MarkFlagsMutuallyExclusive. Not for the shape of the command tree itself, and not for the interactive prompt behind a flag.
user-invocable: false
---

# Go CLI Commands

**What a command offers its caller: the arguments it takes, the flags it reads, and the one value an absent flag falls back to.**

## The Command Surface

Every tool starts at one baseline and buys anything past it deliberately. The baseline is `tool <command> <args> --flags`: arguments name what is acted on, flags change how, and that is the whole surface a command gets without anyone asking for more.

One extra sits on top. An interactive prompt, filling a value the user did not pass as a flag, is proposed while the surface is being designed rather than discovered halfway through an implementation, because it adds a path every later change has to keep working.

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

A flag whose value has an interactive prompt behind it is never marked required. Cobra validates required flags at `command.go:1007` and reaches `c.Run` at `:1019` (`cobra@v1.10.2`), so the invocation is rejected before the prompt could run and the flag meant to be optional is mandatory after all.

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
