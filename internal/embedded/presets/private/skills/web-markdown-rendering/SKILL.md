---
name: web-markdown-rendering
description: Rendering Markdown to styled HTML in a browser page with Marked, Highlight.js, and the Catppuccin palette. Use when a frontend displays notes, documentation, or any user-supplied Markdown, when adding syntax highlighting, callout blocks, or code copy buttons, or when styling rendered Markdown. Triggers on marked.parse, marked.use, hljs.highlight, .markdown-body, callout blockquotes such as [!TIP] and [!WARNING], and copy-to-clipboard buttons on code blocks.
user-invocable: false
---

# Web Markdown Rendering

**Marked parses, Highlight.js colors the code, Lucide draws the callout icons, and one CSS block styles everything in Catppuccin Mocha.**

The libraries are vendored and pinned, never loaded from a CDN at run time: `marked@18.0.9`, `highlight.js@11.12.0` with its `github-dark` theme.

```html
<script src="/static/js/marked.min.js"></script>
<script src="/static/js/highlight.min.js"></script>
<link rel="stylesheet" href="/static/css/github-dark.min.css">
```

## Call Order

The four steps run in this order after every content change, because each depends on the DOM the previous one produced.

```javascript
container.innerHTML = marked.parse(markdownSource);
addCopyButtons();
if (typeof mermaid !== 'undefined') {
    mermaid.initialize(mermaidConfig);
    mermaid.run({ nodes: container.querySelectorAll('.mermaid') });
}
lucide.createIcons();
```

`lucide.createIcons()` runs last and unconditionally, since it replaces `<i data-lucide>` placeholders with inline SVG and the callout renderer emits those placeholders on every parse.

## The Renderer

`marked.use({ renderer })` is called once at startup. Overriding four token types covers code fences, heading anchors, images, and callouts; everything else keeps Marked's default output.

```javascript
function generateId(text) {
    return String(text).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '');
}

function initMarked() {
    const renderer = {
        code(token) {
            const text = token.text;
            const language = token.lang;
            if (language === 'mermaid') {
                return `<div class="overflow-x-auto my-6"><div class="mermaid">${text}</div></div>`;
            }
            const validLang = hljs.getLanguage(language) ? language : 'plaintext';
            let highlighted = text;
            try {
                highlighted = hljs.highlight(text, { language: validLang }).value;
            } catch {
                // an unknown grammar falls back to the raw text rather than dropping the block
            }
            return `<pre><code class="hljs language-${validLang}">${highlighted}</code></pre>`;
        },
        heading(token) {
            const { tokens, depth } = token;
            const text = this.parser.parseInline(tokens);
            const slug = generateId(text.replace(/<[^>]*>/g, ''));
            return `<h${depth} id="${slug}">${text}</h${depth}>`;
        },
        image(token) {
            return `<img src="${token.href}" alt="${token.text || ''}" style="max-width:100%; border-radius:0.5rem;">`;
        },
        blockquote(token) {
            const body = this.parser.parse(token.tokens);
            const match = token.text.match(/^\[!(TIP|NOTE|INFO|WARNING|DANGER)\]/i);
            if (!match) {
                return `<blockquote>${body}</blockquote>`;
            }
            const type = match[1].toLowerCase();
            const iconMap = {
                tip: 'lightbulb',
                info: 'info',
                danger: 'triangle-alert',
                warning: 'triangle-alert',
                note: 'sticky-note',
            };
            const cleanBody = body.replace(/<p>\s*\[!(TIP|NOTE|INFO|WARNING|DANGER)\]\s*/i, '<p>');
            return `<div class="callout ${type}"><div class="callout-icon"><i data-lucide="${iconMap[type] || 'info'}"></i></div><div class="callout-content">${cleanBody}</div></div>`;
        },
    };
    marked.use({ renderer });
}
```

A `mermaid` fence becomes a `<div class="mermaid">` rather than a highlighted code block, because Mermaid's own renderer takes over that element afterwards.

Heading IDs are slugged from the rendered text with tags stripped, so a heading containing inline code or a link still produces a usable anchor.

The unknown-grammar catch keeps the original text. Letting the exception propagate would abort the whole parse over one fence with a typo in its language tag.

