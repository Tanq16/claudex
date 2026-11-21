views.dashboard = {
    local: { tab: 'active', filter: '', busy: new Map(), dirs: null },
    root: null,
    shape: null,
    dirSeq: 0,

    mount(el) {
        el.innerHTML = `<section class="flex h-full flex-col gap-3 md:gap-4 3xl:gap-6">
            <header data-dash-header class="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-3"></header>
            <div data-dash-body class="min-h-0 flex-1"></div>
        </section>`;
        this.root = el.firstElementChild;
        this.shape = null;
        this.root.addEventListener('ui-change', (e) => {
            if (e.detail.name === 'dash-tab') this.setTab(e.detail.value);
        });
        this.update();
    },

    unmount() {
        this.root = null;
    },

    update() {
        this.paint();
        this.loadDirs();
    },

    async loadDirs() {
        const seq = ++this.dirSeq;
        let dirs;
        try {
            dirs = await api.directories();
        } catch (err) {
            ui.toast(`Untracked directories failed to load: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            return;
        }
        if (seq !== this.dirSeq || !this.root) return;
        this.local.dirs = dirs;
        this.paint();
    },

    paint() {
        if (!this.root || !store.state) return;
        const { projects } = store.state;
        this.renderHeader();
        const shape = projects.length || this.local.dirs?.length ? 'full' : 'empty';
        if (shape !== this.shape) {
            this.shape = shape;
            this.renderBody();
        }
        if (shape === 'full') {
            this.renderActive();
            this.renderTracked();
        }
        ui.icons(this.root);
    },

    accountLabel(dir) {
        return store.state.accounts.find((a) => a.dir === dir)?.label ?? dir;
    },

    multiAccount() {
        return store.state.accounts.length > 1;
    },

    renderHeader() {
        const { root, projects, accounts } = store.state;
        const active = projects.filter((p) => p.active).length;
        const inactive = projects.length - active;
        const header = this.root.querySelector('[data-dash-header]');
        header.innerHTML = `
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-mantle text-mauve"><i data-lucide="folder-tree" class="size-5"></i></span>
            <div class="min-w-0 flex-1">
                <h1 class="truncate font-mono text-sm text-text">${ui.esc(root)}</h1>
                <p class="truncate text-xs text-overlay1">${projects.length} tracked ${projects.length === 1 ? 'directory' : 'directories'}${this.multiAccount() ? ` across ${accounts.length} accounts` : ''}</p>
            </div>
            ${projects.length ? `<span class="hidden items-center gap-2 md:flex">
                ${ui.chip(`${active} active`, { tone: active ? 'green' : 'neutral', icon: 'activity' })}
                ${ui.chip(`${inactive} inactive`, { icon: 'moon' })}
            </span>
            <span class="w-full md:hidden">${ui.segmented({
                name: 'dash-tab',
                value: this.local.tab,
                full: true,
                options: [
                    { value: 'active', label: `Active ${active}`, icon: 'activity' },
                    { value: 'tracked', label: `Inactive ${inactive}`, icon: 'folders' },
                ],
            })}</span>` : ''}`;
    },

    renderBody() {
        const body = this.root.querySelector('[data-dash-body]');
        if (this.shape === 'empty') {
            body.className = 'min-h-0 flex-1 rounded-3xl bg-mantle';
            body.innerHTML = ui.empty({
                icon: 'folder-plus',
                title: 'Nothing tracked yet',
                hint: `Start a session in a directory under <span class="font-mono text-subtext0">${ui.esc(store.state.root)}</span>. Claudex tracks every directory it bootstraps, applies presets to, or launches in.`,
                action: ui.button('New session', { icon: 'plus', variant: 'primary', action: 'new-session' }),
            });
            return;
        }
        body.className = 'grid min-h-0 flex-1 grid-cols-1 grid-rows-[minmax(0,1fr)] gap-4 md:grid-cols-[minmax(0,1fr)_20rem] lg:grid-cols-[minmax(0,1fr)_24rem] 3xl:grid-cols-[minmax(0,1fr)_40rem] 3xl:gap-6';
        body.innerHTML = `
            <section data-region="active" class="flex min-h-0 flex-col">
                <header class="flex items-center gap-3 pb-3 pl-1">
                    <h2 class="flex-1 font-display text-lg font-medium text-text">Active sessions</h2>
                    ${ui.button('New session', { icon: 'plus', variant: 'primary', size: 'sm', action: 'new-session' })}
                </header>
                <div data-scroll="lane" class="-mr-3.5 min-h-0 flex-1 overflow-y-auto rounded-3xl">
                    <ul data-list="active" class="grid grid-cols-[repeat(auto-fill,minmax(min(100%,20rem),1fr))] gap-3 3xl:grid-cols-[repeat(auto-fill,minmax(26rem,1fr))] 3xl:gap-4"></ul>
                </div>
            </section>
            <section data-region="tracked" class="flex min-h-0 flex-col rounded-3xl bg-mantle">
                <header class="flex shrink-0 flex-col gap-3 p-3 md:p-4 md:pb-3">
                    <h2 class="px-1 font-display text-lg font-medium text-text max-md:hidden">Inactive</h2>
                    <label class="relative block">
                        <i data-lucide="search" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i>
                        <input data-dash-filter type="text" autocomplete="off" spellcheck="false" placeholder="Filter" class="${ui.cls.input} pr-12 pl-11" value="${ui.esc(this.local.filter)}">
                        <span data-dash-clear class="absolute top-1/2 right-1 -translate-y-1/2 ${this.local.filter ? '' : 'hidden'}">${ui.iconButton('x', { title: 'Clear filter', action: 'dash-clear-filter', size: 'sm' })}</span>
                    </label>
                </header>
                <div data-scroll="lane" class="min-h-0 flex-1 overflow-y-auto px-2 pb-2 md:px-3 md:pb-3">
                    <ul data-list="tracked" class="flex flex-col gap-1"></ul>
                </div>
            </section>`;
        body.querySelector('[data-dash-filter]').addEventListener('input', (e) => {
            this.local.filter = e.target.value;
            body.querySelector('[data-dash-clear]').classList.toggle('hidden', !this.local.filter);
            this.renderTracked();
            ui.icons(body.querySelector('[data-list="tracked"]'));
        });
        this.applyTab();
    },

    setTab(tab) {
        const regions = this.root?.querySelectorAll('[data-region] > [data-scroll]') ?? [];
        regions.forEach((r) => { if (r.offsetParent) r.closest('[data-region]').dataset.top = r.scrollTop; });
        this.local.tab = tab;
        this.applyTab();
        regions.forEach((r) => { if (r.offsetParent) r.scrollTop = Number(r.closest('[data-region]').dataset.top || 0); });
    },

    applyTab() {
        this.root.querySelectorAll('[data-region]').forEach((r) => {
            r.classList.toggle('max-md:hidden', r.dataset.region !== this.local.tab);
        });
    },

    renderActive() {
        const list = this.root.querySelector('[data-list="active"]');
        const active = store.state.projects.filter((p) => p.active).sort((a, b) => b.startedAt - a.startedAt);
        list.classList.toggle('h-full', !active.length);
        if (!active.length) {
            const tracked = store.state.projects.length;
            list.innerHTML = `<li class="col-span-full h-full rounded-3xl bg-mantle">${ui.empty({
                icon: 'moon',
                title: 'No active sessions',
                hint: `${tracked} tracked ${tracked === 1 ? 'directory is' : 'directories are'} idle. Start a session from Inactive, or start something new.`,
                action: ui.button('New session', { icon: 'plus', action: 'new-session' }),
            })}</li>`;
            return;
        }
        list.innerHTML = active.map((p) => this.card(p)).join('');
    },

    card(p) {
        const busy = this.local.busy.get(p.name);
        const name = ui.esc(p.name);
        const meta = [
            { icon: 'clock', label: 'Started', value: ui.esc(ui.timeAgo(p.startedAt)) },
            this.multiAccount() && { icon: 'user-round', label: 'Account', value: ui.esc(this.accountLabel(p.account)) },
            { icon: 'plug', label: 'MCP servers', value: p.mcp.length ? `${p.mcp.length} <span class="text-overlay1">${ui.esc(p.mcp.join(', '))}</span>` : '<span class="text-overlay1">None</span>' },
        ].filter(Boolean);
        return `<li>
            <article data-action="dash-open" data-name="${name}" class="flex h-full cursor-pointer flex-col gap-3 rounded-3xl bg-mantle p-4 transition-colors hover:bg-base ${busy ? 'opacity-70' : ''}">
                <header class="flex items-center gap-3">
                    <span class="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-mauve/15 text-mauve"><i data-lucide="folder-git-2" class="size-5"></i></span>
                    <div class="min-w-0 flex-1">
                        <h3 class="font-display leading-snug font-medium wrap-anywhere text-text"><a href="#/project/${encodeURIComponent(p.name)}" class="outline-none focus-visible:underline">${name}</a></h3>
                        <p class="truncate font-mono text-xs text-overlay1" title="${ui.esc(p.dir)}">${ui.esc(p.dir)}</p>
                    </div>
                </header>
                <dl class="grid grid-cols-2 gap-x-4 gap-y-3">
                    ${meta.map((m) => `<div class="min-w-0">
                        <dt class="flex items-center gap-1.5 text-xs text-overlay1"><i data-lucide="${m.icon}" class="size-3.5"></i>${m.label}</dt>
                        <dd class="mt-0.5 truncate text-sm text-subtext1">${m.value}</dd>
                    </div>`).join('')}
                    <div class="col-span-2 min-w-0">
                        <dt class="flex items-center gap-1.5 text-xs text-overlay1"><i data-lucide="layers" class="size-3.5"></i>Presets</dt>
                        <dd class="mt-1 flex flex-wrap gap-1.5">${p.presets.map((x) => ui.chip(x)).join('')}</dd>
                    </div>
                </dl>
                <footer class="mt-auto flex items-center gap-2">
                    ${ui.button(busy === 'restart' ? 'Restarting' : 'Restart', { icon: busy === 'restart' ? 'loader-circle' : 'rotate-cw', spin: busy === 'restart', size: 'sm', action: 'dash-restart', disabled: !!busy, attrs: `data-name="${name}"` })}
                    ${ui.button(busy === 'stop' ? 'Stopping' : 'Stop', { icon: busy === 'stop' ? 'loader-circle' : 'square', spin: busy === 'stop', size: 'sm', variant: 'danger', action: 'dash-stop', disabled: !!busy, attrs: `data-name="${name}"` })}
                </footer>
            </article>
        </li>`;
    },

    renderTracked() {
        const list = this.root.querySelector('[data-list="tracked"]');
        const inactive = store.state.projects.filter((p) => !p.active);
        const untracked = this.local.dirs || [];
        const q = this.local.filter.trim();
        const match = (items, text, recent) => (q
            ? items.map((x) => ({ x, s: ui.fuzzy(q, text(x)) })).filter((m) => m.s >= 0).sort((a, b) => b.s - a.s || recent(b.x) - recent(a.x)).map((m) => m.x)
            : [...items].sort((a, b) => recent(b) - recent(a)));
        const rows = match(inactive, (p) => `${p.name} ${p.dir}`, (p) => p.lastActive);
        const dirs = match(untracked, (d) => d.name, () => 0);
        list.classList.toggle('h-full', !rows.length && !dirs.length);
        if (!inactive.length && !untracked.length) {
            list.innerHTML = `<li class="h-full">${ui.empty({ icon: 'check-check', title: 'Everything is running', hint: 'Every tracked directory has an active session.' })}</li>`;
            return;
        }
        if (!rows.length && !dirs.length) {
            list.innerHTML = `<li class="h-full">${ui.empty({
                icon: 'search-x',
                title: 'No matches',
                hint: `No directory matches <span class="font-mono text-subtext0">${ui.esc(q)}</span>.`,
                action: ui.button('Clear filter', { icon: 'x', size: 'sm', variant: 'ghost', action: 'dash-clear-filter' }),
            })}</li>`;
            return;
        }
        const group = (label, n) => `<li class="${ui.cls.group} flex items-center gap-2 first:pt-1">${label}${ui.chip(String(n))}</li>`;
        list.innerHTML = [
            rows.length && dirs.length ? group('Tracked', rows.length) : '',
            ...rows.map((p) => this.row(p)),
            dirs.length ? group('Untracked', dirs.length) : '',
            ...dirs.map((d) => this.untrackedRow(d)),
        ].join('');
    },

    row(p) {
        const name = ui.esc(p.name);
        return `<li class="flex items-center gap-3 rounded-2xl py-2.5 pr-2 pl-3 transition-colors hover:bg-base">
            <span class="min-w-0 flex-1">
                <span class="flex items-baseline gap-2">
                    <span class="min-w-0 text-sm font-medium wrap-anywhere text-text">${name}</span>
                    <span class="shrink-0 text-xs text-overlay1" title="Last active">${ui.esc(ui.timeAgo(p.lastActive))}</span>
                </span>
                <span class="block truncate font-mono text-xs text-overlay1" title="${ui.esc(p.dir)}">${ui.esc(p.dir)}</span>
                ${this.multiAccount() ? `<span class="mt-0.5 flex items-center gap-1.5 text-xs text-overlay1"><i data-lucide="user-round" class="size-3 shrink-0"></i><span class="truncate">${ui.esc(this.accountLabel(p.account))}</span></span>` : ''}
            </span>
            <span class="hidden max-w-[45%] flex-wrap justify-end gap-1.5 3xl:flex">${p.presets.map((x) => ui.chip(x)).join('')}</span>
            ${ui.iconButton('plus', { title: 'New session', size: 'sm', variant: 'secondary', action: 'dash-new', attrs: `data-name="${name}"` })}
            ${ui.iconButton('ellipsis-vertical', { title: 'More', size: 'sm', action: 'dash-tracked-menu', attrs: `data-name="${name}"` })}
        </li>`;
    },

    untrackedRow(d) {
        const name = ui.esc(d.name);
        const path = `${store.state.root}/${d.name}`;
        const busy = this.local.busy.get(d.name);
        return `<li class="flex items-center gap-3 rounded-2xl py-2.5 pr-2 pl-3 transition-colors hover:bg-base">
            <i data-lucide="folder" class="size-4 shrink-0 text-overlay0"></i>
            <span class="min-w-0 flex-1">
                <span class="block text-sm wrap-anywhere text-subtext0">${name}</span>
                <span class="block truncate font-mono text-xs text-overlay0" title="${ui.esc(path)}">${ui.esc(path)}</span>
                ${d.foreign.length ? `<span class="mt-0.5 flex items-center gap-1.5 text-xs text-peach"><i data-lucide="file-warning" class="size-3 shrink-0"></i><span class="truncate">${d.foreign.length} ${d.foreign.length === 1 ? 'file' : 'files'} claudex did not create</span></span>` : ''}
            </span>
            ${ui.iconButton(busy === 'track' ? 'loader-circle' : 'folder-plus', { title: 'Start tracking', size: 'sm', variant: 'secondary', action: 'dash-track', spin: busy === 'track', disabled: !!busy, attrs: `data-name="${name}"` })}
            ${ui.iconButton('plus', { title: 'New session', size: 'sm', variant: 'secondary', action: 'dash-new-untracked', disabled: !!busy, attrs: `data-name="${name}"` })}
        </li>`;
    },

    askForeign(d) {
        return new Promise((resolve) => {
            let answered = false;
            const m = ui.modal.open({
                title: `Start tracking ${d.name}?`,
                icon: 'file-warning',
                size: 'sm',
                body: `<div class="flex flex-col gap-4 pt-1">
                    <p class="text-sm leading-relaxed text-subtext0">It holds files claudex did not create. Claudex moves them aside before tracking it.</p>
                    <p class="flex flex-wrap gap-1.5">${d.foreign.map((f) => ui.chip(f, { tone: 'peach', mono: true })).join('')}</p>
                    ${ui.choices({
                        name: 'dash-foreign',
                        value: 'preserve',
                        options: [
                            { value: 'preserve', label: 'Preserve', icon: 'archive', hint: 'Zip them into .agents/old-ai-files.zip' },
                            { value: 'clean', label: 'Clean', icon: 'eraser', hint: 'Delete them without a backup' },
                        ],
                    })}
                </div>`,
                footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-confirm="no"' })}${ui.button('Start tracking', { icon: 'folder-plus', variant: 'primary', attrs: 'data-confirm="yes"' })}`,
                onClose: () => { if (!answered) resolve(null); },
            });
            m.el.addEventListener('click', (e) => {
                const b = e.target.closest('[data-confirm]');
                if (!b) return;
                answered = true;
                const choice = ui.value(m.el, 'dash-foreign');
                m.close();
                resolve(b.dataset.confirm === 'yes' ? choice : null);
            });
        });
    },

    async run(name, kind, label, fn, success) {
        this.local.busy.set(name, kind);
        this.paint();
        await app.act(label, fn, { success });
        this.local.busy.delete(name);
        this.paint();
    },
};

