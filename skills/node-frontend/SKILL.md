---
name: node-frontend
description: Use when implementing the vanilla-JS single-page frontend served by a Node Web Only backend from public/ - covers HTML structure, vendored self-hosted fonts, Catppuccin Mocha theme, static asset layout, and the reconnecting WebSocket client. Not for framework SPAs (React/Vue/Svelte), bundler-driven builds, or externally-hosted frontends.
user-invocable: false
---

# Node Frontend

**Vanilla-JS single-page frontend served by a Node Web Only backend.**

## When to Use

Use this skill when:
- Building the frontend of a Node Web Only app (an HTTP + WebSocket server that also serves an embedded/vendored single-page UI)
- Laying out `public/` (index.html, css/, js/, fonts/)
- Vendoring self-hosted fonts and third-party assets from `node_modules` into `public/`
- Implementing the Catppuccin Mocha dark theme
- Writing the realtime WebSocket client (connect, reconnect, dispatch)

**Requires:** `node-foundations` for project layout and the Node Web Only taxonomy. The server that serves `public/` is owned by `node-backend`.

**Related skills:**
- `node-backend` - serves `public/` over `node:http`, owns the WebSocket server end
- `project-ci-cd` - the Makefile `assets` target that vendors fonts/JS from `node_modules`
- `go-frontend` - the Go analog; shares the same vendored font set and Catppuccin palette byte-for-byte

Node Web Only is the only Node project type today. There is no Node "CLI Only", "CLI + Web", or "Library" type yet — those are out of scope.

## Start here — required reading

Read the **Always** file now, in full, before building the page — it carries the boilerplate and theme you'll be held to. Read the **When** file before the sub-task it names; a subagent may read it if you delegate that work.

**Always:**
- `./references/html-template.md` — full `index.html` boilerplate, Catppuccin palette, vendored-font `@font-face` wiring

**When adding the realtime WebSocket client:**
- `./references/websocket-client.md` — reconnecting client: backoff, `type`-keyed dispatch, queued send

## Directory Structure

```
public/
├── index.html          # the single page (see references/html-template.md)
├── css/
│   ├── inter.css       # @font-face for Inter (body/UI)
│   ├── jetbrains-mono.css  # @font-face for JetBrains Mono Nerd Font (code/mono)
│   └── app.css         # app styles (optional; inline in index.html is also fine)
├── js/
│   ├── app.js          # application logic
│   └── ws.js           # reconnecting WebSocket client (see references/websocket-client.md)
└── fonts/
    ├── inter-*.woff2               # vendored Inter weights
    └── JetBrainsMonoNerdFontMono-*.woff2  # vendored JetBrains Mono Nerd Font weights
```

The backend serves this tree at the site root: `public/index.html` at `/`, `public/css/inter.css` at `/css/inter.css`, `public/fonts/*.woff2` at `/fonts/*.woff2`. `node-backend`'s static handler owns the routing; this skill covers what goes inside `public/`.

## Key Rules

### Vanilla JS, No Framework
- No React/Vue/Svelte, no mandatory bundler. Plain ES modules (`<script type="module">`), the DOM API, and `fetch`/`WebSocket`.
- A build step is optional and only for a binary release path (`bun build --compile` / Node SEA — see `project-ci-cd`), never a hard dependency for developing or running the page.

### Single Page Default
- One `index.html`. Views are shown/hidden client-side; multi-page only when explicitly needed.
- Shared logic lives in `js/app.js`; the WebSocket client in `js/ws.js`.

### Catppuccin Mocha Dark Theme
- Default to Catppuccin Mocha (dark) — the same palette the Go frontend uses. Match an existing project's palette instead of re-theming it.
- Colors are CSS variables on `:root` (block below); reference them as `var(--blue)`, `background: var(--base)`, etc. Do not hardcode hex values in component styles.

### Styles Inline or in css/
- Small apps keep custom CSS in an inline `<style>` block in `index.html`; larger ones use `css/app.css`. Either is fine — pick one, don't scatter.
- The two `@font-face` files (`inter.css`, `jetbrains-mono.css`) always live in `css/` as separate linked stylesheets.

## Vendored Assets

All third-party assets are **self-hosted** — vendored from `node_modules` into `public/` at build time and served from the app's own origin. No runtime CDN, no Google Fonts fetch, no external `<script src>`.

### Shared Vendored Font Set (woff2 only)

The same two-font set as the Go frontend, self-hosted:

| Font | Role | Vendored to | `@font-face` in |
|------|------|-------------|-----------------|
| **Inter** | body / UI text | `public/fonts/inter-*.woff2` | `public/css/inter.css` |
| **JetBrains Mono Nerd Font** | monospace / code (Nerd Font variant, with glyphs) | `public/fonts/JetBrainsMonoNerdFontMono-*.woff2` | `public/css/jetbrains-mono.css` |

- **woff2 only** — no ttf, no woff, no CDN link.
- Both stylesheets are linked in the HTML `<head>`; `font-family: 'Inter'` drives the body, `font-family: 'JetBrains Mono'` drives code/mono.
- Full `@font-face` file contents and the `<head>` wiring are in `./references/html-template.md`.

### Copying from node_modules

The Makefile `assets` target vendors the fonts (and any other JS deps) out of `node_modules` into `public/` — e.g. copying `@fontsource/inter`'s woff2 files into `public/fonts/` and its face CSS into `public/css/`. This keeps `public/` self-contained and reproducible. See `project-ci-cd`'s `assets` / `verify-assets` targets for the copy-and-verify wiring; treat font files under `public/fonts/` as generated, not hand-edited.

## WebSocket Client

The realtime client opens a `WebSocket` to the backend, reconnects with backoff on close, and dispatches incoming messages by a `type` field to registered handlers. Keep it a small standalone ES module (`js/ws.js`), imported by `app.js`. The `node-backend` skill owns the server end of the same protocol.

Full reconnecting-client module (open/message/close, exponential backoff with jitter, a `type`-keyed dispatch table, and a send helper that queues until open) is in `./references/websocket-client.md`.

## Catppuccin Mocha

Declare the palette once on `:root`, then reference the variables everywhere. Same values as the Go frontend.

```css
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
```

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/html-template.md` | Template | Full `index.html` boilerplate (head wiring, inline `:root` palette, body/code font-family) plus the `inter.css` and `jetbrains-mono.css` `@font-face` file contents |
| `./references/websocket-client.md` | Template | Reconnecting WebSocket client module — open/message/close, backoff reconnect, `type`-keyed dispatch, queued send |
