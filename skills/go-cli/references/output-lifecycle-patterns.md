# Output Lifecycle Patterns

**Sequential progress reporting for CLI tools — the simple alternative to the Highway pattern.**

**Applies to:** CLI Only projects with the `utils/` package.

Use when a command needs to show running/done state for sequential operations. For concurrent multi-job pipelines with shared progress reporting, use the Highway pattern (`go-concurrency`) instead.

---

## Phase Lifecycle

A phase is a named group of sequential operations. Show a running header, print indented sub-results as they complete, then clear everything and replace with a single summary line.

### Terminal Output (Human Mode)

While running:
```
↻ (Running) Phase 2: System packages
  ✓ tmux: installed system-managed
  ✓ openssl: already at system-managed
  ✗ nmap: apt install failed
```

After completion (all lines cleared, replaced with):
```
→ Phase 2: System packages                      ← no errors
```
Or:
```
✗ Phase 2: partially completed with errors       ← errors present
  ✗ nmap: apt install failed                      ← only errors persist
```

### Pattern

```go
func runPhase(phaseName string, tools []Tool, ...) bool {
    if len(tools) == 0 {
        return false
    }
    utils.PrintRunning("(Running) " + phaseName)

    var lineCount int
    var errors []jobResult

    for _, t := range tools {
        // ... do work ...
        if err != nil {
            utils.PrintIndentedError(t.Name, err)
            errors = append(errors, jobResult{name: t.Name, err: err})
        } else {
            utils.PrintIndentedSuccess(fmt.Sprintf("%s: installed %s", t.Name, version))
        }
        lineCount++
    }

    utils.ClearLines(lineCount + 1) // tool lines + running header
    if len(errors) > 0 {
        utils.PrintError(phaseName+": partially completed with errors", nil)
        for _, e := range errors {
            utils.PrintIndentedError(e.name, e.err)
        }
    } else {
        utils.PrintInfo(phaseName)
    }

    return len(errors) > 0
}
```

### ClearLines Count Rule

`ClearLines(lineCount + 1)` — always `+1` for the running header. `lineCount` tracks sub-lines printed during the loop. In debug and AI modes, `ClearLines` is a no-op, so all output persists.

---

## Single-Operation Lifecycle

For operations with no sub-tasks (single install, self-update steps). Show a running line, do the work, clear it, show the result.

### Terminal Output (Human Mode)

```
↻ installing bat                                 ← transient
```
Cleared and replaced:
```
✓ bat: installed v0.24.0                         ← persists
```

### Pattern

```go
utils.PrintRunning("installing " + toolName)
result := inst.Install(tool, p, gh, st)
utils.ClearLines(1)

if result.Err != nil {
    utils.PrintFatal(fmt.Sprintf("%s: install failed", toolName), result.Err)
}
utils.PrintSuccess(fmt.Sprintf("%s: installed %s", toolName, result.Version))
```

### Multi-Step Variant

For operations with multiple sequential steps (self-update: check, auth, download):

```go
utils.PrintRunning("checking latest version")
release, err := checkVersion()
utils.ClearLines(1)

utils.PrintRunning("authenticating sudo")
err = ensureSudo()
utils.ClearLines(1)

utils.PrintRunning(fmt.Sprintf("downloading %s", release.Tag))
err = download(release)
utils.ClearLines(1)

utils.PrintSuccess(fmt.Sprintf("updated: %s → %s", old, new))
```

Each step independently prints running, clears, moves to next. The final result persists.

---

## Check Lifecycle

For read-only operations that scan multiple items and report results. No per-item output during scanning — just a single running indicator, then a summary.

### Terminal Output (Human Mode)

```
↻ Checking tools                                 ← transient
```
Cleared and replaced:
```
→ Check complete
  ! bat: update available (v0.23.0 → v0.24.0)
  ✗ nuclei: check failed
  ! tmux-config: config differs
```
Or if clean:
```
✓ everything is up to date
```

### Pattern

