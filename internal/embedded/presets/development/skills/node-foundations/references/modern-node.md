# Modern Node (target: Node >=24)

All Node in these projects targets **Node 24 or newer**, pinned via `.node-version` and declared as `"engines": { "node": ">=24" }`. Write to that baseline: pure ESM, `node:`-prefixed builtins, and the standard-library helpers Node already ships instead of dependencies. There is no transpiler and no bundler in dev/test — code runs on real Node.

---

## Pure ESM

`package.json` sets `"type": "module"`. Use `import`/`export` exclusively.

```js
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';

export function makeRouter(routes) {
  return (req, res) => { /* ... */ };
}
```

- No `require`, no `module.exports`, no `__dirname`/`__filename`.
- For a module's own directory, derive it from `import.meta`:

```js
import { fileURLToPath } from 'node:url';
import { dirname } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
```

- Node 24 also exposes `import.meta.dirname` and `import.meta.filename` directly:

```js
const here = import.meta.dirname;
```

## Top-level await

ESM modules support top-level `await` — use it for boot-time async work instead of an IIFE wrapper.

```js
import { loadConfig } from './config.js';
import { startServer } from './server.js';

const config = await loadConfig(process.argv);
await startServer(config);
```

## `node:`-prefixed builtins

Always import builtins with the `node:` prefix — it is unambiguous and cannot be shadowed by a same-named dependency.

| Need | Builtin |
|------|---------|
| HTTP server/client | `node:http`, `node:https` |
| Crypto (hashing, random) | `node:crypto` |
| Filesystem (promises) | `node:fs/promises`, `node:fs` |
| Paths | `node:path` |
| URL / file URLs | `node:url` |
| Tests | `node:test`, `node:assert/strict` |
| Streams | `node:stream`, `node:stream/promises` |
| Events | `node:events` |
| Subprocesses | `node:child_process` |

```js
import { scrypt, randomBytes, timingSafeEqual } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
```

## `node:test` + `node:assert/strict` (table-driven)

Use the built-in runner — no test-framework dependency. Run with `node --test` (`--watch` in dev). Prefer `node:assert/strict` so equality is strict by default. Enumerate edge cases in a table.

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { deepMerge } from '../src/config.js';

test('deepMerge', async (t) => {
  const cases = [
    {
      name: 'override scalar',
      base: { port: 8080, tls: false },
      over: { port: 9090 },
      want: { port: 9090, tls: false },
    },
    {
      name: 'nested merge keeps sibling keys',
      base: { auth: { enabled: true, users: 'users.json' } },
      over: { auth: { enabled: false } },
      want: { auth: { enabled: false, users: 'users.json' } },
    },
    {
      name: 'array replaces, not concatenates',
      base: { origins: ['a', 'b'] },
      over: { origins: ['c'] },
      want: { origins: ['c'] },
    },
  ];
  for (const c of cases) {
    await t.test(c.name, () => {
      assert.deepEqual(deepMerge(c.base, c.over), c.want);
    });
  }
});
```

Async setup/teardown and subtests:

```js
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';

test('handler', async (t) => {
  let store;
  before(() => { store = new Map(); });
  after(() => { store.clear(); });

  await t.test('rejects unknown id', () => {
    assert.throws(() => lookup(store, 'nope'), /not found/);
  });
});
```

## `structuredClone` and deep merge

Node ships `structuredClone` globally — use it for deep copies instead of `JSON.parse(JSON.stringify(...))` or a clone dependency.

```js
const snapshot = structuredClone(config);
```

A recursive deep-merge for config layering (plain objects merge, everything else — including arrays — replaces):

```js
const isObj = (v) => v !== null && typeof v === 'object' && !Array.isArray(v);

export function deepMerge(base, override) {
  const out = structuredClone(base);
  for (const [key, value] of Object.entries(override)) {
    out[key] = isObj(value) && isObj(out[key]) ? deepMerge(out[key], value) : value;
  }
  return out;
}
```

## AbortController and signals

Use `AbortController`/`AbortSignal` to cancel async work, time out fetches, and unwind server lifecycles — the standard cancellation primitive.

```js
const ac = new AbortController();
const timer = setTimeout(() => ac.abort(new Error('timeout')), 5000);
try {
  const res = await fetch(url, { signal: ac.signal });
  return await res.json();
} finally {
  clearTimeout(timer);
}
```

`AbortSignal.timeout(ms)` wraps the common timeout case, and process signals can drive graceful shutdown:

```js
const res = await fetch(url, { signal: AbortSignal.timeout(5000) });

process.on('SIGTERM', () => server.close());
```

## Console logger with level prefixes

Node Web Only projects log through `console.*` with manual prefixes — no logging dependency. A minimal reusable helper:

```js
const ts = () => new Date().toISOString();

export const log = {
  info: (msg) => console.log(`${ts()} INFO ${msg}`),
  error: (msg) => console.error(`${ts()} ERROR ${msg}`),
  debug: (msg) => {
    if (process.env.DEBUG) console.debug(`${ts()} DEBUG ${msg}`);
  },
};
```

Timestamped, sequential, no color; errors go to stderr via `console.error`.
