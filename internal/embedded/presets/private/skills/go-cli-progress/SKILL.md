---
name: go-cli-progress
description: Sequential running/done progress for Go CLI tools - phase, single-operation, multi-step, and check lifecycles plus the in-place progress bar. Use when a command runs a series of steps and should show what it is doing, when clearing terminal lines, or when adding a percentage bar. Triggers on PrintRunning, PrintIndentedSuccess, PrintIndentedError, ClearLines, ClearPreviousLine, PrintProgress, and any command that prints "Running..." then replaces it with a result.
user-invocable: false
---

# Go CLI Progress

**Four lifecycles for showing sequential work in a CLI Only tool, and the in-place progress bar.**

These cover work that runs one step after another. A concurrent pipeline over many items uses a job pipeline with an aggregated display instead, because line clearing assumes one writer.

The contract underneath all four: human mode is ephemeral and transient lines are cleared, while AI and debug modes are permanent so a parser or a log keeps the full progression. `ClearLines` and `ClearPreviousLine` are no-ops in those two modes for that reason.

| Behavior | Human | AI | Debug |
|---|---|---|---|
| Styled icons and colors | yes | no | no |
| `ClearLines` / `ClearPreviousLine` | clears | no-op | no-op |
| Progress bar | overwrites one line | one new line per tick | one zerolog entry per tick |
| Everything printed persists | no | yes | yes |

## Line Clearing

```go
// ClearLines removes n lines of terminal output.
// It is a no-op in debug and AI modes so all output persists for logging and parsing.
func ClearLines(n int) {
    if GlobalDebugFlag || GlobalForAIFlag {
        return
    }
    for range n {
        fmt.Print("\033[A\033[2K")
    }
}

func ClearPreviousLine() {
    if GlobalDebugFlag || GlobalForAIFlag {
        return
    }
    fmt.Print("\033[A\033[2K")
}
```

The count is always `lineCount + 1`, where the `+1` is the running header itself. Counting only the sub-lines leaves the header stranded above the summary that was meant to replace it.

## Phase Lifecycle

A phase is a named group of sequential sub-tasks. It prints a running header, prints indented results as they land, then clears everything and replaces it with one summary line. Collapsing a finished phase keeps the final screen proportional to what went wrong rather than to how much work was done.

While running:

```
↻ (Running) Phase 2: System packages
  ✓ tmux: installed system-managed
  ✓ openssl: already at system-managed
  ✗ nmap: apt install failed
```

After completion, with errors:

```
✗ Phase 2: partially completed with errors
  ✗ nmap: apt install failed
```

After completion, clean:

```
→ Phase 2: System packages
```

```go
func runPhase(phaseName string, tools []Tool) bool {
    if len(tools) == 0 {
        return false
    }
    utils.PrintRunning("(Running) " + phaseName)

    var lineCount int
    var errs []jobResult

    for _, t := range tools {
        version, err := install(t)
        if err != nil {
            utils.PrintIndentedError(t.Name, err)
            errs = append(errs, jobResult{name: t.Name, err: err})
        } else {
            utils.PrintIndentedSuccess(fmt.Sprintf("%s: installed %s", t.Name, version))
        }
        lineCount++
    }

    utils.ClearLines(lineCount + 1)

    if len(errs) > 0 {
        utils.PrintError(phaseName+": partially completed with errors", nil)
        for _, e := range errs {
            utils.PrintIndentedError(e.name, e.err)
        }
    } else {
        utils.PrintInfo(phaseName)
    }
    return len(errs) > 0
}
```

Only the failures are reprinted after the clear, so what stays on screen is what the user still has to act on.

## Single-Operation Lifecycle

One task with no sub-steps: print running, do the work, clear the one line, print the result.

```go
utils.PrintRunning("installing " + toolName)
result := inst.Install(tool)
utils.ClearLines(1)

if result.Err != nil {
    utils.PrintFatal(fmt.Sprintf("%s: install failed", toolName), result.Err)
}
utils.PrintSuccess(fmt.Sprintf("%s: installed %s", toolName, result.Version))
```

## Multi-Step Lifecycle

Several sequential steps, each announcing and clearing itself, with only the final result persisting. A self-update that checks a version, authenticates, and downloads reads as one operation to the user, so it leaves one line behind.

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

## Check Lifecycle

A read-only scan over many items prints nothing per item, just one running indicator and then a summary. Per-item output during a check would scroll the findings off the screen before the user could read them.

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

## Progress Bar

For one long operation whose completion percentage is known. A braille-dot bar overwrites a single line.

```
  ↻ video-3.mp4: ⣿⣿⣿⣿⣿⣀⣀⣀⣀⣀ 50%
```

```go
func PrintProgress(label string, percent int) {
    percent = min(percent, 100)

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
    bar := strings.Repeat("⣿", filled) + strings.Repeat("⣀", barWidth-filled)
    fmt.Println(infoStyle.Render(fmt.Sprintf("  ↻ %s: %s %d%%", label, bar, percent)))
}
```

Debug mode logs the percentage as a structured field rather than a formatted string, so a log query can filter on it.

The caller runs a ticking goroutine that owns the line:

```go
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

encode(input, output)

close(done)
if printed.Load() {
    utils.ClearPreviousLine()
}
utils.PrintIndentedSuccess("video-3.mp4: encoded")
```

### Progress Rules

One progress line is active at a time, since two goroutines clearing lines will each erase the other's output.

The goroutine owns the line while it runs. The main goroutine closes `done` and clears the final line only after the goroutine has exited, because clearing while it is still ticking races with its next write.

The `atomic.Bool` guards that final clear. Work that finishes before the first tick means the goroutine never printed, and clearing unconditionally would eat whatever line was above it.

A progress indicator inside a phase counts as 1 toward `lineCount`, not one per tick, since it overwrites itself. What counts is the success or error line the caller prints after cleanup.

One second is the default tick. Faster suits a short task and slower a long one, and anything under 250ms costs more in redraw than it conveys.

AI mode prints every tick as a new line and never clears, which is deliberate: an agent reading the output sees the full progression rather than one final number.
