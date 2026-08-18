---
name: node-frontend
description: The vanilla-JS single-page frontend served from public/ by a Node Web Only backend - HTML structure, self-hosted fonts, the Catppuccin Mocha theme, and the reconnecting WebSocket client. Use when building or changing that page, laying out public/, theming it, or wiring realtime updates. Triggers on public/index.html, public/js/, public/css/, public/fonts/, @font-face, new WebSocket, reconnect backoff, and Catppuccin. Not for React, Vue, or Svelte, and not for a bundler-driven build.
user-invocable: false
---

# Node Frontend

**One `index.html` under `public/`, plain ES modules, self-hosted fonts, and a WebSocket client that reconnects on its own.**

The backend serves this tree at the site root: `public/index.html` at `/`, `public/css/inter.css` at `/css/inter.css`, `public/fonts/*.woff2` at `/fonts/*.woff2`.

## Layout

```
public/
├── index.html
├── css/
│   ├── inter.css               # @font-face for Inter
│   ├── google-sans.css         # @font-face for Google Sans
│   ├── jetbrains-mono.css      # @font-face for JetBrains Mono Nerd Font Mono
│   └── app.css                 # optional, when inline styles outgrow the page
├── js/
│   ├── app.js                  # application logic
│   └── ws.js                   # reconnecting WebSocket client
├── fonts/                      # woff2 only
├── icons/
│   └── favicon.png
└── vendor/                     # third-party JS and CSS copied out of node_modules
```

Nothing under `fonts/` or `vendor/` is committed. A Makefile target produces them and the release workflow calls the same target, so the tree in git holds only what a person wrote and a downloaded asset never lands in a diff nobody reviews.

## Rules

No framework and no mandatory bundler. Plain ES modules via `<script type="module">`, the DOM API, `fetch`, and `WebSocket`. A build step exists only for the binary release path and is never required to develop or run the page, which keeps the page a file you can open rather than a target you have to compile.

One `index.html`. Views are shown and hidden client-side, and a second page is added only when something genuinely needs its own URL.

Shared logic lives in `js/app.js` and the socket client in `js/ws.js`. Splitting the transport out means a page that does not need realtime simply does not import it.

Custom CSS goes either in an inline `<style>` block or in `css/app.css`, whichever suits the size, and never in both. Scattering the same rules across two places is how a color ends up defined twice with different values.

The two or three `@font-face` stylesheets always stay as separate linked files under `css/`, since the asset step regenerates that directory wholesale.

## Fonts

| Font | Role | `font-family` | Source |
|---|---|---|---|
| Inter | body and UI text | `'Inter'` | Google Fonts |
| Google Sans | display headings and branding | `'Google Sans'` | Google Fonts |
| JetBrains Mono Nerd Font Mono | code and monospace, with glyphs | `'JetBrains Mono'` | the nerd-fonts release |

All three are downloaded whether or not a given page uses all three, so the asset step is identical across projects and a later design change needs no build change.

Fonts are woff2, never ttf. woff2 is roughly half the bytes for the same glyphs, and every browser targeted here supports it. Google Fonts already serves woff2; the Nerd Font ships as ttf and is compressed during the asset step.

Everything is self-hosted and served from the app's own origin. No runtime CDN, no Google Fonts fetch, no external `<script src>`. A page that fetches a font from a third party leaks every visitor's address and stops rendering correctly offline.

```css
/* public/css/inter.css */
@font-face {
    font-family: 'Inter';
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    src: url('/fonts/inter-400.woff2') format('woff2');
}

@font-face {
    font-family: 'Inter';
    font-style: normal;
    font-weight: 600;
    font-display: swap;
    src: url('/fonts/inter-600.woff2') format('woff2');
}
```

```css
/* public/css/jetbrains-mono.css */
@font-face {
    font-family: 'JetBrains Mono';
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    src: url('/fonts/JetBrainsMonoNerdFontMono-Regular.woff2') format('woff2');
}

@font-face {
    font-family: 'JetBrains Mono';
    font-style: normal;
    font-weight: 700;
    font-display: swap;
    src: url('/fonts/JetBrainsMonoNerdFontMono-Bold.woff2') format('woff2');
}
```

`public/css/google-sans.css` follows the same shape for the weights the design uses.

`font-display: swap` renders text in a fallback face immediately and swaps when the woff2 arrives, so a slow font never leaves the page blank.

## The Page

