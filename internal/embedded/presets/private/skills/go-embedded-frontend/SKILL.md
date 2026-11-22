---
name: go-embedded-frontend
description: A single-page frontend embedded into a Go binary from internal/server/static/ - HTML boilerplate, Tailwind v4 browser build, the Catppuccin palette and its opt-in light mode, the ui component kit, self-hosted fonts, icons, and PWA files. Use when building or changing the web UI of a Go Web Only or CLI + Web project, building a control or a view, laying out static assets, theming a page, adding a light mode, choosing which surface a panel or control sits on, or adding PWA support. Triggers on internal/server/static/, index.html, ui.js, tailwind, @theme, Catppuccin, crust, surface0, data-scroll, favicon, manifest.json, sw.js, Lucide, and Font Awesome. Not for templ or htmx server-rendered HTML, and not for a framework SPA built outside the repository.
user-invocable: false
---

# Go Embedded Frontend

**One `index.html` under `internal/server/static/`, styled with the Tailwind browser build and the Catppuccin Mocha palette, with every asset self-hosted.**

## Start here: required reading

`ui.js` holds all of kit-core, plus each member of the other groups that the app calls or that a called member calls in turn.

**Always, for any view, control, or theme work:**
- `./references/foundations.md`: color roles, fill and state rules, type, radius, icon sizes, base CSS, and light mode
- `./references/kit-core.md`: the `ui` object, buttons, chips, fields, delegation, popovers, tooltips, modals, and toasts

**When building that part:**
- `./references/layout.md`: the shell, router, nav, page heads, galleries, workspaces, scroll regions, or sidebars
- `./references/controls.md`: inputs, textareas, numbers, checkboxes, switches, radios, pill tracks, choice cards, swatches, or tags
- `./references/selects-menus.md`: a select, a date picker, or a menu
- `./references/feedback.md`: toasts, tooltips, modals, banners, badges, loading states, or empty states
- `./references/data.md`: stats, cards, details, a table, or list rows
- `./references/media-music.md`: an audio player, or the seek bar and transport any media view uses
- `./references/media-video.md`: a video player, read with media-music.md
- `./references/media-reader.md`: a PDF reader

## Layout

```
internal/server/static/
├── index.html          # the single page
├── ui.js               # the component kit
├── app.js              # application logic, when it outgrows an inline block
├── manifest.json       # PWA, when enabled
├── sw.js               # PWA, when enabled, served at /sw.js
├── css/
│   ├── inter.css               # @font-face for Inter
│   ├── google-sans.css         # @font-face for Google Sans
│   ├── jetbrains-mono.css      # @font-face for JetBrains Mono
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
│   └── logo.svg                # the project's logo, in-app branding
└── js/                 # tailwind.js, lucide.min.js, and anything else vendored
```

Nothing under `css/`, `js/`, `fonts/`, or `fontawesome/` is committed, since the Makefile downloads it.

## Assets

Every asset is pinned to an exact version, so two builds of one commit produce the same bytes.

| Asset | Version | Lands at |
|---|---|---|
| Tailwind browser build | `@tailwindcss/browser@4.3.3` | `js/tailwind.js` |
| Lucide | `lucide@1.34.0` | `js/lucide.min.js` |
| Font Awesome | `@fortawesome/fontawesome-free@7.3.1` | `fontawesome/css/`, `fontawesome/webfonts/` |
| Dev Icons | `devicon@2.17.0` | `css/devicon.min.css` |
| Inter | Google Fonts | `css/inter.css`, `fonts/*.woff2` |
| Google Sans | Google Fonts | `css/google-sans.css`, `fonts/*.woff2` |
| JetBrains Mono | Google Fonts | `css/jetbrains-mono.css`, `fonts/*.woff2` |
| JetBrains Mono Nerd Font Mono | nerd-fonts `v3.5.1`, opt-in | `css/jetbrains-mono.css`, `fonts/*.woff2` |
| Chart.js | `chart.js@4.5.1` | `js/chart.umd.js` |

Tailwind, Lucide, and the three fonts are always downloaded, and the rest only when the project uses them. Marked, Highlight.js, Mermaid, and PDF.js are pinned where their usage is documented.

