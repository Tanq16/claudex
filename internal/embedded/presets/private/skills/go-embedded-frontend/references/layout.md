# Layout

**The page never scrolls: the header holds still, `<main>` fills the rest, and each region that overflows scrolls on its own behind a drawn thumb.**

## The Shell

The body is an `h-dvh overflow-hidden` column, the header is `shrink-0`, and `<main>` is `flex-1 min-h-0`, so the header and the nav hold still at every size.

A gallery or settings view gets one scroll region filling `<main>`, and a workspace view gets side-by-side panels that each scroll on their own. One page-level scroll in a workspace would push a sibling panel off screen whenever a list grows.

The router keeps one `pages` object mapping a route to `{ nav: [label, icon], render(), mount(el), leave(), theme(el) }`, where `nav` and the last three are optional. A route change replaces `#view` with a fresh clone, which drops every listener the last view attached to it, and then renders, mounts, and draws the nav.

A route is `#/page` or `#/page/param`. `app.param` holds the decoded rest of the path, for a page showing one item such as a note or a task being edited. A page without `nav` is reachable by its route and stays out of the nav.

```javascript
const app = {
    current: null,
    param: '',

    route() {
        const [name, ...rest] = location.hash.replace(/^#\/?/, '').split('/');
        if (app.current) pages[app.current].leave?.();
        app.current = pages[name] ? name : 'foundations';
        app.param = decodeURIComponent(rest.join('/'));
        const old = document.getElementById('view');
        const el = old.cloneNode(false);
        old.replaceWith(el);
        ui.popover.close();
        ui.tooltip.hide();
        el.innerHTML = pages[app.current].render();
        pages[app.current].mount?.(el);
        app.renderNav();
        ui.icons(el);
    },

    renderNav() {
        const nav = document.getElementById('nav');
        nav.innerHTML = Object.entries(pages).filter(([, p]) => p.nav).map(([key, p]) => {
            const on = app.current === key;
            const [label, icon] = p.nav;
            return `<a href="#/${key}" class="flex h-9 items-center gap-2 rounded-full px-4 text-sm whitespace-nowrap transition-colors max-sm:flex-1 max-sm:justify-center max-sm:px-3 ${on ? 'bg-surface0 text-text' : 'text-overlay1 hover:text-text'}" ${on ? 'aria-current="page"' : `data-tip="${label}" data-tip-side="bottom"`}><i data-lucide="${icon}" class="size-4 ${on ? 'text-mauve' : ''}"></i><span class="${on ? 'max-sm:hidden' : 'max-xl:hidden'}">${label}</span></a>`;
        }).join('');
        ui.icons(nav);
    },

    renderTheme() {
        const light = ui.theme.light();
        const host = document.getElementById('theme-toggle');
        host.innerHTML = ui.iconButton(light ? 'moon' : 'sun', { tip: light ? 'Dark mode' : 'Light mode', attrs: 'data-action="theme-toggle" data-tip-side="bottom"' });
        ui.icons(host);
    },
};
```

The nav is one `mantle` pill group. The current link is a `surface0` pill with a mauve icon, and every other link is an `overlay1` icon whose label shows from `xl` up and as a tooltip below it. Below `sm` the nav takes its own full-width row under the logo. Its links share that row equally, so two pages read as a pair rather than as two icons pinned to opposite edges.

Boot wires the delegation, the tooltips, and the scroll observer once, then routes. `renderTheme` runs only when the app has a light mode.

```javascript
ui.initDelegation();
ui.tooltip.init();
ui.initScrollObserver();
window.addEventListener('hashchange', () => app.route());
app.renderTheme();
app.route();
```

## Page Heads and Panels

A page head is the title on the left and either a pill track on the ground or the page's actions on the right, sharing the header's gutters. A panel is a `mantle` card with a `font-display text-[16px]` title.

A gallery is one scroll region holding the page head and a grid of panels that adds a column at `lg`, `2xl`, and `3xl`, so a wide monitor fills with panels rather than stretching two.

