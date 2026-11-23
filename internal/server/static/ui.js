const ui = {
    actions: {},
    selectOptions: {},

    esc(value) {
        return String(value ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    },

    icons(root = document) {
        lucide.createIcons({ root });
    },

    on(name, fn) {
        ui.actions[name] = fn;
    },

    cls: {
        input: 'w-full rounded-2xl bg-surface0 px-4 py-2.5 text-sm text-text placeholder:text-overlay0 outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 aria-[invalid=true]:ring-2 aria-[invalid=true]:ring-red/50 pointer-coarse:text-[16px]',
        label: 'text-xs font-medium uppercase tracking-wide text-overlay1',
        group: 'px-3 pt-4 pb-1 text-[11px] font-medium uppercase tracking-wider text-overlay0',
        panel: 'rounded-3xl bg-mantle',
        well: 'rounded-2xl bg-base',
        tile: 'rounded-2xl bg-surface0/50',
        row: 'rounded-2xl transition-colors hover:bg-surface0/50',
        focus: 'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40',
        count: 'inline-flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 font-mono text-[11px] tabular-nums',
        rowText: 'truncate pointer-coarse:whitespace-normal pointer-coarse:wrap-break-word',
        reveal: 'opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100 has-[[aria-expanded=true]]:opacity-100 max-md:opacity-100',
    },

    tone: {
        chip: (t) => (t === 'neutral' ? 'bg-surface0 text-subtext0' : `bg-${t}/15 text-${t}`),
        text: (t) => (t === 'neutral' ? 'text-text' : `text-${t}`),
    },

    tracks: {
        panel: { track: 'bg-surface0', on: 'aria-checked:bg-surface1', count: 'bg-surface1' },
        ground: { track: 'bg-mantle', on: 'aria-checked:bg-surface0', count: 'bg-surface0' },
    },

    initDelegation() {
        document.addEventListener('click', (e) => {
            const control = e.target.closest('[data-ui]');
            if (control && !control.hasAttribute('disabled')) ui.handleControl(control, e);
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
                ui.tooltip.hide();
                if (ui.popover.close()) return;
                const top = ui.modal.stack.at(-1);
                if (top && top.dismissible) top.close();
                return;
            }
            const control = e.target.closest?.('[data-ui="radio"]');
            if (control && ['ArrowRight', 'ArrowDown', 'ArrowLeft', 'ArrowUp'].includes(e.key)) {
                e.preventDefault();
                const radios = [...control.closest('[role="radiogroup"]').querySelectorAll('[role="radio"]:not([disabled])')];
                const step = e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : -1;
                const next = radios[(radios.indexOf(control) + step + radios.length) % radios.length];
                next.focus();
                ui.handleControl(next, e);
            }
        });

        document.addEventListener('input', (e) => {
            if (e.target.matches('input[data-name], textarea[data-name]')) ui.emit(e.target, e.target.value);
        });

        let pasted = '';
        // Chrome reads the clipboard again to insert a paste and inserts nothing if another app changed it since the event.
        document.addEventListener('paste', (e) => { pasted = e.clipboardData?.getData('text/plain') || ''; }, true);
        document.addEventListener('paste', (e) => {
            if (e.defaultPrevented || !pasted || e.target.readOnly || !e.target.matches?.('textarea, input:not([type]), input[type=text], input[type=search], input[type=url], input[type=email], input[type=tel], input[type=password]')) return;
            e.preventDefault();
            document.execCommand('insertText', false, pasted);
        });

        document.addEventListener('pointerdown', (e) => {
            const grip = e.target.closest('[data-ui="grip"]');
            if (!grip) return;
            const ta = grip.parentElement.querySelector('textarea');
            e.preventDefault();
            grip.setPointerCapture(e.pointerId);
            const y0 = e.clientY;
            const h0 = ta.offsetHeight;
            const move = (ev) => { ta.style.height = `${Math.max(80, h0 + ev.clientY - y0)}px`; };
            const up = () => {
                grip.removeEventListener('pointermove', move);
                grip.removeEventListener('pointerup', up);
                grip.removeEventListener('pointercancel', up);
            };
            grip.addEventListener('pointermove', move);
            grip.addEventListener('pointerup', up);
            grip.addEventListener('pointercancel', up);
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

    button(label, { icon, iconRight, variant = 'secondary', size = 'md', action, attrs = '', disabled = false, collapse = false, spin = false, type = 'button', tip = '' } = {}) {
        const variants = {
            primary: 'bg-mauve text-crust hover:bg-mauve/85',
            secondary: 'bg-surface0 text-text hover:bg-surface1',
            ghost: 'text-subtext0 hover:bg-surface0 hover:text-text',
            danger: 'bg-red/15 text-red hover:bg-red/25',
            success: 'bg-green/15 text-green hover:bg-green/25',
            'danger-solid': 'bg-red text-crust hover:bg-red/85',
        };
        const sizes = { sm: 'h-8 px-3 text-xs gap-1.5', md: 'h-10 px-4 text-sm gap-2', lg: 'h-12 px-6 text-sm gap-2' };
        const iconSize = size === 'sm' ? 'size-3.5' : 'size-4';
        const i = (name, extra = '') => `<i data-lucide="${name}" class="${iconSize} shrink-0 ${extra}"></i>`;
        const labelHtml = collapse ? `<span data-label class="max-sm:hidden">${ui.esc(label)}</span>` : `<span>${ui.esc(label)}</span>`;
        const tipText = tip || (collapse ? label : '');
        const tipAttrs = `data-tip="${ui.esc(tipText)}" ${tip ? '' : 'data-tip-collapse'}`;
        const btn = `<button type="${type}" ${tipText && !disabled ? tipAttrs : ''} class="inline-flex shrink-0 items-center justify-center rounded-full font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus} ${collapse ? 'max-sm:w-10 max-sm:px-0' : ''} ${variants[variant]} ${sizes[size]}" ${action ? `data-action="${action}"` : ''} ${disabled ? 'disabled' : ''} ${attrs}>${icon ? i(icon, spin ? 'animate-spin' : '') : ''}${labelHtml}${iconRight ? i(iconRight, '-mr-1 opacity-70') : ''}</button>`;
        return disabled && tipText ? `<span ${tipAttrs} class="inline-flex">${btn}</span>` : btn;
    },

    iconButton(icon, { tip = '', action, variant = 'ghost', size = 'md', attrs = '', spin = false, disabled = false, pressed } = {}) {
        const variants = {
            ghost: 'text-overlay1 hover:bg-surface0 hover:text-text aria-pressed:bg-surface0 aria-pressed:text-text',
            inset: 'text-overlay1 hover:bg-surface1 hover:text-text aria-pressed:bg-surface1 aria-pressed:text-text',
            secondary: 'bg-surface0 text-text hover:bg-surface1',
            primary: 'bg-mauve text-crust hover:bg-mauve/85',
            danger: 'text-red hover:bg-red/15',
        };
        const sizes = { sm: 'size-8', md: 'size-10' };
        const btn = `<button type="button" aria-label="${ui.esc(tip)}" ${disabled ? '' : `data-tip="${ui.esc(tip)}"`} ${pressed === undefined ? '' : `aria-pressed="${pressed}"`} class="inline-flex shrink-0 items-center justify-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus} ${variants[variant]} ${sizes[size]}" ${action ? `data-action="${action}"` : ''} ${disabled ? 'disabled' : ''} ${attrs}><i data-lucide="${icon}" class="size-4 ${spin ? 'animate-spin' : ''}"></i></button>`;
        return disabled ? `<span data-tip="${ui.esc(tip)}" class="inline-flex">${btn}</span>` : btn;
    },

    chip(text, { tone = 'neutral', icon, mono = false, tip = '' } = {}) {
        return `<span ${tip ? `data-tip="${ui.esc(tip)}"` : ''} class="inline-flex max-w-full items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${ui.tone.chip(tone)} ${mono ? 'font-mono' : ''}">${icon ? `<i data-lucide="${icon}" class="size-3 shrink-0"></i>` : ''}<span class="truncate">${ui.esc(text)}</span></span>`;
    },

    count(n, tone = 'neutral') {
        return `<span class="${ui.cls.count} ${ui.tone.chip(tone)}">${n}</span>`;
    },

    dot(tone = 'overlay0', { pulse = false } = {}) {
        return `<span class="inline-block size-2 shrink-0 rounded-full bg-${tone} ${pulse ? 'animate-pulse' : ''}"></span>`;
    },

    status(label, tone = 'neutral', { pulse = false } = {}) {
        return `<span class="inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-xs font-medium ${ui.tone.chip(tone)}">${ui.dot(tone === 'neutral' ? 'overlay0' : tone, { pulse })}${ui.esc(label)}</span>`;
    },

    kbd(keys) {
        return keys.map((k) => `<kbd class="inline-flex h-5 min-w-5 items-center justify-center rounded-md bg-surface1 px-1 font-mono text-[11px] text-subtext1">${ui.esc(k)}</kbd>`).join('');
    },

    tile(icon, tone = 'mauve', size = 'md') {
        const s = { sm: 'size-8 rounded-xl [&_svg]:size-4', md: 'size-10 rounded-2xl [&_svg]:size-5', lg: 'size-14 rounded-2xl [&_svg]:size-6' }[size];
        return `<span class="flex ${s} shrink-0 items-center justify-center ${ui.tone.chip(tone)}"><i data-lucide="${icon}"></i></span>`;
    },

    spinner(cls = 'size-4') {
        return `<i data-lucide="loader-circle" class="${cls} animate-spin"></i>`;
    },

    meter(pct, tone = 'mauve', h = 'h-2') {
        return `<span class="block ${h} overflow-hidden rounded-full bg-surface0"><span class="block h-full rounded-full bg-${tone} transition-[width] duration-300" style="width:${Math.max(0, Math.min(100, pct))}%"></span></span>`;
    },

    banner({ tone = 'blue', icon = 'info', title = '', body = '', actions = '' }) {
        return `<div class="flex flex-wrap items-start gap-3 rounded-2xl bg-${tone}/10 p-4 text-sm">
            <i data-lucide="${icon}" class="mt-0.5 size-4 shrink-0 text-${tone}"></i>
            <div class="min-w-48 flex-1">${title ? `<p class="font-medium text-${tone}">${ui.esc(title)}</p>` : ''}${body ? `<div class="${title ? 'mt-0.5' : ''} text-subtext0">${body}</div>` : ''}</div>
            ${actions ? `<div class="flex flex-wrap items-center gap-2">${actions}</div>` : ''}
        </div>`;
    },

    field(label, control, { hint = '', group = false } = {}) {
        const tag = group ? 'div' : 'label';
        return `<${tag} ${group ? `role="group" aria-label="${ui.esc(label)}"` : ''} class="flex min-w-0 flex-col gap-1.5"><span class="${ui.cls.label} px-1">${ui.esc(label)}</span>${control}${hint ? `<span class="px-1 text-xs text-overlay1">${hint}</span>` : ''}</${tag}>`;
    },

    input({ name, value = '', placeholder = '', type = 'text', icon, mono = false, attrs = '' }) {
        const base = `<input type="${type}" data-name="${ui.esc(name)}" value="${ui.esc(value)}" placeholder="${ui.esc(placeholder)}" spellcheck="false" autocomplete="off" class="${ui.cls.input} ${icon ? 'pl-10' : ''} ${mono ? 'font-mono' : ''}" ${attrs}>`;
        if (!icon) return base;
        return `<span class="relative flex"><i data-lucide="${icon}" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i>${base}</span>`;
    },

    search({ name, placeholder = 'Search', value = '', shortcut, attrs = '' }) {
        return `<span class="relative flex"><i data-lucide="search" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i><input type="search" data-name="${ui.esc(name)}" value="${ui.esc(value)}" placeholder="${ui.esc(placeholder)}" spellcheck="false" autocomplete="off" class="w-full rounded-full bg-surface0 py-2.5 pr-4 pl-10 text-sm text-text placeholder:text-overlay0 outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px] [&::-webkit-search-cancel-button]:appearance-none" ${attrs}>${shortcut ? `<span class="pointer-events-none absolute top-1/2 right-3 flex -translate-y-1/2 gap-1 max-md:hidden">${ui.kbd(shortcut)}</span>` : ''}</span>`;
    },

    textarea({ name, value = '', placeholder = '', rows = 4, mono = false, attrs = '' }) {
        return `<span class="relative flex min-w-0">
            <textarea data-name="${ui.esc(name)}" data-scroll rows="${rows}" spellcheck="false" placeholder="${ui.esc(placeholder)}" class="${ui.cls.input} min-h-20 resize-none pr-6 leading-relaxed ${mono ? 'font-mono' : ''}" ${attrs}>${ui.esc(value)}</textarea>
            <span data-ui="grip" aria-hidden="true" class="absolute right-0.5 bottom-0.5 flex size-6 cursor-ns-resize touch-none items-center justify-center text-overlay0 transition-colors hover:text-overlay2"><svg viewBox="0 0 12 12" class="size-3" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M10.5 1.5 1.5 10.5M10.5 6.5l-4 4"/></svg></span>
        </span>`;
    },

    invalid(input, message = '') {
        const field = input.closest('label, [role="group"]');
        field.querySelector('[data-error]')?.remove();
        input.setAttribute('aria-invalid', String(Boolean(message)));
        if (message) field.insertAdjacentHTML('beforeend', `<span data-error class="px-1 text-xs text-red">${ui.esc(message)}</span>`);
    },

    checkbox({ name, label, hint = '', checked = false, disabled = false }) {
        return `<button type="button" role="checkbox" data-ui="checkbox" data-name="${ui.esc(name)}" aria-checked="${checked}" ${disabled ? 'disabled' : ''} class="group flex w-full items-start gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0/50 disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">
            <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-lg bg-surface1 text-crust transition-colors group-aria-checked:bg-mauve"><i data-lucide="check" class="size-3.5 scale-50 opacity-0 transition-all group-aria-checked:scale-100 group-aria-checked:opacity-100" stroke-width="3"></i></span>
            <span class="min-w-0"><span class="block text-sm text-text">${ui.esc(label)}</span>${hint ? `<span class="block text-xs text-overlay1">${ui.esc(hint)}</span>` : ''}</span>
        </button>`;
    },

    switch({ name, label, hint = '', checked = false, disabled = false }) {
        return `<button type="button" role="switch" data-ui="switch" data-name="${ui.esc(name)}" aria-checked="${checked}" ${disabled ? 'disabled' : ''} class="group flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0/50 disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">
            <span class="min-w-0 flex-1"><span class="block text-sm text-text">${ui.esc(label)}</span>${hint ? `<span class="block text-xs text-overlay1">${ui.esc(hint)}</span>` : ''}</span>
            <span class="flex h-6 w-11 shrink-0 items-center rounded-full bg-surface1 p-0.5 transition-colors group-aria-checked:bg-mauve"><span class="size-5 rounded-full bg-text transition-transform group-aria-checked:translate-x-5 group-aria-checked:bg-crust"></span></span>
        </button>`;
    },

    toggle({ name, checked = false, label, disabled = false }) {
        return `<button type="button" role="switch" data-ui="switch" data-name="${ui.esc(name)}" aria-checked="${checked}" aria-label="${ui.esc(label)}" ${disabled ? 'disabled' : ''} class="group flex h-6 w-11 shrink-0 items-center rounded-full bg-surface1 p-0.5 transition-colors aria-checked:bg-mauve disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}"><span class="size-5 rounded-full bg-text transition-transform group-aria-checked:translate-x-5 group-aria-checked:bg-crust"></span></button>`;
    },

    segmented({ name, value, options, full = false, ground = false }) {
        const fill = ui.tracks[ground ? 'ground' : 'panel'];
        return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="${full ? 'flex w-full' : 'inline-flex max-w-full overflow-x-auto'} items-center gap-1 rounded-full ${fill.track} p-1">
            ${options.map((o) => {
                const on = o.value === value;
                return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="group flex ${full ? 'flex-1' : ''} min-w-0 items-center justify-center gap-2 whitespace-nowrap rounded-full px-3.5 py-1.5 text-sm font-medium text-overlay1 transition-colors hover:text-text ${fill.on} aria-checked:text-text disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">${o.icon ? `<i data-lucide="${o.icon}" class="size-4 shrink-0 group-aria-checked:text-mauve"></i>` : ''}<span class="truncate ${o.icon && o.compact ? 'max-sm:hidden' : ''}">${ui.esc(o.label)}</span>${o.count !== undefined ? `<span class="${ui.cls.count} ${fill.count} text-subtext0 group-aria-checked:bg-mauve/15 group-aria-checked:text-mauve">${o.count}</span>` : ''}</button>`;
            }).join('')}
        </span>`;
    },

    choices({ name, value, options, columns = 1 }) {
        const grid = { 1: 'grid-cols-1', 2: 'grid-cols-1 sm:grid-cols-2', 3: 'grid-cols-1 sm:grid-cols-3' }[columns];
        return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="grid ${grid} gap-2">
            ${options.map((o) => {
                const on = o.value === value;
                return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="group flex items-start gap-3 ${ui.cls.tile} p-3 text-left transition-colors hover:bg-surface0 aria-checked:bg-mauve/10 disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">
                    <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-surface1 transition-colors group-aria-checked:bg-mauve"><span class="size-2 scale-0 rounded-full bg-crust transition-transform group-aria-checked:scale-100"></span></span>
                    <span class="min-w-0"><span class="flex items-center gap-2 text-sm font-medium text-text">${o.icon ? `<i data-lucide="${o.icon}" class="size-4 text-overlay1 group-aria-checked:text-mauve"></i>` : ''}${ui.esc(o.label)}</span>${o.hint ? `<span class="mt-0.5 block text-xs text-overlay1">${ui.esc(o.hint)}</span>` : ''}</span>
                </button>`;
            }).join('')}
        </span>`;
    },

    select({ name, value = '', options, placeholder = 'Select', searchable = false, icon, pill = false, attrs = '' }) {
        const current = options.find((o) => o.value === value);
        ui.selectOptions[name] = { options, searchable };
        const lead = current?.dot ? ui.dot(current.dot) : current?.icon ? `<i data-lucide="${current.icon}" class="size-4 shrink-0 text-overlay1"></i>` : icon ? `<i data-lucide="${icon}" class="size-4 shrink-0 text-overlay1"></i>` : '';
        return `<button type="button" data-ui="select" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" aria-haspopup="listbox" aria-expanded="false" class="flex ${pill ? 'h-10 w-auto max-w-full rounded-full' : 'w-full rounded-2xl'} items-center gap-2.5 bg-surface0 px-4 py-2.5 text-left text-sm transition-colors hover:bg-surface1 aria-expanded:bg-surface1 disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus} pointer-coarse:text-[16px]" ${attrs}>
            <span data-select-lead class="contents">${lead}</span>
            <span data-select-label class="min-w-0 flex-1 truncate ${current ? 'text-text' : 'text-overlay0'}">${ui.esc(current ? current.label : placeholder)}</span>
            <i data-lucide="chevrons-up-down" class="size-4 shrink-0 text-overlay1"></i>
        </button>`;
    },

    openSelect(button) {
        const name = button.dataset.name;
        const { options, searchable } = ui.selectOptions[name] || { options: [] };
        let query = '';
        let active = Math.max(0, options.findIndex((o) => o.value === button.dataset.value));
        button.setAttribute('aria-expanded', 'true');

        const filtered = () => (query ? options.map((o) => ({ o, s: ui.fuzzy(query, `${o.label} ${o.hint || ''}`) })).filter((x) => x.s >= 0).sort((a, b) => b.s - a.s).map((x) => x.o) : options);
        const lead = (o) => (o.dot ? ui.dot(o.dot) : o.icon ? `<i data-lucide="${o.icon}" class="size-4 shrink-0 text-overlay1"></i>` : '');

        const listHtml = () => {
            const items = filtered();
            if (!items.length) return '<li class="px-3 py-6 text-center text-sm text-overlay1">No matches</li>';
            return items.map((o, i) => {
                const selected = o.value === button.dataset.value;
                return `<li role="option" aria-selected="${selected}" data-index="${i}" data-value="${ui.esc(o.value)}" class="flex items-center gap-3 rounded-xl px-3 py-2 text-sm ${i === active ? 'bg-surface0 text-text' : 'text-subtext0'}">
                    ${lead(o)}
                    <span class="min-w-0 flex-1"><span class="block truncate">${ui.esc(o.label)}</span>${o.hint ? `<span class="block truncate text-xs text-overlay1">${ui.esc(o.hint)}</span>` : ''}</span>
                    ${selected ? '<i data-lucide="check" class="size-4 shrink-0 text-mauve"></i>' : ''}
                </li>`;
            }).join('');
        };

        const pop = ui.popover.open(button, `
            ${searchable ? `<span class="mb-1 block">${ui.search({ name: `${name}-search` })}</span>` : ''}
            <ul role="listbox" data-scroll="lane" class="-mr-1.5 max-h-72 overflow-y-auto">${listHtml()}</ul>`, {
            matchWidth: true,
            onClose: () => button.setAttribute('aria-expanded', 'false'),
        });
        const list = pop.querySelector('ul');
        const search = pop.querySelector('input');
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
            label.classList.replace('text-overlay0', 'text-text');
            const l = button.querySelector('[data-select-lead]');
            if (opt.dot || opt.icon) { l.innerHTML = lead(opt); ui.icons(l); }
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
            search.addEventListener('input', (e) => {
                e.stopPropagation();
                query = search.value;
                active = 0;
                redraw();
            });
            search.focus();
        } else {
            list.tabIndex = -1;
            list.focus({ preventScroll: true });
        }
    },

    layer() {
        return document.fullscreenElement || document.getElementById('layer-root');
    },

    scheme(el) {
        const scope = el.closest('.dark, .light');
        return scope && scope !== document.documentElement ? (scope.classList.contains('light') ? 'light' : 'dark') : '';
    },

    popover: {
        current: null,

        open(anchor, html, { matchWidth = false, align = 'start', onClose, width } = {}) {
            ui.popover.close();
            ui.tooltip.hide();
            const el = document.createElement('div');
            el.className = `fixed z-[70] rounded-2xl bg-mantle p-1.5 shadow-2xl shadow-shade/80 ring-1 ring-surface0 [animation:pop-in_.12s_ease-out] ${ui.scheme(anchor)}`;
            el.innerHTML = html;
            ui.layer().appendChild(el);
            const r = anchor.getBoundingClientRect();
            const w = width || (matchWidth ? Math.max(r.width, 220) : el.offsetWidth);
            el.style.width = `${Math.min(w, innerWidth - 16)}px`;
            const place = () => {
                const left = align === 'end' ? r.right - el.offsetWidth : r.left;
                el.style.left = `${Math.max(8, Math.min(left, innerWidth - el.offsetWidth - 8))}px`;
                const below = r.bottom + 6;
                el.style.top = below + el.offsetHeight > innerHeight - 8 && r.top - el.offsetHeight - 6 > 8 ? `${r.top - el.offsetHeight - 6}px` : `${below}px`;
            };
            ui.icons(el);
            place();
            new ResizeObserver(place).observe(el);
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

    tooltip: {
        el: null,
        timer: null,
        warmUntil: 0,
        target: null,
        pointer: { x: 0, y: 0 },

        init() {
            document.addEventListener('pointermove', (e) => {
                ui.tooltip.pointer = { x: e.clientX, y: e.clientY };
            }, { passive: true });
            document.addEventListener('pointerover', (e) => {
                if (e.pointerType !== 'mouse') return;
                const t = e.target.closest('[data-tip]');
                if (t === ui.tooltip.target) return;
                if (!t) { ui.tooltip.hide(); return; }
                ui.tooltip.schedule(t);
            });
            document.addEventListener('pointerout', (e) => {
                const t = ui.tooltip.target;
                if (t && !t.contains(e.relatedTarget)) ui.tooltip.hide();
            });
            document.addEventListener('focusin', (e) => {
                const t = e.target.closest?.('[data-tip]');
                if (t && e.target.matches(':focus-visible')) ui.tooltip.show(t);
            });
            document.addEventListener('focusout', () => ui.tooltip.hide());
            document.addEventListener('pointerdown', () => ui.tooltip.hide(), true);
            document.addEventListener('scroll', () => ui.tooltip.hide(), true);
        },

        schedule(t) {
            clearTimeout(ui.tooltip.timer);
            ui.tooltip.target = t;
            const delay = Date.now() < ui.tooltip.warmUntil ? 0 : 450;
            ui.tooltip.timer = setTimeout(() => ui.tooltip.show(t), delay);
        },

        show(t) {
            clearTimeout(ui.tooltip.timer);
            if (ui.tooltip.el) ui.tooltip.el.remove();
            if (ui.popover.current || !t.dataset.tip) return;
            if (t.hasAttribute('data-tip-collapse') && t.querySelector('[data-label]')?.checkVisibility()) return;
            ui.tooltip.target = t;
            const el = document.createElement('div');
            el.setAttribute('role', 'tooltip');
            el.className = `pointer-events-none fixed z-[90] max-w-64 rounded-xl bg-surface0 px-2.5 py-1.5 text-xs text-text shadow-lg shadow-shade/60 [animation:fade-in_.1s_ease-out] ${ui.scheme(t)}`;
            el.textContent = t.dataset.tip;
            ui.layer().appendChild(el);
            ui.tooltip.el = el;
            const r = t.hasAttribute('data-tip-pointer') ? new DOMRect(ui.tooltip.pointer.x, ui.tooltip.pointer.y, 0, 0) : t.getBoundingClientRect();
            // The Tailwind browser build compiles a first-seen class after insertion, so a synchronous measure reads an unstyled, full-width box.
            requestAnimationFrame(() => {
                const side = t.dataset.tipSide || 'top';
                const gap = 8;
                let top;
                let left;
                if (side === 'right' || side === 'left') {
                    top = r.top + r.height / 2 - el.offsetHeight / 2;
                    left = side === 'right' ? r.right + gap : r.left - el.offsetWidth - gap;
                } else {
                    left = r.left + r.width / 2 - el.offsetWidth / 2;
                    top = side === 'bottom' || r.top - el.offsetHeight - gap < 8 ? r.bottom + gap : r.top - el.offsetHeight - gap;
                }
                el.style.left = `${Math.max(8, Math.min(left, innerWidth - el.offsetWidth - 8))}px`;
                el.style.top = `${Math.max(8, top)}px`;
            });
        },

        hide() {
            clearTimeout(ui.tooltip.timer);
            if (ui.tooltip.el) {
                ui.tooltip.el.remove();
                ui.tooltip.el = null;
                ui.tooltip.warmUntil = Date.now() + 400;
            }
            ui.tooltip.target = null;
        },
    },

    menu(anchor, items) {
        anchor.setAttribute('aria-expanded', 'true');
        const el = ui.popover.open(anchor, `<div data-scroll class="max-h-[min(28rem,70dvh)] overflow-y-auto">${items.map((item, i) => (item === '-' ? '<span class="my-1 block h-px"></span>' : `
            <button type="button" data-menu-index="${i}" ${item.disabled ? 'disabled' : ''} class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus} ${item.danger ? 'text-red hover:bg-red/15' : 'text-subtext0 hover:bg-surface0 hover:text-text'}">
                <i data-lucide="${item.icon}" class="size-4 shrink-0"></i><span class="min-w-0 flex-1"><span class="block">${ui.esc(item.label)}</span>${item.hint ? `<span class="block text-xs text-overlay1">${ui.esc(item.hint)}</span>` : ''}</span>${item.keys ? `<span class="flex gap-1">${ui.kbd(item.keys)}</span>` : ''}
            </button>`)).join('')}</div>`, { align: 'end', width: 260, onClose: () => anchor.setAttribute('aria-expanded', 'false') });
        el.addEventListener('click', (e) => {
            const b = e.target.closest('[data-menu-index]');
            if (!b) return;
            ui.popover.close();
            items[Number(b.dataset.menuIndex)].onSelect?.();
        });
    },

    modal: {
        stack: [],

        open({ title, icon, tone = 'mauve', subtitle = '', body = '', footer = '', size = 'md', dismissible = true, onClose }) {
            ui.tooltip.hide();
            const sizes = { sm: 'md:max-w-md', md: 'md:max-w-2xl', lg: 'md:max-w-4xl', xl: 'md:max-w-6xl 3xl:max-w-[100rem]' };
            const root = document.createElement('div');
            root.className = 'fixed inset-0 z-50 flex items-end justify-center bg-crust/70 backdrop-blur-sm md:items-center md:p-6 [animation:fade-in_.15s_ease-out]';
            root.innerHTML = `
                <section role="dialog" aria-modal="true" class="flex max-h-[92dvh] w-full flex-col rounded-t-3xl bg-mantle shadow-2xl shadow-shade md:max-h-[86dvh] md:rounded-3xl ${sizes[size]} max-md:[animation:sheet-in_.2s_ease-out] md:[animation:pop-in_.15s_ease-out]">
                    <header class="flex shrink-0 items-center gap-3 px-5 pt-5 pb-3 md:px-6 md:pt-6">
                        ${icon ? `<span class="flex size-10 shrink-0 items-center justify-center rounded-full ${ui.tone.chip(tone)}"><i data-lucide="${icon}" class="size-5"></i></span>` : ''}
                        <div class="min-w-0 flex-1"><h2 class="truncate font-display text-lg font-medium text-text">${ui.esc(title)}</h2>${subtitle ? `<p class="truncate text-sm text-overlay1">${subtitle}</p>` : ''}</div>
                        ${dismissible ? ui.iconButton('x', { tip: 'Close', attrs: 'data-modal-close' }) : ''}
                    </header>
                    <div data-modal-body data-scroll class="min-h-0 flex-1 overflow-y-auto px-5 pb-4 md:px-6">${body}</div>
                    ${footer ? `<footer data-modal-footer class="flex shrink-0 flex-wrap items-center justify-end gap-2 px-5 py-4 md:px-6">${footer}</footer>` : ''}
                </section>`;
            ui.layer().appendChild(root);
            const entry = {
                el: root.firstElementChild,
                dismissible,
                close() {
                    if (!root.isConnected) return;
                    ui.popover.close();
                    ui.tooltip.hide();
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
            root.querySelector('[autofocus]')?.focus();
            return entry;
        },
    },

    confirm({ title, message, confirmLabel = 'Confirm', danger = false, icon = 'circle-help' }) {
        return new Promise((resolve) => {
            let answered = false;
            const m = ui.modal.open({
                title,
                icon,
                tone: danger ? 'red' : 'mauve',
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

    toast(message, { tone = 'neutral', icon = 'info', timeout = 3500, action } = {}) {
        let host = document.getElementById('toasts');
        if (!host) {
            host = document.createElement('div');
            host.id = 'toasts';
            host.className = 'pointer-events-none fixed inset-x-3 bottom-3 z-[80] flex flex-col items-center gap-2 md:inset-x-auto md:right-5 md:bottom-5 md:items-end';
        }
        if (host.parentElement !== ui.layer()) ui.layer().appendChild(host);
        const el = document.createElement('div');
        el.className = `pointer-events-auto flex min-h-11 max-w-md items-center gap-3 rounded-full bg-surface0 py-2 pl-4 ${action ? 'pr-2' : 'pr-5'} text-sm text-text shadow-xl shadow-shade/70 [animation:toast-in_.18s_ease-out]`;
        el.innerHTML = `<i data-lucide="${icon}" class="size-4 shrink-0 ${ui.tone.text(tone)}"></i><span class="min-w-0">${ui.esc(message)}</span>${action ? `<button type="button" class="ml-1 h-7 shrink-0 rounded-full px-3 text-xs font-medium text-mauve transition-colors hover:bg-surface1">${ui.esc(action.label)}</button>` : ''}`;
        host.appendChild(el);
        ui.icons(el);
        const dismiss = () => {
            el.style.transition = 'opacity .25s, transform .25s';
            el.style.opacity = '0';
            el.style.transform = 'translateY(8px)';
            setTimeout(() => el.remove(), 260);
        };
        el.querySelector('button')?.addEventListener('click', () => {
            action.onClick?.();
            dismiss();
        });
        setTimeout(dismiss, timeout);
    },

    empty({ icon, title, hint = '', action = '', fill = true }) {
        return `<div class="flex ${fill ? 'h-full' : ''} min-h-40 flex-col items-center justify-center gap-3 p-6 text-center">
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
        if (ts == null) return 'never';
        const s = Math.max(0, Math.round((Date.now() - ts) / 1000));
        if (s < 60) return 'just now';
        const units = [['d', 86400], ['h', 3600], ['m', 60]];
        for (const [u, n] of units) if (s >= n) return `${Math.floor(s / n)}${u} ago`;
        return 'just now';
    },

    async copy(text, button) {
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            const ta = Object.assign(document.createElement('textarea'), { value: text });
            ta.style.cssText = 'position:fixed;opacity:0';
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            ta.remove();
        }
        if (!button) {
            ui.toast('Copied to clipboard', { tone: 'green', icon: 'clipboard-check', timeout: 1800 });
            return;
        }
        button.dataset.tipIdle ??= button.dataset.tip ?? '';
        button.innerHTML = '<i data-lucide="check" class="size-4 text-green"></i>';
        button.dataset.tip = 'Copied';
        if (ui.tooltip.target === button) ui.tooltip.show(button);
        ui.icons(button);
        clearTimeout(button._copied);
        button._copied = setTimeout(() => {
            button.innerHTML = '<i data-lucide="copy" class="size-4"></i>';
            button.dataset.tip = button.dataset.tipIdle;
            if (ui.tooltip.target === button) ui.tooltip.show(button);
            ui.icons(button);
        }, 1500);
    },

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
                Math.max(6, corner, wide ? ui.scrollLane : 0),
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

    media: {
        mobile: () => matchMedia('(max-width: 767px)').matches,
        wide: () => matchMedia('(min-width: 1920px)').matches,
    },
};
