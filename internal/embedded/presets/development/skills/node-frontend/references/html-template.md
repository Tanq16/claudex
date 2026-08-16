# HTML Template

The Node Web Only frontend is a single self-hosted `index.html` served from `public/` at the site root. Fonts and styles are vendored under `public/` and linked with root-absolute paths (`/css/...`, `/fonts/...`). No CDN, no framework, no bundler required.

## index.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APP_NAME</title>

    <link rel="icon" type="image/png" sizes="32x32" href="/icons/favicon.png">

    <link rel="stylesheet" href="/css/inter.css">
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

        code, pre, kbd, samp {
            font-family: 'JetBrains Mono', monospace;
        }

        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 1rem;
            background: var(--mantle);
            border-bottom: 1px solid var(--surface0);
        }

        header .brand { font-weight: 600; }

        header #status {
            font-size: 0.85rem;
            color: var(--overlay1);
        }
        header #status.online { color: var(--green); }
        header #status.offline { color: var(--red); }

        main {
            max-width: 60rem;
            margin: 0 auto;
            padding: 2rem 1rem;
        }

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

`app.js` wires the page to the backend; it imports the WebSocket client from `./references/websocket-client.md`:

```javascript
import { connect } from '/js/ws.js';

const statusEl = document.getElementById('status');
const appEl = document.getElementById('app');

const socket = connect('/ws', {
    onStatus(online) {
        statusEl.textContent = online ? 'online' : 'reconnecting…';
        statusEl.className = online ? 'online' : 'offline';
    },
});

socket.on('update', (payload) => {
    appEl.textContent = JSON.stringify(payload, null, 2);
});
```

## css/inter.css

Inter drives body and UI text. Vendored woff2 only — the two weights the Makefile vendors (400 and 600).

```css
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

## css/jetbrains-mono.css

The Nerd Font variant (with glyphs) drives code and monospace. Vendored woff2 only.

```css
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
