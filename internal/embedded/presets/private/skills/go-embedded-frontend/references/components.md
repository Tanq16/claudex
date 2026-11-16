# Components

**Every control on the page comes from one `ui` object in `static/ui.js`, so a fill, a radius, or a focus ring decided once lands identically in every view.**

## The Kit

`ui.js` is a plain script loaded before `app.js`. It defines one `ui` object whose builders return HTML strings, and a view composes those strings in a template literal. A control's markup and classes therefore live in one place instead of being restated per view, which is where two shades of the same control come from.

```javascript
const ui = {
    cls: {
        input: 'w-full rounded-2xl bg-surface0 px-4 py-2.5 text-sm text-text placeholder:text-overlay0 outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px]',
        label: 'text-xs font-medium uppercase tracking-wide text-overlay1',
        group: 'px-3 pt-4 pb-1 text-[11px] font-medium uppercase tracking-wider text-overlay0',
        panel: 'rounded-3xl bg-mantle',
        well: 'rounded-2xl bg-base',
        tile: 'rounded-2xl bg-surface0/50',
        row: 'rounded-2xl transition-colors hover:bg-surface0/50',
        focus: 'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40',
        count: 'inline-flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 font-mono text-[11px] tabular-nums',
    },
    tone: {
        chip: (t) => (t === 'neutral' ? 'bg-surface0 text-subtext0' : `bg-${t}/15 text-${t}`),
        text: (t) => (t === 'neutral' ? 'text-text' : `text-${t}`),
    },
    tracks: {
        panel: { track: 'bg-surface0', on: 'aria-checked:bg-surface1', count: 'bg-surface1' },
        ground: { track: 'bg-mantle', on: 'aria-checked:bg-surface0', count: 'bg-surface0' },
    },
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
};
```

The builders are `button`, `iconButton`, `chip`, `count`, `status`, `kbd`, `tile`, `avatar`, `field`, `input`, `search`, `textarea`, `stepper`, `number`, `checkbox`, `switch`, `toggle`, `radios`, `segmented`, `tonePills`, `choices`, `swatches`, `select`, `datePicker`, `banner`, `empty`, `skeleton`, `meter`, `spinner`, and `seek`. The layers are `popover`, `menu`, `tooltip`, `modal`, `confirm`, and `toast`, and the page machinery is `scrollbar`, `sidebar`, `theme`, `layer`, and `scheme`.

Every interpolated value passes through `ui.esc`, because a builder returns markup and an unescaped name or path becomes an injection.

Behavior is wired once, by delegation on `document`. A control carries `data-ui` for a built-in behavior such as a radio, a switch, a select, or a stepper, and `data-action` for a handler registered with `ui.on`. A view re-renders by replacing `innerHTML`, which drops every per-element listener, while a delegated one never needs binding again.

A control reports a change as a bubbling `ui-change` event carrying `{ name, value }`, so a view listens once on its root for every control inside it.

`ui.icons(root)` runs after every insertion, since Lucide replaces each `<i data-lucide>` once and never observes the DOM.

## Fills and States

A control at rest is `surface0`, and it steps to `surface1` on hover, while open, or when it is the selected segment of a `surface0` track. Anything nested inside a `surface0` control takes `surface1` for the same reason: a chip in a tag field, a `kbd` in a search field, and a stepper's buttons all have to read against the control around them.

A pill track is a control when it sits in a panel and a surface when it sits on the page ground. In a panel it takes `surface0` like the inputs beside it and selects at `surface1`. On the ground it takes `mantle` like the nav and selects at `surface0`. The `tracks` entry a builder reads carries both pairs, so a track never picks its colors by hand.

