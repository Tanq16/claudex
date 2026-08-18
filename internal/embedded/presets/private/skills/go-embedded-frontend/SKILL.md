---
name: go-embedded-frontend
description: A single-page frontend embedded into a Go binary from internal/server/static/ - HTML boilerplate, Tailwind v4 browser build, the Catppuccin Mocha palette, self-hosted fonts, icons, and PWA files. Use when building or changing the web UI of a Go Web Only or CLI + Web project, laying out static assets, theming a page, or adding PWA support. Triggers on internal/server/static/, index.html, tailwind, Catppuccin, favicon, manifest.json, sw.js, Lucide, and Font Awesome. Not for templ or htmx server-rendered HTML, and not for a framework SPA built outside the repository.
user-invocable: false
---

# Go Embedded Frontend

**One `index.html` under `internal/server/static/`, styled with the Tailwind browser build and the Catppuccin Mocha palette, with every asset self-hosted.**

The Go server owns how these bytes are served. This covers what goes inside `static/`.

## Layout

```
internal/server/static/
├── index.html          # the single page
├── app.js              # application logic, when it outgrows an inline block
├── manifest.json       # PWA, when enabled
├── sw.js               # PWA, when enabled
├── css/
│   ├── inter.css               # @font-face for Inter
│   ├── google-sans.css         # @font-face for Google Sans
│   ├── jetbrains-mono.css      # @font-face for JetBrains Mono Nerd Font Mono
│   ├── github-dark.min.css     # highlight.js theme, when rendering markdown
│   └── devicon.min.css         # when using tech logos
├── fonts/              # woff2 only
├── fontawesome/
│   ├── css/
│   └── webfonts/
├── icons/
│   ├── favicon.ico             # 32x32, legacy browser tabs
│   ├── favicon.png             # 32x32
│   ├── apple-touch-icon.png    # 180x180, iOS home screen
│   ├── icon-192.png            # PWA, Android
│   ├── icon-512.png            # PWA, splash screen
│   └── logo.png                # in-app branding
└── js/                 # tailwind.js, lucide.min.js, and anything else vendored
```

Nothing under `css/`, `js/`, `fonts/`, or `fontawesome/` is committed to the repository. A Makefile target downloads them on demand and the release workflow calls that same target, so the tree in git holds only what a person wrote. A downloaded asset in a diff is a binary nobody reviews and a version nobody can trace.

## Assets

Every asset is pinned to an exact version. A floating `@latest` makes two builds of the same commit produce different bytes, and a new major arriving overnight breaks rendering with no diff to point at.

| Asset | Version | Lands at |
|---|---|---|
| Tailwind browser build | `@tailwindcss/browser@4.3.3` | `js/tailwind.js` |
| Lucide | `lucide@1.31.0` | `js/lucide.min.js` |
| Font Awesome | `@fortawesome/fontawesome-free@7.3.1` | `fontawesome/css/`, `fontawesome/webfonts/` |
| Dev Icons | `devicon@2.17.0` | `css/devicon.min.css` |
| Inter | Google Fonts | `css/inter.css`, `fonts/*.woff2` |
| Google Sans | Google Fonts | `css/google-sans.css`, `fonts/*.woff2` |
| JetBrains Mono Nerd Font Mono | nerd-fonts `v3.5.0` | `css/jetbrains-mono.css`, `fonts/*.woff2` |
| Marked | `marked@18.0.9` | `js/marked.min.js` |
| Highlight.js | `highlight.js@11.12.0` | `js/highlight.min.js`, `css/github-dark.min.css` |
| Mermaid | `mermaid@11.16.1` | `js/mermaid.min.js` |
| Chart.js | `chart.js@4.5.1` | `js/chart.umd.js` |

The three fonts are always downloaded, whether or not a given page uses all three, so the asset step is the same everywhere and a later design change needs no build change. Everything below them in the table is downloaded when the project actually uses it.

