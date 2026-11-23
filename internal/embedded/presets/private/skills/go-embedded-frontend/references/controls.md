# Controls

**Every form control is a kit builder drawn in the app's own design language, never the browser's native look, and every one reports through `ui-change`.**

A native checkbox, radio, select, or number spinner ignores the palette and the radius, so each control here is a styled button or a bare input inside a kit wrapper.

## Text Inputs

```javascript
input({ name, value = '', placeholder = '', type = 'text', icon, mono = false, attrs = '' }) {
    const base = `<input type="${type}" name="${ui.esc(name)}" data-name="${ui.esc(name)}" value="${ui.esc(value)}" placeholder="${ui.esc(placeholder)}" spellcheck="false" autocomplete="off" class="${ui.cls.input} ${icon ? 'pl-10' : ''} ${type === 'password' ? 'pr-12' : ''} ${mono ? 'font-mono' : ''}" ${attrs}>`;
    if (!icon && type !== 'password') return base;
    return `<span class="relative flex">${icon ? `<i data-lucide="${icon}" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i>` : ''}${base}${type === 'password' ? `<span class="absolute inset-y-0 right-1 flex items-center">${ui.iconButton('eye', { tip: 'Show', size: 'sm', variant: 'inset', attrs: 'data-ui="reveal"' })}</span>` : ''}</span>`;
},

search({ name, placeholder = 'Search', value = '', shortcut, attrs = '' }) {
    return `<span class="relative flex"><i data-lucide="search" class="pointer-events-none absolute top-1/2 left-4 size-4 -translate-y-1/2 text-overlay1"></i><input type="search" name="${ui.esc(name)}" data-name="${ui.esc(name)}" value="${ui.esc(value)}" placeholder="${ui.esc(placeholder)}" spellcheck="false" autocomplete="off" class="w-full rounded-full bg-surface0 py-2.5 pr-4 pl-10 text-sm text-text placeholder:text-overlay0 outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px] [&::-webkit-search-cancel-button]:appearance-none" ${attrs}>${shortcut ? `<span class="pointer-events-none absolute top-1/2 right-3 flex -translate-y-1/2 gap-1 max-md:hidden">${ui.kbd(shortcut)}</span>` : ''}</span>`;
},
```

An input with an `icon` gets a leading icon, and a `password` input gets a reveal button. The reveal is an `inset` icon button that flips the input's type through the delegation's `reveal` branch.

Every text field carries `name` beside `data-name`. Chrome's Issues panel flags a form field with neither a `name` nor an `id`, and the delegation reads only `data-name`.

A search field is `rounded-full`, unlike the `rounded-2xl` inputs, and shows its shortcut as `kbd` keys from `md` up. Its `attrs` land on the `<input>`, as they do for `ui.input`.

An invalid field carries `aria-invalid="true"`, which `ui.cls.input` rings in `red/50`, and puts its message in the field's hint in `text-red`, so the error sits where the eye already is.

```javascript
invalid(input, message = '') {
    const field = input.closest('label, [role="group"]');
    field.querySelector('[data-error]')?.remove();
    input.setAttribute('aria-invalid', String(Boolean(message)));
    if (message) field.insertAdjacentHTML('beforeend', `<span data-error class="px-1 text-xs text-red">${ui.esc(message)}</span>`);
},
```

`ui.invalid` marks a field after render, such as when a submit fails validation, and clears the mark and the message when called without one.

```javascript
const inputs = `<div class="grid gap-3">
    ${ui.field('Name', ui.input({ name: 'name', placeholder: 'orbit-api' }))}
    ${ui.field('Repository', ui.input({ name: 'repo', icon: 'folder-git-2', value: 'tanq16/claudex', mono: true }))}
    ${ui.field('Token', ui.input({ name: 'token', type: 'password', value: 'sk-live-0123456789' }))}
    ${ui.field('Port', ui.input({ name: 'port', value: '99999', attrs: 'aria-invalid="true"' }), { hint: '<span class="text-red">Port must be between 1 and 65535</span>' })}
    ${ui.field('Search', ui.search({ name: 'q', placeholder: 'Filter projects', shortcut: ['⌘', 'K'] }))}
