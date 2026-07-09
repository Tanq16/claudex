# Review Domain: Go CLI

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web **Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../go-cli/SKILL.md`

---

## Category 5: Cobra CLI Setup (go-cli)

**CLI Only checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Root command structure | `rootCmd` with `Use`, `Short`, `Version`, `CompletionOptions.HiddenDefaultCmd: true` | Read `cmd/root.go` |
| AppVersion variable | `var AppVersion = "dev-build"` set via ldflags at build time | Grep for `AppVersion` in `cmd/root.go` |
| Execute function | `func Execute()` wrapping `rootCmd.Execute()` with stderr error output and `os.Exit(1)` | Read `cmd/root.go` |
| setupLogs function | Called via `cobra.OnInitialize(setupLogs)` | Read `cmd/root.go` |
| Debug flag | `rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, ...)` | Read `cmd/root.go` |
| AI flag | `rootCmd.PersistentFlags().BoolVar(&forAIFlag, "for-ai", false, ...)` with `utils.GlobalForAIFlag = true` in `setupLogs()` | Read `cmd/root.go` |
| Hidden help command | `rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})` | Read `cmd/root.go` |
| Command registration | Commands added via `rootCmd.AddCommand()` in `init()` | Read `cmd/root.go` |

**Web Only checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Root command structure | `rootCmd` with `Use`, `Short`, `Version`, `CompletionOptions.HiddenDefaultCmd: true` | Read `cmd/root.go` |
| AppVersion variable | `var AppVersion = "dev-build"` set via ldflags at build time | Grep for `AppVersion` in `cmd/root.go` |
| Execute function | `func Execute()` wrapping `rootCmd.Execute()` with stderr error output and `os.Exit(1)` | Read `cmd/root.go` |
| NO setupLogs function | No `setupLogs` function, no `cobra.OnInitialize(setupLogs)` | Read `cmd/root.go`, flag if present |
| NO debug flag | No `--debug` persistent flag | Read `cmd/root.go`, flag if present |
| NO for-ai flag | No `--for-ai` persistent flag | Read `cmd/root.go`, flag if present |
| NO zerolog import | No `rs/zerolog` import | Read `cmd/root.go`, flag if present |
| NO utils import | No utils package import | Read `cmd/root.go`, flag if present |
| Hidden help command | `rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})` | Read `cmd/root.go` |
| Command registration | Commands added via `rootCmd.AddCommand()` in `init()` | Read `cmd/root.go` |

---

## Category 6: Command Patterns (go-cli)

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Flag structs | Flags grouped in structs per command (e.g., `var serveFlags struct { ... }`) | Grep for flag struct patterns in `cmd/` files |
| Run function pattern | Validate flags, build config struct, call internal package, output result | Read command `Run` functions |
| Flag registration | Flags registered in `init()` function using `cmd.Flags().TypeVarP()` | Read `init()` functions in command files |

**CLI Only additional check:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Utils for output | Commands use `u.PrintInfo`, `u.PrintSuccess`, `u.PrintFatal` etc., not raw `fmt.Println` | Grep for `fmt.Println` or `fmt.Printf` in `cmd/` files (should use utils instead) |

**Web Only additional check:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| log.Printf/log.Fatalf for output | Commands use `log.Printf` with prefixes and `log.Fatalf` for fatal errors. No utils functions. | Grep for `utils.Print` or `u.Print` in `cmd/` files (flag if present) |

---

## Category 6b: Output Lifecycle Patterns (go-cli)

**Applies to: CLI Only (and the CLI surface of CLI + Web hybrids).** SKIP for Web Only projects. Only check if the project uses output lifecycle patterns (phases, running indicators, progress bars).

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Phase lifecycle | Phases use `PrintRunning` header → indented sub-results → `ClearLines(lineCount + 1)` → summary. Errors-only phase uses `PrintError` header with `PrintIndentedError` for each failure; clean phase uses `PrintInfo` | Grep for `PrintRunning` and `ClearLines` patterns in command files |
| ClearLines count | `ClearLines(lineCount + 1)` — always `+1` for the running header. `lineCount` tracks only sub-lines printed during the loop | Verify `+1` in all `ClearLines` calls that follow a `PrintRunning` header |
| Single-operation lifecycle | Single ops use `PrintRunning` → work → `ClearLines(1)` → result (`PrintSuccess` or `PrintFatal`) | Check that `ClearLines(1)` follows single running lines |
| Progress indicator guard | Progress goroutines use `atomic.Bool` to track whether the goroutine printed. Final `ClearPreviousLine()` after `close(done)` is guarded by `if printed.Load()` | Grep for progress goroutine patterns, verify `atomic.Bool` guard exists |
| Error discipline | `PrintError`/`PrintFatal`/`PrintIndentedError` pass the actual `err` as the error parameter, never baked into `msg` via `fmt.Sprintf` | Grep for `PrintFatal` and `PrintIndentedError` calls, verify `err` is passed as second arg not embedded in first |
| Subprocess stderr capture | Direct `exec.Command` calls (not via `utils.RunCmd`) capture stderr into the error before passing to `PrintFatal` | Grep for `exec.Command` calls, verify stderr is captured when not using `RunCmd` |

---

## Category 7: TUI Output (go-cli)

**Applies to: CLI Only (and the CLI surface of CLI + Web hybrids).** SKIP for Web Only projects.

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Terminal-adaptive colors | Lipgloss color constants use ANSI standard indices (0-15) that adapt to the user's terminal theme, not hardcoded hex values | Grep for lipgloss color definitions, verify they use `lipgloss.ANSIColor(N)` (e.g., `lipgloss.ANSIColor(12)`) not hex (e.g., `lipgloss.Color("#89b4fa")`) |
| Table output | If tables used, lipgloss table formatting via utils | Grep for table rendering |
| Clean output without debug | Without `--debug`, user sees clean colored output (no log lines) | Check that print functions route through utils, not zerolog directly |
| AI mode output | With `--for-ai`, print functions emit prefixed plain text (`[OK]`, `[ERROR]`, `[WARN]`, `[INFO]`) and tables render as markdown | Grep for `GlobalForAIFlag` usage in utils print/table functions |

---

## Output Format

Report findings in this exact format:

```
## Domain: Go CLI

### [PASS] Category Name (go-cli)

All checks passed.

### [ISSUES] Category Name (go-cli)

1. **[Issue title]** (go-cli: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (go-cli)

Not applicable to this project type.
```

End your response with exactly:
```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

---

## Out of Scope (Hard Boundary)

Do NOT flag any of the following — they are not defined in any loaded skill:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | No golangci-lint, no gofmt, inconsistent formatting |
| Pre-commit | No pre-commit hooks, no husky |
| Code Quality CI | No lint/format CI steps |
| Documentation beyond README | No godoc, no changelogs, no contributing guide |
| Docker Compose | No docker-compose for development |
| Database | No migrations, no schema files |
| Dependency tooling | No dependabot, no renovate |
| Security scanning | No SAST, no container scanning |
| Code style opinions | Naming conventions not in skills, personal preferences |

**Rule:** If you cannot cite a specific section in a loaded skill for a finding, do not report it.