## Copy Buttons

A clipboard button is injected on each `<pre>` block. It appears on hover, confirms with a check icon for two seconds, then reverts.

```javascript
function addCopyButtons() {
    document.querySelectorAll('.markdown-body pre').forEach((block) => {
        if (block.querySelector('.copy-code-btn')) return;
        if (block.querySelector('.mermaid')) return;

        const codeEl = block.querySelector('code');
        if (!codeEl) return;

        const button = document.createElement('button');
        button.className = 'copy-code-btn';
        button.type = 'button';
        button.innerHTML = '<i data-lucide="copy" class="w-4 h-4"></i>';

        button.onclick = async (e) => {
            e.preventDefault();
            e.stopPropagation();
            try {
                await navigator.clipboard.writeText(codeEl.textContent);
            } catch {
                const textarea = document.createElement('textarea');
                textarea.value = codeEl.textContent;
                textarea.style.position = 'fixed';
                textarea.style.opacity = '0';
                document.body.appendChild(textarea);
                textarea.select();
                document.execCommand('copy');
                document.body.removeChild(textarea);
            }
            button.innerHTML = '<i data-lucide="check" class="w-4 h-4"></i>';
            button.classList.add('copied');
            lucide.createIcons({ nodes: [button] });
            setTimeout(() => {
                button.innerHTML = '<i data-lucide="copy" class="w-4 h-4"></i>';
                button.classList.remove('copied');
                lucide.createIcons({ nodes: [button] });
            }, 2000);
        };

        block.appendChild(button);
    });
    lucide.createIcons();
}
```

The early return on an existing button makes the function safe to call after every render, which it has to be, since re-parsing replaces the container's contents.

Mermaid blocks are skipped because a diagram's source is not what a reader wants on the clipboard.

The `textarea` fallback covers pages served over plain HTTP, where `navigator.clipboard` is unavailable because the context is not secure.

## Styles

```css
.markdown-body {
    background-color: transparent !important;
    font-family: 'Inter', sans-serif !important;
    color: #a6adc8 !important;
    line-height: 1.6;
    font-size: 16px;
}
```

### Headings

Each level takes a distinct Catppuccin color so the outline is readable at a glance rather than by font size alone. H1 and H2 carry bottom borders.

```css
.markdown-body h1, .markdown-body h2, .markdown-body h3 {
    margin-top: 24px;
    margin-bottom: 16px;
    font-weight: 600;
    line-height: 1.25;
    padding-bottom: 0.3em;
}
.markdown-body h1 { font-size: 2em;     color: #b4befe !important; border-bottom: 1px solid #313244; }
.markdown-body h2 { font-size: 1.5em;   color: #cba6f7 !important; border-bottom: 1px solid rgba(49, 50, 68, 0.5); }
.markdown-body h3 { font-size: 1.25em;  color: #89b4fa !important; }
.markdown-body h4 { font-size: 1em;     color: #cdd6f4; font-weight: 600; }
.markdown-body h5 { font-size: 0.875em; color: #cdd6f4; font-weight: 600; }
.markdown-body h6 { font-size: 0.85em;  color: #a6adc8; }
```

### Text, links, and lists

```css
.markdown-body p { margin-bottom: 16px; }
.markdown-body a { color: #89b4fa; text-decoration: none; }
.markdown-body a:hover { text-decoration: underline; }

.markdown-body ul, .markdown-body ol { padding-left: 2em; margin-bottom: 16px; }
.markdown-body ul { list-style-type: disc; }
.markdown-body ol { list-style-type: decimal; }
.markdown-body li { margin-bottom: 0.25em; }
```

### Code

Inline code is peach on surface0, and a fenced block is plain text on mantle. The two need different treatments because inline code has to stand out inside a sentence while a block already stands out by being a block.

```css
.markdown-body code {
    font-family: 'JetBrains Mono', monospace;
    color: #fab387 !important;
    background-color: #313244 !important;
    border-radius: 4px;
    padding: 0.2em 0.4em;
    font-size: 0.9375em;
}
.markdown-body pre {
    position: relative;
    background-color: #181825 !important;
    border-radius: 0.75rem;
    padding: 1rem !important;
    margin-bottom: 16px;
    overflow: auto;
}
.markdown-body pre code {
    color: inherit !important;
    background-color: transparent !important;
    padding: 0;
    font-size: 0.9375em;
}
```

