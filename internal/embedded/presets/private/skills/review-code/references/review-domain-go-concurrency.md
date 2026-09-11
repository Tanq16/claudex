# Review Domain: Go Concurrency

**Applies to:** any Go project type, and only where the patterns are actually present.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-concurrency/SKILL.md`

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Pre-check

Grep for `go func`, `sync.WaitGroup`, `wg.Go`, and `errgroup`. Skip Category 1 when none appear.

Reporting a skip is the correct outcome for a project that runs nothing concurrently, and inventing findings for absent patterns is the failure mode this pre-check exists to prevent.

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

Not applicable: [reason, such as "no goroutines detected"].
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
