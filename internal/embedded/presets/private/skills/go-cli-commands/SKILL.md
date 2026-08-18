---
name: go-cli-commands
description: Cobra command wiring for Go CLI tools - root command, simple commands, subcommand packages, and flags. Use when scaffolding main.go or cmd/root.go, adding a command or subcommand, registering flags, or setting up --debug and --for-ai. Triggers on cobra.Command, rootCmd, AddCommand, PersistentFlags, StringVarP, MarkFlagRequired, MarkFlagsMutuallyExclusive, cmd/ files, and AppVersion ldflags injection.
user-invocable: false
---

# Go CLI Commands

**How a Cobra command tree is wired: the root, the commands hanging off it, and the flags that feed them.**

CLI Only projects and Web Only projects use different roots, and a CLI + Web hybrid uses the CLI Only root plus a `serve` command.

| Aspect | CLI Only | Web Only |
|---|---|---|
| Imports | zerolog, utils, subcommand packages | cobra and subcommand packages only |
| Global flags | `--debug` and `--for-ai`, mutually exclusive | none |
| Logging setup | `setupLogs()` via `cobra.OnInitialize` | none, the standard `log` package needs no setup |
| Output | `utils.Print*` | `log.Printf` and `log.Fatalf` |

Both set `Use`, `Short`, `Version` from the `AppVersion` ldflag, and `CompletionOptions.HiddenDefaultCmd: true`.

## Entry Point

```go
package main

import "github.com/[GITHUB_USER]/REPO_NAME/cmd"

func main() {
    cmd.Execute()
}
```

`main.go` holds nothing else, so no logic ends up in the one function a test cannot call.

## Root Command, CLI Only

```go
package cmd

import (
    "fmt"
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/utils"

    featureCmd "github.com/[GITHUB_USER]/REPO_NAME/cmd/feature-cmd"
)

var AppVersion = "dev-build" // set at build time via ldflags

var debugFlag bool
var forAIFlag bool

var rootCmd = &cobra.Command{
    Use:               "appname",
    Short:             "Brief description of the application",
    Version:           AppVersion,
    CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func setupLogs() {
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime, NoColor: false}
    log.Logger = zerolog.New(output).With().Timestamp().Logger()
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    if debugFlag {
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
        utils.GlobalDebugFlag = true
    }
    if forAIFlag {
        utils.GlobalForAIFlag = true
        zerolog.SetGlobalLevel(zerolog.Disabled)
    }
}

func init() {
    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

    rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
    rootCmd.PersistentFlags().BoolVar(&forAIFlag, "for-ai", false, "AI-friendly output (plain text, piped input)")
    rootCmd.MarkFlagsMutuallyExclusive("debug", "for-ai")

    cobra.OnInitialize(setupLogs)

    rootCmd.AddCommand(serveCmd)
    rootCmd.AddCommand(featureCmd.FeatureCmd)
}
```

`setupLogs` runs through `cobra.OnInitialize` rather than at package init, because the flag values it reads are not parsed until Cobra has matched the command.

The help and completion commands are hidden so `appname --help` lists only the commands the tool actually offers.

## Root Command, Web Only

```go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var AppVersion = "dev-build"

var rootCmd = &cobra.Command{
    Use:               "appname",
    Short:             "Brief description of the application",
    Version:           AppVersion,
    CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
    rootCmd.AddCommand(serveCmd)
}
```

No `--debug`, no `--for-ai`, no zerolog, and no `utils` import, since a containerized server has one output format and adding a second one only splits its logs.

## Simple Command

A command without subcommands is defined directly in `cmd/`, with its own flag struct and `init()`. Grouping flags in a struct per command keeps two commands from colliding over a variable named `output` in the shared `cmd` package.

