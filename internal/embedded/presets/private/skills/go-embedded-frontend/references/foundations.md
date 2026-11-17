# Foundations

**Every color is a `--ctp-*` token picked by role, every size comes from one scale, and hand-written CSS covers only what Tailwind cannot reach.**

## Color Roles

The text ramp carries the hierarchy, so a view reads in order before any accent appears.

| Token | Role |
|---|---|
| `text` | headings, and the one value a row exists to show |
| `subtext1` | emphasized body |
| `subtext0` | body copy |
| `overlay1` | labels, metadata, and idle icons |
| `overlay0` | placeholders, disabled text, and sidebar group labels |

Each accent carries one meaning, so a color never says two things in one app.

| Accent | Meaning |
|---|---|
| `mauve` | the primary action, the active item, and focus |
| `green` | success, running |
| `red` | danger, error |
| `peach` | warning |
| `yellow` | pending |
| `blue` | info, links |
| the rest | labels only, such as a tag color the user picks |

## Fills and States

A control at rest is `surface0`, and it steps to `surface1` on hover, while open, or when pressed. Anything nested inside a `surface0` control takes `surface1` for the same reason: a chip in a tag field, a `kbd` in a search field, and a stepper's buttons all have to read against the control around them.

A list row hovers at `surface0/50` and is selected at `surface0`, with its icon and count turning mauve. A hover at the selected fill would make the row under the pointer indistinguishable from the chosen one.

A menu item or a listbox option hovers at full `surface0`. A menu has no selected row to confuse with, and the highlight is what the arrow keys move.

A tile, stat block, or card inside a panel is `surface0/50` and keeps that fill on hover unless the whole tile is the click target. A `surface0` hover swallows the `surface0` chips and buttons inside it.

An accent tint is `/15` on a small element such as a chip, a toned pill, or an icon tile, and `/10` on a large area such as a banner, a callout, an overdue row, or a selected choice card. A large area at `/15` reads as a colored panel rather than a marked one.

Shadows sit only under floating layers, as `shadow-shade/N`, and nothing glows. Elevation inside the page is carried by tone alone, so a shadow always means the element floats.

`shade` is the one token outside Catppuccin's names. It is `crust` in Mocha and a translucent `text` in the light palette. The light `crust` is near-white, so a shadow in it veils whatever sits under a popover.

Every interactive element takes `ui.cls.focus`, and a text field shows the same ring on plain `focus`. Mauve marks focus, the primary action, and the selected icon, which is why no structural fill borrows it.

An action revealed on hover takes `ui.cls.reveal`, which also shows it while it holds focus or its menu is open, and always below `md`. A trigger that hides while focused or while its menu is open leaves the user acting on a control they cannot see. A touch screen has no hover at all. A button whose label does not fit a phone drops it below `sm` and keeps it as the tooltip.

## Type

| Text | Classes |
|---|---|
| Page title | `font-display text-2xl text-text` |
| Modal title | `font-display text-lg font-medium text-text` |
| Section or panel title | `font-display text-[16px] font-medium text-text` |
| Field label | `ui.cls.label` |
| Sidebar group label | `ui.cls.group` |
| Body | `text-sm text-subtext0` |
| Hint and metadata | `text-xs text-overlay1` |
| A path, an ID, or a hash | `font-mono` |
| A number in a column, a count, or a clock | `tabular-nums` |

## Radius

| Radius | Used by |
|---|---|
| `rounded-full` | buttons, icon buttons, chips, counts, pill tracks and segments, the nav, search fields, toasts |
| `rounded-3xl` | panels, cards, modals |
| `rounded-2xl` | inputs, selects, rows, popovers, wells, banners, `md` and `lg` tiles |
| `rounded-xl` | menu items, listbox options, tooltips, sidebar rows, `sm` tiles |
| `rounded-lg` | checkboxes |
| `rounded-md` | `kbd` |

## Icons

| Size | Used by |
|---|---|
| `size-3` | chips |
| `size-3.5` | small buttons, steppers |
| `size-4` | the default: buttons, inputs, rows, menus |
| `size-5` | `md` tiles, a modal header, an empty state |
| `size-6` | `lg` tiles |

## Base Styles

The hand-written block covers states Tailwind does not reach and the animations the kit names in `[animation:...]` utilities. It sits in the palette `<style>`, after the theme blocks.

```css
* { scrollbar-width: none; }
*::-webkit-scrollbar { display: none; }
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

Native scrollbars are hidden page-wide, because every scroll region draws its own thumb. The autofill rule paints an autofilled field back to `surface0` with `text`, because the browser's own autofill colors ignore the palette in both themes.

## Light Mode

A light mode exists only when it is asked for. Mocha stays on `:root`, so a page without the `.light` class on `<html>` is dark.

The `.light` block sits beside the Mocha block in the palette `<style>`.

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
    --ctp-mantle: #eff1f5;    --ctp-crust: #f8f9fb;     --ctp-shade: rgb(76 79 105 / 0.25);
  }
</style>
```

The ground is the one color Latte lacks, a near-white `#f8f9fb`, and each step forward is one Latte rung darker. Every layer keeps the order it has in Mocha, so the layering table holds in both themes.

The text ramp runs one Latte step darker, because stock Latte `overlay1` measures 2.8:1 on a light panel and metadata has to stay legible. Mauve, red, maroon, and blue are stock Latte. The other accents are Latte hues darkened in OKLCH until a chip's text on its 15% tint clears 3:1, where stock Latte yellow measures 2.05:1.

A script directly after the `theme-color` meta applies the stored choice before the body renders, which prevents a flash of the wrong theme.

```html
<script>
    if (localStorage.getItem('APP_NAME.theme') === 'light') {
        document.documentElement.classList.add('light');
        document.querySelector('meta[name=theme-color]').content = '#f8f9fb';
    }
</script>
```

`ui.theme.set(light)` toggles `.light` on `<html>`, sets `theme-color` to the new `--ctp-crust`, persists the choice, and dispatches `themechange`. A view that captured colors when it rendered, such as a Mermaid diagram or a chart, renders again on that event, since its SVG keeps the colors it was drawn with.

```javascript
theme: {
    light: () => document.documentElement.classList.contains('light'),

    set(light) {
        document.documentElement.classList.toggle('light', light);
        document.querySelector('meta[name=theme-color]').content = getComputedStyle(document.documentElement).getPropertyValue('--ctp-crust').trim();
        try { localStorage.setItem('APP_NAME.theme', light ? 'light' : 'dark'); } catch {}
        window.dispatchEvent(new CustomEvent('themechange', { detail: { light } }));
    },
},
```

The toggle is a ghost icon button after the nav, showing the icon of the theme it switches to. It renders through `app.renderTheme`, a member of the router in layout.md, and a view re-renders through its `theme` hook.

```html
<span id="theme-toggle" class="max-sm:ml-auto"></span>
```

```javascript
ui.on('theme-toggle', () => {
    ui.theme.set(!ui.theme.light());
    app.renderTheme();
});
window.addEventListener('themechange', () => pages[app.current].theme?.(document.getElementById('view')));
```

A surface that stays dark in both themes, such as a video player, carries `class="dark"`. `@theme inline` re-themes that subtree, and its floating layers follow it through `ui.scheme`.