```javascript
const card = (title, body, { span = '', source = '' } = {}) => `<section class="${ui.cls.panel} flex min-w-0 flex-col gap-4 p-5 ${span}">
    <header class="flex items-center gap-2"><h2 class="font-display text-[16px] font-medium text-text">${title}</h2>${source ? `<span class="ml-auto font-mono text-[11px] text-overlay0">${source}</span>` : ''}</header>
    ${body}
</section>`;

const pageHead = (title, right = '') => `<div class="mb-5 flex flex-wrap items-center justify-between gap-3"><h1 class="font-display text-2xl text-text">${title}</h1>${right}</div>`;
const gallery = (title, right, sections) => `<div data-scroll="lane" class="-mr-3.5 h-full overflow-y-auto">${pageHead(title, right)}<div class="grid items-start gap-4 lg:grid-cols-2 2xl:grid-cols-3 3xl:grid-cols-4">${sections.join('')}</div></div>`;
```

## Scroll Regions

Native scrollbars are hidden page-wide, and every scroll region carries `data-scroll` and gets a drawn thumb. A native bar takes layout width on one platform and overlays content on another, and it cannot be inset from a rounded corner.

A scroll region's thumbs float over its content, so the region keeps its size whether or not it overflows. A file tree that grows and collapses never reflows its rows as its thumb comes and goes, and a PDF, an image, a diff, or a code view runs to the edge with the thumb over it.

A region whose content lines up against its right edge takes `data-scroll="lane"`, which reserves a constant 14px lane on the right for the vertical thumb. A thumb floating there would cut across a table's last column, a list's right-aligned counts, times, or row actions, or a transcript's right-aligned bubbles. The lane counts the region's right padding toward its 14px, so a region padded that much already looks the same either way.

A lane region flush against its parent's right edge pulls out by the lane with `-mr-3.5`, which keeps its content column aligned with the page.

The thumb sits 4px from the region's edge as `w-1.5 rounded-full bg-overlay0/60`. It widens to `w-2 bg-overlay1` under the pointer and fades 900ms after the last scroll, unless the pointer is on it. A faded thumb takes no pointer events, so it never swallows a click on the content under it. The pointer coming within 16px of the region's right or bottom edge shows it again, ready to drag. Its track is inset at least 6px top and bottom, and by half the corner radius on a rounded region or parent, so the thumb never pokes past a curve.

The thumb shows on a scroll only when the user caused it: within a second of a wheel, touch, or key event on the region, a key press with focus on the page body, or while dragging. A scroll the app makes, such as keeping a chat pinned to its newest message, re-places the thumb without showing it, so streaming content never flashes a thumb nobody asked for.

A region that overflows sideways also gets a horizontal thumb, `h-1.5` along its bottom edge, which always floats. A bottom lane would exist only while the content overflows, so the region would change height as it appeared. Only a computed `overflow-x` of `auto` or `scroll` counts, because a region clipping with `hidden` cannot be scrolled sideways by the user.

A region carrying `data-scroll-cue` keeps its horizontal thumb showing for as long as it overflows sideways, and re-places that thumb when its parent resizes. A wide code block, table, or diagram without the cue reads as clipped at the column's edge. The vertical thumb still fades as usual.

When a region scrolls both ways, each thumb's track stops 14px short of the other's edge, so the two never meet in the corner.

The observer attaches only to added nodes still in the document. A node added and removed within one task still appears in the mutation records, and a detached node has no layout for its thumb to measure against.

The thumb is positioned against the region's parent, which `ui.anchor` makes `relative` only when the class list carries no position utility. It reads class names rather than computed style, because the browser build compiles classes after the script runs and computed style still reads `static` for an element about to become `fixed`.