```go
// cmd/serve.go
package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/internal/server"
    u "github.com/[GITHUB_USER]/REPO_NAME/utils"
)

var serveFlags struct {
    port int
    host string
}

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the web server",
    Run: func(cmd *cobra.Command, args []string) {
        srv := server.New(serveFlags.host, serveFlags.port)
        if err := srv.Setup(); err != nil {
            u.PrintFatal("Failed to set up server", err)
        }
        u.PrintInfo(fmt.Sprintf("Starting server on %s:%d", serveFlags.host, serveFlags.port))
        if err := srv.Run(); err != nil {
            u.PrintFatal("Server error", err)
        }
    },
}

func init() {
    serveCmd.Flags().IntVarP(&serveFlags.port, "port", "p", 8080, "Port to listen on")
    serveCmd.Flags().StringVarP(&serveFlags.host, "host", "H", "0.0.0.0", "Host to bind to")
}
```

The same command in a Web Only project swaps `u.PrintFatal(msg, err)` for `log.Fatalf("ERROR %s: %v", msg, err)` and `u.PrintInfo` for `log.Printf("INFO ...")`, and drops the `utils` import.

## Run Function Shape

A `Run` body validates flags, builds a config struct, calls into `internal/`, and reports the result. Keeping the work in `internal/` leaves the command as the only layer that knows about flags and terminals, which is what lets the logic be tested without one.

```go
Run: func(cmd *cobra.Command, args []string) {
    if flags.required == "" {
        u.PrintFatal("--required flag is required", nil)
    }

    cfg := internal.Config{Field1: flags.field1, Field2: flags.field2}

    result, err := internal.DoThing(cfg)
    if err != nil {
        u.PrintFatal("Failed to do thing", err)
    }

    u.PrintSuccess("Thing completed")
    u.PrintGeneric(result)
}
```

## Subcommand Package

A group of related subcommands gets a package under `cmd/`. The parent command is exported and has no `Run`, so invoking it bare prints help instead of doing something arbitrary; the children are unexported and each carry a `Run`.

```go
// cmd/feature-cmd/feature.go
package featureCmd

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/internal/feature"
    u "github.com/[GITHUB_USER]/REPO_NAME/utils"
)

var createFlags struct {
    name   string
    config string
}

var FeatureCmd = &cobra.Command{
    Use:   "feature",
    Short: "Feature management commands",
}

var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new feature",
    Run: func(cmd *cobra.Command, args []string) {
        cfg := feature.CreateConfig{Name: createFlags.name, Config: createFlags.config}
        if err := feature.Create(cfg); err != nil {
            u.PrintFatal("Failed to create feature", err)
        }
        u.PrintSuccess(fmt.Sprintf("Created feature: %s", createFlags.name))
    },
}

func init() {
    FeatureCmd.AddCommand(createCmd)

    createCmd.Flags().StringVarP(&createFlags.name, "name", "n", "", "Feature name (required)")
    createCmd.MarkFlagRequired("name")
    createCmd.Flags().StringVarP(&createFlags.config, "config", "c", "", "Config file path")
}
```

## Flags

Flags are registered in `init()`, one call per flag, next to the command they belong to.

```go
cmd.Flags().StringVarP(&flags.name, "name", "n", "default", "Description")
cmd.Flags().IntVarP(&flags.count, "count", "c", 10, "Number of items")
cmd.Flags().BoolVarP(&flags.all, "all", "a", false, "Include all items")
cmd.Flags().StringSliceVarP(&flags.tags, "tag", "t", []string{}, "Tags (repeatable)")
cmd.Flags().DurationVarP(&flags.timeout, "timeout", "T", 30*time.Second, "Request timeout")
```

`MarkFlagRequired` states a requirement Cobra enforces before `Run` is reached, which produces a usage message rather than a nil dereference:

```go
cmd.Flags().StringVarP(&flags.input, "input", "i", "", "Input file (required)")
cmd.MarkFlagRequired("input")
```

An environment variable supplies a default rather than being read inside `Run`, so `--help` shows the value the command will actually use:

```go
defaultToken := os.Getenv("GITHUB_TOKEN")
cmd.Flags().StringVarP(&flags.token, "token", "t", defaultToken, "GitHub token (or GITHUB_TOKEN env)")
```

`MarkFlagsMutuallyExclusive` rejects contradictory combinations at parse time, which is where the user can still fix them:

```go
cmd.MarkFlagsMutuallyExclusive("file", "stdin")
```
