---
name: go-cli-progress
description: The progress surface for Go CLI tools - the two-line live block, the in-place meter carrying rate and ETA, how it degrades with terminal width, and the settled and summary lines it collapses into. Use when a command runs work worth watching, when clearing terminal lines, when showing a download or a file count advance, or when a command prints "Running..." then replaces it with a result. Triggers on ClearLines, ClearPreviousLine, NewMeter, Meter.Add, Meter.Done, Meter.Fail, StdoutIsTerminal, GlobalDebugFlag, term.GetSize, and an in-place progress bar.
user-invocable: false
---

# Go CLI Progress

**One operation at a time in a two-line block that redraws in place, carrying how much is left and how fast it is going, collapsing into a single settled line when it finishes.**

This applies to CLI Only projects and the command surface of a CLI + Web hybrid, since it assumes the `utils` package exists.

One meter is live at a time. Two writers clearing lines each erase the other's output, so concurrent work reports through a single meter rather than a lane per worker. How that work is scheduled is the project's own decision, and nothing here imposes a pipeline, a job interface, or a resume file on it.

The contract underneath everything below: redrawing is a terminal affordance, so the live block exists only when stdout is a terminal and `--debug` is off. Everywhere else every step is still announced and nothing is cleared, which leaves a log or a pipe holding the full progression instead of a stream of cursor escapes.

| Behavior | Terminal | Piped | `--debug` |
|---|---|---|---|
| Styled icons and colors | yes | glyphs kept, color stripped | no |
| The live block | redraws in place | one line per tick | one zerolog entry per tick |
| `ClearLines` / `ClearPreviousLine` | clears | no-op | no-op |
| Everything printed persists | no | yes | yes |

Glyphs survive a pipe because `✓`, `↻`, and the bar rune are text rather than escape sequences, and only color and cursor control are stripped on the way out. What disappears outside a terminal is the redraw, never the fact that a step ran, because a step that failed is the one a log is read for.

## Line Clearing

```go
func ClearLines(n int) {
    if GlobalDebugFlag || !StdoutIsTerminal {
        return
    }
    for range n {
        fmt.Print("\033[A\033[2K")
    }
}

func ClearPreviousLine() {
    ClearLines(1)
}
```

The escape sequences go out through `fmt.Print` rather than a printer, since they are cursor control rather than content and the guard above has already established there is a cursor to control.

The count is always `lineCount + 1`, where the `+1` is the running header itself. Counting only the sub-lines leaves the header stranded above the summary that was meant to replace it.

## The Live Block

A meter owns two lines: a header naming the work, and an indented meter line carrying the numbers.

```
↻ Downloading ubuntu-24.04.3-desktop.iso  file 1 of 2
  ─────────────────────────────   36%  242 / 661 MB  28.3 MB/s  eta 15s  avg 40.7 MB/s
```

The header glyph costs two columns, so its text starts at column 2 and the meter line's bar starts there too. Every baseline line in a CLI puts content at column 2 and every indented detail line at column 4, which is what lets a meter, a settled line, and a failure sit under one another without looking ragged.

Context after the name is secondary and takes the muted color: the position in a set, a running total across the set, or the item currently being worked. It drops entirely before the name is ever clipped, because the name is what the user is waiting on.

```go
m := utils.NewMeter(name, resp.ContentLength, utils.UnitBytes)
m.Context("file 1 of 2")
if _, err := io.Copy(dst, io.TeeReader(resp.Body, m)); err != nil {
    m.Fail(err)
    return err
}
m.Done()
```

A meter is an `io.Writer`, so a byte stream feeds it through `io.TeeReader` and the caller counts nothing. Work measured in items calls `Add` once per item instead.

The meter owns its own ticker and its own line count. A caller that tracks either one has to get the clear count right on every path out of the function, and the path it misses is the error path.

A frame is assembled into one string and written with one `fmt.Print`. Two writes let the terminal paint a half-cleared block, which reads as a flicker on every tick.

```go
func (m *Meter) draw(lines []string) {
    if !m.live() {
        return
    }
    var b strings.Builder
    b.WriteString(strings.Repeat("\033[1A\033[2K", m.drawn))
    for _, l := range lines {
        b.WriteString("\r\033[2K" + l + "\n")
    }
    fmt.Print(b.String())
    m.drawn = len(lines)
}
```

The cursor is hidden while a meter is live and restored when it settles, including on the error path, because a process that exits with the cursor hidden leaves the user's shell without one.

## The Meter Line

Six fields in a fixed order, so a reader's eye lands in the same place moving from a download to a file count.

| Field | Example | Reserved | Drops |
|---|---|---|---|
| bar | `─────────` | 8 to 30 cells | last |
| percent | ` 36%` | 4 | never |
| transferred | `242 / 661 MB` | 14 | fourth |
| current rate | `28.3 MB/s` | 11 | third |
| eta | `eta 15s` | 11 | second |
| average rate | `avg 40.7 MB/s` | 15 | first |

Each field reserves its widest form rather than its current one, so the bar does not shift by a cell when `eta 9s` becomes `eta 15s`. A bar that jitters every second draws the eye to the jitter instead of the progress.

Fields drop from the right as the terminal narrows, then the bar shrinks toward its eight-cell floor, and only then does the bar itself go and the stats stand alone. Nothing wraps and no field is cut mid-value, because a wrapped frame makes the next redraw clear the wrong number of lines.

```
↻ Copying db.sqlite
  71%  45.8 / 64.0 MB
```

Width comes from the terminal and falls back rather than guessing.

```go
func termWidth() int {
    if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
        return w
    }
    if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n >= minWidth {
        return n
    }
    return defaultWidth
}
```