```javascript
segmented({ name, value, options, full = false, ground = false }) {
    const fill = ui.tracks[ground ? 'ground' : 'panel'];
    return `<span role="radiogroup" data-ui="radiogroup" data-name="${ui.esc(name)}" data-value="${ui.esc(value)}" class="${full ? 'flex w-full' : 'inline-flex max-w-full overflow-x-auto'} items-center gap-1 rounded-full ${fill.track} p-1">
        ${options.map((o) => {
            const on = o.value === value;
            return `<button type="button" role="radio" data-ui="radio" data-value="${ui.esc(o.value)}" aria-checked="${on}" tabindex="${on ? 0 : -1}" ${o.disabled ? 'disabled' : ''} class="group flex ${full ? 'flex-1' : ''} items-center justify-center gap-2 whitespace-nowrap rounded-full px-3.5 py-1.5 text-sm font-medium text-overlay1 transition-colors hover:text-text ${fill.on} aria-checked:text-text disabled:cursor-not-allowed disabled:opacity-50 ${ui.cls.focus}">${o.icon ? `<i data-lucide="${o.icon}" class="size-4 group-aria-checked:text-mauve"></i>` : ''}<span class="${o.icon && o.compact ? 'max-sm:hidden' : ''}">${ui.esc(o.label)}</span>${o.count !== undefined ? `<span class="${ui.cls.count} ${fill.count} text-subtext0 group-aria-checked:bg-mauve/15 group-aria-checked:text-mauve">${o.count}</span>` : ''}</button>`;
        }).join('')}
    </span>`;
},
```

A count inside a segment steps one rung past the track and turns mauve with the selected segment, the same way a selected sidebar row's count does.

A list row hovers at `surface0/50` and is selected at `surface0`, with its icon turning mauve. A hover at the selected fill would make the row under the pointer indistinguishable from the chosen one.

A menu item or a listbox option hovers at full `surface0`. A menu has no selected row to confuse with, and the highlight is what the arrow keys move.

A tile, stat block, or card inside a panel is `surface0/50` and keeps that fill on hover unless the whole tile is the click target. A `surface0` hover swallows the `surface0` chips and buttons inside it.

An accent tint is `/15` on a small element such as a chip, a toned pill, or an icon tile, and `/10` on a large area such as a banner, a callout, or a selected choice card. A large area at `/15` reads as a colored panel rather than a marked one.

Shadows sit only under floating layers, as `shadow-crust/N`, and nothing glows. Elevation inside the page is carried by tone alone, so a shadow always means the element floats.

Every interactive element takes `ui.cls.focus`, and a text field shows the same ring on plain `focus`. Mauve marks focus, the primary action, and the selected icon, which is why no structural fill borrows it.

## Catalog

