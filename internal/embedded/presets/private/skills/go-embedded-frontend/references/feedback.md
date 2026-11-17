# Feedback

**State reaches the user through one toast stack, one modal stack, and inline banners, badges, meters, and empty states that share the kit's tones.**

The toast, tooltip, and modal machinery lives in kit-core.md, and this file holds the builders and patterns built on it.

## Toasts

A toast confirms an action that has no other visible result. One that offers Undo gets a longer timeout, because the action is the reason it exists.

```javascript
const toasts = row([
    ['Success', 'notify', 'green'], ['With undo', 'notify', 'undo'],
].map(([l, a, t]) => ui.button(l, { action: a, attrs: `data-tone="${t}"` })).join(''));

ui.on('notify', (b) => {
    const t = b.dataset.tone;
    if (t === 'undo') return ui.toast('Task deleted', { icon: 'trash-2', action: { label: 'Undo', onClick: () => ui.toast('Restored', { tone: 'green', icon: 'undo-2' }) }, timeout: 5000 });
    const map = { green: ['Presets applied', 'circle-check'] };
    ui.toast(map[t][0], { tone: t, icon: map[t][1] });
});
```

## Tooltips

Any element takes a tooltip through `data-tip`, and `data-tip-side` picks the side. A truncated path or name carries its full text as its tooltip, since the truncation hides exactly what the user came to read.

A tooltip never opens on touch, so the text a row exists to show takes `ui.cls.rowText`, which truncates on a mouse and wraps on a coarse pointer.

```javascript
const tips = `${row(['top', 'right', 'bottom', 'left'].map((s) => `<button type="button" data-tip="Tooltip on the ${s}" data-tip-side="${s}" class="h-10 rounded-full bg-surface0 px-4 text-sm text-text hover:bg-surface1">${s}</button>`).join(''))}
    ${row(`${ui.iconButton('git-branch', { tip: 'Switch branch' })}${ui.iconButton('terminal', { tip: 'Open terminal' })}${ui.iconButton('settings-2', { tip: 'Settings' })}${ui.button('Publish', { icon: 'upload', disabled: true, tip: 'Nothing to publish yet' })}`)}
    <p data-tip="~/Documents/claudex/internal/server/static/views/project.js" class="truncate rounded-2xl bg-base px-4 py-2.5 font-mono text-xs text-subtext1">~/Documents/claudex/internal/server/static/views/project.js</p>`;
```

## Modals

```javascript
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
```

`ui.confirm` resolves to a boolean, and closing without an answer resolves `false`. A destructive action passes `danger`, which turns the header tile and the confirm button red.

A form modal's body is a `<form novalidate>` laying its fields in a grid that collapses to one column on a phone. Its primary button is `type="submit"` with a `form` attribute naming the form, because the footer sits outside the body. That pairing is what lets Enter in a field submit. The submit handler checks the fields through `ui.invalid`, since the browser's validation bubbles ignore the palette. Cancel carries `data-close`. A removal that cannot be undone asks the user to type the resource's name, and its `danger-solid` button stays disabled until the text matches.

