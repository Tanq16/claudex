# Review Domain: Go Concurrency

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web (only if concurrency/pipeline patterns are detected) **Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../go-concurrency/SKILL.md`

---

## Pre-Check: Detect Usage

Before running these categories, first determine if the project uses concurrency or pipeline patterns:

1. **Concurrency:** Grep for `go func`, `sync.WaitGroup`, `errgroup`, goroutine launches. If none found, mark Category 10 as SKIP.
2. **Pipeline:** Glob for `internal/highway/` or similar job execution engine directory. If none found, mark Category 11 as SKIP.

---

## Category 10: Concurrency Patterns (go-concurrency)

Only check if the project uses goroutines or concurrent patterns.

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| errgroup preferred | Default choice for concurrent operations with error handling is `errgroup` | Grep for errgroup imports vs manual WaitGroup+error channel patterns |
| Context cancellation | Long-running operations check `ctx.Done()` in loops and before work | Grep for `ctx.Done()` checks near goroutine code |
| Loop variable semantics | Go 1.22+ scopes loop variables per-iteration; no `item := item` capture needed | Grep for goroutine launches inside loops, verify no unnecessary variable capture |
| wg.Go over manual Add/Done | Goroutines spawned with `sync.WaitGroup.Go` (Go 1.25+), not manual `wg.Add(1)` + `go func` + `defer wg.Done()` | Grep for `wg.Add`/`wg.Done`; flag manual pairing in favor of `wg.Go` |
| No goroutine leaks | Channels are properly closed by senders; goroutines have exit paths | Review channel usage patterns |
| Bounded concurrency | If many concurrent operations, uses `errgroup.SetLimit()` or buffered channel semaphore | Grep for concurrency limiting patterns |

---

## Category 11: Job Pipeline Pattern (go-concurrency)

Check only if `internal/highway/` or similar job execution engine exists.

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Job interface | `Job` interface with `ID()`, `Type()`, `Run(ctx, progress)`, `Marshal()` methods | Grep for Job interface definition |
| Progress struct | `Progress` struct with `JobID`, `Type` (Progress/SubStatus), `Message`, `Current`, `Total`, `Done`, `Error` | Read progress type definition |
| Highway struct | `Highway` with `workers`, `jobs` channel, `progress` channel, `state`, `unmarshalers` | Read highway implementation |
| State persistence | State file (`.toolname-resume-state.json`) with `completed` and `pending` arrays | Grep for state file save/load logic |
| Resume capability | `LoadState()` method that deserializes pending jobs via registered unmarshalers | Read resume/load code |
| Ctrl+C handling | `signal.NotifyContext` in command, state saved on context cancellation | Grep for signal handling |
| Display manager | Separate `internal/display/` package consuming progress channel | Glob for display package |
| Display AI mode | Display manager checks `utils.GlobalForAIFlag` and prints sequential `[INFO]`/`[OK]`/`[ERROR]` lines instead of interactive TUI | Read display.go, check for `GlobalForAIFlag` branch in `Start()` and `renderFinal()` |
| Job registration | Job types registered via `RegisterType()` before `Run()` | Read command setup code |

---

## Output Format

Report findings in this exact format:

```
## Domain: Go Concurrency

### [PASS] Category Name (source-skill)

All checks passed.

### [ISSUES] Category Name (source-skill)

1. **[Issue title]** (source-skill: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (source-skill)

Not applicable — [reason, e.g., "no goroutines detected" or "no pipeline engine found"].
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