| Builder | At rest | States |
|---|---|---|
| `button` primary | `bg-mauve text-crust` | `hover:bg-mauve/85` |
| `button` secondary | `bg-surface0 text-text` | `hover:bg-surface1` |
| `button` ghost | `text-subtext0` | `hover:bg-surface0 hover:text-text` |
| `button` danger | `bg-red/15 text-red` | `hover:bg-red/25` |
| `iconButton` ghost | `text-overlay1` | `hover:bg-surface0 hover:text-text`, pressed `bg-surface0 text-text` |
| `iconButton` inset, inside a `surface0` control | `text-overlay1` | `hover:bg-surface1 hover:text-text` |
| `input`, `textarea`, `number` | `ui.cls.input` | `focus:ring-2 focus:ring-mauve/40` |
| `search` | `ui.cls.input` with `rounded-full` and a leading icon | as `input` |
| `select`, `datePicker` | `ui.cls.input` with a trailing `chevrons-up-down` | `hover:bg-surface1 aria-expanded:bg-surface1` |
| `segmented`, `tonePills` | `ui.tracks` | toned segment `bg-<tone>/15 text-<tone>` |
| `checkbox` | box `size-5 rounded-lg bg-surface1` | checked `bg-mauve` with a `crust` check |
| `radios` | dot `size-5 rounded-full bg-surface1` | checked `bg-mauve` with a `crust` dot |
| `switch`, `toggle` | track `h-6 w-11 rounded-full bg-surface1`, knob `bg-text` | checked track `bg-mauve`, knob `bg-crust` |
| checkbox, radio, and switch rows | `rounded-2xl px-3 py-2.5` | `hover:bg-surface0/50` |
| `choices` | `ui.cls.tile p-3` | `hover:bg-surface0`, checked `bg-mauve/10` |
| `chip`, `status` | `rounded-full px-2.5 py-1 text-xs font-medium`, `ui.tone.chip` | |
| `count` | `ui.cls.count`, `ui.tone.chip` | |
| `kbd` | `h-5 min-w-5 rounded-md bg-surface1 px-1 font-mono text-[11px] text-subtext1` | |
| `tile` | `rounded-2xl`, `ui.tone.chip` | |
| `banner` | `rounded-2xl bg-<tone>/10 p-4`, icon and title in the tone | |
| `meter` | `rounded-full bg-surface0`, fill in the tone | |
| `empty` | icon in a `size-12 rounded-full bg-surface0` | |
| popover | `rounded-2xl bg-mantle p-1.5 ring-1 ring-surface0 shadow-2xl shadow-crust/80` | |
| menu item | `rounded-xl px-3 py-2 text-sm text-subtext0` | `hover:bg-surface0 hover:text-text`, danger `text-red hover:bg-red/15` |
| tooltip | `rounded-xl bg-surface0 px-2.5 py-1.5 text-xs text-text shadow-lg shadow-crust/60` | |
| modal | `rounded-3xl bg-mantle shadow-2xl shadow-crust`, backdrop `bg-crust/70 backdrop-blur-sm` | |
| toast | `rounded-full bg-surface0 text-sm text-text shadow-xl shadow-crust/70` | |

| Size | Button | Icon button | Button icon |
|---|---|---|---|
| `sm` | `h-8 px-3 text-xs` | `size-8` | `size-3.5` |
| `md` | `h-10 px-4 text-sm` | `size-10` | `size-4` |
| `lg` | `h-12 px-6 text-sm` | | `size-4` |

Icons run `size-3` in chips, `size-4` by default, `size-5` in a `md` tile, a modal header, and an empty state, and `size-6` in a large tile.

| Text | Classes |
|---|---|
| Page title | `font-display text-2xl text-text` |
| Modal title | `font-display text-lg font-medium text-text` |
| Section or panel title | `font-display text-[16px] font-medium text-text` |
| Field label | `ui.cls.label` |
| Group label in a sidebar | `ui.cls.group` |
| Body | `text-sm text-subtext0` |
| Hint and metadata | `text-xs text-overlay1` |

## Radius

| Radius | Used by |
|---|---|
| `rounded-full` | buttons, icon buttons, chips, counts, pill tracks and segments, the nav, search fields, toasts |
| `rounded-3xl` | panels, cards, modals |
| `rounded-2xl` | inputs, selects, rows, popovers, tiles, wells, banners |
| `rounded-xl` | menu items, listbox options, tooltips |
| `rounded-lg` | checkboxes |
| `rounded-md` | `kbd` |

## The Shell and Scrolling

The page never scrolls. The body is an `h-dvh overflow-hidden` column, the header is `shrink-0`, and `<main>` is `flex-1 min-h-0`, so the header and nav hold still at every size.

A gallery or settings view gets one scroll region filling `<main>`, and a workspace view gets side-by-side panels that each scroll on their own. One page-level scroll in a workspace would push a sibling panel off screen whenever a list grows.

A gallery lays its cards on `grid items-start gap-4 lg:grid-cols-2 2xl:grid-cols-3 3xl:grid-cols-4`, with `--breakpoint-3xl: 120rem;` in the page's `@theme`, so a wide monitor fills with cards rather than stretching two.

Native scrollbars are hidden page-wide, and every scroll region carries `data-scroll` and gets a drawn thumb. A native bar takes layout width on one platform and overlays content on another, and it cannot be inset from a rounded corner.

```css
* { scrollbar-width: none; }
*::-webkit-scrollbar { display: none; }
```

