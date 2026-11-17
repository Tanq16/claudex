# Kit Core

**One `ui` object in `static/ui.js` holds the machinery and the primitives every view calls, so a fill, a radius, or a focus ring decided once lands identically everywhere.**

## The Object

`ui.js` is a plain script loaded before `app.js`, defining one `ui` object whose builders return HTML strings. A view composes those strings in a template literal, so a control's markup lives in one place instead of being restated per view, which is where two shades of the same control come from.

Each `ui` block below is a member of the object opened here, and the other references add their builders to the same object.

```javascript
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
```

Every interpolated value passes through `ui.esc`, because a builder returns markup and an unescaped name or path becomes an injection. A field `hint`, a banner `body`, a modal `subtitle`, and a confirm `message` are inserted as HTML so they can carry markup, so the caller escapes any user value inside one.

`ui.icons(root)` runs after every insertion, since Lucide replaces each `<i data-lucide>` once and never observes the DOM.

`ui.cls` holds every class string more than one builder shares. `ui.tone.chip` turns a tone into its `/15` tint and its text color, with `neutral` falling back to `surface0`.

```javascript
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
```

## Buttons

```javascript
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
    const labelHtml = collapse ? `<span class="max-sm:hidden">${ui.esc(label)}</span>` : `<span>${ui.esc(label)}</span>`;
    const tipText = tip || (collapse ? label : '');
    const btn = `<button type="${type}" ${tipText && !disabled ? `data-tip="${ui.esc(tipText)}"` : ''} class="inline-flex shrink-0 items-center justify-center rounded-full font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus} ${collapse ? 'max-sm:w-10 max-sm:px-0' : ''} ${variants[variant]} ${sizes[size]}" ${action ? `data-action="${action}"` : ''} ${disabled ? 'disabled' : ''} ${attrs}>${icon ? i(icon, spin ? 'animate-spin' : '') : ''}${labelHtml}${iconRight ? i(iconRight, '-mr-1 opacity-70') : ''}</button>`;
    return disabled && tipText ? `<span data-tip="${ui.esc(tipText)}" class="inline-flex">${btn}</span>` : btn;
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
```

`primary` is the one action a view exists for, and `secondary` is every other filled button. `ghost` sits in toolbars and footers, `danger` is the tinted destructive action, and `danger-solid` is kept for a removal the user has typed out to confirm.

An icon-only button carries its name as both `aria-label` and `data-tip`. A disabled button's tooltip moves to a wrapping `<span>`, since a disabled button fires no pointer events.

`collapse` drops the label below `sm` and keeps it as the tooltip. `pressed` renders `aria-pressed` for a toggle such as a pin. `inset` is the icon button inside a `surface0` control, hovering one rung further at `surface1`.

The examples in every reference build their rows with two layout helpers.

```javascript
const row = (inner, cls = '') => `<div class="flex flex-wrap items-center gap-2 ${cls}">${inner}</div>`;
const sub = (t) => `<p class="${ui.cls.label} px-1">${t}</p>`;

const buttons = `
    ${sub('Variants')}${row(['primary', 'secondary', 'ghost', 'danger', 'success'].map((v) => ui.button(v[0].toUpperCase() + v.slice(1), { variant: v })).join(''))}
    ${sub('With icons and sizes')}${row(`${ui.button('New session', { icon: 'plus', variant: 'primary', size: 'lg' })}${ui.button('Restart', { icon: 'rotate-cw' })}${ui.button('Stop', { icon: 'square', variant: 'danger', size: 'sm' })}${ui.button('Open', { iconRight: 'arrow-up-right', variant: 'ghost' })}`)}
    ${sub('States')}${row(`${ui.button('Saving', { icon: 'loader-circle', spin: true, disabled: true, variant: 'primary' })}${ui.button('Deploy', { icon: 'rocket', disabled: true, tip: 'Connect a target first' })}${ui.button('Export', { icon: 'download', collapse: true })}`)}
    ${sub('Icon buttons')}${row(`${ui.iconButton('copy', { tip: 'Copy' })}${ui.iconButton('pencil', { tip: 'Rename', variant: 'secondary' })}${ui.iconButton('plus', { tip: 'Add', variant: 'primary' })}${ui.iconButton('trash-2', { tip: 'Delete', variant: 'danger' })}${ui.iconButton('pin', { tip: 'Pin', pressed: true, attrs: 'data-action="press"' })}${ui.iconButton('refresh-cw', { tip: 'Refreshing', spin: true, size: 'sm' })}`)}`;

ui.on('press', (b) => b.setAttribute('aria-pressed', String(b.getAttribute('aria-pressed') !== 'true')));
```

