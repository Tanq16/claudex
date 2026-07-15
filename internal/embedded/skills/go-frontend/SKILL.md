---
name: go-frontend
description: Use when implementing an embedded single-page frontend bundled into a Go binary via embed.FS - covers HTML structure, Tailwind setup, Catppuccin theme, static asset management, embed.FS patterns, icons, and PWA. Not for templ/htmx server-rendered HTML or externally-built SPAs.
user-invocable: false
---

# Go Frontend

**Embedded frontend patterns for Go web services.**

## When to Use

Use this skill when:
- Creating an embedded single-page web UI bundled into a Go binary (not templ/htmx or an external SPA)
- Setting up embedded static assets
- Implementing Catppuccin color theme
- Downloading and embedding third-party JS/CSS
- Structuring HTML with Tailwind CSS
- Rendering Markdown with styled output (tables, code, Mermaid diagrams, callouts)
- Setting up app icons and favicons
- Adding PWA (Progressive Web App) support

**Requires:** `go-foundations` for project layout.

## Directory Structure

```
internal/server/
├── server.go           # HTTP server setup
└── static/
    ├── index.html      # Main (often only) HTML file
    ├── manifest.json   # PWA manifest (if PWA enabled)
    ├── sw.js           # Service worker (if PWA enabled)
    ├── app.js          # Application JavaScript (if needed)
    ├── css/            # Downloaded CSS (GitHub markdown, Inter font, etc.)
    ├── fonts/          # Downloaded fonts (Inter woff2 files)
    ├── fontawesome/    # Font Awesome assets
    │   ├── css/
    │   └── webfonts/
    ├── icons/          # App icons, favicons
    │   ├── favicon.ico         # 32x32 ICO for legacy browsers
    │   ├── favicon.png         # 32x32 PNG
    │   ├── apple-touch-icon.png # 180x180 for iOS
    │   ├── icon-192.png        # 192x192 for PWA
    │   ├── icon-512.png        # 512x512 for PWA splash
    │   └── logo.png            # App logo (any size)
    └── js/             # Downloaded JS (tailwindcss.js, marked.js, etc.)
```

## Embedding Pattern

The `embed.FS` server that serves these assets is owned by `go-backend`. Use `../go-backend/references/http-server-template.md` for the full `server.go` (`//go:embed static`, `fs.Sub`, `http.StripPrefix` for `/static/`, and `handleIndex` serving `static/index.html`). This skill covers what goes *inside* `static/`; `go-backend` covers how it is served.

## HTML Template

See `./references/html-template.md` for the complete HTML template with:
- Catppuccin Mocha color scheme (CSS variables)
- Tailwind configuration with custom colors
- Dark theme by default
- System theme detection (when needed)
- Icon links and PWA meta tags
- PWA manifest.json and service worker templates
- Common UI components (cards, buttons, inputs, tables, modals)

For Markdown rendering styles, see the Markdown Rendering section below.

The minimal page is: `<head>` with icon links, font CSS (`css/inter.css` for body, `css/jetbrains-mono.css` for mono/code) plus Font Awesome CSS, an inline `:root` block of Catppuccin Mocha CSS variables, and the Tailwind Play-CDN script wiring those variables into `tailwind.config.theme.extend.colors`; `<body class="bg-base text-text min-h-screen">`. The full boilerplate is in `./references/html-template.md` — copy from there rather than retyping it.

## Key Rules

### No Custom CSS Files
- All custom CSS goes inline in `<style>` blocks within HTML
- Downloaded CSS (Font Awesome, Inter, markdown renderers) goes in `css/` directory
- Override external styles inline when needed (e.g., mermaid SVG colors)

### Dark Theme by Default
- Default to Catppuccin Mocha (dark) for new frontends in this style; if the project already has a palette or theme, match that instead of re-theming it
- When light theme is needed, add Catppuccin Latte with `html.dark` class switching

### Tailwind for New Frontends
- For new embedded frontends in this style, prefer Tailwind utility classes over hand-written layout CSS
- Custom colors via CSS variables mapped to Tailwind config
- Don't rip out an existing project's working CSS or templating approach just to impose Tailwind
- **Play CDN caveat:** `tailwindcss.js` (the Play CDN, vendored to `js/`) compiles utility classes in-browser at runtime. It's fine for embedded tools and dashboards, but Tailwind explicitly marks it "not for production." If the app needs production-grade CSS (smaller payload, no runtime compile), switch to a build step that emits a static `tailwind.css`.

