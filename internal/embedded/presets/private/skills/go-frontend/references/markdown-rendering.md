# Markdown Rendering

Complete patterns for rendering Markdown in Go embedded frontends. Uses Marked.js for parsing, Highlight.js for syntax highlighting, and Lucide for callout icons.

## Required Assets

| Asset | Source | Location |
|-------|--------|----------|
| Marked.js | `cdn.jsdelivr.net/npm/marked@latest` | `js/marked.min.js` |
| Highlight.js | `cdn.jsdelivr.net/gh/highlightjs/cdn-release@latest/build` | `js/highlight.min.js` |
| Highlight.js Theme | Same CDN, `styles/github-dark.min.css` | `css/github-dark.min.css` |
| JetBrains Mono | `fonts.googleapis.com` | `css/jetbrains-mono.css`, `fonts/*.ttf` |

HTML includes:

```html
<script src="/static/js/marked.min.js"></script>
<script src="/static/js/highlight.min.js"></script>
<link rel="stylesheet" href="/static/css/github-dark.min.css">
```

## Marked.js Custom Renderer

Initialize before first use. Handles code blocks (with Highlight.js + Mermaid detection), heading IDs, inline images, and callout blockquotes.

```javascript
function generateId(text) {
    return String(text).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '');
}

function initMarked() {
    const renderer = {
        code(token) {
            const text = token.text;
            const language = token.lang;
            // Mermaid blocks rendered as divs (processed later by mermaid.run())
            if (language === 'mermaid') {
                return `<div class="overflow-x-auto my-6"><div class="mermaid">${text}</div></div>`;
            }
            const validLang = hljs.getLanguage(language) ? language : 'plaintext';
            let highlighted = text;
            try { highlighted = hljs.highlight(text, { language: validLang }).value; } catch (e) {}
            return `<pre><code class="hljs language-${validLang}">${highlighted}</code></pre>`;
        },
        heading(token) {
            const { tokens, depth } = token;
            const text = this.parser.parseInline(tokens);
            const slug = generateId(text.replace(/<[^>]*>/g, ''));
            return `<h${depth} id="${slug}">${text}</h${depth}>`;
        },
        image(token) {
            const alt = token.text || '';
            return `<img src="${token.href}" alt="${alt}" style="max-width:100%; border-radius:0.5rem;">`;
        },
        blockquote(token) {
            const body = this.parser.parse(token.tokens);
            const rawText = token.text;
            // Detect callout syntax: > [!TIP], > [!NOTE], > [!INFO], > [!WARNING], > [!DANGER]
            const match = rawText.match(/^\[!(TIP|NOTE|INFO|WARNING|DANGER)\]/i);
            if (match) {
                const type = match[1].toLowerCase();
                const iconMap = {
                    tip: 'lightbulb',
                    info: 'info',
                    danger: 'triangle-alert',
                    warning: 'triangle-alert',
                    note: 'sticky-note'
                };
                const cleanBody = body.replace(/<p>\s*\[!(TIP|NOTE|INFO|WARNING|DANGER)\]\s*/i, '<p>');
                return `<div class="callout ${type}"><div class="callout-icon"><i data-lucide="${iconMap[type] || 'info'}"></i></div><div class="callout-content">${cleanBody}</div></div>`;
            }
            return `<blockquote>${body}</blockquote>`;
        }
    };
    marked.use({ renderer });
}
```

## Usage Flow

After loading/updating markdown content:

```javascript
// 1. Parse markdown to HTML
container.innerHTML = marked.parse(markdownSource);

// 2. Add copy buttons to code blocks
addCopyButtons();

// 3. Render Mermaid diagrams (if mermaid.js loaded)
if (typeof mermaid !== 'undefined') {
    mermaid.initialize(mermaidConfig);  // see mermaid-config reference
    mermaid.run({ nodes: container.querySelectorAll('.mermaid') });
}

// 4. Initialize Lucide icons (for callout icons)
lucide.createIcons();
```

## Code Copy Buttons

Injects a copy-to-clipboard button on each `<pre>` block (excluding Mermaid). Uses Lucide icons with a 2-second success state.

### JavaScript

