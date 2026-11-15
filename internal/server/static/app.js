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
        ui.initScrollObserver();
        app.renderNav();
        window.addEventListener('hashchange', () => app.route());
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') app.refresh();
        });
        window.addEventListener('pageshow', (e) => {
            if (e.persisted) app.refresh();
        });
        await app.refresh({ silent: true });
        app.route();
    },

    async refresh({ silent = false } = {}) {
        store.loading = true;
        app.renderNav();
        try {
            store.state = await api.state();
        } catch (err) {
            if (!silent) ui.toast(err.message, { tone: 'red', icon: 'circle-alert' });
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
        navLinks.innerHTML = links.map((l) => {
            const on = app.current === l.name;
            return `<a href="${l.hash}" class="flex h-9 items-center gap-2 rounded-full px-4 text-sm whitespace-nowrap transition-colors ${on ? 'bg-surface0 text-text' : 'text-overlay1 hover:text-text'}" ${on ? 'aria-current="page"' : ''}>
                <i data-lucide="${l.icon}" class="size-4 ${on ? 'text-mauve' : ''}"></i><span class="${on ? '' : 'max-sm:hidden'}">${l.label}</span></a>`;
        }).join('');
        const status = document.getElementById('nav-status');
        status.innerHTML = store.loading ? '<i data-lucide="loader-circle" class="size-4 animate-spin"></i>' : '';
        ui.icons(navLinks);
        ui.icons(status);
    },
};