```javascript
keyedAt: 0,

initScrollObserver() {
    document.addEventListener('keydown', (e) => {
        if (e.target === document.body) ui.keyedAt = performance.now();
    }, true);
    const attach = (root) => {
        if (root.matches?.('[data-scroll]')) ui.scrollbar(root);
        root.querySelectorAll?.('[data-scroll]').forEach((el) => ui.scrollbar(el));
    };
    attach(document.body);
    new MutationObserver((records) => {
        for (const r of records) r.addedNodes.forEach((n) => n.nodeType === 1 && n.isConnected && attach(n));
    }).observe(document.body, { childList: true, subtree: true });
},

scrollLane: 14,

anchor(el) {
    if (!/(^|\s)(relative|absolute|fixed|sticky)(\s|$)/.test(el.className)) el.classList.add('relative');
},

scrollbar(el) {
    if (el._scrollbar) return;
    el._scrollbar = true;
    const host = el.parentElement;
    ui.anchor(el);
    ui.anchor(host);
    const thumb = (cls) => {
        const t = document.createElement('span');
        t.setAttribute('aria-hidden', 'true');
        t.className = `absolute z-20 rounded-full bg-overlay0/60 opacity-0 transition-[opacity,width,height,background-color] duration-300 hover:bg-overlay1 cursor-default ${cls}`;
        t.style.pointerEvents = 'none';
        host.appendChild(t);
        return t;
    };
    const ty = thumb('top-0 w-1.5 hover:w-2');
    const tx = thumb('left-0 h-1.5 hover:h-2');
    const grip = el.tagName === 'TEXTAREA' ? host.querySelector('[data-ui="grip"]') : null;
    const cue = el.hasAttribute('data-scroll-cue');
    let hideTimer;
    let dragging = null;
    let lane = -1;
    let tall = false;
    let wide = false;
    let track = [6, 6];
    let trackX = [6, 6];

    const fit = () => {
        const px = (node, prop) => parseFloat(getComputedStyle(node)[prop]) || 0;
        const want = el.dataset.scroll === 'lane' ? Math.max(0, ui.scrollLane - px(el, 'paddingRight')) : 0;
        if (want !== lane) {
            lane = want;
            el.style.borderRight = lane ? `${lane}px solid transparent` : '';
        }
        wide = ['auto', 'scroll'].includes(getComputedStyle(el).overflowX) && el.scrollWidth - el.clientWidth > 1;
        tall = el.scrollHeight - el.clientHeight > 1;
        const above = el.offsetTop;
        const below = host.clientHeight - el.offsetTop - el.offsetHeight;
        const corner = Math.max(px(el, 'borderBottomRightRadius') / 2, px(host, 'borderBottomRightRadius') / 2 - below);
        track = [
            Math.max(6, px(el, 'borderTopRightRadius') / 2, px(host, 'borderTopRightRadius') / 2 - above),
            Math.max(6, corner, wide ? ui.scrollLane : 0, grip ? el.offsetTop + el.offsetHeight - grip.offsetTop + 4 : 0),
        ];
        trackX = [
            Math.max(6, px(el, 'borderBottomLeftRadius') / 2, px(host, 'borderBottomLeftRadius') / 2 - below),
            Math.max(6, corner, tall ? ui.scrollLane - lane : 0),
        ];
    };

    const place = () => {
        fit();
        ty.style.display = tall ? '' : 'none';
        tx.style.display = wide ? '' : 'none';
        if (tall) {
            const { scrollTop, scrollHeight, clientHeight } = el;
            const room = clientHeight - track[0] - track[1];
            const h = Math.max(28, (clientHeight / scrollHeight) * room);
            ty.style.right = `${host.clientWidth - el.offsetLeft - el.offsetWidth + 4}px`;
            ty.style.height = `${h}px`;
            ty.style.transform = `translateY(${el.offsetTop + track[0] + (scrollTop / (scrollHeight - clientHeight)) * (room - h)}px)`;
        }
        if (wide) {
            const { scrollLeft, scrollWidth, clientWidth } = el;
            const room = clientWidth - trackX[0] - trackX[1];
            const w = Math.max(28, (clientWidth / scrollWidth) * room);
            tx.style.bottom = `${host.clientHeight - el.offsetTop - el.offsetHeight + 4}px`;
            tx.style.width = `${w}px`;
            tx.style.transform = `translateX(${el.offsetLeft + trackX[0] + (scrollLeft / (scrollWidth - clientWidth)) * (room - w)}px)`;
            if (cue) {
                tx.style.opacity = '1';
                tx.style.pointerEvents = '';
            }
        }
        return tall || wide;
    };
    const hide = () => {
        if (dragging || ty.matches(':hover') || tx.matches(':hover')) return;
        for (const t of cue && wide ? [ty] : [ty, tx]) {
            t.style.opacity = '0';
            t.style.pointerEvents = 'none';
        }
    };
    const show = () => {
        if (!place()) return;
        for (const t of [ty, tx]) {
            t.style.opacity = '1';
            t.style.pointerEvents = '';
        }
        clearTimeout(hideTimer);
        hideTimer = setTimeout(hide, 900);
    };
    let touched = 0;
    const touch = () => { touched = performance.now(); };
    el.addEventListener('wheel', touch, { passive: true });
    el.addEventListener('touchmove', touch, { passive: true });
    el.addEventListener('touchend', touch, { passive: true });
    el.addEventListener('keydown', touch);
    el.addEventListener('scroll', () => (dragging || performance.now() - Math.max(touched, ui.keyedAt) < 1000 ? show() : place()), { passive: true });
    el.addEventListener('pointerenter', show);
    el.addEventListener('pointermove', (e) => {
        const r = el.getBoundingClientRect();
        if (r.right - e.clientX < 16 || r.bottom - e.clientY < 16) show();
    });
    const resized = new ResizeObserver(() => place());
    resized.observe(el);
    if (cue) resized.observe(host);
    new MutationObserver(() => place()).observe(el, { childList: true, subtree: true });

    const drag = (t, vertical) => {
        t.addEventListener('pointerdown', (e) => {
            e.preventDefault();
            e.stopPropagation();
            t.setPointerCapture(e.pointerId);
            dragging = vertical ? { at: e.clientY, from: el.scrollTop } : { at: e.clientX, from: el.scrollLeft };
        });
        t.addEventListener('pointermove', (e) => {
            if (!dragging) return;
            if (vertical) el.scrollTop = dragging.from + ((e.clientY - dragging.at) * (el.scrollHeight - el.clientHeight)) / (el.clientHeight - t.offsetHeight - track[0] - track[1]);
            else el.scrollLeft = dragging.from + ((e.clientX - dragging.at) * (el.scrollWidth - el.clientWidth)) / (el.clientWidth - t.offsetWidth - trackX[0] - trackX[1]);
        });
        t.addEventListener('pointerup', () => {
            dragging = null;
            show();
        });
        t.addEventListener('pointerleave', () => {
            clearTimeout(hideTimer);
            hideTimer = setTimeout(hide, 900);
        });
    };
    drag(ty, true);
    drag(tx, false);
    place();
},
```

