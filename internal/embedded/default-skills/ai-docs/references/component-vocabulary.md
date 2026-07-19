# AI-docs component vocabulary

The viewer (`index.html`) and print stylesheet (`print.html`) bind every class below. Use these patterns and only these patterns. Don't invent new classes, colors, or callout styles — the chrome already handles palette, headings, TOC, and PDF.

Every curated file's first line MUST be `<!-- curated -->` so `convert.mjs` never overwrites it. Write body fragments only — no `<html>`, `<head>`, `<body>`, `<script>`, `<style>`, or `<link>` tags.

## Page header

The title block. `eyebrow` is an optional kicker (category / number). `lede` is one sentence stating the headline finding.

```html
<header class="doc-header">
  <div class="eyebrow">comparison · caching</div>
  <h1>Redis vs in-memory cache</h1>
  <p class="lede">Redis wins on multi-instance correctness; in-memory wins on p99 latency under 1 node.</p>
</header>
```

## TL;DR (optional)

A short aside of conclusions. Use when the doc is long enough that a reader wants the answer up front.

```html
<aside class="tldr">
  <div class="label">TL;DR</div>
  <ul>
    <li>Claim with a concrete anchor.</li>
    <li>Claim with a concrete anchor.</li>
  </ul>
</aside>
```

## Facts grid (numbers / quick stats)

For a row of headline numbers. Auto-wraps.

```html
<div class="facts">
  <div class="fact"><div class="n">9/65</div><div class="l">Endpoints migrated</div></div>
  <div class="fact"><div class="n">~15</div><div class="l">Left to convert</div></div>
</div>
```

## Steps (ordered process)

For sequences. Bold the action, then the detail.

```html
<ol class="steps">
  <li><strong>Receive request.</strong> Handler accepts the call.</li>
  <li><strong>Authorize.</strong> JWT validated, tenant resolved.</li>
  <li><strong>Return.</strong> Response serialized and sent.</li>
</ol>
```

## Two- / three-column layout

For side-by-side comparisons. `cols-2` or `cols-3`. Collapses to one column on narrow screens.

```html
<div class="cols-2">
  <div class="col"><h4>Present</h4><ul><li>...</li></ul></div>
  <div class="col"><h4>Missing</h4><ul><li>...</li></ul></div>
</div>
```

## Code with caption

For code listings that need a file:line or source label.

```html
<figure class="code">
  <figcaption>internal/cache/redis.go:42 <span class="lang">go</span></figcaption>
  <pre><code class="language-go">func (c *Cache) Get(ctx context.Context, k string) ([]byte, error) { ... }</code></pre>
</figure>
```

## Diagram (architecture / flow / sequence / bars)

For any topology, data-flow, **sequence/swimlane**, or **bar-chart** picture, hand-author an inline `<svg>` wrapped in `figure.diagram`. The wrapper frames it (mantle surface), centers it, caps width, and scales it responsively. **The full house style — the two diagram families (topology + swimlane), the "pick the representation first" guide, canvas size, per-element `<title>` tooltips, semantic color-coding, connectors, the bar-chart pattern, and copyable worked examples — lives in `./diagrams.md`. Read it before authoring any diagram.** Quick shape:

```html
<figure class="diagram">
  <svg viewBox="0 0 960 460" width="100%" xmlns="http://www.w3.org/2000/svg" fill="#cdd6f4" stroke="none">
    <rect x="0" y="0" width="960" height="460" fill="#1e1e2e"/>
    <g><title>Client — the only origin surface</title>
      <rect x="380" y="20" width="200" height="44" rx="6" fill="#313244" stroke="#b4befe" stroke-width="1.5"/>
      <text x="480" y="47" text-anchor="middle" font-size="13" font-weight="600" fill="#b4befe">Client</text>
    </g>
    <!-- ...more <g><title> nodes, boundaries, and connectors — see diagrams.md -->
  </svg>
  <figcaption>Diagram title</figcaption>
</figure>
```

Palette hexes for SVG fills/strokes (match the theme): base `#1e1e2e`, mantle `#181825`, crust `#11111b`, surface0 `#313244`, surface1 `#45475a`, text `#cdd6f4`, subtext0 `#a6adc8`, overlay1 `#7f849c`/`#9399b2`, lavender `#b4befe`, mauve `#cba6f7`, pink `#f5c2e7`, peach `#fab387`, yellow `#f9e2af`, green `#a6e3a1`, blue `#89b4fa`, teal `#74c7ec`/`#89dceb`, sky-green `#94e2d5`, red `#f38ba8`. Use `stroke-dasharray="6 4"` for "future/planned" nodes, a solid stroke for shipped ones.

## Expandable detail

For supporting evidence, long quotes, or raw material the reader can skip. This is the primary tool for keeping the main flow consumable — tuck depth beneath its conclusion. Force-opened in PDF.

```html
<details class="expand">
  <summary>Full benchmark output</summary>
  <pre><code>... long listing ...</code></pre>
</details>
```

## Note (subtle full-width block)

A single observation, footnote, or aside. NO left-border accent bar.

```html
<div class="note">
  <p>Single observation or caveat.</p>
</div>
```

Color variants: add `note-mauve`, `note-yellow`, `note-red`, or `note-green`.

## Chips (status indicators)

Inline status tags.

```html
<span class="chip chip-green">present</span>
<span class="chip chip-red">missing</span>
<span class="chip chip-yellow">partial</span>
<span class="chip chip-mauve">net-new</span>
<span class="chip chip-overlay">n/a</span>
```

## Inline references

```html
<code class="path">internal/cache/redis.go:42</code>   <!-- file path / location, rendered blue -->
<code class="kbd">CACHE_BACKEND</code>                  <!-- env var / key, rendered as a key cap -->
<code>regular inline code</code>
```

## Tables, lists, prose

Plain markdown-style HTML (`<table>`, `<ul>`, `<ol>`, `<p>`, `<blockquote>`) is already styled. Use tables for comparisons, lists when there are more than two items, prose for everything else. A right-hand table of contents is generated automatically from `<h2>` and `<h3>` once a doc has 3+ of them — so lean on headings to make long docs navigable.

Write a plain `<table>`: the viewer auto-wraps it in a scroll container so a wide table wraps its cells while there is room and scrolls horizontally inside its own box once columns hit their readable floor. Don't hand-wrap tables in a div and don't add wrapper classes.

## Copy buttons (automatic — author nothing)

The viewer injects a hover copy control onto every code block and every diagram, so you never hand-author copy UI:

- **Code blocks** (`<pre>`) get a `copy` button that copies the source text.
- **Diagrams / any content `<svg>`** get `svg` (copies the SVG source) and `png` (copies a rendered PNG, with the diagram's dark canvas background baked in) buttons.

Just write normal `<pre>` / `<svg>`; the controls appear on load and never show up in the printed PDF.

## UI rules (do not violate)

- No left-border accent callouts. No `border-l-4`. Use `.note` or `<blockquote>`.
- No emoji-prefixed callouts ("📝 Note:", "⚠️ Warning:"). Use `.note` or chips.
- No gradient text, glow, or shimmer. No big icon hero sections.
- No "ProTip" / "Quick win" boxes.
- Heading colors are fixed (h1 lavender, h2 mauve, h3 pink, h4 peach, h5/h6 yellow). Don't override.
- `strong` is peach, `em` is yellow. Don't introduce new prose accent colors.
- Don't add npm deps (only `marked`), don't add a bundler, don't add a light mode or new theme.
