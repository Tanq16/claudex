# Node Web Only — Project Templates

Copy-paste scaffolding for a Node Web Only project. Adjust names to the project; keep the ESM + `node:` builtin + no-framework discipline intact.

---

## `package.json`

Minimal, pure ESM, Node 24 baseline. Scripts run real Node — `node --test` for tests, `--watch` for dev.

```json
{
  "name": "example",
  "version": "0.1.0",
  "type": "module",
  "engines": { "node": ">=24" },
  "bin": { "example": "bin/app.js" },
  "scripts": {
    "start": "node bin/app.js",
    "dev": "node --watch bin/app.js",
    "test": "node --test",
    "test:watch": "node --test --watch"
  },
  "dependencies": {
    "ws": "^8.18.0"
  }
}
```

Add a dependency only for a gap a `node:` builtin cannot cover (see node-foundations → Dependency Selection). Keep the list tiny.

## `.node-version`

Pins the Node version for dev, CI, and tooling (`fnm`/`nvm`/`node-version` readers).

```
24
```

## Config loader — deep-merge `config.json` over defaults

Built-in defaults are the source of truth for shape; a user `config.json` is deep-merged over them; env/flags may override on top. Missing file falls back to defaults.

```js
import { readFile } from 'node:fs/promises';

const DEFAULTS = {
  port: 8080,
  host: '127.0.0.1',
  auth: { enabled: false, users: 'users.json' },
};

const isObj = (v) => v !== null && typeof v === 'object' && !Array.isArray(v);

function deepMerge(base, override) {
  const out = structuredClone(base);
  for (const [key, value] of Object.entries(override)) {
    out[key] = isObj(value) && isObj(out[key]) ? deepMerge(out[key], value) : value;
  }
  return out;
}

export async function loadConfig(path) {
  let user = {};
  if (path) {
    try {
      user = JSON.parse(await readFile(path, 'utf8'));
    } catch (err) {
      if (err.code !== 'ENOENT') throw err;
    }
  }
  const merged = deepMerge(DEFAULTS, user);
  if (process.env.PORT) merged.port = Number(process.env.PORT);
  return merged;
}
```

## Ephemeral secret + durable `state.json`

The session secret is regenerated every boot and never persisted — a restart invalidates sessions by design. Only durable facts are written to `state.json`, at mode `0600`.

```js
import { randomBytes } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';

export function newSessionSecret() {
  return randomBytes(32).toString('hex');
}

export async function loadState(path) {
  try {
    return JSON.parse(await readFile(path, 'utf8'));
  } catch (err) {
    if (err.code === 'ENOENT') return { mode: 'default' };
    throw err;
  }
}

export async function saveState(path, state) {
  await writeFile(path, JSON.stringify(state, null, 2), { mode: 0o600 });
}
```

## Thin `bin/` launcher

No CLI framework. Reads an optional `--config <path>` from argv, loads config, starts the server in `src/`. Top-level await, ESM.

```js
#!/usr/bin/env node
import { loadConfig } from '../src/config.js';
import { startServer } from '../src/server.js';

function configPath(argv) {
  const i = argv.indexOf('--config');
  return i !== -1 ? argv[i + 1] : undefined;
}

const config = await loadConfig(configPath(process.argv.slice(2)));
await startServer(config);
```

The argv read stays a hand-rolled few lines — do not pull in commander/yargs. The launcher's only job is to resolve the config path and hand off to `src/`.