## Workspace

A workspace is a sidebar flush with the ground, a `mantle` list panel, and an inspector panel from `2xl`. The sidebar holds a search field and nav rows under `ui.cls.group` labels. A nav row is `h-9 rounded-xl`, hovers at `surface0/50`, and when selected takes `surface0` with a mauve icon and a mauve count.

Below `md` the sidebar is a drawer translated in from the edge over a `bg-crust/70` backdrop. The header button that collapses it on a wider screen opens the drawer instead.

```javascript
const groups = [['Workspaces', [['inbox', 'Inbox', 12, 'mauve'], ['star', 'Starred', 3]]], ['Labels', [[null, 'Bug', 4, 'red'], [null, 'Feature', 9, 'green']]], ['Projects', ['claudex', 'anbu'].map((p) => ['folder', p])]];
const side = groups.map(([g, items]) => `<p class="${ui.cls.group}">${g}</p>${items.map(([ic, label, n, tone], i) => `<button type="button" class="group flex h-9 w-full items-center gap-3 rounded-xl px-3 text-left text-sm transition-colors ${g === 'Workspaces' && i === 0 ? 'bg-surface0 text-text' : 'text-subtext0 hover:bg-surface0/50 hover:text-text'}">${ic ? `<i data-lucide="${ic}" class="size-4 shrink-0 ${g === 'Workspaces' && i === 0 ? 'text-mauve' : 'text-overlay1'}"></i>` : `<span class="grid size-4 place-items-center">${ui.dot(tone)}</span>`}<span class="min-w-0 flex-1 truncate">${label}</span>${n ? ui.count(n, g === 'Workspaces' && i === 0 ? 'mauve' : 'neutral') : ''}</button>`).join('')}`).join('');

const items = Array.from({ length: 2 }, (_, i) => {
    const tones = ['red', 'green'];
    const names = ['Bug', 'Feature'];
    const k = i % 2;
    return `<li><button type="button" class="flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-left transition-colors ${i === 1 ? 'bg-surface0' : 'hover:bg-surface0/50'}">
        ${ui.avatar(['Ana', 'Bo'][k], tones[(k + 1) % 2])}
        <span class="min-w-0 flex-1"><span class="flex items-center gap-2"><span class="${ui.cls.rowText} text-sm font-medium text-text">${['Scrollbar jumps on every second wheel event', 'Tooltip pattern for icon-only buttons'][k]} #${40 + i}</span></span><span class="mt-0.5 block truncate text-xs text-overlay1">${['opened 2h ago', 'updated yesterday'][k]}</span></span>
        ${ui.chip(names[k], { tone: tones[k] })}
    </button></li>`;
}).join('');

const inspector = `<div class="flex flex-col gap-5">
    <header class="flex items-center gap-3">${ui.tile('message-square', 'mauve')}<div class="min-w-0"><h3 class="font-display font-medium text-text">Tooltip pattern</h3><p class="font-mono text-xs text-overlay1">#41</p></div></header>
    <dl class="grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2.5 text-sm"><dt class="text-xs text-overlay1">Status</dt><dd>${ui.status('Open', 'green')}</dd><dt class="text-xs text-overlay1">Label</dt><dd>${ui.chip('Feature', { tone: 'green' })}</dd><dt class="text-xs text-overlay1">Due</dt><dd class="text-text">Nov 20</dd><dt class="text-xs text-overlay1">Owner</dt><dd class="flex items-center gap-2 text-text">${ui.avatar('Bo', 'blue', 'size-6 text-xs')}Bo</dd></dl>
    <p class="text-sm leading-relaxed">Icon-only buttons carry their name in a tooltip that appears after a short delay on hover and immediately on keyboard focus.</p>
    ${Array.from({ length: 1 }, (_, i) => `<div class="flex gap-3">${ui.avatar(['Ana', 'Bo', 'Cy'][i % 3], ['mauve', 'blue', 'teal'][i % 3], 'size-7 text-xs')}<div class="min-w-0 ${ui.cls.tile} px-3 py-2 text-sm">Comment ${i + 1} in the inspector scrolls on its own.</div></div>`).join('')}
