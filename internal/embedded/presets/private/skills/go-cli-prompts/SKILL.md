---
name: go-cli-prompts
description: Interactive input for Go CLI tools - the utils prompt helpers and their piped-stdin equivalents under --for-ai. Use when a command needs to ask the user something, when building or changing utils/input.go, or when adding a password, free-text, or selection prompt. Triggers on PromptInput, PromptPassword, PromptTextArea, PromptSelect, PromptMultiSelect, ReadPipedLine, ReadPipedInput, bubbletea textinput, textarea, and stdin piping into a CLI.
user-invocable: false
---

# Go CLI Prompts

**Everything a CLI Only tool asks the user goes through a `utils` prompt helper, which is a bubbletea TUI for a human and a stdin read under `--for-ai`.**

Branching inside the helper rather than at each call site is what keeps every command scriptable without each command remembering to handle a pipe.

```
echo "my input" | toolname command --for-ai
echo -e "username\npassword" | toolname login --for-ai
```

| Helper | Human | AI (`--for-ai`) | Returns |
|---|---|---|---|
| `PromptInput(prompt, placeholder)` | single-line textinput | one line from stdin | `(string, error)` |
| `PromptPassword(prompt)` | masked textinput | one line from stdin | `(string, error)` |
| `PromptTextArea(prompt, placeholder)` | multi-line textarea, Ctrl+D submits | all remaining stdin | `(string, error)` |
| `PromptSelect(label, options)` | single-choice list | a 1-based index from stdin | `(int, error)`, `-1` on cancel |
| `PromptMultiSelect(label, options)` | multi-choice list, space toggles | comma-separated indices or `none` | `(map[int]bool, error)`, `nil` on cancel |

Every cancel path is a clean no-op abort: `idx < 0` from `PromptSelect` and a `nil` map from `PromptMultiSelect` mean the user pressed Escape, and treating that as an empty selection would run the operation they just declined.

A password returned from `PromptPassword` never reaches a `Print` function, because the AI and debug tiers write it somewhere durable.

## Reading Piped Input

One scanner is shared across calls so that sequential prompts each consume the next line. A fresh `bufio.Scanner` per call would drain the whole pipe on the first prompt and leave the rest empty.

```go
var stdinScanner *bufio.Scanner

func getStdinScanner() *bufio.Scanner {
    if stdinScanner == nil {
        stdinScanner = bufio.NewScanner(os.Stdin)
    }
    return stdinScanner
}

// ReadPipedLine returns one line, or "" when stdin is a terminal or exhausted.
func ReadPipedLine() string {
    fi, err := os.Stdin.Stat()
    if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
        return ""
    }
    if s := getStdinScanner(); s.Scan() {
        return strings.TrimSpace(s.Text())
    }
    return ""
}

// ReadPipedInput drains the rest of stdin as one string.
func ReadPipedInput() string {
    fi, err := os.Stdin.Stat()
    if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
        return ""
    }
    var lines []string
    s := getStdinScanner()
    for s.Scan() {
        lines = append(lines, s.Text())
    }
    return strings.TrimSpace(strings.Join(lines, "\n"))
}
```

The `ModeCharDevice` check distinguishes a pipe from a terminal, so a tool run with `--for-ai` but no pipe returns empty rather than blocking forever on a read nobody will satisfy.

## Single-Line Input

One bubbletea model serves both `PromptInput` and `PromptPassword`; the password variant only sets `EchoMode`.

```go
type inputModel struct {
    textInput textinput.Model
    done      bool
    value     string
    initCmd   tea.Cmd
}

func (m inputModel) Init() tea.Cmd { return m.initCmd }

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

func PromptInput(prompt, placeholder string) (string, error) {
    if GlobalForAIFlag {
        return ReadPipedLine(), nil
    }
    ti := textinput.New()
    ti.Placeholder = placeholder
    ti.Prompt = prompt + " "
    m := inputModel{textInput: ti, initCmd: ti.Focus()}

    final, err := tea.NewProgram(m).Run()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(final.(inputModel).value), nil
}

func PromptPassword(prompt string) (string, error) {
    if GlobalForAIFlag {
        return ReadPipedLine(), nil
    }
    ti := textinput.New()
    ti.Placeholder = "••••••••"
    ti.Prompt = prompt + " "
    ti.EchoMode = textinput.EchoPassword
    m := inputModel{textInput: ti, initCmd: ti.Focus()}

    final, err := tea.NewProgram(m).Run()
    if err != nil {
        return "", err
    }
    return final.(inputModel).value, nil
}
```