</div>`;
```

## Textarea

```javascript
textarea({ name, value = '', placeholder = '', rows = 4, mono = false }) {
    return `<span class="relative flex min-w-0">
        <textarea name="${ui.esc(name)}" data-name="${ui.esc(name)}" data-scroll rows="${rows}" spellcheck="false" placeholder="${ui.esc(placeholder)}" class="${ui.cls.input} min-h-20 resize-none pr-6 leading-relaxed ${mono ? 'font-mono' : ''}">${ui.esc(value)}</textarea>
        <span data-ui="grip" aria-hidden="true" class="absolute right-0.5 bottom-0.5 flex size-6 cursor-ns-resize touch-none items-center justify-center text-overlay0 transition-colors hover:text-overlay2"><svg viewBox="0 0 12 12" class="size-3" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M10.5 1.5 1.5 10.5M10.5 6.5l-4 4"/></svg></span>
    </span>`;
},
```

A textarea is `resize-none` and carries its own grip, two diagonal strokes in the bottom-right corner that drag its height down to an 80px floor. The native resize handle ignores the palette and the radius. The textarea is a scroll region, and its thumb's track ends above the grip.

```javascript
const textareas = `<div class="grid gap-3">
    ${ui.field('Notes', ui.textarea({ name: 'notes', placeholder: 'Drag the marker to resize', rows: 4 }))}
    ${ui.field('Value', ui.textarea({ name: 'json', mono: true, rows: 6, value: JSON.stringify({ model: 'opus', cleanupPeriodDays: 30 }, null, 2) }))}
</div>`;
```

## Numbers

```javascript
stepper({ name, value = 0, min = 0, max = 99, step = 1 }) {
    const b = (dir, icon) => `<button type="button" data-ui="step" data-step="${dir}" tabindex="-1" aria-label="${dir > 0 ? 'Increase' : 'Decrease'}" class="grid size-8 place-items-center rounded-full text-subtext0 transition-colors hover:bg-surface1 hover:text-text"><i data-lucide="${icon}" class="size-3.5"></i></button>`;
    return `<span data-number class="inline-flex h-10 items-center rounded-full bg-surface0 p-1 focus-within:ring-2 focus-within:ring-mauve/40">${b(-1, 'minus')}<input type="number" name="${ui.esc(name)}" data-name="${ui.esc(name)}" value="${value}" min="${min}" max="${max}" step="${step}" class="w-10 bg-transparent text-center font-mono text-sm text-text tabular-nums outline-none pointer-coarse:text-[16px]">${b(1, 'plus')}</span>`;
},

number({ name, value = 0, min, max, step = 1, unit = '' }) {
    return `<span data-number class="flex h-10 items-center rounded-2xl bg-surface0 pr-1 focus-within:ring-2 focus-within:ring-mauve/40">
        <input type="number" name="${ui.esc(name)}" data-name="${ui.esc(name)}" value="${value}" ${min !== undefined ? `min="${min}"` : ''} ${max !== undefined ? `max="${max}"` : ''} step="${step}" class="w-full min-w-0 bg-transparent pl-4 text-sm text-text tabular-nums outline-none pointer-coarse:text-[16px]">
        ${unit ? `<span class="pr-2 pl-1 text-xs text-overlay1">${ui.esc(unit)}</span>` : ''}
        <span class="flex h-8 flex-col">${[[1, 'chevron-up'], [-1, 'chevron-down']].map(([d, ic]) => `<button type="button" data-ui="step" data-step="${d}" tabindex="-1" aria-label="${d > 0 ? 'Increase' : 'Decrease'}" class="grid w-7 flex-1 place-items-center rounded-lg text-overlay1 hover:bg-surface1 hover:text-text"><i data-lucide="${ic}" class="size-3.5"></i></button>`).join('')}</span>
    </span>`;
},
```

A stepper is a pill for a small integer range, and a number field is an input with an optional unit and stacked chevrons for an open range. Their buttons are `tabindex="-1"`, so Tab moves between fields rather than through every button. A step goes through `stepUp` and `stepDown`, which honor the input's `min`, `max`, and `step`.