</div>`;

return `<div class="flex h-full min-h-0 flex-col">
    ${pageHead('Layout', ui.segmented({ name: 'layout-tab', value: 'issues', ground: true, options: [{ value: 'issues', label: 'Issues', icon: 'circle-dot', compact: true }, { value: 'prs', label: 'Pulls', icon: 'git-pull-request', compact: true }, { value: 'board', label: 'Board', icon: 'kanban', compact: true }] }))}
    <div data-shell class="flex min-h-0 flex-1">
        <div data-backdrop class="fixed inset-0 z-30 hidden bg-crust/70 md:hidden"></div>
        <aside data-side class="flex min-h-0 flex-col md:mr-4 max-md:fixed max-md:inset-y-3 max-md:left-3 max-md:z-40 max-md:w-[min(21rem,calc(100vw-3rem))] max-md:-translate-x-[120%] max-md:rounded-3xl max-md:bg-mantle max-md:p-2 max-md:shadow-2xl max-md:shadow-shade max-md:transition-transform">
            <div class="shrink-0 pr-3.5 pb-2">${ui.search({ name: 'side-q', placeholder: 'Filter' })}</div>
            <nav data-scroll="lane" class="min-h-0 flex-1 overflow-y-auto pb-2">${side}</nav>
        </aside>
        <section class="flex min-w-0 flex-1 flex-col rounded-3xl bg-mantle">
            <header class="flex shrink-0 items-center gap-2 px-3 pt-3 pb-2 sm:px-4">
                ${ui.iconButton('panel-left', { tip: 'Toggle sidebar', attrs: 'data-action="side-toggle"' })}
                <h2 class="min-w-0 flex-1 truncate font-display text-lg font-medium text-text">Inbox</h2>
                ${ui.iconButton('list-filter', { tip: 'Filter' })}<span class="max-2xl:hidden">${ui.iconButton('panel-right', { tip: 'Toggle details', attrs: 'data-action="inspector-toggle"' })}</span>
                ${ui.button('New issue', { icon: 'plus', variant: 'primary', collapse: true })}
            </header>
            <ul data-scroll="lane" class="min-h-0 flex-1 overflow-y-auto px-2 pb-2">${items}</ul>
        </section>
        <aside data-inspector class="hidden min-h-0 flex-col rounded-3xl bg-mantle 2xl:ml-4 2xl:flex"><div data-scroll class="min-h-0 flex-1 overflow-y-auto p-5">${inspector}</div></aside>
    </div>
