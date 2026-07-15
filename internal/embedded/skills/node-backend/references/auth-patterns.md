# Authentication & Storage Patterns

Password + session-cookie authentication and JSON-file persistence for Node Web Only servers, built entirely on `node:crypto` and `node:fs` — no auth framework, no database. All code below is comment-free; the load-bearing "why" is in the prose.

## src/auth.js — password hashing

`scrypt` is the memory-hard KDF exposed by `node:crypto`. Each password gets a fresh 16-byte salt; the stored value is `salt:derivedKey` in hex. Verification derives with the stored salt and compares with `timingSafeEqual` so comparison time does not leak how many leading bytes matched.

```js
import { randomBytes, scrypt, timingSafeEqual } from 'node:crypto';
import { promisify } from 'node:util';

const scryptAsync = promisify(scrypt);
const KEY_LEN = 64;

export async function hashPassword(password) {
    const salt = randomBytes(16);
    const derived = await scryptAsync(password, salt, KEY_LEN);
    return `${salt.toString('hex')}:${derived.toString('hex')}`;
}

export async function verifyPassword(password, stored) {
    const [saltHex, keyHex] = stored.split(':');
    if (!saltHex || !keyHex) return false;
    const expected = Buffer.from(keyHex, 'hex');
    const derived = await scryptAsync(password, Buffer.from(saltHex, 'hex'), expected.length);
    return expected.length === derived.length && timingSafeEqual(expected, derived);
}
```

## src/auth.js — sessions with an ephemeral secret

The session secret is generated once at module load with `randomBytes(32)` and never written to disk. A restart mints a new secret and therefore invalidates every outstanding session — the accepted trade-off for a single-process app that keeps no session store. A token is `base64url(payload).hmacSignature`; validation recomputes the HMAC, compares with `timingSafeEqual`, then enforces expiry.

```js
import { randomBytes, createHmac, timingSafeEqual } from 'node:crypto';

const SESSION_SECRET = randomBytes(32);
const SESSION_TTL_MS = 12 * 60 * 60 * 1000;

function sign(body) {
    return createHmac('sha256', SESSION_SECRET).update(body).digest('base64url');
}

export function createSession(username) {
    const payload = JSON.stringify({ username, exp: Date.now() + SESSION_TTL_MS });
    const body = Buffer.from(payload).toString('base64url');
    return `${body}.${sign(body)}`;
}

export function validateSession(token) {
    if (!token) return null;
    const [body, sig] = token.split('.');
    if (!body || !sig) return null;
    const expected = sign(body);
    const given = Buffer.from(sig);
    if (given.length !== expected.length || !timingSafeEqual(given, Buffer.from(expected))) {
        return null;
    }
    let claims;
    try {
        claims = JSON.parse(Buffer.from(body, 'base64url').toString());
    } catch {
        return null;
    }
    return claims.exp > Date.now() ? claims.username : null;
}
```

## src/auth.js — cookies

`HttpOnly` keeps the token out of `document.cookie`; `SameSite=Lax` blocks it from cross-site subrequests while still allowing top-level navigation. Add `Secure` when the server sits behind TLS. Clearing sets `Max-Age=0`.

```js
const MAX_AGE = 12 * 60 * 60;

export function sessionCookie(token) {
    return `session=${token}; HttpOnly; SameSite=Lax; Path=/; Max-Age=${MAX_AGE}`;
}

export function clearedCookie() {
    return `session=; HttpOnly; SameSite=Lax; Path=/; Max-Age=0`;
}

export function readCookie(req, name) {
    const header = req.headers.cookie;
    if (!header) return null;
    for (const part of header.split(';')) {
        const eq = part.indexOf('=');
        if (eq === -1) continue;
        if (part.slice(0, eq).trim() === name) return part.slice(eq + 1).trim();
    }
    return null;
}
```

## src/auth.js — users.json (read on login)

`users.json` is re-read on every login attempt, never cached at boot, so an operator can add or remove users without a restart. On an unknown username the code still runs a `verifyPassword` against a fixed dummy hash: without it, a missing account would return noticeably faster than a wrong password and expose which usernames exist.

```js
import { readFile } from 'node:fs/promises';

const DUMMY_HASH =
    '0'.repeat(32) + ':' + '0'.repeat(128);

export async function authenticate(usersFile, username, password) {
    const users = JSON.parse(await readFile(usersFile, 'utf8'));
    const record = users[username];
    if (!record) {
        await verifyPassword(password, DUMMY_HASH);
        return false;
    }
    return verifyPassword(password, record.password);
}
```

`users.json` shape:

```json
{
    "admin": { "password": "a1b2...:9f8e..." }
}
```

Generate the stored value with `hashPassword` (e.g. from a one-off script or an admin endpoint); never store plaintext.

## src/state.js — durable state with atomic writes

`state.json` holds the only server-written durable state (a mode flag, a counter — small, non-secret). Writes go to a temp file first and then `rename` over the target: `rename` is atomic on POSIX, so a crash mid-write leaves either the old file or the new one, never a half-written JSON. Mode `0600` keeps it owner-only.

```js
import { readFile, writeFile, rename } from 'node:fs/promises';

export async function readState(path, fallback) {
    try {
        return JSON.parse(await readFile(path, 'utf8'));
    } catch (err) {
        if (err.code === 'ENOENT') return fallback;
        throw err;
    }
}

export async function writeState(path, state) {
    const tmp = `${path}.${process.pid}.tmp`;
    await writeFile(tmp, JSON.stringify(state, null, 2), { mode: 0o600 });
    await rename(tmp, path);
}
```

## Wiring into the router

The login handler authenticates, sets the cookie, and returns; protected handlers read and validate the cookie up front. Keep these at the request boundary — the helpers above throw or return, and the handler decides the HTTP response.

```js
import { authenticate, createSession, sessionCookie, readCookie, validateSession } from './auth.js';

export async function handleLogin(req, res, body, config) {
    const ok = await authenticate(config.usersFile, body.username, body.password);
    if (!ok) {
        res.writeHead(401).end();
        return;
    }
    res.writeHead(204, { 'Set-Cookie': sessionCookie(createSession(body.username)) }).end();
}

export function requireSession(req) {
    return validateSession(readCookie(req, 'session'));
}
```
