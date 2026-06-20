# Utils Package Templates

**Applies to: CLI Only projects.** CLI + Web projects use `log.Printf` with manual prefixes and do not have a `utils/` package.

## Package Structure

```
utils/
├── globals.go      # Global variables (GlobalDebugFlag, GlobalForAIFlag)
├── printer.go      # Console output abstractions
├── input.go        # User input handling
├── table.go        # Table formatting
└── config.go       # Config loading (optional)
```

## globals.go

```go
package utils

// GlobalDebugFlag is set by cobra root command when --debug is passed
var GlobalDebugFlag bool

// GlobalForAIFlag is set by cobra root command when --for-ai is passed
// When true, output uses plain text with prefixes and input reads from stdin pipe
var GlobalForAIFlag bool
```

## printer.go

Three-way branch: debug (zerolog) → AI (prefixed plain text) → human (styled lipgloss).

```go
package utils

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
)

var (
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12)) // bright blue
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10)) // bright green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9))  // bright red
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(11)) // bright yellow
)

// PrintInfo prints an info message in blue
func PrintInfo(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		fmt.Println("[INFO] " + msg)
	} else {
		fmt.Println(infoStyle.Render("→ " + msg))
	}
}

// PrintSuccess prints a success message in green
func PrintSuccess(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		fmt.Println("[OK] " + msg)
	} else {
		fmt.Println(successStyle.Render("✓ " + msg))
	}
}

// PrintError prints an error message in red (does not exit)
// Only --debug shows the underlying error; human and AI modes show only the friendly message
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

// PrintFatal prints an error message and exits
// Only --debug shows the underlying error; human and AI modes show only the friendly message
func PrintFatal(msg string, err error) {
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
	os.Exit(1)
}

// PrintWarn prints a warning message in yellow
// Only --debug shows the underlying error; human and AI modes show only the friendly message
func PrintWarn(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
	} else if GlobalForAIFlag {
		fmt.Println("[WARN] " + msg)
	} else {
		fmt.Println(warnStyle.Render("! " + msg))
	}
}

// PrintGeneric prints plain text without styling
func PrintGeneric(msg string) {
	fmt.Println(msg)
}

// --- Running and Indented functions (for output lifecycle patterns) ---

// PrintRunning prints a transient "in progress" indicator (top-level)
func PrintRunning(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		fmt.Println("[RUNNING] " + msg)
	} else {
		fmt.Println(infoStyle.Render("↻ " + msg))
	}
}

// PrintIndentedSuccess prints an indented success line (sub-task under a phase)
func PrintIndentedSuccess(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		fmt.Println("[OK] " + msg)
	} else {
		fmt.Println(successStyle.Render("  ✓ " + msg))
	}
}

// PrintIndentedError prints an indented error line (sub-task under a phase)
func PrintIndentedError(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
	} else if GlobalForAIFlag {
		fmt.Println("[ERROR] " + msg)
	} else {
		fmt.Println(errorStyle.Render("  ✗ " + msg))
	}
}

// PrintIndentedWarn prints an indented warning line (sub-task under a phase)
func PrintIndentedWarn(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
	} else if GlobalForAIFlag {
		fmt.Println("[WARN] " + msg)
	} else {
		fmt.Println(warnStyle.Render("  ! " + msg))
	}
}

// PrintIndentedRunning prints an indented "in progress" indicator (sub-task)
func PrintIndentedRunning(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		fmt.Println("[RUNNING] " + msg)
	} else {
		fmt.Println(infoStyle.Render("  ↻ " + msg))
	}
}

// --- Line Clearing ---

// ClearLines removes N lines of terminal output (ANSI escape).
// No-op in debug and AI modes (all output persists for logging/parsing).
func ClearLines(n int) {
	if GlobalDebugFlag || GlobalForAIFlag {
		return
	}
	for range n {
		fmt.Print("\033[A\033[2K")
	}
}

// ClearPreviousLine removes the single line above the cursor.
// Used by PrintProgress to overwrite itself on each tick.
func ClearPreviousLine() {
	if GlobalDebugFlag || GlobalForAIFlag {
		return
	}
	fmt.Print("\033[A\033[2K")
}

// --- Progress Indicator ---

// PrintProgress overwrites the previous line with an indented progress bar.
// First call prints a new line; subsequent calls clear and reprint.
// In AI mode, prints a new line each tick (no clearing).
// In debug mode, logs percentage as a structured field.
func PrintProgress(label string, percent int) {
	if percent > 100 {
		percent = 100
	}

	if GlobalDebugFlag {
		log.Info().Int("percent", percent).Msg(label)
		return
	}

	if GlobalForAIFlag {
		fmt.Printf("[PROGRESS] %s: %d%%\n", label, percent)
		return
	}

	const barWidth = 10
	filled := barWidth * percent / 100
	empty := barWidth - filled

	bar := strings.Repeat("⣿", filled) + strings.Repeat("⣀", empty)
	fmt.Println(infoStyle.Render(fmt.Sprintf("  ↻ %s: %s %d%%", label, bar, percent)))
}
```

