# Review Domain: Node

**Applies to:** Node Web Only **Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../node-foundations/SKILL.md`
- `../../node-backend/SKILL.md`
- `../../node-frontend/SKILL.md`

> There is only ONE Node project type today: **Node Web Only** (a process that serves — HTTP + WebSocket backend serving an embedded/vendored single-page frontend). Node CLI Only, CLI + Web, and Library types are out of scope; if the project does not match the Web Only shape, report that and skip Node checks.

---

## Category 16: Node Foundations (node-foundations)

**Project layout & ESM checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Canonical layout | `bin/` (thin launcher), `src/` (backend), `public/` (frontend), `test/`, `config.example.json`, `Makefile`, `.github/workflows/` | Glob project root and confirm each directory/file |
| Launcher is thin | `bin/` entry (e.g. `app.js`) only reads the config path and hands off to `src/` — no server/routing logic | Read the `bin/` entry file |
| Pure ESM | `package.json` has `"type": "module"`; source uses `import`/`export`, not `require`/`module.exports` | Read `package.json`; grep `src/`, `bin/`, `test/` for `require(` or `module.exports` (flag if present) |
| Engines pinned | `package.json` has `"engines": { "node": ">=24" }` | Read `package.json` |
| Node version pinned | `.node-version` exists and pins Node >=24 | Read `.node-version` |

**Dependencies checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Builtins first | Uses `node:` builtins (`node:http`, `node:crypto`, `node:fs`, `node:path`, `node:url`, `node:test`) before reaching for a dependency | Grep imports for `node:` prefixes; check whether any dep duplicates a builtin capability |
| Minimal dep list | Every `dependencies` entry fills a genuine need a builtin cannot reasonably cover (KISS/YAGNI/DRY) | Read `package.json` dependencies; for each, confirm no simpler builtin alternative exists |
| No logging framework | No logging library (winston, pino, bunyan, etc.) in dependencies | Read `package.json`, flag any logging dependency |
| Deps current | Direct dependencies at latest stable versions | Read `package.json`, compare against latest stable |
| Deps actually used | Every dependency is imported somewhere in source | Cross-reference `package.json` deps against actual imports |

**Logging checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| console.* with prefixes | Logging via `console.*` with manual level prefixes (`INFO`/`ERROR`/`DEBUG`), timestamped and sequential | Grep source for `console.log`/`console.error`; verify level prefix + timestamp |
| No color in logs | Log output is plain — no ANSI color codes or color libraries | Grep for color libraries (chalk, kleur, etc.) and raw ANSI escapes in log calls |

**Testing checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| node:test runner | Tests use the built-in runner: `import { test } from 'node:test'`, `import assert from 'node:assert/strict'`, run via `node --test` | Read `test/` files; grep for `node:test` and `node:assert/strict` |
| No test framework | No jest/mocha/vitest/ava in dependencies or config | Read `package.json`, flag any third-party test framework |
| Scenario/edge-case driven | Tests cover boundary/empty/malformed inputs, table-driven where it compacts cases; not coverage-chasing | Read test files, assess case selection |
| Units don't stand up servers | Unit tests do not boot servers or open network connections | Grep unit tests for server/socket setup (belongs in e2e, not units) |
| e2e is separate & optional | A live end-to-end script (e.g. `test/e2e.mjs` booting the server over http+ws) is kept distinct from `node:test` units — its absence is not a defect | Glob for an e2e script; confirm it is separate from unit suites |

**Principles checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Comments — load-bearing why only | Default to zero comments; a comment earns its place only when its *why* is load-bearing (non-obvious constraint/trade-off). No what-narration | For each comment, ask whether removing it would let a competent reader misread intent/trade-off/constraint; flag any that would not |
| KISS / YAGNI / DRY | No speculative abstractions, no empty modules, no unused imports; common operations factored once | Read source, flag over-engineering and duplication |

---

## Category 17: Node Backend (node-backend)

**Server checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| node:http + ws server | HTTP server on `node:http`; realtime via the `ws` library | Read `src/` server code; grep for `node:http` and `ws` imports |
| No CLI framework | No commander/yargs/oclif — a bare serve entry, optionally a thin hand-rolled argv read for the config path | Read `bin/` and `src/` entry; grep `package.json` for CLI frameworks (flag if present) |
| Serves the frontend | Backend serves the vendored `public/` single-page frontend | Read static-serving code in `src/` |

**Config & state checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Deep-merged config | A user `config.json` is deep-merged over built-in defaults; env/flags may override | Read config-loading code in `src/` |
| Ephemeral session secret | Session secret is regenerated on boot and NOT persisted to disk | Grep for secret generation; confirm it is not written to a state file |
| Durable state file mode | Only durable state (e.g. a mode flag) is written to `state.json` at file mode `0600` | Grep for `state.json` writes; verify `0o600` mode |

**Auth checks (only if auth is present):**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| scrypt hashing | Password hashing via `node:crypto` scrypt (`crypto.scrypt`/`scryptSync`) | Grep for `scrypt` in `src/`; flag other hashing (bcrypt lib, plain sha) |
| Session cookies | Session cookies are `HttpOnly` and `SameSite=Lax` | Grep for `Set-Cookie`; verify `HttpOnly` and `SameSite=Lax` attributes |
| Users re-read on login | Users live in a JSON file that is re-read on each login (not cached at boot) | Read the login path; confirm the user file is read per-login |

**Native addon checks (only if a native addon is present):**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Built from source per platform | Native addon (e.g. `node-pty`) compiled from source per platform (`npm_config_build_from_source=true`, node-gyp with a managed Python) | Read Makefile/CI build steps for the addon |
| Build verified before release | A `make verify` (or equivalent) target spawns the addon to confirm the compiled `.node` loads | Read Makefile for a verify target that exercises the addon |

---

## Category 18: Node Frontend (node-frontend)

**Vanilla SPA checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Framework-free SPA | Vanilla JS single-page app — no React/Vue/Svelte, no mandatory bundler | Read `public/js/`; grep `package.json` for frontend frameworks (flag if present) |
| Assets vendored | Frontend assets vendored from `node_modules` into `public/` (not loaded from a CDN at runtime) | Read `public/`; check the Makefile vendor step; grep HTML for external CDN `src`/`href` |
| WebSocket client | Realtime handled by a WebSocket client in the frontend | Grep `public/js/` for `new WebSocket(` |

**Catppuccin theme checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Catppuccin Mocha | Theme uses the Catppuccin Mocha palette (same palette the Go frontend uses) | Read CSS for Catppuccin Mocha color values/variables |

**Shared vendored font checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| woff2, self-hosted | Fonts vendored as **woff2 only** under `public/fonts/`, self-hosted — no Google Fonts CDN at runtime | Glob `public/fonts/*.woff2`; grep HTML/CSS for `fonts.googleapis.com` / `fonts.gstatic.com` (flag if present) |
| Inter (body/UI) | Inter vendored with a `public/css/inter.css` `@font-face`; body/UI text uses `font-family: 'Inter'` | Read `public/css/inter.css` and the HTML `<head>` link; grep CSS for `font-family: 'Inter'` |
| JetBrains Mono Nerd Font (code) | JetBrains Mono **Nerd Font** variant (with glyphs) vendored with a `public/css/jetbrains-mono.css` `@font-face`; code/mono uses `font-family: 'JetBrains Mono'` | Read `public/css/jetbrains-mono.css` and the HTML `<head>` link; grep CSS for `font-family: 'JetBrains Mono'` |
| Font links in head | Both font-face stylesheet `<link>` tags live in the HTML `<head>` | Read the HTML `<head>` section |

---

## Output Format

Report findings in this exact format:

```
## Domain: Node

### [PASS] Category Name (source-skill)

All checks passed.

### [ISSUES] Category Name (source-skill)

1. **[Issue title]** (source-skill: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (source-skill)

Not applicable to this project type.
```

End your response with exactly:
```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

---

## Out of Scope (Hard Boundary)

Do NOT flag any of the following — they are not defined in any loaded skill:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | No eslint, no prettier, inconsistent formatting |
| Pre-commit | No pre-commit hooks, no husky |
| Code Quality CI | No lint/format CI steps |
| Documentation beyond README | No jsdoc, no changelogs, no contributing guide |
| Docker Compose | No docker-compose for development |
| Database | No migrations, no schema files |
| Dependency tooling | No dependabot, no renovate |
| Security scanning | No SAST, no container scanning |
| Code style opinions | Naming conventions not in skills, personal preferences |

**Rule:** If you cannot cite a specific section in a loaded skill for a finding, do not report it.
