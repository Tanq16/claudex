# Review Domain: Go CLI

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web. Categories 3 through 5 apply to CLI Only and the command surface of a hybrid, and are skipped for Web Only.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-cli-structure/SKILL.md`
- `[SKILLS_DIR]/go-cli-commands/SKILL.md`
- `[SKILLS_DIR]/go-cli-output/SKILL.md`
- `[SKILLS_DIR]/go-cli-prompts/SKILL.md`
- `[SKILLS_DIR]/go-cli-progress/SKILL.md`

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

Several checks invert by project type: the same construct is required in CLI Only and is a defect in Web Only. Establish the type before running any of them.

---

## Category 1: Root Command

| Check | How to verify |
|---|---|
| Root command fields | Read `cmd/root.go` |
| `AppVersion` and its injection | Grep `cmd/root.go` for `AppVersion`; grep the Makefile for the matching `-X` path |
| `Execute` behavior | Read `cmd/root.go`; check it sets the exit code without reprinting an error Cobra has already written |
| Help and completion visibility | Read `cmd/root.go` |
| Logging setup | Read `cmd/root.go` for `setupLogs` and `cobra.OnInitialize`, then compare against the project type |
| Global flags | Read `cmd/root.go` for `--debug`, and for any flag added to obtain machine-readable output, then compare against the type |
| Imports | Read the import block for zerolog and `utils`, then compare against the type |
| Command registration | Read `init()` |

## Category 2: Commands and Flags

| Check | How to verify |
|---|---|
| Flag structs | Grep `cmd/` for flag variable declarations and check they are grouped per command |
| Flag registration site | Read each `init()` in `cmd/` |
| Registration level | Grep for `PersistentFlags`; check each is honored by the whole subtree it sits on |
| Flag names unique across the tree | Collect every registered flag name in `cmd/`; a name appearing on both a parent's `PersistentFlags` and a child's `Flags()` is shadowed silently |
| Boolean shorthands | Grep `cmd/` for `BoolVarP` and `BoolVar`; a switch takes a long name only |
| `SortFlags` untouched | Grep `cmd/` for `SortFlags`; the default of `true` is what `--help` should use |
| `Args` on every runnable command | Read each `cobra.Command` literal carrying a `Run` for an `Args` field; one taking no positionals declares `cobra.NoArgs` rather than omitting it. A command with no `Run` is not a finding, and `Args` on the root is: it disables the only unknown-command check a root with children has |
| Enum flags validate at parse time | Grep each `Run` for a switch or an `if` rejecting a flag value; that check belongs in a `pflag.Value` registered with `Var` |
| Flag relationships | Grep for `MarkFlagRequired`, `MarkFlagsRequiredTogether`, `MarkFlagsOneRequired`, and `MarkFlagsMutuallyExclusive`; read each `Run` for hand-rolled validation that should be one of them |
| A prompted flag is not required | For each flag with a prompt behind it, check the command does not call `MarkFlagRequired` on it |
| `Run` against `RunE` | Grep `cmd/` for `RunE`; each one has a `defer` in its body, and the root sets `SilenceErrors` and `SilenceUsage` with `Execute` reporting through `PrintFatal` |
| Every prompt has a flag | For each prompt call site, check the command registers a flag or accepts a positional supplying the same value |
| Stdin is marked, never inferred | Grep for `os.Stdin`, `io.ReadAll`, and `bufio.NewScanner` outside the resolver; every read is reached through a marked flag whose value is `-`, never through a terminal check |
| Read mode matches the value's shape | For each `MarkStdinLine` and `MarkStdinStream`, check `line` sits on a flag whose value is inherently single and `stream` only on a `<thing>-file` flag |
| Multi-line values are `-file` flags | For each flag whose value can span lines, check the bare form is absent and only `<thing>-file` is registered |
| The resolver is wired once | Grep for `ResolveStdin` and `PersistentPreRunE`; the root wires it, and any subcommand with its own hook calls it first |
| `Run` shape | Read each `Run` body and trace where the work happens |
| Subcommand package shape | Read each `cmd/*/` package: whether the parent is exported, whether it has a `Run`, whether children do |
| Output calls | Grep `cmd/` for `fmt.Println`, `fmt.Printf`, `utils.Print`, and `u.Print`, then compare against the type |

## Category 3: Output Tiers

| Check | How to verify |
|---|---|
| `globals.go` | Read `utils/globals.go` for `GlobalDebugFlag` and the terminal checks |
| Printer branch order and completeness | Read `utils/printer.go`; check every structured printer for the same two-way branch, debug first |
| Printers write through the color profile | Grep `utils/printer.go` for `fmt.Print`; content goes out through `lipgloss.Println` and only cursor escapes use `fmt.Print` |
| `PrintGeneric` carries no prefix or style | Read `utils/printer.go` |
| Error detail confined to debug | Read the non-debug branch of `PrintError`, `PrintFatal`, and `PrintWarn` and check whether `err` reaches it |
| Error passed as `err` | Grep call sites of `PrintFatal`, `PrintError`, and `PrintIndentedError` for an error formatted into the message string |
| Subprocess stderr | Grep for `exec.Command`; for each, check whether stderr is captured before the error is reported |
| Table rendering | Read `utils/table.go`; one renderer, not one per audience |
| Terminal colors | Grep for lipgloss color construction and check whether the values are ANSI indices or hex |
| zerolog writer selection | Read `setupLogs` for `ConsoleWriter` chosen on the terminal check, with the plain JSON writer otherwise |

## Category 4: Prompts

Skip when the project takes no interactive input.

| Check | How to verify |
|---|---|
| Prompt helpers exist | Glob for `utils/input.go` and list its exported functions |
| Every prompt guards on a terminal | Read each prompt function for a `StdinIsTerminal` guard returning `ErrNoTerminal` before any bubbletea program starts |
| `ErrNoTerminal` is handled | Grep prompt call sites for `errors.Is`; check the message names every path that would have supplied the value, including `-` where the flag is stdin-eligible |
| Resolution order | For each value reachable more than one way, read the call site for flag, then stdin, then prompt, then an error |
| The resolver's own guards | Read `ResolveStdin` for the two-flag error, the terminal check, and the empty-input error |
| Prompts are the exception | For each prompt, judge whether a flag could carry the value instead |
| Cancel paths | Grep call sites of the selectors for handling of the cancel return |
| Password handling | Grep for the password prompt's return value reaching any `Print` call |

## Category 5: Progress

Skip when the project shows no progress output at all.

| Check | How to verify |
|---|---|
| Clear count | For each `ClearLines` following a running header, check the count against the lines printed |
| Clearing is inert outside a terminal | Read `ClearLines` and `ClearPreviousLine` for the `GlobalDebugFlag` and `StdoutIsTerminal` guard |
| Lifecycle shapes | Grep for `PrintRunning` and trace each to its clear and its final line |
| One meter at a time | Grep for meter construction and check that no two are live in the same scope |
| The meter owns its line count | Read the meter for its own `drawn` count, and check no caller tracks lines on its behalf |
| One write per frame | Read the draw path for a single `fmt.Print` of an assembled string |
| Cursor restored on every path | Trace the hide to its show, including the failure path |
| Width is measured, not assumed | Read the width helper for `term.GetSize` ahead of `COLUMNS` and a final constant |
| Fields reserve their widest form | Read the field widths for reservation rather than current length |
| Degradation order | Read the fit loop for fields dropping right to left, then the bar shrinking, then the bar dropping |
| Two rates | Read the meter for a windowed rate distinct from the whole-operation average |
| Rate window floor | Read the windowed rate for the minimum sample span before it reports |
| ETA source and unknowns | Read the ETA for the windowed rate, and for unknown on a zero rate, an unknown total, and an absurd result |
| Settled and summary share a builder | Read both for one function and one field order |
| Failure carries the error | Grep the failure path for an error formatted into the message string instead of passed as `err` |
| Progress outside a terminal | Read the non-terminal branch; one line per tick, glyph kept, no bar |
| Debug tier fields | Read the debug branch for structured zerolog fields rather than a formatted string |

---

## Output Format

```
## Domain: Go CLI

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** [severity] (skill-name: section)
   - **Where:** file:line
   - **Current:** [what the code does now]
   - **Expected:** [what the cited skill section says]
   - **Fix:** [the specific action]

### [SKIP] Category Name

Not applicable: [reason].
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
