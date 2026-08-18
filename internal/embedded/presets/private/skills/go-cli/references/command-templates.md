# Command Templates

Full Cobra code templates for CLI Only and Web Only projects (a CLI + Web hybrid uses the CLI Only root plus the Web Only `serve` command). The `SKILL.md` describes when to use each; copy the concrete code from here. Replace `[GITHUB_USER]`/`REPO_NAME`/`appname` placeholders.

## main.go (Entry Point)

```go
package main

import "github.com/[GITHUB_USER]/REPO_NAME/cmd"

func main() {
    cmd.Execute()
}
```

## cmd/root.go — CLI Only

Full root: zerolog, utils, `--debug`, `--for-ai`, `setupLogs`.

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

    // Import subcommand packages
    featureCmd "github.com/[GITHUB_USER]/REPO_NAME/cmd/feature-cmd"
)

var AppVersion = "dev-build"  // Set at build time via ldflags
var debugFlag bool
var forAIFlag bool

var rootCmd = &cobra.Command{
    Use:     "appname",
    Short:   "Brief description of the application",
    Version: AppVersion,
    CompletionOptions: cobra.CompletionOptions{
        HiddenDefaultCmd: true,
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func setupLogs() {
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
    if forAIFlag {
        utils.GlobalForAIFlag = true
        zerolog.SetGlobalLevel(zerolog.Disabled)
    }
}

func init() {
    // Hide default help and completion commands
    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

    // Global flags (mutually exclusive)
    rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
    rootCmd.PersistentFlags().BoolVar(&forAIFlag, "for-ai", false, "AI-friendly output (plain text, piped input)")
    rootCmd.MarkFlagsMutuallyExclusive("debug", "for-ai")

    // Initialize logging on startup
    cobra.OnInitialize(setupLogs)

    // Add commands (simple commands)
    rootCmd.AddCommand(serveCmd)
    rootCmd.AddCommand(versionCmd)

    // Add subcommand packages
    rootCmd.AddCommand(featureCmd.FeatureCmd)
}
```

## cmd/root.go — Web Only

No debug/for-ai flags, no setupLogs, no zerolog, no utils import.

```go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    // Import subcommand packages
    featureCmd "github.com/[GITHUB_USER]/REPO_NAME/cmd/feature-cmd"
)

var AppVersion = "dev-build"  // Set at build time via ldflags

var rootCmd = &cobra.Command{
    Use:     "appname",
    Short:   "Brief description of the application",
    Version: AppVersion,
    CompletionOptions: cobra.CompletionOptions{
        HiddenDefaultCmd: true,
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    // Hide default help and completion commands
    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

    // Add commands
    rootCmd.AddCommand(serveCmd)

    // Add subcommand packages
    rootCmd.AddCommand(featureCmd.FeatureCmd)
}
```

## Simple Command — CLI Only

For commands without subcommands, define in `cmd/` directly. Uses utils print functions:

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
            u.PrintFatal("Failed to setup server", err)
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

## Simple Command — Web Only (serve example)

Uses `log.Printf` with manual prefixes and `log.Fatalf` for fatal errors. No utils import:

```go
// cmd/serve.go
package cmd

import (
    "log"

    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/internal/server"
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
            log.Fatalf("ERROR Failed to setup: %v", err)
        }
        log.Printf("INFO Starting on %s:%d", serveFlags.host, serveFlags.port)
        if err := srv.Run(); err != nil {
            log.Fatalf("ERROR Server error: %v", err)
        }
    },
}

func init() {
    serveCmd.Flags().IntVarP(&serveFlags.port, "port", "p", 8080, "Port to listen on")
    serveCmd.Flags().StringVarP(&serveFlags.host, "host", "H", "0.0.0.0", "Host to bind to")
}
```

## Subcommand Package

For grouped commands with subcommands, create a package under `cmd/`:

```go
// cmd/feature-cmd/feature.go
package featureCmd

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/internal/feature"
    u "github.com/[GITHUB_USER]/REPO_NAME/utils"
)

// Flag structs - one per subcommand
var createFlags struct {
    name   string
    config string
}

var listFlags struct {
    format string
    all    bool
}

// Parent command (exported, no Run function)
var FeatureCmd = &cobra.Command{
    Use:   "feature",
    Short: "Feature management commands",
}