const ui = {
    actions: {},

    esc(value) {
        return String(value ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    },

    icons(root = document) {
        lucide.createIcons({ root });
    },

    on(name, fn) {
        ui.actions[name] = fn;
    },

    initDelegation() {
        ui.on('app-home', () => {
            if (app.current === 'dashboard' && !app.params.length) app.refresh();
            else {
                app.go('#/');
                app.refresh();
            }
        });
        ui.on('new-session', () => views.newSession.open());

        document.addEventListener('click', (e) => {
            const control = e.target.closest('[data-ui]');
            if (control && !control.hasAttribute('disabled') && control.getAttribute('aria-disabled') !== 'true') {
                ui.handleControl(control, e);
            }
            const trigger = e.target.closest('[data-action]');
            if (trigger && !trigger.hasAttribute('disabled')) {
                const fn = ui.actions[trigger.dataset.action];
                if (fn) {
                    e.preventDefault();
                    fn(trigger, e);
                }
            }
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (ui.popover.close()) return;
                const top = ui.modal.stack.at(-1);
                if (top && top.dismissible) top.close();
                return;
            }
            const control = e.target.closest?.('[data-ui]');
            if (!control) return;
            const role = control.getAttribute('role');
            if (role === 'radio' && ['ArrowRight', 'ArrowDown', 'ArrowLeft', 'ArrowUp'].includes(e.key)) {
                e.preventDefault();
                const group = control.closest('[role="radiogroup"]');
                const radios = [...group.querySelectorAll('[role="radio"]:not([disabled])')];
                const step = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : -1;
                const next = radios[(radios.indexOf(control) + step + radios.length) % radios.length];
                next.focus();
                ui.handleControl(next, e);
            }
        });
    },

    handleControl(control, e) {
        const kind = control.dataset.ui;
        if (kind === 'checkbox' || kind === 'switch') {
            const checked = control.getAttribute('aria-checked') !== 'true';
            control.setAttribute('aria-checked', String(checked));
            ui.emit(control, checked);
        } else if (kind === 'radio') {
            const group = control.closest('[role="radiogroup"]');
            group.querySelectorAll('[role="radio"]').forEach((r) => {
                r.setAttribute('aria-checked', String(r === control));
                r.tabIndex = r === control ? 0 : -1;
            });
            group.dataset.value = control.dataset.value;
            ui.emit(group, control.dataset.value);
        } else if (kind === 'select') {
            e.preventDefault();
            ui.openSelect(control);
        }
    },

    emit(el, value) {
        el.dispatchEvent(new CustomEvent('ui-change', { bubbles: true, detail: { name: el.dataset.name, value } }));
    },

    value(root, name) {
        const el = root.querySelector(`[data-name="${name}"]`);
        if (!el) return undefined;
        const kind = el.dataset.ui;
        if (kind === 'checkbox' || kind === 'switch') return el.getAttribute('aria-checked') === 'true';
        if (kind === 'radiogroup' || kind === 'select') return el.dataset.value ?? '';
        if (el.matches('input, textarea')) return el.value;
        return undefined;
    },

    cls: {
        input: 'w-full rounded-2xl bg-surface0 px-4 py-2.5 text-sm text-text placeholder:text-overlay0 outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px]',
        label: 'text-xs font-medium uppercase tracking-wide text-overlay1',
        panel: 'rounded-3xl bg-mantle',
        well: 'rounded-2xl bg-base',
        row: 'rounded-2xl transition-colors hover:bg-surface0',
    },

    button(label, { icon, variant = 'secondary', size = 'md', action, attrs = '', disabled = false, collapse = false, spin = false, type = 'button' } = {}) {
        const variants = {
            primary: 'bg-mauve text-crust hover:bg-mauve/85',
            secondary: 'bg-surface0 text-text hover:bg-surface1',
            ghost: 'text-subtext0 hover:bg-surface0 hover:text-text',
            danger: 'bg-red/15 text-red hover:bg-red/25',
            success: 'bg-green/15 text-green hover:bg-green/25',
        };
        const sizes = { sm: 'h-8 px-3 text-xs gap-1.5', md: 'h-10 px-4 text-sm gap-2', lg: 'h-12 px-6 text-sm gap-2' };
        const iconSize = size === 'sm' ? 'size-3.5' : 'size-4';
        const iconHtml = icon ? `<i data-lucide="${icon}" class="${iconSize} shrink-0 ${spin ? 'animate-spin' : ''}"></i>` : '';
        const labelHtml = collapse ? `<span class="hidden sm:inline">${ui.esc(label)}</span>` : `<span>${ui.esc(label)}</span>`;
        return `<button type="${type}" class="inline-flex shrink-0 items-center justify-center rounded-full font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none ${collapse ? 'max-sm:w-10 max-sm:px-0' : ''} ${variants[variant]} ${sizes[size]}" ${action ? `data-action="${action}"` : ''} ${disabled ? 'disabled' : ''} ${collapse ? `title="${ui.esc(label)}"` : ''} ${attrs}>${iconHtml}${labelHtml}</button>`;
    },

    iconButton(icon, { title = '', action, variant = 'ghost', size = 'md', attrs = '', spin = false, disabled = false } = {}) {
        const variants = {
            ghost: 'text-overlay1 hover:bg-surface0 hover:text-text',
            secondary: 'bg-surface0 text-text hover:bg-surface1',
            danger: 'text-red hover:bg-red/15',
        };
        const sizes = { sm: 'size-8', md: 'size-10' };
        return `<button type="button" title="${ui.esc(title)}" aria-label="${ui.esc(title)}" class="inline-flex shrink-0 items-center justify-center rounded-full transition-colors disabled:opacity-50 disabled:pointer-events-none ${variants[variant]} ${sizes[size]}" ${action ? `data-action="${action}"` : ''} ${disabled ? 'disabled' : ''} ${attrs}><i data-lucide="${icon}" class="size-4 ${spin ? 'animate-spin' : ''}"></i></button>`;
    },

    chip(text, { tone = 'neutral', icon, mono = false, title = '' } = {}) {
        const tones = {
            neutral: 'bg-surface0 text-subtext0',
            mauve: 'bg-mauve/15 text-mauve',
            green: 'bg-green/15 text-green',
            red: 'bg-red/15 text-red',
            peach: 'bg-peach/15 text-peach',
            blue: 'bg-blue/15 text-blue',
            yellow: 'bg-yellow/15 text-yellow',
        };
        return `<span ${title ? `title="${ui.esc(title)}"` : ''} class="inline-flex max-w-full items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${tones[tone]} ${mono ? 'font-mono' : ''}">${icon ? `<i data-lucide="${icon}" class="size-3 shrink-0"></i>` : ''}<span class="truncate">${ui.esc(text)}</span></span>`;
    },

    status(active) {
        return active
            ? `<span class="inline-flex items-center gap-2 rounded-full bg-green/15 px-2.5 py-1 text-xs font-medium text-green"><span class="relative flex size-2"><span class="absolute inline-flex size-full animate-ping rounded-full bg-green opacity-60"></span><span class="relative inline-flex size-2 rounded-full bg-green"></span></span>Active</span>`
            : `<span class="inline-flex items-center gap-2 rounded-full bg-surface0 px-2.5 py-1 text-xs font-medium text-overlay1"><span class="size-2 rounded-full bg-overlay0"></span>Inactive</span>`;
    },

    checkbox({ name, label, hint = '', checked = false, disabled = false }) {
        return `<button type="button" role="checkbox" data-ui="checkbox" data-name="${ui.esc(name)}" aria-checked="${checked}" ${disabled ? 'disabled' : ''} class="group flex w-full items-start gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
            <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-lg bg-surface1 text-crust transition-colors group-aria-checked:bg-mauve"><i data-lucide="check" class="size-3.5 scale-50 opacity-0 transition-all group-aria-checked:scale-100 group-aria-checked:opacity-100" stroke-width="3"></i></span>
            <span class="min-w-0"><span class="block text-sm text-text">${ui.esc(label)}</span>${hint ? `<span class="block text-xs text-overlay1">${ui.esc(hint)}</span>` : ''}</span>
        </button>`;
    },

    switch({ name, label, hint = '', checked = false, disabled = false }) {
        return `<button type="button" role="switch" data-ui="switch" data-name="${ui.esc(name)}" aria-checked="${checked}" ${disabled ? 'disabled' : ''} class="group flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
            <span class="min-w-0 flex-1"><span class="block text-sm text-text">${ui.esc(label)}</span>${hint ? `<span class="block text-xs text-overlay1">${ui.esc(hint)}</span>` : ''}</span>
            <span class="flex h-6 w-11 shrink-0 items-center rounded-full bg-surface1 p-0.5 transition-colors group-aria-checked:bg-mauve"><span class="size-5 rounded-full bg-text shadow transition-transform group-aria-checked:translate-x-5 group-aria-checked:bg-crust"></span></span>
        </button>`;
    },

    segmented({ name, value, options, full = false }) {
        return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="${full ? 'flex w-full' : 'inline-flex'} items-center gap-1 rounded-full bg-crust p-1">
            ${options.map((o) => {
                const on = o.value === value;
                return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="flex ${full ? 'flex-1' : ''} items-center justify-center gap-2 rounded-full px-3.5 py-1.5 text-sm font-medium text-overlay1 transition-colors hover:text-text aria-checked:bg-surface0 aria-checked:text-text disabled:opacity-40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">${o.icon ? `<i data-lucide="${o.icon}" class="size-4"></i>` : ''}<span class="${o.icon && o.compact ? 'hidden sm:inline' : ''}">${ui.esc(o.label)}</span></button>`;
            }).join('')}
        </span>`;
    },

    choices({ name, value, options, columns = 1 }) {
        const grid = { 1: 'grid-cols-1', 2: 'grid-cols-1 sm:grid-cols-2', 3: 'grid-cols-1 sm:grid-cols-3' }[columns];
        return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="grid ${grid} gap-2">
            ${options.map((o) => {
                const on = o.value === value;
                return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="group flex items-start gap-3 rounded-2xl bg-surface0/50 p-3 text-left transition-colors hover:bg-surface0 aria-checked:bg-mauve/12 disabled:opacity-40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
                    <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-surface1 transition-colors group-aria-checked:bg-mauve"><span class="size-2 scale-0 rounded-full bg-crust transition-transform group-aria-checked:scale-100"></span></span>
                    <span class="min-w-0"><span class="flex items-center gap-2 text-sm font-medium text-text">${o.icon ? `<i data-lucide="${o.icon}" class="size-4 text-overlay1 group-aria-checked:text-mauve"></i>` : ''}${ui.esc(o.label)}</span>${o.hint ? `<span class="mt-0.5 block text-xs text-overlay1">${ui.esc(o.hint)}</span>` : ''}</span>
                </button>`;
            }).join('')}
        </span>`;
    },

    select({ name, value = '', options, placeholder = 'Select', searchable = false, icon }) {
        const current = options.find((o) => o.value === value);
        ui.selectOptions[name] = { options, searchable, placeholder };
        return `<button type="button" data-ui="select" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" aria-haspopup="listbox" aria-expanded="false" class="flex w-full items-center gap-2.5 rounded-2xl bg-surface0 px-4 py-2.5 text-left text-sm transition-colors hover:bg-surface1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40 pointer-coarse:text-[16px]">
            ${icon ? `<i data-lucide="${icon}" class="size-4 shrink-0 text-overlay1"></i>` : ''}
            <span data-select-label class="min-w-0 flex-1 truncate ${current ? 'text-text' : 'text-overlay0'}">${ui.esc(current ? current.label : placeholder)}</span>
            <i data-lucide="chevrons-up-down" class="size-4 shrink-0 text-overlay1"></i>
        </button>`;
    },

    selectOptions: {},

    openSelect(button) {
        const name = button.dataset.name;
        const { options, searchable } = ui.selectOptions[name] || { options: [] };
        let query = '';
        let active = Math.max(0, options.findIndex((o) => o.value === button.dataset.value));
        button.setAttribute('aria-expanded', 'true');

        const filtered = () => (query ? options.map((o) => ({ o, s: ui.fuzzy(query, `${o.label} ${o.hint || ''}`) })).filter((x) => x.s >= 0).sort((a, b) => b.s - a.s).map((x) => x.o) : options);

        const listHtml = () => {
            const items = filtered();
            if (!items.length) return `<li class="px-3 py-6 text-center text-sm text-overlay1">No matches</li>`;
            return items.map((o, i) => {
                const selected = o.value === button.dataset.value;
                return `<li role="option" aria-selected="${selected}" data-index="${i}" data-value="${ui.esc(o.value)}" class="flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2 text-sm ${i === active ? 'bg-surface0 text-text' : 'text-subtext0'}">
                    ${o.icon ? `<i data-lucide="${o.icon}" class="size-4 shrink-0 text-overlay1"></i>` : ''}
                    <span class="min-w-0 flex-1"><span class="block truncate">${ui.esc(o.label)}</span>${o.hint ? `<span class="block truncate text-xs text-overlay1">${ui.esc(o.hint)}</span>` : ''}</span>
                    ${selected ? '<i data-lucide="check" class="size-4 shrink-0 text-mauve"></i>' : ''}
                </li>`;
            }).join('');
        };

        const pop = ui.popover.open(button, `
            ${searchable ? `<input data-select-search class="${ui.cls.input} mb-1" placeholder="Search">` : ''}
            <ul role="listbox" data-scroll class="max-h-72 overflow-y-auto">${listHtml()}</ul>`, {
            matchWidth: true,
            onClose: () => button.setAttribute('aria-expanded', 'false'),
        });
        const list = pop.querySelector('ul');
        const search = pop.querySelector('[data-select-search]');
        const redraw = () => {
            list.innerHTML = listHtml();
            ui.icons(list);
            list.querySelector(`[data-index="${active}"]`)?.scrollIntoView({ block: 'nearest' });
        };
        const choose = (value) => {
            const opt = options.find((o) => o.value === value);
            button.dataset.value = value;
            const label = button.querySelector('[data-select-label]');
            label.textContent = opt.label;
            label.classList.remove('text-overlay0');
            label.classList.add('text-text');
            ui.popover.close();
            button.focus();
            ui.emit(button, value);
        };
        list.addEventListener('mousemove', (e) => {
            const li = e.target.closest('[data-index]');
            if (li && Number(li.dataset.index) !== active) {
                active = Number(li.dataset.index);
                redraw();
            }
        });
        list.addEventListener('click', (e) => {
            const li = e.target.closest('[data-value]');
            if (li) choose(li.dataset.value);
        });
        const keys = (e) => {
            const items = filtered();
            if (e.key === 'ArrowDown') { e.preventDefault(); active = Math.min(items.length - 1, active + 1); redraw(); }
            else if (e.key === 'ArrowUp') { e.preventDefault(); active = Math.max(0, active - 1); redraw(); }
            else if (e.key === 'Enter' && items[active]) { e.preventDefault(); choose(items[active].value); }
        };
        (search || list).addEventListener('keydown', keys);
        if (search) {
            search.addEventListener('input', () => {
                query = search.value;
                active = 0;
                redraw();
            });
            search.focus();
        } else {
            list.tabIndex = -1;
            list.focus();
        }
    },

    popover: {
        current: null,

        open(anchor, html, { matchWidth = false, align = 'start', onClose, width } = {}) {
            ui.popover.close();
            const el = document.createElement('div');
            el.className = 'fixed z-[70] rounded-2xl bg-mantle p-1.5 shadow-2xl shadow-crust/80 ring-1 ring-surface0 [animation:pop-in_.12s_ease-out]';
            el.innerHTML = html;
            document.getElementById('layer-root').appendChild(el);
            const r = anchor.getBoundingClientRect();
            const w = width || (matchWidth ? Math.max(r.width, 220) : el.offsetWidth);
            el.style.width = `${Math.min(w, innerWidth - 16)}px`;
            const left = align === 'end' ? r.right - el.offsetWidth : r.left;
            el.style.left = `${Math.max(8, Math.min(left, innerWidth - el.offsetWidth - 8))}px`;
            const below = r.bottom + 6;
            el.style.top = below + el.offsetHeight > innerHeight - 8 && r.top - el.offsetHeight - 6 > 8 ? `${r.top - el.offsetHeight - 6}px` : `${below}px`;
            ui.icons(el);
            const outside = (e) => {
                if (!el.contains(e.target) && !anchor.contains(e.target)) ui.popover.close();
            };
            setTimeout(() => document.addEventListener('pointerdown', outside), 0);
            ui.popover.current = { el, close: () => { document.removeEventListener('pointerdown', outside); el.remove(); onClose?.(); } };
            return el;
        },

        close() {
            if (!ui.popover.current) return false;
            const c = ui.popover.current;
            ui.popover.current = null;
            c.close();
            return true;
        },
    },

    menu(anchor, items) {
        const el = ui.popover.open(anchor, items.map((item, i) => item === '-' ? '<span class="my-1 block h-px"></span>' : `
            <button type="button" data-menu-index="${i}" ${item.disabled ? 'disabled' : ''} class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-sm transition-colors disabled:opacity-40 ${item.danger ? 'text-red hover:bg-red/10' : 'text-subtext0 hover:bg-surface0 hover:text-text'}">
                <i data-lucide="${item.icon}" class="size-4 shrink-0"></i><span class="min-w-0 flex-1"><span class="block">${ui.esc(item.label)}</span>${item.hint ? `<span class="block text-xs text-overlay1">${ui.esc(item.hint)}</span>` : ''}</span>
            </button>`).join(''), { align: 'end', width: 260 });
        el.addEventListener('click', (e) => {
            const b = e.target.closest('[data-menu-index]');
            if (!b) return;
            ui.popover.close();
            items[Number(b.dataset.menuIndex)].onSelect();
        });
    },

    modal: {
        stack: [],

        open({ title, icon, subtitle = '', body = '', footer = '', size = 'md', dismissible = true, onClose }) {
            const sizes = { sm: 'md:max-w-md', md: 'md:max-w-2xl', lg: 'md:max-w-4xl', xl: 'md:max-w-6xl 3xl:max-w-[100rem]' };
            const root = document.createElement('div');
            root.className = 'fixed inset-0 z-50 flex items-end md:items-center justify-center bg-crust/70 backdrop-blur-sm md:p-6 [animation:fade-in_.15s_ease-out]';
            root.innerHTML = `
                <section role="dialog" aria-modal="true" class="flex max-h-[92dvh] w-full flex-col rounded-t-3xl bg-mantle shadow-2xl shadow-crust md:max-h-[86dvh] md:rounded-3xl ${sizes[size]} max-md:[animation:sheet-in_.2s_ease-out] md:[animation:pop-in_.15s_ease-out]">
                    <header class="flex shrink-0 items-center gap-3 px-5 pt-5 pb-3 md:px-6 md:pt-6">
                        ${icon ? `<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-mauve/15 text-mauve"><i data-lucide="${icon}" class="size-5"></i></span>` : ''}
                        <span class="min-w-0 flex-1"><h2 class="truncate font-display text-lg font-medium text-text">${ui.esc(title)}</h2>${subtitle ? `<p class="truncate text-sm text-overlay1">${subtitle}</p>` : ''}</span>
                        ${dismissible ? ui.iconButton('x', { title: 'Close', attrs: 'data-modal-close' }) : ''}
                    </header>
                    <div data-modal-body data-scroll class="min-h-0 flex-1 overflow-y-auto px-5 pb-4 md:px-6">${body}</div>
                    ${footer ? `<footer data-modal-footer class="flex shrink-0 flex-wrap items-center justify-end gap-2 rounded-b-3xl bg-crust/40 px-5 py-4 md:px-6">${footer}</footer>` : ''}
                </section>`;
            document.getElementById('layer-root').appendChild(root);
            const entry = {
                el: root.firstElementChild,
                dismissible,
                close() {
                    if (!root.isConnected) return;
                    ui.popover.close();
                    root.remove();
                    ui.modal.stack.splice(ui.modal.stack.indexOf(entry), 1);
                    onClose?.();
                },
            };
            ui.modal.stack.push(entry);
            root.addEventListener('pointerdown', (e) => {
                if (e.target === root && dismissible) entry.close();
            });
            root.querySelector('[data-modal-close]')?.addEventListener('click', () => entry.close());
            ui.icons(root);
            return entry;
        },
    },

    confirm({ title, message, confirmLabel = 'Confirm', danger = false, icon = 'circle-help' }) {
        return new Promise((resolve) => {
            let answered = false;
            const m = ui.modal.open({
                title,
                icon,
                size: 'sm',
                body: `<p class="text-sm leading-relaxed text-subtext0">${message}</p>`,
                footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-confirm="no"' })}${ui.button(confirmLabel, { variant: danger ? 'danger' : 'primary', attrs: 'data-confirm="yes"' })}`,
                onClose: () => { if (!answered) resolve(false); },
            });
            m.el.addEventListener('click', (e) => {
                const b = e.target.closest('[data-confirm]');
                if (!b) return;
                answered = true;
                m.close();
                resolve(b.dataset.confirm === 'yes');
            });
        });
    },

    toast(message, { tone = 'neutral', icon = 'info', timeout = 3500 } = {}) {
        let host = document.getElementById('toasts');
        if (!host) {
            host = document.createElement('div');
            host.id = 'toasts';
            host.className = 'pointer-events-none fixed inset-x-3 bottom-3 z-[80] flex flex-col items-center gap-2 md:inset-x-auto md:right-5 md:bottom-5 md:items-end';
            document.getElementById('layer-root').appendChild(host);
        }
        const tones = { neutral: 'text-text', green: 'text-green', red: 'text-red', peach: 'text-peach', mauve: 'text-mauve' };
        const el = document.createElement('div');
        el.className = 'pointer-events-auto flex max-w-md items-center gap-3 rounded-full bg-surface0 py-2.5 pl-4 pr-5 text-sm text-text shadow-xl shadow-crust/70 [animation:toast-in_.18s_ease-out]';
        el.innerHTML = `<i data-lucide="${icon}" class="size-4 shrink-0 ${tones[tone]}"></i><span>${ui.esc(message)}</span>`;
        host.appendChild(el);
        ui.icons(el);
        setTimeout(() => {
            el.style.transition = 'opacity .25s, transform .25s';
            el.style.opacity = '0';
            el.style.transform = 'translateY(8px)';
            setTimeout(() => el.remove(), 260);
        }, timeout);
    },

    empty({ icon, title, hint = '', action = '' }) {
        return `<div class="flex h-full min-h-40 flex-col items-center justify-center gap-3 p-6 text-center">
            <span class="flex size-12 items-center justify-center rounded-full bg-surface0 text-overlay1"><i data-lucide="${icon}" class="size-5"></i></span>
            <p class="font-display text-[16px] text-text">${ui.esc(title)}</p>
            ${hint ? `<p class="max-w-sm text-sm text-overlay1">${hint}</p>` : ''}
            ${action}
        </div>`;
    },

    fuzzy(query, text) {
        const q = query.toLowerCase().replace(/\s+/g, '');
        const t = text.toLowerCase();
        if (!q) return 0;
        const direct = t.indexOf(q);
        if (direct >= 0) return 1000 - direct;
        let score = 0;
        let ti = 0;
        let streak = 0;
        for (const c of q) {
            const found = t.indexOf(c, ti);
            if (found < 0) return -1;
            streak = found === ti ? streak + 1 : 0;
            score += 1 + streak * 2;
            ti = found + 1;
        }
        return score;
    },

    timeAgo(ts) {
        const s = Math.max(0, Math.round((Date.now() - ts) / 1000));
        if (s < 60) return 'just now';
        const units = [['d', 86400], ['h', 3600], ['m', 60]];
        for (const [u, n] of units) if (s >= n) return `${Math.floor(s / n)}${u} ago`;
        return 'just now';
    },

    async copy(text) {
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            const ta = document.createElement('textarea');
            ta.value = text;
            ta.style.position = 'fixed';
            ta.style.opacity = '0';
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            ta.remove();
        }
        ui.toast('Copied to clipboard', { tone: 'green', icon: 'clipboard-check', timeout: 1800 });
    },

    initScrollObserver() {
        const attach = (root) => {
            if (root.matches?.('[data-scroll]')) ui.scrollbar(root);
            root.querySelectorAll?.('[data-scroll]').forEach((el) => ui.scrollbar(el));
        };
        attach(document.body);
        new MutationObserver((records) => {
            for (const r of records) r.addedNodes.forEach((n) => n.nodeType === 1 && attach(n));
        }).observe(document.body, { childList: true, subtree: true });
    },

    scrollbar(el) {
        if (el._scrollbar) return;
        el._scrollbar = true;
        if (getComputedStyle(el).position === 'static') el.style.position = 'relative';
        const thumb = document.createElement('span');
        thumb.setAttribute('aria-hidden', 'true');
        thumb.className = 'absolute right-0.5 z-20 w-1.5 rounded-full bg-overlay0/60 opacity-0 transition-[opacity,width,background-color] duration-300 hover:w-2 hover:bg-overlay1 cursor-default';
        thumb.style.top = '0px';
        el.appendChild(thumb);
        let hideTimer;
        let dragging = null;

        const place = () => {
            const { scrollTop, scrollHeight, clientHeight } = el;
            const overflow = scrollHeight - clientHeight;
            if (overflow <= 1) {
                thumb.style.display = 'none';
                return false;
            }
            thumb.style.display = '';
            const h = Math.max(28, (clientHeight / scrollHeight) * clientHeight - 8);
            const top = scrollTop + 4 + (scrollTop / overflow) * (clientHeight - h - 8);
            thumb.style.height = `${h}px`;
            thumb.style.transform = `translateY(${top}px)`;
            return true;
        };
        const show = () => {
            if (!place()) return;
            thumb.style.opacity = '1';
            clearTimeout(hideTimer);
            hideTimer = setTimeout(() => { if (!dragging) thumb.style.opacity = '0'; }, 900);
        };
        el.addEventListener('scroll', show, { passive: true });
        el.addEventListener('pointerenter', show);
        new ResizeObserver(() => place()).observe(el);
        new MutationObserver(() => {
            if (thumb.parentNode !== el) el.appendChild(thumb);
            place();
        }).observe(el, { childList: true, subtree: true });

        thumb.addEventListener('pointerdown', (e) => {
            e.preventDefault();
            e.stopPropagation();
            thumb.setPointerCapture(e.pointerId);
            dragging = { y: e.clientY, top: el.scrollTop };
        });
        thumb.addEventListener('pointermove', (e) => {
            if (!dragging) return;
            const ratio = (el.scrollHeight - el.clientHeight) / (el.clientHeight - thumb.offsetHeight - 8);
            el.scrollTop = dragging.top + (e.clientY - dragging.y) * ratio;
        });
        thumb.addEventListener('pointerup', () => {
            dragging = null;
            show();
        });
        place();
    },

    sidebar(panel, { key, min = 200, max = 520, initial = 280, side = 'left', collapsed = false }) {
        const load = (k, fallback) => {
            try {
                const v = localStorage.getItem(`claudex.${key}.${k}`);
                return v === null ? fallback : JSON.parse(v);
            } catch {
                return fallback;
            }
        };
        const save = (k, v) => {
            try { localStorage.setItem(`claudex.${key}.${k}`, JSON.stringify(v)); } catch {}
        };
        const state = { width: load('width', initial), collapsed: load('collapsed', collapsed) };
        panel.style.flexShrink = '0';
        if (getComputedStyle(panel).position === 'static') panel.style.position = 'relative';

        const handle = document.createElement('span');
        handle.setAttribute('role', 'separator');
        handle.setAttribute('aria-orientation', 'vertical');
        handle.title = 'Drag to resize, double-click to reset';
        handle.className = `group/handle absolute inset-y-0 ${side === 'left' ? '-right-2.5' : '-left-2.5'} z-30 flex w-3 cursor-col-resize touch-none items-center justify-center`;
        handle.innerHTML = '<span class="h-10 w-1 rounded-full bg-surface1 opacity-0 transition-opacity group-hover/handle:opacity-100 group-active/handle:bg-mauve group-active/handle:opacity-100"></span>';
        panel.appendChild(handle);

        const apply = (animate) => {
            panel.style.transition = animate ? 'width .2s ease, opacity .2s ease, margin .2s ease' : 'none';
            panel.style.width = state.collapsed ? '0px' : `${state.width}px`;
            panel.style.opacity = state.collapsed ? '0' : '1';
            panel.style.overflow = state.collapsed ? 'hidden' : '';
            panel.style.pointerEvents = state.collapsed ? 'none' : '';
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

    media: {
        mobile: () => matchMedia('(max-width: 767px)').matches,
        wide: () => matchMedia('(min-width: 1920px)').matches,
    },
};