```javascript
const numbers = `<div class="grid gap-4 sm:grid-cols-2">
    ${ui.field('Connections', `<span>${ui.stepper({ name: 'conn', value: 4, min: 1, max: 16 })}</span>`, { group: true })}
    ${ui.field('Timeout', ui.number({ name: 'timeout', value: 30, min: 1, unit: 'seconds' }))}
</div>`;
```

## Checkbox, Switch, and Radio

```javascript
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

toggle({ name, checked = false, label }) {
    return `<button type="button" role="switch" data-ui="switch" data-name="${ui.esc(name)}" aria-checked="${checked}" aria-label="${ui.esc(label)}" class="group flex h-6 w-11 shrink-0 items-center rounded-full bg-surface1 p-0.5 transition-colors aria-checked:bg-mauve ${ui.cls.focus}"><span class="size-5 rounded-full bg-text transition-transform group-aria-checked:translate-x-5 group-aria-checked:bg-crust"></span></button>`;
},

radios({ name, value, options }) {
    return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="flex flex-col">
        ${options.map((o) => {
            const on = o.value === value;
            return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" class="group flex items-start gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-surface0/50 ${ui.cls.focus}">
                <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-surface1 transition-colors group-aria-checked:bg-mauve"><span class="size-2 scale-0 rounded-full bg-crust transition-transform group-aria-checked:scale-100"></span></span>
                <span class="min-w-0"><span class="block text-sm text-text">${ui.esc(o.label)}</span>${o.hint ? `<span class="block text-xs text-overlay1">${ui.esc(o.hint)}</span>` : ''}</span>
            </button>`;
        }).join('')}
    </span>`;
},
```

Each is a `<button>` with a `role` and `aria-checked`, styled through `group-aria-checked:`. The delegation flips the attribute and emits, so the markup is the state and nothing else holds a copy to drift.

A checkbox, switch, or radio row takes the whole row as its target and hovers at `surface0/50`. `toggle` is the bare switch for the right-hand side of a settings row.

A settings row's control never shrinks, so a narrow screen wraps the label rather than clipping a segment of the control.

```javascript
const settingRow = (label, hint, control) => `<div class="flex items-center justify-between gap-6 rounded-2xl px-3 py-2.5"><div class="min-w-0"><p class="text-sm text-text">${label}</p><p class="text-xs text-overlay1">${hint}</p></div><div class="shrink-0">${control}</div></div>`;

const toggles = `<div class="grid gap-x-4 sm:grid-cols-2">
    <div>${ui.checkbox({ name: 'a', label: 'Remote Control', hint: 'Reachable from claude.ai', checked: true })}${ui.checkbox({ name: 'b', label: 'Connectors' })}${ui.checkbox({ name: 'c', label: 'Locked', disabled: true, checked: true })}</div>
    <div>${ui.radios({ name: 'theme', value: 'mocha', options: [{ value: 'mocha', label: 'Mocha' }, { value: 'latte', label: 'Latte' }, { value: 'system', label: 'System', hint: 'Follow the OS' }] })}</div>
</div>
${ui.switch({ name: 'sw', label: 'Auto refresh', hint: 'Reload when the tab regains focus', checked: true })}
${settingRow('Dashboard window', 'Range for charts', ui.segmented({ name: 'window', value: '7', options: [{ value: '7', label: '7 days' }, { value: '30', label: '30 days' }] }))}
${settingRow('Telemetry', 'Send anonymous usage', ui.toggle({ name: 'telemetry', label: 'Telemetry' }))}`;
```

## Pill Tracks