`github.com/charmbracelet/x/term` supplies both `GetSize` and the `IsTerminal` that `utils/globals.go` already calls, and it arrives under the lipgloss stack a CLI Only project has anyway. Taking `golang.org/x/term` for the same pair adds a second module for nothing.

The bar is a single `─` rune for both halves, filled in the info blue that every other live line uses and unfilled in dimmed chrome. One rune throughout means the bar's length never changes as it fills, and a two-glyph bar has to reserve the wider of them everywhere.

A total that is not known ahead of time gets a sweep across the track instead of a fill, since a percentage of an unknown quantity is a number the tool does not have.

## Rates and ETA

Two rates are shown. The instantaneous one comes from a trailing window and the average from the whole operation, because a single rate hides a stall behind a healthy-looking average and the user is watching precisely to see the stall.

The window is 800ms wide and reports zero until 200ms of samples have accumulated. A two-sample window microseconds wide divides a chunk by almost no time and reports hundreds of MB/s on the first tick.

```go
func (r *rateWindow) current() float64 {
    if len(r.samples) < 2 {
        return 0
    }
    first, last := r.samples[0], r.samples[len(r.samples)-1]
    if last.at.Sub(first.at) < 200*time.Millisecond {
        return 0
    }
    delta := float64(last.val - first.val)
    if delta <= 0 {
        return 0
    }
    return delta / last.at.Sub(first.at).Seconds()
}
```

ETA is computed from the windowed rate rather than the average, so a stall reads `eta unknown` instead of a slowly climbing lie the user then has to discount.

ETA is unknown whenever the total is unknown, the windowed rate is zero, or the result runs past about a hundred hours. Printing `eta 3170h` is worse than printing nothing, because the user reads it as a real estimate before working out that it is not.

A rate during a stall reads `0.00 B/s` rather than being blanked, since a blank field reads as a rendering bug and a zero reads as the truth.

## Settling

`Done` clears the live block and leaves one line in its place, so a finished run shows one line per operation regardless of how long each took.

```
✓ ubuntu-24.04.3-desktop.iso  661 MB  9.5s  avg 69.6 MB/s
```

`Fail` clears the block and prints an indented failure that persists, because a failure is what the user still has to act on after the run.

```
  ✗ receipts/hotel-0913.heic: unsupported image format
```

The failure carries the underlying error rather than a message with the reason already formatted into it. Normal output shows the label and the debug tier records `.Err(err)` with the whole wrapped chain, which is the entire reason the debug tier is worth having.

A long name is clipped with `…` on both the header and the settled line. Clipping on one and hard-cutting on the other makes the same name look different depending on which line it lands in.

## Grouped Work

Several operations under one heading print their settled lines as they land and close with a summary.

```
✓ assets.tar  180 MB  3.4s  avg 52.6 MB/s
✓ db.sqlite  64.0 MB  2.4s  avg 26.6 MB/s
↻ Copying media/clip-01.mp4  file 3 of 4  244 / 664 MB total
  ─────────────────────────────   31%  133 / 420 MB  73.0 MB/s  eta 4s  avg 91.5 MB/s
```

```
→ Copy  4 files  664 MB  11.9s  avg 56.0 MB/s
✗ Process  10 ok, 2 failed  12 items  11.0s  avg 1.1 items/s
```

The summary and the settled line are built by one function with the same field order, so the two read as the same shape with a count in front. Two hand-rolled formats drift apart on the first change to either.

A summary is printed when more than one operation ran or when any of them failed. A single clean operation is already fully described by its settled line, and repeating it as a summary says nothing twice.

A total that would sum unlike units is omitted rather than printed. Adding bytes to a file count produces a number that is wrong in a way nobody can see.

The glyph carries the outcome: `→` when everything succeeded, `✗` when anything failed, and the count reads `10 ok, 2 failed` instead of the plain total in that case.

## Work with No Quantity

Work with nothing to count uses the running line and the clear, with no meter at all. A bar that cannot move is worse than no bar, because it suggests a progress the tool is not actually tracking.

Several sequential steps that read as one operation to the user each announce and clear themselves, leaving one line behind.

```go
utils.PrintRunning("checking latest version")
release, err := checkVersion()
utils.ClearLines(1)

utils.PrintRunning(fmt.Sprintf("downloading %s", release.Tag))
err = download(release)
utils.ClearLines(1)

utils.PrintSuccess(fmt.Sprintf("updated: %s → %s", old, new))
```

A read-only scan over many items prints nothing per item, just one running line and then the findings. Per-item output during a check scrolls the findings off the screen before the user can read them.

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
    utils.PrintIndentedWarn(fmt.Sprintf("%s: update available (%s → %s)", r.Name, r.Current, r.Latest), nil)
}
```

## Units

```go
type Unit string

const UnitBytes Unit = ""
```

The empty unit means bytes and formats in binary multiples with a scaled pair such as `242 / 661 MB`. Any other value is the plural noun printed after a plain count, so `utils.Unit("items")` renders `7 / 12 items` and `7.0 items/s` with no change anywhere in the renderer.

Both halves of a pair are scaled by the total rather than each by itself, so `39.9 / 64.0 MB` stays readable where `40874 KB / 64.0 MB` does not.

## Debug

The debug tier emits one zerolog entry per tick with the numbers as structured fields, never a formatted string, so a log query can filter on them.

```go
log.Info().
    Int("percent", pct).
    Int64("current", cur).
    Int64("total", m.total).
    Float64("rate", rate).
    Str("eta", eta).
    Msg(m.label)
```

zerolog chooses its writer from the same terminal check the printers use: `ConsoleWriter` on a terminal and its own JSON otherwise. That is settled once in `setupLogs` and nothing in the meter re-decides it.