Web fonts are woff2, never ttf, since woff2 is roughly half the bytes and every browser here supports it.

Font Awesome's stylesheet points at its webfonts by a relative path that breaks under `/static/`, so the asset step rewrites it to `/static/fontawesome/webfonts/`.

## Fonts

| Font | Role |
|---|---|
| Inter | body and UI text |
| Google Sans | display headings and branding |
| JetBrains Mono | code and monospace |

Each has its own `@font-face` stylesheet under `css/`, pointing at local woff2 files. Nothing loads from `fonts.googleapis.com` at run time, since a page fetching a font from a third party leaks every visitor's IP and stops rendering offline.

The mono slot takes the plain family. The Nerd Font variant is earned only by a page rendering Nerd Font glyphs, such as a Powerline separator or terminal output carrying them, and an icon Lucide or Font Awesome already has is not a reason. The Makefile switches it under the same family name.

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
    <meta name="theme-color" content="#11111b">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-title" content="APP_NAME">

    <link rel="stylesheet" href="/static/css/inter.css">
    <link rel="stylesheet" href="/static/css/google-sans.css">
    <link rel="stylesheet" href="/static/css/jetbrains-mono.css">

    <script src="/static/js/lucide.min.js"></script>
    <script src="/static/js/tailwind.js"></script>
    <style>
      :root, .dark {
        color-scheme: dark;
        --ctp-rosewater: #f5e0dc; --ctp-flamingo: #f2cdcd; --ctp-pink: #f5c2e7;
        --ctp-mauve: #cba6f7;     --ctp-red: #f38ba8;      --ctp-maroon: #eba0ac;
        --ctp-peach: #fab387;     --ctp-yellow: #f9e2af;   --ctp-green: #a6e3a1;
        --ctp-teal: #94e2d5;      --ctp-sky: #89dceb;      --ctp-sapphire: #74c7ec;
        --ctp-blue: #89b4fa;      --ctp-lavender: #b4befe; --ctp-text: #cdd6f4;
        --ctp-subtext1: #bac2de;  --ctp-subtext0: #a6adc8; --ctp-overlay2: #9399b2;
        --ctp-overlay1: #7f849c;  --ctp-overlay0: #6c7086; --ctp-surface2: #585b70;
        --ctp-surface1: #45475a;  --ctp-surface0: #313244; --ctp-base: #1e1e2e;
        --ctp-mantle: #181825;    --ctp-crust: #11111b;    --ctp-shade: #11111b;
      }
    </style>
    <style type="text/tailwindcss">
      @theme inline {
        --color-rosewater: var(--ctp-rosewater); --color-flamingo: var(--ctp-flamingo);
        --color-pink: var(--ctp-pink);           --color-mauve: var(--ctp-mauve);
        --color-red: var(--ctp-red);             --color-maroon: var(--ctp-maroon);
        --color-peach: var(--ctp-peach);         --color-yellow: var(--ctp-yellow);
        --color-green: var(--ctp-green);         --color-teal: var(--ctp-teal);
        --color-sky: var(--ctp-sky);             --color-sapphire: var(--ctp-sapphire);
        --color-blue: var(--ctp-blue);           --color-lavender: var(--ctp-lavender);
        --color-text: var(--ctp-text);           --color-subtext1: var(--ctp-subtext1);
        --color-subtext0: var(--ctp-subtext0);   --color-overlay2: var(--ctp-overlay2);
        --color-overlay1: var(--ctp-overlay1);   --color-overlay0: var(--ctp-overlay0);
        --color-surface2: var(--ctp-surface2);   --color-surface1: var(--ctp-surface1);
        --color-surface0: var(--ctp-surface0);   --color-base: var(--ctp-base);
        --color-mantle: var(--ctp-mantle);       --color-crust: var(--ctp-crust);
        --color-shade: var(--ctp-shade);
      }
      @theme {
        --font-sans: 'Inter', sans-serif;
        --font-display: 'Google Sans', sans-serif;
        --font-mono: 'JetBrains Mono', monospace;
        --font-mono--font-feature-settings: "calt" 0;
        --breakpoint-3xl: 120rem;
      }
      @layer base {
        button:not(:disabled), [role="button"], [role="option"], a[href], summary { cursor: pointer; }
      }
    </style>
