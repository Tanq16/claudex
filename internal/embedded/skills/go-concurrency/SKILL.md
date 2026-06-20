---
name: go-concurrency
description: Use when running concurrent work or orchestrating multi-job pipelines in Go - covers goroutine patterns (WaitGroup, errgroup, semaphores, fan-out/fan-in), error handling, cancellation, and the Highway pattern for progress tracking and resume
user-invocable: false
---

# Go Concurrency

**Concurrency primitives and the Highway job-pipeline pattern for Go execution.**

This skill has two layers:

- **Part 1 — Concurrency primitives:** vanilla goroutine patterns for work inside a job's
  `Run()` method or simple standalone operations.
- **Part 2 — Highway pattern:** job orchestration on top of those primitives, adding progress
  tracking, a UI-agnostic display, and Ctrl+C resume.

Reach for Part 1 when you just need to parallelize work. Reach for Part 2 when a CLI tool runs
many items through a unified pipeline that needs progress reporting and resumability.

## When to Use

Use this skill when:
- Running multiple operations concurrently with error handling or bounded concurrency
- Parallelizing work inside a job's `Run()` method or a standalone function
- Needing graceful cancellation on error or Ctrl+C
- Building CLI tools that process many items (downloads, scans, migrations) through one pipeline
- Needing progress tracking that can hook to any UI (terminal, web) plus resume capability

**Requires:** `go-foundations` for project layout, modern Go idioms, and (for the Highway) the
`utils/` package and `--for-ai` flag — load it alongside this skill.

**Related skills:**
- `go-cli` - Cobra setup, flags, and the sequential output-lifecycle patterns (the simple,
  non-concurrent counterpart to the Highway display)

---

# Part 1 — Concurrency Primitives

**Vanilla concurrency for use inside jobs or standalone operations.**

## Pattern Selection

| Need | Pattern |
|------|---------|
| Run N things concurrently, wait for all | WaitGroup (`wg.Go`) |
| Run N things, stop all on first error | errgroup |
| Limit concurrent operations to M | errgroup `SetLimit` or buffered-channel semaphore |
| Fan-out work, fan-in results | Fan-out/Fan-in |

**Default choice:** use `errgroup` whenever operations can fail; add `g.SetLimit(N)` for bounded
concurrency. Drop to a raw `sync.WaitGroup` (via `wg.Go`) only when errors are genuinely
fire-and-forget.

```go
import "golang.org/x/sync/errgroup"

// The default: bounded concurrency with first-error cancellation.
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10)
for _, item := range items {
    g.Go(func() error {
        return process(ctx, item)
    })
}
return g.Wait() // first error, or nil
```

Full code for all six patterns (WaitGroup fire-and-forget, errgroup, errgroup+limit, buffered-channel
semaphore, result collection, fan-out/fan-in) plus context-cancellation snippets lives in
`./references/concurrency-patterns.md`.

## Quick Reference

| Pattern | Import | When to Use |
|---------|--------|-------------|
| `sync.WaitGroup` (`wg.Go`) | `sync` | Fire N, wait, ignore errors |
| `errgroup.Group` | `golang.org/x/sync/errgroup` | Fire N, stop on first error |
| `errgroup.SetLimit(M)` | `golang.org/x/sync/errgroup` | Bounded concurrency + errors |
| Buffered chan semaphore | builtin | Bounded concurrency without errgroup |
| Fan-out/Fan-in | builtin | Stream processing with workers |

## Common Mistakes

| Mistake | Problem | Fix |
|---------|---------|-----|
| Manual `wg.Add(1)` + `go func` + `defer wg.Done()` | Verbose, easy to mis-pair | Use `wg.Go(fn)` — it handles Add/Done |
| Not checking `ctx.Done()` | Can't cancel | Check in loops and before work |
| Closing channel from receiver | Panic | Only the sender closes |
| Acquiring a semaphore inside the goroutine | Spawns unbounded goroutines | Acquire before `wg.Go`, release with `defer` inside |

**Note:** Go 1.22+ scopes loop variables per-iteration, so the old `item := item` capture trick
before goroutines is unnecessary on the 1.26+ baseline.

---

# Part 2 — Highway Job Pipeline Pattern (CLI Only)

**Unified job execution with progress tracking and state persistence.**

**Applies to CLI Only projects.** The highway/display pattern assumes the `utils/` package and
`--for-ai` flag exist (see `go-foundations` and `go-cli`). CLI + Web projects do not use it.

## When to Use the Highway

Use the Highway pattern when:
- Building CLI tools that process multiple items (downloads, scans, migrations)
- Need configurable concurrency (1 worker or N workers)
- Want progress tracking that can hook to any UI (terminal, web)
- Need Ctrl+C graceful shutdown with resume capability
- Multiple entry points should use the same execution pipeline

The Highway is built on the Part 1 primitives — workers are goroutines pulling from a channel,
exactly the fan-out pattern, wrapped with state tracking and a progress display.

## The Highway Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Entry                            │
│   cmd/download.go, cmd/scan.go, cmd/batch.go                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Job Creation                           │
│   Parse flags/config → Create []Job                         │
│   Each job knows its type, payload, and how to run          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        HIGHWAY                              │
│   Worker 1 / Worker 2 / Worker N (lanes)                    │
│   • Pull jobs from queue                                    │
│   • Execute job.Run(ctx, progress)                          │
│   • Track completion in state                               │
│   • Persist state on Ctrl+C                                 │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
   Progress Channel (UI agnostic)   State Persistence (.toolname-state)
   Terminal, Web, or nothing        Resume from where you left off
