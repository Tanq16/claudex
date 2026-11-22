views.shell = (() => {
    const runs = new Map();

    const pump = async (name) => {
        const s = runs.get(name);
        if (s.list.some((r) => r.state === 'running')) return;
        const next = s.list.find((r) => r.state === 'queued');
        if (!next) return;
        next.state = 'running';
        next.started = Date.now();
        s.draw?.(true);
        try {
            next.result = await api.shell(name, next.command);
        } catch (err) {
            next.error = err.message;
        }
        next.state = 'done';
        s.draw?.(true);
        pump(name);
    };

    const entryHtml = (r, i) => {
        const res = r.result;
        const chips = [];
        if (r.state === 'queued') chips.push(ui.chip('Queued', { icon: 'clock' }));
        else if (r.state === 'running') chips.push(`<span class="flex items-center gap-1.5 text-xs text-overlay1">${ui.spinner('size-3.5')}<span data-start="${r.started}"></span></span>`);
        else if (r.error) chips.push(ui.chip('Failed', { tone: 'red', icon: 'circle-alert' }));
        else {
            if (res.timedOut) chips.push(ui.chip('Timed out', { tone: 'peach', icon: 'timer-off' }));
            else chips.push(ui.chip(`Exit ${res.exitCode}`, { tone: res.exitCode === 0 ? 'green' : 'red', mono: true }));
            if (res.truncated) chips.push(ui.chip('Truncated', { tone: 'peach', icon: 'scissors' }));
            chips.push(`<span class="text-xs text-overlay1 tabular-nums">${fmt.ms(res.durationMs)}</span>`);
        }
        const out = (text, tone) => (text ? `<pre data-scroll class="max-h-72 overflow-auto rounded-2xl bg-base p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap wrap-anywhere ${tone}">${ui.esc(text.replace(/\n$/, ''))}</pre>` : '');
        const body = r.error ? out(r.error, 'text-red') : res ? `${out(res.stdout, 'text-subtext1')}${out(res.stderr, 'text-red')}${!res.stdout && !res.stderr ? '<p class="px-1 text-xs text-overlay1">No output</p>' : ''}` : '';
        return `<li class="group flex flex-col gap-2">
            <header class="flex min-w-0 items-center gap-2">
                <code class="min-w-0 flex-1 truncate font-mono text-sm text-text" data-tip="${ui.esc(r.command)}"><span class="select-none text-mauve">$ </span>${ui.esc(r.command)}</code>
                <span class="flex shrink-0 items-center gap-1.5">${chips.join('')}</span>
                ${res ? `<span class="${ui.cls.reveal}">${ui.iconButton('copy', { tip: 'Copy output', size: 'sm', attrs: `data-shell-copy="${i}"` })}</span>` : ''}
            </header>
            ${body}
        </li>`;
    };

    function open(name) {
        if (!runs.has(name)) runs.set(name, { list: [], draw: null });
        const s = runs.get(name);
        const { list } = s;
        const p = store.state?.projects.find((x) => x.name === name);
        let recall = list.length;
        const m = ui.modal.open({
            title: 'Shell',
            icon: 'square-terminal',
            subtitle: `<span class="font-mono">${ui.esc(p?.dir || name)}</span>`,
            size: 'lg',
            body: '<ol data-shell-runs class="flex flex-col gap-5 pt-1"></ol>',
            footer: `<form data-shell-form class="flex w-full items-center gap-2">
                <label class="relative flex min-w-0 flex-1"><span class="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 font-mono text-sm text-mauve">$</span><input data-shell-input class="${ui.cls.input} pl-9 font-mono" placeholder="git status" autocomplete="off" autocapitalize="off" spellcheck="false" aria-label="Command"></label>
                ${ui.button('Run', { icon: 'play', variant: 'primary', type: 'submit' })}
            </form>`,
            onClose: () => {
                clearInterval(ticker);
                s.draw = null;
            },
        });
        const host = m.el.querySelector('[data-shell-runs]');
        const scroller = m.el.querySelector('[data-modal-body]');
        const input = m.el.querySelector('[data-shell-input]');
        const draw = (follow) => {
            host.innerHTML = list.length ? list.map(entryHtml).join('') : `<li>${ui.empty({ icon: 'square-terminal', title: 'No commands yet', hint: 'Commands run one after another in the project directory, with its account environment.' })}</li>`;
            ui.icons(host);
            tick();
            if (follow) scroller.scrollTop = scroller.scrollHeight;
        };
        const tick = () => host.querySelectorAll('[data-start]').forEach((n) => { n.textContent = fmt.dur(Date.now() - Number(n.dataset.start)); });
        const ticker = setInterval(tick, 250);
        s.draw = draw;
        m.el.querySelector('[data-shell-form]').addEventListener('submit', (e) => {
            e.preventDefault();
            const command = input.value.trim();
            if (!command) return;
            list.push({ command, state: 'queued' });
            recall = list.length;
            input.value = '';
            draw(true);
            pump(name);
        });
        input.addEventListener('keydown', (e) => {
            if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return;
            e.preventDefault();
            recall = Math.max(0, Math.min(list.length, recall + (e.key === 'ArrowUp' ? -1 : 1)));
            input.value = list[recall]?.command ?? '';
        });
        host.addEventListener('click', (e) => {
            const b = e.target.closest('[data-shell-copy]');
            if (!b) return;
            const r = list[Number(b.dataset.shellCopy)];
            const res = r.result || {};
            ui.copy([res.stdout, res.stderr].filter(Boolean).join('\n') || r.error || '', b);
        });
        draw(true);
        input.focus();
    }

    return { open };
})();
