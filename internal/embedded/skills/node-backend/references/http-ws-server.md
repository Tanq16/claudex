# HTTP + WebSocket Server Template

Canonical `node:http` + `ws` server for Node Web Only projects. This is the single source of truth for the request-routing, static-serving, upgrade-handling, and graceful-shutdown boilerplate — `node-frontend` references this file instead of re-defining it.

One `node:http` server handles both HTTP and the WebSocket upgrade; `ws` runs in `noServer` mode so the handshake is completed manually (letting you authenticate before accepting the socket). Builtins only apart from `ws`.

## src/server.js

```js
import { createServer } from 'node:http';
import { WebSocketServer } from 'ws';
import { route } from './router.js';
import { handleMessage } from './ws.js';

function log(level, msg) {
    console.log(`${new Date().toISOString()} ${level} ${msg}`);
}

export function createApp(config) {
    const server = createServer((req, res) => route(req, res, config));
    const wss = new WebSocketServer({ noServer: true });

    server.on('upgrade', (req, socket, head) => {
        const { pathname } = new URL(req.url, 'http://localhost');
        if (pathname !== '/ws') {
            socket.destroy();
            return;
        }
        wss.handleUpgrade(req, socket, head, (ws) => wss.emit('connection', ws, req));
    });

    wss.on('connection', (ws) => {
        ws.on('message', (data) => handleMessage(wss, ws, data));
        ws.on('error', (err) => log('ERROR', `ws: ${err.message}`));
    });

    return { server, wss };
}

export function start(config) {
    const { server, wss } = createApp(config);

    server.listen(config.port, config.host, () => {
        log('INFO', `listening on ${config.host}:${config.port}`);
    });

    let closing = false;
    const shutdown = (signal) => {
        if (closing) return;
        closing = true;
        log('INFO', `${signal} received, shutting down`);
        for (const client of wss.clients) client.close(1001, 'server shutdown');
        server.close(() => process.exit(0));
        setTimeout(() => process.exit(1), 10_000).unref();
    };
    process.on('SIGTERM', () => shutdown('SIGTERM'));
    process.on('SIGINT', () => shutdown('SIGINT'));

    return { server, wss };
}
```

The force-exit timer is `.unref()`'d so it never keeps the process alive on its own — it only fires if `server.close` is still waiting on a stuck connection after the grace period.

## src/router.js

`route` dispatches API paths and delegates everything else to static serving. `serveStatic` resolves the request path under `public/`, rejects any path that escapes that root (traversal guard), and falls back to `index.html` for unknown non-API paths so the SPA's client-side routing works on deep links.

```js
import { readFile } from 'node:fs/promises';
import { join, normalize, extname, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const PUBLIC_DIR = fileURLToPath(new URL('../public/', import.meta.url));

const MIME = {
    '.html': 'text/html; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
    '.svg': 'image/svg+xml',
    '.png': 'image/png',
    '.ico': 'image/x-icon',
    '.woff2': 'font/woff2',
};

export function route(req, res, config) {
    const url = new URL(req.url, 'http://localhost');
    try {
        if (url.pathname === '/api/health') {
            return sendJSON(res, 200, { status: 'ok' });
        }
        if (url.pathname.startsWith('/api/')) {
            return sendJSON(res, 404, { error: 'not found' });
        }
        return serveStatic(url.pathname, res);
    } catch (err) {
        console.error(`${new Date().toISOString()} ERROR ${req.method} ${url.pathname}: ${err.message}`);
        return sendJSON(res, 500, { error: 'internal error' });
    }
}

async function serveStatic(pathname, res) {
    const rel = pathname === '/' ? 'index.html' : pathname.slice(1);
    const target = normalize(join(PUBLIC_DIR, rel));
    if (target !== PUBLIC_DIR.slice(0, -1) && !target.startsWith(PUBLIC_DIR)) {
        res.writeHead(403).end();
        return;
    }
    try {
        const body = await readFile(target);
        res.writeHead(200, { 'Content-Type': MIME[extname(target)] ?? 'application/octet-stream' });
        res.end(body);
    } catch (err) {
        if (err.code === 'ENOENT' || err.code === 'EISDIR') {
            const index = await readFile(join(PUBLIC_DIR, 'index.html'));
            res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
            res.end(index);
            return;
        }
        throw err;
    }
}

function sendJSON(res, status, body) {
    const data = JSON.stringify(body);
    res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(data);
}
```

`PUBLIC_DIR` ends in a separator, so `startsWith(PUBLIC_DIR)` on the normalized absolute `target` is what blocks `../` escapes; the `sep` import is available if you need to compose sub-roots. Keep the `MIME` table explicit — do not pull in a `mime` dependency for a handful of asset types.

## src/ws.js

`handleMessage` parses one client frame, dispatches on a `type` field, and uses `broadcast` to fan a message out to every open peer. `broadcast` serializes once and skips sockets that are not in the `OPEN` state.

```js
export function broadcast(wss, message) {
    const data = JSON.stringify(message);
    for (const client of wss.clients) {
        if (client.readyState === client.OPEN) {
            client.send(data);
        }
    }
}

export function handleMessage(wss, ws, raw) {
    let msg;
    try {
        msg = JSON.parse(raw);
    } catch {
        return;
    }
    switch (msg.type) {
        case 'ping':
            ws.send(JSON.stringify({ type: 'pong' }));
            break;
        case 'broadcast':
            broadcast(wss, { type: 'message', payload: msg.payload });
            break;
        default:
            ws.send(JSON.stringify({ type: 'error', error: 'unknown type' }));
    }
}
```

## bin/app.js

The launcher is thin: read an optional `--config <path>`, load config, start the server. Argv parsing is a hand-rolled read — no CLI framework.

```js
import { loadConfig } from '../src/config.js';
import { start } from '../src/server.js';

function configPath(argv) {
    const i = argv.indexOf('--config');
    return i !== -1 && argv[i + 1] ? argv[i + 1] : null;
}

start(loadConfig(configPath(process.argv.slice(2))));
```
