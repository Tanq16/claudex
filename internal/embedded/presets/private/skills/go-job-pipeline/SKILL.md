---
name: go-job-pipeline
description: The Highway pattern - a resumable multi-job pipeline for Go CLI tools, with worker lanes, a UI-agnostic progress channel, and Ctrl+C resume. Use when a CLI processes many items through one pipeline (downloads, scans, migrations), needs configurable worker counts, needs progress that can drive a terminal or a web UI, or needs to pick up where an interrupt left off. Triggers on internal/highway/, internal/display/, internal/jobs/, a Job interface, a Progress channel, RegisterType, LoadState, and a resume command.
user-invocable: false
---

# Go Job Pipeline

**Many items, N worker lanes, one progress channel, and a state file that survives Ctrl+C.**

This applies to CLI Only projects and the command surface of a CLI + Web hybrid, since it assumes the `utils` package and the `--for-ai` flag exist.

Reach for it when a tool runs many items through a shared pipeline that needs progress reporting and resumability. One-shot work with no resume and no progress uses plain goroutine primitives, because the `Job` interface and the state file are pure overhead when nothing is going to be resumed.

```
CLI entry (cmd/download.go, cmd/scan.go)
   ↓  parse flags into []Job
Highway: N worker lanes pull from the queue, run job.Run(ctx, progress), track completion
   ↓                                    ↓
Progress channel (any UI)      State file (.toolname-resume-state.json)
```

## Layout

```
cmd/
├── download.go     # builds download jobs, submits them
├── scan.go         # builds scan jobs, submits them
└── resume.go       # loads the state file, submits what is pending

internal/
├── highway/        # the engine: highway.go, state.go, progress.go
├── display/        # terminal UI consuming the progress channel
└── jobs/           # concrete job types
```

## Design Decisions

| Decision | Reason |
|---|---|
| A job writes its own output | the engine never learns what a job produces, so a new job type needs no engine change |
| A job marshals its own state | only the job knows which fields matter for resume |
| `Progress` is data with no behavior | any consumer can render it, including a websocket feeding a browser |
| The engine knows only the `Job` interface | adding a job type is one file, not a switch statement in the engine |
| The state file sits in the working directory | it is discoverable next to the output it accompanies |
| A failed job is marked done | resume skips it rather than retrying automatically, since a retry loop on a permanent failure never terminates |

## The Job Interface

```go
type Job interface {
    ID() string    // unique, used for tracking and for skipping on resume
    Type() string  // resolves the unmarshaler on resume
    Run(ctx context.Context, progress chan<- Progress) error
    Marshal() ([]byte, error)
}

type JobUnmarshaler func(data []byte) (Job, error)
```

A `Run` does its work, emits `Progress` as it goes, and finishes by sending `Progress{JobID: j.ID(), Done: true}` or returning an error.

A job that carries partial progress records it in its own fields and skips what is already finished, which is what makes resume mean something more than restart:

```go
type HTTPDownloadJob struct {
    URL            string `json:"url"`
    OutputDir      string `json:"outputDir"`
    TotalSize      int64  `json:"totalSize"`      // discovered on the first run
    CompletedParts []int  `json:"completedParts"` // the resume state
}

func (j *HTTPDownloadJob) Run(ctx context.Context, progress chan<- highway.Progress) error {
    totalParts := int((j.TotalSize + partSize - 1) / partSize)
    for part := range totalParts {
        if slices.Contains(j.CompletedParts, part) {
            continue
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        if err := j.downloadPart(ctx, part); err != nil {
            return fmt.Errorf("download part %d: %w", part, err)
        }
        j.CompletedParts = append(j.CompletedParts, part) // only after the part lands
        progress <- highway.Progress{
            JobID:   j.ID(),
            Message: fmt.Sprintf("Part %d/%d", len(j.CompletedParts), totalParts),
            Current: int64(len(j.CompletedParts)),
            Total:   int64(totalParts),
        }
    }
    progress <- highway.Progress{JobID: j.ID(), Done: true}
    return nil
}

func (j *HTTPDownloadJob) Marshal() ([]byte, error) { return json.Marshal(j) }

func UnmarshalHTTPDownload(data []byte) (highway.Job, error) {
    var job HTTPDownloadJob
    if err := json.Unmarshal(data, &job); err != nil {
        return nil, err
    }
    return &job, nil
}
```

A part is appended to `CompletedParts` only after it succeeds, because recording it first means a crash mid-part loses those bytes forever.

## Progress

Two kinds of update, distinguished so a renderer knows whether it can draw a bar.

| Field | Purpose |
|---|---|
| `JobID` | which job sent it |
| `Type` | `ProgressTypeProgress` when the total is known, `ProgressTypeSubStatus` when it is not |
| `Message` | short status shown next to the job ID |
| `SubStatus` | detailed line, for the SubStatus kind |
| `Current` / `Total` | numerator and denominator |
| `Extra` | trailing detail after the percentage, such as `125MB/1GB` |
| `Done` | the job finished |
| `Error` | the job failed |

## The Engine

