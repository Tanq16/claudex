views.settings = (() => {
    const sections = [
        { id: 'accounts', label: 'Accounts', icon: 'users-round' },
        { id: 'defaults', label: 'Defaults', icon: 'sliders-horizontal' },
        { id: 'presets', label: 'Presets', icon: 'package' },
        { id: 'mcp', label: 'MCP library', icon: 'server' },
    ];

    const s = {
        el: null,
        section: 'accounts',
        data: null,
        error: '',
        defaults: null,
        defaultsError: '',
        dmode: 'keys',
        dquery: '',
        open: new Set(),
        json: { text: '', dirty: false, error: '', saving: false },
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
        const attention = s.data?.accounts.some((a) => compliance(a).tone !== 'green' || a.unmanaged?.length);
        return `<header class="flex shrink-0 flex-wrap items-center justify-between gap-3">
            <h1 class="font-display text-2xl text-text max-md:hidden">Settings</h1>
            <nav aria-label="Settings sections" class="inline-flex max-w-full gap-1 overflow-x-auto rounded-full bg-mantle p-1 max-md:w-full 3xl:hidden">
                ${sections.map((x) => {
                    const on = x.id === s.section;
                    return `<a href="#/settings/${x.id}" data-action="settings-section" data-section="${x.id}" ${on ? 'aria-current="page"' : ''} class="flex h-8 items-center justify-center gap-2 rounded-full px-4 text-sm whitespace-nowrap transition-colors max-md:flex-1 ${ui.cls.focus} ${on ? 'bg-surface0 text-text' : 'text-overlay1 hover:text-text'}">
                        ${icon(x.icon, `size-4 ${on ? 'text-mauve' : ''}`)}<span class="${on ? '' : 'max-sm:hidden'}">${x.label}</span>
                        ${x.id === 'accounts' && attention ? '<span data-tip="An account needs attention" class="size-1.5 rounded-full bg-peach"></span>' : ''}
                        ${c[x.id] === undefined ? '' : `<span class="text-xs tabular-nums text-overlay1 max-sm:hidden">${c[x.id]}</span>`}
                    </a>`;
                }).join('')}
            </nav>
        </header>`;
    };

    const driftRow = (d) => `<li class="rounded-xl bg-surface0/50 px-3 py-2 font-mono text-xs">
        <p class="break-all text-subtext1">${esc(d.key)}</p>
        <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
            <span data-tip="Current" class="break-all ${d.actual === 'missing' ? 'italic text-overlay1' : 'text-peach line-through decoration-peach/50'}">${esc(d.actual)}</span>
            <i data-lucide="arrow-right" class="size-3 shrink-0 text-overlay0"></i>
            <span data-tip="Default" class="break-all text-green">${esc(d.expected)}</span>
        </p>
    </li>`;

    const unmanagedRow = (a, k) => {
        const attrs = `data-dir="${esc(a.dir)}" data-key="${esc(k.key)}"`;
        const pending = busy(`key:${a.dir}:${k.key}`);
        return `<li class="flex items-center gap-2 rounded-xl bg-surface0/50 py-1.5 pr-1.5 pl-3">
            <p class="min-w-0 flex-1 font-mono text-xs">
                <span data-tip="${esc(k.key)}" class="block truncate text-subtext1">${esc(k.key)}</span>
                <span data-tip="${esc(k.value)}" class="block truncate text-blue">${esc(k.value)}</span>
            </p>
            ${ui.button('Adopt', { icon: pending ? 'loader-circle' : 'import', spin: pending, disabled: pending, size: 'sm', variant: 'ghost', action: 'settings-adopt', attrs })}
            ${ui.iconButton('trash-2', { tip: 'Remove from this account', size: 'sm', variant: 'danger', action: 'settings-remove-key', disabled: pending, attrs })}
        </li>`;
    };

    const accountCard = (a) => {
        const c = compliance(a);
        const pending = busy(`apply:${a.dir}`);
        const unmanaged = a.unmanaged || [];
        return `<li class="flex flex-col gap-3 rounded-2xl bg-base p-4">
            <header class="flex items-start gap-3">
                ${circle(icon('user-round'))}
                <hgroup class="min-w-0 flex-1">
                    <h3 class="truncate text-sm font-medium text-text">${esc(a.label)}</h3>
                    <p class="truncate font-mono text-xs text-overlay1">${esc(a.dir)}</p>
                </hgroup>
                ${a.isDefault ? ui.chip('Default', { icon: 'star' }) : ''}
            </header>
            <p class="flex flex-wrap items-center gap-2">${ui.chip(c.text, { tone: c.tone, icon: c.icon })}${unmanaged.length ? ui.chip(`${plural(unmanaged.length, 'key')} unmanaged`, { tone: 'blue', icon: 'shield-question' }) : ''}${ui.chip(plural(a.projects, 'project'), { icon: 'folder' })}</p>
            ${a.drift.length ? `<ul class="flex flex-col gap-1.5">${a.drift.map(driftRow).join('')}</ul>` : ''}
            ${unmanaged.length ? `<section class="flex flex-col gap-1.5">
                ${groupHeader('Not in the defaults', unmanaged.length)}
                <ul class="flex flex-col gap-1.5">${unmanaged.map((k) => unmanagedRow(a, k)).join('')}</ul>
            </section>` : ''}
            ${a.settingsFound ? '' : `<p class="text-sm text-overlay1">Applying defaults creates <span class="font-mono text-subtext1">${esc(a.dir)}/settings.json</span>.</p>`}
            ${c.tone === 'green' ? '' : `<footer class="flex justify-end">${ui.button(pending ? 'Applying' : 'Apply defaults', { icon: pending ? 'loader-circle' : 'wand-sparkles', spin: pending, disabled: pending, size: 'sm', action: 'settings-apply', attrs: `data-dir="${esc(a.dir)}"` })}</footer>`}
        </li>`;
    };

    const accountsBody = () => `
        <p class="flex items-center gap-3 rounded-2xl bg-base px-4 py-3">
            <i data-lucide="folder-root" class="size-4 shrink-0 text-overlay1"></i>
            <span class="${ui.cls.label}">Projects root</span>
            <span class="min-w-0 flex-1 truncate text-right font-mono text-sm text-text">${esc(s.data.root)}</span>
            <span data-tip="Read-only" class="text-overlay0">${icon('lock', 'size-3.5')}</span>
        </p>
        <a href="#/settings/defaults" data-action="settings-section" data-section="defaults" class="flex items-center gap-3 rounded-2xl bg-base px-4 py-3 transition-colors hover:bg-surface0 3xl:hidden">
            <i data-lucide="sliders-horizontal" class="size-4 shrink-0 text-overlay1"></i>
            <span class="${ui.cls.label}">Defaults</span>
            <span class="min-w-0 flex-1 text-right text-sm text-text">${plural(s.data.defaultsCount, 'active key')}</span>
            <i data-lucide="chevron-right" class="size-4 shrink-0 text-overlay0"></i>
        </a>
        <ul class="grid items-start gap-3 xl:grid-cols-2 3xl:grid-cols-1">${s.data.accounts.map(accountCard).join('')}</ul>`;

    const valueText = (v) => (v === '' ? '""' : fmt(v));
    const groupOf = (key) => (key.includes('.') ? key.split('.')[0] : '');
    const summary = (v) => (Array.isArray(v) ? plural(v.length, 'item') : v && typeof v === 'object' ? plural(Object.keys(v).length, 'key') : valueText(v));

    const schemaLine = (sc) => (sc.error
        ? `<p class="flex items-center gap-2 rounded-2xl bg-red/10 px-4 py-3 text-sm text-red">${icon('cloud-off', 'size-4')}<span class="min-w-0 flex-1">${esc(sc.error)}</span></p>`
        : `<p class="flex items-center gap-2 px-1 text-xs text-overlay1">${icon('file-json', 'size-3.5')}<span class="min-w-0 flex-1 truncate">${plural(sc.keys, 'key')} in the schema, fetched ${ui.timeAgo(sc.fetchedAt)}</span></p>`);

    const valueControl = (r) => {
        const attrs = `data-key="${esc(r.key)}"`;
        switch (r.kind) {
        case 'boolean':
            return ui.toggle({ name: `dv:${r.key}`, checked: r.value === true, label: r.key });
        case 'enum':
            return ui.select({ name: `dv:${r.key}`, value: r.value, pill: true, options: r.enum.map((v) => ({ value: v, label: v })) });
        case 'string':
        case 'number':
        case 'integer':
            return `<input data-dv="${esc(r.key)}" value="${esc(r.value)}" ${r.kind === 'string' ? '' : 'inputmode="decimal"'} aria-label="${esc(r.key)}" autocomplete="off" spellcheck="false" class="${ui.cls.input} h-9 py-1.5 font-mono sm:w-72">`;
        default:
            return ui.button(summary(r.value), { icon: 'braces', size: 'sm', action: 'settings-default-edit', attrs });
        }
    };

    const keyRow = (r, group) => {
        const attrs = `data-key="${esc(r.key)}"`;
        const pending = busy(`default:${r.key}`);
        const open = s.open.has(r.key);
        return `<li class="group flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-2xl px-3 py-1.5 transition-colors hover:bg-surface0/50">
            <span data-tip="${esc(r.key)}" class="min-w-0 flex-1 truncate font-mono text-sm text-text">${esc(group ? r.key.slice(group.length + 1) : r.key)}</span>
            ${r.inSchema ? '' : ui.chip('custom', { icon: 'pencil-line' })}
            <span class="flex shrink-0 items-center gap-1 ${['string', 'number', 'integer'].includes(r.kind) ? 'max-sm:basis-full' : ''}">
                ${pending ? ui.spinner('size-4 text-overlay1') : ''}
                ${valueControl(r)}
                ${r.description ? ui.iconButton('info', { tip: open ? 'Hide description' : 'Show description', size: 'sm', action: 'settings-default-info', pressed: open, attrs }) : '<span class="size-8 shrink-0"></span>'}
                <span class="${ui.cls.reveal}">${ui.iconButton('rotate-ccw', { tip: 'Reset', size: 'sm', action: 'settings-default-reset', disabled: pending, attrs })}</span>
            </span>
            ${open ? `<p class="basis-full pb-1 text-xs leading-relaxed text-subtext0">${esc(r.description)}</p>` : ''}
        </li>`;
    };

    const keysView = (d) => {
        const q = s.dquery.trim().toLowerCase();
        const active = d.rows.filter((r) => r.active && (!q || r.key.toLowerCase().includes(q) || (r.description || '').toLowerCase().includes(q)));
        const groups = new Map();
        for (const r of active) {
            const g = groupOf(r.key);
            if (!groups.has(g)) groups.set(g, []);
            groups.get(g).push(r);
        }
        const names = [...groups.keys()].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b)));
        const envAdd = ui.button('Variable', { icon: 'plus', size: 'sm', variant: 'ghost', action: 'settings-env-add' });
        return `${names.length
            ? names.map((g) => `<section class="flex flex-col gap-1">
                ${groupHeader(g ? `<span class="font-mono normal-case tracking-normal">${esc(g)}</span>` : 'General', groups.get(g).length, g === 'env' ? envAdd : '')}
                <ul class="flex flex-col">${groups.get(g).map((r) => keyRow(r, g)).join('')}</ul>
            </section>`).join('')
            : ui.empty({ icon: q ? 'search-x' : 'sliders-horizontal', title: q ? 'No matching keys' : 'No active keys', action: q ? '' : ui.button('Add key', { icon: 'plus', action: 'settings-default-add' }) })}
            ${!q ? `<section class="flex flex-col gap-2">
                ${groupHeader('Managed by claudex', d.owned.length, ui.chip('per account', { icon: 'lock' }))}
                <p class="flex flex-wrap gap-2 px-1">${d.owned.map((k) => ui.chip(k, { mono: true })).join('')}</p>
            </section>` : ''}`;
    };

    const sorted = (v) => (Array.isArray(v) ? v.map(sorted) : v && typeof v === 'object' ? Object.fromEntries(Object.keys(v).sort().map((k) => [k, sorted(v[k])])) : v);
    const documentText = () => (s.json.dirty ? s.json.text : JSON.stringify(sorted(s.defaults.document), null, 2));

    const jsonView = () => `<section class="flex min-h-0 flex-1 flex-col gap-2">
        <textarea data-field="defaults.json" aria-label="Defaults as settings.json" spellcheck="false" class="${ui.cls.input} min-h-64 flex-1 resize-none font-mono text-xs leading-relaxed">${esc(documentText())}</textarea>
        <p data-json-error role="alert" class="${s.json.error ? '' : 'hidden'} px-1 text-sm whitespace-pre-line text-red">${esc(s.json.error)}</p>
        <footer class="flex flex-wrap items-center justify-end gap-2">
            <span class="mr-auto px-1 text-xs text-overlay1">statusLine stays out of this document</span>
            ${ui.button('Revert', { icon: 'undo-2', size: 'sm', variant: 'ghost', action: 'settings-json-revert', disabled: !s.json.dirty || s.json.saving, attrs: 'data-json-btn' })}
            ${ui.button(s.json.saving ? 'Saving' : 'Save', { icon: s.json.saving ? 'loader-circle' : 'check', spin: s.json.saving, size: 'sm', variant: 'primary', action: 'settings-json-save', disabled: !s.json.dirty || s.json.saving, attrs: 'data-json-btn' })}
        </footer>
    </section>`;

    const defaultsBody = () => {
        const d = s.defaults;
        if (!d && s.defaultsError) {
            return ui.empty({ icon: 'circle-alert', title: 'Could not load defaults', hint: esc(s.defaultsError), action: ui.button('Retry', { icon: 'refresh-cw', action: 'settings-retry' }) });
        }
        if (!d) return `<p class="flex flex-1 items-center justify-center text-overlay1">${icon('loader-circle', 'size-5 animate-spin')}</p>`;
        return `${schemaLine(d.schema)}
            <div class="flex flex-wrap items-center gap-2">
                ${ui.segmented({ name: 'defaults-mode', value: s.dmode, options: [{ value: 'keys', label: 'Keys', icon: 'list' }, { value: 'json', label: 'JSON', icon: 'braces' }] })}
                ${s.dmode === 'keys' ? `<span class="min-w-40 flex-1">${ui.search({ name: 'defaults-query', value: s.dquery, placeholder: 'Filter active keys', attrs: 'data-field="defaults.query" aria-label="Filter keys"' })}</span>` : ''}
            </div>
            ${s.dmode === 'json' ? jsonView() : `<div data-keys class="contents">${keysView(d)}</div>`}`;
    };

    const renderKeys = () => {
        const host = s.el?.querySelector('[data-keys]');
        if (!host) return render();
        host.innerHTML = keysView(s.defaults);
        ui.icons(host);
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
                ${ui.iconButton(removing ? 'loader-circle' : 'trash-2', { tip: 'Remove', variant: 'danger', size: 'sm', action: 'settings-preset-remove', spin: removing, disabled: removing || updating, attrs })}
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
                : ui.empty({ icon: 'package-open', title: 'No tracked presets', hint: 'Presets tracked from a GitHub repository show up here and in the new-session modal.', fill: false })}
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
            <p class="rounded-xl bg-surface0/50 px-3 py-2 font-mono text-xs leading-relaxed break-all text-subtext1">${esc(http ? m.url : commandLine(m))}</p>
            ${secrets.length ? `<section class="flex flex-col gap-1.5">
                <header class="flex items-center gap-2 pl-1">
                    <h5 class="flex flex-1 items-center gap-1.5 ${ui.cls.label}">${icon('key-round', 'size-3.5')}${http ? 'Headers' : 'Env'}</h5>
                    ${ui.iconButton(shown ? 'eye-off' : 'eye', { tip: shown ? 'Hide secrets' : 'Reveal secrets', size: 'sm', action: 'settings-mcp-reveal', attrs })}
                </header>
                <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1.5 rounded-xl bg-surface0/50 px-3 py-2 font-mono text-xs">
                    ${secrets.map(([k, v]) => `<dt class="text-overlay1">${esc(k)}</dt><dd class="break-all ${shown ? 'text-text' : 'tracking-widest text-overlay0'}">${shown ? esc(v) : '••••••••••••'}</dd>`).join('')}
                </dl>
            </section>` : `<p class="flex items-center gap-1.5 px-1 text-xs text-overlay1">${icon('key-round', 'size-3.5')}No ${http ? 'headers' : 'env'} set</p>`}
            <footer class="mt-auto flex items-center justify-end gap-1">
                ${ui.button('Edit', { icon: 'pencil', size: 'sm', variant: 'ghost', action: 'settings-mcp-edit', disabled: deleting, attrs })}
                ${ui.iconButton(deleting ? 'loader-circle' : 'trash-2', { tip: 'Delete', variant: 'danger', size: 'sm', action: 'settings-mcp-delete', spin: deleting, disabled: deleting, attrs })}
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
            <h2 id="settings-${x.id}" class="min-w-0 flex-1 font-display text-lg font-medium text-text">${x.label}</h2>
            ${x.id === 'mcp' && s.data?.mcpLibrary.length ? ui.button('Add server', { icon: 'plus', size: 'sm', action: 'settings-mcp-add', collapse: true }) : ''}
            ${x.id === 'defaults' ? `${ui.iconButton(busy('schema') ? 'loader-circle' : 'cloud-download', { tip: 'Refresh upstream schema', size: 'sm', variant: 'secondary', action: 'settings-schema-refresh', spin: busy('schema'), disabled: busy('schema') })}
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
        const before = JSON.stringify([s.data, s.error]);
        try {
            s.data = await api.settings();
            s.error = '';
        } catch (err) {
            if (s.data) ui.toast(`Settings refresh failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            else s.error = err.message;
        }
        if (JSON.stringify([s.data, s.error]) !== before) render();
    };

    const loadDefaults = async () => {
        const before = JSON.stringify([s.defaults, s.defaultsError]);
        try {
            s.defaults = await api.defaults();
            s.defaultsError = '';
        } catch (err) {
            if (s.defaults) ui.toast(`Defaults refresh failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            else s.defaultsError = err.message;
        }
        if (JSON.stringify([s.defaults, s.defaultsError]) !== before) render();
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

    const field = (label, name, value, placeholder) => ui.field(label, ui.input({ name: `mcp-${name}`, value, placeholder, mono: true, attrs: `data-field="${name}"` }));

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
                ${ui.iconButton('x', { tip: `Remove ${noun}`, size: 'sm', action: 'settings-mcp-row-remove', attrs: `data-list="${list}" data-index="${i}"` })}
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
        for (const key of ['name', 'url', 'command']) {
            const input = s.editor.form.querySelector(`[data-field="${key}"]`);
            if (input) ui.invalid(input, s.editor.errors[key] || '');
        }
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
                ${ui.field('Transport', `<span>${ui.segmented({ name: 'mcp-transport', value: ed.transport, options: [{ value: 'http', label: 'HTTP', icon: 'globe' }, { value: 'stdio', label: 'stdio', icon: 'square-terminal' }] })}</span>`, { group: true })}
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
            return ui.input({ name: 'default-input', value: value ?? '', mono: true, attrs: `data-default-input ${kind === 'string' ? '' : 'inputmode="decimal"'}` });
        default:
            return ui.textarea({ name: 'default-input', rows: 10, mono: true, value: value === undefined ? '' : JSON.stringify(value, null, 2), attrs: 'data-default-input' });
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

    const openDefault = (row, presetKey = '') => {
        const ed = { kind: row?.kind ?? 'string', saving: false };
        const initial = row ? (row.active ? row.value : row.default) : undefined;
        const m = ui.modal.open({
            title: row ? row.key : 'Add a key',
            icon: 'sliders-horizontal',
            body: `<form id="settings-default-form" novalidate class="flex flex-col gap-5 pt-1">
                ${row
                    ? (row.description ? `<p class="text-sm leading-relaxed text-subtext0">${esc(row.description)}</p>` : '')
                    : `${ui.field('Key', ui.input({ name: 'default-key', value: presetKey, placeholder: 'env.MY_VARIABLE', mono: true, attrs: 'data-default-key' }))}
                    ${ui.field('Type', `<span>${ui.segmented({ name: 'default-kind', value: ed.kind, options: [{ value: 'string', label: 'Text' }, { value: 'json', label: 'JSON' }] })}</span>`, { group: true })}`}
                ${ui.field('Value', `<div data-default-field class="flex flex-col">${valueField(ed.kind, initial, row?.enum)}</div>`, { group: ed.kind === 'boolean' })}
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
        const valueControl = () => form.querySelector('[data-default-field] :is([data-default-input], [data-ui])');
        m.el.addEventListener('ui-change', (e) => {
            if (e.detail.name === 'default-value') ui.invalid(e.target);
            if (e.detail.name !== 'default-kind') return;
            ed.kind = e.detail.value;
            ui.invalid(valueControl());
            form.querySelector('[data-default-field]').innerHTML = valueField(ed.kind, undefined);
            showError('');
        });
        form.addEventListener('input', (e) => {
            showError('');
            if (e.target.getAttribute('aria-invalid') === 'true') ui.invalid(e.target);
        });
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (ed.saving) return;
            const keyInput = form.querySelector('[data-default-key]');
            const key = row ? row.key : keyInput.value.trim();
            if (!key) return ui.invalid(keyInput, 'Enter a key');
            const res = readValue(form, ed.kind);
            if (res.error) return ui.invalid(valueControl(), res.error);
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
        if (!ui.media.mobile()) form.querySelector(presetKey ? '[data-default-input]' : '[data-default-key], [data-default-input]')?.focus();
    };

    const openPicker = (initial = '') => {
        const inactive = s.defaults.rows.filter((r) => !r.active);
        const known = new Set(s.defaults.rows.map((r) => r.key));
        const limit = 50;
        let query = initial;
        let active = 0;
        let items = [];
        let total = 0;
        const m = ui.modal.open({
            title: 'Add a key',
            icon: 'plus',
            body: `<div class="flex flex-col gap-3 pt-1">
                ${ui.search({ name: 'picker-q', value: initial, placeholder: `Search ${plural(inactive.length, 'key')}, or type a new one`, attrs: 'data-picker-q aria-label="Search keys" autofocus' })}
                <ul data-picker-list role="listbox" class="flex flex-col gap-0.5"></ul>
            </div>`,
        });
        const input = m.el.querySelector('[data-picker-q]');
        const list = m.el.querySelector('[data-picker-list]');
        const compute = () => {
            const q = query.trim();
            const lower = q.toLowerCase();
            const matched = q
                ? inactive.map((r) => ({ r, score: Math.max(ui.fuzzy(q, r.key), (r.description || '').toLowerCase().includes(lower) ? 1 : -1) }))
                    .filter((x) => x.score >= 0).sort((a, b) => b.score - a.score).map((x) => x.r)
                : inactive;
            total = matched.length;
            items = matched.slice(0, limit).map((row) => ({ key: row.key, row }));
            if (/^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$/.test(q) && !known.has(q)) items.push({ key: q, custom: true });
            active = Math.min(active, Math.max(0, items.length - 1));
        };
        const draw = () => {
            list.innerHTML = items.length
                ? items.map((it, i) => `<li role="option" data-index="${i}" aria-selected="${i === active}" class="flex flex-col gap-0.5 rounded-xl px-3 py-2 ${i === active ? 'bg-surface0' : ''}">
                    <span class="flex items-center gap-2"><span class="min-w-0 flex-1 truncate font-mono text-sm text-text">${esc(it.key)}</span>${it.custom ? ui.chip('new key', { icon: 'pencil-line' }) : ui.chip(it.row.kind, { mono: true })}</span>
                    ${it.row?.description ? `<span class="line-clamp-1 text-xs text-overlay1">${esc(it.row.description)}</span>` : ''}
                </li>`).join('') + (total > limit ? `<li class="px-3 py-2 text-center text-xs text-overlay1">Showing ${limit} of ${total}; keep typing to narrow</li>` : '')
                : '<li class="px-3 py-6 text-center text-sm text-overlay1">No matching keys</li>';
            ui.icons(list);
            list.querySelector(`[data-index="${active}"]`)?.scrollIntoView({ block: 'nearest' });
        };
        const choose = (i) => {
            const it = items[i];
            if (!it) return;
            m.close();
            if (it.custom) openDefault(null, it.key);
            else openDefault(it.row);
        };
        input.addEventListener('input', () => {
            query = input.value;
            active = 0;
            compute();
            draw();
        });
        input.addEventListener('keydown', (e) => {
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                active = Math.min(items.length - 1, active + 1);
                draw();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                active = Math.max(0, active - 1);
                draw();
            } else if (e.key === 'Enter') {
                e.preventDefault();
                choose(active);
            }
        });
        list.addEventListener('click', (e) => {
            const li = e.target.closest('[data-index]');
            if (li) choose(Number(li.dataset.index));
        });
        compute();
        draw();
        input.focus();
        input.setSelectionRange(input.value.length, input.value.length);
    };

    const saveValue = (key, value) => run(`default:${key}`, 'Save default', () => api.setDefault(key, value), `${key} saved`);

    const saveInput = (el) => {
        const row = s.defaults.rows.find((r) => r.key === el.dataset.dv);
        if (!row) return;
        const raw = el.value;
        if (row.kind === 'string') {
            if (raw !== row.value) saveValue(row.key, raw);
            return;
        }
        const n = Number(raw.trim());
        if (raw.trim() === '' || !Number.isFinite(n) || (row.kind === 'integer' && !Number.isInteger(n))) {
            ui.toast(`${row.key} takes ${row.kind === 'integer' ? 'a whole number' : 'a number'}`, { tone: 'red', icon: 'circle-alert' });
            render();
            return;
        }
        if (n !== row.value) saveValue(row.key, n);
    };

    const showJsonError = (msg) => {
        s.json.error = msg;
        const node = s.el?.querySelector('[data-json-error]');
        if (!node) return;
        node.textContent = msg;
        node.classList.toggle('hidden', !msg);
    };

    const saveJson = async () => {
        if (s.json.saving || !s.json.dirty) return;
        let doc;
        try {
            doc = JSON.parse(s.json.text);
        } catch (err) {
            showJsonError(`Not valid JSON: ${err.message}`);
            return;
        }
        if (!doc || typeof doc !== 'object' || Array.isArray(doc)) {
            showJsonError('The document must be a JSON object');
            return;
        }
        s.json.saving = true;
        render();
        const out = await app.act('Save defaults', () => api.replaceDefaults(doc), { success: (r) => `Saved ${plural(r.keys, 'default key')}` });
        await s.loading;
        s.json.saving = false;
        if (out.ok) s.json = { text: '', dirty: false, error: '', saving: false };
        else s.json.error = out.error.message;
        render();
    };

    const rowOf = (el) => s.defaults.rows.find((r) => r.key === el.dataset.key);

    ui.on('settings-default-add', () => openPicker());
    ui.on('settings-env-add', () => openPicker('env.'));
    ui.on('settings-default-edit', (el) => openDefault(rowOf(el)));

    ui.on('settings-default-info', (el) => {
        const key = el.dataset.key;
        if (!s.open.delete(key)) s.open.add(key);
        render();
    });

    ui.on('settings-json-save', () => saveJson());
    ui.on('settings-json-revert', () => {
        s.json = { text: '', dirty: false, error: '', saving: false };
        render();
    });

    ui.on('settings-adopt', (el) => {
        const { dir, key } = el.dataset;
        run(`key:${dir}:${key}`, 'Adopt key', () => api.adoptKey(dir, key), `${key} added to the defaults`);
    });

    ui.on('settings-remove-key', async (el) => {
        const { dir, key } = el.dataset;
        const ok = await ui.confirm({
            title: `Remove ${key}?`,
            icon: 'trash-2',
            danger: true,
            confirmLabel: 'Remove',
            message: `Deletes <span class="font-mono text-text">${esc(key)}</span> from <span class="font-mono text-text">${esc(dir)}/settings.json</span>.`,
        });
        if (ok) await run(`key:${dir}:${key}`, 'Remove key', () => api.removeKey(dir, key), `${key} removed from ${dir}`);
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
                    if (key === 'defaults.json') {
                        s.json.text = e.target.value;
                        if (!s.json.dirty) {
                            s.json.dirty = true;
                            s.el.querySelectorAll('[data-json-btn]').forEach((b) => { b.disabled = false; });
                        }
                        if (s.json.error) showJsonError('');
                        return;
                    }
                    if (key === 'defaults.query') {
                        s.dquery = e.target.value;
                        renderKeys();
                        return;
                    }
                    if (!key?.startsWith('track.')) return;
                    s.track[key.slice(6)] = e.target.value;
                    if (s.track.error) {
                        s.track.error = '';
                        s.el.querySelector('[data-form="track"] [role="alert"]')?.remove();
                    }
                }],
                ['change', (e) => {
                    if (e.target.dataset.dv !== undefined) saveInput(e.target);
                }],
                ['keydown', (e) => {
                    if (e.key === 'Enter' && e.target.dataset.dv !== undefined) e.target.blur();
                }],
                ['submit', (e) => {
                    if (e.target.dataset.form !== 'track') return;
                    e.preventDefault();
                    track();
                }],
                ['ui-change', (e) => {
                    if (e.detail.name === 'settings-section') show(e.detail.value);
                    if (e.detail.name === 'defaults-mode') {
                        s.dmode = e.detail.value;
                        render();
                    }
                    if (e.detail.name?.startsWith('dv:')) saveValue(e.detail.name.slice(3), e.detail.value);
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
