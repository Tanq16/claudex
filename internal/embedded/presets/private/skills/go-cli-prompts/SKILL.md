---
name: go-cli-prompts
description: Interactive input for Go CLI tools - the symmetry rule that every prompted value also has a flag, the utils prompt helpers built on one bubbletea text model and one list model, $EDITOR for a value that spans lines, a masked secret that reveals its tail, and the ErrNoTerminal contract that keeps every command headless. Use when a command needs to ask the user something, when building or changing utils/input.go, or when adding a text, secret, selection, or multi-line prompt. Triggers on PromptInput, PromptPassword, PromptSelect, PromptMultiSelect, PromptEditor, MaskSecret, IsInteractive, ErrNoTerminal, StdinIsTerminal, textinput, tea.NewProgram, $EDITOR, and MarkFlagRequired on a prompted flag.
user-invocable: false
---

# Go CLI Prompts

**Every value a prompt collects also arrives through a flag, every prompt is one line or one list, and a value longer than a line comes from a file or the user's editor.**

## The Symmetry Rule

A prompt exists only for a value that a flag also supplies. A tool driven from a script, a scheduler, or a chat interface has no terminal, so a value reachable only by a prompt is a value those callers cannot provide at all.

The rule is one-directional, and most flags have no prompt. Every prompt is one more path that each later change to the command has to keep working.

A flag whose value has a prompt is never marked required. Cobra validates required flags at `command.go:1007` and reaches `c.Run` at `:1019` (`cobra@v1.10.2`), so the invocation is rejected before the prompt could run and the flag meant to be optional is mandatory after all. The requirement is enforced inside `Run` instead, at the point the value turns out to be missing.

A value resolves in two steps, and the absence of the flag is the whole trigger.

| The flag holds | The value comes from |
|---|---|
| a value | the flag itself |
| nothing | the prompt |
| nothing, and there is no terminal | an error naming the flag that would have worked |

```go
account := launchFlags.account
if account == "" {
    idx, err := u.PromptSelect("Account", labels)
    if errors.Is(err, u.ErrNoTerminal) {
        u.PrintFatal("launch needs --account when there is no interactive terminal", nil)
    }
    if err != nil {
        u.PrintFatal("TUI error", err)
    }
    if idx < 0 {
        return
    }
    account = accounts[idx]
}
```

A value is validated once, after it resolves, whichever channel it came from. A check placed inside the prompt covers only the interactive path, and the flag then accepts what the prompt would have refused.

## No Terminal

Every helper, the editor included, starts only when both stdin and stdout are terminals, and returns `ErrNoTerminal` otherwise. With stdout redirected, a prompt draws into the file while the user types blind, and an editor writes its screen into the output.

```go
var ErrNoTerminal = errors.New("no interactive terminal")

func IsInteractive() bool {
    return StdinIsTerminal && StdoutIsTerminal
}
```

The message reads `<command> needs <flag> when there is no interactive terminal`. Naming the flag ends the problem in one line, where a message saying only that input was expected sends the caller to `--help` to work out which flag it was.

A prompt with a documented default takes that default instead of failing, since a default is an answer the command already has.

A command that cannot work without a terminal at all, because it hands the session to another program, checks `IsInteractive` once at the top of `Run`. Checking at each prompt instead reaches the first one before failing, after the command may already have acted.

## The Helpers

| Helper | Behavior | Returns |
|---|---|---|
| `PromptInput(prompt, placeholder)` | one line, scrolling sideways past the terminal edge | `(string, error)`, `""` on cancel |
| `PromptPassword(prompt)` | one masked line | `(string, error)`, `""` on cancel |
| `PromptSelect(label, options)` | single choice | `(int, error)`, `-1` on cancel |
| `PromptMultiSelect(label, options)` | one or more choices, space toggles | `(map[int]bool, error)`, `nil` on cancel |
| `PromptEditor(name, initial)` | the user's editor on a temporary file | `(string, error)`, `""` when saved empty |

Every cancel path is a clean no-op abort. An empty string, `idx < 0`, and a `nil` map each mean the user pressed Escape, and treating that as an empty answer runs the operation they just declined.

No helper confirms an empty answer. Enter does nothing on a blank line or on a list with nothing toggled, so the empty return means cancel and nothing else, and a flag never has to express an empty value the prompt allowed.

Reusing these helpers rather than building a prompt per command keeps the key bindings identical across a tool. Arrows or `j`/`k` move, Enter confirms, Escape or Ctrl+C cancels, and space toggles in the multi variant.

A yes-or-no question is `PromptSelect` over `No` then `Yes`, with a boolean flag such as `--yes` supplying the answer. No sits under the cursor, so a reflexive Enter declines. A confirmation has no default, so without a terminal it fails naming `--yes`.

The helpers are two bubbletea models over `bubbles/textinput` and lipgloss, and no form library. A form library brings its own theme, key map, width, and error rendering, and each one has to be overridden before a prompt looks and behaves like the rest of the tool.

The models need `charm.land/bubbletea/v2` v2.1.0 or later. Under v2.0.9 the inline renderer does not return to the top of a frame that shrinks on exit, so a list collapsing to its answer leaves every row above the cursor on screen.

