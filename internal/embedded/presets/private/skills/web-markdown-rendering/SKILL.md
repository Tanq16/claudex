---
name: web-markdown-rendering
description: Rendering Markdown to sanitized, styled HTML in a browser page with Marked, DOMPurify, Highlight.js, and the Catppuccin palette. Use when a frontend displays notes, documentation, or any user-supplied Markdown, when adding syntax highlighting, callout blocks, or code copy buttons, or when styling rendered Markdown. Triggers on marked.parse, marked.use, DOMPurify.sanitize, hljs.highlight, .hljs token colors, .markdown-body, callout blockquotes such as [!TIP] and [!WARNING], copy-to-clipboard buttons on code blocks, and a table of contents beside a document.
user-invocable: false
---

# Web Markdown Rendering

**Marked parses, DOMPurify sanitizes, Highlight.js tokenizes the code, and one CSS block colors all of it from the page's `--ctp-*` palette in either theme.**

The libraries are vendored and pinned, never loaded from a CDN at run time: `marked@18.2.0`, `dompurify@3.4.18`, and `highlight.js@11.12.0`. Highlight.js contributes its grammars and no stylesheet, since a fixed-hex theme such as `github-dark` stays dark when the page turns light.

```html
<script src="/static/js/marked.umd.js"></script>
<script src="/static/js/purify.min.js"></script>
<script src="/static/js/highlight.min.js"></script>
```

`ui` below is the page's component kit, which supplies `esc`, `iconButton`, `on`, and `icons`. Building the copy button from the kit keeps it identical to every other icon button on the page.

## Call Order

The three steps run in this order after every content change, because each depends on the DOM the previous one produced.

```javascript
render(container, source) {
    container.innerHTML = DOMPurify.sanitize(marked.parse(source));
    md.mermaid(container);
    ui.icons(container);
},
```

Marked passes raw HTML in the source straight through, so its output is sanitized before it reaches the DOM. Without that, rendering a README or a note runs any `<script>` or `onerror` handler the file carries.

`md.mermaid` is the diagram step from `web-mermaid-diagrams`, and it returns early when the container holds no diagram.

`ui.icons` runs last and unconditionally, since the callouts and the copy buttons emit `<i data-lucide>` placeholders on every parse.

## The Renderer

`marked.use({ renderer })` is called once at startup. Overriding three token types covers code fences, heading anchors, and callouts, and everything else keeps Marked's default output.

```javascript
const md = {
    slug(text) {
        return String(text).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '');
    },

    init() {
        const icons = { tip: 'lightbulb', info: 'info', danger: 'triangle-alert', warning: 'triangle-alert', note: 'sticky-note' };
        marked.use({
            renderer: {
                code({ text, lang }) {
                    if (lang === 'mermaid') return `<div class="mermaid">${ui.esc(text)}</div>`;
                    const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                    let html = ui.esc(text);
                    try {
                        html = hljs.highlight(text, { language }).value;
                    } catch {}
                    return `<div class="group relative"><pre><code class="hljs language-${language}">${html}</code></pre><span class="absolute top-2 right-2 opacity-0 transition-opacity group-hover:opacity-100 max-md:opacity-100">${ui.iconButton('copy', { tip: 'Copy', size: 'sm', variant: 'secondary', attrs: 'data-action="md-copy"' })}</span></div>`;
                },
                heading({ tokens, depth }) {
                    const text = this.parser.parseInline(tokens);
                    return `<h${depth} id="${md.slug(text.replace(/<[^>]*>/g, ''))}">${text}</h${depth}>`;
                },
                blockquote(token) {
                    const body = this.parser.parse(token.tokens);
                    const match = token.text.match(/^\[!(TIP|NOTE|INFO|WARNING|DANGER)\]/i);
                    if (!match) return `<blockquote>${body}</blockquote>`;
                    const type = match[1].toLowerCase();
                    return `<div class="callout ${type}"><span class="callout-icon"><i data-lucide="${icons[type]}"></i></span><div class="callout-content">${body.replace(/<p>\s*\[!\w+\]\s*/i, '<p>')}</div></div>`;
                },
            },
        });
    },
};
```

A `mermaid` fence becomes a `<div class="mermaid">` holding the escaped source rather than a highlighted block, and it carries no copy button. The escape keeps an arrow such as `-->` or a `<br/>` label from being rewritten by the HTML parser and the sanitizer before Mermaid reads the text back.

Heading IDs are slugged from the rendered text with tags stripped, so a heading containing inline code or a link still produces a usable anchor.

The unknown-grammar catch keeps the escaped source. Letting the exception propagate would abort the whole parse over one fence with a typo in its language tag.

## Copy Buttons

Each fence's copy button is the kit's small secondary icon button in the block's top-right corner. It appears on hover, and it stays visible below `md`, where there is no hover.

