# Selects and Menus

**A select, a date picker, and a menu are buttons that open one kit popover, and each is driven from the keyboard as fully as from the pointer.**

## Select

```javascript
select({ name, value = '', options, placeholder = 'Select', searchable = false, icon, pill = false, attrs = '' }) {
    const current = options.find((o) => o.value === value);
    ui.selectOptions[name] = { options, searchable };
    const lead = current?.dot ? ui.dot(current.dot) : current?.icon ? `<i data-lucide="${current.icon}" class="size-4 shrink-0 text-overlay1"></i>` : icon ? `<i data-lucide="${icon}" class="size-4 shrink-0 text-overlay1"></i>` : '';
    return `<button type="button" data-ui="select" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" aria-haspopup="listbox" aria-expanded="false" class="flex ${pill ? 'h-10 w-auto rounded-full' : 'w-full rounded-2xl'} items-center gap-2.5 bg-surface0 px-4 py-2.5 text-left text-sm transition-colors hover:bg-surface1 aria-expanded:bg-surface1 ${ui.cls.focus} pointer-coarse:text-[16px]" ${attrs}>
        <span data-select-lead class="contents">${lead}</span>
        <span data-select-label class="min-w-0 flex-1 truncate ${current ? 'text-text' : 'text-overlay0'}">${ui.esc(current ? current.label : placeholder)}</span>
        <i data-lucide="chevrons-up-down" class="size-4 shrink-0 text-overlay1"></i>
    </button>`;
},
```

A select is a button opening a listbox popover rather than a native `<select>`, because a native list ignores the palette and the radius. Its options register in `ui.selectOptions` under the select's `name`, so a name is unique on the page.

An option is `{ value, label, hint, icon, dot }`, and the trigger shows the chosen option's icon or dot. `pill` gives the toolbar form, a `rounded-full` trigger sized to its label. `attrs` land on the trigger button, as they do for `ui.datePicker`.

```javascript
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
        search.addEventListener('input', () => {
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
```

The listbox matches the trigger's width with a 220px floor. The arrow keys move the active option, Enter chooses, and the pointer moves the same highlight, so the keyboard and the mouse never disagree about which option is active. A `searchable` select puts a search field above the list and ranks the options with `ui.fuzzy`.

Choosing updates the trigger in place, returns focus to it, and emits `ui-change`.

```javascript
const selects = `<div class="grid gap-3">
    ${ui.field('Account', ui.select({ name: 'account', value: 'work', options: [{ value: 'personal', label: 'Personal', icon: 'user-round', hint: '~/.claude' }, { value: 'work', label: 'Work', icon: 'briefcase', hint: '~/.claude-work' }] }))}
    ${ui.field('Region', ui.select({ name: 'region', placeholder: 'Choose a region', searchable: true, icon: 'globe', options: ['us-east-1', 'eu-west-1'].map((r) => ({ value: r, label: r })) }))}
    ${ui.field('Label', ui.select({ name: 'label', value: 'infra', options: [['bug', 'Bug', 'red'], ['feature', 'Feature', 'green']].map(([value, label, dot]) => ({ value, label, dot })) }))}
    ${sub('Toolbar')}${row(`${ui.select({ name: 'sort', value: 'recent', pill: true, icon: 'arrow-down-up', options: [{ value: 'recent', label: 'Recent' }, { value: 'name', label: 'Name' }] })}${ui.iconButton('ellipsis-vertical', { tip: 'More', variant: 'secondary', attrs: 'data-action="more-menu"' })}`)}
</div>`;
```

## Date Picker

```javascript
datePicker({ name, value = '', placeholder = 'Pick a date', pill = false, attrs = '' }) {
    return `<button type="button" data-ui="date" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" aria-haspopup="dialog" aria-expanded="false" class="flex ${pill ? 'h-10 w-auto rounded-full' : 'w-full rounded-2xl'} items-center gap-2.5 bg-surface0 px-4 py-2.5 text-left text-sm transition-colors hover:bg-surface1 aria-expanded:bg-surface1 ${ui.cls.focus} pointer-coarse:text-[16px]" ${attrs}>
        <i data-lucide="calendar" class="size-4 shrink-0 text-overlay1"></i>
        <span data-date-label data-placeholder="${ui.esc(placeholder)}" class="min-w-0 flex-1 truncate ${value ? 'text-text' : 'text-overlay0'}">${value ? ui.fmtDate(value) : ui.esc(placeholder)}</span>
    </button>`;
},

fmtDate(v) {
    return new Date(`${v}T00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
},

