# Music

**One `Audio` element plays a queue behind a track list, a docked bar, and a now-playing sheet, and each of them is redrawn from the player's state.**

## Transport Primitives

The video player uses this section too.

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

clock(sec) {
    const t = Math.max(0, Math.floor(sec || 0));
    const h = Math.floor(t / 3600);
    const m = Math.floor(t / 60) % 60;
    const s = String(t % 60).padStart(2, '0');
    return h ? `${h}:${String(m).padStart(2, '0')}:${s}` : `${m}:${s}`;
},
```

```css
input[type=range][data-seek]::-webkit-slider-thumb { -webkit-appearance: none; appearance: none; width: 12px; height: 12px; border-radius: 9999px; background: var(--ctp-text); transform: scale(0); transition: transform .15s; }
input[type=range][data-seek]::-moz-range-thumb { width: 12px; height: 12px; border: none; border-radius: 9999px; background: var(--ctp-text); transform: scale(0); transition: transform .15s; }
input[type=range][data-seek]:is(:hover, :focus-visible, :active)::-webkit-slider-thumb { transform: scale(1); }
input[type=range][data-seek]:is(:hover, :focus-visible, :active)::-moz-range-thumb { transform: scale(1); }
@keyframes eq { 0%, 100% { transform: scaleY(0.3); } 50% { transform: scaleY(1); } }
```

Progress fills in mauve and volume fills with a neutral text token through `--fill`, so only the bar that tracks playback carries the accent. A bar next to an icon is wrapped in `flex items-center`, so its center lines up with the icon's.

```javascript
const $$ = (sel, root = document) => root.querySelectorAll(sel);
const fillIcon = (name, cls) => `<i data-lucide="${name}" class="${cls} fill-current"></i>`;
const playIcons = (key, cls) => `<span data-when="${key}-paused" class="inline-flex">${fillIcon('play', `${cls} translate-x-px`)}</span><span data-when="${key}-playing" class="inline-flex" hidden>${fillIcon('pause', cls)}</span>`;
const showWhen = (key, playing) => {
    $$(`[data-when="${key}-playing"]`).forEach((e) => { e.hidden = !playing; });
    $$(`[data-when="${key}-paused"]`).forEach((e) => { e.hidden = playing; });
};

const transport = (icon, action, tip, { size = 'size-10', iconCls = 'size-[18px]', fill = true, toggle = false, attrs = '' } = {}) => `<button type="button" data-action="${action}" data-tip="${tip}" aria-label="${tip}" ${toggle ? 'aria-pressed="false"' : ''} ${attrs} class="inline-flex shrink-0 items-center justify-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent ${toggle ? 'text-overlay1 hover:bg-surface0 aria-pressed:text-mauve' : 'text-subtext0 hover:bg-surface0 hover:text-text'} ${ui.cls.focus} ${size}"><i data-lucide="${icon}" class="${iconCls} ${fill ? 'fill-current' : ''}"></i></button>`;
const playButton = (key, action, size, iconCls) => `<button type="button" data-action="${action}" aria-label="Play or pause" class="inline-flex shrink-0 items-center justify-center rounded-full bg-mauve text-crust transition-colors hover:bg-mauve/85 ${ui.cls.focus} ${size}">${playIcons(key, iconCls)}</button>`;

