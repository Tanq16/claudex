# Video

**The video player is a pinned-dark surface whose controls fade after three seconds of stillness and come back on any movement, tap, or key.**

It builds on the transport primitives in media-music.md and reads `MEDIA.videos`.

## The View

```javascript
video() {
    const btn = (icon, action, tip, { attrs = '', iconCls = 'size-[18px]' } = {}) => `<button type="button" data-action="${action}" data-tip="${tip}" aria-label="${tip}" ${attrs} class="inline-flex size-10 shrink-0 items-center justify-center rounded-full text-text transition-colors hover:bg-surface0/50 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent aria-pressed:text-mauve ${ui.cls.focus}"><i data-lucide="${icon}" class="${iconCls}"></i></button>`;
    const track = 'style="--track: color-mix(in oklab, var(--ctp-text) 22%, transparent)"';
    const fade = 'transition-opacity duration-200 group-[.idle]/player:pointer-events-none group-[.idle]/player:opacity-0';
    return `<div class="flex min-h-0 flex-1 flex-col gap-4 xl:flex-row">
        <section data-video-player tabindex="0" class="dark group/player relative aspect-video max-h-[65%] w-full shrink-0 overflow-hidden rounded-3xl bg-crust text-text outline-none select-none focus-visible:ring-2 focus-visible:ring-mauve/40 xl:aspect-auto xl:max-h-none xl:min-h-0 xl:flex-1 [&.idle]:cursor-none [&:fullscreen]:rounded-none [&:fullscreen]:bg-black">
            <div class="absolute inset-0 flex items-center justify-center [container-type:size]">
                <div data-video-frame class="relative aspect-[var(--ar)] w-[min(100cqw,calc(100cqh*var(--ar)))] [--ar:1.7778]"><video playsinline preload="metadata" class="size-full bg-black object-contain"></video></div>
            </div>
            <div data-video-captions hidden class="pointer-events-none absolute inset-x-0 bottom-32 flex flex-col items-center gap-1 px-6 text-center transition-[bottom] duration-200 group-[.idle]/player:bottom-6"></div>
            <div data-video-loading hidden class="pointer-events-none absolute inset-0 grid place-items-center">${ui.spinner('size-8')}</div>
            <div data-video-flash class="pointer-events-none absolute top-1/2 -translate-y-1/2 rounded-full bg-crust/70 px-4 py-2 text-sm font-semibold tabular-nums opacity-0 backdrop-blur-sm transition-opacity duration-200"></div>
            <div data-controls class="absolute inset-x-0 top-0 flex items-center gap-3 bg-linear-to-b from-crust/80 to-transparent px-5 pt-4 pb-12 ${fade}">
                <h2 data-video-title class="min-w-0 flex-1 truncate font-display text-[16px] font-medium"></h2>
                <span data-video-index class="shrink-0 font-mono text-xs text-subtext1"></span>
            </div>
            <div data-controls class="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 bg-linear-to-t from-crust/90 via-crust/50 to-transparent px-3 pt-16 pb-3 sm:px-4 ${fade}">
                <div class="flex items-center gap-3 px-2">
                    <span data-video="time" class="min-w-10 text-right text-xs tabular-nums text-subtext1">0:00</span>
                    ${ui.seek({ attrs: `data-video="seek" ${track}` })}
                    <span data-video="duration" class="min-w-10 text-xs tabular-nums text-subtext1">0:00</span>
                </div>
                <div class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2">
                    <div class="flex min-w-0 items-center gap-1">${btn('volume-2', 'vp-mute', 'Mute')}<span class="flex w-20 items-center max-md:hidden">${ui.seek({ label: 'Volume', attrs: `data-video="volume" style="--fill: var(--ctp-text); --track: color-mix(in oklab, var(--ctp-text) 22%, transparent)"` })}</span></div>
                    <div class="flex items-center gap-1 sm:gap-3">
                        ${btn('skip-back', 'vp-prev', 'Previous', { iconCls: 'size-5 fill-current' })}
                        <button type="button" data-action="vp-toggle" aria-label="Play" class="inline-flex size-12 items-center justify-center rounded-full bg-mauve text-crust transition-colors hover:bg-mauve/85 ${ui.cls.focus}">${playIcons('video', 'size-5')}</button>
                        ${btn('skip-forward', 'vp-next', 'Next', { iconCls: 'size-5 fill-current' })}
                    </div>
                    <div class="flex items-center justify-end gap-1">
                        <button type="button" data-action="vp-speed" data-tip="Playback speed" aria-haspopup="menu" class="inline-flex h-10 min-w-10 items-center justify-center rounded-full px-2 text-sm font-semibold tabular-nums text-text transition-colors hover:bg-surface0/50 max-sm:hidden ${ui.cls.focus}">1x</button>
                        ${btn('captions', 'vp-captions', 'Captions', { attrs: 'aria-pressed="true"' })}
                        ${btn('maximize', 'vp-fullscreen', 'Full screen')}
                    </div>
                </div>
            </div>
        </section>
        <aside class="flex min-h-0 flex-1 flex-col rounded-3xl bg-mantle xl:w-80 xl:flex-none">
            <header class="flex shrink-0 items-center gap-2 px-4 pt-4 pb-2"><h2 class="flex-1 font-display text-[16px] font-medium text-text">Up next</h2>${ui.count(MEDIA.videos.length)}</header>
            <ul data-scroll data-video-list class="min-h-0 flex-1 overflow-y-auto px-2 pb-2"></ul>
        </aside>
    </div>`;
},
```