Every button routes through one delegated `md-copy` action, so a re-render needs no re-binding and no guard against injecting a second button. On success the icon turns to a green check and the tooltip to `Copied` for a second and a half.

```javascript
ui.on('md-copy', async (b) => {
    const text = b.closest('.group').querySelector('code').textContent;
    try {
        await navigator.clipboard.writeText(text);
    } catch {
        const ta = Object.assign(document.createElement('textarea'), { value: text });
        ta.style.cssText = 'position:fixed;opacity:0';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        ta.remove();
    }
    b.innerHTML = '<i data-lucide="check" class="size-4 text-green"></i>';
    b.dataset.tip = 'Copied';
    ui.icons(b);
    setTimeout(() => {
        b.innerHTML = '<i data-lucide="copy" class="size-4"></i>';
        b.dataset.tip = 'Copy';
        ui.icons(b);
    }, 1500);
});
```

The `textarea` fallback covers pages served over plain HTTP, where `navigator.clipboard` is unavailable because the context is not secure.

## Document View

A document view renders into `<article class="markdown-body mx-auto max-w-3xl">` inside a `mantle` panel whose body is the scroll region. A line much past `max-w-3xl` is too long to read comfortably on a wide screen.

A Rendered and Source toggle sits in the page heading as a segmented control on the ground. The source view is the raw Markdown highlighted with the `markdown` grammar in a `base` well inside the same scroll region.

A long document gets an "On this page" aside built from the rendered `h2` and `h3` ids, with `h3` entries indented. Below `lg` the aside gives way to a Contents menu in the panel header, since a phone has no room for a second column.

The aside follows the scroll: the last heading above the top of the scroll region is the selected row at `bg-surface0 text-text`, and the rest are `text-overlay1` rows hovering at `surface0/50`. These are the kit's row states, so the aside reads as the same kind of list as every sidebar.

## Styles

Marked generates this subtree, so its classes are not yours to write and Tailwind utilities cannot reach it. Hand-written CSS is the correct tool here and is one of the named exceptions to the utility-first rule.

Every color reads a `--ctp-*` variable from the page palette rather than a literal. A hardcoded hex here is a color that stops following the theme the moment the page gains a light mode, and it is the one place a stale value survives a palette change unnoticed.

The rendered body sits inside a `mantle` panel, so a fenced block takes `base`, the well one rung further in. Every block radius is `1rem`, matching the kit's `rounded-2xl` wells.

```css
.markdown-body { color: var(--ctp-subtext0); line-height: 1.7; font-size: 15px; }
.markdown-body > :first-child { margin-top: 0; }
```

### Headings

Each level takes a distinct Catppuccin color so the outline is readable at a glance rather than by font size alone. H1 and H2 carry bottom borders, and the top three levels are set in the display face.

```css
.markdown-body h1, .markdown-body h2, .markdown-body h3 { margin-top: 28px; margin-bottom: 14px; font-family: 'Google Sans', sans-serif; font-weight: 500; line-height: 1.25; padding-bottom: 0.3em; scroll-margin-top: 16px; }
.markdown-body h1 { font-size: 2em; color: var(--ctp-lavender); border-bottom: 1px solid var(--ctp-surface0); }
.markdown-body h2 { font-size: 1.5em; color: var(--ctp-mauve); border-bottom: 1px solid color-mix(in oklab, var(--ctp-surface0) 50%, transparent); }
.markdown-body h3 { font-size: 1.25em; color: var(--ctp-blue); }
.markdown-body h4 { font-size: 1em; color: var(--ctp-text); font-weight: 600; margin: 20px 0 8px; }
```

`scroll-margin-top` keeps a heading reached from the aside clear of the scroll region's top edge.

### Text, Links, and Lists

```css
.markdown-body p { margin-bottom: 16px; }
.markdown-body strong { color: var(--ctp-text); font-weight: 600; }
.markdown-body a { color: var(--ctp-blue); text-decoration: none; }
.markdown-body a:hover { text-decoration: underline; }
.markdown-body ul, .markdown-body ol { padding-left: 1.75em; margin-bottom: 16px; }
.markdown-body ul { list-style-type: disc; }
.markdown-body ol { list-style-type: decimal; }
.markdown-body li { margin-bottom: 0.25em; }
.markdown-body li::marker { color: var(--ctp-overlay1); }
```

### Code

Inline code is peach on surface0, and a fenced block is `text` on base. The two need different treatments because inline code has to stand out inside a sentence while a block already stands out by being a block.

```css
.markdown-body code { font-family: 'JetBrains Mono', monospace; color: var(--ctp-peach); background-color: var(--ctp-surface0); border-radius: 6px; padding: 0.15em 0.4em; font-size: 0.875em; }
.markdown-body pre { position: relative; background-color: var(--ctp-base); border-radius: 1rem; padding: 1rem 1.25rem; margin-bottom: 16px; overflow: auto; }
.markdown-body pre code { color: var(--ctp-text); background-color: transparent; padding: 0; font-size: 0.8125rem; line-height: 1.65; }
```

