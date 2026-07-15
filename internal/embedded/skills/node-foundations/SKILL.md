---
name: node-foundations
description: Use when writing, refactoring, reviewing, or testing ANY Node.js code. The canonical reference for Node conventions — project taxonomy (Node Web Only), project layout, modern Node idioms (pure ESM, node: builtins, target Node >=24, top-level await, node:test, structuredClone, AbortController), edge-case-driven unit testing with node:test, core principles, dependency selection, console logging, and config management. Load this for any Node work. The `develop` skill is the per-task entry point that selects and applies these conventions.
user-invocable: false
---

# Node Foundations

**Shared patterns and conventions for all Node.js projects.**

## When to Use

Use this skill when:
- Writing, refactoring, reviewing, or testing any Node.js code
- Checking the canonical project taxonomy and layout for Node projects
- Implementing console logging or config management
- Reviewing a Node project for convention compliance

This skill is the *reference* for how Node projects should look. The `develop` skill is the entry point that loads this and the other relevant skills for a given task and holds the work to them. The other Node skills (`node-backend`, `node-frontend`) defer to the taxonomy and layout defined here rather than redefining them.

**Related skills:**
- `node-backend` - HTTP + WebSocket server, routing, auth, state
- `node-frontend` - vendored single-page frontend served by the Node process
- `project-readme` - README structure and templates
- `project-ci-cd` - Makefile, GitHub Actions, releases (Node artifact/toolchain layer)

## Project Taxonomy

This is the **canonical project taxonomy** for all Node work. Other skills (`node-backend`, `node-frontend`, `develop`, `review-code`) refer back to this type name rather than redefining it.

| Type | What it is | Defining markers |
|------|-----------|------------------|
| **Node Web Only** | A process that can be called and it serves — an HTTP + WebSocket server that also serves an embedded/vendored single-page frontend | `package.json` `"type":"module"` + `"engines":{"node":">=24"}`; `.node-version`; pure ESM under `src/`; vendored frontend under `public/`; a thin `bin/` launcher; `node:http` + `ws`; `console.*` logging; NO CLI framework |

**Node Web Only** is the **only** Node project type defined today, and it parallels the Go **Web Only** discipline: no utils package, no logging framework, an embedded/vendored frontend, and a Docker-or-tarball artifact.

**Out of scope (future).** "CLI Only", "CLI + Web", and "Library / Module" Node types do **not** exist in this taxonomy yet. Do not invent them. A Node project here is a Node Web Only server; anything else is future work and out of scope until this skill defines it.

## Project Layout

A Node Web Only project serves an HTTP + WebSocket backend plus a vendored single-page frontend. NO CLI framework (no commander/yargs equivalent) — a bare serve entry, optionally a thin hand-rolled argv read for the config path.

```
project-root/
├── package.json          # "type":"module", minimal deps, "engines": { "node": ">=24" }
├── .node-version         # pinned Node version
├── bin/                  # thin launcher entry (e.g. app.js) — parses config path, calls src/
├── src/                  # backend: server, routing, auth, state (node:http + ws)
├── public/               # frontend: index.html, css/, js/, fonts/ (vendored woff2)
├── test/                 # node:test suites (+ optional e2e.mjs)
├── config.example.json
├── Makefile              # vendor assets + build/release orchestration
└── .github/workflows/    # release
```

**Key rules:**
- `bin/` holds only the launcher: it reads a `--config` path from argv and calls into `src/`. No command framework, no subcommands beyond serving.
- Backend code lives in `src/` (server, routing, auth, state); the frontend is vendored into `public/`. Keep them separate.
- Frontend assets are self-hosted under `public/` (`index.html`, `css/`, `js/`, `fonts/`) — no runtime CDN. See `node-frontend` for the vendoring rules.
- `test/` holds `node:test` unit suites; a live end-to-end script (e.g. `test/e2e.mjs`) is separate and optional.
- Config ships as `config.example.json`; a user `config.json` is deep-merged over built-in defaults at boot.

## Core Principles