## Chips and Markers

```javascript
chip(text, { tone = 'neutral', icon, mono = false, tip = '', remove } = {}) {
    return `<span ${tip ? `data-tip="${ui.esc(tip)}"` : ''} class="inline-flex max-w-full items-center gap-1.5 rounded-full ${remove ? 'pr-1' : 'pr-2.5'} pl-2.5 py-1 text-xs font-medium ${ui.tone.chip(tone)} ${mono ? 'font-mono' : ''}">${icon ? `<i data-lucide="${icon}" class="size-3 shrink-0"></i>` : ''}<span class="truncate">${ui.esc(text)}</span>${remove ? `<button type="button" aria-label="Remove ${ui.esc(text)}" data-action="${remove}" data-value="${ui.esc(text)}" class="-my-0.5 grid size-5 place-items-center rounded-full opacity-70 hover:bg-current/15 hover:opacity-100"><i data-lucide="x" class="size-3"></i></button>` : ''}</span>`;
},

count(n, tone = 'neutral') {
    return `<span class="${ui.cls.count} ${ui.tone.chip(tone)}">${n}</span>`;
},

dot(tone = 'overlay0', { pulse = false } = {}) {
    return `<span class="size-2 shrink-0 rounded-full bg-${tone} ${pulse ? 'animate-pulse' : ''}"></span>`;
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

avatar(name, tone = 'mauve', size = 'size-8 text-sm') {
    return `<span class="inline-flex ${size} shrink-0 items-center justify-center rounded-full font-display font-semibold ${ui.tone.chip(tone)}">${ui.esc(String(name).charAt(0).toUpperCase())}</span>`;
},

spinner(cls = 'size-4') {
    return `<i data-lucide="loader-circle" class="${cls} animate-spin"></i>`;
},
```

A chip, a count, and a status take their fill from `ui.tone.chip`. A status adds a dot that pulses for a live state such as running or pending. A chip's `remove` names the action of an inline remove button, which the tag field uses. The button hovers at `current/15`, a tint of the chip's own text, so the hover shows on any chip fill.

A `kbd` sits inside a `surface0` control such as the search field or a menu, so it takes `surface1`.

A tile is an icon on its tone's tint, `sm` in a panel header and `md` in a stat, a card, or a detail header. An avatar is a name's initial on its tone's tint.

## Fields

```javascript
field(label, control, { hint = '', group = false } = {}) {
    const tag = group ? 'div' : 'label';
    return `<${tag} ${group ? `role="group" aria-label="${ui.esc(label)}"` : ''} class="flex min-w-0 flex-col gap-1.5"><span class="${ui.cls.label} px-1">${ui.esc(label)}</span>${control}${hint ? `<span class="px-1 text-xs text-overlay1">${hint}</span>` : ''}</${tag}>`;
},
```

A field stacks a label above its control and a hint below it. It is a `<label>`, so a click on its text activates the first button or input inside. For an input, a textarea, a number field, a select, or a date picker, that is the control itself.

Any other control passes `group: true` and renders a `role="group"` `div` instead. In a pill track, a stepper, or a tag field, the first button is one pill, a decrease, or a chip's remove. A `<label>` around one turns a click on its text into a selection or a deletion.

## Delegation

