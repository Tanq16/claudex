# Review Domain: Go CLI

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web. Categories 3 through 5 apply to CLI Only and the command surface of a hybrid, and are skipped for Web Only.

**Skills to load, in full, before running any check below:**
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
| `Execute` behavior | Read `cmd/root.go` |
| Help and completion visibility | Read `cmd/root.go` |
| Logging setup | Read `cmd/root.go` for `setupLogs` and `cobra.OnInitialize`, then compare against the project type |
| Global flags | Read `cmd/root.go` for `--debug` and `--for-ai` and their exclusivity, then compare against the type |
| Imports | Read the import block for zerolog and `utils`, then compare against the type |
| Command registration | Read `init()` |

## Category 2: Commands and Flags

| Check | How to verify |
|---|---|
| Flag structs | Grep `cmd/` for flag variable declarations and check they are grouped per command |
| Flag registration site | Read each `init()` in `cmd/` |
| Required and exclusive flags | Grep for `MarkFlagRequired` and `MarkFlagsMutuallyExclusive`; read each `Run` for hand-rolled validation that should be one of them |
| `Run` shape | Read each `Run` body and trace where the work happens |
| Subcommand package shape | Read each `cmd/*/` package: whether the parent is exported, whether it has a `Run`, whether children do |
| Output calls | Grep `cmd/` for `fmt.Println`, `fmt.Printf`, `utils.Print`, `u.Print`, and `log.Printf`, then compare against the type |

## Category 3: Output Tiers

| Check | How to verify |
|---|---|
| `globals.go` | Read `utils/globals.go` |
| Printer branch order and completeness | Read `utils/printer.go`; check every structured printer for the same three-way branch |
| `PrintGeneric` stays unbranched | Read `utils/printer.go` |
| Error detail confined to debug | Read the human and AI branches of `PrintError`, `PrintFatal`, and `PrintWarn` and check whether `err` reaches them |
| Error passed as `err` | Grep call sites of `PrintFatal`, `PrintError`, and `PrintIndentedError` for an error formatted into the message string |
| Subprocess stderr | Grep for `exec.Command`; for each, check whether stderr is captured before the error is reported |
| Table branch | Read `utils/table.go` |
| Cell escaping | Read the markdown branch of `utils/table.go` |
| Terminal colors | Grep for lipgloss color construction and check whether the values are ANSI indices or hex |

## Category 4: Prompts

Skip when the project takes no interactive input.

| Check | How to verify |
|---|---|
| Prompt helpers exist | Glob for `utils/input.go` and list its exported functions |
| Every prompt branches on AI mode | Read each prompt function for a `GlobalForAIFlag` branch |
| Piped reads use the right helper | Check which of the line and bulk readers each prompt uses |
| Shared scanner | Read the scanner construction and check it is not per-call |
| Cancel paths | Grep call sites of the selectors for handling of the cancel return |
| Password handling | Grep for the password prompt's return value reaching any `Print` call |

## Category 5: Progress

Skip when the project shows no sequential progress.

| Check | How to verify |
|---|---|
| Clear count | For each `ClearLines` following a running header, check the count against the lines printed |
| Clearing is inert outside human mode | Read `ClearLines` and `ClearPreviousLine` |
| Lifecycle shapes | Grep for `PrintRunning` and trace each to its clear and its final line |
| Progress goroutine guard | Grep for progress goroutines and check for the atomic guard on the final clear |
| Progress in AI and debug modes | Read `PrintProgress` |

---

## Output Format

```
## Domain: Go CLI

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** (skill-name: section)
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
