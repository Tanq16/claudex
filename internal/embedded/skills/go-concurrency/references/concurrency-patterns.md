# Concurrency Patterns

Full code for the vanilla concurrency primitives. These run inside a job's `Run()` method or in
standalone functions. Pick the pattern from the selection table in `SKILL.md`.

> **Logging note:** the fire-and-forget pattern below logs errors. Use the project's logging
> convention from `go-foundations` — zerolog (`log.Error()...`) for CLI Only, `log.Printf("ERROR ...")`
> for CLI + Web. Never mix the two.

---

## Pattern 1: WaitGroup (Fire and Forget)

Run N operations concurrently, wait for all to complete. Errors are logged, not collected. Use
`wg.Go` (Go 1.25+) — it spawns the goroutine and handles `Add`/`Done` internally.

```go
import "github.com/rs/zerolog/log" // CLI Only; CLI + Web uses the std "log" package

func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Go(func() {
            select {
            case <-ctx.Done():
                return
            default:
            }

            if err := process(item); err != nil {
                log.Error().Err(err).Str("item", item.ID).Msg("error processing item")
            }
        })
    }

    wg.Wait()
    return nil
}
```

**Use when:** errors can be logged and ignored, no need to stop on failure.

---

## Pattern 2: errgroup (Stop on First Error)

Run N operations concurrently. If any fails, cancel the rest and return the error.

```go
import "golang.org/x/sync/errgroup"

func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait() // Returns first error, or nil if all succeed
}
```

**Use when:** any failure should stop everything and return the error.

---

## Pattern 3: errgroup with Concurrency Limit

Same as above, but limit to M concurrent operations.

```go
import "golang.org/x/sync/errgroup"

func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // Max 10 concurrent operations

    for _, item := range items {
        g.Go(func() error {
            return process(ctx, item)
        })
    }

    return g.Wait()
}
```

**Use when:** need bounded concurrency together with error handling. This is the preferred way to
limit concurrency.

---

## Pattern 4: Buffered Channel Semaphore

Limit concurrency without errgroup, when you don't need first-error cancellation.

```go
func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    sem := make(chan struct{}, 10) // Max 10 concurrent
    var wg sync.WaitGroup

    for _, item := range items {
        sem <- struct{}{} // Acquire (blocks if 10 already running)
        wg.Go(func() {
            defer func() { <-sem }() // Release
            process(ctx, item)
        })
    }

    wg.Wait()
    return nil
}
```

**Use when:** need bounded concurrency but not errgroup's error semantics.

---

## Pattern 5: Collecting Results

Run N operations concurrently, collect all results (not just errors).

```go
func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(items))

    for i, item := range items {
        g.Go(func() error {
            result, err := process(ctx, item)
            if err != nil {
                return err
            }
            results[i] = result // Safe: each goroutine writes to a unique index
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return err
    }

    // Use results...
    return nil
}
```

**Use when:** need to collect results from concurrent operations.

---

## Pattern 6: Fan-Out/Fan-In (Simplified)

Distribute work to N workers, merge results. Use when processing a stream of items.

```go
func (j *MyJob) Run(ctx context.Context, progress chan<- Progress) error {
    numWorkers := 5

    jobs := make(chan Item)
    results := make(chan Result)

    // Start workers
    var wg sync.WaitGroup
    for range numWorkers {
        wg.Go(func() {
            for item := range jobs {
                result := process(ctx, item)
                select {
                case <-ctx.Done():
                    return
                case results <- result:
                }
            }
        })
    }

    // Close results when workers done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Send jobs (in goroutine to not block)
    go func() {
        defer close(jobs)
        for _, item := range items {
            select {
            case <-ctx.Done():
                return
            case jobs <- item:
            }
        }
    }()

    // Collect results
    var allResults []Result
    for result := range results {
        allResults = append(allResults, result)
    }

    return nil
}
```

**Use when:** processing many items through a fixed number of workers. More complex than errgroup
— only use if you specifically need the channel-based worker pool.

---

## Context Cancellation

Always respect context in long-running operations:

```go
// Check before starting work
select {
case <-ctx.Done():
    return ctx.Err()
default:
}

// Check during loops
for _, item := range items {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    process(item)
}

// Pass context to functions that support it
result, err := http.DefaultClient.Do(req.WithContext(ctx))
```

When you need to know *why* a context was cancelled, use `context.WithCancelCause(parent)` and
read `context.Cause(ctx)` — see the modern-Go reference in `go-foundations`.
