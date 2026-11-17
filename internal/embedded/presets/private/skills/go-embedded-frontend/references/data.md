# Data

**Data sits in `surface0/50` tiles and rows inside `mantle` panels, with values in `text`, labels in `overlay1`, and row actions revealed on hover.**

## Stats

A stat is a toned tile beside a label and a `font-display text-2xl` value, in a grid that grows from two columns to four.

```javascript
const stats = `<div class="grid grid-cols-2 gap-3 md:grid-cols-4">${[['Sessions', '4', 'activity', 'green'], ['Tracked', '17', 'folder-git-2', 'mauve']].map(([l, v, ic, t]) => `<div class="flex items-center gap-3 ${ui.cls.tile} p-4">${ui.tile(ic, t)}<div class="min-w-0"><p class="text-xs text-overlay1">${l}</p><p class="font-display text-2xl text-text tabular-nums">${v}</p></div></div>`).join('')}</div>`;
```

## Cards

A card inside a panel is `rounded-3xl bg-surface0/50` and does not hover, since its buttons are the targets. Its header is a tile, a title over a mono path, and a status. Its facts are a two-column `dl`, and its actions sit at the bottom through `mt-auto`, so the cards in one row line their buttons up.

```javascript
const cards = `<ul class="grid gap-3 sm:grid-cols-2">${[['orbit-api', '~/Documents/orbit-api', ['go-idioms', 'go-http-server']]].map(([n, d, presets], i) => `<li><article class="flex h-full flex-col gap-3 rounded-3xl bg-surface0/50 p-4">
    <header class="flex items-center gap-3">${ui.tile('folder-git-2', i ? 'teal' : 'mauve')}<div class="min-w-0 flex-1"><h3 class="font-display font-medium text-text">${n}</h3><p class="truncate font-mono text-xs text-overlay1">${d}</p></div>${ui.status('Running', 'green')}</header>
    <dl class="grid grid-cols-2 gap-x-4 gap-y-3">${[['clock', 'Started', '12m ago'], ['user-round', 'Account', 'Work']].map(([ic, l, v]) => `<div class="min-w-0"><dt class="flex items-center gap-1.5 text-xs text-overlay1"><i data-lucide="${ic}" class="size-3.5"></i>${l}</dt><dd class="mt-0.5 truncate text-sm text-subtext1">${v}</dd></div>`).join('')}<div class="col-span-2"><dt class="flex items-center gap-1.5 text-xs text-overlay1"><i data-lucide="layers" class="size-3.5"></i>Presets</dt><dd class="mt-1 flex flex-wrap gap-1.5">${presets.map((p) => ui.chip(p)).join('')}</dd></div></dl>
    <footer class="mt-auto flex gap-2">${ui.button('Restart', { icon: 'rotate-cw', size: 'sm' })}${ui.button('Stop', { icon: 'square', size: 'sm', variant: 'danger' })}</footer>
</article></li>`).join('')}</ul>`;
```

## Details

Details are a `dl` grid whose label column is as wide as its widest label. A value the user copies is mono with a small copy button beside it, and a command block is a `base` well whose copy button appears on hover.

```javascript
const kv = `<dl class="grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2.5 text-sm">${[['Instance', 'i-0a1b2c3d4e5f', true], ['Shape', '4 vCPU · 16 GB']].map(([k, v, copy]) => `<dt class="text-xs text-overlay1">${k}</dt><dd class="flex min-w-0 items-center gap-1"><span class="truncate ${copy ? 'font-mono text-xs' : ''} text-text">${v}</span>${copy ? ui.iconButton('copy', { tip: 'Copy', size: 'sm', attrs: `data-action="copy-value" data-copy="${v}"` }) : ''}</dd>`).join('')}</dl>
    <div class="group relative"><pre class="${ui.cls.well} overflow-x-auto p-4 font-mono text-xs leading-relaxed text-subtext1">ssh -i ~/.ssh/anbu ec2-user@10.0.4.17
