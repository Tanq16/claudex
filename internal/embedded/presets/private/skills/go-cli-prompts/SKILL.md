---
name: go-cli-prompts
description: Interactive input for Go CLI tools - the symmetry rule that every prompted value also has a flag, the huh fields behind the utils prompt helpers, a masked secret that reveals its tail, grouped forms, and the ErrNoTerminal contract that keeps every command headless. Use when a command needs to ask the user something, when building or changing utils/input.go, or when adding a text, secret, confirm, selection, or multi-field prompt. Triggers on PromptInput, PromptPassword, PromptTextArea, PromptSelect, PromptMultiSelect, PromptConfirm, huh.NewInput, huh.NewSelect, huh.NewConfirm, huh.NewForm, ErrNoTerminal, StdinIsTerminal, and MarkFlagRequired on a prompted flag.
user-invocable: false
---

# Go CLI Prompts

**Every value a prompt collects also arrives through a flag, so the same command runs unattended, and a prompt that cannot open names the flag that would have replaced it.**

## The Symmetry Rule

A prompt exists only for a value that a flag also supplies. A tool driven from a script, a scheduler, or a chat interface has no terminal, so a value reachable only by a prompt is a value those callers cannot provide at all.

The rule is one-directional. A flag needs no prompt, and most flags have none; what never exists is a prompt without its flag.

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

Reading a value from a pipe is not part of this surface. A command that genuinely needs piped input reads it with a `bufio.Reader` in that command, as a decision local to the one tool that needs it.

## No Terminal

Every helper checks `StdinIsTerminal` before opening a program and returns `ErrNoTerminal` when there is not one, so a piped or backgrounded invocation fails immediately rather than hanging on a read nobody will satisfy.

```go
var ErrNoTerminal = errors.New("no interactive terminal")

func PromptSelect(label string, options []string) (int, error) {
    if !StdinIsTerminal {
        return -1, ErrNoTerminal
    }
    // ...
}
```

The message reads `<command> needs <flag> when there is no interactive terminal`. Naming the flag ends the problem in one line, where a message saying only that input was expected sends the caller to `--help` to work out which flag it was.

A prompt with a documented default takes that default instead of failing, since a default is an answer the command already has.

A command that cannot work without a terminal at all, because it hands the session to another program, says so once at the top of `Run` rather than at each prompt.

## The Helpers

| Helper | Behavior | Returns |
|---|---|---|
| `PromptInput(prompt, placeholder)` | single-line text | `(string, error)` |
| `PromptPassword(prompt)` | masked text, echoes a masked summary after entry | `(string, error)` |
| `PromptTextArea(prompt, placeholder)` | multi-line text | `(string, error)` |
| `PromptSelect(label, options)` | single choice | `(int, error)`, `-1` on cancel |
| `PromptMultiSelect(label, options)` | multiple choices, space toggles | `(map[int]bool, error)`, `nil` on cancel |
| `PromptConfirm(prompt)` | yes or no | `(bool, error)` |

Every cancel path is a clean no-op abort. `idx < 0` from `PromptSelect` and a `nil` map from `PromptMultiSelect` mean the user pressed Escape, and treating that as an empty selection runs the operation they just declined.

Reusing these helpers rather than building a prompt per command is what keeps the key bindings identical across a tool: arrows or `j`/`k` to move, Enter to confirm, Escape to cancel, and space to toggle in the multi variant.

## Building the Helpers

Each helper wraps one `huh` field. A field constructed and run in one chain replaces a hand-written bubbletea model, and the model is where every divergence between two prompts in the same tool comes from.

```go
func PromptInput(prompt, placeholder string) (string, error) {
    if !StdinIsTerminal {
        return "", ErrNoTerminal
    }
    var value string
    ok, err := runForm(2, huh.NewInput().Title(prompt).Placeholder(placeholder).Value(&value))
    if err != nil || !ok {
        return "", err
    }
    return strings.TrimSpace(value), nil
}
```

One `runForm` carries the theme, the key bindings, the width, and the height for every helper. `huh.Run(field)` wraps a bare field in a group and form with help suppressed (`huh@v2.0.3 run.go:4-8`) but exposes no hook for any of those, so the form is built directly.

```go
func runForm(height int, fields ...huh.Field) (bool, error) {
    form := huh.NewForm(huh.NewGroup(fields...)).
        WithTheme(theme()).
        WithKeyMap(keymap()).
        WithShowHelp(false).
        WithWidth(Width()).
        WithHeight(height)
    if err := form.Run(); err != nil {
        if errors.Is(err, huh.ErrUserAborted) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}
```

The width comes from `term.GetSize` on stdout, then `COLUMNS`, then a default of 80, with 24 as the floor. huh otherwise pins every field to `defaultWidth = 80` (`huh@v2.0.3 form.go:20,130`), so a prompt on a 200-column terminal renders into the left 80 and wraps a value that had room to sit on one line.

The height is fixed rather than left to grow. huh adds a blank line and an error line when a `Validate` fails, and bubbletea's inline renderer moves the cursor up by the previous frame's line count, so the grown frame is only partly erased and each attempt stacks another copy on screen. A constant height makes every frame the same size and the redraw lands in place. A field with no `Validate` reserves no error line.

| Helper | Height |
|---|---|
| `PromptInput`, `PromptPassword` | `2` |
| the same with a `Validate` | `4` |
| `PromptConfirm` | `3` |
| `PromptSelect`, `PromptMultiSelect` | the list height plus `2` |