### KISS (Keep It Stupid Simple)
- **Builtins first.** Reach for `node:` built-in modules before any dependency: `node:http`, `node:crypto`, `node:fs`, `node:path`, `node:url`, `node:test`. Add a dependency only when a builtin genuinely cannot cover the need.
- A well-scoped Node Web Only app has a **tiny** dependency list. The reference model's entire runtime dep set is `ws` (WebSocket), `node-pty` (native PTY), and `@xterm/xterm` (terminal frontend) — three deps, each filling a gap no builtin covers.
- Evaluate whether a dependency is *justified*, not whether it appears on a fixed allow-list — the same judgment go-foundations applies to Go deps.

### YAGNI (You Ain't Gonna Need It)
- Don't build for all future needs
- Make code extendable, not comprehensive
- Focus on current requirements

### DRY (Don't Repeat Yourself)
- Abstract utilities: config loading, the HTTP router, common request handling
- DON'T abstract: ad-hoc async flows, one-off event wiring — keep them vanilla and explicit
- Prefer explicit control flow over generic abstractions

## Comments and Code Style

Default to zero comments. Code should read as self-documenting; a comment earns its place only by saying what the code cannot.

The sole test for a comment is whether its *why* is **load-bearing** — would a competent reader misread the intent, a trade-off, or a non-obvious constraint without it? If not, delete it. That it reads as a reasonable explanation, or that the code is intricate, does not qualify it. Judge every comment on its own merit against this test.

- **Comment the *why*, not the *what*** — the code already states what it does. Reserve comments for intent, trade-offs, and non-obvious constraints (why this order, why this bound, what a subtle edge case guards against).
- **No redundant comments** — never restate the code or narrate obvious control flow (`// increment i`, `// loop over items`). A comment that mirrors the line below it is noise. This holds even when the task says "add comments" or "explain what it does": tidy the code and add a why only where one is warranted, never what-narration. Never embed example or scaffolding code behind `//` — that is documentation, not a comment.
- **One line by default** — a single line is usually enough to state the why, and often none is needed. Only span multiple lines when there is genuinely more to explain. Internal, unexported helpers rarely need a doc comment at all — don't add a paragraph that just restates the signature.
- **Keep them current** — a stale comment is worse than none. Update or delete comments when the code they describe changes.

## Modern Node (target Node >=24)

All Node in these projects targets **Node 24 or newer**, pinned via `.node-version` and declared in `package.json` `"engines": { "node": ">=24" }`. Write to that baseline:

- **Pure ESM.** `package.json` has `"type": "module"`; use `import`/`export`, never `require`/`module.exports`. Top-level `await` is available and fine to use in module scope.
- **`node:`-prefixed builtins** always — `import { createServer } from 'node:http'`, `import { scrypt } from 'node:crypto'`, `import { readFile } from 'node:fs/promises'`. The prefix makes the builtin unambiguous and unshadowable.
- **Real Node.** Dev and test run on real Node (not a transpiler/bundler); prefer the standard-library helpers Node already ships (`structuredClone`, `AbortController`, `node:test`) over dependencies.

See `./references/modern-node.md` for the full curated set of idioms with examples.

## Dependency Selection

Choosing dependencies is a KISS decision. Apply this judgment when adding, auditing, or upgrading any dependency:

- **Prefer the built-in module** whenever a `node:` builtin can reasonably do the job (`node:http`, `node:crypto`, `node:fs`, `node:path`, `node:url`, `node:test`, `node:stream`, `node:events`, `node:child_process`).
- **Add a dependency only for a genuine gap** a builtin cannot cover — e.g. `ws` (a spec-complete WebSocket server, which `node:http` does not provide), a native addon like `node-pty`, or a vendored frontend package like `@xterm/xterm`.
- **Prefer well-maintained, standard choices** over niche alternatives.
- **Keep dependencies current** — track latest stable versions, prefer stable over pre-release, and read release notes for breaking changes before a major bump.

Apply this judgment whenever you add, audit, or upgrade a dependency — scan `package.json` (and any vendored/CDN pins in the frontend or Makefile), check the latest stable versions, and read release notes before bumping a major.

