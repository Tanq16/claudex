# Review Domain: Node

**Applies to:** Node Web Only, the only Node project type defined. A Node project that does not match that shape is reported as out of scope rather than reviewed against an invented taxonomy.

**Skills to load, in full, before running any check below:**
- `[SKILLS_DIR]/node-project-layout/SKILL.md`
- `[SKILLS_DIR]/node-idioms/SKILL.md`
- `[SKILLS_DIR]/node-config-state/SKILL.md`
- `[SKILLS_DIR]/node-http-ws-server/SKILL.md`
- `[SKILLS_DIR]/node-auth/SKILL.md` (Category 5 only)
- `[SKILLS_DIR]/node-frontend/SKILL.md`
- `[SKILLS_DIR]/unit-testing/SKILL.md`

The expected pattern for every check lives in those skills. This file states what to look at and how to look at it.

---

## Pre-check

**Auth:** grep `src/` for `scrypt`, session handling, or a users file. Skip Category 5 when absent.

---

## Category 1: Layout and Module System

| Check | How to verify |
|---|---|
| Canonical layout | Glob the project root against the directories the skill lists |
| Launcher is thin | Read the `bin/` entry and check what it does beyond resolving config and handing off |
| No CLI framework | Read `package.json` dependencies and grep `bin/` and `src/` for argument-parsing libraries |
| Pure ESM | Read `package.json` for the module type; grep `src/`, `bin/`, and `test/` for `require(`, `module.exports`, `__dirname`, and `__filename` |
| Node version pinned | Read `package.json` engines and `.node-version`, and check they agree |
| Module exports | Read each `src/` module for default-export grab-bags |
| Builtin imports | Grep imports for builtin modules without the `node:` prefix |
| Modern idioms | Grep for `JSON.parse(JSON.stringify(`, for IIFE wrappers around top-level async setup, and for the long `import.meta.url` directory dance |

## Category 2: Dependencies and Logging

| Check | How to verify |
|---|---|
| Every dependency is justified | Read `package.json` dependencies; for each, work out whether a `node:` builtin covers it |
| No logging framework | Read `package.json` for a logging library |
| No test framework | Read `package.json` for jest, mocha, vitest, ava, or a runner config file |
| No frontend framework | Read `package.json` for React, Vue, Svelte, and for a bundler treated as required |
| Nothing unused | Cross-reference dependencies against actual imports |
| Versions current | Read `package.json`, check each direct dependency against its latest stable release |
| Log form | Grep `src/` and `bin/` for `console.` calls and read what precedes the message |
| No color in logs | Grep for color libraries and for raw ANSI escapes in log calls |

## Category 3: Config and State

| Check | How to verify |
|---|---|
| Defaults are the source of truth | Read the config loader for its defaults object |
| Merge behavior | Read the merge function and check what it does with nested objects and with arrays |
| Missing-file handling | Read the loader's catch and check which error codes it swallows |
| Example file shipped | Glob for `config.example.json` and check `config.json` itself is not committed |
| Config loaded once | Grep `src/` for repeated reads of the config file or of `process.env` below the entry layer |
| Session secret not persisted | Grep for the secret's generation and trace whether it ever reaches a write |
| State file mode | Read the mode argument on the state write |
| State write is atomic | Read the write path for a temp file and a rename |

## Category 4: Server

| Check | How to verify |
|---|---|
| One listener | Grep for `createServer` and for `WebSocketServer` construction; read how the two are joined |
| Upgrade handling | Read the upgrade handler for its path check and what it does with a non-matching path |
| Static serving | Read the static handler |
| Traversal guard | Read the path resolution and the containment check, including whether the root ends in a separator |
| SPA fallback | Read what an unknown non-API path returns, and what an unknown API path returns |
| MIME handling | Read the content-type resolution |
| Broadcast | Read the broadcast helper for where serialization happens and how closing sockets are treated |
| Message dispatch | Read the message handler for its parse failure path |
| Shutdown | Read `start` for signal handlers, socket closing, the force-exit timer, and whether that timer is unreferenced |
| Error boundaries | Grep `src/` feature modules for `process.exit`; read the route handler for its catch and what it sends |

## Category 5: Auth

| Check | How to verify |
|---|---|
| Hashing | Grep `src/` for the KDF used and for any other hashing library |
| Salt handling | Read the hash function for where the salt comes from and how it is stored |
| Comparison | Read the verify function for how the two values are compared, and for a length check before it |
| Session form | Read the session creation and validation |
| Validation order | Read the validation function for whether the signature is checked before the payload is parsed |
| Secret lifetime | Trace the signing secret from generation to every use |
| Cookie attributes | Grep for `Set-Cookie` construction and read the attributes present |
| Users file lifecycle | Read the login path for where the users file is read |
| Timing on unknown users | Read the unknown-username branch |

## Category 6: Frontend

| Check | How to verify |
|---|---|
| Framework-free | Read `public/js/` and grep `package.json` for frontend frameworks |
| Assets not committed | Run `git ls-files` over the vendored directories and read `.gitignore` |
| Nothing fetched at run time | Grep `public/` HTML and CSS for external origins |
| Fonts present | Glob the fonts directory against the three the skill names |
| Font format | Glob the fonts directory for anything that is not woff2 |
| Font wiring | Read the `@font-face` stylesheets and the HTML head links |
| Palette declared once | Grep the CSS for hardcoded hex values in component styles alongside the variable declarations |
| WebSocket client | Read the client module for reconnect backoff, its dispatch, its send queue, and how it distinguishes a deliberate close |

## Category 7: Tests

| Check | How to verify |
|---|---|
| Runner and assertions | Read the test files for their imports |
| Scenario-driven | Read the case tables and assess whether the cases are edge cases |
| Units stay units | Grep unit tests for server startup and socket construction |
| End-to-end separated | Glob for an end-to-end script and check it is not picked up by the unit run |

---

## Output Format

```
## Domain: Node

### [PASS] Category Name

All checks passed.

### [ISSUES] Category Name

1. **[Issue title]** [severity] (skill-name: section)
   - **Where:** file:line
   - **Current:** [what the code does now]
   - **Expected:** [what the cited skill section says]
   - **Fix:** [the specific action]

### [SKIP] Category Name

Not applicable: [reason].
```

End with exactly:

```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```