Catppuccin Mocha is the default for a new frontend in this style, declared once on `:root` and referenced as `var(--blue)` everywhere else. A project that already has a palette keeps it, since re-theming a working app is a design decision rather than a convention. Hardcoding a hex value in a component style is what makes a palette change a search-and-replace.

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APP_NAME</title>

    <link rel="icon" type="image/png" sizes="32x32" href="/icons/favicon.png">

    <link rel="stylesheet" href="/css/inter.css">
    <link rel="stylesheet" href="/css/google-sans.css">
    <link rel="stylesheet" href="/css/jetbrains-mono.css">

    <style>
        :root {
            --rosewater: #f5e0dc; --flamingo: #f2cdcd; --pink: #f5c2e7;
            --mauve: #cba6f7; --red: #f38ba8; --maroon: #eba0ac;
            --peach: #fab387; --yellow: #f9e2af; --green: #a6e3a1;
            --teal: #94e2d5; --sky: #89dceb; --sapphire: #74c7ec;
            --blue: #89b4fa; --lavender: #b4befe; --text: #cdd6f4;
            --subtext1: #bac2de; --subtext0: #a6adc8; --overlay2: #9399b2;
            --overlay1: #7f849c; --overlay0: #6c7086; --surface2: #585b70;
            --surface1: #45475a; --surface0: #313244; --base: #1e1e2e;
            --mantle: #181825; --crust: #11111b;
        }

        * { box-sizing: border-box; }

        body {
            margin: 0;
            min-height: 100vh;
            font-family: 'Inter', system-ui, sans-serif;
            background: var(--base);
            color: var(--text);
        }

        h1, h2, .brand { font-family: 'Google Sans', 'Inter', sans-serif; }
        code, pre, kbd, samp { font-family: 'JetBrains Mono', monospace; }

        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 1rem;
            background: var(--mantle);
            border-bottom: 1px solid var(--surface0);
        }
        header .brand { font-weight: 600; }
        header #status { font-size: 0.85rem; color: var(--overlay1); }
        header #status.online { color: var(--green); }
        header #status.offline { color: var(--red); }

        main { max-width: 60rem; margin: 0 auto; padding: 2rem 1rem; }

        pre {
            background: var(--mantle);
            border: 1px solid var(--surface0);
            border-radius: 0.5rem;
            padding: 1rem;
            overflow-x: auto;
        }
        code {
            color: var(--peach);
            background: var(--surface0);
            padding: 0.1rem 0.35rem;
            border-radius: 0.25rem;
        }
        pre code { color: var(--text); background: none; padding: 0; }

        button {
            font-family: inherit;
            font-weight: 500;
            color: var(--crust);
            background: var(--blue);
            border: none;
            border-radius: 0.5rem;
            padding: 0.5rem 1rem;
            cursor: pointer;
        }
        button:hover { background: var(--sapphire); }
    </style>
</head>
<body>
    <header>
        <span class="brand">APP_NAME</span>
        <span id="status" class="offline">connecting…</span>
    </header>

    <main>
        <h1>APP_NAME</h1>
        <div id="app"></div>
    </main>

    <script type="module" src="/js/app.js"></script>
</body>
</html>
```

## WebSocket Client

The client opens a socket, reconnects with backoff on close, dispatches messages by a `type` field, and queues sends until the socket is open. Queueing rather than throwing means `app.js` can call `send` during startup without waiting for the connection.

Messages are JSON objects with a `type` string; the rest of the object is that type's payload. The server end of the same protocol lives in the backend's `ws.js`.

```javascript
// public/js/ws.js
export function connect(path, options = {}) {
    const { onStatus = () => {}, baseDelay = 500, maxDelay = 15000 } = options;
    const handlers = new Map();
    const queue = [];

    let socket = null;
    let attempts = 0;
    let closed = false;

    function url() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${proto}//${location.host}${path}`;
    }

    function backoff() {
        const capped = Math.min(baseDelay * 2 ** attempts, maxDelay);
        return capped / 2 + Math.random() * (capped / 2);
    }

    function open() {
        socket = new WebSocket(url());

        socket.addEventListener('open', () => {
            attempts = 0;
            onStatus(true);
            while (queue.length) socket.send(queue.shift());
        });

        socket.addEventListener('message', (event) => {
            let msg;
            try {
                msg = JSON.parse(event.data);
            } catch {
                return;
            }
            const fns = handlers.get(msg.type);
            if (fns) for (const fn of fns) fn(msg);
        });

        socket.addEventListener('close', () => {
            onStatus(false);
            if (closed) return;
            const delay = backoff();
            attempts += 1;
            setTimeout(open, delay);
        });

        socket.addEventListener('error', () => socket.close());
    }

    open();

    return {
        on(type, handler) {
            const fns = handlers.get(type) ?? [];
            fns.push(handler);
            handlers.set(type, fns);
            return this;
        },
        send(type, payload = {}) {
            const data = JSON.stringify({ type, ...payload });
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(data);
            } else {
                queue.push(data);
            }
        },
        close() {
            closed = true;
            if (socket) socket.close();
        },
    };
}
```

The delay is half the capped value plus a random half, so a server restart does not bring every client back in the same millisecond.

`attempts` resets on a successful open, which keeps a long-lived connection from inheriting the backoff of an outage hours earlier.

The `closed` flag distinguishes a deliberate `close()` from a dropped connection, so an intentional teardown does not immediately reconnect.

An `error` event is followed by `close`, so the handler only calls `socket.close()` and lets the close path own the reconnect. Reconnecting from both would open two sockets.

```javascript
// public/js/app.js
import { connect } from '/js/ws.js';

const statusEl = document.getElementById('status');
const appEl = document.getElementById('app');

const socket = connect('/ws', {
    onStatus(online) {
        statusEl.textContent = online ? 'online' : 'reconnecting…';
        statusEl.className = online ? 'online' : 'offline';
    },
});

socket.on('update', (msg) => {
    appEl.textContent = JSON.stringify(msg.payload, null, 2);
});

socket.send('subscribe', { channel: 'events' });
```