```javascript
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
    } else if (kind === 'date') {
        e.preventDefault();
        ui.openDate(control);
    } else if (kind === 'step') {
        const input = control.closest('[data-number]').querySelector('input');
        if (Number(control.dataset.step) > 0) input.stepUp(); else input.stepDown();
        input.dispatchEvent(new Event('input', { bubbles: true }));
    } else if (kind === 'reveal') {
        const input = control.closest('.relative').querySelector('input');
        const show = input.type === 'password';
        input.type = show ? 'text' : 'password';
        control.innerHTML = `<i data-lucide="${show ? 'eye-off' : 'eye'}" class="size-4"></i>`;
        control.dataset.tip = show ? 'Hide' : 'Show';
        ui.icons(control);
    }
},

emit(el, value) {
    el.dispatchEvent(new CustomEvent('ui-change', { bubbles: true, detail: { name: el.dataset.name, value } }));
},
```

Behavior is wired once, by delegation on `document`. A control carries `data-ui` for a built-in behavior such as a radio, a switch, a select, or a stepper, and `data-action` for a handler registered with `ui.on`. A view re-renders by replacing `innerHTML`, which drops every per-element listener, while a delegated one never needs binding again.

A control reports a change as a bubbling `ui-change` event carrying `{ name, value }`, so a view listens once on its root for every control inside it. A text field, a textarea, or a number field reports on every keystroke through the delegated `input` listener. A stepper button reaches that listener by dispatching `input`.

```javascript
el.addEventListener('ui-change', (e) => { if (e.detail.name === 'media-tab') show(e.detail.value); });
```

Escape closes the innermost layer first: the tooltip, then a popover, then the top dismissible modal. The arrow keys move the selection within a radio group and select as they move, as a native radio group does.

The `select`, `date`, `step`, `reveal`, and grip branches drive builders from controls.md and selects-menus.md. They stay inert on a page that has none of those controls.

## Floating Layers

```javascript
layer() {
    return document.fullscreenElement || document.getElementById('layer-root');
},

scheme(el) {
    const scope = el.closest('.dark, .light');
    return scope && scope !== document.documentElement ? (scope.classList.contains('light') ? 'light' : 'dark') : '';
},
```

Popovers, tooltips, modals, and toasts are appended to `ui.layer()`, which returns `document.fullscreenElement` when there is one and `#layer-root` otherwise. A layer appended anywhere else is hidden behind a fullscreen element.

A layer copies the `.dark` or `.light` scope of the element that opened it, through `ui.scheme`. The layer root sits outside that subtree, so a menu opened from a pinned-dark player would otherwise wear the page theme.

```javascript
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
```

One popover is open at a time. It closes on an outside `pointerdown` or Escape, and it flips above its anchor when the space below runs out. `matchWidth` sizes it to the trigger with a 220px floor, and `align: 'end'` lines its right edge up with the trigger's.

```javascript
tooltip: {
    el: null,
    timer: null,
    warmUntil: 0,
    target: null,

    init() {
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
        ui.tooltip.target = t;
        const el = document.createElement('div');
        el.setAttribute('role', 'tooltip');
        el.className = `pointer-events-none fixed z-[90] max-w-64 rounded-xl bg-surface0 px-2.5 py-1.5 text-xs text-text shadow-lg shadow-shade/60 [animation:fade-in_.1s_ease-out] ${ui.scheme(t)}`;
        el.textContent = t.dataset.tip;
        ui.layer().appendChild(el);
        const r = t.getBoundingClientRect();
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
        ui.tooltip.el = el;
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
```

A tooltip comes from `data-tip` on any element. On a mouse it shows after 450ms, and for 400ms after one hides the next shows at once, so sweeping along a toolbar does not wait on every button. Keyboard focus shows it at once on `:focus-visible`, and a `pointerdown` or a scroll hides it.

`data-tip-side` picks the side, and a top tooltip flips below its target when there is no room above.