Every scroll region reserves a 14px lane on its right, so content never runs under the thumb. A region flush against its parent's right edge pulls out by the lane with `-mr-3.5`, which keeps its content column aligned with the page.

The thumb sits 4px from the region's edge as `w-1.5 rounded-full bg-overlay0/60`. It widens to `w-2 bg-overlay1` under the pointer and fades 900ms after the last scroll. Its track is inset at least 6px top and bottom, and by half the corner radius on a rounded region or parent, so the thumb never pokes past a curve.

The thumb is positioned against the region's parent, which `ui.anchor` makes `relative` only when the class list carries no position utility. It reads class names rather than computed style, because the browser build compiles classes after the script runs and computed style still reads `static` for an element about to become `fixed`.

```javascript
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
    const thumb = document.createElement('span');
    thumb.setAttribute('aria-hidden', 'true');
    thumb.className = 'absolute top-0 z-20 w-1.5 rounded-full bg-overlay0/60 opacity-0 transition-[opacity,width,background-color] duration-300 hover:w-2 hover:bg-overlay1 cursor-default';
    host.appendChild(thumb);
    const grip = el.tagName === 'TEXTAREA' ? host.querySelector('[data-ui="grip"]') : null;
    let hideTimer;
    let dragging = null;
    let lane = -1;
    let track = [6, 6];

    const fit = () => {
        const px = (node, prop) => parseFloat(getComputedStyle(node)[prop]) || 0;
        const want = Math.max(0, ui.scrollLane - px(el, 'paddingRight'));
        if (want !== lane) {
            lane = want;
            el.style.borderRight = `${lane}px solid transparent`;
        }
        const above = el.offsetTop;
        const below = host.clientHeight - el.offsetTop - el.offsetHeight;
        track = [
            Math.max(6, px(el, 'borderTopRightRadius') / 2, px(host, 'borderTopRightRadius') / 2 - above),
            Math.max(6, px(el, 'borderBottomRightRadius') / 2, px(host, 'borderBottomRightRadius') / 2 - below, grip ? el.offsetTop + el.offsetHeight - grip.offsetTop + 4 : 0),
        ];
    };
    const place = () => {
        fit();
        const { scrollTop, scrollHeight, clientHeight } = el;
        const overflow = scrollHeight - clientHeight;
        if (overflow <= 1) {
            thumb.style.display = 'none';
            return false;
        }
        thumb.style.display = '';
        const room = clientHeight - track[0] - track[1];
        const h = Math.max(28, (clientHeight / scrollHeight) * room);
        thumb.style.right = `${host.clientWidth - el.offsetLeft - el.offsetWidth + 4}px`;
        thumb.style.height = `${h}px`;
        thumb.style.transform = `translateY(${el.offsetTop + track[0] + (scrollTop / overflow) * (room - h)}px)`;
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
    new MutationObserver(() => place()).observe(el, { childList: true, subtree: true });

    thumb.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        e.stopPropagation();
        thumb.setPointerCapture(e.pointerId);
        dragging = { y: e.clientY, top: el.scrollTop };
    });
    thumb.addEventListener('pointermove', (e) => {
        if (!dragging) return;
        const ratio = (el.scrollHeight - el.clientHeight) / (el.clientHeight - thumb.offsetHeight - track[0] - track[1]);
        el.scrollTop = dragging.top + (e.clientY - dragging.y) * ratio;
    });
    thumb.addEventListener('pointerup', () => {
        dragging = null;
        show();
    });
    place();
},
```

A resizable textarea carries the same thumb, with its track ending above a `data-ui="grip"` drag handle drawn as two diagonal strokes in the bottom-right corner. The native resize handle is `resize-none`, since it ignores the palette and the radius.

## Floating Layers

Popovers, tooltips, modals, and toasts are appended to `ui.layer()`, which returns `document.fullscreenElement` when there is one and `#layer-root` otherwise. A layer appended anywhere else is hidden behind a fullscreen element.