```javascript
tracks: {
    panel: { track: 'bg-surface0', on: 'aria-checked:bg-surface1', count: 'bg-surface1' },
    ground: { track: 'bg-mantle', on: 'aria-checked:bg-surface0', count: 'bg-surface0' },
},

segmented({ name, value, options, full = false, ground = false }) {
    const fill = ui.tracks[ground ? 'ground' : 'panel'];
    return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="${full ? 'flex w-full' : 'inline-flex max-w-full overflow-x-auto'} items-center gap-1 rounded-full ${fill.track} p-1">
        ${options.map((o) => {
            const on = o.value === value;
            return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="group flex ${full ? 'flex-1' : ''} items-center justify-center gap-2 whitespace-nowrap rounded-full px-3.5 py-1.5 text-sm font-medium text-overlay1 transition-colors hover:text-text ${fill.on} aria-checked:text-text disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">${o.icon ? `<i data-lucide="${o.icon}" class="size-4 group-aria-checked:text-mauve"></i>` : ''}<span class="${o.icon && o.compact ? 'max-sm:hidden' : ''}">${ui.esc(o.label)}</span>${o.count !== undefined ? `<span class="${ui.cls.count} ${fill.count} text-subtext0 group-aria-checked:bg-mauve/15 group-aria-checked:text-mauve">${o.count}</span>` : ''}</button>`;
        }).join('')}
    </span>`;
},

tonePills({ name, value, options, ground = false }) {
    const fill = ui.tracks[ground ? 'ground' : 'panel'];
    return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="inline-flex max-w-full flex-wrap items-center gap-1 rounded-full ${fill.track} p-1">
        ${options.map((o) => {
            const on = o.value === value;
            const t = o.tone || 'neutral';
            const onCls = t === 'neutral' ? `${fill.on} aria-checked:text-text` : `aria-checked:bg-${t}/15 aria-checked:text-${t}`;
            return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" class="flex items-center gap-2 rounded-full px-3 py-1.5 text-xs font-medium text-overlay1 transition-colors hover:text-text ${onCls} ${ui.cls.focus}">${o.icon ? `<i data-lucide="${o.icon}" class="size-3.5"></i>` : ui.dot(t === 'neutral' ? 'overlay0' : t)}${ui.esc(o.label)}</button>`;
        }).join('')}
    </span>`;
},
```

A pill track is a control when it sits in a panel and a surface when it sits on the page ground. In a panel it takes `surface0` like the inputs beside it and selects at `surface1`. On the ground it takes `mantle` like the nav and selects at `surface0`. A builder reads both pairs from `ui.tracks`, so a track never picks its colors by hand, and a page-heading tab bar passes `ground: true`.

A count inside a segment steps one rung past the track and turns mauve with the selected segment, the same way a selected sidebar row's count does.

An option with `compact` drops its label below `sm` and keeps its icon. `full` stretches the segments across the row, for a choice that spans a form.

Tone pills mark the selected option in that option's own tone rather than the neutral step, for a choice whose values carry meaning, such as a priority.

```javascript
const pills = `
    ${sub('Segmented')}${row(ui.segmented({ name: 'mode', value: 'rendered', options: [{ value: 'rendered', label: 'Rendered', icon: 'eye' }, { value: 'source', label: 'Source', icon: 'code' }] }))}
    ${row(ui.segmented({ name: 'filter', value: 'active', options: [{ value: 'active', label: 'Active', count: 3 }, { value: 'idle', label: 'Idle', count: 12 }, { value: 'all', label: 'All' }] }))}
    ${ui.segmented({ name: 'source', value: 'new', full: true, options: [{ value: 'new', label: 'New directory', icon: 'folder-plus' }, { value: 'existing', label: 'Existing', icon: 'folder-open' }, { value: 'clone', label: 'Clone', icon: 'git-branch' }] })}
    ${sub('Toned')}${row(ui.tonePills({ name: 'priority', value: 'high', options: [{ value: 'default', label: 'Default' }, { value: 'high', label: 'High', tone: 'peach' }, { value: 'critical', label: 'Critical', tone: 'red' }] }))}
    ${row(ui.tonePills({ name: 'type', value: 'login', options: [{ value: 'login', label: 'Login', tone: 'blue', icon: 'user-round' }, { value: 'ssh', label: 'SSH key', tone: 'green', icon: 'key-round' }] }))}`;
```

## Choice Cards

```javascript
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
```

A choice card is a radio with an icon, a title, and a hint, for a choice that needs explaining. It hovers at full `surface0`, because the whole card is the target, and selects at `mauve/10`.