</head>
<body class="bg-crust text-subtext0 h-dvh overflow-hidden flex flex-col font-sans antialiased">
    <header class="shrink-0 flex flex-wrap items-center gap-3 px-4 pt-4 sm:flex-nowrap sm:px-6 lg:px-10 lg:pt-6">
        <a href="#/" aria-label="APP_NAME" class="flex items-center gap-2.5 font-display text-xl font-medium text-text">
            <img src="/static/icons/logo.svg" alt="" class="size-11 sm:size-13">
            <span class="max-sm:hidden">APP_NAME</span>
        </a>
        <nav id="nav" class="flex gap-1 rounded-full bg-mantle p-1 max-sm:order-last max-sm:w-full sm:ml-auto"></nav>
    </header>

    <main id="view" class="flex-1 min-h-0 px-4 py-4 sm:px-6 lg:px-10 lg:py-6"></main>

    <div id="layer-root"></div>

    <script src="/static/ui.js"></script>
    <script src="/static/app.js"></script>
    <script>
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js');
        }
    </script>
</body>
</html>
```

The v4 browser build reads `@theme` from the `<style type="text/tailwindcss">` block, and the v3 `tailwind.config = {...}` global produces no CSS and no error.

A `--color-mauve` entry generates `bg-mauve`, `text-mauve`, and a real `var(--color-mauve)` for hand-written CSS. Naming the entries after Catppuccin's own names keeps the class written and the swatch picked identical.

`@theme inline` makes every utility emit the `--ctp-*` variable directly, so redefining the set on any element re-themes its subtree. Plain `@theme` resolves the mapping once at `:root`, and a subtree pinned with `class="dark"` would follow the page instead. Opacity modifiers survive it, since v4 emits `color-mix(in oklab, var(--ctp-mauve) 50%, transparent)` rather than substituting a value.

The `@layer base` rule restores the pointer cursor, because the v4 preflight leaves buttons on the default arrow.

`@import "tailwindcss";` is omitted, because the browser build injects it when no `@import` appears in the block. Writing one takes over the import graph and drops the base styles.

## Layering

The page is grounded on `crust`, the step furthest back in both themes, and every surface is raised from there. Grounding on `base` instead leaves the chrome a step behind the page it sits on and costs contrast against every text color.

| Role | Token |
|---|---|
| Page ground, and any chrome flush with it such as a sidebar, header, or rail | `crust` |
| A surface that reads as lifted off the page, such as a card, modal, popover, or a pill bar on the ground | `mantle` |
| A control sitting on either of those, such as an input, select, chip, button, or pill track, and a selected row | `surface0` |
| Borders, dividers, the hover or open state of a `surface0` control, its selected segment, and anything nested in it | `surface1` |
| A well recessed inside a `mantle` panel, such as a code block, an embedded preview, or a media frame | `base` |

Adjacent layers step one rung and no more. `crust` to `mantle` is a contrast ratio of 1.07 and `crust` to `surface0` is 1.49, which separates them without drawing a seam. Skipping a rung reads as two unrelated panels rather than one raised off the other.

Every fill steps forward from the layer under it, never back toward the ground. The ramp runs dark to light in Mocha and light to dark in the light palette, so a `crust` track inside a `mantle` panel reads as a groove in one theme and a bright strip in the other.

Body copy takes `subtext0` rather than `text`, which would make every word shout and leave nothing louder for a heading.

Structure never borrows from the accent ramp. The neutral steps carry it, and an accent marks one thing per view: the active item, the primary action, or a state. Two accents competing means neither is signal.

## Theme

Catppuccin Mocha is the default for a new frontend. A project that already has a palette keeps it, since re-theming an existing app is a design decision rather than a convention.

A light mode is added only when asked for, from `./references/foundations.md`, since stock Latte washes out.

## Styling Rules

Styling is Tailwind utility classes on the element. Hand-written CSS earns its place only in a case that has no utility form:

- `@font-face` declarations
- scrollbar pseudo-elements, `::selection`, and other pseudo-elements Tailwind does not reach
- a subtree a third party generates, where the classes are not yours to write, such as rendered Markdown, a Mermaid diagram, or an editor widget
- `@media print` rules
- `@keyframes`

Everything else is a class. A hand-written rule that duplicates a utility is a second place a color or a spacing value can drift, and it is invisible to anyone reading the element.

The custom CSS that remains lives in inline `<style>` blocks rather than `css/`, so the asset step can wipe and re-download `css/` without touching anything a person wrote.

An existing project's working CSS stays, since the rule governs what you write.

The browser build compiles utility classes at run time, which Tailwind documents as development-only. It is fine for an embedded tool or a dashboard, and an app needing a small payload moves to a build step emitting a static `tailwind.css`.

One `index.html` is the default. Views are shown and hidden client-side, and shared logic moves into `app.js` only once more than one place needs it.

Every form control renders at 16px on a coarse pointer through `pointer-coarse:text-[16px]`, because iOS Safari zooms in on a focused control under 16px and does not zoom back out. `text-base` cannot carry it, since the palette's `--color-base` turns `text-base` into a color with no font size.

The viewport meta stays `width=device-width, initial-scale=1.0`, since `user-scalable=no` and `maximum-scale=1` suppress the zoom only by taking pinch zoom away from everyone.

## Icons

| Library | Use for |
|---|---|
| Lucide | general UI icons, the default |
| Font Awesome | brand icons, and gaps in Lucide |
| Dev Icons | technology and language logos |

```html
<i data-lucide="settings"></i>
<script>lucide.createIcons();</script>