`pre code` resets the inline rules, or every fenced block would render orange on a second background.

### Highlighting

Each Highlight.js token class reads a palette variable, so highlighted code re-themes with the page.

```css
.hljs { color: var(--ctp-text); background: transparent; }
.hljs-keyword, .hljs-selector-tag, .hljs-doctag { color: var(--ctp-mauve); }
.hljs-string, .hljs-regexp, .hljs-addition { color: var(--ctp-green); }
.hljs-number, .hljs-literal, .hljs-meta { color: var(--ctp-peach); }
.hljs-comment, .hljs-quote { color: var(--ctp-overlay2); font-style: italic; }
.hljs-title, .hljs-section, .hljs-title.function_ { color: var(--ctp-blue); }
.hljs-type, .hljs-title.class_, .hljs-class .hljs-title { color: var(--ctp-yellow); }
.hljs-attr, .hljs-attribute, .hljs-property, .hljs-name, .hljs-selector-class, .hljs-selector-id { color: var(--ctp-blue); }
.hljs-built_in, .hljs-deletion { color: var(--ctp-red); }
.hljs-params, .hljs-variable, .hljs-template-variable { color: var(--ctp-maroon); }
.hljs-symbol, .hljs-bullet, .hljs-link { color: var(--ctp-teal); }
.hljs-operator, .hljs-punctuation { color: var(--ctp-sky); }
.hljs-emphasis { font-style: italic; }
.hljs-strong { font-weight: 600; }
```

`background: transparent` lets the block's `base` well show through, and the same rules color a source view outside `.markdown-body`.

### Tables

```css
.markdown-body table { width: 100%; border-collapse: separate; border-spacing: 0; border: 1px solid color-mix(in oklab, var(--ctp-surface1) 50%, transparent); border-radius: 1rem; overflow: hidden; margin-bottom: 1.5rem; font-size: 14px; }
.markdown-body thead { background-color: color-mix(in oklab, var(--ctp-mauve) 10%, transparent); }
.markdown-body tr:nth-child(2n) { background-color: color-mix(in oklab, var(--ctp-surface0) 30%, transparent); }
.markdown-body th { color: var(--ctp-mauve); font-weight: 600; border-bottom: 1px solid color-mix(in oklab, var(--ctp-surface1) 50%, transparent); border-right: 1px solid color-mix(in oklab, var(--ctp-surface0) 50%, transparent); padding: 10px 16px; text-align: left; }
.markdown-body td { border-bottom: 1px solid color-mix(in oklab, var(--ctp-surface0) 30%, transparent); border-right: 1px solid color-mix(in oklab, var(--ctp-surface0) 30%, transparent); padding: 10px 16px; text-align: left; }
.markdown-body th:last-child, .markdown-body td:last-child { border-right: none; }
.markdown-body tr:last-child td { border-bottom: none; }
```

`border-collapse: separate` with `overflow: hidden` is what lets the rounded corners clip the header background; collapsed borders ignore the radius.

### Blockquotes, Rules, and Images

```css
.markdown-body blockquote { border-left: 0.25em solid color-mix(in oklab, var(--ctp-surface1) 60%, transparent); padding: 0 1em; margin-bottom: 16px; }
.markdown-body hr { border: none; border-top: 1px solid var(--ctp-surface0); margin: 1.5em 0; }
.markdown-body img { max-width: 100%; border-radius: 1rem; }
```

### Callouts

```css
.callout { display: flex; gap: 0.75rem; align-items: flex-start; padding: 1rem; border-radius: 1rem; margin-bottom: 1rem; background-color: color-mix(in oklab, var(--ctp-surface0) 20%, transparent); }
.callout-icon { display: inline-flex; flex-shrink: 0; padding-top: 0.3em; }
.callout-icon svg { width: 1em; height: 1em; }
.callout-content { flex: 1; min-width: 0; }
.callout-content p { margin: 0; }
.callout.tip { background-color: color-mix(in oklab, var(--ctp-green) 10%, transparent); } .callout.tip .callout-icon { color: var(--ctp-green); }
.callout.info { background-color: color-mix(in oklab, var(--ctp-blue) 10%, transparent); } .callout.info .callout-icon { color: var(--ctp-blue); }
.callout.danger { background-color: color-mix(in oklab, var(--ctp-red) 10%, transparent); } .callout.danger .callout-icon { color: var(--ctp-red); }
.callout.warning { background-color: color-mix(in oklab, var(--ctp-peach) 10%, transparent); } .callout.warning .callout-icon { color: var(--ctp-peach); }
.callout.note { background-color: color-mix(in oklab, var(--ctp-mauve) 10%, transparent); } .callout.note .callout-icon { color: var(--ctp-mauve); }
```

Each type tints its background at 10% opacity and saturates only the icon, so a page of callouts stays readable instead of turning into five blocks of solid color.