const equalizer = () => `<span class="inline-flex h-3.5 items-end gap-[2px] text-mauve">${['0s', '-0.45s', '-0.25s'].map((d) => `<span class="h-full w-[3px] origin-bottom rounded-full bg-current [animation:eq_.9s_ease-in-out_${d}_infinite] group-[.paused]:[animation-play-state:paused]"></span>`).join('')}</span>`;
```

Each play button renders both icons, and `showWhen` swaps them for every button of one source at once. `transport` is the ghost button beside play, and `playButton` is the mauve one. The equalizer's bars run while a row has `.playing` and freeze while it also has `.paused`.

## Data Shape

```javascript
const MEDIA = {
    album: { title: 'Quiet Rooms', artist: 'Mock Ensemble', year: 2026, genre: 'Ambient', cover: '/static/media/cover-1.svg' },
    tracks: [['First Light', 1], ['Room Tone', 2, 'Mock Ensemble, Ines Var']]
        .map(([title, n, artist, lyrics = true], id) => ({ id, title, artist: artist || 'Mock Ensemble', album: 'Quiet Rooms', src: `/static/media/track-${n}.m4a`, cover: '/static/media/cover-1.svg', duration: 40 + n * 7, lyrics })),
    lyrics: `The hum under the floor
counts the hours we keep`,
    videos: [
        { title: 'Test pattern', meta: 'testsrc2 · 720p · captions', src: '/static/media/clip-1.mp4', poster: '/static/media/poster-1.jpg', captions: '/static/media/clip-1.vtt', duration: 40 },
        { title: 'Colour bars', meta: 'smptehdbars · 720p', src: '/static/media/clip-2.mp4', poster: '/static/media/poster-2.jpg', duration: 25 },
    ],
    book: { title: 'Field Notes on Quiet Rooms', src: '/static/media/field-notes.pdf' },
};
```

A track carries its own `duration`, so the list and the bar show a length before the audio's metadata loads. `videos` and `book` feed the video and reader views.

## The Player

```javascript
const player = {
    audio: new Audio(),
    source: [],
    queue: [],
    idx: -1,
    shuffle: false,
    repeat: false,
    seeking: false,
    tab: 'lyrics',
    sheet: null,

    get track() {
        return player.queue[player.idx];
    },

    play(tracks, idx) {
        player.source = tracks.slice();
        player.queue = tracks.slice();
        player.shuffle = false;
        player.start(idx);
    },

    start(idx) {
        player.idx = idx;
        player.audio.src = player.track.src;
        player.audio.play().catch(() => {});
        player.render();
    },

    toggle() {
        if (!player.track) return;
        if (player.audio.paused) player.audio.play().catch(() => {});
        else player.audio.pause();
    },

    step(delta) {
        if (!player.track) return;
        if (delta < 0 && player.audio.currentTime > 3) {
            player.audio.currentTime = 0;
            return;
        }
        const next = player.idx + delta;
        if (next >= 0 && next < player.queue.length) player.start(next);
        else if (delta > 0 && player.repeat) player.start(0);
    },

    close() {
        player.sheet?.close();
        player.audio.pause();
        player.audio.removeAttribute('src');
        player.audio.load();
        player.queue = [];
        player.idx = -1;
        player.render();
    },

    toggleShuffle() {
        const current = player.track;
        player.shuffle = !player.shuffle;
        if (player.shuffle) {
            const rest = player.source.filter((t) => t !== current);
            for (let i = rest.length - 1; i > 0; i--) {
                const j = Math.floor(Math.random() * (i + 1));
                [rest[i], rest[j]] = [rest[j], rest[i]];
            }
            player.queue = [current, ...rest];
            player.idx = 0;
        } else {
            player.queue = player.source.slice();
            player.idx = player.queue.indexOf(current);
        }
        player.render();
    },
```

`source` keeps the album order and `queue` the play order, so turning shuffle off restores the album around the current track. Shuffle moves the current track to the front. Previous restarts a track that is past three seconds, and next wraps to the start only with repeat on.

```javascript
    render() {
        const t = player.track;
        $$('[data-player-bar]').forEach((b) => { b.hidden = !t; });
        $$('[data-track]').forEach((row) => row.classList.toggle('playing', t?.id === Number(row.dataset.track)));
        $$('[data-action="np-shuffle"]').forEach((b) => b.setAttribute('aria-pressed', String(player.shuffle)));
        $$('[data-action="np-repeat"]').forEach((b) => b.setAttribute('aria-pressed', String(player.repeat)));
        if (t) {
            $$('[data-player="title"]').forEach((e) => { e.textContent = t.title; });
            $$('[data-player="subtitle"]').forEach((e) => { e.textContent = `${t.artist} · ${t.album}`; });
            $$('[data-player="cover"]').forEach((e) => {
                if (e.dataset.src === t.cover) return;
                e.dataset.src = t.cover;
                e.innerHTML = `<img src="${t.cover}" alt="" class="size-full object-cover">`;
            });
            const lyrics = document.querySelector('[data-np-lyrics]');
            if (lyrics) {
                lyrics.innerHTML = t.lyrics ? ui.esc(MEDIA.lyrics) : ui.empty({ icon: 'mic-vocal', title: 'No lyrics', hint: 'This track has no lyrics file.' });
                ui.icons(lyrics);
            }
            player.renderQueue();
        }
        player.renderState();
        player.renderProgress();
    },

    renderState() {
        const playing = Boolean(player.track) && !player.audio.paused;
        showWhen('music', playing);
        $$('[data-track], [data-queue]').forEach((row) => row.classList.toggle('paused', !playing));
    },

    renderProgress() {
        if (!player.track || player.seeking) return;
        const d = Number.isFinite(player.audio.duration) ? player.audio.duration : player.track.duration;
        const c = Math.min(player.audio.currentTime || 0, d);
        $$('[data-player="seek"]').forEach((r) => ui.setSeek(r, c, d));
        $$('[data-player="progress"]').forEach((b) => { b.style.width = `${d ? (c / d) * 100 : 0}%`; });
        $$('[data-player="time"]').forEach((e) => { e.textContent = ui.clock(c); });
        $$('[data-player="duration"]').forEach((e) => { e.textContent = ui.clock(d); });
    },

    renderVolume() {
        const a = player.audio;
        $$('[data-player="volume"]').forEach((r) => ui.setSeek(r, a.muted ? 0 : a.volume, 1));
        $$('[data-player="mute"]').forEach((b) => {
            b.innerHTML = `<i data-lucide="${a.muted || a.volume === 0 ? 'volume-x' : 'volume-2'}" class="size-4"></i>`;
            b.dataset.tip = a.muted ? 'Unmute' : 'Mute';
            ui.icons(b);
        });
    },

    renderQueue() {
        const list = document.querySelector('[data-np-queue]');
        if (!list) return;
        list.innerHTML = player.queue.map((t, i) => {
            const current = i === player.idx;
            return `<li data-queue="${i}" class="group ${current ? 'playing' : ''} flex items-center gap-2.5 rounded-xl py-1 pr-1 pl-1 transition-colors hover:bg-surface0/50">
                ${current ? `<span class="grid w-7 shrink-0 place-items-center">${equalizer()}</span>` : '<button type="button" data-queue-handle aria-label="Drag to reorder" class="grid h-9 w-7 shrink-0 cursor-grab touch-none place-items-center rounded-lg text-overlay0 transition-colors hover:text-text active:cursor-grabbing"><i data-lucide="grip-vertical" class="size-4"></i></button>'}
                <span class="size-10 shrink-0 overflow-hidden rounded-xl bg-surface0"><img src="${t.cover}" alt="" class="size-full object-cover"></span>
                <button type="button" data-action="np-queue-play" data-index="${i}" class="min-w-0 flex-1 rounded-lg py-0.5 text-left ${ui.cls.focus}">
                    <span class="block truncate text-sm ${current ? 'text-mauve' : 'text-text'}">${ui.esc(t.title)}</span>
                    <span class="block truncate text-xs text-overlay1">${ui.esc(t.artist)}</span>
                </button>
                <span class="shrink-0 text-xs tabular-nums text-overlay1">${ui.clock(t.duration)}</span>
                ${current ? '<span class="w-8 shrink-0"></span>' : ui.iconButton('x', { tip: 'Remove from queue', size: 'sm', attrs: `data-action="np-queue-remove" data-index="${i}"` })}
            </li>`;
        }).join('');
        ui.icons(list);
    },
```

Every view of the player is found by a `data-player` attribute and redrawn from the player's state, so the bar, the sheet, and the track rows never disagree. `seeking` holds off `timeupdate` while the user drags, or the thumb would jump back under the pointer.

```javascript
    openSheet(tab = player.tab) {
        if (!player.track) return;
        player.tab = tab;
        if (!player.sheet) {
            player.sheet = ui.modal.open({
                title: 'Now playing',
                subtitle: `From ${ui.esc(MEDIA.album.title)}`,
                icon: 'disc-3',
                size: 'lg',
                body: `<div class="grid gap-6 pb-2 md:grid-cols-[minmax(0,18rem)_minmax(0,1fr)] md:gap-8">
                    <div class="mx-auto flex w-full max-w-72 flex-col gap-5 md:mx-0">
                        <span data-player="cover" class="block aspect-square w-full overflow-hidden rounded-2xl bg-base"></span>
                        <div class="min-w-0"><p data-player="title" class="truncate font-display text-xl font-medium text-text"></p><p data-player="subtitle" class="truncate text-sm text-overlay1"></p></div>
                        <div class="flex flex-col gap-2">${ui.seek({ attrs: 'data-player="seek"' })}<div class="flex justify-between text-[11px] tabular-nums text-overlay1"><span data-player="time">0:00</span><span data-player="duration">0:00</span></div></div>
                        <div class="flex items-center justify-between">
                            ${transport('shuffle', 'np-shuffle', 'Shuffle', { fill: false, toggle: true })}
                            ${transport('skip-back', 'np-prev', 'Previous', { size: 'size-11', iconCls: 'size-5' })}
                            ${playButton('music', 'np-toggle', 'size-14', 'size-6')}
                            ${transport('skip-forward', 'np-next', 'Next', { size: 'size-11', iconCls: 'size-5' })}
                            ${transport('repeat', 'np-repeat', 'Repeat', { fill: false, toggle: true })}
                        </div>
                    </div>
                    <div class="flex min-w-0 flex-col gap-3">
                        <span class="max-md:self-center">${ui.segmented({ name: 'np-tab', value: tab, options: [{ value: 'lyrics', label: 'Lyrics', icon: 'mic-vocal' }, { value: 'queue', label: 'Queue', icon: 'list-music' }] })}</span>
                        <div class="relative">
                            <div data-np-lyrics data-scroll class="h-72 overflow-y-auto rounded-2xl bg-base px-5 py-4 text-sm leading-7 whitespace-pre-line text-subtext1 select-text md:h-[25rem]"></div>
                            <ul data-np-queue data-scroll="lane" class="flex h-72 flex-col gap-0.5 overflow-y-auto rounded-2xl bg-base p-2 md:h-[25rem]"></ul>
                        </div>
                    </div>
                </div>`,
                onClose: () => { player.sheet = null; },
            });
            const el = player.sheet.el;
            el.addEventListener('ui-change', (e) => {
                if (e.detail.name !== 'np-tab') return;
                player.tab = e.detail.value;
                player.showTab();
            });
            player.dragQueue(el.querySelector('[data-np-queue]'));
            player.render();
            ui.icons(el);
        }
        player.showTab();
    },

    showTab() {
        const el = player.sheet?.el;
        if (!el) return;
        el.querySelector('[data-np-lyrics]').hidden = player.tab !== 'lyrics';
        el.querySelector('[data-np-queue]').hidden = player.tab !== 'queue';
        el.querySelectorAll('[data-ui="radio"]').forEach((r) => {
            const on = r.dataset.value === player.tab;
            r.setAttribute('aria-checked', String(on));
            r.tabIndex = on ? 0 : -1;
        });
    },

    dragQueue(list) {
        list.addEventListener('pointerdown', (e) => {
            const handle = e.target.closest('[data-queue-handle]');
            if (!handle) return;
            e.preventDefault();
            const row = handle.closest('[data-queue]');
            row.classList.add('bg-surface0');
            const move = (ev) => {
                const over = [...list.children].find((r) => r !== row && ev.clientY < r.getBoundingClientRect().top + r.offsetHeight / 2);
                if (!over) list.appendChild(row);
                else if (over !== row.nextElementSibling) list.insertBefore(row, over);
            };
            const up = () => {
                document.removeEventListener('pointermove', move);
                document.removeEventListener('pointerup', up);
                document.removeEventListener('pointercancel', up);
                const order = [...list.children].map((r) => Number(r.dataset.queue));
                const current = player.track;
                player.queue = order.map((i) => player.queue[i]);
                player.idx = player.queue.indexOf(current);
                player.renderQueue();
                player.renderState();
            };
            document.addEventListener('pointermove', move);
            document.addEventListener('pointerup', up);
            document.addEventListener('pointercancel', up);
        });
    },
};
```

The now-playing sheet is an `lg` modal with the cover, the title, the seek bar, and the transport on the left, and a Lyrics and Queue track over `base` wells on the right. Each queue row has a drag handle and a remove button, and the current row shows the equalizer in their place.

A drag moves the row in the DOM while the pointer moves, and rebuilds `queue` from the new order on release, so the current track keeps playing.

```javascript
player.audio.preload = 'auto';
player.audio.addEventListener('timeupdate', () => player.renderProgress());
player.audio.addEventListener('durationchange', () => player.renderProgress());
player.audio.addEventListener('play', () => { vp.el?.pause(); player.renderState(); });
player.audio.addEventListener('pause', () => player.renderState());
player.audio.addEventListener('ended', () => player.step(1));
player.audio.addEventListener('volumechange', () => player.renderVolume());