ui.on('dash-open', (el) => app.go(`#/project/${encodeURIComponent(el.dataset.name)}`));

ui.on('dash-restart', (el) => {
    const { name } = el.dataset;
    views.dashboard.run(name, 'restart', 'Restart', () => api.restart(name), `${name} restarted with --continue`);
});

ui.on('dash-stop', async (el) => {
    const { name } = el.dataset;
    const ok = await ui.confirm({
        title: `Stop ${name}?`,
        icon: 'square',
        message: `This closes the tmux window <span class="font-mono text-text">claudex:${ui.esc(name)}</span> and ends its Remote Control session. The conversation stays on disk, so Continue picks it up.`,
        confirmLabel: 'Stop session',
        danger: true,
    });
    if (ok) views.dashboard.run(name, 'stop', 'Stop', () => api.stop(name), `${name} stopped`);
});

ui.on('dash-new', (el) => views.newSession.open({ name: el.dataset.name, mode: 'new' }));

ui.on('dash-new-untracked', (el) => views.newSession.open({ name: el.dataset.name }));

ui.on('dash-track', async (el) => {
    const { name } = el.dataset;
    const dir = views.dashboard.local.dirs?.find((d) => d.name === name);
    let choice;
    if (dir?.foreign.length) {
        choice = await views.dashboard.askForeign(dir);
        if (!choice) return;
    }
    views.dashboard.run(name, 'track', 'Start tracking', () => api.track(name, choice), `${name} is tracked`);
});

ui.on('dash-tracked-menu', (el) => {
    const { name } = el.dataset;
    ui.menu(el, [
        { icon: 'folder-open', label: 'Browse files', onSelect: () => app.go(`#/project/${encodeURIComponent(name)}`) },
        '-',
        {
            icon: 'folder-minus',
            label: 'Untrack',
            hint: 'Files stay on disk',
            danger: true,
            onSelect: async () => {
                const ok = await ui.confirm({
                    title: `Untrack ${name}?`,
                    icon: 'folder-minus',
                    message: 'Claudex forgets this directory and removes it from the dashboard. Nothing on disk changes, and starting a session there tracks it again.',
                    confirmLabel: 'Untrack',
                    danger: true,
                });
                if (ok) app.act('Untrack', () => api.untrack(name), { success: `${name} untracked` });
            },
        },
    ]);
});

ui.on('dash-clear-filter', () => {
    const input = views.dashboard.root?.querySelector('[data-dash-filter]');
    if (!input) return;
    input.value = '';
    input.dispatchEvent(new Event('input'));
    input.focus();
});
