const views = {};

const store = {
    state: null,
    loading: false,
};

const app = {
    current: null,
    params: [],

    async start() {
        ui.initDelegation();
        ui.tooltip.init();
        ui.initScrollObserver();
        app.renderNav();
        live.start();
        window.addEventListener('hashchange', () => app.route());
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') app.refresh();
        });
        window.addEventListener('pageshow', (e) => {
            if (e.persisted) app.refresh();
        });
        await app.refresh();
        app.route();
    },

    async refresh() {
        store.loading = true;
        app.renderNav();
        try {
            store.state = await api.state();
        } catch (err) {
            ui.toast(err.message, { tone: 'red', icon: 'circle-alert' });
        }
        store.loading = false;
        app.renderNav();
        views[app.current]?.update?.();
        document.dispatchEvent(new CustomEvent('store-change'));
    },

    async act(label, fn, { success, refresh = true } = {}) {
        try {
            const result = await fn();
            if (success) ui.toast(typeof success === 'function' ? success(result) : success, { tone: 'green', icon: 'circle-check' });
            if (refresh) await app.refresh();
            return { ok: true, result };
        } catch (err) {
            ui.toast(`${label} failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            if (refresh) await app.refresh();
            return { ok: false, error: err };
        }
    },

    route() {
        const parts = location.hash.replace(/^#\/?/, '').split('/').filter(Boolean).map(decodeURIComponent);
        const name = { project: 'project', settings: 'settings' }[parts[0]] || 'dashboard';
        const el = document.getElementById('view');
        views[app.current]?.unmount?.();
        app.current = name;
        app.params = parts.slice(1);
        el.innerHTML = '';
        views[name].mount(el, app.params);
        app.renderNav();
        ui.icons(el);
    },

    go(hash) {
        if (location.hash === hash) app.route();
        else location.hash = hash;
    },

    renderNav() {
        const links = [
            { name: 'dashboard', hash: '#/', icon: 'layout-dashboard', label: 'Dashboard' },
            { name: 'settings', hash: '#/settings', icon: 'settings-2', label: 'Settings' },
        ];
        const navLinks = document.getElementById('nav-links');
        const waiting = live.awayPending();
        navLinks.innerHTML = links.map((l) => {
            const on = app.current === l.name;
            const dot = l.name === 'dashboard' && waiting ? `<span data-tip="${waiting} waiting for approval">${ui.dot('yellow', { pulse: true })}</span>` : '';
            return `<a href="${l.hash}" class="flex h-9 items-center gap-2 rounded-full px-4 text-sm whitespace-nowrap transition-colors ${ui.cls.focus} ${on ? 'bg-surface0 text-text' : 'text-overlay1 hover:text-text'}" ${on ? 'aria-current="page"' : ''}>
                <i data-lucide="${l.icon}" class="size-4 ${on ? 'text-mauve' : ''}"></i><span class="${on ? '' : 'max-sm:hidden'}">${l.label}</span>${dot}</a>`;
        }).join('');
        const status = document.getElementById('nav-status');
        status.innerHTML = store.loading ? '<i data-lucide="loader-circle" class="size-4 animate-spin"></i>' : '';
        ui.icons(navLinks);
        ui.icons(status);
    },
};

ui.on('app-home', () => {
    if (app.current === 'dashboard' && !app.params.length) app.refresh();
    else {
        app.go('#/');
        app.refresh();
    }
});
ui.on('new-session', () => views.newSession.open());

const column = {
    confined: localStorage.getItem('claudex.column') === 'confined',
    cls: () => (column.confined ? 'max-w-4xl' : 'max-w-full'),
    toggle(b) {
        column.confined = !column.confined;
        try { localStorage.setItem('claudex.column', column.confined ? 'confined' : 'full'); } catch {}
        document.querySelectorAll('[data-column]').forEach((n) => {
            n.classList.toggle('max-w-4xl', column.confined);
            n.classList.toggle('max-w-full', !column.confined);
        });
        b.setAttribute('aria-pressed', String(column.confined));
    },
};
ui.on('column-toggle', (b) => column.toggle(b));

const fmt = {
    tokens(n) {
        if (n < 1000) return String(n);
        if (n < 1e6) return `${(n / 1000).toFixed(n < 1e4 ? 1 : 0).replace(/\.0$/, '')}k`;
        return `${(n / 1e6).toFixed(1).replace(/\.0$/, '')}M`;
    },
    dur(ms) {
        const s = Math.max(0, Math.round(ms / 1000));
        if (s < 60) return `${s}s`;
        if (s < 3600) return `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`;
        return `${Math.floor(s / 3600)}h ${String(Math.floor(s / 60) % 60).padStart(2, '0')}m`;
    },
    ms(ms) {
        return ms < 1000 ? `${Math.round(ms)}ms` : fmt.dur(ms);
    },
    time(epochSec) {
        return new Date(epochSec * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
    },
    day(epochSec) {
        return new Date(epochSec * 1000).toLocaleDateString([], { weekday: 'short', hour: 'numeric' });
    },
    plural(n, word) {
        return `${n} ${word}${n === 1 ? '' : 's'}`;
    },
};

const md = {
    ready: false,

    slug(text) {
        return String(text).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '');
    },

    init() {
        if (md.ready) return;
        md.ready = true;
        const icons = { tip: 'lightbulb', info: 'info', danger: 'triangle-alert', warning: 'triangle-alert', note: 'sticky-note' };
        marked.use({
            renderer: {
                code({ text, lang }) {
                    if (lang === 'mermaid') return `<div class="mermaid" data-scroll data-scroll-cue>${ui.esc(text)}</div>`;
                    const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                    let html = ui.esc(text);
                    try {
                        html = hljs.highlight(text, { language }).value;
                    } catch {}
                    return `<div class="group relative"><pre data-scroll data-scroll-cue><code class="hljs language-${language}">${html}</code></pre><span class="absolute top-2 right-2 ${ui.cls.reveal}">${ui.iconButton('copy', { tip: 'Copy', size: 'sm', variant: 'secondary', attrs: 'data-action="md-copy"' })}</span></div>`;
                },
                checkbox({ checked }) {
                    return `<span role="checkbox" aria-checked="${checked}" aria-disabled="true" class="mr-2 inline-flex size-5 items-center justify-center rounded-lg align-middle ${checked ? 'bg-mauve text-crust' : 'bg-surface1'}">${checked ? '<i data-lucide="check" class="size-3.5" stroke-width="3"></i>' : ''}</span>`;
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

    render(container, source, opts) {
        md.init();
        container.innerHTML = DOMPurify.sanitize(marked.parse(source, opts));
        container.querySelectorAll('table').forEach((table) => {
            const wrap = Object.assign(document.createElement('div'), { className: 'mb-6 overflow-x-auto' });
            wrap.dataset.scroll = '';
            wrap.dataset.scrollCue = '';
            wrap.dataset.table = '';
            table.replaceWith(wrap);
            wrap.appendChild(table);
        });
        md.mermaid(container);
        ui.icons(container);
    },

    mermaid(container) {
        const nodes = [...container.querySelectorAll('.mermaid')];
        if (!nodes.length) return;
        nodes.forEach((n) => {
            n.dataset.src ??= n.textContent;
            n.removeAttribute('data-processed');
            n.textContent = n.dataset.src;
        });
        mermaid.initialize(mermaidConfig());
        document.fonts.ready
            .then(() => {
                const attached = nodes.filter((n) => n.isConnected);
                return attached.length ? mermaid.run({ nodes: attached }).then(() => attached) : [];
            })
            .then((attached) => attached.forEach((n) => {
                md.fit(n);
                md.fitter.observe(n);
            }))
            .catch((err) => ui.toast(`Diagram failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' }));
    },

    fitter: new ResizeObserver((entries) => entries.forEach((e) => md.fit(e.target))),

    fit(node) {
        const svg = node.querySelector(':scope > svg');
        if (!node.isConnected || !svg) {
            md.fitter.unobserve(node);
            return;
        }
        const box = svg.viewBox.baseVal;
        if (!node.clientWidth || !box?.width) return;
        const css = getComputedStyle(node);
        const room = node.clientWidth - parseFloat(css.paddingLeft) - parseFloat(css.paddingRight);
        const scale = Math.max(0.6, Math.min(1, room / box.width));
        svg.style.maxWidth = 'none';
        svg.style.width = `${box.width * scale}px`;
        svg.style.height = `${box.height * scale}px`;
    },
};
ui.on('md-copy', (b) => ui.copy(b.closest('.group').querySelector('code').textContent, b));

const mermaidConfig = () => {
    const css = getComputedStyle(document.documentElement);
    const c = (name) => css.getPropertyValue(`--ctp-${name}`).trim();
    return {
        startOnLoad: false,
        theme: 'base',
        fontFamily: 'Inter',
        ...Object.fromEntries(['flowchart', 'sequence', 'gantt', 'pie', 'gitGraph', 'state', 'timeline', 'mindmap'].map((d) => [d, { useMaxWidth: false }])),
        themeVariables: {
            darkMode: !document.documentElement.classList.contains('light'),
            background: c('base'),
            mainBkg: c('base'),

            primaryColor: c('surface0'),
            primaryTextColor: c('text'),
            primaryBorderColor: c('blue'),
            secondaryColor: c('surface1'),
            secondaryTextColor: c('text'),
            secondaryBorderColor: c('overlay1'),
            tertiaryColor: c('surface0'),
            tertiaryTextColor: c('text'),
            tertiaryBorderColor: c('surface2'),

            lineColor: c('blue'),
            arrowheadColor: c('blue'),
            textColor: c('text'),
            titleColor: c('mauve'),
            noteBkgColor: c('surface1'),
            noteTextColor: c('yellow'),
            noteBorderColor: c('surface2'),

            nodeBkg: c('surface0'),
            nodeBorder: c('blue'),
            clusterBkg: c('mantle'),
            clusterBorder: c('surface2'),
            defaultLinkColor: c('blue'),
            edgeLabelBackground: c('surface0'),
            nodeTextColor: c('text'),

            actorBkg: c('surface0'),
            actorBorder: c('blue'),
            actorTextColor: c('text'),
            actorLineColor: c('surface2'),
            signalColor: c('pink'),
            signalTextColor: c('text'),
            labelBoxBkgColor: c('surface1'),
            labelBoxBorderColor: c('surface2'),
            labelTextColor: c('text'),
            loopTextColor: c('yellow'),
            activationBorderColor: c('mauve'),
            activationBkgColor: c('surface1'),
            sequenceNumberColor: c('base'),

            sectionBkgColor: c('mantle'),
            altSectionBkgColor: c('base'),
            sectionBkgColor2: c('crust'),
            taskBkgColor: c('blue'),
            taskBorderColor: c('lavender'),
            taskTextColor: c('base'),
            taskTextLightColor: c('base'),
            taskTextDarkColor: c('text'),
            taskTextOutsideColor: c('text'),
            taskTextClickableColor: c('sky'),
            activeTaskBkgColor: c('mauve'),
            activeTaskBorderColor: c('pink'),
            doneTaskBkgColor: c('surface1'),
            doneTaskBorderColor: c('surface2'),
            critBkgColor: c('red'),
            critBorderColor: c('maroon'),
            gridColor: c('surface0'),
            todayLineColor: c('red'),

            pie1: c('mauve'), pie2: c('blue'), pie3: c('green'), pie4: c('yellow'),
            pie5: c('red'), pie6: c('teal'), pie7: c('peach'), pie8: c('sky'),
            pie9: c('pink'), pie10: c('sapphire'), pie11: c('maroon'), pie12: c('lavender'),
            pieTitleTextColor: c('text'),
            pieSectionTextColor: c('base'),
            pieLegendTextColor: c('text'),
            pieStrokeColor: c('base'),
            pieOuterStrokeColor: c('surface0'),

            git0: c('blue'), git1: c('mauve'), git2: c('green'), git3: c('yellow'),
            git4: c('red'), git5: c('teal'), git6: c('peach'), git7: c('sapphire'),
            gitInv0: c('base'), gitInv1: c('base'), gitInv2: c('base'), gitInv3: c('base'),
            gitInv4: c('base'), gitInv5: c('base'), gitInv6: c('base'), gitInv7: c('base'),
            commitLabelColor: c('subtext1'),
            commitLabelBackground: c('base'),
            tagLabelColor: c('base'),
            tagLabelBackground: c('yellow'),
            tagLabelBorder: c('peach'),

            labelBackgroundColor: c('surface0'),

            cScale0: c('surface0'), cScale1: c('blue'), cScale2: c('mauve'), cScale3: c('green'),
            cScale4: c('yellow'), cScale5: c('red'), cScale6: c('teal'), cScale7: c('peach'),
            cScale8: c('sky'), cScale9: c('pink'), cScale10: c('sapphire'), cScale11: c('lavender'),
        },
    };
};