</div>`;
```

The view's `mount` attaches the sidebars and swaps the collapse for the drawer on a phone.

```javascript
mount(el) {
    const side = el.querySelector('[data-side]');
    const backdrop = el.querySelector('[data-backdrop]');
    const insp = el.querySelector('[data-inspector]');
    const sideCtl = ui.media.mobile() ? null : ui.sidebar(side, { key: 'layout-side', initial: 264, min: 200, max: 420 });
    const inspCtl = ui.sidebar(insp, { key: 'layout-inspector', initial: 340, min: 280, max: 560, side: 'right' });
    const drawer = (open) => {
        side.dataset.open = String(open);
        side.style.translate = open ? '0' : '';
        backdrop.classList.toggle('hidden', !open);
    };
    backdrop.addEventListener('click', () => drawer(false));
    ui.on('side-toggle', () => (sideCtl ? sideCtl.toggle() : drawer(side.dataset.open !== 'true')));
    ui.on('inspector-toggle', () => inspCtl.toggle());
},
```

## Reading Column

A panel of prose, such as a chat transcript or a rendered document, fills its panel by default. A pressed `fold-horizontal` icon button at the right of the panel header confines every `data-column` element to `max-w-4xl`, about 120 characters of body text. The choice persists in `localStorage`, and the column eases between widths because `max-width` interpolates between a percentage and a length.

```javascript
const column = {
    confined: localStorage.getItem('APP_NAME.column') === 'confined',
    cls: () => (column.confined ? 'max-w-4xl' : 'max-w-full'),
    toggle(b) {
        column.confined = !column.confined;
        try { localStorage.setItem('APP_NAME.column', column.confined ? 'confined' : 'full'); } catch {}
        document.querySelectorAll('[data-column]').forEach((n) => {
            n.classList.toggle('max-w-4xl', column.confined);
            n.classList.toggle('max-w-full', !column.confined);
        });
        b.setAttribute('aria-pressed', String(column.confined));
    },
};
ui.on('column-toggle', (b) => column.toggle(b));

const toggle = ui.iconButton('fold-horizontal', { tip: 'Narrow column', pressed: column.confined, action: 'column-toggle' });
const body = `<article data-column class="markdown-body mx-auto w-full transition-[max-width] duration-300 ease-out ${column.cls()}"></article>`;
```