```javascript
const modals = row(`${ui.button('Confirm', { action: 'restart' })}${ui.button('Delete', { variant: 'danger', icon: 'trash-2', action: 'delete' })}${ui.button('Form', { icon: 'square-pen', action: 'new-session' })}${ui.button('Typed confirm', { icon: 'shield-alert', action: 'remove-machine' })}`);

ui.on('restart', async () => {
    const ok = await ui.confirm({ title: 'Restart session?', message: 'The tmux window is replaced and Claude Code continues the last conversation.', confirmLabel: 'Restart', icon: 'rotate-cw' });
    if (ok) ui.toast('Restarted', { tone: 'green', icon: 'circle-check' });
});
ui.on('delete', async () => {
    const ok = await ui.confirm({ title: 'Delete orbit-api?', message: 'The directory and its <span class="font-mono text-text">.claude</span> folder are removed. This cannot be undone.', confirmLabel: 'Delete', danger: true, icon: 'trash-2' });
    if (ok) ui.toast('Deleted', { tone: 'red', icon: 'trash-2' });
});
ui.on('new-session', () => {
    const m = ui.modal.open({
        title: 'New session',
        icon: 'plus',
        body: `<form id="new-session-form" novalidate class="grid gap-4 pt-1 sm:grid-cols-2">
            ${ui.field('Name', ui.input({ name: 'm-name', placeholder: 'orbit-api', attrs: 'autofocus' }))}
            ${ui.field('Account', ui.select({ name: 'm-account', value: 'work', options: [{ value: 'personal', label: 'Personal', icon: 'user-round' }, { value: 'work', label: 'Work', icon: 'briefcase' }] }))}
            <div class="sm:col-span-2">${ui.field('Source', ui.segmented({ name: 'm-source', value: 'new', full: true, options: [{ value: 'new', label: 'New', icon: 'folder-plus' }, { value: 'existing', label: 'Existing', icon: 'folder-open' }, { value: 'clone', label: 'Clone', icon: 'git-branch' }] }), { group: true })}</div>
            <div class="sm:col-span-2">${ui.field('Prompt', ui.textarea({ name: 'm-prompt', rows: 4, placeholder: 'Optional first message' }))}</div>
            <div class="sm:col-span-2">${ui.switch({ name: 'm-rc', label: 'Remote Control', checked: true })}</div>
        </form>`,
        footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-close' })}${ui.button('Start', { icon: 'play', variant: 'primary', type: 'submit', attrs: 'form="new-session-form"' })}`,
    });
    m.el.addEventListener('click', (e) => { if (e.target.closest('[data-close]')) m.close(); });
    m.el.addEventListener('input', (e) => { if (e.target.getAttribute('aria-invalid') === 'true') ui.invalid(e.target); });
    m.el.querySelector('form').addEventListener('submit', (e) => {
        e.preventDefault();
        const name = e.target.querySelector('[data-name="m-name"]');
        if (!name.value.trim()) return ui.invalid(name, 'Name the session');
        m.close();
        ui.toast('Session started', { tone: 'green', icon: 'circle-check' });
    });
});
ui.on('remove-machine', () => {
    const m = ui.modal.open({
        title: 'Remove machine',
        icon: 'shield-alert',
        tone: 'red',
        size: 'sm',
        body: `<p class="mb-3 text-sm">The instance and its disk are terminated. Type <span class="font-mono text-text">dev-box</span> to confirm.</p><input data-typed class="${ui.cls.input} font-mono" placeholder="dev-box" autofocus>`,
        footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-close' })}${ui.button('Remove', { variant: 'danger-solid', icon: 'trash-2', disabled: true, attrs: 'data-remove' })}`,
    });
    const input = m.el.querySelector('[data-typed]');
    const go = m.el.querySelector('[data-remove]');
    input.addEventListener('input', () => { go.disabled = input.value !== 'dev-box'; });
    m.el.addEventListener('click', (e) => {
        if (e.target.closest('[data-close]')) m.close();
        if (e.target.closest('[data-remove]') && !go.disabled) { m.close(); ui.toast('dev-box removed', { tone: 'red', icon: 'trash-2' }); }
    });
});
```

## Banners

```javascript
banner({ tone = 'blue', icon = 'info', title = '', body = '', actions = '' }) {
    return `<div class="flex flex-wrap items-start gap-3 rounded-2xl bg-${tone}/10 p-4 text-sm">
        <i data-lucide="${icon}" class="mt-0.5 size-4 shrink-0 text-${tone}"></i>
        <div class="min-w-48 flex-1">${title ? `<p class="font-medium text-${tone}">${ui.esc(title)}</p>` : ''}${body ? `<p class="${title ? 'mt-0.5' : ''} text-subtext0">${body}</p>` : ''}</div>
        ${actions ? `<div class="flex flex-wrap gap-2">${actions}</div>` : ''}
    </div>`;
},
```

A banner is a large area, so its tint is `/10` and only the icon and the title take the tone. Its actions wrap below the text on a narrow panel.

```javascript
const banners = `<div class="flex flex-col gap-3">
    ${ui.banner({ tone: 'blue', icon: 'info', body: 'Remote Control sessions show up in claude.ai within a minute.' })}
    ${ui.banner({ tone: 'peach', icon: 'triangle-alert', title: 'Key pair has no key here', body: 'Paste the private key the machines authorize.', actions: ui.button('Add key', { size: 'sm', icon: 'key-round' }) })}
</div>`;
```

## Badges

Chips, statuses, counts, and `kbd` keys are kit-core builders. A notification marker is a `size-2` red dot positioned over the corner of an icon button.

```javascript
const badges = `
    ${sub('Chips')}${row(['neutral', 'green'].map((t) => ui.chip(t, { tone: t })).join(''))}
    ${row(`${ui.chip('opus', { tone: 'mauve', icon: 'sparkles' })}${ui.chip('a0338d7', { mono: true, icon: 'git-commit-horizontal' })}`)}
    ${sub('Status')}${row(`${ui.status('Running', 'green', { pulse: true })}${ui.status('Inactive')}`)}
    ${sub('Counts and markers')}${row(`${ui.count(3)}${ui.count(12, 'mauve')}${ui.count(2, 'red')}<span class="relative inline-flex">${ui.iconButton('bell', { tip: 'Notifications', variant: 'secondary' })}<span class="absolute top-1.5 right-1.5 size-2 rounded-full bg-red"></span></span><span class="flex gap-1">${ui.kbd(['⌘', 'K'])}</span>`)}`;