<link rel="stylesheet" href="/static/fontawesome/css/all.min.css">
<i class="fab fa-github"></i>

<link rel="stylesheet" href="/static/css/devicon.min.css">
<i class="devicon-go-original-wordmark"></i>
```

App icons are PNG exports of the project's own `logo.svg`, so the mark is drawn once. They keep transparency, read at 16 pixels, and use the Catppuccin palette so they show on light and dark browser chrome. PWA icons keep their content inside the center 80%, because the installer rounds the corners off.

## PWA

Add PWA support when the app is used often on a phone, when offline behavior is useful, or when installability was requested. Skip it for an admin dashboard or a developer tool, where an install prompt is noise.

`static/manifest.json`:

```json
{
  "name": "APP_NAME",
  "short_name": "APP_SHORT",
  "description": "APP_DESCRIPTION",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#11111b",
  "theme_color": "#11111b",
  "icons": [
    { "src": "/static/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/static/icons/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

The page carries no `apple-mobile-web-app-status-bar-style` tag. Without it, iOS draws an opaque status bar in the page's color and starts the page below it. `black-translucent` instead lays a blur over the top of the page that no CSS removes, and WebKit has deprecated it.

iOS reads the status bar style only at install, so an existing install keeps the blur until the app is removed and added again.

The body is sized in `dvh` rather than `vh`, because a Home Screen app without the status bar tag computes `100vh` incorrectly and the page scrolls (WebKit bug 316008).

`static/sw.js` is a no-op worker:

```javascript
self.addEventListener('fetch', () => {});
```

It caches nothing, so the app behaves like a normal tab. A caching worker serves stale assets after a deploy and gives users a version they cannot refresh away.

The manifest, the two icon sizes, and HTTPS make the app installable, not the worker. The worker earns its line by controlling the app's pages, which later offline or push behavior needs.

A worker's scope is the directory it is served from, so one at `/static/sw.js` never controls the app at `/`, and it registers without error. A `Service-Worker-Allowed` header on the static mount is the wrong repair, because it widens every file under `/static/` to fix one.

The file stays under `static/` so `//go:embed static` picks it up, and a route at `/sw.js` serves it from the embedded FS through `http.ServeFileFS` with a `text/javascript` content type.

Drop the manifest, the worker, its route, the registration script, the manifest link, and the two `apple-mobile-web-app-*` meta tags together when not building a PWA. The manifest alone prompts an install, which an app gaining nothing from one should not offer. `theme-color` stays, because it colors the browser's own chrome and light mode updates it.