```javascript
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
                    ${icon ? `<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-${tone}/15 text-${tone}"><i data-lucide="${icon}" class="size-5"></i></span>` : ''}
                    <div class="min-w-0 flex-1"><h2 class="truncate font-display text-lg font-medium text-text">${ui.esc(title)}</h2>${subtitle ? `<p class="truncate text-sm text-overlay1">${subtitle}</p>` : ''}</div>
                    ${dismissible ? ui.iconButton('x', { tip: 'Close', attrs: 'data-modal-close' }) : ''}
                </header>
                <div data-modal-body data-scroll class="min-h-0 flex-1 overflow-y-auto px-5 pb-4 md:px-6">${body}</div>
                ${footer ? `<footer class="flex shrink-0 flex-wrap items-center justify-end gap-2 px-5 py-4 md:px-6">${footer}</footer>` : ''}
            </section>`;
        ui.layer().appendChild(root);
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
        root.querySelector('[autofocus]')?.focus();
        return entry;
    },
},
```

A modal is centered from `md` up at `max-w-md`, `max-w-2xl`, `max-w-4xl`, or `max-w-6xl`, and below `md` it is a bottom sheet with `rounded-t-3xl` that slides up. Its header holds an optional tone tile, the title, and a close button, its body is the scroll region, and its footer right-aligns the actions on the modal's own `mantle`.

Modals stack, Escape closes the top dismissible one, and a backdrop click closes it. An `[autofocus]` element in the body takes focus on open.

```javascript
toast(message, { tone = 'neutral', icon = 'info', timeout = 3500, action } = {}) {
    let host = document.getElementById('toasts');
    if (!host) {
        host = document.createElement('div');
        host.id = 'toasts';
        host.className = 'pointer-events-none fixed inset-x-3 bottom-3 z-[80] flex flex-col items-center gap-2 md:inset-x-auto md:right-5 md:bottom-5 md:items-end';
        ui.layer().appendChild(host);
    }
    const el = document.createElement('div');
    el.className = `pointer-events-auto flex max-w-md items-center gap-3 rounded-full bg-surface0 py-2 pl-4 ${action ? 'pr-2' : 'pr-5'} min-h-11 text-sm text-text shadow-xl shadow-shade/70 [animation:toast-in_.18s_ease-out]`;
    el.innerHTML = `<i data-lucide="${icon}" class="size-4 shrink-0 ${ui.tone.text(tone)}"></i><span class="min-w-0">${ui.esc(message)}</span>${action ? `<button type="button" class="ml-1 h-7 shrink-0 rounded-full px-3 text-xs font-medium text-mauve transition-colors hover:bg-surface1">${ui.esc(action.label)}</button>` : ''}`;
    host.appendChild(el);
    ui.icons(el);
    const dismiss = () => {
        el.style.transition = 'opacity .25s, transform .25s';
        el.style.opacity = '0';
        el.style.transform = 'translateY(8px)';
        setTimeout(() => el.remove(), 260);
    };
    el.querySelector('button')?.addEventListener('click', () => { action.onClick?.(); dismiss(); });
    setTimeout(dismiss, timeout);
},
```

A toast sits bottom center on a phone and bottom right from `md`, dismisses itself after 3.5 seconds, and carries at most one action button. Its tone colors only the icon.

## Clipboard

```javascript
async copy(text) {
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
},
```

The `textarea` fallback covers a page served over plain HTTP, where `navigator.clipboard` is unavailable because the context is not secure.

## Fuzzy Search

```javascript
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
```

A substring match ranks above a scattered one, and an earlier substring ranks above a later one. A scattered match scores each character found in order, with a bonus for consecutive runs, and `-1` drops the option.

## Breakpoint in Script

```javascript
media: {
    mobile: () => matchMedia('(max-width: 767px)').matches,
},
```

`ui.media.mobile()` matches below Tailwind's `md` breakpoint of `48rem`, so script and the `max-md:` utilities switch layouts at the same width.