`PromptPassword` skips the `TrimSpace` that `PromptInput` applies, since a trailing space can be part of a password and silently removing it produces an authentication failure nobody can explain.

## Multi-Line Input

`PromptTextArea` submits on Ctrl+D rather than Enter, because Enter has to stay available for the newlines that make the field multi-line.

```go
func (m textAreaModel) View() tea.View {
    if m.done {
        return tea.NewView("")
    }
    return tea.NewView(m.textarea.View() + "\n(Ctrl+D to submit, Esc to cancel)")
}

func PromptTextArea(prompt, placeholder string) (string, error) {
    if GlobalForAIFlag {
        return ReadPipedInput(), nil
    }
    PrintInfo(prompt)

    ta := textarea.New()
    ta.Placeholder = placeholder
    m := textAreaModel{textarea: ta, initCmd: ta.Focus()}

    final, err := tea.NewProgram(m).Run()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(final.(textAreaModel).value), nil
}
```

The `Update` method mirrors `inputModel`, matching `"ctrl+d"` for submit instead of `"enter"`.

## Selection

Both selectors share one model that tracks a cursor and, for the multi variant, a set of toggled indices. Reusing the shared helpers rather than hand-rolling a bubbletea model per command is what keeps the key bindings identical everywhere: arrows or `j`/`k` to move, Enter to confirm, Escape or Ctrl+C to cancel, and space to toggle in the multi variant.

```go
type selectModel struct {
    label    string
    options  []string
    cursor   int
    chosen   map[int]bool // nil for single-choice
    multi    bool
    done     bool
    canceled bool
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    key, ok := msg.(tea.KeyPressMsg)
    if !ok {
        return m, nil
    }
    switch key.String() {
    case "up", "k":
        m.cursor = max(m.cursor-1, 0)
    case "down", "j":
        m.cursor = min(m.cursor+1, len(m.options)-1)
    case " ":
        if m.multi {
            m.chosen[m.cursor] = !m.chosen[m.cursor]
        }
    case "enter":
        m.done = true
        return m, tea.Quit
    case "esc", "ctrl+c":
        m.canceled = true
        return m, tea.Quit
    }
    return m, nil
}

// View renders the label, then one line per option: a "> " marker on the cursor
// row, and for the multi variant a "[x]"/"[ ]" box reflecting m.chosen.
```

```go
func PromptSelect(label string, options []string) (int, error) {
    if GlobalForAIFlag {
        line := ReadPipedLine()
        if line == "" {
            return -1, nil
        }
        n, err := strconv.Atoi(line)
        if err != nil || n < 1 || n > len(options) {
            return -1, fmt.Errorf("expected a number between 1 and %d, got %q", len(options), line)
        }
        return n - 1, nil // stdin indices are 1-based for the human writing the pipe
    }

    final, err := tea.NewProgram(selectModel{label: label, options: options}).Run()
    if err != nil {
        return -1, err
    }
    m := final.(selectModel)
    if m.canceled {
        return -1, nil
    }
    return m.cursor, nil
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
    if GlobalForAIFlag {
        line := ReadPipedLine()
        if line == "" || line == "none" {
            return map[int]bool{}, nil // an explicit empty choice, distinct from cancel
        }
        chosen := map[int]bool{}
        for part := range strings.SplitSeq(line, ",") {
            n, err := strconv.Atoi(strings.TrimSpace(part))
            if err != nil || n < 1 || n > len(options) {
                return nil, fmt.Errorf("expected comma-separated numbers between 1 and %d, got %q", len(options), line)
            }
            chosen[n-1] = true
        }
        return chosen, nil
    }

    final, err := tea.NewProgram(selectModel{
        label: label, options: options, chosen: map[int]bool{}, multi: true,
    }).Run()
    if err != nil {
        return nil, err
    }
    m := final.(selectModel)
    if m.canceled {
        return nil, nil
    }
    return m.chosen, nil
}
```

Stdin indices are 1-based while the returned index is 0-based, because the person writing `echo "2" | tool cmd --for-ai` is counting the options they can see on screen.

`none` and an empty map are a deliberate choice of nothing, while `nil` is a cancel. Collapsing the two would make an aborted prompt indistinguishable from a user who selected no options on purpose.
