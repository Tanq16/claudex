# Modern Go (target: 1.26+)

All Go in these projects targets **Go 1.26 or newer**. Write to that baseline: prefer modern
built-ins and standard-library helpers over legacy patterns, and never reach for an outdated
idiom when a current one exists. There is no version detection — assume 1.26+.

The `go.mod` `go` directive should read `go 1.26` (or newer) on every project.

This reference covers production code. Test-specific modern idioms (`t.Context()`, `b.Loop()`,
`omitzero` JSON tags) live in the **Unit Testing** section of `SKILL.md`.

---

## Types and built-ins

- **`any`** instead of `interface{}`.
- **`min` / `max`** instead of hand-rolled comparisons: `max(a, b)`, `min(start+size-1, end)`.
- **`clear`** to empty a map (`clear(m)`) or zero a slice's elements (`clear(s)`).

## `slices` package

Prefer these over manual loops:

- `slices.Contains(items, x)` — membership test.
- `slices.Index(items, x)` / `slices.IndexFunc(items, func(it T) bool { ... })` — find index (`-1` if absent).
- `slices.Sort(items)` for ordered types; `slices.SortFunc(items, func(a, b T) int { return cmp.Compare(a.X, b.X) })`.
- `slices.Max(items)` / `slices.Min(items)`.
- `slices.Reverse(items)`, `slices.Compact(items)` (drop consecutive dupes), `slices.Clone(s)`, `slices.Clip(s)`.

## `maps` package

- `maps.Clone(m)` instead of manual copy loops.
- `maps.Copy(dst, src)` to merge entries.
- `maps.DeleteFunc(m, func(k K, v V) bool { ... })` for conditional deletion.
- `maps.Keys(m)` / `maps.Values(m)` return iterators (see Iteration below).

## `cmp` package

- `cmp.Compare(a, b)` for ordered comparison (pairs with `slices.SortFunc`).
- `cmp.Or(a, b, c, "default")` returns the first non-zero value:

```go
name := cmp.Or(os.Getenv("NAME"), cfg.Name, "default")
```

## Iteration (range-over-int and iterators)

- **Range over an int** when you just need a count: `for i := range len(items)` or `for range n`,
  instead of `for i := 0; i < n; i++`.
- **`slices.Collect(iter)`** to build a slice from an iterator; **`slices.Sorted(iter)`** to
  collect and sort in one step.

```go
keys := slices.Collect(maps.Keys(m))       // not: for k := range m { keys = append(keys, k) }
sortedKeys := slices.Sorted(maps.Keys(m))  // collect + sort
for k := range maps.Keys(m) { process(k) } // iterate directly
```

## `sync` package

- **`wg.Go(fn)`** (Go 1.25+) instead of `wg.Add(1)` + `go func() { defer wg.Done(); ... }()`.
  This is the default way to spawn WaitGroup goroutines (see `go-concurrency`).

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() { process(item) })
}
wg.Wait()
```

- **`sync.OnceFunc`** / **`sync.OnceValue`** instead of `sync.Once` plus a wrapper:

```go
warmCache := sync.OnceFunc(func() { /* runs at most once */ })
getConfig := sync.OnceValue(func() Config { return loadConfig() })
```

- Typed atomics — `atomic.Bool`, `atomic.Int64`, `atomic.Pointer[T]` — instead of the
  `atomic.StoreInt32`/`LoadInt32` free functions.

## `errors` package

- **`errors.Is(err, target)`** instead of `err == target` (works through wrapped errors).
- **`errors.Join(err1, err2, ...)`** to combine multiple errors.
- **`errors.AsType[T](err)`** (Go 1.26) instead of the `errors.As(err, &target)` two-step:

```go
if pathErr, ok := errors.AsType[*os.PathError](err); ok {
    handle(pathErr)
}
```

## `context` cancellation causes

- `ctx, cancel := context.WithCancelCause(parent)` then `cancel(err)`; read it with
  `context.Cause(ctx)`.
- `context.WithTimeoutCause(parent, d, err)` / `context.WithDeadlineCause(...)` attach a cause to
  timeouts.
- `context.AfterFunc(ctx, cleanup)` runs `cleanup` when `ctx` is cancelled.

## `strings` / `bytes`

- **`strings.Cut(s, sep)`** → `before, after, found := strings.Cut(s, ",")` instead of
  `Index` + slicing. Same for `bytes.Cut`.
- **`strings.CutPrefix` / `strings.CutSuffix`**:

```go
if rest, ok := strings.CutPrefix(s, "id:"); ok {
    use(rest)
}
```

- **`strings.SplitSeq` / `strings.FieldsSeq`** (and `bytes.SplitSeq` / `bytes.FieldsSeq`) when
  iterating over split results — they avoid allocating the full slice:

```go
for part := range strings.SplitSeq(s, ",") {
    process(part)
}
```

## `time`

- **`time.Since(start)`** instead of `time.Now().Sub(start)`.
- **`time.Until(deadline)`** instead of `deadline.Sub(time.Now())`.
- **`time.Tick`** is safe to use freely — as of Go 1.23 the garbage collector reclaims unreferenced
  tickers, so there is no longer a reason to prefer `NewTicker` + `Stop` when `Tick` will do.

## Pointers — `new(value)`

Go 1.26 extends `new` to take an expression, returning a pointer to it. Use it for pointer
struct fields instead of the `x := v; &x` dance. Type is inferred (`new(0)` → `*int`).

```go
cfg := Config{
    Timeout: new(30),   // *int
    Debug:   new(true), // *bool
}
```

Do not write `x := v; &x`, and do not add redundant casts like `new(int(0))` — just `new(0)`.