Web fonts are woff2, never ttf. woff2 is roughly half the bytes of the same ttf and every browser targeted here supports it. Inter and Google Sans come from Google Fonts, which already serves woff2. JetBrains Mono Nerd Font Mono is a patched Nerd Font that Google Fonts does not carry, so it is downloaded from the nerd-fonts release as ttf and compressed to woff2 during the asset step.

Font Awesome's stylesheet references its webfonts by a relative path that does not survive being served from `/static/`, so the asset step rewrites it:

```bash
sed -i '' 's|../webfonts/|/static/fontawesome/webfonts/|g' fontawesome/css/all.min.css
```

## Fonts

| Font | Role | `font-family` |
|---|---|---|
| Inter | body and UI text | `'Inter'` |
| Google Sans | display headings and branding | `'Google Sans'` |
| JetBrains Mono Nerd Font Mono | code, monospace, and glyphs | `'JetBrains Mono'` |

Each has its own `@font-face` stylesheet under `css/`, linked from `<head>`, pointing at local woff2 files. Nothing loads from `fonts.googleapis.com` at run time, because a page that fetches a font from a third party leaks every visitor's IP and stops rendering correctly offline.

## The Page

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APP_NAME</title>

    <link rel="icon" type="image/x-icon" href="/static/icons/favicon.ico">
    <link rel="icon" type="image/png" sizes="32x32" href="/static/icons/favicon.png">
    <link rel="apple-touch-icon" sizes="180x180" href="/static/icons/apple-touch-icon.png">

    <link rel="manifest" href="/static/manifest.json">
    <meta name="theme-color" content="#1e1e2e">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="apple-mobile-web-app-title" content="APP_NAME">

    <link rel="stylesheet" href="/static/css/inter.css">
    <link rel="stylesheet" href="/static/css/google-sans.css">
    <link rel="stylesheet" href="/static/css/jetbrains-mono.css">
    <link rel="stylesheet" href="/static/fontawesome/css/all.min.css">

    <script src="/static/js/lucide.min.js"></script>
    <script src="/static/js/tailwind.js"></script>
    <style type="text/tailwindcss">
      @theme {
        --color-rosewater: #f5e0dc; --color-flamingo: #f2cdcd; --color-pink: #f5c2e7;
        --color-mauve: #cba6f7;     --color-red: #f38ba8;      --color-maroon: #eba0ac;
        --color-peach: #fab387;     --color-yellow: #f9e2af;   --color-green: #a6e3a1;
        --color-teal: #94e2d5;      --color-sky: #89dceb;      --color-sapphire: #74c7ec;
        --color-blue: #89b4fa;      --color-lavender: #b4befe; --color-text: #cdd6f4;
        --color-subtext1: #bac2de;  --color-subtext0: #a6adc8; --color-overlay2: #9399b2;
        --color-overlay1: #7f849c;  --color-overlay0: #6c7086; --color-surface2: #585b70;
        --color-surface1: #45475a;  --color-surface0: #313244; --color-base: #1e1e2e;
        --color-mantle: #181825;    --color-crust: #11111b;

        --font-sans: 'Inter', sans-serif;
        --font-display: 'Google Sans', sans-serif;
        --font-mono: 'JetBrains Mono', monospace;
      }
    </style>
</head>
<body class="bg-base text-text min-h-screen font-sans">
    <nav class="bg-mantle border-b border-surface0 px-4 py-3">
        <div class="max-w-6xl mx-auto flex items-center justify-between">
            <div class="flex items-center gap-2">
                <img src="/static/icons/logo.png" alt="Logo" class="w-8 h-8">
                <span class="text-lg font-semibold font-display">APP_NAME</span>
            </div>
        </div>
    </nav>

    <main class="max-w-6xl mx-auto px-4 py-8">
        <h1 class="text-2xl font-bold mb-6 font-display">Page Title</h1>
    </main>

    <script src="/static/app.js"></script>
    <script>lucide.createIcons();</script>
    <script>
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/static/sw.js');
        }
    </script>