The player carries `class="dark"`, so it stays dark in both themes, and its popovers and tooltips follow it through `ui.scheme`. The frame is a container-query box that fits the video's aspect ratio inside both dimensions, and `--ar` updates from the video's metadata.

The controls sit over gradients from `crust`: the title at the top, and at the bottom the seek bar over a centered row of previous, a `size-12` mauve play button, and next. Volume drops below `md` and speed below `sm`.

Captions are drawn by the player rather than the browser, so they rise above the controls while those show and drop to the edge when they hide.

## The Controller

```javascript
const vp = {
    idx: 0,
    captions: true,
    rates: [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2],
    root: null,
    el: null,
    idleTimer: 0,
    tap: null,
    tapTimer: 0,
    flash: 0,
    flashTimer: 0,
    pointer: 'mouse',
    seeking: false,

    mount(root) {
        vp.root = root;
        vp.el = root.querySelector('video');
        const v = vp.el;
        const q = (sel) => root.querySelector(sel);

        v.addEventListener('play', () => { player.audio.pause(); vp.renderState(); vp.wake(); });
        v.addEventListener('pause', () => { vp.renderState(); vp.wake(); });
        ['timeupdate', 'durationchange', 'loadedmetadata'].forEach((t) => v.addEventListener(t, vp.renderProgress));
        v.addEventListener('loadedmetadata', () => {
            if (v.videoWidth && v.videoHeight) q('[data-video-frame]').style.setProperty('--ar', v.videoWidth / v.videoHeight);
        });
        v.addEventListener('waiting', () => { q('[data-video-loading]').hidden = false; });
        ['playing', 'canplay', 'pause', 'emptied', 'error'].forEach((t) => v.addEventListener(t, () => { q('[data-video-loading]').hidden = true; }));
        v.addEventListener('volumechange', vp.renderVolume);
        v.addEventListener('ratechange', () => { q('[data-action="vp-speed"]').textContent = `${v.playbackRate}x`; });
        v.addEventListener('ended', () => { if (vp.idx + 1 < MEDIA.videos.length) vp.load(vp.idx + 1, true); });

        const seek = q('[data-video="seek"]');
        seek.addEventListener('input', () => {
            vp.seeking = true;
            ui.setSeek(seek, Number(seek.value), Number(seek.max));
            q('[data-video="time"]').textContent = ui.clock(seek.value);
        });
        seek.addEventListener('change', () => {
            vp.seeking = false;
            v.currentTime = Number(seek.value);
            vp.wake();
        });
        q('[data-video="volume"]').addEventListener('input', (e) => {
            v.muted = false;
            v.volume = Number(e.target.value);
        });

        root.addEventListener('pointerdown', (e) => {
            vp.pointer = e.pointerType;
            if (e.target.closest('[data-controls]')) vp.wake();
        });
        root.addEventListener('pointermove', (e) => { if (e.pointerType === 'mouse') vp.wake(); });
        root.addEventListener('click', (e) => {
            if (e.target.closest('button, input')) return;
            root.focus({ preventScroll: true });
            if (vp.pointer === 'mouse') vp.toggle();
            else vp.tapAt((e.clientX - root.getBoundingClientRect().left) / root.offsetWidth);
        });
        root.addEventListener('dblclick', (e) => {
            if (vp.pointer === 'mouse' && !e.target.closest('button, input')) vp.fullscreen();
        });
        root.addEventListener('keydown', vp.key);
        root.addEventListener('fullscreenchange', () => {
            ui.popover.close();
            ui.tooltip.hide();
            const full = document.fullscreenElement === root;
            const b = q('[data-action="vp-fullscreen"]');
            b.innerHTML = `<i data-lucide="${full ? 'minimize' : 'maximize'}" class="size-[18px]"></i>`;
            b.dataset.tip = full ? 'Exit full screen' : 'Full screen';
            ui.icons(b);
        });

        vp.load(vp.idx, false);
        vp.renderVolume();
    },

    detach() {
        if (document.fullscreenElement && document.fullscreenElement === vp.root) document.exitFullscreen();
        vp.el?.pause();
        clearTimeout(vp.idleTimer);
        clearTimeout(vp.tapTimer);
        vp.root = null;
        vp.el = null;
    },

    load(idx, autoplay) {
        vp.idx = idx;
        const v = vp.el;
        const item = MEDIA.videos[idx];
        const q = (sel) => vp.root.querySelector(sel);
        v.querySelectorAll('track').forEach((t) => t.remove());
        v.poster = item.poster;
        v.src = item.src;
        if (item.captions) {
            const t = Object.assign(document.createElement('track'), { kind: 'captions', srclang: 'en', label: 'English', src: item.captions });
            v.appendChild(t);
            t.track.mode = vp.captions ? 'hidden' : 'disabled';
            t.track.addEventListener('cuechange', vp.renderCaptions);
        }
        q('[data-video-title]').textContent = item.title;
        q('[data-video-index]').textContent = `${idx + 1} / ${MEDIA.videos.length}`;
        q('[data-action="vp-prev"]').disabled = idx === 0;
        q('[data-action="vp-next"]').disabled = idx === MEDIA.videos.length - 1;
        const cc = q('[data-action="vp-captions"]');
        cc.hidden = !item.captions;
        cc.setAttribute('aria-pressed', String(vp.captions));
        vp.renderList();
        vp.renderCaptions();
        vp.renderState();
        vp.renderProgress();
        if (autoplay) v.play().catch(() => {});
    },
```