tmux attach -t '=claudex'</pre><span class="absolute top-2 right-2 opacity-0 transition-opacity group-hover:opacity-100 max-md:opacity-100">${ui.iconButton('copy', { tip: 'Copy', size: 'sm', variant: 'secondary', attrs: 'data-action="copy-value" data-copy="ssh"' })}</span></div>`;

ui.on('copy-value', () => ui.toast('Copied to clipboard', { tone: 'green', icon: 'clipboard-check', timeout: 1800 }));
```

## Table

A table lives in a scroll region with a sticky `mantle` header, so the column names stay in view over a long list. It uses `border-separate`, because a collapsed table ignores the radius on the end cells that round each row.

A row hovers at `surface0/50`, its action button appears on hover, and a secondary column drops below `sm`.

```javascript
const rows = [['api.github.com', 'allowed', 1284, '2.1 ms', 'green'], ['doubleclick.net', 'blocked', 822, '0.0 ms', 'red']];
const th = 'px-3 pb-2 text-left text-[11px] font-medium uppercase tracking-wider text-overlay1';
const table = `<div class="relative -mr-3.5 min-h-0"><div data-scroll class="max-h-80 overflow-auto"><table class="w-full border-separate border-spacing-0 text-sm">
    <thead class="sticky top-0 z-10 bg-mantle"><tr><th class="${th}">Domain</th><th class="${th}">Result</th><th class="${th} text-right">Queries</th><th class="${th} text-right max-sm:hidden">Latency</th><th class="${th} w-10"></th></tr></thead>
    <tbody>${rows.map(([d, r, q, l, t]) => `<tr class="group transition-colors hover:bg-surface0/50 [&>td:first-child]:rounded-l-2xl [&>td:last-child]:rounded-r-2xl"><td class="px-3 py-2.5 font-mono text-xs text-text">${d}</td><td class="px-3 py-2.5">${ui.chip(r, { tone: t })}</td><td class="px-3 py-2.5 text-right font-mono text-xs text-subtext0 tabular-nums">${q.toLocaleString()}</td><td class="px-3 py-2.5 text-right font-mono text-xs text-overlay1 tabular-nums max-sm:hidden">${l}</td><td class="px-1 py-1 text-right"><span class="opacity-0 transition-opacity group-hover:opacity-100 max-md:opacity-100">${ui.iconButton('ellipsis', { tip: 'Actions', size: 'sm', attrs: 'data-action="more-menu"' })}</span></td></tr>`).join('')}</tbody>
</table></div></div>`;
```

## List Rows

A list row is a round complete button, the text, an optional priority chip, and a due label. An overdue row takes `red/10` and keeps it on hover, and a done row strikes its text through in `overlay0`.

```javascript
const tasks = [['Rotate the staging deploy key', 'critical', 'red', '2d overdue', true], ['Ship v2.4.0', '', '', 'done', false, true]];
const list = `<ul class="flex flex-col gap-1">${tasks.map(([text, p, t, due, over, done]) => `<li class="group flex items-center gap-3 rounded-2xl px-3 py-2.5 ${over ? 'bg-red/10' : 'hover:bg-surface0/50'}">
    <button type="button" data-tip="${done ? 'Reopen' : 'Complete'}" class="grid size-5 shrink-0 place-items-center rounded-full ${done ? 'bg-green text-crust' : 'bg-surface1 hover:bg-surface2'}">${done ? '<i data-lucide="check" class="size-3.5" stroke-width="3"></i>' : ''}</button>
    <span class="min-w-0 flex-1 truncate text-sm ${done ? 'text-overlay0 line-through' : 'text-text'}">${text}</span>
    ${p ? ui.chip(p, { tone: t }) : ''}
    <span class="w-20 shrink-0 text-right text-xs ${over ? 'font-medium text-red' : 'text-overlay1'}">${due}</span>
    <span class="opacity-0 transition-opacity group-hover:opacity-100 max-md:opacity-100">${ui.iconButton('x', { tip: 'Delete', size: 'sm', variant: 'danger' })}</span></li>`).join('')}</ul>`;
```
