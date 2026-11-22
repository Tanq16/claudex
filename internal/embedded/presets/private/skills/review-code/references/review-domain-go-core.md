# Review Domain: Go Core

**Applies to:** every Go project type.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/go-project-layout/SKILL.md`
- `[SKILLS_DIR]/go-idioms/SKILL.md`
- `[SKILLS_DIR]/write-unit-tests/SKILL.md`

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Category 1: Project Layout

| Check | How to verify |
|---|---|
| Project type is unambiguous | Glob for `go.mod`, `cmd/`, `utils/`, `internal/server/static/`, `main.go`; name the type before checking anything else, since several later checks invert by type |
| Entry point | Read `main.go` |
| Root command file | Glob for `cmd/root.go` |
| `internal/` over `pkg/` | Glob both; for anything under `pkg/`, check whether another repository could plausibly import it |
| Subcommand packages | Glob `cmd/*/`; read each parent command declaration |
| Frontend asset location | Glob for a `static/` tree and check where it sits |
| `utils/` presence | Glob for `utils/` at the project root, then compare against what the project type calls for |

## Category 2: Modern Go

Flag only the patterns below. A classic `for i := 0; i < n; i++` loop, a manual slice operation that `slices` could do, and similar purely stylistic modernizations generate noise without changing behavior, so they are out of scope here even though the skill prefers the modern form.

| Check | How to verify |
|---|---|
| `go.mod` version directive | Read `go.mod` |
| `any` over `interface{}` | Grep `.go` files for `interface{}` |
| `wg.Go` over manual pairing | Grep for `wg.Add(1)` immediately preceding a `go func` |
| Typed atomics | Grep for `atomic.AddInt`, `atomic.LoadInt`, `atomic.StoreInt`, `atomic.SwapInt` |
| `errors.AsType` over the two-step | Grep for `errors.As(` |
| `encoding/json/v2` in new code | Grep for `"encoding/json"`, `MarshalIndent`, and `json.RawMessage`; v1 accepts duplicate object member names and invalid UTF-8 that v2 rejects, so this is a behavior gap rather than a rename |

## Category 3: Dependencies

| Check | How to verify |
|---|---|
| Every dependency is justified | Read `go.mod`; for each non-pre-approved entry, work out whether the standard library covers it. Flag only where a clear stdlib alternative exists, as `uuid` now is for `google/uuid` |
| Stack matches the project type | Read `go.mod` and compare the CLI-only packages present or absent against what the type calls for |
| Nothing unused | Cross-reference `go.mod` requires against actual imports |
| Versions current | Read `go.mod`, check each direct dependency against its latest stable release |

## Category 4: Logging and Config

| Check | How to verify |
|---|---|
| One logging library | Grep for zerolog imports and for `"log"`; the standard library logger is a defect in every project type |
| Debug gating | Read `cmd/root.go` and its `setupLogs`; `--debug` moves the level and nothing else |
| Level comes from the library | Grep log calls for a level token written into the message text |
| Writer follows the terminal | Read `setupLogs` for `ConsoleWriter` on a terminal and the plain writer otherwise |
| No cross-contamination | In a hybrid, grep `internal/server/` for `utils` imports; the printers are the CLI surface's and the logger is shared |
| Config loader location | Glob for `utils/config.go` or a config package under `internal/`, then compare against the type |
| Config precedence | Read the loader and trace the order in which each source is applied, expecting flag, then environment, then file, then default |
| Config directory | Grep for `os.UserHomeDir` and `UserConfigDir`; check the path resolves to `~/.config/<app>` and that no flag relocates it |
| Credential file modes | Grep for `MkdirAll` and `WriteFile` on that directory for `0700` and `0600` |
| Environment variable naming | Grep `os.Getenv` calls; a variable the tool owns is prefixed with the tool's name |

## Category 5: Comments

| Check | How to verify |
|---|---|
| Comments map to an exception | Left to Step 3c, which audits every comment against the two exceptions in AGENTS.md once for the whole review |
| No scaffolding behind `//` | Grep for commented-out code and example blocks |
| No stale comments | Compare each comment against the code beneath it |

## Category 6: Tests

| Check | How to verify |
|---|---|
| Placement | Glob `*_test.go`; check it sits in the package it covers and that there is one file per package |
| Scenario-driven | Read the test tables and assess whether the cases are edge cases or restatements of the happy path |
| Can fail for a real reason | For each test, ask what change to the implementation would make it fail |
| Nothing trivial tested | Look for tests over getters, thin wrappers, and code with no branching |
| Units stay units | Grep tests for network calls, server startup, and cross-package end-to-end flows |
| Modern test idioms | Grep for `context.Background()` inside tests, `b.N` in benchmarks, and `omitempty` on time, duration, struct, slice, and map fields |

---

## Output Format

```
## Domain: Go Core

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** [severity] (skill-name: section)
   - **Where:** file:line
   - **Current:** [what the code does now]
   - **Expected:** [what the cited skill section says]
   - **Fix:** [the specific action]

### [SKIP] Category Name

Not applicable to this project type.
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
