---
name: node-backend
description: Use when implementing the backend of a Node Web Only server - covers src/ module architecture, node:http + ws server, routing, serving a vendored public/ SPA, error handling, JSON-file storage, scrypt/cookie authentication, native-addon release implications, and config loading
user-invocable: false
---

# Node Backend

**Architecture and implementation patterns for the backend of a Node Web Only server.**

## When to Use

Use this skill when:
- Structuring the `src/` backend of a Node Web Only project
- Building the `node:http` + `ws` server, routing, and static serving
- Handling errors across ESM modules
- Designing JSON-file storage/persistence
- Adding password + session authentication
- Deciding the release path when a native addon is involved

**Requires:** `node-foundations` for project layout and principles.

**Web Only constraint:** Node Web Only is the *only* Node project type today — a process that is invoked and serves (HTTP + WebSocket) a vendored single-page frontend. There is no CLI Only, CLI + Web, or Library Node type yet; those are future/out-of-scope. The backend uses builtins first (`node:http`, `node:crypto`, `node:fs`, `node:path`, `node:url`) and adds a dependency only when a builtin genuinely cannot cover the need. Logging is `console.*` with manual level prefixes (`INFO`/`ERROR`/`DEBUG`), timestamped and sequential — no logging-framework dependency, no color. This mirrors the Go Web Only `log`-package discipline.

## Package/Module Architecture

Backend code lives under `src/`, organized by feature — not by technical layer:

```
src/
├── server.js       # node:http server, upgrade handler, graceful shutdown
├── router.js       # request routing + static serving from public/
├── auth.js         # scrypt hashing, session cookies, users.json
├── state.js        # state.json read/atomic-write
├── config.js       # defaults + config.json deep-merge
└── ws.js           # websocket message dispatch + broadcast
```

`bin/app.js` is a thin launcher: it reads the config path from argv, calls `loadConfig`, and hands off to `start(config)` in `src/`. Keep entry-point concerns (argv, process signals, `process.exit`) in `bin/` and `src/server.js`; the feature modules stay pure.

### ESM module boundaries

Pure ESM (`"type": "module"` in `package.json`): `import`/`export` only, no `require`. Each module exports the functions its callers need and nothing else — no default-export grab-bags. Import builtins with the `node:` prefix (`import { createServer } from 'node:http'`). Top-level `await` is available in entry modules; use it for one-time async setup (loading users, reading state) rather than wrapping the whole file in an IIFE.

## HTTP + WebSocket Server

One `node:http` server handles both HTTP and the WebSocket upgrade — do not stand up a second listener. Create the server with a request handler that routes API paths and otherwise serves the vendored `public/` SPA; attach a `ws` `WebSocketServer({ noServer: true })` and complete the handshake yourself in the server's `upgrade` event so you can authenticate before accepting the socket.

```js
const server = createServer((req, res) => route(req, res, config));
const wss = new WebSocketServer({ noServer: true });

server.on('upgrade', (req, socket, head) => {
    if (new URL(req.url, 'http://localhost').pathname !== '/ws') {
        socket.destroy();
        return;
    }
    wss.handleUpgrade(req, socket, head, (ws) => wss.emit('connection', ws, req));
});
```

Static serving reads files from `public/` (the frontend `node-frontend` vendors), guards against path traversal, and falls back to `index.html` for unknown non-API paths so client-side routing works. Graceful shutdown lives in the launcher/`start`: close open sockets, `server.close()`, and force-exit after a bounded timeout. `node-backend` owns the canonical server — the full routing, static-serving, upgrade, broadcast, and shutdown code lives in `./references/http-ws-server.md`; `node-frontend` references that file rather than redefining it.

## Error Handling

Feature modules **return or throw** — they never call `process.exit`. `process.exit` and `process.on('SIGTERM', ...)` belong to the entry layer (`bin/app.js`, `src/server.js` `start`), the Node analog of Go's "context and logging at boundaries."