## One Line

`PromptInput` and `PromptPassword` share one model, and the secret variant only changes the echo and the settled line.

The value is trimmed when Enter is pressed, and that one trimmed value is both what settles on screen and what is returned. textinput turns a pasted newline into a space (`bubbles@v2.2.1 textinput/textinput.go:301-302`), so an untrimmed secret settles showing a tail that differs from the one stored.

An answered prompt settles into one line, clipped with `…` at the terminal width, and a cancelled one leaves nothing. Settling on the input's own view would show the scrolled window, and the middle of a long command reads as a different command. The settled line is the program's final frame rather than a printer call, so a secret never passes through a function that can branch to the debug log.

```go
type inputModel struct {
    input  textinput.Model
    focus  tea.Cmd
    secret bool
    done   bool
    value  string
}

func (m inputModel) Init() tea.Cmd { return m.focus }

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyPressMsg); ok {
        switch key.String() {
        case "enter":
            value := strings.TrimSpace(m.input.Value())
            if value == "" {
                return m, nil
            }
            m.done, m.value = true, value
            return m, tea.Quit
        case "esc", "ctrl+c":
            m.done = true
            return m, tea.Quit
        }
    }
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m inputModel) View() tea.View {
    switch {
    case !m.done:
        return tea.NewView(m.input.View())
    case m.value == "":
        return tea.NewView("")
    case m.secret:
        return tea.NewView(m.input.Prompt + MaskSecret(m.value) + "\n")
    }
    return tea.NewView(ansi.Truncate(m.input.Prompt+m.value, Width()-1, "…") + "\n")
}
```

The input's width is the terminal's less the prompt, so a long paste scrolls sideways and its end stays in view. With no width set, the renderer clips the line at the terminal edge and hides the tail, which is where a truncated paste shows.

`Width()` is the one terminal width in `utils`, shared with the progress meter: `term.GetSize` on stdout, then `COLUMNS` when it is 24 or more, then 80.

The cursor is textinput's virtual one. The real terminal cursor would take the user's own shape, but `Cursor()` adds the position without subtracting the scroll offset (`textinput.go:924`), so after a long paste, moving left leaves it pinned at the right edge while the insertion point moves.

The styles start from zero rather than textinput's defaults, with only the placeholder set. The defaults carry 256-color indices, such as `240` for the placeholder (`styles.go:18`), which the user's terminal theme does not remap.

```go
func runInput(prompt, placeholder string, secret bool) (string, error) {
    if !IsInteractive() {
        return "", ErrNoTerminal
    }
    ti := textinput.New()
    ti.Prompt = labelStyle.Render(prompt) + " "
    ti.Placeholder = placeholder
    ti.SetWidth(Width() - lipgloss.Width(ti.Prompt) - 1)
    ti.SetStyles(textinput.Styles{Focused: textinput.StyleState{Placeholder: hintStyle}})
    if secret {
        ti.EchoMode = textinput.EchoPassword
        ti.EchoCharacter = '•'
    }
    final, err := tea.NewProgram(inputModel{input: ti, focus: ti.Focus(), secret: secret}).Run()
    if err != nil {
        return "", err
    }
    return final.(inputModel).value, nil
}

func PromptInput(prompt, placeholder string) (string, error) {
    return runInput(prompt, placeholder, false)
}

func PromptPassword(prompt string) (string, error) {
    return runInput(prompt, "", true)
}
```

## Lists

`PromptSelect` and `PromptMultiSelect` share one model that tracks a cursor, a window, and, in the multi variant, the set of toggled indices.

A list shows at most ten rows, fewer when the terminal is shorter, and scrolls under the cursor. A frame taller than the terminal cannot be redrawn in place. The position prints beside the hint once the list overflows, which is what tells the user it continues past the window.

Space matches as `"space"`. bubbletea v2 names the key rather than sending the character, so a `case " ":` never fires and the multi variant cannot toggle anything.

An untoggled row is deleted from the set rather than stored as `false`, so the returned map holds only what was chosen.

The four styles are ANSI indices, which the user's terminal theme remaps: the label in 12, the row under the cursor in 10, other rows in 15, and the hint in 8.

