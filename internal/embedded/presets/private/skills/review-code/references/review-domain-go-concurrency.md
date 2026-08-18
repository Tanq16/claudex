# Review Domain: Go Concurrency

**Applies to:** any Go project type, and only where the patterns are actually present.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-concurrency/SKILL.md`
- `[SKILLS_DIR]/go-job-pipeline/SKILL.md` (Category 2 only)

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Pre-check

1. **Concurrency:** grep for `go func`, `sync.WaitGroup`, `wg.Go`, and `errgroup`. Skip Category 1 when none appear.
2. **Pipeline:** glob for `internal/highway/`, `internal/jobs/`, or an equivalent job execution package. Skip Category 2 when absent.

Reporting a skip is the correct outcome for a project that does neither, and inventing findings for absent patterns is the failure mode this pre-check exists to prevent.

---

## Category 1: Concurrency Primitives

| Check | How to verify |
|---|---|
| Primitive matches the error semantics | For each concurrent site, read whether the operations can fail and what happens to the error |
| Goroutine spawning form | Grep for `wg.Add` and `wg.Done` |
| Semaphore acquisition point | For each buffered-channel semaphore, read whether the acquire is inside or outside the goroutine |
| Channel closing side | For each `close(` on a channel, trace whether the closer is the sender |
| Cancellation is observed | Grep for `ctx.Done()` near long-running loops and before blocking work |
| Context propagation | Grep for `context.Background()` and `context.TODO()` inside functions that already receive a context |
| Bounded concurrency where it matters | Read sites that spawn one goroutine per input for any limit at all |
| Result collection safety | For shared slices and maps written from goroutines, check whether each write targets a unique index or is guarded |
| No leftover loop capture | Grep for `x := x` immediately before a goroutine launch |

## Category 2: Job Pipeline

| Check | How to verify |
|---|---|
| Job interface | Read the interface declaration and its method set |
| Progress type | Read the progress struct and its fields |
| Engine structure | Read the engine for its worker count, queue, progress channel, and completion tracking |
| Progress channel is buffered | Read the channel construction |
| Feeder does not hold the lock while sending | Read the feeder goroutine |
| State file format | Read the persisted structures |
| State file mode | Read the mode argument on the state write |
| State output goes through the printer | Grep the engine for `fmt.Print` and `fmt.Printf` |
| State deleted on clean completion | Read the successful-completion path |
| Failed jobs on resume | Read what happens to a job that returned an error |
| Partial progress is recorded after success | For a resumable job, read where its completed-work list is appended relative to the work itself |
| Type registration precedes load | Read the command wiring for the order of registration and state loading |
| Signal handling | Grep for `signal.NotifyContext` |
| Display package separation | Glob for a display package and check the engine does not import it |
| Display AI branch | Read the display entry point for a `GlobalForAIFlag` branch |
| Display colors | Grep the display for color construction and check ANSI indices against hex |

---

## Output Format

```
## Domain: Go Concurrency

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** [severity] (skill-name: section)
   - **Where:** file:line
   - **Current:** [what the code does now]
   - **Expected:** [what the cited skill section says]
   - **Fix:** [the specific action]

### [SKIP] Category Name

Not applicable: [reason, such as "no goroutines detected" or "no pipeline engine found"].
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