**Note:** `PrintProgress` requires `"strings"` in the import block.

### Error Discipline

The `msg` parameter is the human-readable label. The `err` parameter is the Go error object.

- **Always pass the actual `err`** when one is available. Never bake `err` into the `msg` string via `fmt.Sprintf`.
- Debug mode uses zerolog `.Err(err)` which logs the error as a structured field — this only works if `err` is passed as the error parameter.
- Human and AI modes show only `msg`. The `err` is exclusively for debug introspection.
- Pass `nil` for `err` only when there genuinely is no underlying error (validation failures, summary messages, informational warnings).

```go
// CORRECT — err passed as error parameter, debug mode sees it via .Err(err)
utils.PrintFatal("git not found in PATH", err)
utils.PrintIndentedError(toolName, result.Err)

// WRONG — err baked into msg, debug .Err(err) gets nil
utils.PrintIndentedError(fmt.Sprintf("%s: %s", toolName, result.Err), nil)
utils.PrintFatal("failed: "+err.Error(), nil)
```

### Subprocess Error Capture

When running external commands outside of `utils.RunCmd`, capture stderr so the error contains the real failure reason, not just "exit status 1":

```go
cmd := exec.Command("sudo", "cp", src, dst)
var stderr strings.Builder
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
    detail := strings.TrimSpace(stderr.String())
    if detail != "" {
        err = fmt.Errorf("%s: %w", detail, err)
    }
    utils.PrintFatal("failed to copy binary", err)
}
```

`utils.RunCmd` already handles this — it captures both stdout and stderr into the returned error. Only apply this pattern for direct `exec.Command` calls that don't go through `RunCmd`.

## input.go

In AI mode, input functions read from stdin pipe instead of launching interactive Bubbletea TUI.

- **Single-line** (`PromptInput`, `PromptPassword`): Bubbletea `textinput` (human) / `ReadPipedLine` (AI)
- **Multi-line** (`PromptTextArea`): Bubbletea `textarea` with Ctrl+D submit (human) / `ReadPipedInput` bulk read (AI)

AI mode examples:
- Single-line: `echo "my input" | toolname command --for-ai`
- Sequential prompts: `echo -e "username\npassword" | toolname login --for-ai`
- Multi-line: `echo "line1\nline2\nline3" | toolname edit --for-ai`