```javascript
function addCopyButtons() {
    const codeBlocks = document.querySelectorAll('.markdown-body pre');
    codeBlocks.forEach(block => {
        if (block.querySelector('.copy-code-btn')) return;
        if (block.querySelector('.mermaid')) return;

        const codeEl = block.querySelector('code');
        if (!codeEl) return;

        block.style.position = 'relative';
        const button = document.createElement('button');
        button.className = 'copy-code-btn';
        button.type = 'button';
        button.innerHTML = '<i data-lucide="copy" class="w-4 h-4"></i>';

        button.onclick = async function(e) {
            e.preventDefault();
            e.stopPropagation();

            try {
                await navigator.clipboard.writeText(codeEl.textContent);
            } catch (err) {
                // Fallback for non-secure contexts
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

### CSS

```css
.copy-code-btn {
    position: absolute; top: 0.5rem; right: 0.5rem;
    padding: 0.5rem;
    background-color: rgba(49, 50, 68, 0.95);
    border-radius: 0.375rem;
    color: #a6adc8;
    cursor: pointer;
    opacity: 0;
    transition: all 0.2s ease;
    display: flex; align-items: center; justify-content: center;
    z-index: 10;
}
pre { position: relative; }
pre:hover .copy-code-btn { opacity: 1; }
.copy-code-btn:hover { background-color: rgba(69, 71, 90, 1); color: #cba6f7; }
.copy-code-btn.copied { color: #a6e3a1; }
```

## Markdown Element Styles

Complete Catppuccin Mocha styling for rendered markdown. Place inside `<style>` in HTML.

### Container

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

Each heading level uses a distinct Catppuccin color. H1 and H2 have bottom borders.

```css
.markdown-body h1, .markdown-body h2, .markdown-body h3 {
    margin-top: 24px;
    margin-bottom: 16px;
    font-weight: 600;
    line-height: 1.25;
    padding-bottom: 0.3em;
}
.markdown-body h1 { border-bottom: 1px solid #313244; }
.markdown-body h2 { border-bottom: 1px solid rgba(49, 50, 68, 0.5); }

.markdown-body h1 { font-size: 2em; color: #b4befe !important; }       /* lavender */
.markdown-body h2 { font-size: 1.5em; color: #cba6f7 !important; }     /* mauve */
.markdown-body h3 { font-size: 1.25em; color: #89b4fa !important; }    /* blue */
.markdown-body h4 { font-size: 1em; font-weight: 600; color: #cdd6f4; }
.markdown-body h5 { font-size: 0.875em; font-weight: 600; color: #cdd6f4; }
.markdown-body h6 { font-size: 0.85em; color: #a6adc8; }
```

### Text and Links

```css
.markdown-body p { margin-bottom: 16px; }
.markdown-body a { color: #89b4fa; text-decoration: none; }
.markdown-body a:hover { text-decoration: underline; }
```

### Code

Inline code uses peach on surface0. Code blocks use mantle background. `pre code` resets inline styles.

```css
.markdown-body code {
    font-family: 'JetBrains Mono', monospace;
    color: #fab387 !important;                  /* peach */
    background-color: #313244 !important;       /* surface0 */
    border-radius: 4px;
    padding: 0.2em 0.4em;
    font-size: 0.9375em;
}
.markdown-body pre {
    background-color: #181825 !important;       /* mantle */
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

### Lists

```css
.markdown-body ul, .markdown-body ol {
    padding-left: 2em;
    margin-bottom: 16px;
}
.markdown-body ul { list-style-type: disc; }
.markdown-body ol { list-style-type: decimal; }
.markdown-body li { margin-bottom: 0.25em; }
```

### Tables

Rounded corners, mauve header background, striped rows, subtle borders.

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
    color: #cba6f7 !important; font-weight: 600;
    border: none !important; border-bottom: 1px solid rgba(69, 71, 90, 0.5) !important;
    border-right: 1px solid rgba(49, 50, 68, 0.5); padding: 12px 16px !important; text-align: left;
}
.markdown-body table td {
    border: none !important; border-bottom: 1px solid rgba(49, 50, 68, 0.3) !important;
    border-right: 1px solid rgba(49, 50, 68, 0.3); color: #a6adc8 !important;
    padding: 12px 16px !important; text-align: left;
}
.markdown-body table th:last-child, .markdown-body table td:last-child { border-right: none; }
.markdown-body table tr:last-child td { border-bottom: none !important; }
```

### Blockquotes

```css
.markdown-body blockquote {
    border-left: 0.25em solid rgba(69, 71, 90, 0.5);
    padding: 0 1em;
    color: #a6adc8;
    margin-bottom: 16px;
}
```

### Horizontal Rules

```css
.markdown-body hr {
    border: none;
    border-top: 1px solid #313244;
    margin: 1.5em 0;
}
```

### Callout Blocks

Styled alert boxes for `> [!TYPE]` syntax. Uses Lucide icons and type-specific background colors at 10% opacity.

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

/* Type-specific colors */
.callout.tip     { background-color: rgba(166, 227, 161, 0.1); }   /* green */
.callout.tip     .callout-icon { color: #a6e3a1; }
.callout.info    { background-color: rgba(137, 180, 250, 0.1); }   /* blue */
.callout.info    .callout-icon { color: #89b4fa; }
.callout.danger  { background-color: rgba(243, 139, 168, 0.1); }   /* red */
.callout.danger  .callout-icon { color: #f38ba8; }
.callout.warning { background-color: rgba(250, 179, 135, 0.1); }   /* peach */
.callout.warning .callout-icon { color: #fab387; }
.callout.note    { background-color: rgba(203, 166, 247, 0.1); }   /* mauve */
.callout.note    .callout-icon { color: #cba6f7; }
```

### Scrollbars

```css
::-webkit-scrollbar { width: 8px; }
::-webkit-scrollbar-track { background: #11111b; }         /* crust */
::-webkit-scrollbar-thumb { background: #313244; border-radius: 4px; }  /* surface0 */
::-webkit-scrollbar-thumb:hover { background: #45475a; }   /* surface1 */
```