</body>
</html>
```

Theme values are declared in a `<style type="text/tailwindcss">` block containing an `@theme` at-rule. The Tailwind v4 browser build reads that block and rejects a JavaScript config entirely, so the `tailwind.config = {...}` global from v3 produces no CSS and no error.

A `--color-mauve` entry generates `bg-mauve`, `text-mauve`, `border-mauve`, and a real `var(--color-mauve)` for hand-written CSS. Naming the palette entries after Catppuccin's own names keeps the class you write and the swatch you picked identical.

`@import "tailwindcss";` is omitted, because the browser build injects it when no `@import` appears anywhere in the block. Writing one takes over the whole import graph and drops the base styles unless you add it back yourself.

## Theme

Catppuccin Mocha is the default for a new frontend in this style. A project that already has a palette keeps it rather than being re-themed, since re-theming an existing app is a design decision rather than a convention.

Light and dark switching is added only when it is asked for. Tailwind v4 drives it with a custom variant rather than a config key:

```html
<style type="text/tailwindcss">
  @custom-variant dark (&:where(.dark, .dark *));
  @theme {
    /* Catppuccin Latte, the light default */
    --color-base: #eff1f5; --color-mantle: #e6e9ef; --color-crust: #dce0e8;
    --color-text: #4c4f69; --color-blue: #1e66f5;  --color-mauve: #8839ef;
  }
</style>
```

The Mocha values are then applied under `html.dark`, and a script in `<head>` sets that class from `localStorage` or `prefers-color-scheme` before the body renders. Running it in `<head>` rather than at the end of the document is what prevents a flash of the wrong theme.

## Styling Rules

Custom CSS lives in inline `<style>` blocks. Downloaded stylesheets live in `css/`. Keeping the two apart means the asset step can wipe and re-download `css/` without touching anything a person wrote.

Tailwind utility classes are preferred over hand-written layout CSS for a new frontend in this style. An existing project's working CSS is not ripped out to impose them.

The browser build compiles utility classes in the page at run time. Tailwind documents it as development-only. It is fine for an embedded tool or a dashboard, and an app that needs a small payload and no runtime compile moves to a build step emitting a static `tailwind.css`.

One `index.html` is the default. Views are shown and hidden client-side, and shared logic moves into `app.js` only once more than one place needs it.

## Icons

| Library | Use for | Vendored to |
|---|---|---|
| Lucide | general UI icons, the default | `js/lucide.min.js` |
| Font Awesome | brand icons, and gaps in Lucide | `fontawesome/` |
| Dev Icons | technology and language logos | `css/devicon.min.css` |

```html
<i data-lucide="settings"></i>
<script>lucide.createIcons();</script>

<i class="fab fa-github"></i>

<i class="devicon-go-original-wordmark"></i>
```

`lucide.createIcons()` runs after any DOM update that inserts an `<i data-lucide>`, since it replaces those elements with inline SVG once and does not observe later insertions.

App icons are PNG with transparency, recognizable at 16 pixels, and drawn from the Catppuccin palette so they read on both light and dark browser chrome. PWA icons keep their content inside the centre 80%, because the installer rounds the corners off.

## PWA

Add PWA support when the app is used often on a phone, when offline behavior is useful, or when installability was requested. Skip it for an admin dashboard, a developer tool, or anything opened twice a year, where an install prompt is noise.

`static/manifest.json`:

```json
{
  "name": "APP_NAME",
  "short_name": "APP_SHORT",
  "description": "APP_DESCRIPTION",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#1e1e2e",
  "theme_color": "#1e1e2e",
  "icons": [
    { "src": "/static/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/static/icons/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

`static/sw.js` is a no-op worker that exists only to make the app installable:

```javascript
self.addEventListener('fetch', () => {});
```

It caches nothing and every request goes to the network, so the app behaves exactly like a normal tab. A caching worker serves stale assets after a deploy and gives users a version they cannot refresh away.

Drop the manifest, the worker, the registration script, and the PWA meta tags together when not building a PWA. A manifest with no worker is an install prompt that leads nowhere.