A layer copies the `.dark` or `.light` scope of the element that opened it, through `ui.scheme`. The layer root sits outside that subtree, so a menu opened from a pinned-dark player would otherwise wear the page theme.

One popover is open at a time. It closes on an outside `pointerdown` or Escape, flips above its anchor when the space below runs out, and a select's popover matches the trigger's width with a 220px floor.

A select is a button opening a listbox popover rather than a native `<select>`, because a native list ignores the palette and the radius. The arrow keys move the active option, Enter chooses, and a `searchable` select puts a search field above the list with fuzzy ranking.

A tooltip comes from `data-tip` on any element. On a mouse it shows after 450ms, and for 400ms after one hides the next shows at once, so sweeping along a toolbar does not wait on every button. Keyboard focus shows it at once on `:focus-visible`, and `pointerdown` or a scroll hides it. `data-tip-side` picks the side, and the tooltip flips when that side has no room.

An icon-only button carries its name as both `aria-label` and `data-tip`. A disabled control's tooltip sits on a wrapping `<span>`, since a disabled button fires no pointer events.

A modal is centred from `md` up at `max-w-md`, `max-w-2xl`, `max-w-4xl`, or `max-w-6xl`, and below `md` it is a bottom sheet with `rounded-t-3xl` that slides up. Its header holds an optional tone tile, the title, and a close icon button, its body is the scroll region, and its footer right-aligns the actions on the modal's own `mantle`. Modals stack, Escape closes the top dismissible one, and a backdrop click closes it. `ui.confirm` resolves a promise to a boolean, with the red tone for a destructive action.

A toast sits bottom centre on a phone and bottom right from `md`, dismisses itself after 3.5 seconds, and carries at most one action button.

## Sidebars and Touch

`ui.sidebar(panel, { key, min, max, initial, side })` gives a panel a drag handle on its inner edge, a double-click that resets the width, and a collapse to zero width. The width and the collapsed state persist in `localStorage` under the key, so a layout survives a reload. On a phone the sidebar is instead a drawer translated in from the edge over a `bg-crust/70` backdrop.

An action revealed on row hover is always visible below `md` through `max-md:opacity-100`, since a touch screen has no hover. A button with a label that does not fit a phone drops the label below `sm` and keeps it as the tooltip.

## Light Mode

The `.light` block sits beside the Mocha block in the palette `<style>`, and the class goes on `<html>`.

```html
<style>
  .light {
    color-scheme: light;
    --ctp-rosewater: #c06552; --ctp-flamingo: #cc5a5d; --ctp-pink: #cd48ad;
    --ctp-mauve: #8839ef;     --ctp-red: #d20f39;      --ctp-maroon: #e64553;
    --ctp-peach: #d75200;     --ctp-yellow: #b46f00;   --ctp-green: #269200;
    --ctp-teal: #008c93;      --ctp-sky: #0086bb;      --ctp-sapphire: #008a9f;
    --ctp-blue: #1e66f5;      --ctp-lavender: #5f71f8; --ctp-text: #4c4f69;
    --ctp-subtext1: #545770;  --ctp-subtext0: #5c5f77; --ctp-overlay2: #6c6f85;
    --ctp-overlay1: #737689;  --ctp-overlay0: #8c8fa1; --ctp-surface2: #bcc0cc;
    --ctp-surface1: #ccd0da;  --ctp-surface0: #dce0e8; --ctp-base: #e6e9ef;
    --ctp-mantle: #eff1f5;    --ctp-crust: #f8f9fb;
  }
</style>
```

The ground is the one color Latte lacks, a near-white `#f8f9fb`, and each step forward is one Latte rung darker. Every layer keeps the order it has in Mocha, so the layering table holds in both themes.

The text ramp runs one Latte step darker, because stock Latte `overlay1` measures 2.8:1 on a light panel and metadata has to stay legible. Mauve, red, maroon, and blue are stock Latte. The other accents are Latte hues darkened in OKLCH until a chip's text on its 15% tint clears 3:1, where stock Latte yellow measures 2.05:1.