A captions track loads with `mode` set to `hidden`, so its cues fire without the browser drawing them. Full screen is requested on the player's root rather than the `<video>`, so the controls and the speed popover stay visible, and `webkitEnterFullscreen` on the video is the fallback where the root has no `requestFullscreen`.

```javascript
    toggle() {
        if (!vp.el) return;
        if (vp.el.paused) vp.el.play().catch(() => {});
        else vp.el.pause();
    },

    skip(delta) {
        const v = vp.el;
        v.currentTime = Math.max(0, Math.min(v.currentTime + delta, Number.isFinite(v.duration) ? v.duration : v.currentTime + delta));
        const flash = vp.root.querySelector('[data-video-flash]');
        const showing = !flash.classList.contains('opacity-0');
        vp.flash = showing && Math.sign(vp.flash) === Math.sign(delta) ? vp.flash + delta : delta;
        flash.textContent = `${vp.flash > 0 ? '+' : '-'}${Math.abs(vp.flash)}s`;
        flash.classList.toggle('left-[12%]', delta < 0);
        flash.classList.toggle('right-[12%]', delta > 0);
        flash.classList.remove('opacity-0');
        clearTimeout(vp.flashTimer);
        vp.flashTimer = setTimeout(() => flash.classList.add('opacity-0'), 700);
    },

    fullscreen() {
        if (document.fullscreenElement) document.exitFullscreen();
        else if (vp.root.requestFullscreen) vp.root.requestFullscreen().catch(() => ui.toast('Full screen was refused by the browser', { tone: 'red', icon: 'circle-alert' }));
        else vp.el.webkitEnterFullscreen?.();
    },

    toggleCaptions() {
        const track = vp.el?.textTracks[0];
        if (!track) return;
        vp.captions = !vp.captions;
        track.mode = vp.captions ? 'hidden' : 'disabled';
        vp.root.querySelector('[data-action="vp-captions"]').setAttribute('aria-pressed', String(vp.captions));
        vp.renderCaptions();
    },

    wake() {
        if (!vp.root) return;
        vp.root.classList.remove('idle');
        clearTimeout(vp.idleTimer);
        vp.idleTimer = setTimeout(() => {
            const hovered = matchMedia('(hover: hover)').matches && vp.root?.querySelector('[data-controls]:hover');
            if (!vp.root || vp.el.paused || vp.seeking || ui.popover.current || hovered) return;
            vp.root.classList.add('idle');
        }, 3000);
    },

    tapAt(at) {
        const side = at < 0.3 ? -1 : at > 0.7 ? 1 : 0;
        const now = performance.now();
        const last = vp.tap;
        vp.tap = { side, at: now };
        clearTimeout(vp.tapTimer);
        if (side && last && last.side === side && now - last.at < 300) {
            vp.skip(side * 10);
            return;
        }
        const flip = () => {
            if (vp.root.classList.contains('idle')) vp.wake();
            else vp.root.classList.add('idle');
        };
        if (side) vp.tapTimer = setTimeout(flip, 300);
        else flip();
    },

    key(e) {
        if (e.ctrlKey || e.metaKey || e.altKey || e.target.closest('input:not([type="range"])')) return;
        const actions = {
            ' ': vp.toggle,
            k: vp.toggle,
            ArrowLeft: () => vp.skip(e.shiftKey ? -30 : -10),
            ArrowRight: () => vp.skip(e.shiftKey ? 30 : 10),
            f: vp.fullscreen,
            c: vp.toggleCaptions,
            m: () => { vp.el.muted = !vp.el.muted; },
        };
        const action = actions[e.key.length === 1 ? e.key.toLowerCase() : e.key];
        if (!action) return;
        e.preventDefault();
        action();
        vp.wake();
    },
```