```go
utils.PrintRunning("Checking tools")
results := checkAll(tools)
utils.ClearLines(1)

if len(results) == 0 {
    utils.PrintSuccess("everything is up to date")
    return
}

utils.PrintInfo("Check complete")
for _, r := range results {
    switch r.Status {
    case "update":
        utils.PrintIndentedWarn(fmt.Sprintf("%s: update available (%s → %s)", r.Name, r.Current, r.Latest), nil)
    case "error":
        utils.PrintIndentedError(fmt.Sprintf("%s: check failed", r.Name), r.Err)
    }
}
```

---

## Progress Indicator

For long-running single operations where percentage completion is known (encoding, downloads, file processing). A goroutine periodically overwrites a single line with a braille-dot progress bar.

### Terminal Output (Human Mode)

```
  ↻ video-3.mp4: ⣿⣿⣿⣿⣿⣀⣀⣀⣀⣀ 50%
```

The line updates in-place. On completion, it's cleared and replaced with a success line.

### Caller Pattern

The caller runs a goroutine that ticks on an interval and overwrites the progress line. The `printed` flag tracks whether the goroutine has written any output (guards against clearing the wrong line if work completes before the first tick).

```go
// Track progress in a goroutine
done := make(chan struct{})
var printed atomic.Bool
go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    firstTick := true
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            if !firstTick {
                utils.ClearPreviousLine()
            }
            firstTick = false
            printed.Store(true)
            utils.PrintProgress("video-3.mp4", currentPercent)
        }
    }
}()

// Do the actual work
encode(input, output)

// Stop the goroutine, clean up the progress line
close(done)
if printed.Load() {
    utils.ClearPreviousLine()
}
utils.PrintIndentedSuccess("video-3.mp4: encoded")
```

### Progress Rules

- **One progress line at a time.** Never have two progress indicators active simultaneously.
- **Goroutine owns the line.** Only the progress goroutine writes and clears the progress line. The main goroutine closes the `done` channel and clears the final line after the goroutine exits.
- **Guard the final clear.** Use an `atomic.Bool` to track whether the goroutine ever printed. Only call `ClearPreviousLine()` after `close(done)` if the goroutine actually wrote output — otherwise you eat the wrong line.
- **Counts as 1 line in phase lifecycle.** If a progress indicator runs inside a phase, it contributes 1 to `lineCount`, not N (since it overwrites itself). After the goroutine cleans up, the final printed line (success/error) is the one that counts.
- **Tick interval:** 1 second is the default. Adjust based on the operation — faster for short tasks, slower for long ones. Don't tick faster than 250ms.
- **AI mode prints every tick.** No clearing in AI mode — each tick is a new `[PROGRESS]` line. This is intentional so AI agents see the full progression.
- **Debug mode logs with structured field.** `Int("percent", N)` enables filtering/querying in log analysis.

---

## Mode Behavior Summary

| Behavior | Human | AI | Debug |
|----------|-------|-----|-------|
| Styled ANSI icons/colors | yes | no | no |
| `ClearLines` / `ClearPreviousLine` | clears lines | no-op | no-op |
| Progress bar overwrites | yes | new line per tick | zerolog per tick |
| Error in `err` param | not shown | not shown | `.Err(err)` structured field |
| All output persists | no (transient cleared) | yes | yes |

The fundamental contract: **human mode is ephemeral** (transient lines are cleared), **AI and debug modes are permanent** (everything persists for parsing/logging).

---

## When to Use This vs Highway Pattern

| Aspect | Output Lifecycle (this) | Highway Pattern |
|--------|------------------------|-----------------|
| Execution model | Sequential operations | Concurrent job pipeline |
| Concurrency | Single goroutine (+ optional progress goroutine) | N workers with shared progress |
| State persistence | None | Resume capability, state file |
| Progress | Per-phase line clearing, optional progress bar | Aggregated multi-job display |
| Complexity | Low — direct print/clear calls | High — job interface, dispatcher, display engine |
| Use cases | Install sequences, self-update, checks, single file ops | Batch downloads, scans, migrations |