## Sidebars

```javascript
sidebar(panel, { key, min = 200, max = 520, initial = 280, side = 'left', collapsed = false }) {
    const load = (k, fallback) => {
        try {
            const v = localStorage.getItem(`APP_NAME.${key}.${k}`);
            return v === null ? fallback : JSON.parse(v);
        } catch {
            return fallback;
        }
    };
    const save = (k, v) => {
        try { localStorage.setItem(`APP_NAME.${key}.${k}`, JSON.stringify(v)); } catch {}
    };
    const state = { width: load('width', initial), collapsed: load('collapsed', collapsed) };
    panel.style.flexShrink = '0';
    ui.anchor(panel);

    const handle = document.createElement('span');
    handle.setAttribute('role', 'separator');
    handle.setAttribute('aria-orientation', 'vertical');
    handle.dataset.tip = 'Drag to resize, double-click to reset';
    handle.setAttribute('data-tip-pointer', '');
    handle.className = `group/handle absolute inset-y-0 ${side === 'left' ? '-right-2.5' : '-left-2.5'} z-30 flex w-3 cursor-col-resize touch-none items-center justify-center`;
    handle.innerHTML = '<span class="h-10 w-1 rounded-full bg-surface1 opacity-0 transition-opacity group-hover/handle:opacity-100 group-active/handle:bg-mauve group-active/handle:opacity-100"></span>';
    panel.appendChild(handle);

    const apply = (animate) => {
        panel.style.transition = animate ? 'width .2s ease, opacity .2s ease, margin .2s ease' : 'none';
        panel.style.width = state.collapsed ? '0px' : `${state.width}px`;
        panel.style.margin = state.collapsed ? '0' : '';
        panel.style.opacity = state.collapsed ? '0' : '1';
        panel.style.overflow = state.collapsed ? 'hidden' : '';
        panel.style.pointerEvents = state.collapsed ? 'none' : '';
        panel.inert = state.collapsed;
        panel.dataset.collapsed = String(state.collapsed);
        handle.style.display = state.collapsed ? 'none' : '';
        ctl.onToggle?.(state.collapsed);
    };

    handle.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        handle.setPointerCapture(e.pointerId);
        const startX = e.clientX;
        const startW = state.width;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
        const move = (ev) => {
            const dx = (ev.clientX - startX) * (side === 'left' ? 1 : -1);
            state.width = Math.round(Math.min(max, Math.max(min, startW + dx)));
            apply(false);
        };
        const up = () => {
            handle.removeEventListener('pointermove', move);
            handle.removeEventListener('pointerup', up);
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            save('width', state.width);
        };
        handle.addEventListener('pointermove', move);
        handle.addEventListener('pointerup', up);
    });
    handle.addEventListener('dblclick', () => {
        state.width = initial;
        save('width', state.width);
        apply(true);
    });

    const ctl = {
        onToggle: null,
        get collapsed() { return state.collapsed; },
        toggle(force) {
            state.collapsed = typeof force === 'boolean' ? force : !state.collapsed;
            save('collapsed', state.collapsed);
            apply(true);
        },
    };
    apply(false);
    return ctl;
},
```

`ui.sidebar(panel, { key, min, max, initial, side })` gives a panel a drag handle on its inner edge, a double-click that resets the width, and a collapse to zero width. The width and the collapsed state persist in `localStorage` under the app's name and the key, so a layout survives a reload.

The handle's tooltip anchors above the pointer through `data-tip-pointer`. The handle runs the panel's full height, so a tooltip centered on it lands far from where the pointer is.

A collapsed panel is `inert`, so Tab never lands on a control hidden at zero width.

A collapsed panel also drops its margin. A sidebar keeps its distance from the panel beside it with a margin on its inner edge rather than the row's `gap`, which would leave a 16px seam where the panel was.