- Task-style helpers (storage, hashing, parsing) throw or return as-is — no logging, no wrapping for its own sake. Let the error carry its `code` (`ENOENT`, etc.) so callers can branch on it.
- The request handler is the boundary: wrap the route body in `try/catch`, log with `console.error('... ERROR ...')`, and send a generic 500 — never leak stack traces or internal messages to the client.
- An uncaught throw inside an async request handler will not crash a well-formed server if you catch at the route boundary; do not rely on `process.on('uncaughtException')` as normal control flow.

## Storage Pattern

Most Node Web Only projects are JSON-file backed — no database. Two distinct files with different lifecycles:

- **`users.json`** — credentials, **re-read on each login** (not cached at boot), so an operator can edit users without restarting. Read-only from the server's perspective.
- **`state.json`** — the only durable server-written state (e.g. a mode flag). Written at mode `0600`, and written **atomically** via write-to-temp + `rename`, because `rename` is atomic on POSIX and prevents a crash mid-write from leaving a truncated JSON file.

```js
export async function writeState(path, state) {
    const tmp = `${path}.${process.pid}.tmp`;
    await writeFile(tmp, JSON.stringify(state, null, 2), { mode: 0o600 });
    await rename(tmp, path); // atomic replace; a crash leaves the old file intact
}
```

Full read/write helpers are in `./references/auth-patterns.md`.

## Authentication

Password + session-cookie auth, all from `node:crypto` — no auth framework.

- **Password hashing:** `scrypt` with a per-user random salt; store `salt:derivedKey` (hex). Verify with `timingSafeEqual`. On an unknown username, still run a verify against a dummy hash so response time does not reveal whether the account exists.
- **Sessions:** a signed token, not server-side session storage. Sign `base64url(payload)` with HMAC-SHA256 using an **ephemeral secret** (`randomBytes(32)`) generated on boot and never persisted — so every restart invalidates all sessions, which is the intended trade-off for a single-process app. Validate by recomputing the HMAC and comparing with `timingSafeEqual`, then checking expiry.
- **Cookies:** `HttpOnly` (no JS access), `SameSite=Lax` (CSRF mitigation for top-level navigations), `Path=/`, and a bounded `Max-Age`. Set `Secure` too when served behind TLS.

Complete hash/verify, session, and cookie helpers are in `./references/auth-patterns.md`.

## The native-addon note

A pure-JS Web Only app can ship as a single self-contained binary (`bun build --compile` or Node SEA — see `project-ci-cd`). The moment the backend needs a **C++ N-API addon** (e.g. `node-pty` for a real PTY, `sharp`, `better-sqlite3`), that single-binary path breaks: the addon is a platform-specific compiled `.node` file that the binary embedders cannot bundle and load the way they bundle JS. The release path then becomes the **runtime-bundled tarball**: a per-platform `.tar.gz` carrying the Node runtime + the compiled `.node` (+ any helper such as `spawn-helper`) + vendored assets, with a launcher that injects `--config`. Detailed toolchain, the `debian:*-slim` (glibc, never Alpine/musl) Docker base, and the `make verify` addon-smoke-test all live in `project-ci-cd`. Reach for a native addon only when a builtin or pure-JS dependency genuinely cannot do the job — it is a real cost to the release story.

## Config

Built-in defaults, deep-merged with an optional user `config.json`; env/flags may override on top. Load once at boot and pass the resulting object into `start`.

```js
const defaults = {
    host: '127.0.0.1',
    port: 8080,
    usersFile: 'users.json',
    stateFile: 'state.json',
};

export function loadConfig(path) {
    const overrides = path ? JSON.parse(readFileSync(path, 'utf8')) : {};
    return deepMerge(defaults, overrides);
}
```

The ephemeral session secret is **not** config — it is regenerated on every boot and never written to disk. Only durable state belongs in `state.json`; secrets and derived runtime values do not.

## References

| File | Purpose |
|------|---------|
| `./references/http-ws-server.md` | Canonical `node:http` + `ws` server — routing, static serving from `public/`, upgrade handler, broadcast helper, graceful shutdown (referenced by `node-frontend`) |
| `./references/auth-patterns.md` | scrypt hash/verify, signed-cookie sessions with an ephemeral HMAC secret, `users.json` + `state.json` atomic persistence |