ymd(d) {
    const p = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
},
```

A date is a `YYYY-MM-DD` string in local time. `ui.ymd` reads the date's local parts rather than calling `toISOString`, which converts to UTC and moves the day in every timezone ahead of it. `ui.fmtDate` parses `${v}T00:00`, because a bare `YYYY-MM-DD` parses as UTC midnight and shows the previous day in every timezone behind it.

```javascript
openDate(button) {
    const value = button.dataset.value;
    let month = value ? new Date(`${value}T00:00`) : new Date();
    month = new Date(month.getFullYear(), month.getMonth(), 1);
    button.setAttribute('aria-expanded', 'true');
    const pop = ui.popover.open(button, '', { width: 296, onClose: () => button.setAttribute('aria-expanded', 'false') });

    const draw = () => {
        const today = ui.ymd(new Date());
        const start = new Date(month.getFullYear(), month.getMonth(), 1 - month.getDay());
        const days = Array.from({ length: 42 }, (_, i) => new Date(start.getFullYear(), start.getMonth(), start.getDate() + i));
        const day = (d) => {
            const v = ui.ymd(d);
            const tone = v === button.dataset.value ? 'bg-mauve font-medium text-crust'
                : v === today ? 'text-mauve hover:bg-surface0'
                : d.getMonth() === month.getMonth() ? 'text-text hover:bg-surface0' : 'text-overlay0 hover:bg-surface0';
            return `<button type="button" data-set="${v}" class="h-9 rounded-full text-sm tabular-nums transition-colors ${tone}">${d.getDate()}</button>`;
        };
        pop.innerHTML = `<div class="p-1.5">
            <div class="mb-2 flex items-center gap-1"><span class="flex-1 px-2 font-display text-sm font-medium text-text">${month.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}</span>${ui.iconButton('chevron-left', { tip: 'Previous month', size: 'sm', attrs: 'data-nav="-1"' })}${ui.iconButton('chevron-right', { tip: 'Next month', size: 'sm', attrs: 'data-nav="1"' })}</div>
            <div class="mb-1 grid grid-cols-7 text-center text-[11px] text-overlay0">${['S', 'M', 'T', 'W', 'T', 'F', 'S'].map((d) => `<span class="py-1">${d}</span>`).join('')}</div>
            <div class="grid grid-cols-7 gap-0.5">${days.map(day).join('')}</div>
            <div class="mt-2 flex justify-between">${ui.button('Clear', { variant: 'ghost', size: 'sm', attrs: 'data-set=""' })}${ui.button('Today', { variant: 'ghost', size: 'sm', attrs: `data-set="${today}"` })}</div>
        </div>`;
        ui.icons(pop);
    };
    pop.addEventListener('click', (e) => {
        const nav = e.target.closest('[data-nav]');
        if (nav) {
            month = new Date(month.getFullYear(), month.getMonth() + Number(nav.dataset.nav), 1);
            draw();
            return;
        }
        const set = e.target.closest('[data-set]');
        if (!set) return;
        const v = set.dataset.set;
        button.dataset.value = v;
        const label = button.querySelector('[data-date-label]');
        label.textContent = v ? ui.fmtDate(v) : label.dataset.placeholder;
        label.classList.toggle('text-overlay0', !v);
        label.classList.toggle('text-text', !!v);
        ui.popover.close();
        button.focus();
        ui.emit(button, v);
    });
    draw();
},
```

The calendar always shows six weeks from the Sunday before the 1st, so its height never changes between months. Today is mauve text, the chosen day is a `bg-mauve` fill, days outside the month fall back to `overlay0`, and Clear and Today sit under the grid.

```javascript
const dates = `<div class="grid gap-3 sm:grid-cols-2">
    ${ui.field('Due', ui.datePicker({ name: 'due' }))}
    ${ui.field('Starts', ui.datePicker({ name: 'starts', value: ui.ymd(new Date()) }))}
    <div class="sm:col-span-2">${sub('Toolbar')}<div class="mt-1.5">${ui.datePicker({ name: 'toolbar-date', pill: true, placeholder: 'Due date' })}</div></div>
</div>`;
```

## Menu

```javascript
menu(anchor, items) {
    anchor.setAttribute('aria-expanded', 'true');
    const el = ui.popover.open(anchor, items.map((item, i) => (item === '-' ? '<span class="my-1 block h-px"></span>' : `
        <button type="button" data-menu-index="${i}" ${item.disabled ? 'disabled' : ''} class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${item.danger ? 'text-red hover:bg-red/15' : 'text-subtext0 hover:bg-surface0 hover:text-text'}">
            <i data-lucide="${item.icon}" class="size-4 shrink-0"></i><span class="min-w-0 flex-1"><span class="block">${ui.esc(item.label)}</span>${item.hint ? `<span class="block text-xs text-overlay1">${ui.esc(item.hint)}</span>` : ''}</span>${item.keys ? `<span class="flex gap-1">${ui.kbd(item.keys)}</span>` : ''}
        </button>`)).join(''), { align: 'end', width: 260, onClose: () => anchor.setAttribute('aria-expanded', 'false') });
    el.addEventListener('click', (e) => {
        const b = e.target.closest('[data-menu-index]');
        if (!b) return;
        ui.popover.close();
        items[Number(b.dataset.menuIndex)].onSelect?.();
    });
},
```

A menu is a 260px popover aligned to its trigger's end. An item is `{ icon, label, hint, keys, danger, disabled, onSelect }`, and `'-'` leaves a gap between groups rather than drawing a rule. A danger item is red text hovering at `red/15`.

The trigger is an `ellipsis` or `ellipsis-vertical` icon button, and it carries `aria-expanded` while its menu is open, which keeps a trigger hidden behind `ui.cls.reveal` in view. Choosing closes the menu before `onSelect` runs, because the popover's `z-[70]` would otherwise float the menu over a modal the handler opens at `z-50`.

```javascript
ui.on('more-menu', (b) => ui.menu(b, [
    { icon: 'pencil', label: 'Rename', keys: ['R'], onSelect: () => ui.toast('Rename') },
    { icon: 'archive', label: 'Archive', hint: 'Hidden from the dashboard', onSelect: () => ui.toast('Archived', { icon: 'archive' }) },
    '-',
    { icon: 'trash-2', label: 'Delete', danger: true, onSelect: () => ui.toast('Deleted', { tone: 'red', icon: 'trash-2' }) },
]));
```
