# Document View

**A rendered document reads at `max-w-3xl` inside a `mantle` panel, beside an outline that follows the scroll and a toggle to its highlighted source.**

The view is a page in the frontend's router, built from the page kit in `go-embedded-frontend` and rendered through `md.render`. `md.sample` stands for the document's Markdown source, and the path and size in the panel header are that file's.

## The View

```javascript
const markdownPage = {
    nav: ['Markdown', 'file-text'],
    mode: 'rendered',

    render() {
        return `<div class="flex h-full min-h-0 flex-col">
            ${pageHead('Markdown', ui.segmented({ name: 'md-mode', value: markdownPage.mode, ground: true, options: [{ value: 'rendered', label: 'Rendered', icon: 'eye', compact: true }, { value: 'source', label: 'Source', icon: 'code', compact: true }] }))}
            <div class="flex min-h-0 flex-1 gap-4">
                <aside class="flex min-h-0 w-60 shrink-0 flex-col max-lg:hidden">
                    <p class="${ui.cls.group} !pt-0">On this page</p>
                    <nav data-toc data-scroll class="min-h-0 flex-1 overflow-y-auto pb-2"></nav>
                </aside>
                <section class="flex min-w-0 flex-1 flex-col rounded-3xl bg-mantle">
                    <header class="flex shrink-0 items-center gap-3 px-3 pt-3 pb-2 sm:px-4">
                        ${ui.tile('file-text', 'blue', 'sm')}
                        <span class="min-w-0 flex-1 truncate font-mono text-sm text-text">docs/orbit.md</span>
                        ${ui.chip('4.1 KB', { mono: true })}
                        <span class="lg:hidden">${ui.iconButton('list', { tip: 'Contents', attrs: 'data-action="md-contents"' })}</span>
                    </header>
                    <div data-doc data-scroll class="relative min-h-0 flex-1 overflow-y-auto px-5 pt-2 pb-8 sm:px-8">
                        <article class="markdown-body mx-auto max-w-3xl"></article>
                        <pre data-source class="mx-auto hidden max-w-3xl overflow-x-auto ${ui.cls.well} p-5 font-mono text-[13px] leading-relaxed whitespace-pre-wrap"><code class="hljs">${hljs.highlight(md.sample, { language: 'markdown' }).value}</code></pre>
                    </div>
                </section>
            </div>
        </div>`;
    },
```

The document renders into `<article class="markdown-body mx-auto max-w-3xl">` inside a `mantle` panel whose body is the scroll region. A line much past `max-w-3xl` is too long to read comfortably on a wide screen.

A Rendered and Source toggle sits in the page head as a pill track on the ground. The source view is the raw Markdown highlighted with the `markdown` grammar, in a `base` well inside the same scroll region.

A long document gets an "On this page" aside built from the rendered `h2` and `h3` ids, with `h3` entries indented. Below `lg` the aside gives way to a Contents menu in the panel header, since a phone has no room for a second column.

## Outline and Toggle

```javascript
    mount(el) {
        const doc = el.querySelector('[data-doc]');
        const article = el.querySelector('article');
        const toc = el.querySelector('[data-toc]');
        md.render(article, md.sample);

        const heads = [...article.querySelectorAll('h2, h3')];
        toc.innerHTML = heads.map((h) => `<button type="button" data-goto="${h.id}" class="flex min-h-8 w-full items-center rounded-xl py-1.5 pr-3 text-left text-sm transition-colors ${h.tagName === 'H3' ? 'pl-7 text-[13px]' : 'pl-3'}">${ui.esc(h.textContent)}</button>`).join('');

        const go = (id) => {
            const h = article.querySelector(`#${CSS.escape(id)}`);
            doc.scrollTo({ top: h.offsetTop - 16, behavior: 'smooth' });
        };
        const spy = () => {
            const visible = article.offsetParent !== null;
            let active = heads[0];
            for (const h of heads) if (h.offsetTop - doc.scrollTop <= 48) active = h;
            if (doc.scrollTop + doc.clientHeight >= doc.scrollHeight - 2) active = heads.at(-1);
            toc.querySelectorAll('[data-goto]').forEach((b) => {
                const on = visible && b.dataset.goto === active.id;
                b.classList.toggle('bg-surface0', on);
                b.classList.toggle('text-text', on);
                b.classList.toggle('text-overlay1', !on);
                b.classList.toggle('hover:text-text', !on);
                b.classList.toggle('hover:bg-surface0/50', !on);
            });
        };
        toc.addEventListener('click', (e) => {
            const b = e.target.closest('[data-goto]');
            if (b) go(b.dataset.goto);
        });
        doc.addEventListener('scroll', spy, { passive: true });
        spy();

        const setMode = (mode) => {
            markdownPage.mode = mode;
            article.classList.toggle('hidden', mode !== 'rendered');
            el.querySelector('[data-source]').classList.toggle('hidden', mode === 'rendered');
            doc.scrollTop = 0;
            spy();
        };
        setMode(markdownPage.mode);
        el.addEventListener('ui-change', (e) => { if (e.detail.name === 'md-mode') setMode(e.detail.value); });

        ui.on('md-contents', (b) => ui.menu(b, heads.map((h) => ({ icon: h.tagName === 'H2' ? 'hash' : 'corner-down-right', label: h.textContent, onSelect: () => {
            if (markdownPage.mode !== 'rendered') el.querySelector('[data-ui="radio"][data-value="rendered"]').click();
            go(h.id);
        } }))));
    },

    theme(el) {
        md.mermaid(el.querySelector('article'));
    },
};
```

The outline follows the scroll. The selected row is the last heading within 48px of the top, or the last heading once the document reaches its end, at `bg-surface0 text-text`. The rest are `text-overlay1` rows hovering at `surface0/50`, the kit's row states, so the outline reads as the same kind of list as every sidebar.

Switching modes resets the scroll, and a heading picked from the Contents menu switches back to Rendered first.

The view's `theme` hook runs the diagrams again, since Mermaid's SVG keeps the colors it was drawn with.