```

## Loading

```javascript
skeleton(cls = 'h-4 w-full') {
    return `<span class="block rounded-full bg-linear-to-r from-surface0 from-25% via-surface1 via-50% to-surface0 to-75% bg-[length:200%_100%] [animation:shimmer_1.4s_linear_infinite] ${cls}"></span>`;
},

meter(pct, tone = 'mauve', h = 'h-2') {
    return `<span class="block ${h} overflow-hidden rounded-full bg-surface0"><span class="block h-full rounded-full bg-${tone} transition-[width] duration-300" style="width:${pct}%"></span></span>`;
},
```

A skeleton is a shimmering bar in the shape of the content it stands in for, so the layout does not jump when the content lands. A meter is a `surface0` track filled in its tone.

```javascript
const loading = `
    ${row(`<span class="flex items-center gap-2 text-sm text-overlay1">${ui.spinner()}Loading sessions</span>${ui.button('Refreshing', { icon: 'loader-circle', spin: true, disabled: true })}`)}
    <div class="flex flex-col gap-3 ${ui.cls.tile} p-4">${[1].map(() => `<div class="flex items-center gap-3">${ui.skeleton('size-10 !rounded-2xl shrink-0')}<div class="flex flex-1 flex-col gap-2">${ui.skeleton('h-3 w-2/5')}${ui.skeleton('h-3 w-4/5')}</div></div>`).join('')}</div>
    <div class="flex flex-col gap-3">${[['Downloading', 64, 'mauve', '12.4 MB/s']].map(([l, p, t, v]) => `<div><div class="mb-1.5 flex justify-between text-xs"><span class="text-subtext0">${l}</span><span class="font-mono text-overlay1">${v}</span></div>${ui.meter(p, t)}</div>`).join('')}</div>`;
```

## Empty States

```javascript
empty({ icon, title, hint = '', action = '' }) {
    return `<div class="flex h-full min-h-40 flex-col items-center justify-center gap-3 p-6 text-center">
        <span class="flex size-12 items-center justify-center rounded-full bg-surface0 text-overlay1"><i data-lucide="${icon}" class="size-5"></i></span>
        <p class="font-display text-[16px] text-text">${ui.esc(title)}</p>
        ${hint ? `<p class="max-w-sm text-sm text-overlay1">${hint}</p>` : ''}
        ${action}
    </div>`;
},
```

An empty state is an icon in a `surface0` circle, a title, an optional hint, and an optional action, centered in the space the content would fill.

```javascript
const empties = `<div class="grid gap-3 sm:grid-cols-2">
    <div class="${ui.cls.tile}">${ui.empty({ icon: 'inbox', title: 'No active sessions', action: ui.button('New session', { icon: 'plus', size: 'sm' }) })}</div>
    <div class="${ui.cls.tile}">${ui.empty({ icon: 'search-x', title: 'No matches', hint: 'Nothing matches <span class="font-mono text-subtext0">orbt</span>.' })}</div>
</div>`;
```