document.addEventListener('input', (e) => {
    const r = e.target;
    if (r.matches?.('[data-player="seek"]')) {
        player.seeking = true;
        ui.setSeek(r, Number(r.value), Number(r.max));
        $$('[data-player="time"]').forEach((t) => { t.textContent = ui.clock(r.value); });
    } else if (r.matches?.('[data-player="volume"]')) {
        player.audio.muted = false;
        player.audio.volume = Number(r.value);
    }
});
document.addEventListener('change', (e) => {
    if (!e.target.matches?.('[data-player="seek"]')) return;
    player.seeking = false;
    player.audio.currentTime = Number(e.target.value);
    player.renderProgress();
});

ui.on('np-toggle', () => player.toggle());
ui.on('np-prev', () => player.step(-1));
ui.on('np-next', () => player.step(1));
ui.on('np-open', () => player.openSheet());
ui.on('np-lyrics', () => player.openSheet('lyrics'));
ui.on('np-queue', () => player.openSheet('queue'));
ui.on('np-close', () => player.close());
ui.on('np-shuffle', () => player.toggleShuffle());
ui.on('np-repeat', () => { player.repeat = !player.repeat; player.render(); });
ui.on('np-mute', () => { player.audio.muted = !player.audio.muted; });
ui.on('np-queue-play', (b) => player.start(Number(b.dataset.index)));
ui.on('np-queue-remove', (b) => {
    const i = Number(b.dataset.index);
    player.queue.splice(i, 1);
    if (i < player.idx) player.idx--;
    player.render();
});
ui.on('album-play', () => (player.track ? player.toggle() : player.play(MEDIA.tracks, 0)));
ui.on('album-shuffle', () => {
    player.play(MEDIA.tracks, Math.floor(Math.random() * MEDIA.tracks.length));
    player.toggleShuffle();
});
ui.on('track-play', (row) => {
    const i = Number(row.dataset.index);
    if (player.track?.id === MEDIA.tracks[i].id) player.toggle();
    else player.play(MEDIA.tracks, i);
});
ui.on('track-lyrics', (b) => {
    const i = Number(b.dataset.index);
    if (player.track?.id !== MEDIA.tracks[i].id) player.play(MEDIA.tracks, i);
    player.openSheet('lyrics');
});
```

Audio starting pauses the video element, and the video starting pauses the audio, so one source plays at a time. The seek bar previews the time on `input` and seeks only on `change`.

## Track List

```javascript
music() {
    const a = MEDIA.album;
    const total = MEDIA.tracks.reduce((s, t) => s + t.duration, 0);
    const rows = MEDIA.tracks.map((t, i) => `<li data-track="${t.id}" data-action="track-play" data-index="${i}" class="group grid cursor-pointer grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-3 rounded-2xl px-3 py-2 transition-colors hover:bg-surface0/50">
            <span class="grid place-items-center text-sm tabular-nums text-overlay1">
                <span class="group-hover:hidden group-[.playing]:hidden">${i + 1}</span>
                <span class="hidden text-text group-[:hover:not(.playing)]:inline-flex">${fillIcon('play', 'size-3.5')}</span>
                <span class="hidden group-[.playing]:inline-flex">${equalizer()}</span>
            </span>
            <span class="min-w-0"><span class="block truncate text-sm text-text group-[.playing]:text-mauve">${ui.esc(t.title)}</span>${t.artist !== a.artist ? `<span class="block truncate text-xs text-overlay1">${ui.esc(t.artist)}</span>` : ''}</span>
            <span class="flex items-center gap-1"><span class="flex size-8 ${ui.cls.reveal}">${t.lyrics ? ui.iconButton('mic-vocal', { tip: 'Lyrics', size: 'sm', attrs: `data-action="track-lyrics" data-index="${i}"` }) : ''}</span><span class="w-10 text-right text-xs tabular-nums text-overlay1">${ui.clock(t.duration)}</span></span>
        </li>`).join('');
    return `<section class="flex min-h-0 flex-1 flex-col rounded-3xl bg-mantle">
        <div data-scroll="lane" class="min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
            <header class="mb-6 flex flex-col items-center gap-5 sm:flex-row sm:items-end sm:gap-7">
                <img src="${a.cover}" alt="" class="size-44 shrink-0 rounded-2xl sm:size-48 lg:size-56">
                <div class="flex min-w-0 flex-col items-center gap-1.5 text-center sm:items-start sm:text-left">
                    <span class="${ui.cls.label}">Album</span>
                    <h2 class="font-display text-3xl font-medium text-text lg:text-4xl">${ui.esc(a.title)}</h2>
                    <p class="text-sm text-subtext0">${ui.esc(a.artist)} · ${a.year} · ${a.genre}</p>
                    <p class="text-xs text-overlay1">${MEDIA.tracks.length} tracks · ${Math.round(total / 60)} min</p>
                    <div class="mt-3 flex items-center gap-2">
                        <button type="button" data-action="album-play" class="inline-flex h-10 items-center gap-2 rounded-full bg-mauve px-5 text-sm font-medium text-crust transition-colors hover:bg-mauve/85 ${ui.cls.focus}">${playIcons('music', 'size-4')}<span data-when="music-paused">Play</span><span data-when="music-playing" hidden>Pause</span></button>
                        ${ui.iconButton('shuffle', { tip: 'Shuffle play', variant: 'secondary', attrs: 'data-action="album-shuffle"' })}
                        ${ui.iconButton('ellipsis', { tip: 'More', attrs: 'data-action="more-menu"' })}
                    </div>
                </div>
            </header>
            <div class="grid grid-cols-[2rem_minmax(0,1fr)_auto] gap-3 px-3 pb-2 ${ui.cls.label}"><span class="text-center">#</span><span>Title</span><span class="flex w-10 justify-end"><i data-lucide="clock" class="size-3.5"></i></span></div>
            <ul class="flex flex-col gap-0.5">${rows}</ul>
        </div>
    </section>`;
},
```

A track row shows its number, a play icon on hover, and the equalizer while it plays, with the title in mauve. A second line names the artist only when it differs from the album's.

## The Media Page

```javascript
const mediaPage = {
    nav: ['Media', 'circle-play'],
    tab: 'music',
    render() {
        return `<div class="flex h-full min-h-0 flex-col">
            ${pageHead('Media', ui.segmented({ name: 'media-tab', value: mediaPage.tab, ground: true, options: [{ value: 'music', label: 'Music', icon: 'music', compact: true }, { value: 'video', label: 'Video', icon: 'clapperboard', compact: true }, { value: 'reader', label: 'Reader', icon: 'book-open', compact: true }] }))}
            <div data-media-body class="flex min-h-0 flex-1 flex-col"></div>
            <div data-player-bar hidden class="shrink-0 pt-3 sm:pt-4">
                <div class="relative flex items-center gap-2 overflow-hidden rounded-3xl bg-mantle p-2 pr-3 md:grid md:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_minmax(0,1fr)] md:gap-6 md:p-3">
                    <button type="button" data-action="np-open" aria-label="Open the player" class="flex min-w-0 flex-1 items-center gap-3 rounded-2xl text-left ${ui.cls.focus}">
                        <span data-player="cover" class="size-11 shrink-0 overflow-hidden rounded-xl bg-base md:size-14 md:rounded-2xl"></span>
                        <span class="min-w-0"><span data-player="title" class="block truncate text-sm font-medium text-text"></span><span data-player="subtitle" class="block truncate text-xs text-overlay1"></span></span>
                    </button>
                    <div class="flex items-center gap-1 md:flex-col md:gap-1">
                        <div class="flex items-center gap-1 md:gap-3">
                            <span class="max-md:hidden">${transport('skip-back', 'np-prev', 'Previous')}</span>
                            ${playButton('music', 'np-toggle', 'size-10', 'size-[18px]')}
                            ${transport('skip-forward', 'np-next', 'Next')}
                        </div>
                        <div class="flex w-full items-center gap-2 max-md:hidden">
                            <span data-player="time" class="w-10 shrink-0 text-right text-[11px] tabular-nums text-overlay1">0:00</span>
                            ${ui.seek({ attrs: 'data-player="seek"' })}
                            <span data-player="duration" class="w-10 shrink-0 text-[11px] tabular-nums text-overlay1">0:00</span>
                        </div>
                    </div>
                    <div class="flex items-center justify-end gap-1 max-md:hidden">
                        ${ui.iconButton('mic-vocal', { tip: 'Lyrics', attrs: 'data-action="np-lyrics"' })}
                        ${ui.iconButton('list-music', { tip: 'Queue', attrs: 'data-action="np-queue"' })}
                        ${ui.iconButton('volume-2', { tip: 'Mute', attrs: 'data-action="np-mute" data-player="mute"' })}
                        <span class="flex w-20 shrink-0 items-center max-lg:hidden">${ui.seek({ label: 'Volume', attrs: 'data-player="volume" style="--fill: var(--ctp-subtext0)"' })}</span>
                        ${ui.iconButton('x', { tip: 'Close the player', attrs: 'data-action="np-close"' })}
                    </div>
                    <div class="absolute inset-x-5 bottom-0 h-0.5 overflow-hidden rounded-full bg-surface0 md:hidden"><div data-player="progress" class="h-full w-0 bg-mauve"></div></div>
                </div>
            </div>
        </div>`;
    },

    mount(el) {
        const body = el.querySelector('[data-media-body]');
        const show = (tab) => {
            mediaPage.tab = tab;
            reader.close();
            vp.detach();
            body.innerHTML = mediaPage[tab]();
            ui.icons(body);
            if (tab === 'video') vp.mount(body.querySelector('[data-video-player]'));
            if (tab === 'reader') reader.open(body);
            player.render();
        };
        el.addEventListener('ui-change', (e) => { if (e.detail.name === 'media-tab') show(e.detail.value); });
        show(mediaPage.tab);
        player.renderVolume();
    },

    leave() {
        player.sheet?.close();
        player.audio.pause();
        reader.close();
        vp.detach();
    },
};
```

`music()`, `video()`, and `reader()` are members of `mediaPage` between `tab` and `render`.

The page head switches tabs on a ground track, and the body swaps per tab after closing the reader and detaching the video. The docked bar sits outside the body, so music keeps playing across tabs. From `md` it is a three-column grid, and on a phone it shrinks to the cover, the title, play, and next over a 2px progress line.

Leaving the page closes the sheet and stops every source.
