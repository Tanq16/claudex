views.settings = (() => {
    const sections = [
        { id: 'accounts', label: 'Accounts', icon: 'users-round', hint: 'Discovered from ~/.claude and ~/.claudeN' },
        { id: 'defaults', label: 'Defaults', icon: 'sliders-horizontal', hint: 'Active keys are written to every account on apply' },
        { id: 'presets', label: 'Presets', icon: 'package', hint: 'Tracked presets update automatically on every apply' },
        { id: 'mcp', label: 'MCP library', icon: 'server', hint: "Credentials are stored inline in claudex's config dir" },
    ];

    const s = {
        el: null,
        section: 'accounts',
        data: null,
        error: '',
        defaults: null,
        defaultsError: '',
        dfilter: 'active',
        dquery: '',
        loading: Promise.resolve(),
        scroll: {},
        revealed: new Set(),
        pending: new Set(),
        track: { repo: '', path: '', error: '' },
        editor: null,
        listeners: [],
    };

    const esc = ui.esc;
    const busy = (key) => s.pending.has(key);
    const plural = (n, word) => `${n} ${word}${n === 1 ? '' : 's'}`;
    const fmt = (v) => (typeof v === 'string' ? v : JSON.stringify(v));
    const circle = (inner) => `<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-surface0 text-subtext0">${inner}</span>`;
    const icon = (name, size = 'size-5') => `<i data-lucide="${name}" class="${size} shrink-0"></i>`;
    const groupHeader = (title, count, extra = '') => `<header class="flex items-center gap-2 px-1">
        <h3 class="flex-1 ${ui.cls.label}">${title}${count === undefined ? '' : ` <span class="tabular-nums text-overlay0">${count}</span>`}</h3>${extra}
    </header>`;
    const commandLine = (m) => [m.command, ...(m.args || [])].map((a) => (/[\s'"]/.test(a) ? `'${a}'` : a)).join(' ');

    const compliance = (a) => {
        if (!a.settingsFound) return { tone: 'red', icon: 'file-x-corner', text: 'settings.json missing' };
        if (a.drift.length) return { tone: 'peach', icon: 'shield-alert', text: `${plural(a.drift.length, 'key')} drift` };
        return { tone: 'green', icon: 'shield-check', text: 'Matches defaults' };
    };

    const counts = () => ({
        ...(s.data ? { accounts: s.data.accounts.length, presets: s.data.presets.builtin.length + s.data.presets.remote.length, mcp: s.data.mcpLibrary.length } : {}),
        ...(s.defaults ? { defaults: s.defaults.rows.filter((r) => r.active).length } : {}),
    });

    const pageHead = () => {
        const c = counts();
        const attention = s.data?.accounts.some((a) => compliance(a).tone !== 'green');
        return `<header class="flex shrink-0 flex-wrap items-center justify-between gap-3">
            <h1 class="font-display text-2xl text-text max-md:hidden">Settings</h1>
            <nav aria-label="Settings sections" class="inline-flex max-w-full gap-1 overflow-x-auto rounded-full bg-mantle p-1 max-md:w-full 3xl:hidden">
                ${sections.map((x) => {
                    const on = x.id === s.section;
                    return `<a href="#/settings/${x.id}" data-action="settings-section" data-section="${x.id}" ${on ? 'aria-current="page"' : ''} class="flex h-8 items-center justify-center gap-2 rounded-full px-4 text-sm whitespace-nowrap transition-colors max-md:flex-1 ${on ? 'bg-surface0 text-text' : 'text-overlay1 hover:text-text'}">
                        ${icon(x.icon, `size-4 ${on ? 'text-mauve' : ''}`)}<span class="${on ? '' : 'max-sm:hidden'}">${x.label}</span>
                        ${x.id === 'accounts' && attention ? '<span title="An account needs attention" class="size-1.5 rounded-full bg-peach"></span>' : ''}
                        ${c[x.id] === undefined ? '' : `<span class="text-xs tabular-nums text-overlay1 max-sm:hidden">${c[x.id]}</span>`}
                    </a>`;
                }).join('')}
            </nav>
        </header>`;
    };

    const driftRow = (d) => `<li class="rounded-xl bg-mantle px-3 py-2 font-mono text-xs">
        <p class="break-all text-subtext1">${esc(d.key)}</p>
        <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
            <span title="Current" class="break-all ${d.actual === 'missing' ? 'italic text-overlay1' : 'text-peach line-through decoration-peach/50'}">${esc(d.actual)}</span>
            <i data-lucide="arrow-right" class="size-3 shrink-0 text-overlay0"></i>
            <span title="Default" class="break-all text-green">${esc(d.expected)}</span>
        </p>
    </li>`;

    const accountCard = (a) => {
        const c = compliance(a);
        const pending = busy(`apply:${a.dir}`);
        return `<li class="flex flex-col gap-3 rounded-2xl bg-base p-4">
            <header class="flex items-start gap-3">
                ${circle(icon('user-round'))}
                <hgroup class="min-w-0 flex-1">
                    <h3 class="truncate text-sm font-medium text-text">${esc(a.label)}</h3>
                    <p class="truncate font-mono text-xs text-overlay1">${esc(a.dir)}</p>
                </hgroup>
                ${a.isDefault ? ui.chip('Default', { icon: 'star' }) : ''}
            </header>
            <p class="flex flex-wrap items-center gap-2">${ui.chip(c.text, { tone: c.tone, icon: c.icon })}${ui.chip(plural(a.projects, 'project'), { icon: 'folder' })}</p>
            ${a.drift.length ? `<ul class="flex flex-col gap-1.5">${a.drift.map(driftRow).join('')}</ul>` : ''}
            ${a.settingsFound ? '' : `<p class="text-sm text-overlay1">Applying defaults creates <span class="font-mono text-subtext1">${esc(a.dir)}/settings.json</span>.</p>`}
            ${c.tone === 'green' ? '' : `<footer class="flex justify-end">${ui.button(pending ? 'Applying' : 'Apply defaults', { icon: pending ? 'loader-circle' : 'wand-sparkles', spin: pending, disabled: pending, size: 'sm', action: 'settings-apply', attrs: `data-dir="${esc(a.dir)}"` })}</footer>`}
        </li>`;
    };

    const accountsBody = () => `
        <p class="flex items-center gap-3 rounded-2xl bg-base px-4 py-3">
            <i data-lucide="folder-root" class="size-4 shrink-0 text-overlay1"></i>
            <span class="${ui.cls.label}">Projects root</span>
            <span class="min-w-0 flex-1 truncate text-right font-mono text-sm text-text">${esc(s.data.root)}</span>
            <span title="Read-only" class="text-overlay0">${icon('lock', 'size-3.5')}</span>
        </p>
        <a href="#/settings/defaults" data-action="settings-section" data-section="defaults" class="flex items-center gap-3 rounded-2xl bg-base px-4 py-3 transition-colors hover:bg-surface0 3xl:hidden">
            <i data-lucide="sliders-horizontal" class="size-4 shrink-0 text-overlay1"></i>
            <span class="${ui.cls.label}">Defaults</span>
            <span class="min-w-0 flex-1 text-right text-sm text-text">${plural(s.data.defaultsCount, 'active key')}</span>
            <i data-lucide="chevron-right" class="size-4 shrink-0 text-overlay0"></i>
        </a>
        <ul class="grid items-start gap-3 xl:grid-cols-2 3xl:grid-cols-1">${s.data.accounts.map(accountCard).join('')}</ul>`;

    const listLimit = 100;
    const valueText = (v) => (v === '' ? '""' : fmt(v));

    const schemaLine = (sc) => (sc.error
        ? `<p class="flex items-center gap-2 rounded-2xl bg-red/10 px-4 py-3 text-sm text-red">${icon('cloud-off', 'size-4')}<span class="min-w-0 flex-1">${esc(sc.error)}</span></p>`
        : `<p class="flex items-center gap-2 px-1 text-xs text-overlay1">${icon('file-json', 'size-3.5')}<span class="min-w-0 flex-1 truncate">${plural(sc.keys, 'key')} in the schema, fetched ${ui.timeAgo(sc.fetchedAt)}</span></p>`);

    const defaultCard = (r) => {
        const attrs = `data-key="${esc(r.key)}"`;
        const pending = busy(`default:${r.key}`);
        const hasDefault = r.default != null;
        const actions = r.active
            ? `${ui.iconButton('pencil', { title: 'Edit', size: 'sm', action: 'settings-default-edit', disabled: pending, attrs })}
               ${ui.iconButton(pending ? 'loader-circle' : 'rotate-ccw', { title: 'Reset', size: 'sm', action: 'settings-default-reset', spin: pending, disabled: pending, attrs })}`
            : `${hasDefault ? ui.button('Accept', { icon: pending ? 'loader-circle' : 'check', spin: pending, disabled: pending, size: 'sm', variant: 'ghost', action: 'settings-default-accept', attrs }) : ''}
               ${ui.iconButton('pencil', { title: 'Edit', size: 'sm', action: 'settings-default-edit', disabled: pending, attrs })}`;
        return `<li class="flex flex-col gap-2 rounded-2xl bg-base p-3">
            <header class="flex items-start gap-2">
                <h4 class="min-w-0 flex-1 pt-1.5 font-mono text-sm break-all text-text">${esc(r.key)}</h4>
                ${r.inSchema ? '' : `<span class="pt-1">${ui.chip('custom', { icon: 'pencil-line' })}</span>`}
                ${actions}
            </header>
            ${r.active
                ? `<p class="line-clamp-6 rounded-xl bg-mantle px-3 py-2 font-mono text-xs break-all text-green">${esc(valueText(r.value))}</p>`
                : `<p class="px-1 font-mono text-xs break-all text-overlay1">${hasDefault ? `default ${esc(valueText(r.default))}` : 'no default'}</p>`}
            ${r.description ? `<p title="${esc(r.description)}" class="line-clamp-2 px-1 text-xs text-subtext0">${esc(r.description)}</p>` : ''}
        </li>`;
    };

    const defaultsBody = () => {
        const d = s.defaults;
        if (!d && s.defaultsError) {
            return ui.empty({ icon: 'circle-alert', title: 'Could not load defaults', hint: esc(s.defaultsError), action: ui.button('Retry', { icon: 'refresh-cw', action: 'settings-retry' }) });
        }
        if (!d) return `<p class="flex flex-1 items-center justify-center text-overlay1">${icon('loader-circle', 'size-5 animate-spin')}</p>`;
        const active = d.rows.filter((r) => r.active);
        const inactive = d.rows.filter((r) => !r.active);
        const q = s.dquery.trim().toLowerCase();
        const matched = (s.dfilter === 'active' ? active : inactive)
            .filter((r) => !q || r.key.toLowerCase().includes(q) || (r.description || '').toLowerCase().includes(q));
        const shown = matched.slice(0, listLimit);
        return `${schemaLine(d.schema)}
            <div class="flex flex-wrap items-center gap-2">
                ${ui.segmented({ name: 'defaults-filter', value: s.dfilter, options: [{ value: 'active', label: `Active ${active.length}` }, { value: 'default', label: `Default ${inactive.length}` }] })}
                <label class="relative min-w-40 flex-1">
                    <span class="sr-only">Filter keys</span>
                    <i data-lucide="search" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i>
                    <input data-field="defaults.query" value="${esc(s.dquery)}" placeholder="Filter keys" autocomplete="off" spellcheck="false" class="${ui.cls.input} pl-10 font-mono">
                </label>
            </div>
            ${shown.length
                ? `<ul class="flex flex-col gap-2">${shown.map(defaultCard).join('')}</ul>
                   ${matched.length > shown.length ? `<p class="px-1 text-center text-xs text-overlay1">Showing ${shown.length} of ${matched.length}; filter to narrow</p>` : ''}`
                : ui.empty({ icon: 'search-x', title: q ? 'No matching keys' : s.dfilter === 'active' ? 'No active keys' : 'Every schema key is active' })}
            ${s.dfilter === 'active' && !q ? `<section class="flex flex-col gap-2">
                ${groupHeader('Managed by claudex', d.owned.length, ui.chip('per account', { icon: 'lock' }))}
                <p class="flex flex-wrap gap-2 px-1">${d.owned.map((k) => ui.chip(k, { mono: true })).join('')}</p>
            </section>` : ''}`;
    };

    const builtinCard = (p) => `<li class="flex flex-col gap-3 rounded-2xl bg-base p-4">
        <header class="flex items-center gap-3">
            ${circle(icon('package'))}
            <h4 class="min-w-0 flex-1 truncate font-mono text-sm font-medium text-text">${esc(p.name)}</h4>
        </header>
        <p class="flex flex-wrap items-center gap-2">${ui.chip('embedded', { icon: 'hard-drive' })}${p.defaultOn ? ui.chip('on by default', { icon: 'check' }) : ''}</p>
        <p class="text-sm text-subtext0">${esc(p.description)}</p>
    </li>`;

    const remoteCard = (p) => {
        const attrs = `data-name="${esc(p.name)}"`;
        const updating = busy(`update:${p.name}`);
        const removing = busy(`remove:${p.name}`);
        return `<li class="flex flex-col gap-3 rounded-2xl bg-base p-4">
            <header class="flex items-start gap-3">
                ${circle('<i class="fab fa-github text-[1.125rem]"></i>')}
                <hgroup class="min-w-0 flex-1">
                    <h4 class="truncate font-mono text-sm font-medium text-text">${esc(p.name)}</h4>
                    <p class="truncate font-mono text-xs text-overlay1">${esc(p.repo)}</p>
                </hgroup>
                ${ui.iconButton(removing ? 'loader-circle' : 'trash-2', { title: 'Remove', variant: 'danger', size: 'sm', action: 'settings-preset-remove', spin: removing, disabled: removing || updating, attrs })}
            </header>
            <p class="text-sm text-subtext0">${esc(p.description)}</p>
            <footer class="flex flex-wrap items-center gap-2">
                <span class="mr-auto inline-flex items-center gap-1.5 text-xs text-overlay1">${icon('clock', 'size-3.5')}Updated ${ui.timeAgo(p.updatedAt)}</span>
                ${ui.button(updating ? 'Updating' : 'Update now', { icon: updating ? 'loader-circle' : 'refresh-cw', spin: updating, disabled: updating || removing, size: 'sm', action: 'settings-preset-update', attrs })}
            </footer>
        </li>`;
    };

    const trackForm = () => {
        const pending = busy('track');
        const disabled = pending ? 'disabled' : '';
        return `<form data-form="track" class="flex flex-col gap-3 rounded-2xl bg-base p-4">
            <p class="text-sm font-medium text-text">Track a preset</p>
            <span class="grid gap-2 sm:grid-cols-[minmax(0,3fr)_minmax(0,2fr)_auto]">
                <label class="relative block">
                    <span class="sr-only">Repository</span>
                    <i class="fab fa-github pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 text-overlay1"></i>
                    <input data-field="track.repo" value="${esc(s.track.repo)}" placeholder="owner/repo" autocomplete="off" spellcheck="false" ${disabled} class="${ui.cls.input} pl-10 font-mono disabled:opacity-60">
                </label>
                <input data-field="track.path" value="${esc(s.track.path)}" placeholder="preset path (optional)" aria-label="Preset path" autocomplete="off" spellcheck="false" ${disabled} class="${ui.cls.input} font-mono disabled:opacity-60">
                ${ui.button(pending ? 'Tracking' : 'Track', { icon: pending ? 'loader-circle' : 'plus', spin: pending, disabled: pending, type: 'submit', variant: 'primary' })}
            </span>
            ${s.track.error ? `<p role="alert" class="flex items-center gap-2 px-1 text-sm text-red">${icon('circle-alert', 'size-4')}${esc(s.track.error)}</p>` : ''}
        </form>`;
    };

    const presetsBody = () => {
        const { builtin, remote } = s.data.presets;
        return `<section class="flex flex-col gap-2">
            ${groupHeader('Built-in', builtin.length)}
            <ul class="grid items-start gap-3 xl:grid-cols-2 3xl:grid-cols-1">${builtin.map(builtinCard).join('')}</ul>
        </section>
        <section class="flex flex-col gap-2">
            ${groupHeader('Tracked from GitHub', remote.length)}
            ${trackForm()}
            ${remote.length
                ? `<ul class="grid items-start gap-3 xl:grid-cols-2 3xl:grid-cols-1">${remote.map(remoteCard).join('')}</ul>`
                : ui.empty({ icon: 'package-open', title: 'No tracked presets', hint: 'Presets tracked from a GitHub repository show up here and in the new-session modal.' })}
        </section>`;
    };

    const mcpCard = (m) => {
        const http = m.transport === 'http';
        const secrets = Object.entries((http ? m.headers : m.env) || {});
        const shown = s.revealed.has(m.name);
        const deleting = busy(`mcp:${m.name}`);
        const attrs = `data-name="${esc(m.name)}"`;
        return `<li class="flex flex-col gap-3 rounded-2xl bg-base p-4">
            <header class="flex items-center gap-3">
                ${circle(icon('plug'))}
                <h4 class="min-w-0 flex-1 truncate font-mono text-sm font-medium text-text">${esc(m.name)}</h4>
                ${ui.chip(m.transport, { icon: http ? 'globe' : 'square-terminal', mono: true })}
            </header>
            <p class="rounded-xl bg-mantle px-3 py-2 font-mono text-xs leading-relaxed break-all text-subtext1">${esc(http ? m.url : commandLine(m))}</p>
            ${secrets.length ? `<section class="flex flex-col gap-1.5">
                <header class="flex items-center gap-2 pl-1">
                    <h5 class="flex flex-1 items-center gap-1.5 ${ui.cls.label}">${icon('key-round', 'size-3.5')}${http ? 'Headers' : 'Env'}</h5>
                    ${ui.iconButton(shown ? 'eye-off' : 'eye', { title: shown ? 'Hide secrets' : 'Reveal secrets', size: 'sm', action: 'settings-mcp-reveal', attrs })}
                </header>
                <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1.5 rounded-xl bg-mantle px-3 py-2 font-mono text-xs">
                    ${secrets.map(([k, v]) => `<dt class="text-overlay1">${esc(k)}</dt><dd class="break-all ${shown ? 'text-text' : 'tracking-widest text-overlay0'}">${shown ? esc(v) : '••••••••••••'}</dd>`).join('')}
                </dl>
            </section>` : `<p class="flex items-center gap-1.5 px-1 text-xs text-overlay1">${icon('key-round', 'size-3.5')}No ${http ? 'headers' : 'env'} set</p>`}
            <footer class="mt-auto flex items-center justify-end gap-1">
                ${ui.button('Edit', { icon: 'pencil', size: 'sm', variant: 'ghost', action: 'settings-mcp-edit', disabled: deleting, attrs })}
                ${ui.iconButton(deleting ? 'loader-circle' : 'trash-2', { title: 'Delete', variant: 'danger', size: 'sm', action: 'settings-mcp-delete', spin: deleting, disabled: deleting, attrs })}
            </footer>
        </li>`;
    };

    const mcpBody = () => {
        const list = s.data.mcpLibrary;
        if (!list.length) {
            return ui.empty({ icon: 'server-off', title: 'No MCP servers', hint: 'Servers added here can be picked when starting a session or added to a running project.', action: ui.button('Add server', { icon: 'plus', action: 'settings-mcp-add' }) });
        }
        return `<ul class="grid items-start gap-3 xl:grid-cols-2 3xl:grid-cols-1">${list.map(mcpCard).join('')}</ul>`;
    };

    const body = (id) => {
        if (id === 'defaults') return defaultsBody();
        if (!s.data && s.error) {
            return ui.empty({ icon: 'circle-alert', title: 'Could not load settings', hint: esc(s.error), action: ui.button('Retry', { icon: 'refresh-cw', action: 'settings-retry' }) });
        }
        if (!s.data) return `<p class="flex flex-1 items-center justify-center text-overlay1">${icon('loader-circle', 'size-5 animate-spin')}</p>`;
        return { accounts: accountsBody, presets: presetsBody, mcp: mcpBody }[id]();
    };

    const panel = (x) => `<section aria-labelledby="settings-${x.id}" class="${x.id === s.section ? 'flex' : 'hidden'} min-h-0 flex-1 flex-col rounded-3xl bg-mantle 3xl:flex">
        <header class="flex shrink-0 items-center gap-3 px-5 pt-5 pb-4 md:px-6 md:pt-6">
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-surface0 text-text">${icon(x.icon)}</span>
            <hgroup class="min-w-0 flex-1">
                <h2 id="settings-${x.id}" class="font-display text-lg font-medium text-text">${x.label}</h2>
                <p class="text-sm text-overlay1">${x.hint}</p>
            </hgroup>
            ${x.id === 'mcp' && s.data?.mcpLibrary.length ? ui.button('Add server', { icon: 'plus', size: 'sm', action: 'settings-mcp-add', collapse: true }) : ''}
            ${x.id === 'defaults' ? `${ui.iconButton(busy('schema') ? 'loader-circle' : 'cloud-download', { title: 'Refresh upstream schema', size: 'sm', variant: 'secondary', action: 'settings-schema-refresh', spin: busy('schema'), disabled: busy('schema') })}
                ${ui.button('Add key', { icon: 'plus', size: 'sm', action: 'settings-default-add', collapse: true })}` : ''}
        </header>
        <div data-scroll data-scroll-key="${x.id}" class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-5 pb-5 md:px-6 md:pb-6">${body(x.id)}</div>
    </section>`;

    const view = () => `<section aria-label="Settings" class="flex h-full flex-col gap-4 3xl:gap-6">
        ${pageHead()}
        <div class="flex min-h-0 flex-1 flex-col 3xl:grid 3xl:grid-cols-4 3xl:grid-rows-1 3xl:gap-6">${sections.map(panel).join('')}</div>
    </section>`;

    const render = () => {
        if (!s.el) return;
        s.el.querySelectorAll('[data-scroll-key]').forEach((n) => {
            if (n.offsetParent) s.scroll[n.dataset.scrollKey] = n.scrollTop;
        });
        const active = document.activeElement;
        let focus = null;
        if (s.el.contains(active)) {
            if (active.dataset.field) focus = { selector: `[data-field="${active.dataset.field}"]`, range: [active.selectionStart, active.selectionEnd] };
            else if (active.dataset.ui === 'radio') focus = { selector: `[data-name="${active.closest('[data-name]').dataset.name}"] [aria-checked="true"]` };
        }
        s.el.innerHTML = view();
        ui.icons(s.el);
        s.el.querySelectorAll('[data-scroll-key]').forEach((n) => {
            n.scrollTop = s.scroll[n.dataset.scrollKey] || 0;
        });
        if (focus) {
            const target = s.el.querySelector(focus.selector);
            target?.focus();
            if (focus.range && target?.setSelectionRange) target.setSelectionRange(...focus.range);
        }
    };

    const loadSettings = async () => {
        try {
            s.data = await api.settings();
            s.error = '';
        } catch (err) {
            if (s.data) ui.toast(`Settings refresh failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            else s.error = err.message;
        }
        render();
    };

    const loadDefaults = async () => {
        try {
            s.defaults = await api.defaults();
            s.defaultsError = '';
        } catch (err) {
            if (s.defaults) ui.toast(`Defaults refresh failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            else s.defaultsError = err.message;
        }
        render();
    };

    const load = () => Promise.all([loadSettings(), loadDefaults()]);

    const update = () => {
        s.loading = load();
        return s.loading;
    };

    const run = async (key, label, fn, success) => {
        s.pending.add(key);
        render();
        const res = await app.act(label, fn, { success });
        await s.loading;
        s.pending.delete(key);
        render();
        return res;
    };

    const show = (id) => {
        const next = sections.some((x) => x.id === id) ? id : 'accounts';
        if (next === s.section) return;
        s.section = next;
        history.pushState(null, '', `#/settings/${next}`);
        render();
    };

    const track = async () => {
        if (busy('track')) return;
        const repo = s.track.repo.trim();
        if (!repo) {
            s.track.error = 'Enter a repository as owner/repo';
            render();
            return;
        }
        s.track.error = '';
        const res = await run('track', 'Track preset', () => api.addRemotePreset(repo, s.track.path.trim()), (r) => `Tracking ${r.name}`);
        if (res.ok) s.track = { repo: '', path: '', error: '' };
        else s.track.error = res.error.message;
        render();
    };

    const field = (label, name, value, placeholder) => {
        const error = s.editor.errors[name];
        return `<label class="flex flex-col gap-1.5">
            <span class="px-1 ${ui.cls.label}">${label}</span>
            <input data-field="${name}" value="${esc(value)}" placeholder="${esc(placeholder)}" autocomplete="off" spellcheck="false" class="${ui.cls.input} font-mono">
            <span data-error="${name}" class="${error ? '' : 'hidden'} px-1 text-xs text-red">${esc(error || '')}</span>
        </label>`;
    };

    const rowList = (list, title, noun, rows) => {
        const pair = list !== 'args';
        const keyHint = { headers: 'Authorization', env: 'API_KEY' }[list];
        return `<section class="flex flex-col gap-2">
            <header class="flex items-center gap-2 pl-1">
                <h3 class="flex-1 ${ui.cls.label}">${title}</h3>
                ${ui.button(`Add ${noun}`, { icon: 'plus', size: 'sm', variant: 'ghost', action: 'settings-mcp-row-add', attrs: `data-list="${list}"` })}
            </header>
            ${rows.length ? `<ul class="flex flex-col gap-2">${rows.map((row, i) => `<li class="flex items-center gap-2">
                ${pair
                    ? `<input data-field="${list}.${i}.0" value="${esc(row[0])}" placeholder="${keyHint}" aria-label="${noun} name" autocomplete="off" spellcheck="false" class="${ui.cls.input} min-w-0 flex-[2] font-mono">
                       <input data-field="${list}.${i}.1" value="${esc(row[1])}" placeholder="value" aria-label="${noun} value" autocomplete="off" spellcheck="false" class="${ui.cls.input} min-w-0 flex-[3] font-mono">`
                    : `<input data-field="${list}.${i}" value="${esc(row)}" placeholder="argument ${i + 1}" aria-label="Argument ${i + 1}" autocomplete="off" spellcheck="false" class="${ui.cls.input} min-w-0 flex-1 font-mono">`}
                ${ui.iconButton('x', { title: `Remove ${noun}`, size: 'sm', action: 'settings-mcp-row-remove', attrs: `data-list="${list}" data-index="${i}"` })}
            </li>`).join('')}</ul>` : `<p class="px-1 text-xs text-overlay1">None</p>`}
        </section>`;
    };

    const renderFields = () => {
        const ed = s.editor;
        const target = ed.form.querySelector('[data-mcp-fields]');
        target.innerHTML = ed.transport === 'http'
            ? `${field('URL', 'url', ed.url, 'https://mcp.example.com/mcp')}${rowList('headers', 'Headers', 'header', ed.headers)}`
            : `${field('Command', 'command', ed.command, 'npx')}${rowList('args', 'Arguments', 'argument', ed.args)}${rowList('env', 'Environment', 'variable', ed.env)}`;
        ui.icons(target);
    };

    const renderFooter = () => {
        const ed = s.editor;
        const footer = ed.modal.el.querySelector('[data-modal-footer]');
        footer.innerHTML = `${ui.button('Cancel', { variant: 'ghost', action: 'settings-mcp-cancel' })}
            ${ui.button(ed.saving ? 'Saving' : 'Save', { icon: ed.saving ? 'loader-circle' : 'check', spin: ed.saving, disabled: ed.saving, variant: 'primary', type: 'submit', attrs: 'form="settings-mcp-form"' })}`;
        ui.icons(footer);
    };

    const showErrors = () => {
        s.editor.form.querySelectorAll('[data-error]').forEach((n) => {
            const msg = s.editor.errors[n.dataset.error];
            n.textContent = msg || '';
            n.classList.toggle('hidden', !msg);
        });
    };

    const validate = (ed) => {
        const errors = {};
        const name = ed.name.trim();
        if (!name) errors.name = 'Name is required';
        else if (name !== ed.prev && s.data.mcpLibrary.some((m) => m.name === name)) errors.name = `${name} already exists`;
        if (ed.transport === 'http' && !/^https?:\/\/\S+$/.test(ed.url.trim())) errors.url = 'Enter an http or https URL';
        if (ed.transport === 'stdio' && !ed.command.trim()) errors.command = 'Command is required';
        return errors;
    };

    const save = async () => {
        const ed = s.editor;
        if (!ed || ed.saving) return;
        ed.errors = validate(ed);
        showErrors();
        const invalid = Object.keys(ed.errors)[0];
        if (invalid) {
            ed.form.querySelector(`[data-field="${invalid}"]`)?.focus();
            return;
        }
        const pairs = (rows) => Object.fromEntries(rows.map(([k, v]) => [k.trim(), v]).filter(([k]) => k));
        const name = ed.name.trim();
        const entry = ed.transport === 'http'
            ? { name, transport: 'http', url: ed.url.trim(), headers: pairs(ed.headers) }
            : { name, transport: 'stdio', command: ed.command.trim(), args: ed.args.filter((a) => a !== ''), env: pairs(ed.env) };
        ed.saving = true;
        renderFooter();
        const res = await app.act('Save MCP server', () => api.saveMcp(entry, ed.prev ?? undefined), { success: `${name} saved` });
        await s.loading;
        if (res.ok) {
            if (ed.prev && ed.prev !== name && s.revealed.delete(ed.prev)) s.revealed.add(name);
            ed.modal.close();
            render();
        } else {
            ed.saving = false;
            renderFooter();
        }
    };

    const openEditor = (entry) => {
        const rows = (o) => {
            const r = Object.entries(o || {});
            return r.length ? r : [['', '']];
        };
        const ed = {
            prev: entry?.name ?? null,
            name: entry?.name ?? '',
            transport: entry?.transport ?? 'http',
            url: entry?.url ?? '',
            headers: rows(entry?.headers),
            command: entry?.command ?? '',
            args: entry?.args?.length ? [...entry.args] : [''],
            env: rows(entry?.env),
            errors: {},
            saving: false,
        };
        s.editor = ed;
        ed.modal = ui.modal.open({
            title: entry ? `Edit ${entry.name}` : 'Add MCP server',
            icon: 'server',
            body: `<form id="settings-mcp-form" data-form="mcp" novalidate class="flex flex-col gap-5 pt-1">
                ${field('Name', 'name', ed.name, 'github')}
                <span class="flex flex-col items-start gap-1.5">
                    <span class="px-1 ${ui.cls.label}">Transport</span>
                    ${ui.segmented({ name: 'mcp-transport', value: ed.transport, options: [{ value: 'http', label: 'HTTP', icon: 'globe' }, { value: 'stdio', label: 'stdio', icon: 'square-terminal' }] })}
                </span>
                <div data-mcp-fields class="flex flex-col gap-5"></div>
            </form>`,
            footer: ' ',
            onClose: () => {
                if (s.editor === ed) s.editor = null;
            },
        });
        ed.form = ed.modal.el.querySelector('[data-form="mcp"]');
        renderFields();
        renderFooter();
        ui.icons(ed.modal.el);
        ed.form.addEventListener('submit', (e) => {
            e.preventDefault();
            save();
        });
        ed.modal.el.addEventListener('ui-change', (e) => {
            if (e.detail.name !== 'mcp-transport') return;
            ed.transport = e.detail.value;
            ed.errors = {};
            renderFields();
        });
        ed.modal.el.addEventListener('input', (e) => {
            const key = e.target.dataset.field;
            if (!key) return;
            const [list, i, j] = key.split('.');
            if (i === undefined) ed[list] = e.target.value;
            else if (j === undefined) ed[list][i] = e.target.value;
            else ed[list][i][j] = e.target.value;
            if (ed.errors[key]) {
                delete ed.errors[key];
                showErrors();
            }
        });
        if (!ui.media.mobile()) ed.form.querySelector('[data-field="name"]').focus();
    };

    const valueField = (kind, value, options) => {
        switch (kind) {
        case 'boolean':
            return ui.segmented({ name: 'default-value', value: String(value === true), options: [{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }] });
        case 'enum':
            return ui.select({ name: 'default-value', value: value ?? '', options: options.map((v) => ({ value: v, label: v })), placeholder: 'Pick a value' });
        case 'string':
        case 'number':
        case 'integer':
            return `<input data-default-input value="${esc(value)}" ${kind === 'string' ? '' : 'inputmode="decimal"'} aria-label="Value" autocomplete="off" spellcheck="false" class="${ui.cls.input} font-mono">`;
        default:
            return `<textarea data-default-input rows="10" aria-label="Value" spellcheck="false" class="${ui.cls.input} min-h-40 resize-y font-mono leading-relaxed">${esc(value === undefined ? '' : JSON.stringify(value, null, 2))}</textarea>`;
        }
    };

    const readValue = (root, kind) => {
        if (kind === 'boolean') return { value: ui.value(root, 'default-value') === 'true' };
        if (kind === 'enum') {
            const v = ui.value(root, 'default-value');
            return v ? { value: v } : { error: 'Pick a value' };
        }
        const raw = root.querySelector('[data-default-input]').value;
        if (kind === 'string') return { value: raw };
        if (kind === 'number' || kind === 'integer') {
            const n = Number(raw.trim());
            if (raw.trim() === '' || !Number.isFinite(n)) return { error: 'Enter a number' };
            if (kind === 'integer' && !Number.isInteger(n)) return { error: 'Enter a whole number' };
            return { value: n };
        }
        try {
            return { value: JSON.parse(raw) };
        } catch (err) {
            return { error: `Not valid JSON: ${err.message}` };
        }
    };

    const openDefault = (row) => {
        const ed = { kind: row?.kind ?? 'string', saving: false };
        const initial = row ? (row.active ? row.value : row.default) : undefined;
        const m = ui.modal.open({
            title: row ? row.key : 'Add a key',
            icon: 'sliders-horizontal',
            body: `<form id="settings-default-form" novalidate class="flex flex-col gap-5 pt-1">
                ${row
                    ? (row.description ? `<p class="text-sm leading-relaxed text-subtext0">${esc(row.description)}</p>` : '')
                    : `<label class="flex flex-col gap-1.5">
                        <span class="px-1 ${ui.cls.label}">Key</span>
                        <input data-default-key placeholder="env.MY_VARIABLE" autocomplete="off" spellcheck="false" class="${ui.cls.input} font-mono">
                    </label>
                    <div class="flex flex-col items-start gap-1.5">
                        <span class="px-1 ${ui.cls.label}">Type</span>
                        ${ui.segmented({ name: 'default-kind', value: ed.kind, options: [{ value: 'string', label: 'Text' }, { value: 'json', label: 'JSON' }] })}
                    </div>`}
                <div class="flex flex-col gap-1.5">
                    <span class="px-1 ${ui.cls.label}">Value</span>
                    <div data-default-field class="flex flex-col">${valueField(ed.kind, initial, row?.enum)}</div>
                </div>
                <p data-default-error role="alert" class="hidden px-1 text-sm text-red"></p>
            </form>`,
            footer: ' ',
        });
        const form = m.el.querySelector('form');
        const error = form.querySelector('[data-default-error]');
        const footer = m.el.querySelector('[data-modal-footer]');
        const renderFooter = () => {
            footer.innerHTML = `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-default-cancel' })}
                ${ui.button(ed.saving ? 'Saving' : 'Save', { icon: ed.saving ? 'loader-circle' : 'check', spin: ed.saving, disabled: ed.saving, variant: 'primary', type: 'submit', attrs: 'form="settings-default-form"' })}`;
            ui.icons(footer);
        };
        const showError = (msg) => {
            error.textContent = msg;
            error.classList.toggle('hidden', !msg);
        };
        renderFooter();
        footer.addEventListener('click', (e) => {
            if (e.target.closest('[data-default-cancel]')) m.close();
        });
        m.el.addEventListener('ui-change', (e) => {
            if (e.detail.name !== 'default-kind') return;
            ed.kind = e.detail.value;
            const field = form.querySelector('[data-default-field]');
            field.innerHTML = valueField(ed.kind, undefined);
            showError('');
        });
        form.addEventListener('input', () => showError(''));
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (ed.saving) return;
            const key = row ? row.key : form.querySelector('[data-default-key]').value.trim();
            const res = readValue(form, ed.kind);
            const msg = key ? res.error : 'Enter a key';
            if (msg) {
                showError(msg);
                return;
            }
            ed.saving = true;
            renderFooter();
            const out = await app.act('Save default', () => api.setDefault(key, res.value), { success: `${key} is active` });
            await s.loading;
            if (out.ok) m.close();
            else {
                ed.saving = false;
                renderFooter();
                showError(out.error.message);
            }
        });
        if (!ui.media.mobile()) form.querySelector('[data-default-key], [data-default-input]')?.focus();
    };

    const rowOf = (el) => s.defaults.rows.find((r) => r.key === el.dataset.key);

    ui.on('settings-default-add', () => openDefault(null));
    ui.on('settings-default-edit', (el) => openDefault(rowOf(el)));

    ui.on('settings-default-accept', (el) => {
        const row = rowOf(el);
        run(`default:${row.key}`, 'Accept default', () => api.setDefault(row.key, row.default), `${row.key} is active`);
    });

    ui.on('settings-default-reset', async (el) => {
        const key = el.dataset.key;
        const ok = await ui.confirm({
            title: `Reset ${key}?`,
            icon: 'rotate-ccw',
            confirmLabel: 'Reset',
            message: `Takes <span class="font-mono text-text">${esc(key)}</span> out of the active set. The next apply removes it from every account claudex wrote it to.`,
        });
        if (ok) await run(`default:${key}`, 'Reset default', () => api.resetDefault(key), `${key} reset`);
    });

    ui.on('settings-schema-refresh', () => {
        run('schema', 'Refresh schema', () => api.refreshSchema(), (r) => `Schema refreshed: ${plural(r.keys, 'key')}`);
    });

    ui.on('settings-section', (el) => show(el.dataset.section));
    ui.on('settings-retry', () => {
        s.error = '';
        s.defaultsError = '';
        render();
        update();
    });

    ui.on('settings-apply', async (el) => {
        const dir = el.dataset.dir;
        const account = s.data.accounts.find((a) => a.dir === dir);
        const target = `<span class="font-mono text-text">${esc(dir)}/settings.json</span>`;
        const ok = await ui.confirm({
            title: `Apply defaults to ${dir}`,
            icon: 'wand-sparkles',
            confirmLabel: 'Apply defaults',
            message: account.settingsFound
                ? `Sets ${plural(account.drift.length, 'drifted key')} in ${target} to the configure defaults.`
                : `Creates ${target} with the ${plural(s.data.defaultsCount, 'active default')}.`,
        });
        if (ok) await run(`apply:${dir}`, 'Apply defaults', () => api.applyDefaults(dir), `${dir} matches the defaults`);
    });

    ui.on('settings-preset-update', (el) => {
        const name = el.dataset.name;
        run(`update:${name}`, 'Update preset', () => api.updateRemotePreset(name), `${name} is up to date`);
    });

    ui.on('settings-preset-remove', async (el) => {
        const name = el.dataset.name;
        const preset = s.data.presets.remote.find((p) => p.name === name);
        const ok = await ui.confirm({
            title: `Stop tracking ${name}?`,
            icon: 'trash-2',
            danger: true,
            confirmLabel: 'Remove',
            message: `Applying presets no longer pulls <span class="font-mono text-text">${esc(name)}</span> from <span class="font-mono text-text">${esc(preset.repo)}</span>, and the new-session modal stops offering it.`,
        });
        if (ok) await run(`remove:${name}`, 'Remove preset', () => api.removeRemotePreset(name), `Stopped tracking ${name}`);
    });

    ui.on('settings-mcp-add', () => openEditor(null));
    ui.on('settings-mcp-edit', (el) => openEditor(s.data.mcpLibrary.find((m) => m.name === el.dataset.name)));
    ui.on('settings-mcp-cancel', () => s.editor?.modal.close());

    ui.on('settings-mcp-reveal', (el) => {
        const name = el.dataset.name;
        if (!s.revealed.delete(name)) s.revealed.add(name);
        render();
    });

    ui.on('settings-mcp-delete', async (el) => {
        const name = el.dataset.name;
        const ok = await ui.confirm({
            title: `Delete ${name}?`,
            icon: 'trash-2',
            danger: true,
            confirmLabel: 'Delete',
            message: `Removes <span class="font-mono text-text">${esc(name)}</span> and its stored credentials from the library.`,
        });
        if (!ok) return;
        const res = await run(`mcp:${name}`, 'Delete MCP server', () => api.deleteMcp(name), `${name} deleted`);
        if (res.ok) s.revealed.delete(name);
    });

    ui.on('settings-mcp-row-add', (el) => {
        const ed = s.editor;
        const list = el.dataset.list;
        ed[list].push(list === 'args' ? '' : ['', '']);
        renderFields();
        ed.form.querySelector(`[data-field="${list}.${ed[list].length - 1}${list === 'args' ? '' : '.0'}"]`)?.focus();
    });

    ui.on('settings-mcp-row-remove', (el) => {
        s.editor[el.dataset.list].splice(Number(el.dataset.index), 1);
        renderFields();
    });

    return {
        mount(el, params) {
            s.el = el;
            s.section = sections.some((x) => x.id === params[0]) ? params[0] : 'accounts';
            s.listeners = [
                ['input', (e) => {
                    const key = e.target.dataset.field;
                    if (key === 'defaults.query') {
                        s.dquery = e.target.value;
                        render();
                        return;
                    }
                    if (!key?.startsWith('track.')) return;
                    s.track[key.slice(6)] = e.target.value;
                    if (s.track.error) {
                        s.track.error = '';
                        s.el.querySelector('[data-form="track"] [role="alert"]')?.remove();
                    }
                }],
                ['submit', (e) => {
                    if (e.target.dataset.form !== 'track') return;
                    e.preventDefault();
                    track();
                }],
                ['ui-change', (e) => {
                    if (e.detail.name === 'settings-section') show(e.detail.value);
                    if (e.detail.name === 'defaults-filter') {
                        s.dfilter = e.detail.value;
                        render();
                    }
                }],
            ];
            s.listeners.forEach(([type, fn]) => el.addEventListener(type, fn));
            render();
            update();
        },

        update,

        unmount() {
            s.listeners.forEach(([type, fn]) => s.el.removeEventListener(type, fn));
            s.listeners = [];
            s.editor?.modal.close();
            s.el = null;
        },
    };
})();