```go
const listRows = 10

type selectModel struct {
    label   string
    options []string
    rows    int
    cursor  int
    top     int
    chosen  map[int]bool
    multi   bool
    done    bool
    picked  bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.rows = max(min(listRows, msg.Height-2), 1)
    case tea.KeyPressMsg:
        switch msg.String() {
        case "esc", "ctrl+c":
            m.done = true
            return m, tea.Quit
        case "enter":
            if m.multi && len(m.chosen) == 0 {
                return m, nil
            }
            m.done, m.picked = true, true
            return m, tea.Quit
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.options)-1)
        case "space":
            if m.multi && m.chosen[m.cursor] {
                delete(m.chosen, m.cursor)
            } else if m.multi {
                m.chosen[m.cursor] = true
            }
        }
    }
    m.top = min(m.top, m.cursor)
    m.top = max(m.top, m.cursor-m.rows+1)
    return m, nil
}

func (m selectModel) View() tea.View {
    if m.done && !m.picked {
        return tea.NewView("")
    }
    if m.done {
        return tea.NewView(labelStyle.Render(m.label) + " " + m.answer() + "\n")
    }
    var b strings.Builder
    b.WriteString(labelStyle.Render(m.label) + "\n")
    for i := m.top; i < min(m.top+m.rows, len(m.options)); i++ {
        marker, style := "  ", optionStyle
        if i == m.cursor {
            marker, style = "› ", activeStyle
        }
        row := m.options[i]
        if m.multi && m.chosen[i] {
            row = "[x] " + row
        } else if m.multi {
            row = "[ ] " + row
        }
        b.WriteString(style.Render(marker+row) + "\n")
    }
    hint := "enter select · esc cancel"
    if m.multi {
        hint = "space toggle · enter confirm · esc cancel"
    }
    if len(m.options) > m.rows {
        hint = fmt.Sprintf("%d/%d · %s", m.cursor+1, len(m.options), hint)
    }
    b.WriteString(hintStyle.Render(hint))
    return tea.NewView(b.String())
}

func (m selectModel) answer() string {
    if !m.multi {
        return m.options[m.cursor]
    }
    var names []string
    for i, opt := range m.options {
        if m.chosen[i] {
            names = append(names, opt)
        }
    }
    return strings.Join(names, ", ")
}
```

```go
func runSelect(m selectModel) (selectModel, error) {
    m.rows = listRows
    final, err := tea.NewProgram(m).Run()
    if err != nil {
        return m, err
    }
    return final.(selectModel), nil
}

func PromptSelect(label string, options []string) (int, error) {
    if !IsInteractive() {
        return -1, ErrNoTerminal
    }
    m, err := runSelect(selectModel{label: label, options: options})
    if err != nil || !m.picked {
        return -1, err
    }
    return m.cursor, nil
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
    if !IsInteractive() {
        return nil, ErrNoTerminal
    }
    m, err := runSelect(selectModel{label: label, options: options, chosen: map[int]bool{}, multi: true})
    if err != nil || !m.picked {
        return nil, err
    }
    return m.chosen, nil
}
```

## Long Values

A value that can span lines arrives through a flag naming a file, and the prompt behind that flag is the user's editor. An editor is the right surface for composing text, and a terminal widget is not.

Both paths trim, so the same text stored from a file and from the editor is the same value.

```go
var body string
if sendFlags.bodyFile != "" {
    data, err := os.ReadFile(sendFlags.bodyFile)
    if err != nil {
        u.PrintFatal("cannot read --body-file", err)
    }
    body = strings.TrimSpace(string(data))
} else {
    edited, err := u.PromptEditor("body.md", "")
    if errors.Is(err, u.ErrNoTerminal) {
        u.PrintFatal("send needs --body-file when there is no interactive terminal", nil)
    }
    if err != nil {
        u.PrintFatal("editor failed", err)
    }
    if edited == "" {
        return
    }
    body = edited
}
```

The editor resolves as `$VISUAL`, then `$EDITOR`, then `vi`, and runs through the shell, which is the order and the interpretation git documents for its own editor (`git help var`). A value such as `code --wait` carries arguments that only a shell splits.

`name` becomes the temporary file's suffix, so `body.md` opens in the editor's Markdown mode. `initial` is the current value when a command edits something that already exists.

A buffer saved empty is a cancel, as it is for `git commit`. The user who opened the editor by mistake quits without writing, and the command stops rather than storing nothing.

```go
func PromptEditor(name, initial string) (string, error) {
    if !IsInteractive() {
        return "", ErrNoTerminal
    }
    f, err := os.CreateTemp("", "*-"+name)
    if err != nil {
        return "", err
    }
    defer os.Remove(f.Name())
    if _, err := f.WriteString(initial); err != nil {
        f.Close()
        return "", err
    }
    if err := f.Close(); err != nil {
        return "", err
    }
    editor := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi")
    cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", f.Name())
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    if err := cmd.Run(); err != nil {
        return "", err
    }
    data, err := os.ReadFile(f.Name())
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}
```

## Secrets

A secret is masked while it is typed and settles as a masked summary once it is submitted. A pasted credential is the common case, and a fully hidden field gives the user no way to tell a truncated paste from a complete one.

The tail appears only above eight characters, so a short secret is never mostly revealed by its own confirmation.

```go
func MaskSecret(s string) string {
    if len(s) <= 8 {
        return strings.Repeat("•", 8)
    }
    return strings.Repeat("•", 8) + s[len(s)-4:]
}
```

The value itself goes through no printer at all. The structured printers branch to the debug tier, which writes the string into a log that outlives the session.

A flag carrying a secret says so in its help text, naming both that the value is prompted when the flag is absent and that passing it inline is visible in shell history and in `ps` output for the life of the process.

```go
cmd.Flags().StringVar(&flags.password, "password", "",
    "Account password; prompted when omitted, and visible in shell history and ps output when passed inline")
```