### Single Page Default
- Most projects use single `index.html`
- Multi-page only when explicitly needed
- Shared logic extracted to `app.js` only when many pages need it

## Icons

### Required Icons

| File | Size | Purpose |
|------|------|---------|
| `favicon.ico` | 32x32 | Legacy browser tab icon |
| `favicon.png` | 32x32 | Modern browser tab icon |
| `apple-touch-icon.png` | 180x180 | iOS home screen |
| `icon-192.png` | 192x192 | PWA icon (Android) |
| `icon-512.png` | 512x512 | PWA splash screen |
| `logo.png` | Any | In-app branding |

### Icon Guidelines

- **Format:** PNG with transparency for all icons
- **Style:** Simple, recognizable at small sizes
- **Colors:** Use Catppuccin palette colors that work on both light/dark backgrounds
- **Safe zone:** Keep important content within center 80% for PWA icons (rounded corners crop edges)

### HTML Icon Links

```html
<link rel="icon" type="image/x-icon" href="/static/icons/favicon.ico">
<link rel="icon" type="image/png" sizes="32x32" href="/static/icons/favicon.png">
<link rel="apple-touch-icon" sizes="180x180" href="/static/icons/apple-touch-icon.png">
```

## PWA (Progressive Web App)

Add PWA support when the app should be installable on mobile/desktop. The pieces:
- `static/manifest.json` — name, icons (192/512), `display: standalone`, theme/background colors
- PWA `<meta>` tags + `<link rel="manifest">` in `<head>`
- `static/sw.js` — a no-op service worker (registration only, no caching)
- a small registration script before `</body>`

Full `manifest.json`, meta tags, `sw.js`, and the registration snippet are in `./references/html-template.md` (PWA Files). Drop all of them if not building a PWA.

### When to Add PWA

- App is accessed frequently on mobile
- Offline functionality is useful
- User requested installability
- **Skip PWA** for admin dashboards, dev tools, or rarely-used apps

## Icon Libraries

Use these libraries for UI icons (in order of preference):

| Priority | Library | Use For | CDN |
|----------|---------|---------|-----|
| 1st | Lucide | General UI icons | `unpkg.com/lucide@latest` |
| 2nd | Font Awesome | Fallback, brand icons | `jsdelivr.net` |
| 3rd | Dev Icons | Developer/tech logos | `cdn.jsdelivr.net/gh/devicons/devicon@latest` |

### Lucide (Preferred)

```html
<script src="/static/js/lucide.min.js"></script>
<script>lucide.createIcons();</script>

<!-- Usage -->
<i data-lucide="settings"></i>
<i data-lucide="user"></i>
<i data-lucide="chevron-right"></i>
```

### Font Awesome (Fallback)

```html
<link rel="stylesheet" href="/static/fontawesome/css/all.min.css">

<!-- Usage -->
<i class="fas fa-cog"></i>
<i class="fab fa-github"></i>
```

### Dev Icons (Tech Logos)

```html
<link rel="stylesheet" href="/static/css/devicon.min.css">

<!-- Usage -->
<i class="devicon-go-original-wordmark"></i>
<i class="devicon-docker-plain"></i>
<i class="devicon-kubernetes-plain"></i>
```

### When to Use Each

- **Lucide**: Default choice for all UI icons (settings, navigation, actions)
- **Font Awesome**: Brand icons (`fa-github`, `fa-twitter`) or when Lucide lacks the icon
- **Dev Icons**: Technology/language logos (Go, Docker, Kubernetes, Python, etc.)

## Downloaded Assets

Assets downloaded via `make assets` target:

| Asset | Source | Location |
|-------|--------|----------|
| Tailwind CSS | `cdn.tailwindcss.com` | `js/tailwindcss.js` |
| Lucide | `unpkg.com/lucide@latest` | `js/lucide.min.js` |
| Font Awesome | `jsdelivr.net` | `fontawesome/css/`, `fontawesome/webfonts/` |
| Dev Icons | `jsdelivr.net/gh/devicons/devicon@latest` | `css/devicon.min.css` |
| Inter Font (body/UI) | `fonts.googleapis.com` | `css/inter.css`, `fonts/*.woff2` |
| JetBrains Mono Nerd Font (mono/code, glyphs) | `github.com/ryanoasis/nerd-fonts` | `css/jetbrains-mono.css`, `fonts/*.woff2` |
| Marked.js | `jsdelivr.net` | `js/marked.min.js` |
| Highlight.js | `jsdelivr.net/gh/highlightjs/cdn-release@latest` | `js/highlight.min.js` |
| Highlight.js Theme | Same CDN, `styles/github-dark.min.css` | `css/github-dark.min.css` |
| Chart.js | `jsdelivr.net` | `js/chart.umd.js` |
| Mermaid.js | `jsdelivr.net` | `js/mermaid.min.js` |