// Subcommands (unexported, have Run function)
var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new feature",
    Run: func(cmd *cobra.Command, args []string) {
        if createFlags.name == "" {
            u.PrintFatal("--name flag is required", nil)
        }
        cfg := feature.CreateConfig{
            Name:   createFlags.name,
            Config: createFlags.config,
        }
        if err := feature.Create(cfg); err != nil {
            u.PrintFatal("Failed to create feature", err)
        }
        u.PrintSuccess(fmt.Sprintf("Created feature: %s", createFlags.name))
    },
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List all features",
    Run: func(cmd *cobra.Command, args []string) {
        items, err := feature.List(listFlags.all)
        if err != nil {
            u.PrintFatal("Failed to list features", err)
        }
        if listFlags.format == "table" {
            u.PrintTable([]string{"Name", "Status"}, items)
        } else {
            for _, item := range items {
                u.PrintGeneric(item[0])
            }
        }
    },
}

func init() {
    // Add subcommands to parent
    FeatureCmd.AddCommand(createCmd)
    FeatureCmd.AddCommand(listCmd)

    // Flags for create
    createCmd.Flags().StringVarP(&createFlags.name, "name", "n", "", "Feature name (required)")
    createCmd.Flags().StringVarP(&createFlags.config, "config", "c", "", "Config file path")

    // Flags for list
    listCmd.Flags().StringVarP(&listFlags.format, "format", "f", "table", "Output format (table, plain)")
    listCmd.Flags().BoolVarP(&listFlags.all, "all", "a", false, "Show all including hidden")
}
```

## Login Command — CLI Only

For CLI tools with OAuth authentication, the login command uses mutually exclusive flags to select the login mode. The auth logic lives in `internal/auth/` (see `go-backend`); the command just maps flags to mode and handles output.

```go
// cmd/login.go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/REPO_NAME/internal/auth"
    u "github.com/[GITHUB_USER]/REPO_NAME/utils"
)

var loginFlags struct {
    deviceLogin bool
    manual      bool
}

var loginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with [SERVICE_NAME]",
    Run: func(cmd *cobra.Command, args []string) {
        config, err := auth.LoadCredentials()
        if err != nil {
            u.PrintFatal("failed to load credentials", err)
        }

        mode := "default"
        if loginFlags.deviceLogin {
            mode = "device"
        } else if loginFlags.manual {
            mode = "manual"
        }

        token, err := auth.Login(config, mode)
        if err != nil {
            u.PrintFatal("login failed", err)
        }

        _ = token
        u.PrintSuccess("authenticated successfully — token saved")
    },
}

func init() {
    rootCmd.AddCommand(loginCmd)

    loginCmd.Flags().BoolVar(&loginFlags.deviceLogin, "device-login", false, "Use device code flow (for headless/SSH environments)")
    loginCmd.Flags().BoolVar(&loginFlags.manual, "manual", false, "Manually paste authorization code (last resort)")
    loginCmd.MarkFlagsMutuallyExclusive("device-login", "manual")
}
```

**Modes:**
- `appname login` — default, opens browser with localhost callback
- `appname login --device-login` — shows URL + code for headless/SSH
- `appname login --manual` — paste authorization code (last resort)

If the provider does not support device authorization (RFC 8628), omit `--device-login` and keep only default and `--manual`.

## Terminal Colors (Lipgloss) — CLI Only

Use ANSI standard color indices (0-15) instead of hardcoded hex values. These indices are remapped by the user's terminal theme, so output adapts to Dracula, Catppuccin, Solarized, or any custom scheme automatically. Hardcoded hex colors (e.g. `#89b4fa`) bypass the theme.

```go
// ANSI 0-15 map to the terminal's configured theme colors.
// Bright variants (8-15) are preferred for foreground text readability.
var (
    ColorBlue    = lipgloss.ANSIColor(12) // Bright Blue
    ColorGreen   = lipgloss.ANSIColor(10) // Bright Green
    ColorRed     = lipgloss.ANSIColor(9)  // Bright Red
    ColorYellow  = lipgloss.ANSIColor(11) // Bright Yellow
    ColorMagenta = lipgloss.ANSIColor(13) // Bright Magenta
    ColorCyan    = lipgloss.ANSIColor(14) // Bright Cyan
    ColorFg      = lipgloss.ANSIColor(15) // Bright White - primary text
    ColorMuted   = lipgloss.ANSIColor(7)  // White - secondary/dimmed text
    ColorChrome  = lipgloss.ANSIColor(8)  // Bright Black - borders, dim UI
)

// Common styles
var (
    InfoStyle    = lipgloss.NewStyle().Foreground(ColorBlue)
    SuccessStyle = lipgloss.NewStyle().Foreground(ColorGreen)
    ErrorStyle   = lipgloss.NewStyle().Foreground(ColorRed)
    WarnStyle    = lipgloss.NewStyle().Foreground(ColorYellow)
)
```