```go
package utils

import (
	"bufio"
	"os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// stdinScanner is shared across calls so sequential PromptInput/PromptPassword
// calls each read the next line instead of draining all of stdin on the first call
var stdinScanner *bufio.Scanner

func getStdinScanner() *bufio.Scanner {
	if stdinScanner == nil {
		stdinScanner = bufio.NewScanner(os.Stdin)
	}
	return stdinScanner
}

// ReadPipedInput reads all remaining input from stdin pipe (bulk read)
// Returns empty string if stdin is not a pipe
func ReadPipedInput() string {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return ""
	}
	scanner := getStdinScanner()
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// ReadPipedLine reads a single line from stdin pipe (for sequential prompts)
// Returns empty string if stdin is not a pipe or no more lines
func ReadPipedLine() string {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return ""
	}
	scanner := getStdinScanner()
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

type inputModel struct {
	textInput textinput.Model
	done      bool
	value     string
	initCmd   tea.Cmd
}

func (m inputModel) Init() tea.Cmd {
	return m.initCmd
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textInput.View())
}

// PromptInput displays an inline prompt and returns user input
// In AI mode, reads a single line from stdin pipe instead of launching TUI
func PromptInput(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = prompt + " "
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return strings.TrimSpace(result.value), nil
}

// PromptPassword displays an inline password prompt (masked input)
// In AI mode, reads a single line from stdin pipe instead of launching TUI
// Security note: caller must ensure the returned value is never passed to Print functions
func PromptPassword(prompt string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = "••••••••"
	ti.Prompt = prompt + " "
	ti.EchoMode = textinput.EchoPassword
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return result.value, nil
}

type textAreaModel struct {
	textarea textarea.Model
	done     bool
	value    string
	initCmd  tea.Cmd
}

func (m textAreaModel) Init() tea.Cmd {
	return m.initCmd
}

func (m textAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+d":
			m.value = m.textarea.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textAreaModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textarea.View() + "\n(Ctrl+D to submit, Esc to cancel)")
}

// PromptTextArea displays a multi-line text area and returns user input
// In AI mode, reads all remaining stdin pipe input instead of launching TUI
func PromptTextArea(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedInput(), nil
	}

	PrintInfo(prompt)

	ta := textarea.New()
	ta.Placeholder = placeholder
	focusCmd := ta.Focus()

	m := textAreaModel{textarea: ta, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(textAreaModel)
	return strings.TrimSpace(result.value), nil
}
```

## table.go

In AI mode, renders markdown tables instead of styled lipgloss box-drawing tables.

```go
package utils

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.ANSIColor(15)).
		Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
		Foreground(lipgloss.ANSIColor(7)).
		Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.ANSIColor(8))
)

// PrintTable prints a formatted table with headers and rows
// In AI mode, renders as a markdown table for easy parsing
// Note: table.HeaderRow == -1, data rows start at 0
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

// printMarkdownTable renders headers and rows as a markdown table
func printMarkdownTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	fmt.Println("| " + strings.Join(escapeCells(headers), " | ") + " |")
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Println("| " + strings.Join(seps, " | ") + " |")
	for _, row := range rows {
		fmt.Println("| " + strings.Join(escapeCells(row), " | ") + " |")
	}
}

// escapeCells escapes pipe characters in cell values for valid markdown tables
func escapeCells(cells []string) []string {
	escaped := make([]string, len(cells))
	for i, cell := range cells {
		escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
	}
	return escaped
}
```

## config.go (Optional - When Extended Config Needed)

```go
package utils

import (
	"os"

	"github.com/goccy/go-yaml"
)

// Config represents application configuration
// Customize fields per project
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// LoadConfig loads configuration with priority:
// 1. Environment variables (highest)
// 2. CLI flags (passed as overrides)
// 3. YAML file (if path provided and exists)
// 4. Defaults (lowest)
func LoadConfig(configPath string, flagOverrides map[string]interface{}) (*Config, error) {
	// Start with defaults
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
	}

	// Load from YAML if path provided and file exists
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
		// If file doesn't exist, continue with defaults (don't error)
	}

	// Apply CLI flag overrides
	if port, ok := flagOverrides["port"].(int); ok && port != 0 {
		cfg.Server.Port = port
	}
	if host, ok := flagOverrides["host"].(string); ok && host != "" {
		cfg.Server.Host = host
	}

	// Apply environment variable overrides (highest priority)
	if envHost := os.Getenv("APP_HOST"); envHost != "" {
		cfg.Server.Host = envHost
	}

	return cfg, nil
}

// GetEnv returns environment variable value or default
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```