`Escape` cancels only once it is bound. huh's default keymap binds `Quit` to `ctrl+c` alone (`huh@v2.0.3 keymap.go:109`), and leaves `esc` on the disabled Back and Close actions, so an unbound Escape does nothing and the user presses Enter to escape a prompt they wanted to abandon.

```go
func keymap() *huh.KeyMap {
    k := huh.NewDefaultKeyMap()
    k.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
    return k
}
```

`ThemeBase16` is the closest theme to a tool's own colors, because the fields it overrides use ANSI indices which the user's terminal theme remaps, where `ThemeCharm` and `ThemeCatppuccin` carry hex values that override that theme and fight it. It is not wholly base16: it builds on `ThemeBase` (`huh@v2.0.3 theme.go:241`) and inherits whatever it does not set.

The focused button is the one inherited style worth replacing. `ThemeBase16` leaves it light grey on magenta (`theme.go:259`), which is two bright colors with nothing between them, so the active choice in a confirm is the harder of the two to read.

```go
func theme() huh.Theme {
    return huh.ThemeFunc(func(isDark bool) *huh.Styles {
        s := huh.ThemeBase16(isDark)
        s.Focused.FocusedButton = s.Focused.FocusedButton.
            Foreground(lipgloss.Color("0")).Background(lipgloss.Color("5"))
        s.Blurred.FocusedButton = s.Focused.FocusedButton
        return s
    })
}
```

The field type follows the value being collected.

| Helper | Field |
|---|---|
| `PromptInput` | `huh.NewInput()` |
| `PromptPassword` | `huh.NewInput().EchoMode(huh.EchoModePassword)` |
| `PromptTextArea` | `huh.NewText()` |
| `PromptSelect` | `huh.NewSelect[int]()` |
| `PromptMultiSelect` | `huh.NewMultiSelect[int]()` |
| `PromptConfirm` | `huh.NewConfirm()` |

A list longer than its visible window gets a filter field above it, and the matcher is the tool's own. huh's built-in `Filtering` matches one plain substring through an unexported method with no hook (`huh@v2.0.3 field_select.go:736-739`), so `fix front` finds nothing in a list full of `fix/frontend-border-revert`.

```go
func Matches(option, query string) bool {
    if strings.TrimSpace(query) == "" {
        return true
    }
    o := strings.ToLower(option)
    for _, word := range strings.Fields(strings.ToLower(query)) {
        if !strings.Contains(o, word) {
            return false
        }
    }
    return true
}
```

Every whitespace-separated word matches as a substring in any order, which is how a user recalls an item they have seen rather than one they can spell from the left. The filter is an ordinary `huh.NewInput` bound to the query, and the list re-narrows through `OptionsFunc` with that query as its binding.

```go
filter := huh.NewInput().Title("Filter").Value(&query)
field.OptionsFunc(func() []huh.Option[int] { return filtered(options, query) }, &query)
```

A rule about the value belongs on the field rather than in the caller, because a prompt that rejects a bad answer on the spot keeps the user in the one place they can fix it.

```go
huh.NewInput().
    Title("Port:").
    Validate(func(s string) error {
        n, err := strconv.Atoi(s)
        if err != nil || n < 1 || n > 65535 {
            return errors.New("must be a port between 1 and 65535")
        }
        return nil
    }).
    Value(&port)
```

A multi-line value that a user will compose rather than paste takes `huh.NewText().ExternalEditor(true)`, which opens `$EDITOR` and reads the buffer back. A terminal textarea is the right surface for three lines and the wrong one for thirty.

## Secrets

A secret is masked while it is typed and echoed as a masked summary once it is submitted. A pasted credential is the common case and a silent, fully hidden field gives the user no way to tell a truncated paste from a complete one.

```go
func MaskSecret(s string) string {
    if len(s) <= 8 {
        return strings.Repeat("•", 8)
    }
    return strings.Repeat("•", 8) + s[len(s)-4:]
}
```

The tail appears only above eight characters, so a short secret is never mostly revealed by its own confirmation.

The masked form goes through `PrintGeneric`, and the value itself goes through no printer at all. Every other printer branches to the debug tier, which writes the string into a log that outlives the session.

A flag carrying a secret says so in its help text, naming both that the value is prompted when the flag is absent and that passing it inline is visible in shell history and in `ps` output for the life of the process.

```go
cmd.Flags().StringVar(&flags.password, "password", "",
    "Account password; prompted when omitted, and visible in shell history when passed inline")
```

## Forms

A form collects several values belonging to one object in a single pass. A sequence of separate prompts is right for values a command gathers independently, and wrong for a set the user is filling in together, because each prompt closes before the next opens and nothing they typed stays on screen.

```go
var (
    name    string
    exec    string
    restart string
    enable  bool
)

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("Unit name:").Value(&name).Validate(nonEmpty),
        huh.NewInput().Title("ExecStart:").Value(&exec).Validate(nonEmpty),
        huh.NewSelect[string]().Title("Restart:").
            Options(huh.NewOptions("no", "on-failure", "always")...).
            Value(&restart),
        huh.NewConfirm().Title("Enable at boot?").Value(&enable),
    ),
).WithTheme(huh.ThemeFunc(huh.ThemeBase16))

if err := form.Run(); err != nil {
    return err
}
```

Every field in a form still answers to a flag of its own. A form is a convenience for someone at a keyboard, so a command offering one accepts the whole set as flags and skips the form when they are present.

A form bound to values with `Value(&v)` needs no key lookups afterward. `form.GetString(key)` exists for a field declared with `Key`, and reaching for it when a pointer was already bound adds a second place the same value lives.