```go
package highway

type Highway struct {
    workers      int
    statePath    string
    unmarshalers map[string]JobUnmarshaler

    mu        sync.Mutex
    pending   []Job
    completed map[string]bool
    progress  chan Progress
}

func New(workers int, statePath string) *Highway {
    return &Highway{
        workers:      max(workers, 1),
        statePath:    statePath,
        unmarshalers: make(map[string]JobUnmarshaler),
        completed:    make(map[string]bool),
        progress:     make(chan Progress, 100),
    }
}

func (h *Highway) RegisterType(jobType string, unmarshal JobUnmarshaler) {
    h.unmarshalers[jobType] = unmarshal
}

func (h *Highway) Submit(jobs ...Job) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.pending = append(h.pending, jobs...)
}

func (h *Highway) Progress() <-chan Progress { return h.progress }

func (h *Highway) Run(ctx context.Context) error {
    jobCh := make(chan Job)
    var wg sync.WaitGroup

    for range h.workers {
        wg.Go(func() {
            for job := range jobCh {
                h.executeJob(ctx, job)
            }
        })
    }

    go func() {
        defer close(jobCh)
        h.mu.Lock()
        jobs := slices.Clone(h.pending)
        h.mu.Unlock()

        for _, job := range jobs {
            if h.isCompleted(job.ID()) {
                continue
            }
            select {
            case <-ctx.Done():
                return
            case jobCh <- job:
            }
        }
    }()

    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        close(h.progress)
        os.Remove(h.statePath)
        return nil
    case <-ctx.Done():
        wg.Wait()
        close(h.progress)
        return h.saveState()
    }
}

func (h *Highway) executeJob(ctx context.Context, job Job) {
    if err := job.Run(ctx, h.progress); err != nil {
        h.progress <- Progress{JobID: job.ID(), Done: true, Error: err, ErrMsg: err.Error()}
    }
    h.markCompleted(job.ID()) // a failure is recorded too, so resume skips it
}
```

The progress channel is buffered so a job emitting updates faster than the display consumes them keeps working instead of blocking on a render.

The feeder goroutine clones the pending slice under the lock and then releases it, since holding a mutex while sending on a channel deadlocks against anything that submits during the run.

The state file is deleted on clean completion, because a leftover file makes the next run look like a resume.

## State

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

```go
func (h *Highway) saveState() error {
    h.mu.Lock()
    defer h.mu.Unlock()

    var pendingJobs []persistedJob
    for _, job := range h.pending {
        if h.completed[job.ID()] {
            continue
        }
        data, err := job.Marshal()
        if err != nil {
            continue
        }
        pendingJobs = append(pendingJobs, persistedJob{ID: job.ID(), Type: job.Type(), Data: data})
    }

    state := persistedState{
        Completed: slices.Collect(maps.Keys(h.completed)),
        Pending:   pendingJobs,
    }
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return err
    }
    if err := os.WriteFile(h.statePath, data, 0600); err != nil {
        return err
    }
    utils.PrintInfo("State saved to " + h.statePath)
    return nil
}

func (h *Highway) LoadState() error {
    data, err := os.ReadFile(h.statePath)
    if err != nil {
        return err
    }
    var state persistedState
    if err := json.Unmarshal(data, &state); err != nil {
        return err
    }

    h.mu.Lock()
    for _, id := range state.Completed {
        h.completed[id] = true
    }
    h.mu.Unlock()

    for _, pj := range state.Pending {
        unmarshal, ok := h.unmarshalers[pj.Type]
        if !ok {
            return fmt.Errorf("unknown job type %q, register it with RegisterType", pj.Type)
        }
        job, err := unmarshal(pj.Data)
        if err != nil {
            return fmt.Errorf("failed to unmarshal job %s: %w", pj.ID, err)
        }
        h.Submit(job)
    }
    return nil
}

type persistedState struct {
    Completed []string       `json:"completed"`
    Pending   []persistedJob `json:"pending"`
}

type persistedJob struct {
    ID   string          `json:"id"`
    Type string          `json:"type"`
    Data json.RawMessage `json:"data"`
}
```

The state file is written at `0600`, matching every other file these projects write into a user's directory, because a job payload commonly carries paths, URLs, and identifiers from their environment.

The save reports through `utils.PrintInfo` rather than `fmt.Printf`, so the line obeys the same three tiers as everything else the tool prints.

## Wiring a Command

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

    disp.Start(hw.Progress())
    err := hw.Run(ctx)
    disp.Stop()
    return err
}
```

`signal.NotifyContext` turns Ctrl+C into a context cancellation, which is what lets the engine save state instead of the process dying mid-write.

A `resume` command is the same function with `hw.LoadState()` in place of the submit loop. Every job type is registered before `LoadState` runs, since the unmarshaler is what turns stored bytes back into a job.

## Display Contract

The display consumes the progress channel and owns the terminal. It lives in its own `internal/display/` package so nothing about the engine depends on how progress is rendered, and so the same channel can feed a websocket for a web UI.

- The refresh loop redraws on a ticker, 200ms by default. Redrawing on every update makes the terminal flicker under a fast job.
- A running job renders as two lines: the job line, then either a progress bar or a substatus line, never both. `ProgressTypeProgress` with a non-zero `Total` gets the bar; `ProgressTypeSubStatus` gets the text.
- Only the first few jobs render in detail, with the rest summarized as a queued count, because a hundred concurrent jobs cannot all be on screen.
- A final summary replaces the live view on `Stop`, reporting completed and failed counts.
- Colors follow the same rule as the rest of the CLI: ANSI indices 0 through 15, never hex, so the display adopts the user's terminal theme.

Under `--for-ai` the display skips the interactive view entirely and prints one plain-text line per update, since redrawing in place produces cursor escapes that a reading agent cannot parse.

```go
func (d *Display) Start(updates <-chan Progress) {
    if utils.GlobalForAIFlag {
        d.startAI(updates)
        return
    }
    // consume updates into state, then redraw on the ticker
}
```

```
[INFO] http-bigfile.zip: Downloading 62% (485MB/782MB)
[OK] http-bigfile.zip: Done
[ERROR] s3-scan: timeout connecting to server
```