## Logging

**Node Web Only projects have NO logging-framework dependency** — no pino, no winston. Use `console.*` with manual level prefixes, exactly as Go Web Only projects use the standard `log` package.

```js
const ts = () => new Date().toISOString();

console.log(`${ts()} INFO listening on port ${port}`);
console.error(`${ts()} ERROR failed to validate token: ${err.message}`);
console.debug(`${ts()} DEBUG session ${id} attached`);
```

- Manual level prefixes: `INFO`, `ERROR`, `DEBUG`
- Timestamped, sequential, no color
- Keep messages generic — no module-name prefix required
- `console.error` for errors/fatals (stderr), `console.log` for normal output (stdout)

A small reusable logger helper with these prefixes lives in `./references/modern-node.md`.

## Config Management

Config is a user `config.json` **deep-merged over built-in defaults**, with env vars or flags permitted to override on top. Structure:

1. Built-in default config object (lowest priority) — the source of truth for shape and defaults.
2. User `config.json` deep-merged over the defaults (path passed via a `--config` flag; ships as `config.example.json`).
3. Environment variables / flags override individual values (highest priority).

**Secrets vs. state — keep them apart:**
- **Ephemeral session secret** — regenerated on every boot (e.g. `crypto.randomBytes`), held in memory only, **never persisted**. Restarting invalidates sessions by design.
- **Durable state** — only genuinely persistent facts (e.g. a mode flag) are written to `state.json`, created with file mode `0600`.

The deep-merge loader and a `state.json` writer live in `./references/project-templates.md`.

## Unit Testing

**Tests exist to prove logical correctness and robustness — not to chase coverage.**

Use the built-in `node:test` runner — no test-framework dependency. The philosophy is identical to go-foundations' Unit Testing: pin down edge cases, special conditions, and the inputs prone to breaking or throwing. A test that merely re-asserts the happy path so the suite "passes" adds noise, not safety.

### Principles

- **Scenario-driven, not coverage-driven.** Before writing tests, think through how the code can break: boundary values, empty/null/undefined inputs, zero-length and single-element arrays, malformed data, and anything that could throw. Those scenarios become the test cases.
- **Unit tests, not integration tests.** Test a module's own logic in isolation. Do not stand up servers, hit networks, or wire up WebSockets in unit tests — that is out of scope here.
- **Robustness focus.** A function that can throw should have a test proving it doesn't on the nasty inputs. Prefer table-driven tests to enumerate edge cases compactly.
- **Don't test the trivial.** Skip pedantic tests of getters, trivial wrappers, or code with no branching or failure modes.
- **Fully implement the code first.** Write complete, working implementations; then add tests for the scenarios that matter.

### Placement and running

- Tests live under `test/` as `*.test.js` (or colocated `*.test.js`), imported from `node:test`.
- Run with `node --test`; `node --test --watch` during development.
- A **live end-to-end script** (e.g. `test/e2e.mjs` that boots the server over http+ws) is a **separate, optional** thing — mention it, keep it distinct from unit tests, and do not run it as part of the unit suite.

### Table-driven `node:test`

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parse } from '../src/parse.js';

test('parse', async (t) => {
  const cases = [
    { name: 'empty input', in: '', want: [], throws: true },
    { name: 'only separator', in: ',', want: [], throws: true },
    { name: 'trailing separator', in: 'a,', want: ['a'], throws: false },
  ];
  for (const c of cases) {
    await t.test(c.name, () => {
      if (c.throws) {
        assert.throws(() => parse(c.in));
        return;
      }
      assert.deepEqual(parse(c.in), c.want);
    });
  }
});
```

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/modern-node.md` | Reference | Curated modern Node idioms — ESM + top-level await, `node:` builtins, `node:test` + `node:assert/strict` table-driven template, structuredClone deep-merge, AbortController/signal, console-logger helper |
| `./references/project-templates.md` | Template | Copy-paste scaffolding — minimal `package.json`, `.node-version`, deep-merge config loader + `state.json` writer, thin `bin/` launcher |