`pre code` resets the inline rules, or every fenced block would render orange on a second background.

### Tables

```css
.markdown-body table {
    display: table !important;
    width: 100% !important;
    border-collapse: separate;
    border-spacing: 0;
    border: 1px solid rgba(69, 71, 90, 0.5);
    border-radius: 8px;
    overflow: hidden;
    margin-bottom: 1.5rem;
}
.markdown-body table thead { background-color: rgba(203, 166, 247, 0.1); }
.markdown-body table tr { background-color: transparent !important; border: none !important; }
.markdown-body table tr:nth-child(2n) { background-color: rgba(49, 50, 68, 0.3) !important; }
.markdown-body table th {
    color: #cba6f7 !important;
    font-weight: 600;
    border: none !important;
    border-bottom: 1px solid rgba(69, 71, 90, 0.5) !important;
    border-right: 1px solid rgba(49, 50, 68, 0.5);
    padding: 12px 16px !important;
    text-align: left;
}
.markdown-body table td {
    border: none !important;
    border-bottom: 1px solid rgba(49, 50, 68, 0.3) !important;
    border-right: 1px solid rgba(49, 50, 68, 0.3);
    color: #a6adc8 !important;
    padding: 12px 16px !important;
    text-align: left;
}
.markdown-body table th:last-child, .markdown-body table td:last-child { border-right: none; }
.markdown-body table tr:last-child td { border-bottom: none !important; }
```

`border-collapse: separate` with `overflow: hidden` is what lets the rounded corners clip the header background; collapsed borders ignore the radius.

### Blockquotes, rules, and copy buttons

```css
.markdown-body blockquote {
    border-left: 0.25em solid rgba(69, 71, 90, 0.5);
    padding: 0 1em;
    color: #a6adc8;
    margin-bottom: 16px;
}
.markdown-body hr { border: none; border-top: 1px solid #313244; margin: 1.5em 0; }

.copy-code-btn {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    padding: 0.5rem;
    background-color: rgba(49, 50, 68, 0.95);
    border-radius: 0.375rem;
    color: #a6adc8;
    cursor: pointer;
    opacity: 0;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
}
pre:hover .copy-code-btn { opacity: 1; }
.copy-code-btn:hover { background-color: rgba(69, 71, 90, 1); color: #cba6f7; }
.copy-code-btn.copied { color: #a6e3a1; }
```

### Callouts

```css
.callout {
    padding: 1rem;
    border-radius: 0.5rem;
    margin-bottom: 1rem;
    display: flex;
    gap: 0.75rem;
    align-items: flex-start;
    background-color: rgba(49, 50, 68, 0.2);
}
.callout-icon {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    line-height: 1;
    padding-top: 0.3em;
}
.callout-icon svg { width: 1em; height: 1em; }
.callout-content { flex: 1; }
.callout-content p { margin: 0 !important; }

.callout.tip     { background-color: rgba(166, 227, 161, 0.1); }
.callout.tip     .callout-icon { color: #a6e3a1; }
.callout.info    { background-color: rgba(137, 180, 250, 0.1); }
.callout.info    .callout-icon { color: #89b4fa; }
.callout.danger  { background-color: rgba(243, 139, 168, 0.1); }
.callout.danger  .callout-icon { color: #f38ba8; }
.callout.warning { background-color: rgba(250, 179, 135, 0.1); }
.callout.warning .callout-icon { color: #fab387; }
.callout.note    { background-color: rgba(203, 166, 247, 0.1); }
.callout.note    .callout-icon { color: #cba6f7; }
```

Each type tints its background at 10% opacity and saturates only the icon, so a page of callouts stays readable instead of turning into five blocks of solid color.

### Scrollbars

```css
::-webkit-scrollbar { width: 8px; }
::-webkit-scrollbar-track { background: #11111b; }
::-webkit-scrollbar-thumb { background: #313244; border-radius: 4px; }
::-webkit-scrollbar-thumb:hover { background: #45475a; }
```