The controls hide three seconds after the last movement, unless the video is paused, a seek is in progress, a popover is open, or the pointer rests on the controls. A tap on a touch screen toggles them.

A double tap in the outer 30% on either side skips ten seconds, so a single tap there waits 300ms to rule out a second one. Repeated skips in one direction add up in the flash.

```javascript
    renderState() {
        if (!vp.el) return;
        const playing = !vp.el.paused;
        showWhen('video', playing);
        vp.root.querySelector('[data-action="vp-toggle"]').setAttribute('aria-label', playing ? 'Pause' : 'Play');
    },

    renderProgress() {
        if (!vp.el || vp.seeking) return;
        const v = vp.el;
        const d = Number.isFinite(v.duration) && v.duration > 0 ? v.duration : MEDIA.videos[vp.idx].duration;
        const c = Math.min(v.currentTime || 0, d);
        ui.setSeek(vp.root.querySelector('[data-video="seek"]'), c, d);
        vp.root.querySelector('[data-video="time"]').textContent = ui.clock(c);
        vp.root.querySelector('[data-video="duration"]').textContent = ui.clock(d);
    },

    renderVolume() {
        if (!vp.el) return;
        const v = vp.el;
        ui.setSeek(vp.root.querySelector('[data-video="volume"]'), v.muted ? 0 : v.volume, 1);
        const b = vp.root.querySelector('[data-action="vp-mute"]');
        b.innerHTML = `<i data-lucide="${v.muted || v.volume === 0 ? 'volume-x' : 'volume-2'}" class="size-[18px]"></i>`;
        b.dataset.tip = v.muted ? 'Unmute' : 'Mute';
        ui.icons(b);
    },

    renderCaptions() {
        if (!vp.root) return;
        const box = vp.root.querySelector('[data-video-captions]');
        const track = vp.el.textTracks[0];
        const cues = vp.captions && track ? [...(track.activeCues || [])] : [];
        box.replaceChildren(...cues.flatMap((cue) => cue.text.split('\n')).map((line) => {
            const s = document.createElement('span');
            s.className = 'rounded-lg bg-crust/80 px-2.5 py-0.5 text-sm font-medium text-text sm:text-[16px] lg:text-lg';
            s.textContent = line;
            return s;
        }));
        box.hidden = cues.length === 0;
    },

    renderList() {
        const list = document.querySelector('[data-video-list]');
        if (!list) return;
        list.innerHTML = MEDIA.videos.map((v, i) => {
            const on = i === vp.idx;
            return `<li><button type="button" data-action="vp-pick" data-index="${i}" class="flex w-full items-center gap-3 rounded-2xl p-2 text-left transition-colors ${on ? 'bg-surface0' : 'hover:bg-surface0/50'} ${ui.cls.focus}">
                <span class="relative aspect-video w-28 shrink-0 overflow-hidden rounded-xl bg-base"><img src="${v.poster}" alt="" class="size-full object-cover"><span class="dark absolute right-1 bottom-1 rounded-md bg-crust/80 px-1.5 py-0.5 text-[10px] tabular-nums text-text">${ui.clock(v.duration)}</span></span>
                <span class="min-w-0 flex-1"><span class="block truncate text-sm ${on ? 'text-mauve' : 'text-text'}">${ui.esc(v.title)}</span><span class="block truncate text-xs text-overlay1">${ui.esc(v.meta)}</span></span>
            </button></li>`;
        }).join('');
    },
};

ui.on('vp-toggle', () => vp.toggle());
ui.on('vp-prev', () => vp.load(vp.idx - 1, true));
ui.on('vp-next', () => vp.load(vp.idx + 1, true));
ui.on('vp-pick', (b) => vp.load(Number(b.dataset.index), true));
ui.on('vp-mute', () => { vp.el.muted = !vp.el.muted; });
ui.on('vp-captions', () => vp.toggleCaptions());
ui.on('vp-fullscreen', () => vp.fullscreen());
ui.on('vp-speed', (b) => {
    const pop = ui.popover.open(b, vp.rates.map((r) => `<button type="button" data-rate="${r}" class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left text-sm tabular-nums transition-colors ${r === vp.el.playbackRate ? 'bg-surface0 text-text' : 'text-subtext0 hover:bg-surface0 hover:text-text'}"><span class="flex-1">${r}x</span>${r === vp.el.playbackRate ? '<i data-lucide="check" class="size-4 text-mauve"></i>' : ''}</button>`).join(''), { align: 'end', width: 140 });
    pop.addEventListener('click', (e) => {
        const r = e.target.closest('[data-rate]');
        if (!r) return;
        vp.el.playbackRate = Number(r.dataset.rate);
        ui.popover.close();
        vp.wake();
    });
});
```

Speed is a popover of rates aligned to the button's end, with the current rate checked.
