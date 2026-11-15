views.newSession = (() => {
    const slug = /^[a-z0-9][a-z0-9._-]*$/;
    const quote = (v) => (/^[\w@%+=:,./~-]+$/.test(v) ? v : `'${v.replace(/'/g, `'\\''`)}'`);

    const well = ({ icon, title, aside = '', body, cls = '' }) => `
        <section class="flex flex-col gap-4 rounded-3xl bg-base p-4 md:p-5 ${cls}">
            <header class="flex items-center gap-2.5">
                <i data-lucide="${icon}" class="size-4 text-overlay1"></i>
                <h3 class="flex-1 font-display text-sm font-medium text-text">${title}</h3>
                ${aside}
            </header>
            ${body}
        </section>`;

    const note = (tone, icon, text) => {
        const tones = { muted: 'text-overlay1', red: 'text-red', green: 'text-green', peach: 'text-peach' };
        return `<p class="flex items-start gap-2 px-1 text-xs leading-relaxed ${tones[tone]}"><i data-lucide="${icon}" class="mt-px size-3.5 shrink-0"></i><span class="min-w-0">${text}</span></p>`;
    };

    const spinner = (text) => `<p class="flex items-center gap-2 px-1 text-xs text-overlay1"><i data-lucide="loader-circle" class="size-3.5 animate-spin"></i>${ui.esc(text)}</p>`;

    function open(prefill = {}) {
        const s = {
            source: 'existing',
            dir: prefill.name || '',
            emptyName: '',
            repo: '',
            cloneName: '',
            cloneTouched: false,
            repoQuery: '',
            dirs: null,
            dirsError: '',
            repos: null,
            reposError: '',
            foreignChoice: 'preserve',
            rcName: '',
            rcTouched: false,
            account: '',
            mode: prefill.mode === 'continue' ? 'continue' : 'new',
            presets: new Set(),
            extraPresets: [],
            mcp: new Set(),
            connectors: true,
            prefilledFor: undefined,
            presetDraft: '',
            presetPending: false,
            presetError: '',
            launching: false,
            closed: false,
        };

        const state = () => store.state;
        const root = () => state().root;
        const project = () => (s.source === 'existing' ? state().projects.find((p) => p.name === s.dir) : undefined);
        const untracked = () => (s.source === 'existing' && !project() ? (s.dirs || []).find((d) => d.name === s.dir) : undefined);
        const target = () => ({ existing: s.dir, empty: s.emptyName.trim(), clone: s.cloneName.trim() })[s.source];
        const taken = (name) => state().projects.some((p) => p.name === name) || (s.dirs || []).some((d) => d.name === name);
        const canContinue = () => Boolean(project() || untracked());
        const mode = () => (canContinue() ? s.mode : 'new');
        const multiAccount = () => state().accounts.length > 1;
        const defaultAccount = () => (state().accounts.find((a) => a.isDefault) || state().accounts[0])?.dir || '';
        const library = () => [
            ...state().presets.builtin.map((p) => ({ name: p.name, hint: p.description })),
            ...state().presets.remote.map((p) => ({ name: p.name, hint: p.description })),
            ...s.extraPresets,
        ];

        const nameCheck = (name, verb) => {
            if (!name) return { ok: false, html: note('muted', 'info', 'Lowercase letters, digits, dots, dashes, and underscores.') };
            if (!slug.test(name)) return { ok: false, html: note('red', 'circle-alert', 'Use lowercase letters, digits, dots, dashes, and underscores, starting with a letter or digit.') };
            if (taken(name)) return { ok: false, html: note('red', 'circle-alert', `<span class="font-mono">${ui.esc(`${root()}/${name}`)}</span> already exists. Pick it under Existing instead.`) };
            return { ok: true, html: note('green', 'circle-check', `${verb} <span class="font-mono">${ui.esc(`${root()}/${name}`)}</span>`) };
        };

        const validity = () => {
            if (s.source === 'existing') {
                if (!s.dir) return 'Pick a directory';
                if (!project() && !untracked()) return 'Pick a directory';
                if (project()?.active) return 'A session already runs in this directory';
            }
            if (s.source === 'empty' && !nameCheck(target()).ok) return 'Enter a valid directory name';
            if (s.source === 'clone') {
                if (!s.repo) return 'Pick a repository';
                if (!nameCheck(target()).ok) return 'Enter a valid directory name';
            }
            if (!s.rcName.trim()) return 'Enter a Remote Control name';
            return '';
        };

        const syncRc = () => {
            if (s.rcTouched) return;
            s.rcName = project()?.rcName || target() || '';
            const input = el.querySelector('[data-name="ns-rc"]');
            if (input) input.value = s.rcName;
        };

        const applyDefaults = () => {
            const p = project();
            const key = p?.name ?? null;
            if (s.prefilledFor === key) return;
            s.prefilledFor = key;
            const known = new Set(library().map((x) => x.name));
            for (const name of p?.presets || []) {
                if (!known.has(name)) s.extraPresets.push({ name, hint: 'Applied to this project' });
            }
            s.presets = new Set(p ? p.presets : state().presets.builtin.filter((b) => b.defaultOn).map((b) => b.name));
            s.mcp = new Set(p ? p.mcp : []);
            s.connectors = p ? p.connectors : true;
            s.account = p?.account || defaultAccount();
        };

        const directoryOptions = () => [
            ...state().projects.map((p) => ({
                value: p.name,
                label: p.name,
                icon: 'folder-git-2',
                hint: [p.active ? 'Active' : 'Tracked', p.rcName !== p.name ? `RC ${p.rcName}` : '', multiAccount() ? state().accounts.find((a) => a.dir === p.account)?.label : ''].filter(Boolean).join(' · '),
            })),
            ...(s.dirs || []).map((d) => ({
                value: d.name,
                label: d.name,
                icon: 'folder',
                hint: d.foreign.length ? `Untracked · ${d.foreign.length} files claudex did not create` : 'Untracked',
            })),
        ];

        const selectHtml = () => `
            ${ui.select({ name: 'ns-directory', value: s.dir, options: directoryOptions(), placeholder: 'Pick a directory', searchable: true, icon: 'folder-search' })}
            ${s.dirsError
                ? `<p class="flex items-center gap-2 px-1 text-xs text-red"><i data-lucide="circle-alert" class="size-3.5 shrink-0"></i><span class="flex-1">Untracked directories failed to load: ${ui.esc(s.dirsError)}</span>${ui.button('Retry', { size: 'sm', variant: 'ghost', icon: 'rotate-ccw', attrs: 'data-ns="retry-dirs"' })}</p>`
                : s.dirs ? '' : spinner('Looking for untracked directories')}`;

        const dirInfoHtml = () => {
            const p = project();
            const u = untracked();
            if (!s.dir || (!p && !u)) return note('muted', 'info', 'Tracked projects and untracked directories under the root are listed. Type to filter.');
            const path = `<p class="truncate px-1 font-mono text-xs text-overlay1">${ui.esc(`${root()}/${s.dir}`)}</p>`;
            if (p) {
                const account = state().accounts.find((a) => a.dir === p.account);
                const chips = [
                    ui.status(p.active),
                    ui.chip('Tracked', { tone: 'mauve', icon: 'folder-git-2' }),
                    multiAccount() && account ? ui.chip(account.label, { icon: 'user-round' }) : '',
                    ui.chip(`Last active ${ui.timeAgo(p.lastActive)}`, { icon: 'clock' }),
                ].join('');
                const blocked = p.active
                    ? `<p class="flex flex-wrap items-center gap-3 rounded-2xl bg-red/10 p-3 text-sm text-red"><i data-lucide="circle-alert" class="size-4 shrink-0"></i><span class="min-w-0 flex-1">A session already runs here. One directory runs one session at a time.</span>${ui.button('Open project', { size: 'sm', variant: 'danger', icon: 'arrow-up-right', attrs: 'data-ns="open-project"' })}</p>`
                    : note('muted', 'info', 'Claudex manages this directory, so presets re-apply without asking.');
                return `${path}<p class="flex flex-wrap gap-1.5">${chips}</p>${blocked}`;
            }
            if (!u.foreign.length) return `${path}<p class="flex flex-wrap gap-1.5">${ui.chip('Untracked', { icon: 'folder' })}</p>${note('muted', 'info', 'Claudex bootstraps it, then applies the presets you pick.')}`;
            return `${path}
                <p class="flex flex-col gap-3 rounded-2xl bg-peach/10 p-4">
                    <span class="flex items-start gap-2.5 text-sm text-peach"><i data-lucide="file-warning" class="mt-0.5 size-4 shrink-0"></i><span>Found files claudex did not create. Both choices end in a fresh apply.</span></span>
                    <span class="flex flex-wrap gap-1.5">${u.foreign.map((f) => ui.chip(f, { tone: 'peach', mono: true })).join('')}</span>
                </p>
                ${ui.choices({
                    name: 'ns-foreign',
                    value: s.foreignChoice,
                    columns: 2,
                    options: [
                        { value: 'preserve', label: 'Preserve', icon: 'archive', hint: 'Zip them into .agents/old-ai-files.zip' },
                        { value: 'clean', label: 'Clean', icon: 'eraser', hint: 'Run clean-cwd and drop them' },
                    ],
                })}`;
        };

        const reposHtml = () => {
            if (s.reposError) return `<li class="flex flex-col items-center gap-3 px-3 py-10 text-center text-sm text-red"><i data-lucide="circle-alert" class="size-5"></i><span>${ui.esc(s.reposError)}</span>${ui.button('Retry', { size: 'sm', icon: 'rotate-ccw', attrs: 'data-ns="retry-repos"' })}</li>`;
            if (!s.repos) {
                return Array.from({ length: 6 }, (_, i) => `<li class="flex items-center gap-3 rounded-2xl px-3 py-2.5"><span class="size-4 shrink-0 animate-pulse rounded-full bg-surface0"></span><span class="flex min-w-0 flex-1 flex-col gap-1.5"><span class="h-3 animate-pulse rounded-full bg-surface0" style="width:${40 + ((i * 17) % 35)}%"></span><span class="h-2.5 w-1/2 animate-pulse rounded-full bg-surface0/60"></span></span></li>`).join('');
            }
            const q = s.repoQuery.trim();
            const items = q ? s.repos.map((r) => ({ r, score: ui.fuzzy(q, r.full) })).filter((x) => x.score >= 0).sort((a, b) => b.score - a.score).map((x) => x.r) : s.repos;
            if (!items.length) return `<li class="px-3 py-10 text-center text-sm text-overlay1">No repository matches “${ui.esc(q)}”</li>`;
            return items.map((r) => {
                const [owner, name] = r.full.split('/');
                const on = r.full === s.repo;
                return `<li><button type="button" role="option" aria-selected="${on}" data-ns="repo" data-repo="${ui.esc(r.full)}" class="group flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0 aria-selected:bg-mauve/12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
                    <i data-lucide="${r.private ? 'lock' : 'book-marked'}" class="size-4 shrink-0 ${r.private ? 'text-yellow' : 'text-overlay1'} group-aria-selected:text-mauve"></i>
                    <span class="min-w-0 flex-1">
                        <span class="block truncate text-sm"><span class="text-overlay1">${ui.esc(owner)}/</span><span class="text-text">${ui.esc(name)}</span></span>
                        <span class="block truncate text-xs text-overlay1">${ui.esc(r.description)}</span>
                    </span>
                    <span class="hidden shrink-0 text-xs text-overlay0 sm:inline">${ui.timeAgo(r.updatedAt)}</span>
                    <i data-lucide="check" class="size-4 shrink-0 text-mauve ${on ? '' : 'invisible'}"></i>
                </button></li>`;
            }).join('');
        };

        const nameField = (field, value, placeholder, verb) => `
            <label for="ns-${field}" class="${ui.cls.label} -mb-2 px-1">Directory name</label>
            <input id="ns-${field}" data-name="ns-${field}" class="${ui.cls.input} font-mono" value="${ui.esc(value)}" placeholder="${placeholder}" autocomplete="off" autocapitalize="off" spellcheck="false">
            <div data-region="${field}-check">${nameCheck(value.trim(), verb).html}</div>`;

        const sourceBodyHtml = () => {
            if (s.source === 'existing') {
                return `<div data-region="select" class="flex flex-col gap-2">${selectHtml()}</div>
                    <div data-region="dir-info" class="flex flex-col gap-3">${dirInfoHtml()}</div>`;
            }
            if (s.source === 'empty') return nameField('empty-name', s.emptyName, 'my-new-project', 'Creates');
            return `
                <p class="flex items-center gap-2.5 px-1 text-sm text-subtext0">
                    <i class="fab fa-github text-[16px] text-overlay1"></i>
                    <span class="flex-1">Your GitHub repositories</span>
                    ${s.repos ? ui.chip(`${s.repos.length} repos`) : ''}
                </p>
                <label class="relative block">
                    <i data-lucide="search" class="pointer-events-none absolute left-4 top-1/2 size-4 -translate-y-1/2 text-overlay1"></i>
                    <input data-name="ns-repo-query" class="${ui.cls.input} pl-10" value="${ui.esc(s.repoQuery)}" placeholder="Filter by owner or name" autocomplete="off" spellcheck="false" ${s.repos ? '' : 'disabled'}>
                </label>
                <div data-scroll class="-mx-1 h-72 overflow-y-auto lg:h-80 3xl:h-[26rem]"><ul role="listbox" data-region="repos" class="flex flex-col gap-0.5 px-1">${reposHtml()}</ul></div>
                ${s.repo ? nameField('clone-name', s.cloneName, 'directory-name', 'Clones into') : note('muted', 'info', 'Pick a repository. The directory name defaults to its name.')}`;
        };

        const sourceHtml = () => well({
            icon: 'folder-tree',
            title: 'Directory',
            aside: `<span class="font-mono text-xs text-overlay1">${ui.esc(root())}</span>`,
            body: `
                ${ui.segmented({
                    name: 'ns-source',
                    value: s.source,
                    full: true,
                    options: [
                        { value: 'existing', label: 'Existing', icon: 'folder-open' },
                        { value: 'empty', label: 'Empty', icon: 'folder-plus' },
                        { value: 'clone', label: 'Clone', icon: 'folder-down' },
                    ],
                })}
                <div data-region="source-body" class="flex flex-col gap-3">${sourceBodyHtml()}</div>`,
        });

        const modeHintHtml = () => {
            if (!canContinue()) return note('muted', 'info', 'Continue needs an existing directory, so this starts a new conversation.');
            if (mode() === 'continue') return note('muted', 'history', 'Picks up the newest conversation in this directory, or starts a new one if there is none.');
            return note('muted', 'sparkles', 'Starts a fresh conversation. Earlier ones stay available to Continue.');
        };

        const sessionHtml = () => {
            const accountField = multiAccount()
                ? `<p class="${ui.cls.label} -mb-2 px-1">Account</p>
                   ${ui.select({ name: 'ns-account', value: s.account, icon: 'user-round', options: state().accounts.map((a) => ({ value: a.dir, label: a.label, hint: a.isDefault ? `${a.dir} · default` : a.dir, icon: 'user-round' })) })}`
                : '';
            return `
                <label for="ns-rc" class="${ui.cls.label} -mb-2 px-1">Remote Control name</label>
                <label class="relative block">
                    <i data-lucide="radio-tower" class="pointer-events-none absolute left-4 top-1/2 size-4 -translate-y-1/2 text-overlay1"></i>
                    <input id="ns-rc" data-name="ns-rc" class="${ui.cls.input} pl-10" value="${ui.esc(s.rcName)}" placeholder="session-name" autocomplete="off" spellcheck="false">
                </label>
                ${accountField}
                <p class="${ui.cls.label} -mb-2 px-1">Conversation</p>
                ${ui.segmented({
                    name: 'ns-mode',
                    value: mode(),
                    full: true,
                    options: [
                        { value: 'new', label: 'New', icon: 'message-square-plus' },
                        { value: 'continue', label: 'Continue', icon: 'history', disabled: !canContinue() },
                    ],
                })}
                <div data-region="mode-hint">${modeHintHtml()}</div>
                <span class="-mx-3 -mb-2 block">${ui.switch({ name: 'ns-connectors', label: 'claude.ai connectors', hint: 'Sets ENABLE_CLAUDEAI_MCP_SERVERS for this session', checked: s.connectors })}</span>`;
        };

        const presetListHtml = () => `<ul class="-mx-3 flex flex-col">${library().map((p) => `<li>${ui.checkbox({ name: `ns-preset:${p.name}`, label: p.name, hint: p.hint, checked: s.presets.has(p.name) })}</li>`).join('')}</ul>`;

        const presetButton = () => ui.button(s.presetPending ? 'Checking' : 'Add', { icon: s.presetPending ? 'loader-circle' : 'plus', spin: s.presetPending, type: 'submit', disabled: s.presetPending });

        const presetsHtml = () => `
            <div data-region="preset-list">${presetListHtml()}</div>
            <form data-ns="preset-form" class="flex gap-2">
                <input data-name="ns-preset-draft" class="${ui.cls.input} min-w-0 font-mono" value="${ui.esc(s.presetDraft)}" placeholder="owner-repo/preset" autocomplete="off" autocapitalize="off" spellcheck="false" aria-label="Add a preset by name">
                <span data-region="preset-button" class="contents">${presetButton()}</span>
            </form>
            <div data-region="preset-error">${s.presetError ? note('red', 'circle-alert', ui.esc(s.presetError)) : ''}</div>`;

        const mcpHtml = () => {
            const lib = state().mcpLibrary;
            if (!lib.length) return note('muted', 'info', 'The MCP library in Settings is empty.');
            return `<ul class="-mx-3 flex flex-col">${lib.map((m) => `<li>${ui.checkbox({ name: `ns-mcp:${m.name}`, label: m.name, hint: m.transport === 'stdio' ? 'stdio, runs locally' : m.transport, checked: s.mcp.has(m.name) })}</li>`).join('')}</ul>`;
        };

        const countChip = (n) => ui.chip(`${n} on`, { tone: n ? 'mauve' : 'neutral' });

        const command = () => {
            const parts = [];
            const account = s.account || defaultAccount();
            if (multiAccount() && account !== defaultAccount()) parts.push(['env', `CLAUDE_CONFIG_DIR=${account}`]);
            if (s.connectors) parts.push(['env', 'ENABLE_CLAUDEAI_MCP_SERVERS=true']);
            parts.push(['bin', 'claude']);
            if (mode() === 'continue') parts.push(['flag', '--continue']);
            parts.push(['flag', '--remote-control'], ['value', s.rcName.trim() ? quote(s.rcName.trim()) : '<name>']);
            return parts;
        };

        const commandHtml = () => {
            const tones = { env: 'text-overlay1', bin: 'text-mauve', flag: 'text-text', value: 'text-peach' };
            return `<span class="select-none text-overlay0">$ </span>${command().map(([k, v]) => `<span class="${tones[k]}">${ui.esc(v)}</span>`).join(' ')}`;
        };

        const footerHtml = () => `
            <p class="flex min-w-0 basis-full items-center gap-1 rounded-full bg-base py-1 pl-4 pr-1 md:mr-auto md:basis-0 md:flex-1">
                <code data-region="command" class="min-w-0 flex-1 overflow-x-auto whitespace-nowrap font-mono text-xs">${commandHtml()}</code>
                ${ui.iconButton('copy', { title: 'Copy command', size: 'sm', attrs: 'data-ns="copy"' })}
            </p>
            <span data-region="footer-actions" class="contents">${actionsHtml()}</span>`;

        const actionsHtml = () => {
            const reason = validity();
            return `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-ns="cancel"', disabled: s.launching })}
                ${ui.button(s.launching ? 'Launching' : mode() === 'continue' ? 'Continue' : 'Launch', {
                    icon: s.launching ? 'loader-circle' : 'rocket',
                    spin: s.launching,
                    variant: 'primary',
                    disabled: Boolean(reason) || s.launching,
                    attrs: `data-ns="launch" ${reason ? `title="${ui.esc(reason)}"` : ''}`,
                })}`;
        };

        applyDefaults();
        if (!s.rcTouched) s.rcName = project()?.rcName || target() || '';

        const m = ui.modal.open({
            title: 'New session',
            icon: 'square-terminal',
            subtitle: 'Runs in the <span class="font-mono">claudex</span> tmux session',
            size: 'xl',
            body: `
                <div class="flex flex-col gap-3 3xl:contents">
                    <div data-region="source" class="contents">${sourceHtml()}</div>
                    <div data-region="session" class="contents">${well({ icon: 'terminal', title: 'Session', body: sessionHtml() })}</div>
                </div>
                <div class="flex flex-col gap-3">
                    ${well({ icon: 'layers', title: 'Presets', aside: `<span data-region="preset-count">${countChip(s.presets.size)}</span>`, body: `<div data-region="presets" class="flex flex-col gap-3">${presetsHtml()}</div>` })}
                    ${well({ icon: 'plug', title: 'MCP servers', aside: `<span data-region="mcp-count">${countChip(s.mcp.size)}</span>`, body: `<div data-region="mcp">${mcpHtml()}</div>` })}
                </div>`,
            footer: footerHtml(),
            onClose: () => {
                s.closed = true;
                document.removeEventListener('store-change', onStore);
            },
        });
        const el = m.el;
        const body = el.querySelector('[data-modal-body]');
        body.classList.add('grid', 'grid-cols-1', 'content-start', 'gap-3', 'lg:grid-cols-2', 'lg:items-start', '3xl:grid-cols-3');

        const paint = (region, html) => {
            const r = el.querySelector(`[data-region="${region}"]`);
            if (!r) return;
            r.innerHTML = html;
            ui.icons(r);
        };

        const paintFooter = () => {
            paint('command', commandHtml());
            paint('footer-actions', actionsHtml());
        };

        const paintCounts = () => {
            paint('preset-count', countChip(s.presets.size));
            paint('mcp-count', countChip(s.mcp.size));
        };

        const paintTargetDependents = () => {
            applyDefaults();
            syncRc();
            paint('session', well({ icon: 'terminal', title: 'Session', body: sessionHtml() }));
            paint('presets', presetsHtml());
            paint('mcp', mcpHtml());
            paintCounts();
        };

        const paintAll = () => {
            paint('source', sourceHtml());
            paintTargetDependents();
            paintFooter();
        };

        const onStore = () => {
            if (!state().accounts.some((a) => a.dir === s.account)) s.account = defaultAccount();
            paintAll();
        };
        document.addEventListener('store-change', onStore);

        const loadDirs = async () => {
            s.dirsError = '';
            if (s.source === 'existing') paint('select', selectHtml());
            try {
                s.dirs = await api.directories();
            } catch (err) {
                s.dirsError = err.message;
            }
            if (s.source === 'existing') {
                paint('select', selectHtml());
                paint('dir-info', dirInfoHtml());
            }
            if (s.source === 'empty') paint('empty-name-check', nameCheck(target(), 'Creates').html);
            if (s.source === 'clone' && s.repo) paint('clone-name-check', nameCheck(target(), 'Clones into').html);
            paintTargetDependents();
            paintFooter();
        };

        const loadRepos = async () => {
            s.reposError = '';
            s.repos = null;
            if (s.source === 'clone') paint('source-body', sourceBodyHtml());
            try {
                s.repos = await api.repos();
            } catch (err) {
                s.reposError = err.message;
            }
            if (s.source === 'clone') {
                paint('source-body', sourceBodyHtml());
                el.querySelector('[data-name="ns-repo-query"]')?.focus();
            }
        };

        const addPreset = async () => {
            const name = s.presetDraft.trim();
            if (!name || s.presetPending) return;
            const input = el.querySelector('[data-name="ns-preset-draft"]');
            if (library().some((p) => p.name === name)) {
                s.presets.add(name);
                s.presetDraft = '';
                input.value = '';
                paint('preset-list', presetListHtml());
                paintCounts();
                return;
            }
            s.presetPending = true;
            s.presetError = '';
            input.readOnly = true;
            paint('preset-button', presetButton());
            paint('preset-error', '');
            try {
                const r = await api.validatePreset(name);
                if (r.ok) {
                    s.extraPresets.push({ name: r.name, hint: r.note || 'Validated' });
                    s.presets.add(r.name);
                    s.presetDraft = '';
                    input.value = '';
                } else {
                    s.presetError = `${r.name}: ${r.error}`;
                }
            } catch (err) {
                s.presetError = err.message;
            }
            s.presetPending = false;
            input.readOnly = false;
            paint('preset-list', presetListHtml());
            paint('preset-button', presetButton());
            paint('preset-error', s.presetError ? note('red', 'circle-alert', ui.esc(s.presetError)) : '');
            paintCounts();
        };

        const launch = async () => {
            if (validity() || s.launching) return;
            const req = {
                name: target(),
                source: s.source,
                rcName: s.rcName.trim(),
                account: s.account || defaultAccount(),
                mode: mode(),
                presets: library().map((p) => p.name).filter((n) => s.presets.has(n)),
                mcp: [...s.mcp],
                connectors: s.connectors,
            };
            if (s.source === 'clone') req.repo = s.repo;
            if (untracked()?.foreign.length) req.foreignChoice = s.foreignChoice;
            s.launching = true;
            body.inert = true;
            body.classList.add('opacity-60');
            paintFooter();
            const r = await app.act('Launch', () => api.start(req), { success: `${req.rcName} is starting` });
            s.launching = false;
            if (r.ok) {
                const wasClosed = s.closed;
                m.close();
                if (!wasClosed) app.go(`#/project/${encodeURIComponent(r.result.name)}`);
                return;
            }
            body.inert = false;
            body.classList.remove('opacity-60');
            paintFooter();
        };

        el.addEventListener('ui-change', (e) => {
            const { name, value } = e.detail;
            if (name === 'ns-source') {
                s.source = value;
                paint('source-body', sourceBodyHtml());
                paintTargetDependents();
                if (value === 'clone' && !s.repos && !s.reposError) loadRepos();
            } else if (name === 'ns-directory') {
                s.dir = value;
                paint('dir-info', dirInfoHtml());
                paintTargetDependents();
            } else if (name === 'ns-foreign') {
                s.foreignChoice = value;
            } else if (name === 'ns-account') {
                s.account = value;
            } else if (name === 'ns-mode') {
                s.mode = value;
                paint('mode-hint', modeHintHtml());
            } else if (name === 'ns-connectors') {
                s.connectors = value;
            } else if (name.startsWith('ns-preset:')) {
                const key = name.slice('ns-preset:'.length);
                if (value) s.presets.add(key);
                else s.presets.delete(key);
                paintCounts();
            } else if (name.startsWith('ns-mcp:')) {
                const key = name.slice('ns-mcp:'.length);
                if (value) s.mcp.add(key);
                else s.mcp.delete(key);
                paintCounts();
            }
            paintFooter();
        });

        el.addEventListener('input', (e) => {
            const { name } = e.target.dataset;
            const value = e.target.value;
            if (name === 'ns-empty-name') {
                s.emptyName = value;
                syncRc();
                paint('empty-name-check', nameCheck(target(), 'Creates').html);
            } else if (name === 'ns-clone-name') {
                s.cloneName = value;
                s.cloneTouched = true;
                syncRc();
                paint('clone-name-check', nameCheck(target(), 'Clones into').html);
            } else if (name === 'ns-repo-query') {
                s.repoQuery = value;
                paint('repos', reposHtml());
            } else if (name === 'ns-rc') {
                s.rcName = value;
                s.rcTouched = value.trim() !== '';
            } else if (name === 'ns-preset-draft') {
                s.presetDraft = value;
                if (s.presetError) {
                    s.presetError = '';
                    paint('preset-error', '');
                }
            }
            paintFooter();
        });

        el.addEventListener('submit', (e) => {
            if (!e.target.matches('[data-ns="preset-form"]')) return;
            e.preventDefault();
            addPreset();
        });

        el.addEventListener('click', (e) => {
            const t = e.target.closest('[data-ns]');
            if (!t || t.hasAttribute('disabled')) return;
            const action = t.dataset.ns;
            if (action === 'repo') {
                s.repo = t.dataset.repo;
                if (!s.cloneTouched) s.cloneName = s.repo.split('/')[1].toLowerCase();
                const list = el.querySelector('[data-region="repos"]');
                const scroll = list.parentElement.scrollTop;
                paint('source-body', sourceBodyHtml());
                el.querySelector('[data-region="repos"]').parentElement.scrollTop = scroll;
                syncRc();
            } else if (action === 'retry-dirs') {
                loadDirs();
            } else if (action === 'retry-repos') {
                loadRepos();
            } else if (action === 'copy') {
                ui.copy(command().map(([, v]) => v).join(' '));
            } else if (action === 'cancel') {
                m.close();
            } else if (action === 'open-project') {
                const name = s.dir;
                m.close();
                app.go(`#/project/${encodeURIComponent(name)}`);
            } else if (action === 'launch') {
                launch();
            } else {
                return;
            }
            paintFooter();
        });

        loadDirs();
        return m;
    }

    return { open };
})();