```javascript
const choices = ui.choices({ name: 'foreign', value: 'preserve', columns: 2, options: [{ value: 'preserve', label: 'Preserve', icon: 'shield-check', hint: 'Keep existing files' }, { value: 'clean', label: 'Clean', icon: 'eraser', hint: 'Back up, then remove' }, { value: 'skip', label: 'Skip', icon: 'circle-slash', hint: 'Leave untouched', disabled: true }] });
```

## Swatches

```javascript
swatches({ name, value, colors }) {
    return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="flex flex-wrap gap-2">
        ${colors.map((c) => {
            const on = c === value;
            return `<button type="button" role="radio" data-ui="radio" data-value="${c}" aria-checked="${on}" tabindex="${on ? 0 : -1}" aria-label="${c}" data-tip="${c}" class="group grid size-8 place-items-center rounded-full bg-${c} text-crust transition-transform hover:scale-110 ${ui.cls.focus}"><i data-lucide="check" class="size-4 scale-50 opacity-0 transition-all group-aria-checked:scale-100 group-aria-checked:opacity-100" stroke-width="3"></i></button>`;
        }).join('')}
    </span>`;
},
```

A swatch is a circle filled with the accent it names, showing a check when selected and the name as its tooltip. `ACCENTS` lists every accent token.

```javascript
const ACCENTS = ['rosewater', 'flamingo', 'pink', 'mauve', 'red', 'maroon', 'peach', 'yellow', 'green', 'teal', 'sky', 'sapphire', 'blue', 'lavender'];

const color = `
    ${sub('Swatches')}${ui.swatches({ name: 'color', value: 'mauve', colors: ACCENTS })}
    ${sub('Label editor')}<div class="flex flex-wrap items-center gap-2"><span data-label-preview>${ui.chip('research', { tone: 'mauve', icon: 'tag' })}</span>${ui.swatches({ name: 'label-color', value: 'mauve', colors: ['mauve', 'blue'] })}</div>`;

const readout = el.querySelector('#readout');
el.addEventListener('ui-change', (e) => {
    readout.textContent = `${e.detail.name} = ${JSON.stringify(e.detail.value)}`;
    if (e.detail.name === 'label-color') {
        const p = el.querySelector('[data-label-preview]');
        p.innerHTML = ui.chip('research', { tone: e.detail.value, icon: 'tag' });
        ui.icons(p);
    }
});
```

## Tag Field

A tag field is a `surface0` well holding chips and a bare input, ringed as one control through `focus-within`. Its chips take `surface1`, because they sit inside a `surface0` control. Enter adds the typed value as a chip, and each chip's remove button deletes it.

The input's minimum width holds its whole placeholder at 16px, so a line with less room wraps the input below the chips instead of clipping the text. A different placeholder resizes `min-w-44` to fit.

```javascript
const tags = `<div data-tags class="flex flex-wrap items-center gap-1.5 rounded-2xl bg-surface0 p-1.5 focus-within:ring-2 focus-within:ring-mauve/40">
    ${['go-idioms', 'go-cli-output'].map((t) => ui.chip(t, { mono: true, remove: 'tag-remove' }).replace('bg-surface0', 'bg-surface1')).join('')}
    <input data-tag-input placeholder="Add and press Enter" class="min-w-44 flex-1 bg-transparent px-2 py-1 text-sm text-text placeholder:text-overlay0 outline-none pointer-coarse:text-[16px]">
</div>`;
```

The view's `mount` wires Enter, and the remove action is registered once.

```javascript
const tags = el.querySelector('[data-tags]');
tags.querySelector('[data-tag-input]').addEventListener('keydown', (e) => {
    const v = e.target.value.trim();
    if (e.key !== 'Enter' || !v) return;
    e.target.insertAdjacentHTML('beforebegin', ui.chip(v, { mono: true, remove: 'tag-remove' }).replace('bg-surface0', 'bg-surface1'));
    e.target.value = '';
    ui.icons(tags);
});

ui.on('tag-remove', (b) => b.closest('span').remove());
```
