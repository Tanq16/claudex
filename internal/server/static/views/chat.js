views.chat = (() => {
    const TOOLS = {
        Bash: ['terminal', (i) => i.command],
        Read: ['file-text', (i) => i.file_path],
        Edit: ['file-pen-line', (i) => i.file_path],
        MultiEdit: ['file-pen-line', (i) => i.file_path],
        Write: ['file-plus', (i) => i.file_path],
        NotebookEdit: ['notebook-pen', (i) => i.notebook_path],
        Glob: ['folder-search', (i) => i.pattern],
        Grep: ['search', (i) => i.pattern],
        WebFetch: ['globe', (i) => i.url],
        WebSearch: ['globe', (i) => i.query],
        ToolSearch: ['search', (i) => i.query],
        Skill: ['sparkles', (i) => i.skill || i.command],
        TaskStop: ['circle-stop', (i) => i.task_id || i.shell_id],
        Monitor: ['activity', (i) => i.description || i.command],
    };
    const PATH_TOOLS = new Set(['Read', 'Edit', 'MultiEdit', 'Write', 'NotebookEdit']);
    const HIDDEN = new Set(['TaskList', 'TaskGet', 'AskUserQuestion', 'ExitPlanMode']);
    const TODO_TOOLS = new Set(['TaskCreate', 'TaskUpdate']);
    const AGENT_TOOLS = new Set(['Agent', 'Task']);
    const CONTEXT_TONES = { system_prompt: 'blue', system_tools: 'sapphire', mcp_tools: 'teal', custom_agents: 'pink', memory_files: 'peach', skills: 'yellow', messages: 'mauve', compact_buffer: 'overlay0' };
    const AGENT_TONES = ['sapphire', 'teal', 'pink', 'lavender'];
    const EFFORTS = { low: 'Low', medium: 'Medium', high: 'High', xhigh: 'Extra high', max: 'Max' };
    const MODES = [
        { value: 'default', label: 'Ask', icon: 'shield-question' },
        { value: 'acceptEdits', label: 'Accept edits', icon: 'file-pen-line' },
        { value: 'plan', label: 'Plan', icon: 'list-todo' },
        { value: 'auto', label: 'Auto', icon: 'zap' },
        { value: 'bypassPermissions', label: 'Bypass', icon: 'shield-off' },
    ];
    const LIVE = new Set(['idle', 'running', 'requires_action']);
    const BUSY = new Set(['running', 'requires_action']);
    const TASK_KINDS = {
        local_agent: ['bot', 'Agent'],
        local_bash: ['terminal', 'Background command'],
        local_workflow: ['workflow', 'Workflow'],
    };

    let C = null;

    const $ = (sel) => C.pane.querySelector(sel);
    const ts = (m) => Date.parse(m?.timestamp) || 0;
    const ansi = (t) => String(t ?? '').replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '');
    const rise = () => (C.replaying ? '' : '[animation:rise-in_.2s_ease-out]');
    const textOf = (content) => (typeof content === 'string' ? content : Array.isArray(content) ? content.filter((b) => b.type === 'text').map((b) => b.text).join('\n') : '');
    const modeLabel = (v) => MODES.find((m) => m.value === v)?.label || v;
    const slug = (name) => String(name).toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '');
    const shortModel = (m) => String(m || '').replace(/^claude-/, '');
    const isLive = () => LIVE.has(C?.snap?.state);

    const runningChip = () => `<span class="inline-flex items-center gap-2 rounded-full bg-green/15 px-2.5 py-1 text-xs font-medium text-green">${ui.dot('green', { pulse: true })}Running</span>`;

    const shortPath = (p) => {
        const s = String(p ?? '');
        const i = s.indexOf(`/${C.name}/`);
        return i >= 0 ? s.slice(i + C.name.length + 2) : s;
    };

    const toolArg = (block) => {
        const [, arg] = TOOLS[block.name] || [null, (i) => Object.values(i || {}).find((v) => typeof v === 'string')];
        const v = arg(block.input || {}) ?? '';
        return PATH_TOOLS.has(block.name) ? shortPath(v) : String(v);
    };

    const paint = (target, html) => {
        const el = typeof target === 'string' ? $(target) : target;
        if (!el || el._html === html) return el;
        el._html = html;
        el.innerHTML = html;
        ui.icons(el);
        return el;
    };

    const parseId = (id) => {
        const i = id.lastIndexOf(':');
        return [id.slice(0, i), Number(id.slice(i + 1))];
    };

    const transcript = () => ({
        rows: new Map(),
        agents: new Map(),
        groups: new Map(),
        workflows: new Map(),
        tasks: new Map(),
        approvals: new Map(),
        group: null,
        todos: { items: [], calls: new Map(), card: null },
        turn: { start: 0, tools: 0, interrupted: false },
        turnState: '',
        activity: '',
        thinking: 0,
        lastTs: 0,
    });

    const html = () => `
        <div data-chat-scroller data-scroll="lane" class="min-h-0 flex-1 overflow-y-auto px-4">
            <ol data-chat-log data-column class="mx-auto flex w-full flex-col gap-4 py-4 transition-[max-width] duration-300 ease-out ${column.cls()}"></ol>
        </div>
        <footer class="shrink-0 px-3 pb-3 sm:px-4">
            <div data-column class="mx-auto w-full transition-[max-width] duration-300 ease-out ${column.cls()}">
                <div data-chat-banner></div>
                <div data-chat-status class="flex h-9 items-center gap-2 px-2 text-xs"></div>
                <form data-chat-composer novalidate class="rounded-3xl bg-surface0/50 p-2 transition-shadow focus-within:ring-2 focus-within:ring-mauve/40">
                    <textarea data-chat-input data-scroll rows="1" spellcheck="false" placeholder="Message, or / for commands" class="block max-h-48 w-full resize-none bg-transparent px-3 py-2 text-sm leading-relaxed text-text placeholder:text-overlay0 outline-none disabled:cursor-not-allowed pointer-coarse:text-[16px]"></textarea>
                    <div class="flex items-center gap-1.5 pt-1">
                        <span data-chat-model class="min-w-0"></span>
                        <span data-chat-effort class="max-sm:hidden"></span>
                        <span data-chat-mode class="max-md:hidden"></span>
                        <span data-chat-ring class="ml-auto"></span>
                        <span data-chat-send></span>
                    </div>
                </form>
            </div>
        </footer>`;

    function mount(pane, name, hooks = {}) {
        unmount();
        C = {
            name,
            pane,
            hooks,
            log: pane.querySelector('[data-chat-log]'),
            scroller: pane.querySelector('[data-chat-scroller]'),
            input: pane.querySelector('[data-chat-input]'),
            queue: [],
            pumping: false,
            epoch: '',
            seq: 0,
            loaded: false,
            snap: null,
            init: null,
            history: [],
            events: [],
            sending: false,
            replaying: false,
            dismissed: 0,
            initRetry: null,
            stateKey: null,
            tailing: new Set(),
            ...transcript(),
        };
        C.log.innerHTML = `<li data-chat-empty class="flex items-center justify-center gap-2 py-16 text-sm text-overlay1">${ui.spinner()}Loading the conversation</li>`;
        C.input.addEventListener('input', () => {
            autosize();
            slash.update();
        });
        C.input.addEventListener('keydown', (e) => {
            if (slash.keys(e)) return;
            if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
                e.preventDefault();
                submit();
            }
        });
        C.input.addEventListener('blur', () => setTimeout(() => slash.close(), 150));
        pane.querySelector('[data-chat-composer]').addEventListener('submit', (e) => {
            e.preventDefault();
            submit();
        });
        pane.addEventListener('ui-change', onPicker);
        ui.icons(pane);
        C.ticker = setInterval(tick, 250);
        C.poller = setInterval(pollTails, 2000);
        renderComposer();
        renderStatus();
        live.watch(name, onLive);
    }

    function unmount() {
        if (!C) return;
        clearInterval(C.ticker);
        clearInterval(C.poller);
        slash.close();
        C = null;
        live.unwatch();
    }

    const onLive = (name, data, id) => {
        if (!C) return;
        C.queue.push({ name, data, id });
        pump();
    };

    const pump = async () => {
        if (!C || C.pumping) return;
        const cur = C;
        cur.pumping = true;
        while (cur === C && cur.queue.length) {
            const item = cur.queue.shift();
            try {
                if (item.name === 'reset') await applyReset(item.data, item.id);
                else if (item.name === 'session') applySnapshot(item.data);
                else if (accept(item.id)) {
                    C.events.push(item);
                    dispatch(item);
                }
            } catch (err) {
                console.error(err);
            }
        }
        cur.pumping = false;
    };

    const accept = (id) => {
        if (!id) return true;
        const [epoch, seq] = parseId(id);
        if (epoch !== C.epoch || seq <= C.seq) return false;
        C.seq = seq;
        return true;
    };

    const dispatch = (item) => {
        if (item.name === 'approval') resolveApproval(item.data);
        else if (item.name === 'message') handle(item.data);
    };

    const applyReset = async (data, id) => {
        if (!data.session && !C.epoch && C.loaded) {
            applySnapshot(null);
            return;
        }
        const cur = C;
        let history = [];
        try {
            history = (await api.history(C.name)).messages || [];
        } catch (err) {
            ui.toast(`Loading the history failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
        if (cur !== C) return;
        Object.assign(C, transcript());
        C.epoch = data.session && id ? parseId(id)[0] : '';
        C.seq = 0;
        C.init = data.init || null;
        C.snap = data.session || null;
        C.history = history;
        C.events = [];
        C.log.innerHTML = '';
        C.replaying = true;
        for (const m of history) safe(() => handle(m));
        settle();
        for (const ev of data.events || []) {
            if (!accept(ev.id)) continue;
            C.events.push(ev);
            safe(() => dispatch({ name: ev.event, data: ev.data }));
        }
        for (const raw of C.snap?.pending || []) {
            if (!C.approvals.has(raw.request_id)) safe(() => approval(raw));
        }
        closeGroup();
        C.replaying = false;
        C.loaded = true;
        if (!C.log.children.length) C.log.innerHTML = emptyHtml();
        ui.icons(C.log);
        applySnapshot(C.snap);
        requestAnimationFrame(() => {
            if (C === cur) C.scroller.scrollTop = C.scroller.scrollHeight;
        });
    };

    const settle = () => {
        for (const wf of C.workflows.values()) {
            if (wf.done) continue;
            wf.done = true;
            paint(wf.el.querySelector('[data-wf-state]'), ui.chip('Ended'));
            paint(wf.el.querySelector('[data-wf-stop]'), '');
            wf.el.removeAttribute('data-live-task');
        }
        for (const a of C.agents.values()) if (!a.done) agentDone(a, { status: 'completed' });
        C.log.querySelectorAll('[data-tool-row] [data-state] .animate-spin').forEach((n) => {
            n.parentElement.innerHTML = '<i data-lucide="circle-minus" class="size-3.5 text-overlay0"></i>';
        });
        C.log.querySelectorAll('[data-tool-row] [data-meta] [data-start]').forEach((n) => n.remove());
    };

    const safe = (fn) => {
        try {
            fn();
        } catch (err) {
            console.error(err);
        }
    };

    const emptyHtml = () => `<li data-chat-empty class="flex flex-col items-center justify-center gap-3 py-16 text-center">
        <span class="flex size-12 items-center justify-center rounded-full bg-surface0 text-overlay1"><i data-lucide="messages-square" class="size-5"></i></span>
        <p class="font-display text-[16px] text-text">No messages yet</p>
    </li>`;

    const applySnapshot = (snap) => {
        if (!C) return;
        C.snap = snap;
        if (snap && !C.init && LIVE.has(snap.state) && C.initRetry !== C.epoch) {
            C.initRetry = C.epoch;
            live.reconnect(true);
        }
        renderComposer();
        renderStatus();
        renderBanner();
        paintLive();
        const key = [C.loaded, snap?.state, snap?.model, snap?.effort, snap?.permissionMode, !!C.init].join(':');
        if (key !== C.stateKey) {
            C.stateKey = key;
            C.hooks.onSession?.();
        }
    };

    const handle = (m) => {
        if (!m || typeof m !== 'object') return;
        const t = ts(m);
        if (t) C.lastTs = t;
        if (m.type === 'assistant') assistant(m);
        else if (m.type === 'user') user(m);
        else if (m.type === 'result') result(m);
        else if (m.type === 'system') system[m.subtype]?.(m);
        else if (m.type === 'control_request') approval(m);
        else if (m.type === 'conversation_reset') conversationReset();
        else if (m.type === 'tool_progress') toolProgress(m);
    };

    const nearBottom = () => {
        const s = C.scroller;
        if (!s.clientHeight) return true;
        return s.scrollHeight - s.scrollTop - s.clientHeight < 120;
    };

    const scrollDown = () => {
        if (C.replaying) return;
        const cur = C;
        requestAnimationFrame(() => {
            if (C === cur) C.scroller.scrollTo({ top: C.scroller.scrollHeight, behavior: 'smooth' });
        });
    };

    const append = (markup) => {
        const stick = !C.replaying && nearBottom();
        C.log.querySelector(':scope > [data-chat-empty]')?.remove();
        const tpl = document.createElement('template');
        tpl.innerHTML = markup.trim();
        const node = C.log.appendChild(tpl.content.firstElementChild);
        ui.icons(node);
        if (stick) scrollDown();
        return node;
    };

    const keepDown = () => {
        if (!C.replaying && nearBottom()) scrollDown();
    };

    const closeGroup = () => {
        const g = C.group;
        C.group = null;
        if (!g) return;
        const n = g.querySelectorAll('[data-tool-row]').length;
        if (n < 3) return;
        const head = g.querySelector('[data-group-head]');
        head.classList.remove('hidden');
        head.classList.add('flex');
        head.querySelector('[data-group-count]').textContent = fmt.plural(n, 'tool call');
        g.querySelector('[data-group-body]').classList.add('hidden');
    };

    const notice = (icon, text, tone = 'overlay1') => {
        closeGroup();
        append(`<li class="flex items-center gap-3 text-xs text-${tone} ${rise()}"><span class="h-px flex-1 bg-surface0"></span><i data-lucide="${icon}" class="size-3.5 shrink-0"></i><span class="max-w-[80%] text-center wrap-anywhere">${ui.esc(text)}</span><span class="h-px flex-1 bg-surface0"></span></li>`);
    };

    const thumbs = (urls, cls = '') => (urls.length ? `<div class="flex flex-wrap gap-2 ${cls}">${urls.map((u) => `<button type="button" data-action="chat-image" data-src="${ui.esc(u)}" class="group relative overflow-hidden rounded-2xl bg-base focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40"><img src="${ui.esc(u)}" alt="" loading="lazy" class="h-36 w-auto max-w-full object-contain transition-transform duration-300 group-hover:scale-[1.03]"><span class="absolute right-2 bottom-2 flex size-7 items-center justify-center rounded-full bg-crust/70 text-text opacity-0 backdrop-blur-sm transition-opacity group-hover:opacity-100"><i data-lucide="maximize-2" class="size-3.5"></i></span></button>`).join('')}</div>` : '');

    const imageUrls = (content) => (Array.isArray(content) ? content.filter((b) => b.type === 'image' && b.source?.type === 'url' && b.source.url).map((b) => b.source.url) : []);

    const userBubble = (text, images = []) => {
        closeGroup();
        const li = append(`<li data-user class="group/msg flex items-end justify-end gap-1 ${rise()}">
            <span class="opacity-0 transition-opacity group-hover/msg:opacity-100 focus-within:opacity-100 max-md:opacity-100">${ui.iconButton('copy', { title: 'Copy', size: 'sm', action: 'chat-copy-prompt' })}</span>
            <div class="flex max-w-[85%] min-w-0 flex-col items-end gap-2">${text ? '<div class="min-w-0 max-w-full rounded-3xl rounded-br-lg bg-surface0 px-4 py-2.5"><div class="markdown-body user-md"></div></div>' : ''}${thumbs(images, 'justify-end')}</div>
        </li>`);
        li.dataset.text = text;
        if (text) md.render(li.querySelector('.user-md'), text, { breaks: true });
        scrollDown();
    };

    const commandPrompt = (text) => {
        const tag = (name) => text.match(new RegExp(`<${name}>([\\s\\S]*?)</${name}>`))?.[1]?.trim() || '';
        const name = tag('command-name') || `/${tag('command-message')}`;
        return [name, tag('command-args')].filter(Boolean).join(' ');
    };

    const prompt = (text, images, m) => {
        const t = text.trimStart();
        if (t.startsWith('<command-name>') || t.startsWith('<command-message>')) return userBubble(commandPrompt(t));
        if (t.startsWith('<local-command-stdout>')) return commandTile(t.replace(/<\/?local-command-stdout>/g, '').trim());
        if (t.startsWith('<local-command-caveat>') || t.startsWith('<system-reminder>')) return;
        if (t.startsWith('<task-notification>')) {
            const summary = t.match(/<summary>([\s\S]*?)<\/summary>/)?.[1]?.trim();
            return notice('bell', summary || 'A background task finished');
        }
        if (t.startsWith('[Request interrupted')) return interrupted();
        if (m.isCompactSummary) return notice('shrink', 'Conversation compacted');
        C.turn.interrupted = false;
        C.turn.tools = 0;
        userBubble(text, images);
    };

    const interrupted = () => {
        C.turn.interrupted = true;
        abortRows();
        notice('octagon-x', 'Interrupted', 'red');
    };

    const abortRows = () => {
        C.log.querySelectorAll('[data-tool-row] [data-state] .animate-spin').forEach((s) => {
            s.parentElement.innerHTML = '<i data-lucide="circle-x" class="size-3.5 text-red"></i>';
        });
        C.log.querySelectorAll('[data-tool-row] [data-meta] [data-start]').forEach((s) => s.remove());
        ui.icons(C.log);
    };

    const user = (m) => {
        if (m.isSynthetic || m.isReplay) return;
        const content = m.message?.content;
        if (m.parent_tool_use_id) {
            if (Array.isArray(content)) content.filter((c) => c.type === 'tool_result').forEach((c) => toolResult(m, c));
            return;
        }
        if (typeof content === 'string') return prompt(content, [], m);
        if (!Array.isArray(content)) return;
        const results = content.filter((c) => c.type === 'tool_result');
        results.forEach((c) => toolResult(m, c));
        if (results.length) return;
        const text = textOf(content);
        const images = imageUrls(content);
        if (text || images.length) prompt(text, images, m);
    };

    const assistant = (m) => {
        const blocks = m.message?.content || [];
        if (m.message?.model === '<synthetic>' && m.local_command_run) return localCommand(m);
        for (const block of blocks) {
            if (m.parent_tool_use_id) agentBlock(m.parent_tool_use_id, block, m);
            else if (block.type === 'thinking' || block.type === 'redacted_thinking') thinking(m, block);
            else if (block.type === 'text') text(m, block);
            else if (block.type === 'tool_use') toolUse(m, block);
        }
    };

    const thinking = (m, block) => {
        closeGroup();
        const secs = ((m.thinking_duration_ms || 0) / 1000).toFixed(1);
        const has = block.thinking?.trim();
        const label = m.thinking_duration_ms ? `Thought for ${secs}s` : 'Thought';
        append(`<li class="${rise()}">
            <button type="button" ${has ? 'data-action="chat-detail"' : 'disabled'} class="group flex items-center gap-2 rounded-full py-1 pr-3 text-xs text-overlay1 transition-colors enabled:hover:text-text">
                <i data-lucide="brain" class="size-3.5"></i>${label}${has ? '<i data-lucide="chevron-right" class="size-3.5 transition-transform group-aria-expanded:rotate-90"></i>' : ''}
            </button>
            ${has ? `<p data-detail class="mt-1 hidden border-l-2 border-surface0 pl-4 text-sm leading-relaxed whitespace-pre-wrap text-overlay1 italic">${ui.esc(block.thinking)}</p>` : ''}
        </li>`);
        C.activity = '';
        C.thinking = 0;
        renderStatus();
    };

    const text = (m, block) => {
        if (!block.text?.trim()) return;
        closeGroup();
        const id = m.message?.id || m.uuid || '';
        const last = C.log.lastElementChild;
        if (id && last?.dataset.msg === id && last.dataset.text !== undefined) {
            last.dataset.text += `\n\n${block.text}`;
            md.render(last.querySelector('article'), last.dataset.text);
        } else {
            const li = append(`<li data-msg="${ui.esc(id)}" class="${rise()}"><article class="markdown-body chat-reply"></article></li>`);
            li.dataset.text = block.text;
            md.render(li.querySelector('article'), block.text);
        }
        keepDown();
    };

    const toolUse = (m, block) => {
        C.turn.tools += 1;
        if (AGENT_TOOLS.has(block.name)) return agentStart(m, block);
        if (block.name === 'Workflow') return workflowStart(m, block);
        if (TODO_TOOLS.has(block.name)) {
            C.todos.calls.set(block.id, block);
            return;
        }
        if (HIDDEN.has(block.name)) return;
        if (!C.group) {
            C.group = append(`<li class="${rise()}">
                <button type="button" data-group-head data-action="chat-group" class="group mb-1 hidden items-center gap-2 rounded-full py-1 pr-3 text-xs text-overlay1 transition-colors hover:text-text"><i data-lucide="wrench" class="size-3.5"></i><span data-group-count></span><i data-lucide="chevron-right" class="size-3.5 transition-transform group-aria-expanded:rotate-90"></i></button>
                <ul data-group-body class="flex flex-col"></ul>
            </li>`);
        }
        const start = ts(m) || Date.now();
        const body = C.group.querySelector('[data-group-body]');
        body.insertAdjacentHTML('beforeend', toolRow(block, false, start));
        const el = body.lastElementChild;
        ui.icons(el);
        C.rows.set(block.id, { el, block, start });
        C.activity = block.name === 'Bash' ? block.input?.description || 'Running a command' : `${block.name} ${toolArg(block)}`.trim();
        renderStatus();
        keepDown();
    };

    const toolRow = (block, nested, start) => {
        const [icon] = TOOLS[block.name] || ['wrench'];
        const value = toolArg(block);
        return `<li data-tool-row data-tool="${ui.esc(block.id)}">
            <button type="button" data-action="chat-detail" class="group flex ${nested ? 'h-7' : 'h-8'} w-full items-center gap-2.5 rounded-xl px-2 text-left text-sm transition-colors hover:bg-surface0/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
                <i data-lucide="${icon}" class="size-3.5 shrink-0 text-overlay1"></i>
                <span class="shrink-0 font-medium text-subtext1">${ui.esc(block.name)}</span>
                <span title="${ui.esc(block.input?.description || value)}" class="min-w-0 flex-1 truncate font-mono text-xs text-overlay1">${ui.esc(value)}</span>
                <span data-meta class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums">${block.name === 'Bash' ? `<span data-start="${start}"></span>` : ''}</span>
                <span data-state class="flex w-10 shrink-0 items-center justify-end gap-1 text-xs text-overlay0 tabular-nums">${ui.spinner('size-3.5 text-overlay1')}</span>
            </button>
            <div data-detail class="hidden pt-1 pb-2 pl-8"></div>
        </li>`;
    };

    const well = (content, cls = 'max-h-72') => `<pre data-scroll class="rounded-2xl bg-base ${cls} overflow-auto p-3 font-mono text-xs leading-relaxed text-subtext0">${content}</pre>`;

    const diff = (hunks) => {
        const rows = hunks.flatMap((h) => {
            let n = h.newStart;
            return (h.lines || []).map((l) => {
                const k = l[0];
                const tone = k === '+' ? 'bg-green/10 text-green' : k === '-' ? 'bg-red/10 text-red' : 'text-subtext0';
                const num = k === '-' ? '' : n++;
                return `<span class="select-none px-2 text-right text-overlay0 tabular-nums">${num}</span><span class="${tone} px-2 whitespace-pre">${ui.esc(l.slice(1)) || ' '}</span>`;
            });
        }).join('');
        return `<div data-scroll class="max-h-72 overflow-auto rounded-2xl bg-base py-2 font-mono text-xs leading-relaxed"><div class="grid min-w-max grid-cols-[auto_1fr]">${rows}</div></div>`;
    };

    const toolResult = (m, c) => {
        if (C.todos.calls.has(c.tool_use_id)) return todoResult(m, c);
        const agent = C.agents.get(c.tool_use_id);
        if (agent) return agentResult(agent, m, c);
        const wf = C.workflows.get(c.tool_use_id);
        if (wf) {
            wf.taskId ||= m.tool_use_result?.taskId || '';
            return;
        }
        const row = C.rows.get(c.tool_use_id);
        if (!row) return;
        const r = m.tool_use_result && typeof m.tool_use_result === 'object' ? m.tool_use_result : {};
        const { block, el } = row;
        const state = el.querySelector('[data-state]');
        const meta = el.querySelector('[data-meta]');
        const detail = el.querySelector('[data-detail]');
        const end = ts(m);
        const dur = end && row.start ? end - row.start : 0;
        const contentText = textOf(c.content);
        state.innerHTML = c.is_error ? '<i data-lucide="circle-x" class="size-3.5 text-red"></i>' : dur >= 1000 ? fmt.dur(dur) : '<i data-lucide="check" class="size-3.5 text-overlay0"></i>';
        meta.innerHTML = '';
        if (block.name === 'Bash') {
            const out = [r.stdout, r.stderr].filter(Boolean).join('\n') || contentText;
            detail.innerHTML = well(`<span class="text-mauve">$</span> <span class="text-text">${ui.esc(block.input?.command)}</span>${out ? `\n${ui.esc(ansi(out))}` : ''}`);
        } else if (!c.is_error && (block.name === 'Edit' || block.name === 'MultiEdit' || (block.name === 'Write' && r.structuredPatch?.length))) {
            const lines = (r.structuredPatch || []).flatMap((h) => h.lines || []);
            const add = lines.filter((l) => l.startsWith('+')).length;
            const del = lines.filter((l) => l.startsWith('-')).length;
            meta.innerHTML = `<span class="text-green">+${add}</span>${del ? `<span class="text-red">-${del}</span>` : ''}`;
            detail.innerHTML = diff(r.structuredPatch || []);
        } else if (!c.is_error && block.name === 'Write') {
            const lines = String(block.input?.content || '').replace(/\n$/, '').split('\n');
            meta.innerHTML = `<span class="text-green">+${lines.length}</span>`;
            detail.innerHTML = diff([{ newStart: 1, lines: lines.map((l) => `+${l}`) }]);
        } else if (!c.is_error && block.name === 'Read' && r.file) {
            if (r.file.numLines) meta.innerHTML = `<span>${fmt.plural(r.file.numLines, 'line')}</span>`;
        } else if (contentText) {
            detail.innerHTML = well(ui.esc(ansi(contentText)), 'max-h-56');
        }
        const images = imageUrls(c.content);
        if (!images.length && r.file?.url) images.push(r.file.url);
        if (images.length) el.insertAdjacentHTML('beforeend', thumbs(images, 'pt-1 pb-2 pl-8'));
        if (!detail.innerHTML) el.querySelector('button').removeAttribute('data-action');
        ui.icons(el);
        C.activity = '';
        renderStatus();
        keepDown();
    };

    const toolProgress = (m) => {
        if (m.parent_tool_use_id || !m.tool_name) return;
        const row = C.rows.get(m.tool_use_id);
        if (row?.block.name === 'Bash') C.activity = row.block.input?.description || 'Running a command';
        renderStatus();
    };

    const agentStart = (m, block) => {
        const id = m.message?.id || block.id;
        let group = C.groups.get(id);
        if (!group?.isConnected) {
            closeGroup();
            group = append(`<li class="${rise()}"><section class="rounded-3xl bg-surface0/50 p-2">
                <header class="flex items-center gap-2 px-2 pt-1 pb-2 text-xs text-overlay1"><i data-lucide="bot" class="size-3.5"></i><span data-agent-count></span></header>
                <ul data-agents class="flex flex-col gap-1"></ul>
            </section></li>`);
            C.groups.set(id, group);
        }
        const list = group.querySelector('[data-agents]');
        const tone = AGENT_TONES[list.children.length % AGENT_TONES.length];
        const input = block.input || {};
        list.insertAdjacentHTML('beforeend', `<li data-agent="${ui.esc(block.id)}">
            <button type="button" data-action="chat-detail" class="group flex w-full items-center gap-3 rounded-2xl px-2 py-2 text-left transition-colors hover:bg-surface0/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">
                ${ui.tile('bot', tone, 'sm')}
                <span class="min-w-0 flex-1"><span class="flex items-center gap-2"><span class="truncate text-sm text-text">${ui.esc(input.description || 'Agent')}</span><span class="shrink-0 text-xs text-overlay0 max-sm:hidden">${ui.esc(input.subagent_type || '')}</span></span><span data-activity class="block truncate font-mono text-xs text-overlay1">Starting</span></span>
                <span data-agent-meta class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums"><span data-start="${ts(m) || Date.now()}"></span></span>
                <span data-state class="flex w-5 shrink-0 justify-end">${ui.spinner(`size-4 text-${tone}`)}</span>
            </button>
            <div data-detail class="hidden pb-2 sm:pl-13"><ul data-nested class="flex flex-col"></ul><div data-summary></div></div>
        </li>`);
        const el = list.lastElementChild;
        ui.icons(el);
        C.agents.set(block.id, { el, group, tone, done: false });
        group.querySelector('[data-agent-count]').textContent = fmt.plural(list.children.length, 'agent');
        group.setAttribute('data-live-task', '');
        C.activity = `Waiting on ${fmt.plural(runningAgents(group), 'agent')}`;
        renderStatus();
        keepDown();
    };

    const runningAgents = (group) => [...C.agents.values()].filter((a) => a.group === group && !a.done).length;

    const agentBlock = (parent, block, m) => {
        const a = C.agents.get(parent);
        if (!a || block.type !== 'tool_use') return;
        const start = ts(m) || Date.now();
        const nested = a.el.querySelector('[data-nested]');
        nested.insertAdjacentHTML('beforeend', toolRow(block, true, start));
        const el = nested.lastElementChild;
        ui.icons(el);
        C.rows.set(block.id, { el, block, start });
        a.el.querySelector('[data-activity]').textContent = `${block.name} ${toolArg(block)}`.trim();
    };

    const agentResult = (a, m, c) => {
        const r = m.tool_use_result && typeof m.tool_use_result === 'object' ? m.tool_use_result : {};
        if (c.is_error) return agentDone(a, { status: 'failed', summary: textOf(c.content) });
        if (r.status === 'completed' || r.totalDurationMs !== undefined) {
            return agentDone(a, { status: 'completed', tools: r.totalToolUseCount, tokens: r.totalTokens, ms: r.totalDurationMs, summary: textOf(r.content) || textOf(c.content) });
        }
        if (!a.done) a.el.querySelector('[data-activity]').textContent = 'Running in the background';
    };

    const agentDone = (a, { status, tools, tokens, ms, summary }) => {
        a.done = true;
        const icon = status === 'completed' ? '<i data-lucide="check" class="size-4 text-overlay0"></i>' : status === 'stopped' || status === 'killed' ? '<i data-lucide="circle-stop" class="size-4 text-peach"></i>' : '<i data-lucide="circle-x" class="size-4 text-red"></i>';
        a.el.querySelector('[data-state]').innerHTML = icon;
        const parts = [];
        if (tools !== undefined) parts.push(`<span class="max-sm:hidden">${fmt.plural(tools, 'tool')}</span>`);
        if (tokens) parts.push(`<span class="max-sm:hidden">${fmt.tokens(tokens)}</span>`);
        if (ms) parts.push(`<span>${fmt.dur(ms)}</span>`);
        a.el.querySelector('[data-agent-meta]').innerHTML = parts.join('');
        const first = String(summary || '').trim();
        if (first) {
            a.el.querySelector('[data-activity]').textContent = first.split('\n')[0];
            const box = a.el.querySelector('[data-summary]');
            box.className = 'mt-2 rounded-2xl bg-base px-4 py-3';
            box.innerHTML = '<div class="markdown-body chat-tile"></div>';
            md.render(box.firstElementChild, first);
        }
        ui.icons(a.el);
        if (!runningAgents(a.group)) a.group.removeAttribute('data-live-task');
        const left = [...C.agents.values()].filter((x) => !x.done).length;
        C.activity = left ? `Waiting on ${fmt.plural(left, 'agent')}` : '';
        renderStatus();
    };

    const workflowTitle = (script = '') => script.match(/description\s*:\s*['"`]([^'"`]+)/)?.[1] || script.match(/name\s*:\s*['"`]([^'"`]+)/)?.[1] || 'Workflow';

    const workflowStart = (m, block) => {
        closeGroup();
        const start = ts(m) || Date.now();
        const li = append(`<li data-workflow="${ui.esc(block.id)}" data-live-task class="${rise()}"><section class="rounded-3xl bg-surface0/50 p-3">
            <header class="flex items-center gap-3 px-1">
                ${ui.tile('workflow', 'lavender', 'sm')}
                <span class="min-w-0 flex-1"><span data-wf-title class="block truncate text-sm text-text">${ui.esc(workflowTitle(block.input?.script))}</span><span data-wf-sub class="block truncate font-mono text-xs text-overlay1">Starting</span></span>
                <span data-wf-state class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums">${runningChip()}<span data-start="${start}"></span></span>
                <span data-wf-stop></span>
            </header>
            <div data-wf-body class="mt-2 flex flex-col gap-2 empty:hidden"></div>
        </section></li>`);
        C.workflows.set(block.id, { el: li, taskId: '', done: false, start });
        keepDown();
    };

    const workflowTask = (wf, m) => {
        wf.taskId = m.task_id;
        if (m.description) wf.el.querySelector('[data-wf-title]').textContent = m.description;
        if (m.workflow_name) wf.el.querySelector('[data-wf-sub]').textContent = m.workflow_name;
        if (!wf.done) paint(wf.el.querySelector('[data-wf-stop]'), ui.iconButton('square', { title: 'Stop workflow', size: 'sm', variant: 'danger', action: 'chat-task-stop', attrs: `data-task="${ui.esc(m.task_id)}"` }));
    };

    const workflowProgress = (wf, progress) => {
        if (!Array.isArray(progress) || !progress.length) return;
        const phases = progress.filter((p) => p.type === 'workflow_phase');
        const agents = progress.filter((p) => p.type === 'workflow_agent');
        const row = (a) => {
            const done = a.state === 'done';
            const failed = ['failed', 'error', 'killed', 'stopped'].includes(a.state);
            const icon = done ? '<i data-lucide="check" class="size-3.5 text-overlay0"></i>' : failed ? '<i data-lucide="circle-x" class="size-3.5 text-red"></i>' : ui.spinner('size-3.5 text-lavender');
            const time = done || failed ? (a.durationMs ? fmt.dur(a.durationMs) : '') : a.startedAt ? `<span data-start="${a.startedAt}"></span>` : '';
            return `<li class="flex items-center gap-3 rounded-2xl px-2 py-1.5"><span class="flex size-4 shrink-0 items-center justify-center">${icon}</span><span title="${ui.esc(a.promptPreview || '')}" class="min-w-0 flex-1 truncate text-sm text-subtext1">${ui.esc(a.label || `Agent ${a.index}`)}</span><span class="shrink-0 text-xs text-overlay0 max-sm:hidden">${ui.esc(shortModel(a.model))}</span>${done && a.tokens ? `<span class="shrink-0 text-xs text-overlay0 tabular-nums max-sm:hidden">${fmt.tokens(a.tokens)}</span>` : ''}<span class="w-14 shrink-0 text-right text-xs text-overlay1 tabular-nums">${time}</span></li>`;
        };
        const section = (title, list) => (list.length ? `<div>${title ? `<p class="${ui.cls.label} px-2 pt-1 pb-1">${ui.esc(title)}</p>` : ''}<ul class="flex flex-col">${list.map(row).join('')}</ul></div>` : '');
        const placed = new Set();
        const body = phases.map((ph) => {
            const list = agents.filter((a) => a.phaseIndex === ph.index);
            list.forEach((a) => placed.add(a));
            return section(ph.title, list);
        }).join('') + section('', agents.filter((a) => !placed.has(a)));
        paint(wf.el.querySelector('[data-wf-body]'), body);
        if (!wf.done) {
            const running = agents.filter((a) => a.state !== 'done').length;
            wf.el.querySelector('[data-wf-sub]').textContent = [fmt.plural(agents.length, 'agent'), running ? `${running} running` : '', phases.length > 1 ? fmt.plural(phases.length, 'phase') : ''].filter(Boolean).join(' · ');
        }
    };

    const workflowDone = (wf, m) => {
        wf.done = true;
        workflowProgress(wf, m.workflow_progress);
        const label = { completed: ['Done', 'neutral'], stopped: ['Stopped', 'peach'], killed: ['Stopped', 'peach'] }[m.status] || ['Failed', 'red'];
        paint(wf.el.querySelector('[data-wf-state]'), `${ui.chip(label[0], { tone: label[1] })}${m.usage?.duration_ms ? `<span>${fmt.dur(m.usage.duration_ms)}</span>` : ''}`);
        paint(wf.el.querySelector('[data-wf-stop]'), '');
        if (m.usage?.total_tokens) wf.el.querySelector('[data-wf-sub]').textContent = `${wf.el.querySelector('[data-wf-sub]').textContent.split(' · ')[0]} · ${fmt.tokens(m.usage.total_tokens)} tokens`;
        wf.el.removeAttribute('data-live-task');
    };

    const bgStart = (m) => {
        closeGroup();
        const command = C.rows.get(m.tool_use_id)?.block.input?.command || '';
        const li = append(`<li data-bg="${ui.esc(m.task_id)}" data-live-task class="${rise()}"><section class="rounded-2xl bg-surface0/50 p-3">
            <header class="flex items-center gap-3">
                ${ui.tile('terminal', 'green', 'sm')}
                <span class="min-w-0 flex-1"><span class="block truncate text-sm text-text">${ui.esc(m.description || 'Background command')}</span><span title="${ui.esc(command)}" class="block truncate font-mono text-xs text-overlay1">${ui.esc(command)}</span></span>
                <span data-bg-state class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums">${runningChip()}<span data-start="${C.lastTs || Date.now()}"></span></span>
                <span data-bg-stop>${ui.iconButton('square', { title: 'Stop', size: 'sm', variant: 'danger', action: 'chat-task-stop', attrs: `data-task="${ui.esc(m.task_id)}"` })}</span>
            </header>
            <pre data-tail data-scroll class="mt-3 hidden max-h-40 overflow-auto rounded-2xl bg-base p-3 font-mono text-xs leading-relaxed text-subtext0"></pre>
        </section></li>`);
        C.tasks.get(m.task_id).card = li;
    };

    const bgDone = (card, status) => {
        if (!card.hasAttribute('data-live-task')) return;
        card.removeAttribute('data-live-task');
        const label = { completed: ['Done', 'neutral'], stopped: ['Stopped', 'peach'], killed: ['Stopped', 'peach'] }[status] || ['Failed', 'red'];
        const start = Number(card.querySelector('[data-bg-state] [data-start]')?.dataset.start || 0);
        paint(card.querySelector('[data-bg-state]'), `${ui.chip(label[0], { tone: label[1] })}${start ? `<span>${fmt.dur((C.replaying ? C.lastTs : Date.now()) - start)}</span>` : ''}`);
        card.querySelector('[data-bg-stop]').remove();
        if (!C.replaying && isLive()) tail(card);
    };

    const tail = async (card) => {
        const id = card.dataset.bg;
        if (C.tailing.has(id)) return;
        C.tailing.add(id);
        const cur = C;
        try {
            const r = await api.taskOutput(C.name, id);
            if (cur !== C || !r?.output) return;
            const pre = card.querySelector('[data-tail]');
            const stick = pre.scrollHeight - pre.scrollTop - pre.clientHeight < 24;
            pre.classList.remove('hidden');
            pre.textContent = `${r.truncated ? '…\n' : ''}${ansi(r.output).replace(/\n$/, '')}`;
            if (stick) pre.scrollTop = pre.scrollHeight;
        } catch {} finally {
            cur.tailing.delete(id);
        }
    };

    const pollTails = () => {
        if (!C || !isLive() || document.visibilityState !== 'visible') return;
        C.log.querySelectorAll(':scope > [data-bg][data-live-task]').forEach((card) => tail(card));
    };

    const todoResult = (m, c) => {
        const block = C.todos.calls.get(c.tool_use_id);
        C.todos.calls.delete(c.tool_use_id);
        if (c.is_error) return;
        const r = m.tool_use_result && typeof m.tool_use_result === 'object' ? m.tool_use_result : {};
        const items = C.todos.items;
        const input = block.input || {};
        if (block.name === 'TaskCreate') {
            if (!r.task?.id) return;
            items.push({ id: r.task.id, subject: r.task.subject || input.subject, activeForm: input.activeForm, status: 'pending' });
            return renderTodos(false);
        }
        if (!r.success) return;
        const i = items.findIndex((t) => t.id === (r.taskId || input.taskId));
        if (i < 0) return;
        if (input.status === 'deleted') items.splice(i, 1);
        else Object.assign(items[i], { status: input.status || items[i].status, subject: input.subject || items[i].subject, activeForm: input.activeForm || items[i].activeForm });
        renderTodos(true);
    };

    const todoRow = (t) => {
        const icon = { completed: 'circle-check', in_progress: 'circle-dot' }[t.status] || 'circle';
        const tone = { completed: 'text-green', in_progress: 'text-mauve' }[t.status] || 'text-overlay0';
        const label = t.status === 'in_progress' && t.activeForm ? t.activeForm : t.subject;
        return `<li class="flex items-center gap-3 px-1 py-1 text-sm"><i data-lucide="${icon}" class="size-4 shrink-0 ${tone}"></i><span class="min-w-0 truncate ${t.status === 'completed' ? 'text-overlay0 line-through' : t.status === 'in_progress' ? 'text-text' : 'text-subtext0'}">${ui.esc(label)}</span></li>`;
    };

    const todoCard = (items) => {
        const done = items.filter((t) => t.status === 'completed').length;
        return `<section class="rounded-3xl bg-surface0/50 p-4">
            <header class="mb-2 flex items-center gap-3"><i data-lucide="list-todo" class="size-4 text-overlay1"></i><span class="flex-1">${ui.meter(items.length ? (done / items.length) * 100 : 0, done === items.length ? 'green' : 'mauve', 'h-1.5')}</span><span class="text-xs text-overlay1 tabular-nums">${done}/${items.length}</span></header>
            <ul class="flex flex-col gap-0.5">${items.map(todoRow).join('')}</ul>
        </section>`;
    };

    const renderTodos = (progress) => {
        const t = C.todos;
        if (!t.items.length) return;
        if (t.card?.isConnected) {
            paint(t.card, todoCard(t.items));
            if (progress && C.log.lastElementChild !== t.card) {
                const now = t.items.find((x) => x.status === 'in_progress');
                const done = t.items.filter((x) => x.status === 'completed').length;
                closeGroup();
                append(`<li class="flex items-center gap-2.5 px-2 text-sm ${rise()}"><i data-lucide="${now ? 'circle-dot' : 'circle-check'}" class="size-3.5 shrink-0 ${now ? 'text-mauve' : 'text-green'}"></i><span class="min-w-0 truncate ${now ? 'text-subtext1' : 'text-overlay1'}">${ui.esc(now ? now.activeForm || now.subject : done === t.items.length ? 'All to-dos done' : 'To-dos updated')}</span><span class="shrink-0 text-xs text-overlay0 tabular-nums">${done}/${t.items.length}</span></li>`);
            }
            return;
        }
        closeGroup();
        t.card = append(`<li class="${rise()}"></li>`);
        paint(t.card, todoCard(t.items));
        keepDown();
    };

    const commandTile = (body) => {
        if (!body.trim()) return;
        closeGroup();
        const li = append(`<li class="rounded-2xl bg-surface0/50 px-4 py-3 ${rise()}"><article class="markdown-body chat-tile"></article></li>`);
        const literal = ansi(body).split(/(```[\s\S]*?```|`[^`\n]*`)/).map((part, i) => (i % 2 ? part : part.replace(/</g, '&lt;'))).join('');
        md.render(li.querySelector('article'), literal, { breaks: true });
        keepDown();
    };

    const localCommand = (m) => {
        const u = m.context_usage;
        if (u) return contextCard({ total: u.total_tokens, max: u.raw_max_tokens || u.max_tokens, categories: u.categories || [] });
        commandTile(textOf(m.message?.content));
    };

    const usedCategories = (list) => (list || []).filter((x) => x.kind === 'used' || x.kind === 'buffer').map((x) => ({ ...x, tone: CONTEXT_TONES[x.id || slug(x.name)] || 'overlay0' }));

    const contextCard = ({ total, max, categories }) => {
        if (!max) return;
        closeGroup();
        const used = usedCategories(categories);
        append(`<li class="${rise()}"><section class="rounded-3xl bg-surface0/50 p-4">
            <header class="mb-3 flex items-baseline gap-2"><span class="font-display text-2xl text-text tabular-nums">${fmt.tokens(total)}</span><span class="text-sm text-overlay1">of ${fmt.tokens(max)}</span><span class="ml-auto text-sm text-overlay1 tabular-nums">${Math.round((total / max) * 100)}%</span></header>
            <span class="flex h-2.5 gap-0.5 overflow-hidden rounded-full bg-surface0">${used.map((x) => `<span class="h-full bg-${x.tone}" style="width:${Math.max(0.4, (x.tokens / max) * 100)}%"></span>`).join('')}</span>
            <dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-3">${used.map((x) => `<div class="flex min-w-0 items-center gap-2">${ui.dot(x.tone)}<dt class="truncate text-overlay1">${ui.esc(x.name)}</dt><dd class="ml-auto text-subtext1 tabular-nums">${fmt.tokens(x.tokens)}</dd></div>`).join('')}</dl>
        </section></li>`);
        keepDown();
    };

    const result = (m) => {
        closeGroup();
        const { interrupted, tools } = C.turn;
        C.turn.interrupted = false;
        C.turn.tools = 0;
        if (m.local_command) return;
        if (m.subtype !== 'success') {
            if (!interrupted) {
                const labels = { error_during_execution: 'The turn ended with an error', error_max_turns: 'Stopped at the turn limit', error_max_budget_usd: 'Stopped at the budget limit' };
                notice('octagon-x', labels[m.subtype] || 'The turn failed', 'red');
            }
            return;
        }
        if (m.num_turns === 0) return;
        const usage = m.usage || {};
        const tokens = (usage.output_tokens || 0) + (usage.input_tokens || 0);
        append(`<li data-footer class="group -mt-2 flex items-center gap-3 px-1 text-xs text-overlay0 tabular-nums">
            ${m.duration_ms ? `<span class="flex items-center gap-1"><i data-lucide="clock" class="size-3"></i>${fmt.dur(m.duration_ms)}</span>` : ''}
            ${tools ? `<span class="flex items-center gap-1"><i data-lucide="wrench" class="size-3"></i>${tools}</span>` : ''}
            ${tokens ? `<span class="flex items-center gap-1"><i data-lucide="arrow-down" class="size-3"></i>${fmt.tokens(tokens)}</span>` : ''}
            <span class="${ui.cls.reveal}">${ui.iconButton('copy', { title: 'Copy reply', size: 'sm', action: 'chat-copy-reply' })}</span>
        </li>`);
    };

    const conversationReset = () => {
        closeGroup();
        const keep = [...C.log.querySelectorAll(':scope > [data-live-task]')];
        C.log.innerHTML = '';
        C.todos = { items: [], calls: new Map(), card: null };
        notice('eraser', 'New conversation');
        keep.forEach((li) => C.log.appendChild(li));
    };

    const system = {
        session_state_changed(m) {
            const prev = C.turnState;
            C.turnState = m.state;
            if (m.state === 'running' && prev !== 'running' && prev !== 'requires_action') {
                C.turn.start = C.replaying ? C.lastTs || Date.now() : Date.now();
                C.thinking = 0;
                C.activity = '';
            }
            if (m.state === 'idle') {
                C.activity = '';
                C.thinking = 0;
            }
            renderStatus();
        },
        thinking_tokens(m) {
            C.thinking = m.estimated_tokens || 0;
            C.activity = 'Thinking';
            renderStatus();
        },
        compact_boundary(m) {
            const meta = m.compact_metadata || {};
            notice('shrink', meta.pre_tokens ? `Compacted ${fmt.tokens(meta.pre_tokens)} to ${fmt.tokens(meta.post_tokens || 0)}` : 'Compacted');
        },
        task_started(m) {
            C.tasks.set(m.task_id, { type: m.task_type, description: m.description, toolUseId: m.tool_use_id });
            if (m.task_type === 'local_bash' && m.is_backgrounded !== false) bgStart(m);
            const wf = C.workflows.get(m.tool_use_id);
            if (wf) workflowTask(wf, m);
            const a = C.agents.get(m.tool_use_id);
            if (a) a.taskId = m.task_id;
        },
        task_progress(m) {
            const a = C.agents.get(m.tool_use_id);
            if (a && !a.done && m.description) a.el.querySelector('[data-activity]').textContent = m.description;
            const wf = C.workflows.get(m.tool_use_id);
            if (wf) workflowProgress(wf, m.workflow_progress);
        },
        task_notification(m) {
            const t = C.tasks.get(m.task_id);
            const a = C.agents.get(m.tool_use_id);
            if (a) agentDone(a, { status: m.status, tools: m.usage?.tool_uses, tokens: m.usage?.total_tokens, ms: m.usage?.duration_ms, summary: m.summary });
            const wf = C.workflows.get(m.tool_use_id);
            if (wf) workflowDone(wf, m);
            if (t?.card) {
                bgDone(t.card, m.status);
                notice(m.status === 'completed' ? 'circle-check' : 'circle-stop', `${t.description} ${m.status === 'killed' ? 'stopped' : m.status}`, m.status === 'completed' ? 'overlay1' : 'peach');
            }
            renderStatus();
        },
        background_tasks_changed(m) {
            const running = new Set((m.tasks || []).map((t) => t.task_id));
            for (const [id, t] of C.tasks) {
                if (t.card && !running.has(id)) bgDone(t.card, 'completed');
            }
        },
        local_command_output(m) {
            commandTile(String(m.content || ''));
        },
        informational(m) {
            notice(m.level === 'warning' ? 'triangle-alert' : 'info', ansi(m.content || m.message || ''), m.level === 'warning' ? 'peach' : 'overlay1');
        },
        api_retry(m) {
            notice('refresh-cw', `Retrying, attempt ${m.attempt} of ${m.max_retries}`, 'peach');
        },
        permission_denied(m) {
            notice('shield-x', `${m.tool_name || 'A tool'} was denied`, 'peach');
        },
        model_refusal_fallback(m) {
            notice('sparkles', m.fallback_model ? `Fell back to ${m.fallback_model}` : 'The model declined and fell back', 'peach');
        },
        session_writes_lost() {
            notice('triangle-alert', 'Some transcript writes were lost', 'peach');
        },
        commands_changed(m) {
            if (C.init && Array.isArray(m.commands)) C.init.commands = m.commands;
        },
    };

    const typing = () => document.activeElement === C.input && C.input.value.trim() !== '';

    const cardFrame = (id, icon, title, sub, body, footer) => `<li data-approval="${ui.esc(id)}" class="${rise()}"><section data-card class="rounded-3xl bg-yellow/10 p-4 transition-colors">
        <header class="mb-3 flex items-start gap-3">${ui.tile(icon, 'yellow', 'sm')}<div class="min-w-0 flex-1 pt-1.5"><h3 data-card-title class="text-sm font-medium text-text wrap-anywhere">${title}</h3>${sub}</div></header>
        <div data-card-body>${body}</div>
        <footer data-decide class="mt-3 flex flex-wrap items-center justify-end gap-2">${footer}</footer>
    </section></li>`;

    const requestMeta = (r) => {
        const rows = [];
        if (r.agent_id) {
            const t = C.tasks.get(r.agent_id);
            rows.push(['bot', t ? `Asked by subagent ${t.description}` : 'Asked by a subagent']);
        }
        if (r.mcp_server) rows.push(['plug', `MCP server ${r.mcp_server.name || r.mcp_server}`]);
        if (r.decision_reason) rows.push(['shield-alert', ansi(r.decision_reason)]);
        if (r.blocked_path) rows.push(['folder-lock', r.blocked_path, true]);
        return rows.length ? `<ul class="mt-3 flex flex-col gap-1.5">${rows.map(([icon, t, mono]) => `<li class="flex items-start gap-2 text-xs text-overlay1"><i data-lucide="${icon}" class="mt-px size-3.5 shrink-0"></i><span class="min-w-0 wrap-anywhere ${mono ? 'font-mono' : ''}">${ui.esc(t)}</span></li>`).join('')}</ul>` : '';
    };

    const requestWell = (tool, input) => {
        if (tool === 'Bash') return well(`<span class="text-mauve">$</span> <span class="text-text">${ui.esc(input.command)}</span>`, 'max-h-56');
        if (tool === 'Write') return `<p class="mb-2 truncate px-1 font-mono text-xs text-subtext1">${ui.esc(shortPath(input.file_path))}</p>${diff([{ newStart: 1, lines: String(input.content || '').replace(/\n$/, '').split('\n').map((l) => `+${l}`) }])}`;
        if (tool === 'Edit') return `<p class="mb-2 truncate px-1 font-mono text-xs text-subtext1">${ui.esc(shortPath(input.file_path))}</p>${diff([{ newStart: 1, lines: [...String(input.old_string || '').split('\n').map((l) => `-${l}`), ...String(input.new_string || '').split('\n').map((l) => `+${l}`)] }])}`;
        if (tool === 'WebFetch') return well(`<span class="text-blue">${ui.esc(input.url)}</span>${input.prompt ? `\n${ui.esc(input.prompt)}` : ''}`, 'max-h-56');
        if (tool === 'Read') return well(ui.esc(shortPath(input.file_path)), 'max-h-56');
        return well(ui.esc(JSON.stringify(input, null, 2)), 'max-h-56');
    };

    const suggestion = (r) => {
        const list = r.permission_suggestions || [];
        if (!list.length || r.suppress_always_allow_rule) return null;
        const s = list[0];
        if (s.type === 'setMode') return { label: s.mode === 'acceptEdits' ? 'Allow, and accept edits this session' : `Allow, and switch to ${modeLabel(s.mode)}` };
        if (s.type === 'addDirectories') return { label: 'Allow, and add the directory', hint: (s.directories || []).join(', ') };
        if (s.type === 'addRules') {
            const rules = s.rules || [];
            const domain = rules.length === 1 && /^domain:/.test(rules[0].ruleContent || '') ? rules[0].ruleContent.slice(7) : '';
            return { label: domain ? `Always allow ${domain}` : 'Always allow', hint: rules.map((x) => (x.ruleContent ? `${x.toolName}(${x.ruleContent})` : x.toolName)).join(', ') };
        }
        return { label: 'Always allow' };
    };

    const approval = (m) => {
        const r = m.request || {};
        if (!m.request_id || C.approvals.has(m.request_id)) return;
        closeGroup();
        let card;
        if (r.subtype === 'elicitation') card = elicitationCard(m);
        else if (r.tool_name === 'AskUserQuestion') card = questionCard(m);
        else if (r.tool_name === 'ExitPlanMode') card = planCard(m);
        else card = actionCard(m);
        C.approvals.set(m.request_id, { el: card, request: m, resolved: false, busy: false });
        if (!C.replaying) {
            scrollDown();
            if (!r.default_to_no && !typing()) card.querySelector('[data-focus]')?.focus({ preventScroll: true });
        }
    };

    const actionCard = (m) => {
        const r = m.request;
        const input = r.input || {};
        const title = r.title ? ui.esc(ansi(r.title)) : `Allow ${ui.esc(r.display_name || r.tool_name)}?`;
        const sub = r.description ? `<p class="mt-0.5 text-xs text-overlay1 wrap-anywhere">${ui.esc(ansi(r.description))}</p>` : '';
        const always = suggestion(r);
        const footer = `${ui.button('Deny', { variant: 'ghost', size: 'sm', attrs: 'data-decision="deny"' })}
            ${always ? ui.button(always.label, { size: 'sm', attrs: `data-decision="always" ${always.hint ? `title="${ui.esc(always.hint)}"` : ''}` }) : ''}
            ${ui.button('Allow', { icon: 'check', variant: r.default_to_no ? 'secondary' : 'primary', size: 'sm', attrs: 'data-decision="allow" data-focus' })}`;
        const li = append(cardFrame(m.request_id, 'shield-question', title, sub, requestWell(r.tool_name, input) + requestMeta(r), footer));
        li.addEventListener('click', (e) => {
            const b = e.target.closest('[data-decision]');
            if (!b || b.disabled) return;
            const d = b.dataset.decision;
            if (d === 'deny') decide(m.request_id, { behavior: 'deny', message: 'The user denied this in claudex.' });
            else if (d === 'always') decide(m.request_id, { behavior: 'allow', updatedPermissions: r.permission_suggestions });
            else decide(m.request_id, { behavior: 'allow' });
        });
        return li;
    };

    const planCard = (m) => {
        const r = m.request;
        const input = r.input || {};
        const always = suggestion(r);
        const body = `<div data-scroll class="max-h-[28rem] overflow-y-auto rounded-2xl bg-base px-5 py-4"><article class="markdown-body chat-tile"></article></div>
            ${input.planFilePath ? `<p title="${ui.esc(input.planFilePath)}" class="mt-2 truncate px-1 font-mono text-xs text-overlay1">${ui.esc(input.planFilePath)}</p>` : ''}
            <div data-plan-feedback hidden class="mt-3"><textarea data-plan-text rows="3" spellcheck="false" class="${ui.cls.input} resize-none leading-relaxed" placeholder="What should change in the plan?"></textarea></div>
            ${requestMeta(r)}`;
        const footer = `${ui.button('Keep planning', { variant: 'ghost', size: 'sm', attrs: 'data-plan="keep"' })}
            ${always ? ui.button(always.label.replace(/^Allow/, 'Approve'), { size: 'sm', attrs: 'data-plan="always"' }) : ''}
            ${ui.button('Approve', { icon: 'check', variant: 'primary', size: 'sm', attrs: 'data-plan="approve" data-focus' })}`;
        const li = append(cardFrame(m.request_id, 'list-checks', 'Approve the plan?', '', body, footer));
        md.render(li.querySelector('article'), String(input.plan || ''));
        li.addEventListener('click', (e) => {
            const b = e.target.closest('[data-plan]');
            if (!b || b.disabled) return;
            const box = li.querySelector('[data-plan-feedback]');
            if (b.dataset.plan === 'keep') {
                if (box.hidden) {
                    box.hidden = false;
                    b.querySelector('span').textContent = 'Send feedback';
                    box.querySelector('textarea').focus();
                    return;
                }
                const note = box.querySelector('textarea').value.trim();
                decide(m.request_id, { behavior: 'deny', message: note || 'Keep planning.' });
            } else if (b.dataset.plan === 'always') {
                decide(m.request_id, { behavior: 'allow', updatedInput: input, updatedPermissions: r.permission_suggestions });
            } else {
                decide(m.request_id, { behavior: 'allow', updatedInput: input });
            }
        });
        return li;
    };

    const questionCard = (m) => {
        const qs = m.request.input?.questions || [];
        const li = append(cardFrame(m.request_id, 'message-circle-question', ui.esc(qs[0]?.question || 'Question'), '', '<div data-q></div>', ''));
        li._q = { step: 0, focus: null, answers: qs.map(() => ({ picked: [], other: '', otherOn: false })) };
        drawQuestion(li, m);
        li.addEventListener('click', (e) => onQuestion(li, m, e));
        li.addEventListener('input', (e) => {
            if (!e.target.matches('[data-q-other-text]')) return;
            li._q.answers[li._q.step].other = e.target.value;
            drawQuestionFooter(li, m);
        });
        li.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && e.target.matches('[data-q-other-text]')) {
                e.preventDefault();
                li.querySelector('[data-q-act="next"], [data-q-act="submit"]')?.click();
            }
        });
        const preview = (e) => {
            const opt = e.target.closest?.('[data-q-opt]');
            if (!opt || li._q.focus === opt.dataset.qOpt) return;
            li._q.focus = opt.dataset.qOpt;
            drawPreview(li, m);
        };
        li.addEventListener('mouseover', preview);
        li.addEventListener('focusin', preview);
        return li;
    };

    const answered = (a) => a.picked.length > 0 || (a.otherOn && a.other.trim() !== '');
    const answerText = (a) => [...a.picked, ...(a.otherOn && a.other.trim() ? [a.other.trim()] : [])].join(', ');

    const drawQuestion = (li, m) => {
        const qs = m.request.input.questions;
        const st = li._q;
        const q = qs[st.step];
        const a = st.answers[st.step];
        const multi = !!q.multiSelect;
        li.querySelector('[data-card-title]').textContent = q.question;
        const mark = (on) => (multi
            ? `<span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-lg text-crust transition-colors ${on ? 'bg-mauve' : 'bg-surface1'}"><i data-lucide="check" class="size-3.5 ${on ? '' : 'opacity-0'}" stroke-width="3"></i></span>`
            : `<span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full transition-colors ${on ? 'bg-mauve' : 'bg-surface1'}"><span class="size-2 rounded-full bg-crust ${on ? '' : 'scale-0'}"></span></span>`);
        const tile = (attrs, on, label, desc) => `<button type="button" ${attrs} aria-pressed="${on}" class="flex items-start gap-3 rounded-2xl p-3 text-left transition-colors ${on ? 'bg-mauve/10' : 'bg-surface0/50 hover:bg-surface0'} focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40">${mark(on)}<span class="min-w-0"><span class="block text-sm font-medium text-text">${ui.esc(label)}</span>${desc ? `<span class="mt-0.5 block text-xs text-overlay1">${ui.esc(desc)}</span>` : ''}</span></button>`;
        const chips = qs.length > 1 ? `<div class="mb-3 flex flex-wrap gap-1.5">${qs.map((x, i) => `<button type="button" data-q-step="${i}" class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition-colors ${i === st.step ? 'bg-mauve/15 text-mauve' : 'bg-surface0 text-subtext0 hover:bg-surface1'}">${answered(st.answers[i]) ? '<i data-lucide="check" class="size-3"></i>' : ''}${ui.esc(x.header || `Question ${i + 1}`)}</button>`).join('')}</div>` : '';
        const opts = (q.options || []).map((o, i) => tile(`data-q-opt="${ui.esc(o.label)}" ${i === 0 ? 'data-focus' : ''}`, a.picked.includes(o.label), o.label, o.description)).join('');
        li.querySelector('[data-q]').innerHTML = `${chips}${multi ? '<p class="mb-2 px-1 text-xs text-overlay1">Pick any that apply.</p>' : ''}
            <div class="grid gap-2 sm:grid-cols-2">${opts}${tile('data-q-other', a.otherOn, 'Other', 'Type your own answer')}</div>
            ${a.otherOn ? `<input data-q-other-text class="${ui.cls.input} mt-2" value="${ui.esc(a.other)}" placeholder="Your answer" autocomplete="off" spellcheck="false">` : ''}
            <div data-q-preview></div>`;
        drawPreview(li, m);
        drawQuestionFooter(li, m);
        ui.icons(li);
    };

    const drawPreview = (li, m) => {
        const q = m.request.input.questions[li._q.step];
        const a = li._q.answers[li._q.step];
        const label = li._q.focus || a.picked[0];
        const p = (q.options || []).find((o) => o.label === label)?.preview;
        const host = li.querySelector('[data-q-preview]');
        if (host) host.innerHTML = p ? `<div class="mt-3">${well(ui.esc(p), 'max-h-64')}</div>` : '';
    };

    const drawQuestionFooter = (li, m) => {
        const qs = m.request.input.questions;
        const st = li._q;
        const last = st.step === qs.length - 1;
        const footer = li.querySelector('[data-decide]');
        footer.innerHTML = `${ui.button('Dismiss', { variant: 'ghost', size: 'sm', attrs: 'data-q-act="dismiss"' })}
            ${st.step > 0 ? ui.button('Back', { variant: 'ghost', size: 'sm', icon: 'chevron-left', attrs: 'data-q-act="back"' }) : ''}
            ${last
                ? ui.button('Submit', { icon: 'check', variant: 'primary', size: 'sm', disabled: !st.answers.every(answered), attrs: 'data-q-act="submit"' })
                : ui.button('Next', { icon: 'chevron-right', size: 'sm', disabled: !answered(st.answers[st.step]), attrs: 'data-q-act="next"' })}`;
        ui.icons(footer);
    };

    const onQuestion = (li, m, e) => {
        const a = C.approvals.get(m.request_id);
        if (!a || a.resolved || a.busy) return;
        const qs = m.request.input.questions;
        const st = li._q;
        const q = qs[st.step];
        const ans = st.answers[st.step];
        const step = e.target.closest('[data-q-step]');
        const opt = e.target.closest('[data-q-opt]');
        const other = e.target.closest('[data-q-other]');
        const act = e.target.closest('[data-q-act]');
        if (step) st.step = Number(step.dataset.qStep);
        else if (opt) {
            const label = opt.dataset.qOpt;
            if (q.multiSelect) ans.picked = ans.picked.includes(label) ? ans.picked.filter((x) => x !== label) : [...ans.picked, label];
            else {
                ans.picked = [label];
                ans.otherOn = false;
                if (st.step < qs.length - 1) st.step += 1;
            }
        } else if (other) {
            ans.otherOn = q.multiSelect ? !ans.otherOn : true;
            if (!q.multiSelect) ans.picked = [];
        } else if (act?.dataset.qAct === 'back') st.step -= 1;
        else if (act?.dataset.qAct === 'next') st.step += 1;
        else if (act?.dataset.qAct === 'dismiss') return decide(m.request_id, { behavior: 'deny', message: 'The user dismissed the question.' });
        else if (act?.dataset.qAct === 'submit') {
            const answers = Object.fromEntries(qs.map((x, i) => [x.question, answerText(st.answers[i])]));
            return decide(m.request_id, { behavior: 'allow', updatedInput: { ...m.request.input, answers } });
        } else return;
        st.focus = null;
        drawQuestion(li, m);
        li.querySelector('[data-q-other-text]')?.focus();
    };

    const elicitationCard = (m) => {
        const r = m.request;
        const id = m.request_id;
        const title = ui.esc(ansi(r.title || r.display_name || r.mcp_server_name || 'MCP server'));
        const sub = `<p class="mt-0.5 text-xs text-overlay1">${ui.esc(r.mcp_server_name ? `MCP server ${r.mcp_server_name}` : 'MCP server')}</p>`;
        const message = r.message ? `<p class="mb-3 text-sm leading-relaxed whitespace-pre-wrap text-subtext0">${ui.esc(ansi(r.message))}</p>` : '';
        let body = message;
        let footer;
        if (r.mode === 'url') {
            body += r.url ? `<a href="${ui.esc(r.url)}" target="_blank" rel="noopener noreferrer" class="block truncate rounded-2xl bg-base px-4 py-3 font-mono text-xs text-blue hover:underline">${ui.esc(r.url)}</a>` : '';
            footer = `${ui.button('Decline', { variant: 'ghost', size: 'sm', attrs: 'data-el="decline"' })}${r.url ? ui.button('Open link', { icon: 'external-link', size: 'sm', attrs: 'data-el="open"' }) : ''}${ui.button('Done', { icon: 'check', variant: 'primary', size: 'sm', attrs: 'data-el="accept" data-focus' })}`;
        } else {
            const schema = r.requested_schema || {};
            const props = schema.properties || {};
            const order = r.property_order?.length ? r.property_order : Object.keys(props);
            const required = new Set(schema.required || []);
            const field = (key) => {
                const p = props[key] || {};
                const label = `${p.title || key}${required.has(key) ? ' *' : ''}`;
                const name = `el:${id}:${key}`;
                let control;
                if (Array.isArray(p.enum)) control = ui.select({ name, value: p.default !== undefined ? String(p.default) : '', placeholder: 'Choose', options: p.enum.map((v, i) => ({ value: String(v), label: p.enumNames?.[i] || String(v) })) });
                else if (p.type === 'boolean') control = `<span class="flex h-10 items-center">${ui.toggle({ name, checked: !!p.default, label })}</span>`;
                else control = `<input data-name="${ui.esc(name)}" type="${p.type === 'number' || p.type === 'integer' ? 'number' : p.format === 'email' ? 'email' : 'text'}" value="${ui.esc(p.default ?? '')}" class="${ui.cls.input}" autocomplete="off" spellcheck="false">`;
                return `<div class="flex flex-col gap-1.5" data-el-field="${ui.esc(key)}"><span class="${ui.cls.label} px-1">${ui.esc(label)}</span>${control}${p.description ? `<span class="px-1 text-xs text-overlay1">${ui.esc(p.description)}</span>` : ''}</div>`;
            };
            body += `<div class="grid gap-3 sm:grid-cols-2">${order.map(field).join('')}</div>`;
            footer = `${ui.button('Decline', { variant: 'ghost', size: 'sm', attrs: 'data-el="decline"' })}${ui.button('Submit', { icon: 'check', variant: 'primary', size: 'sm', attrs: 'data-el="accept" data-focus' })}`;
            const li = append(cardFrame(id, 'plug', title, sub, body, footer));
            li.addEventListener('click', (e) => {
                const b = e.target.closest('[data-el]');
                if (!b || b.disabled) return;
                if (b.dataset.el === 'decline') return decide(id, { action: 'decline' });
                const content = {};
                let missing = false;
                for (const key of order) {
                    const p = props[key] || {};
                    const raw = ui.value(li, `el:${id}:${key}`);
                    const box = li.querySelector(`[data-el-field="${CSS.escape(key)}"] input, [data-el-field="${CSS.escape(key)}"] [data-ui="select"]`);
                    let v = raw;
                    if (p.type === 'number' || p.type === 'integer') v = raw === '' ? undefined : Number(raw);
                    else if (p.type !== 'boolean' && raw === '') v = undefined;
                    if (v === undefined && required.has(key)) {
                        missing = true;
                        box?.setAttribute('aria-invalid', 'true');
                        box?.classList.add('ring-2', 'ring-red/50');
                    } else box?.classList.remove('ring-2', 'ring-red/50');
                    if (v !== undefined) content[key] = v;
                }
                if (!missing) decide(id, { action: 'accept', content });
            });
            return li;
        }
        const li = append(cardFrame(id, 'plug', title, sub, body, footer));
        li.addEventListener('click', (e) => {
            const b = e.target.closest('[data-el]');
            if (!b || b.disabled) return;
            if (b.dataset.el === 'open') return window.open(r.url, '_blank', 'noopener,noreferrer');
            decide(id, { action: b.dataset.el });
        });
        return li;
    };

    const setBusy = (el, on) => el.querySelectorAll('button, input, textarea').forEach((b) => { b.disabled = on; });

    const decide = async (id, payload) => {
        const a = C.approvals.get(id);
        if (!a || a.resolved || a.busy) return;
        a.busy = true;
        setBusy(a.el, true);
        const cur = C;
        try {
            await api.answer(C.name, id, payload);
            if (cur === C) resolveApproval({ request_id: id, response: payload });
        } catch (err) {
            if (cur !== C) return;
            a.busy = false;
            if (a.resolved) return;
            setBusy(a.el, false);
            if (/no such pending/i.test(err.message)) return;
            ui.toast(`Answer failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
    };

    const outcome = (r, data) => {
        if (data.cancelled) return ['Withdrawn', 'neutral', 'undo-2'];
        const res = data.response || {};
        if (r.subtype === 'elicitation') return { accept: ['Submitted', 'green', 'check'], decline: ['Declined', 'red', 'x'] }[res.action] || ['Cancelled', 'neutral', 'x'];
        const allow = res.behavior === 'allow';
        if (r.tool_name === 'AskUserQuestion') return allow ? ['Answered', 'green', 'check'] : ['Dismissed', 'neutral', 'x'];
        if (r.tool_name === 'ExitPlanMode') return allow ? ['Approved', 'green', 'check'] : ['Kept planning', 'neutral', 'pencil'];
        if (!allow) return ['Denied', 'red', 'x'];
        const perms = res.updatedPermissions || [];
        if (perms[0]?.type === 'setMode' && perms[0].mode === 'acceptEdits') return ['Allowed, accepting edits', 'green', 'check'];
        return perms.length ? ['Always allowed', 'green', 'check-check'] : ['Allowed', 'green', 'check'];
    };

    const resolveApproval = (data) => {
        const a = C.approvals.get(data.request_id);
        if (!a || a.resolved) return;
        a.resolved = true;
        const r = a.request.request || {};
        const res = data.response || {};
        const card = a.el.querySelector('[data-card]');
        card.classList.replace('bg-yellow/10', 'bg-surface0/50');
        const [label, tone, icon] = outcome(r, data);
        const answers = r.tool_name === 'AskUserQuestion' ? res.updatedInput?.answers : null;
        if (answers) {
            const qs = r.input?.questions || [];
            a.el.querySelector('[data-card-title]').textContent = qs.length > 1 ? `${qs.length} questions` : qs[0]?.question || 'Question';
            a.el.querySelector('[data-card-body]').innerHTML = `<dl class="flex flex-col gap-2">${qs.map((q) => `<div class="flex flex-wrap items-baseline gap-2">${qs.length > 1 ? `<dt>${ui.chip(q.header || q.question)}</dt>` : ''}<dd class="min-w-0 text-sm text-text wrap-anywhere">${ui.esc(answers[q.question] ?? '')}</dd></div>`).join('')}</dl>`;
        } else if (r.tool_name === 'AskUserQuestion') {
            a.el.querySelector('[data-card-body]').classList.add('opacity-50');
        }
        const note = res.behavior === 'deny' && r.tool_name === 'ExitPlanMode' && res.message ? `<p class="mr-auto min-w-0 text-xs text-overlay1 italic wrap-anywhere">${ui.esc(res.message)}</p>` : '';
        a.el.querySelectorAll('[data-plan-feedback]').forEach((f) => { f.hidden = true; });
        setBusy(a.el.querySelector('[data-card-body]'), true);
        const footer = a.el.querySelector('[data-decide]');
        footer.innerHTML = `${note}${ui.chip(label, { tone, icon })}`;
        ui.icons(a.el);
    };

    const tick = () => {
        if (!C) return;
        const now = Date.now();
        document.querySelectorAll('[data-start]').forEach((n) => { n.textContent = fmt.dur(now - Number(n.dataset.start)); });
        const t = C.pane.querySelector('[data-turn-elapsed]');
        if (t && C.turn.start) t.textContent = fmt.dur(now - C.turn.start);
    };

    const autosize = () => {
        C.input.style.height = 'auto';
        C.input.style.height = `${Math.min(C.input.scrollHeight, 192)}px`;
    };

    const submit = async () => {
        const value = C.input.value.trim();
        if (!value || C.snap?.state !== 'idle' || C.sending) return;
        const cur = C;
        C.sending = true;
        C.input.value = '';
        autosize();
        slash.close();
        renderComposer();
        try {
            await api.prompt(C.name, value);
        } catch (err) {
            if (cur === C && !C.input.value) {
                C.input.value = value;
                autosize();
            }
            ui.toast(`Send failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
        if (cur !== C) return;
        C.sending = false;
        renderComposer();
    };

    const interrupt = async () => {
        if (!C) return;
        try {
            await api.interrupt(C.name);
        } catch (err) {
            ui.toast(`Stop failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
    };

    const stopTask = async (id) => {
        if (!C) return;
        try {
            await api.stopTask(C.name, id);
        } catch (err) {
            ui.toast(`Stop failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
    };

    const onPicker = async (e) => {
        const map = { 'chat-model': ['model', api.setModel], 'chat-effort': ['effort', api.setEffort], 'chat-mode': ['permission mode', api.setPermissionMode] };
        const entry = map[e.detail.name];
        if (!entry || !C) return;
        const cur = C;
        try {
            await entry[1](C.name, e.detail.value);
        } catch (err) {
            ui.toast(`Changing the ${entry[0]} failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            if (cur === C) {
                ['[data-chat-model]', '[data-chat-effort]', '[data-chat-mode]'].forEach((s) => { $(s)._html = ''; });
                renderComposer();
            }
        }
    };

    const models = () => {
        const list = (C.init?.models || []).map((m) => ({ value: m.value, label: m.displayName || m.value, hint: m.description, efforts: m.supportedEffortLevels }));
        const cur = C.snap?.model || 'default';
        if (!list.some((m) => m.value === cur)) list.unshift({ value: cur, label: cur });
        return list;
    };

    const renderComposer = () => {
        if (!C) return;
        const snap = C.snap;
        const ok = isLive();
        const busy = BUSY.has(snap?.state);
        C.input.disabled = !ok;
        C.input.placeholder = ok ? 'Message, or / for commands' : snap?.state === 'starting' ? 'Starting claude' : 'No session is running';
        paint('[data-chat-send]', busy
            ? ui.iconButton('square', { title: 'Stop', variant: 'danger', action: 'chat-interrupt' })
            : ui.iconButton(C.sending ? 'loader-circle' : 'arrow-up', { title: 'Send', variant: 'primary', spin: C.sending, action: 'chat-send', disabled: !ok || C.sending }));
        const list = models();
        const model = list.find((m) => m.value === (snap?.model || 'default'));
        const efforts = model?.efforts?.length ? model.efforts : Object.keys(EFFORTS);
        const off = !ok ? 'disabled' : '';
        paint('[data-chat-model]', ui.select({ name: 'chat-model', value: snap?.model || 'default', pill: true, icon: 'sparkles', options: list, attrs: off }));
        paint('[data-chat-effort]', ui.select({ name: 'chat-effort', value: snap?.effort || '', placeholder: 'Effort', pill: true, icon: 'gauge', options: efforts.map((v) => ({ value: v, label: EFFORTS[v] || v })), attrs: off }));
        paint('[data-chat-mode]', ui.select({ name: 'chat-mode', value: snap?.permissionMode || 'default', pill: true, icon: MODES.find((x) => x.value === (snap?.permissionMode || 'default'))?.icon || 'shield', options: MODES, attrs: off }));
        paint('[data-chat-ring]', ringHtml());
    };

    const ringHtml = () => {
        const c = C.snap?.context;
        if (!c?.maxTokens) return '';
        const pct = Math.round(c.percentage ?? (c.totalTokens / c.maxTokens) * 100);
        const r = 7;
        const len = 2 * Math.PI * r;
        return `<span title="Context ${fmt.tokens(c.totalTokens)} of ${fmt.tokens(c.maxTokens)}" class="flex h-8 items-center gap-1.5 px-2 text-xs text-overlay1 tabular-nums"><svg viewBox="0 0 18 18" class="size-4 -rotate-90"><circle cx="9" cy="9" r="${r}" fill="none" stroke="var(--ctp-surface1)" stroke-width="2.5"/><circle cx="9" cy="9" r="${r}" fill="none" stroke="var(--ctp-${pct > 80 ? 'peach' : 'mauve'})" stroke-width="2.5" stroke-linecap="round" stroke-dasharray="${(len * Math.max(pct, 2)) / 100} ${len}"/></svg>${pct}%</span>`;
    };

    const renderStatus = () => {
        if (!C) return;
        const snap = C.snap;
        const state = snap?.state;
        let left = '';
        if (state === 'requires_action') {
            left = `${ui.dot('yellow', { pulse: true })}<span class="text-yellow">Waiting for approval</span><span class="shrink-0 max-sm:hidden">${ui.kbd(['Esc'])}</span>`;
        } else if (state === 'running') {
            if (!C.turn.start) C.turn.start = Date.now();
            const label = snap.status === 'compacting' ? 'Compacting' : C.activity || 'Working';
            left = `${ui.spinner('size-3.5 shrink-0 text-mauve')}<span class="min-w-0 truncate bg-linear-to-r from-subtext0 via-text to-subtext0 bg-[length:200%_100%] bg-clip-text text-transparent [animation:shimmer_2.4s_linear_infinite]">${ui.esc(label)}</span>
                <span data-turn-elapsed class="shrink-0 text-overlay1 tabular-nums"></span>
                ${C.thinking ? `<span class="flex shrink-0 items-center gap-1 text-overlay1 tabular-nums"><i data-lucide="arrow-down" class="size-3"></i>${fmt.tokens(C.thinking)}</span>` : ''}
                <span class="shrink-0 max-sm:hidden">${ui.kbd(['Esc'])}</span>`;
        } else if (state === 'starting') {
            left = `${ui.spinner('size-3.5 shrink-0 text-overlay1')}<span class="text-overlay1">Starting claude</span>`;
        }
        if (state !== 'running' && state !== 'requires_action') C.turn.start = 0;
        const tasks = (snap?.tasks || []).filter((t) => !t.ambient);
        const pills = Object.entries(TASK_KINDS).map(([kind, [icon]]) => {
            const n = tasks.filter((t) => t.task_type === kind).length;
            return n ? `<button type="button" data-action="chat-tasks" aria-expanded="false" title="${fmt.plural(n, TASK_KINDS[kind][1].toLowerCase())}" class="flex h-7 items-center gap-1.5 rounded-full bg-surface0 px-2.5 text-text transition-colors hover:bg-surface1">${kind === 'local_bash' ? ui.dot('green', { pulse: true }) : ''}<i data-lucide="${icon}" class="size-3.5 text-overlay1"></i>${n}</button>` : '';
        }).join('');
        paint('[data-chat-status]', `<span class="flex min-w-0 flex-1 items-center gap-2">${left}</span><span class="flex shrink-0 items-center gap-1.5">${pills}</span>`);
        tick();
    };

    const fiveHour = () => C.snap?.limits?.unifiedWindows?.five_hour;

    const renderBanner = () => {
        if (!C) return;
        const snap = C.snap;
        let body = '';
        if (!C.loaded) body = '';
        else if (!snap) {
            body = ui.banner({ tone: 'blue', icon: 'moon', body: 'No session is running. Continue the last conversation to chat here.', actions: ui.button('Continue', { icon: 'play', size: 'sm', variant: 'primary', action: 'chat-continue' }) });
        } else if (snap.state === 'exited') {
            const err = ansi(snap.exit?.stderr || snap.exit?.error || '').trim();
            body = ui.banner({
                tone: 'red',
                icon: 'circle-x',
                title: `claude exited with code ${snap.exit?.code ?? '?'}`,
                body: err ? `<pre data-scroll class="mt-2 max-h-32 overflow-auto rounded-2xl bg-base p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap text-subtext0">${ui.esc(err.slice(-4000))}</pre>` : 'Restart continues the same conversation.',
                actions: ui.button('Restart', { icon: 'rotate-cw', size: 'sm', action: 'project-restart' }),
            });
        } else {
            const five = fiveHour();
            if (five && five.utilization >= 0.8 && C.dismissed !== five.resetsAt) {
                body = ui.banner({ tone: 'peach', icon: 'gauge', body: `5-hour limit at <span class="font-medium text-text">${Math.round(five.utilization * 100)}%</span>${five.resetsAt ? `, resets ${fmt.time(five.resetsAt)}` : ''}`, actions: ui.iconButton('x', { title: 'Dismiss', size: 'sm', action: 'chat-banner-dismiss' }) });
            }
        }
        paint('[data-chat-banner]', body ? `<div class="mb-2">${body}</div>` : '');
    };

    const section = (title, body, aside = '') => `<section><header class="mb-2 flex h-8 items-center gap-2 px-2"><h3 class="${ui.cls.label} flex-1">${title}</h3>${aside}</header>${body}</section>`;

    const liveHtml = () => {
        const s = C?.snap;
        if (!s) return '';
        const c = s.context;
        const used = usedCategories(c?.categories);
        const context = c?.maxTokens ? section('Context', `<div class="px-2"><div class="mb-1.5 flex items-baseline justify-between text-xs"><span class="text-text tabular-nums">${fmt.tokens(c.totalTokens)} <span class="text-overlay1">/ ${fmt.tokens(c.maxTokens)}</span></span><span class="text-overlay1 tabular-nums">${Math.round(c.percentage ?? (c.totalTokens / c.maxTokens) * 100)}%</span></div>
            <span class="flex h-1.5 gap-px overflow-hidden rounded-full bg-surface0">${used.map((x) => `<span class="h-full bg-${x.tone}" style="width:${Math.max(0.5, (x.tokens / c.maxTokens) * 100)}%"></span>`).join('')}</span>
            <dl class="mt-2.5 grid grid-cols-2 gap-x-4 gap-y-1 text-xs">${used.map((x) => `<div class="flex min-w-0 items-center gap-1.5">${ui.dot(x.tone)}<dt class="truncate text-overlay1">${ui.esc(x.name)}</dt><dd class="ml-auto text-subtext0 tabular-nums">${fmt.tokens(x.tokens)}</dd></div>`).join('')}</dl></div>`) : '';
        const w = s.limits?.unifiedWindows || {};
        const limit = (label, win, when) => {
            if (!win) return '';
            const pct = Math.round((win.utilization || 0) * 100);
            return `<div class="px-2"><div class="mb-1.5 flex items-baseline justify-between text-xs"><span class="text-subtext0">${label}</span><span class="text-overlay1 tabular-nums"><span class="text-text">${pct}%</span>${win.resetsAt ? ` · ${when(win.resetsAt)}` : ''}</span></div>${ui.meter(pct, pct >= 90 ? 'red' : pct >= 80 ? 'peach' : 'mauve', 'h-1.5')}</div>`;
        };
        const limits = w.five_hour || w.seven_day ? section('Limits', `<div class="flex flex-col gap-3">${limit('5-hour', w.five_hour, fmt.time)}${limit('Weekly', w.seven_day, fmt.day)}</div>`) : '';
        const tasks = (s.tasks || []).filter((t) => !t.ambient);
        const taskRow = (t) => `<li class="flex items-center gap-3 rounded-2xl px-2 py-1.5"><i data-lucide="${TASK_KINDS[t.task_type]?.[0] || 'circle-dot'}" class="size-4 shrink-0 text-overlay1"></i><span class="min-w-0 flex-1"><span class="block truncate text-sm text-subtext1">${ui.esc(t.description || t.task_id)}</span>${t.progress?.description ? `<span class="block truncate font-mono text-[11px] text-overlay1">${ui.esc(t.progress.description)}</span>` : ''}</span>${t.startedAt ? `<span data-start="${t.startedAt}" class="text-xs text-overlay1 tabular-nums"></span>` : ''}${ui.iconButton('square', { title: 'Stop', size: 'sm', variant: 'danger', action: 'chat-task-stop', attrs: `data-task="${ui.esc(t.task_id)}"` })}</li>`;
        const taskList = tasks.length ? section('Tasks', `<ul class="flex flex-col">${tasks.map(taskRow).join('')}</ul>`, ui.count(tasks.length)) : '';
        const todos = s.todos || [];
        const todoList = todos.length ? section('To-dos', `<ul class="flex flex-col px-1">${todos.map(todoRow).join('')}</ul>`, `<span class="text-xs text-overlay1 tabular-nums">${todos.filter((t) => t.status === 'completed').length}/${todos.length}</span>`) : '';
        return `${context}${limits}${taskList}${todoList}`;
    };

    const paintLive = () => {
        if (!C) return;
        const markup = liveHtml();
        document.querySelectorAll('[data-p-live]').forEach((el) => paint(el, markup));
    };

    const slash = {
        open: false,
        active: 0,
        items: [],

        update() {
            const m = C.input.value.match(/^\/(\S*)$/);
            const commands = C.init?.commands || [];
            if (!m || !commands.length) return slash.close();
            const q = m[1];
            slash.items = (q ? commands.map((c) => ({ c, s: ui.fuzzy(q, c.name) })).filter((x) => x.s >= 0).sort((a, b) => b.s - a.s).map((x) => x.c) : commands).slice(0, 8);
            slash.active = Math.min(slash.active, Math.max(0, slash.items.length - 1));
            slash.draw();
        },

        draw() {
            const { items, active } = slash;
            const markup = items.length
                ? items.map((c, i) => `<li role="option" data-cmd="${ui.esc(c.name)}" aria-selected="${i === active}" class="flex cursor-pointer items-baseline gap-3 rounded-xl px-3 py-2 text-sm ${i === active ? 'bg-surface0 text-text' : 'text-subtext0'}"><span class="shrink-0 font-mono text-text">/${ui.esc(c.name)}</span>${c.argumentHint ? `<span class="shrink-0 font-mono text-xs text-overlay0 max-sm:hidden">${ui.esc(c.argumentHint)}</span>` : ''}<span class="min-w-0 flex-1 truncate text-xs text-overlay1">${ui.esc(c.description)}</span></li>`).join('')
                : '<li class="px-3 py-6 text-center text-sm text-overlay1">No matches</li>';
            const anchor = $('[data-chat-composer]');
            if (!slash.open) {
                const pop = ui.popover.open(anchor, '<ul role="listbox" data-slash class="flex flex-col"></ul>', { matchWidth: true, onClose: () => { slash.open = false; } });
                slash.open = true;
                slash.pop = pop;
                pop.addEventListener('mousedown', (e) => {
                    const li = e.target.closest('[data-cmd]');
                    if (!li) return;
                    e.preventDefault();
                    slash.choose(li.dataset.cmd);
                });
            }
            slash.pop.querySelector('[data-slash]').innerHTML = markup;
            const r = anchor.getBoundingClientRect();
            const top = r.top - slash.pop.offsetHeight - 6;
            slash.pop.style.top = `${top > 8 ? top : r.bottom + 6}px`;
        },

        choose(name) {
            C.input.value = `/${name} `;
            slash.close();
            C.input.focus();
            autosize();
        },

        close() {
            if (slash.open) ui.popover.close();
            slash.open = false;
            slash.active = 0;
        },

        keys(e) {
            if (!slash.open || !slash.items.length) return false;
            if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
                e.preventDefault();
                const n = slash.items.length;
                slash.active = (slash.active + (e.key === 'ArrowDown' ? 1 : -1) + n) % n;
                slash.draw();
                return true;
            }
            if (e.key === 'Enter' || e.key === 'Tab') {
                const name = slash.items[slash.active].name;
                if (e.key === 'Enter' && C.input.value.trim() === `/${name}`) {
                    slash.close();
                    return false;
                }
                e.preventDefault();
                slash.choose(name);
                return true;
            }
            return false;
        },
    };

    const promptText = (m) => {
        if (m.type !== 'user' || m.parent_tool_use_id || m.isSynthetic || m.isReplay || m.isCompactSummary) return '';
        const content = m.message?.content;
        if (Array.isArray(content) && content.some((c) => c.type === 'tool_result')) return '';
        const t = textOf(content).trim();
        if (!t || t.startsWith('[Request interrupted') || /^<(local-command-stdout|local-command-caveat|task-notification|system-reminder)>/.test(t)) return '';
        return t.startsWith('<command-name>') || t.startsWith('<command-message>') ? commandPrompt(t) : t;
    };

    const conversationOf = (messages) => {
        let start = 0;
        messages.forEach((m, i) => { if (m?.type === 'conversation_reset') start = i + 1; });
        const out = [];
        const byId = new Map();
        for (const m of messages.slice(start)) {
            if (!m) continue;
            const p = promptText(m);
            if (p) {
                out.push({ role: 'user', text: p, slash: p.startsWith('/') });
                continue;
            }
            if (m.type !== 'assistant' || m.parent_tool_use_id || m.message?.model === '<synthetic>') continue;
            const t = (m.message?.content || []).filter((b) => b.type === 'text' && b.text?.trim()).map((b) => b.text).join('\n\n');
            if (!t) continue;
            const id = m.message?.id;
            const entry = id && byId.get(id);
            if (entry) entry.text += `\n\n${t}`;
            else {
                const e = { role: 'agent', text: t };
                out.push(e);
                if (id) byId.set(id, e);
            }
        }
        return out.filter((e, i) => !e.slash || out[i + 1]?.role === 'agent');
    };

    async function copySession(name, tags) {
        let messages;
        if (C?.name === name && C.loaded) messages = [...C.history, ...C.events.filter((e) => (e.name || e.event) === 'message').map((e) => e.data)];
        else {
            try {
                messages = (await api.history(name)).messages || [];
            } catch (err) {
                return ui.toast(`Copy failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            }
        }
        const items = conversationOf(messages);
        if (!items.length) return ui.toast('The conversation is empty', { icon: 'message-square-dashed' });
        ui.copy(items.map((x) => (tags ? `[${x.role}]\n${x.text}` : x.text)).join('\n\n'));
    }

    ui.on('chat-copy-prompt', (b) => ui.copy(b.closest('li').dataset.text, b));
    ui.on('chat-copy-reply', (b) => {
        const texts = [];
        for (let n = b.closest('li').previousElementSibling; n && !n.hasAttribute('data-user') && !n.hasAttribute('data-footer'); n = n.previousElementSibling) {
            if (n.dataset.msg !== undefined && n.dataset.text) texts.unshift(n.dataset.text);
        }
        ui.copy(texts.join('\n\n'), b);
    });
    ui.on('chat-detail', (b) => {
        const d = b.parentElement.querySelector(':scope > [data-detail]');
        if (!d) return;
        const open = d.classList.toggle('hidden') === false;
        b.setAttribute('aria-expanded', String(open));
    });
    ui.on('chat-group', (b) => {
        const body = b.parentElement.querySelector('[data-group-body]');
        const open = body.classList.toggle('hidden') === false;
        b.setAttribute('aria-expanded', String(open));
    });
    ui.on('chat-image', (b) => ui.modal.open({ title: 'Image', icon: 'image', size: 'xl', body: `<img src="${ui.esc(b.dataset.src)}" alt="" class="mx-auto max-w-full rounded-2xl">` }));
    ui.on('chat-send', () => C && submit());
    ui.on('chat-interrupt', () => interrupt());
    ui.on('chat-task-stop', (b) => stopTask(b.dataset.task));
    ui.on('chat-continue', () => C && views.newSession.open({ name: C.name, continue: true }));
    ui.on('chat-banner-dismiss', () => {
        if (!C) return;
        C.dismissed = fiveHour()?.resetsAt;
        renderBanner();
    });
    ui.on('chat-tasks', (b) => {
        const tasks = (C?.snap?.tasks || []).filter((t) => !t.ambient);
        if (!tasks.length) return;
        b.setAttribute('aria-expanded', 'true');
        ui.menu(b, tasks.map((t) => ({ icon: TASK_KINDS[t.task_type]?.[0] || 'circle-dot', label: t.description || t.task_id, hint: `${TASK_KINDS[t.task_type]?.[1] || 'Task'} · Stop`, onSelect: () => stopTask(t.task_id) })));
    });

    document.addEventListener('keydown', (e) => {
        if (e.key !== 'Escape' || !C || app.current !== 'project' || ui.modal.stack.length || ui.popover.current) return;
        if (!BUSY.has(C.snap?.state)) return;
        e.preventDefault();
        interrupt();
    }, true);

    const modelLabel = (s) => {
        const model = (s.model && C?.init?.models?.find((m) => m.value === s.model)) || null;
        return `<span class="block truncate">${ui.esc(model?.displayName || s.resolvedModel || s.model || 'Default')}${s.effort ? `<span class="text-xs text-overlay1"> · ${ui.esc(EFFORTS[s.effort] || s.effort)}</span>` : ''}</span>`;
    };

    return {
        html,
        mount,
        unmount,
        paintLive,
        copySession,
        modelLabel,
        modeLabel,
        modes: MODES,
        efforts: EFFORTS,
        snapshot: (name) => (C?.name === name && C.loaded ? C.snap : null),
        saveScroll() {
            if (C) C.kept = nearBottom() ? null : C.scroller.scrollTop;
        },
        restoreScroll() {
            if (!C) return;
            C.scroller.scrollTop = C.kept ?? C.scroller.scrollHeight;
        },
        state: (name) => (C?.name === name && C.loaded ? C.snap?.state || null : live.summary[name]?.state || null),
    };
})();