```

## Core Components

### Job Interface

```go
type Job interface {
    ID() string    // Unique identifier for tracking
    Type() string  // Job type for unmarshaling on resume
    Run(ctx context.Context, progress chan<- Progress) error
    Marshal() ([]byte, error) // For state persistence
}

// For resuming - registered per job type
type JobUnmarshaler func(data []byte) (Job, error)
```

### Progress Struct

Jobs send progress updates through a channel. Two update kinds:
- **Progress** (`ProgressTypeProgress`): known total → renders a progress bar
- **SubStatus** (`ProgressTypeSubStatus`): unknown total → renders substatus text

| Field | Purpose |
|-------|---------|
| `JobID` | Which job this is from |
| `Type` | `ProgressTypeProgress` or `ProgressTypeSubStatus` |
| `Message` | Short status shown next to job ID (e.g., "Downloading") |
| `SubStatus` | Detailed substatus text (SubStatus type) |
| `Current` / `Total` | Progress numerator / denominator |
| `Extra` | Extra info after percentage (e.g., "125MB/1GB") |
| `Done` | True when job complete |
| `Error` | Non-nil if job failed |

### Highway API Surface

```go
func New(workers int, statePath string) *Highway
func (h *Highway) RegisterType(jobType string, unmarshal JobUnmarshaler)
func (h *Highway) Submit(jobs ...Job)
func (h *Highway) Run(ctx context.Context) error
func (h *Highway) Progress() <-chan Progress
func (h *Highway) LoadState() error
```

The complete Highway implementation (struct, worker loop, `Run`, state save/load) is in
`./references/highway-template.md`.

## Directory Structure

```
cmd/
├── root.go              # Root command, global flags
├── download.go          # Creates download jobs → submits to highway
├── scan.go              # Creates scan jobs → submits to highway
└── resume.go            # Loads state file → submits pending jobs

internal/
├── highway/             # The execution engine (highway.go, state.go, progress.go)
├── display/             # Terminal UI for job progress (display.go)
├── jobs/                # Concrete job types (http_download.go, s3_scan.go, ...)
└── aws/                 # Shared helpers (optional)
```

## Implementing Concrete Jobs

Each job type is a struct that implements the `Job` interface. See
`./references/job-examples.md` for complete examples — a simple config-only job
(`S3PublicAccessJob`) and a resumable job with partial progress (`HTTPDownloadJob` that tracks
`CompletedParts` and skips them on resume).

Shape of a job's `Run`: do work, emit `Progress` updates through the channel, and send a final
`Progress{JobID: j.ID(), Done: true}` (or return an error) when finished.

## State Persistence

State file: `.toolname-resume-state.json` in the working directory. It records `completed` job IDs
and `pending` jobs (id, type, and marshaled data) so a resume can deserialize pending jobs via
registered unmarshalers.

```json
{
  "completed": ["job-1", "job-2"],
  "pending": [
    {
      "id": "http-bigfile.zip",
      "type": "http-download",
      "data": { "url": "https://example.com/bigfile.zip", "completedParts": [0, 1, 2] }
    }
  ]
}
```

The Highway saves state on Ctrl+C (`ctx.Done()`), deletes it on clean completion, and rebuilds
pending jobs in `LoadState()`. Full save/load code is in `./references/highway-template.md`.

## CLI Usage Pattern

A command creates the Highway, registers job types for resume, submits jobs, starts the display,
and runs until done or Ctrl+C:

```go
func runDownload(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    hw := highway.New(workers, ".downloader-state.json")
    hw.RegisterType("http-download", jobs.UnmarshalHTTPDownload)

    disp := display.New(display.DefaultConfig())
    for _, url := range urls {
        job := jobs.NewHTTPDownload(url, outputDir)
        disp.RegisterJob(job.ID())
        hw.Submit(job)
    }

    disp.Start(hw.Progress()) // consume progress channel
    err := hw.Run(ctx)
    disp.Stop()               // show final summary
    return err
}
```

A `resume` command does the same but calls `hw.LoadState()` instead of submitting fresh jobs.
Full command and resume examples are in `./references/highway-template.md`.

## Progress Display

The display manager aggregates job states and renders an inline terminal UI that updates every
200ms. In AI mode (`--for-ai`), it skips the interactive TUI and prints sequential plain-text
lines instead:

```
[INFO] http-bigfile.zip: Downloading 62% (485MB/782MB)
[OK] http-bigfile.zip: Done
[ERROR] s3-scan: timeout connecting to server
```

Each running job renders in two lines (job line + progress bar OR substatus, never both). The
same `Progress` channel can also feed a websocket for a web UI. Full display implementation —
terminal layout, progress-bar format, AI-mode branch, web hook — is in
`./references/display-template.md`.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Job owns its output | Jobs write their own results; highway doesn't care |
| Job owns its state | Each job type marshals whatever it needs for resume |
| Progress is a struct | Simple data, no behavior; any UI can consume |
| Highway is type-agnostic | Only knows the Job interface; doesn't care what's inside Run() |
| State file in working dir | `.toolname-state.json` — simple, discoverable |
| Continue on error | Mark failed, skip on resume (don't retry automatically) |

## When NOT to Use the Highway

Use the Part 1 primitives directly when:
- Work is one-shot (no resume needed)
- No progress tracking required
- Independent tasks with no coordination
- Quick script, not a polished CLI tool

---

## References

| File | Purpose |
|------|---------|
| `./references/concurrency-patterns.md` | Full code for the six Part 1 concurrency patterns + context cancellation |
| `./references/highway-template.md` | Complete Highway implementation (engine, state save/load, CLI + resume usage) |
| `./references/job-examples.md` | Example job types (simple and resumable) |
| `./references/display-template.md` | Terminal UI display manager (+ AI mode, web hook) |