**Shared font set:** Inter + JetBrains Mono Nerd Font (woff2, self-hosted) are the same two fonts `node-frontend` vendors — keep them aligned across both frontends; only the static-root path differs (`internal/server/static/` here vs `public/` for Node).

**Note:** Font Awesome CSS needs patching to fix webfont paths:
```bash
sed -i '' 's|../webfonts/|/static/fontawesome/webfonts/|g' fontawesome/css/all.min.css
```

**Pin versions for reproducible builds:** the `@latest` URLs above are convenient but make `make assets` non-deterministic — a fresh download can pull a new major version that breaks rendering. For anything beyond a throwaway prototype, pin each asset to a specific version (e.g. `marked@12.0.0`, `mermaid@11.4.1`, `lucide@0.460.0`) in the Makefile download URLs and bump them deliberately.

## Common Patterns

### API Calls

```javascript
async function fetchData() {
    try {
        const response = await fetch('/api/data');
        if (!response.ok) throw new Error('Failed to fetch');
        return await response.json();
    } catch (error) {
        console.error('Error:', error);
        showError('Failed to load data');
    }
}
```

### Toast Notifications

```html
<div id="toast" class="fixed bottom-4 right-4 hidden">
    <div class="bg-surface0 text-text px-4 py-2 rounded-lg shadow-lg border border-surface1">
        <span id="toast-message"></span>
    </div>
</div>

<script>
function showToast(message, type = 'info') {
    const toast = document.getElementById('toast');
    const msg = document.getElementById('toast-message');
    msg.textContent = message;

    // Color based on type
    const colors = {
        info: 'text-blue',
        success: 'text-green',
        error: 'text-red',
        warning: 'text-yellow'
    };
    msg.className = colors[type] || colors.info;

    toast.classList.remove('hidden');
    setTimeout(() => toast.classList.add('hidden'), 3000);
}
</script>
```

### Loading State

```html
<div id="loading" class="flex items-center justify-center p-4">
    <i data-lucide="loader-2" class="text-blue text-2xl animate-spin"></i>
</div>

<script>
function showLoading() {
    document.getElementById('loading').classList.remove('hidden');
    lucide.createIcons();
}
function hideLoading() {
    document.getElementById('loading').classList.add('hidden');
}
</script>
```

## Markdown Rendering

When the frontend renders Markdown content (notes, documentation, etc.), use Marked.js with a custom renderer, Highlight.js for syntax highlighting, and Catppuccin-styled CSS for all elements.

See `./references/markdown-rendering.md` for the complete implementation:
- Marked.js custom renderer (code blocks with Highlight.js, heading IDs, inline images, callout blockquotes)
- Callout blocks — `> [!TIP]`, `> [!NOTE]`, `> [!INFO]`, `> [!WARNING]`, `> [!DANGER]` with Lucide icons
- Code copy buttons — clipboard button injected on `<pre>` blocks with Lucide copy/check icons
- Full CSS for all markdown elements (headings with per-level Catppuccin colors, tables with striped rows and mauve headers, inline code vs code blocks, lists, blockquotes, HR, scrollbars)

### Mermaid Diagrams

See `./references/mermaid-config.md` for the complete Mermaid.js configuration:
- Full `mermaid.initialize()` config with Catppuccin Mocha theme variables
- Covers all diagram types: flowchart, sequence, gantt, pie, git graph, state, timeline/mindmap
- Container CSS and Gantt chart CSS overrides

### Key Rules for Markdown Rendering

- Use `theme: 'base'` for Mermaid so all colors come from `themeVariables` (not a built-in theme)
- Set `startOnLoad: false` and call `mermaid.run()` manually after `marked.parse()`
- Always call `addCopyButtons()` and `lucide.createIcons()` after rendering markdown
- Inline code uses peach (`#fab387`) on surface0; code blocks use mantle background
- Heading colors: H1=lavender, H2=mauve, H3=blue, H4-H6=text
- Table headers use mauve tint background with mauve text
- Use JetBrains Mono for code, Inter for body text