A script directly after the `theme-color` meta applies the stored choice before the body renders, preventing a flash of the wrong theme.

```html
<script>
    if (localStorage.getItem('APP_NAME.theme') === 'light') {
        document.documentElement.classList.add('light');
        document.querySelector('meta[name=theme-color]').content = '#f8f9fb';
    }
</script>
```

`ui.theme.set(light)` toggles `.light` on `<html>`, sets `theme-color` to the new `--ctp-crust`, persists the choice, and dispatches a `themechange` event. A view that captured colors when it rendered, such as a Mermaid diagram or a chart, renders again on that event, since its SVG keeps the colors it was drawn with.

A surface that stays dark in both themes, such as a video player, carries `class="dark"`. `@theme inline` re-themes that subtree, and its floating layers follow it through `ui.scheme`.

## Media Controls

A seek or volume bar is a native range input with its fill painted as a gradient. A native track cannot take two colors, and the gradient keeps the thumb hidden until the bar is touched.

```javascript
seek({ label = 'Seek', attrs = '', cls = '' } = {}) {
    return `<input type="range" data-seek min="0" max="0" step="any" value="0" aria-label="${ui.esc(label)}" ${attrs} class="h-1 w-full appearance-none rounded-full bg-[linear-gradient(to_right,var(--fill,var(--ctp-mauve))_var(--seek,0%),var(--track,var(--ctp-surface1))_var(--seek,0%))] outline-none ${ui.cls.focus} ${cls}">`;
},
setSeek(range, value, max) {
    range.max = max;
    range.value = value;
    range.style.setProperty('--seek', `${max ? (value / max) * 100 : 0}%`);
},
```

```css
input[type=range][data-seek]::-webkit-slider-thumb { -webkit-appearance: none; appearance: none; width: 12px; height: 12px; border-radius: 9999px; background: var(--ctp-text); transform: scale(0); transition: transform .15s; }
input[type=range][data-seek]::-moz-range-thumb { width: 12px; height: 12px; border: none; border-radius: 9999px; background: var(--ctp-text); transform: scale(0); transition: transform .15s; }
input[type=range][data-seek]:is(:hover, :focus-visible, :active)::-webkit-slider-thumb { transform: scale(1); }
input[type=range][data-seek]:is(:hover, :focus-visible, :active)::-moz-range-thumb { transform: scale(1); }
```

Progress fills in mauve and volume fills with a neutral text token through `--fill`, so only the bar that tracks playback carries the accent. A bar sitting next to an icon is wrapped in `flex items-center` so its centre lines up with the icon's.

The primary transport button is a `size-12 rounded-full bg-mauve text-crust` play button between ghost previous and next buttons, centred in a `grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]` row. A document or media frame sits in a `base` well, since white PDF pages vanish against a near-white ground.

## Base Styles

The rest of the hand-written block covers states Tailwind does not reach, and the animations the kit names in `[animation:...]` utilities.

```css
::selection { background: color-mix(in oklab, var(--ctp-mauve) 35%, transparent); color: var(--ctp-text); }
input[type=number] { appearance: textfield; }
input[type=number]::-webkit-inner-spin-button, input[type=number]::-webkit-outer-spin-button { -webkit-appearance: none; margin: 0; }
input:-webkit-autofill { -webkit-text-fill-color: var(--ctp-text); box-shadow: 0 0 0 100px var(--ctp-surface0) inset; caret-color: var(--ctp-text); }

@keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }
@keyframes pop-in { from { opacity: 0; transform: translateY(6px) scale(0.98); } to { opacity: 1; transform: none; } }
@keyframes sheet-in { from { transform: translateY(100%); } to { transform: none; } }
@keyframes toast-in { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: none; } }
@keyframes shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
```

The autofill rule paints an autofilled field back to `surface0` with `text`, because the browser's own autofill colors ignore the palette in both themes.
