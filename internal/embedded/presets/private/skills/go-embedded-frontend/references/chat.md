# Chat

**An agent chat is a transcript column of rows drawn from the agent's event stream, a composer tile under it, and a session sidebar beside it, and every row updates in place as its events arrive.**

The message names below are those of the Claude Code stream-json protocol. The rows fit any agent stream that reports text, tool calls, sub-agents, approvals, and turn state.

## The View

The chat is the workspace from layout.md with the transcript in the middle panel. The transcript is a scroll region holding one centered list, and the composer sits in the panel's footer outside it, so a long reply never pushes the input off screen.

```javascript
pane.innerHTML = `<div data-scroll="lane" data-transcript class="min-h-0 flex-1 overflow-y-auto px-4">
        <ol data-log data-column class="mx-auto flex w-full flex-col gap-4 py-4 transition-[max-width] duration-300 ease-out ${column.cls()}"></ol>
    </div>
    <footer class="shrink-0 px-3 pb-3 sm:px-4">
        <div data-column class="mx-auto w-full transition-[max-width] duration-300 ease-out ${column.cls()}">
            <div data-banner></div>
            <div data-status class="flex h-9 items-center gap-2 px-2 text-xs"></div>
            <form data-composer novalidate>...</form>
        </div>
    </footer>`;
```

The list and the composer are reading columns from layout.md, so the toggle in the panel header confines both, and a rendered Markdown file in the same panel follows them.

The view keeps the list as `chat.log` and its scroll region as `chat.scroller`, which the row builders below share.

A new row scrolls the transcript to the bottom only when the reader is already within 120px of it. A reader who has scrolled back to an earlier reply would otherwise have the page pulled out from under them.

```javascript
append(html) {
    const s = chat.scroller;
    const stick = s.scrollHeight - s.scrollTop - s.clientHeight < 120;
    const tpl = document.createElement('template');
    tpl.innerHTML = html.trim();
    const node = chat.log.appendChild(tpl.content.firstElementChild);
    ui.icons(node);
    if (stick) requestAnimationFrame(() => s.scrollTo({ top: s.scrollHeight, behavior: 'smooth' }));
    return node;
},
```

Each stream message maps to one kind of row, and a later message updates the row an earlier one created.

| Stream message | Row |
|---|---|
| `user` text the page sent | User bubble |
| `assistant` with a `thinking` block | Thinking disclosure |
| `assistant` with a `text` block | Markdown reply, joined to the previous one by `message.id` |
| `assistant` with a `tool_use` block | Tool row, an agent row for `Agent`, or nothing for `AskUserQuestion` |
| `user` with a `tool_result` block | The matching row's state, meta, and detail |
| any message carrying `parent_tool_use_id` | A row nested under that agent |
| `system/task_progress`, `system/task_notification` | An agent's activity line, a task's state |
| `system/task_started` with `task_type: "local_bash"` | Background command card |
| `control_request` with `subtype: "can_use_tool"` | Action card, or a question card for `AskUserQuestion` |
| `assistant` carrying `local_command_run` | Command output tile, or the context card when it carries `context_usage` |
| `system/compact_boundary` | Notice |
| `rate_limit_event` | Limit meters, and the banner past 80% |
| `system/session_state_changed` | Status line and the composer's send button |
| `result` | Turn footer, or a notice for `error_during_execution` |

## Messages

A user prompt is a right-aligned `surface0` bubble with its bottom-right corner tightened toward the speaker, rendered as Markdown with `breaks` on so the user's line breaks survive. Its `.user-md` rules set the text to `text` at 14px and lift inline code and fences to `surface1`, which would otherwise vanish into the bubble. An assistant reply is a bare `.markdown-body` with no bubble, because the reply is the content the page exists for and a bubble would box every paragraph, table, and code block a second time.

A copy button beside the bubble copies the prompt's Markdown source, revealed on hover like `ui.cls.reveal`. The row is a named `group/msg`, because a code fence inside the bubble is its own `group` and an unnamed hover on the row would reveal the fence's copy button too.

```javascript
userBubble(text) {
    const li = chat.append(`<li data-user class="group/msg flex items-end justify-end gap-1">
        <span class="opacity-0 transition-opacity group-hover/msg:opacity-100 focus-within:opacity-100 max-md:opacity-100">${ui.iconButton('copy', { tip: 'Copy', size: 'sm', action: 'copy-prompt' })}</span>
        <div class="markdown-body user-md max-w-[85%] min-w-0 rounded-3xl rounded-br-lg bg-surface0 px-4 py-2.5"></div>
    </li>`);
    li.dataset.text = text;
    md.render(li.lastElementChild, text, { breaks: true });
},

text(m, block) {
    const last = chat.log.lastElementChild;
    if (last?.dataset.msg === m.message.id && last.dataset.text) {
        last.dataset.text += `\n\n${block.text}`;
        md.render(last.querySelector('article'), last.dataset.text);
        return;
    }
    const li = chat.append(`<li data-msg="${ui.esc(m.message.id)}"><article class="markdown-body"></article></li>`);
    li.dataset.text = block.text;
    md.render(li.querySelector('article'), block.text);
},
```

```css
.markdown-body.user-md { color: var(--ctp-text); font-size: 14px; line-height: 1.625; }
.user-md p, .user-md ul, .user-md ol, .user-md pre { margin-bottom: 10px; }
.user-md code, .user-md pre { background-color: var(--ctp-surface1); }
.user-md pre { padding: 0.75rem 1rem; }
```

The blocks of one reply share a `message.id`, so a text block whose id matches the last row joins that row's article. Rendering each block as its own row splits one answer into several.

Each row keeps its Markdown source in `data-text`, so a copy button copies what was written rather than the rendered text. The prompt's button copies its own row, and Copy reply joins every reply row since the last `data-user` row.

```javascript
ui.on('copy-prompt', (b) => ui.copy(b.closest('li').dataset.text, b));
ui.on('copy-reply', (b) => {
    const texts = [];
    for (let n = b.closest('li').previousElementSibling; n && !n.hasAttribute('data-user'); n = n.previousElementSibling) {
        if (n.dataset.text) texts.unshift(n.dataset.text);
    }
    ui.copy(texts.join('\n\n'), b);
});
```

A thinking block is a disclosure row reading `Thought for 2.4s`, timed from `thinking_duration_ms`. Its text is empty unless the process runs with `--thinking-display summarized`, so a row with no text is a disabled button rather than one that opens onto nothing.

## Tool Rows

A tool call is one row: an icon, the tool name, the call's main argument in mono, a meta slot, and a state slot. The argument is what the reader scans for, a command or a path, so it takes the flexible width and truncates with its full text as the tooltip.

```javascript
const TOOLS = {
    Bash: ['terminal', (i) => i.command],
    Read: ['file-text', (i) => i.file_path],
    Edit: ['file-pen-line', (i) => i.file_path],
    Write: ['file-plus', (i) => i.file_path],
    WebFetch: ['globe', (i) => i.url],
};

toolRow(block, nested) {
    const [icon, arg] = TOOLS[block.name] || ['wrench', (i) => Object.values(i)[0]];
    const value = arg(block.input) ?? '';
    return `<li data-tool="${ui.esc(block.id)}">
        <button type="button" data-action="toggle-detail" class="flex ${nested ? 'h-7' : 'h-8'} w-full items-center gap-2.5 rounded-xl px-2 text-left text-sm transition-colors hover:bg-surface0/50 ${ui.cls.focus}">
            <i data-lucide="${icon}" class="size-3.5 shrink-0 text-overlay1"></i>
            <span class="shrink-0 font-medium text-subtext1">${ui.esc(block.name)}</span>
            <span data-tip="${ui.esc(block.input.description || value)}" class="min-w-0 flex-1 truncate font-mono text-xs text-overlay1">${ui.esc(value)}</span>
            <span data-meta class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums">${block.name === 'Bash' ? `<span data-start="${Date.now()}"></span>` : ''}</span>
            <span data-state class="flex w-10 shrink-0 items-center justify-end text-xs text-overlay0 tabular-nums">${ui.spinner('size-3.5 text-overlay1')}</span>
        </button>
        <div data-detail class="hidden pt-1 pb-2 pl-8"></div>
    </li>`;
},
```

The state slot holds a spinner while the call runs, the duration or a muted check once it returns, and a red `circle-x` on an error. A running Bash row shows its elapsed time in the meta slot, because `claude` sends no progress event for a foreground command and a bare spinner gives no sense of a slow one.

A row's output sits in a `base` well under it, hidden until the row is clicked. A Bash row shows the command and its `stdout`, an Edit row shows the diff, and a row with nothing to show loses its `data-action`, so it is not a button that opens an empty well.

Consecutive tool rows share one group, and a group that closes with three or more rows collapses into a `N tool calls` disclosure. A turn of twenty reads and searches otherwise buries the reply that explains them. A group closes when any other kind of row is appended.

## Diffs

An edit renders the `structuredPatch` from its `tool_use_result` as a grid of new line numbers and lines. An added line is `bg-green/10 text-green`, a removed one is `bg-red/10 text-red` with no number, and the row's meta slot carries the `+N` and `-N` counts, so the size of a change reads without opening it.

```javascript
diff(hunks) {
    const rows = hunks.flatMap((h) => {
        let n = h.newStart;
        return h.lines.map((l) => {
            const tone = l[0] === '+' ? 'bg-green/10 text-green' : l[0] === '-' ? 'bg-red/10 text-red' : 'text-subtext0';
            return `<span class="select-none px-2 text-right text-overlay0 tabular-nums">${l[0] === '-' ? '' : n++}</span><span class="${tone} px-2 whitespace-pre">${ui.esc(l.slice(1)) || ' '}</span>`;
        });
    }).join('');
    return `<div data-scroll class="${ui.cls.well} max-h-72 overflow-auto py-2 font-mono text-xs leading-relaxed"><div class="grid min-w-max grid-cols-[auto_1fr]">${rows}</div></div>`;
},
```

A Write that creates a file has an empty patch, so its detail is the new content drawn as added lines.

## Agent Groups

The `Agent` calls of one reply share its `message.id` and render as one `surface0/50` card with a row per agent. Agents started together are one step of the work, and three separate cards read as three unrelated events.

An agent row is a small tile, the description, and an activity line under it. The activity line takes each `task_progress` description while the agent runs and the `task_notification` summary when it ends, so the row always says what the agent is doing or what it found. The agent's own tool calls are nested rows matched through `parent_tool_use_id`, hidden until the agent row is clicked, and its final text renders as Markdown in a `base` well under them.

```javascript
`<li data-agent="${ui.esc(block.id)}">
    <button type="button" data-action="toggle-detail" class="flex w-full items-center gap-3 rounded-2xl px-2 py-2 text-left transition-colors hover:bg-surface0/50 ${ui.cls.focus}">
        ${ui.tile('bot', tone, 'sm')}
        <span class="min-w-0 flex-1"><span class="flex items-center gap-2"><span class="truncate text-sm text-text">${ui.esc(block.input.description)}</span><span class="shrink-0 text-xs text-overlay0 max-sm:hidden">${ui.esc(block.input.subagent_type)}</span></span><span data-activity class="block truncate font-mono text-xs text-overlay1">Starting</span></span>
        <span data-agent-meta class="flex shrink-0 items-center gap-2 text-xs text-overlay1 tabular-nums"><span data-start="${Date.now()}"></span></span>
        <span data-state class="flex w-5 shrink-0 justify-end">${ui.spinner(`size-4 text-${tone}`)}</span>
    </button>
    <div data-detail class="hidden pb-2 sm:pl-13"><ul data-nested class="flex flex-col"></ul><div data-summary></div></div>
</li>`
```

Agent tiles rotate through label accents such as `sapphire`, `teal`, `pink`, and `lavender`. A status accent on an agent would read as success or failure before the agent has finished.

The finished row shows the tool count, the tokens, and the duration from the `Agent` tool result. Below `sm` only the duration stays, so a phone keeps the description readable.

## Action Cards

A request that waits on the user is a card tinted `yellow/10`, the pending tone, holding a tile, the question, and a `base` well with the exact command. Its buttons run Deny as ghost, Always allow as secondary, and Allow as primary, in the order of a modal footer with the primary action last.

```javascript
`<li><section class="rounded-3xl bg-yellow/10 p-4">
    <header class="mb-3 flex items-center gap-3">${ui.tile('shield-question', 'yellow', 'sm')}<h3 class="min-w-0 flex-1 text-sm font-medium text-text">Allow ${ui.esc(r.tool_name)}?</h3></header>
    <pre class="${ui.cls.well} overflow-x-auto p-3 font-mono text-xs text-text"><span class="text-mauve">$</span> ${ui.esc(r.input.command)}</pre>
    <footer data-decide class="mt-3 flex flex-wrap justify-end gap-2">
        ${ui.button('Deny', { variant: 'ghost', size: 'sm', attrs: 'data-decision="deny"' })}
        ${ui.button('Always allow', { size: 'sm', attrs: 'data-decision="always"' })}
        ${ui.button('Allow', { icon: 'check', variant: 'primary', size: 'sm', attrs: 'data-decision="allow"' })}
    </footer>
</section></li>`
```

Answering keeps the card in the transcript, drops its tint to `surface0/50`, and swaps the buttons for a chip naming the decision. The decision is part of the conversation, and a card that vanished would leave a command in the transcript with no record of who let it run.

A question from `AskUserQuestion` arrives as the same `can_use_tool` request with `requires_user_interaction: true`. Its card lays the options out as `surface0/50` tiles in a two-column grid, each a label over a description, and the chosen tile keeps a `mauve/10` fill while the others dim. The reply is an allow whose `updatedInput` adds an `answers` map from question to label.

## Background Commands

A background shell is a card with a green terminal tile, a pulsing Running status, a live elapsed time, and a stop button, over a `base` well tailing its output. When the command ends, the status turns to Done or Stopped and the stop button goes, while the tail stays readable.

## Stopping

The session's `initialize` request declares `perTaskStopAffordance: true`, and a task's stop button sends `stop_task` with the `task_id` from its `task_started`. With the declaration, Stop interrupts only the turn and background shells, agents, and workflows keep running. Without it the CLI fails closed, and every Stop kills them along with the turn.

```json
{"type":"control_request","request_id":"<id>","request":{"subtype":"initialize","perTaskStopAffordance":true}}
{"type":"control_request","request_id":"<id>","request":{"subtype":"stop_task","task_id":"<task_id>"}}
```

## Notices

A change of state that is not a message, such as a model switch, a compaction, or an interrupt, is a centered divider: a hairline, an icon with a short phrase, and a hairline. It marks a boundary in the conversation without competing with the replies around it.

```javascript
notice(icon, text, tone = 'overlay1') {
    chat.append(`<li class="flex items-center gap-3 text-xs text-${tone}"><span class="h-px flex-1 bg-surface0"></span><i data-lucide="${icon}" class="size-3.5"></i><span>${ui.esc(text)}</span><span class="h-px flex-1 bg-surface0"></span></li>`);
},
```

An interrupt takes `red` and a stopped task `peach`, and every other notice stays `overlay1`.

## Turn Footer

A finished turn ends in a row of muted counters, the duration and the tokens, with Copy reply revealed on hover. A turn ending in `error_during_execution` gets the red Interrupted notice instead.

```javascript
result(m) {
    if (m.subtype === 'error_during_execution') return chat.notice('octagon-x', 'Interrupted', 'red');
    const tokens = (m.usage?.input_tokens || 0) + (m.usage?.output_tokens || 0);
    chat.append(`<li class="group -mt-2 flex items-center gap-3 px-1 text-xs text-overlay0 tabular-nums">
        <span class="flex items-center gap-1"><i data-lucide="clock" class="size-3"></i>${fmt.dur(m.duration_ms)}</span>
        <span class="flex items-center gap-1"><i data-lucide="arrow-down" class="size-3"></i>${fmt.tokens(tokens)}</span>
        <span class="${ui.cls.reveal}">${ui.iconButton('copy', { tip: 'Copy reply', size: 'sm', action: 'copy-reply' })}</span>
    </li>`);
},
```

## Status Line

A line between the transcript and the composer shows the running turn: a mauve spinner, a label naming the current activity, the turn's elapsed time, and the `Esc` key that interrupts it. The label shimmers through a moving gradient, because a still label beside a spinner reads as stuck after a few seconds.

```javascript
`${ui.spinner('size-3.5 text-mauve')}<span class="min-w-0 truncate bg-linear-to-r from-subtext0 via-text to-subtext0 bg-[length:200%_100%] bg-clip-text text-transparent [animation:shimmer_2.4s_linear_infinite]">${ui.esc(label)}</span><span data-turn-elapsed class="shrink-0 text-overlay1 tabular-nums"></span><span class="shrink-0 max-sm:hidden">${ui.kbd(['Esc'])}</span>`
```

While the turn waits on the user, the spinner gives way to a pulsing yellow dot and `Waiting for approval`. Running agents and background shells sit at the line's right end as `surface0` count pills that open a menu of them, so work still running after the turn ends stays in view.

## Composer

The composer is a `rounded-3xl` `surface0/50` tile with a `focus-within` ring, holding a bare textarea and a toolbar. The controls in the toolbar keep their own `surface0` fills, which a `surface0` composer would swallow.

```javascript
`<form data-composer novalidate class="rounded-3xl bg-surface0/50 p-2 transition-shadow focus-within:ring-2 focus-within:ring-mauve/40">
    <textarea data-input rows="1" spellcheck="false" placeholder="Message, or / for commands" class="block max-h-48 w-full resize-none bg-transparent px-3 py-2 text-sm leading-relaxed text-text placeholder:text-overlay0 outline-none pointer-coarse:text-[16px]"></textarea>
    <div class="flex items-center gap-1.5 pt-1">
        ${ui.iconButton('paperclip', { tip: 'Attach', size: 'sm' })}
        ${ui.select({ name: 'model', value: 'opus', pill: true, icon: 'sparkles', options: models })}
        <span class="max-sm:hidden">${ui.select({ name: 'effort', value: 'high', pill: true, icon: 'gauge', options: efforts })}</span>
        <span class="max-md:hidden">${ui.select({ name: 'mode', value: 'acceptEdits', pill: true, options: modes })}</span>
        <span data-ring class="ml-auto"></span>
        <span data-send></span>
    </div>
</form>`
```

The textarea grows with its content up to `max-h-48` and then scrolls. Enter sends and Shift+Enter breaks the line.

Send is the primary icon button, and while a turn runs it becomes a danger stop button in the same place. The one control that matters at each moment then sits under the same thumb.

The context ring is a 16px SVG arc of the used share of the window beside the percentage, mauve until 80% and peach after, with the token counts as its tooltip.

```javascript
const r = 7;
const len = 2 * Math.PI * r;
ring.innerHTML = `<span data-tip="Context ${fmt.tokens(total)} of ${fmt.tokens(max)}" class="flex h-8 items-center gap-1.5 px-2 text-xs text-overlay1 tabular-nums"><svg viewBox="0 0 18 18" class="size-4 -rotate-90"><circle cx="9" cy="9" r="${r}" fill="none" stroke="var(--ctp-surface1)" stroke-width="2.5"/><circle cx="9" cy="9" r="${r}" fill="none" stroke="var(--ctp-${pct > 80 ? 'peach' : 'mauve'})" stroke-width="2.5" stroke-linecap="round" stroke-dasharray="${(len * Math.max(pct, 2)) / 100} ${len}"/></svg>${pct}%</span>`;
```

## Slash Commands

A leading `/` opens the command list in a popover anchored to the composer and matched to its width. The list ranks commands with `ui.fuzzy` as the user types, the arrow keys move the highlight, and Enter or Tab completes the command into the textarea instead of sending it, since most commands take an argument.

```javascript
update() {
    const m = chat.input.value.match(/^\/(\S*)$/);
    if (!m) return slash.close();
    slash.items = (m[1] ? commands.map((c) => ({ c, s: ui.fuzzy(m[1], c.name) })).filter((x) => x.s >= 0).sort((a, b) => b.s - a.s).map((x) => x.c) : commands).slice(0, 8);
    slash.draw();
},
```

Each option shows the command in mono, its argument hint in `overlay0` from `sm` up, and its description truncated in `overlay1`. The list comes from the `commands` array in the `initialize` reply.

## Session Sidebar

The right sidebar from layout.md holds the session's live facts: a `dl` of status, model and effort, branch, and changes, then the pull request, and sections for Context, Limits, Tasks, and To-dos under `ui.cls.label` headings.

Context is a stacked bar of the used categories from `get_context_usage`, each in its own label accent, over a legend of dots, names, and counts. The split is what the user acts on, compacting when Messages dominate and pruning when Memory files do, so one fill would hide the reason to act.

```javascript
const TONES = { system_prompt: 'blue', system_tools: 'sapphire', mcp_tools: 'teal', memory_files: 'peach', skills: 'yellow', messages: 'mauve', compact_buffer: 'overlay0' };
const used = categories.filter((x) => x.kind === 'used' || x.kind === 'buffer');
const bar = `<span class="flex h-1.5 gap-px overflow-hidden rounded-full bg-surface0">${used.map((x) => `<span class="h-full bg-${TONES[x.id] || 'overlay0'}" style="width:${Math.max(0.5, (x.tokens / max) * 100)}%"></span>`).join('')}</span>`;
```

Each limit is a `ui.meter` in `mauve` below 80%, `peach` from 80%, and `red` from 90%, with the reset time beside the percentage. A 5-hour window reaching 80% also raises a dismissible peach banner above the composer, since the sidebar may be collapsed when it happens.

The `rate_limit_event` reports `utilization` as a fraction with `resetsAt` in epoch seconds, while `get_usage` reports a percent with an ISO 8601 time. A meter fed from both normalizes to a percent first.

## Live Counters

Durations and token counts go through one formatter, so every counter, meter, and footer on the page reads in the same units.

```javascript
const fmt = {
    tokens: (n) => (n < 1000 ? String(n) : n < 1e6 ? `${(n / 1000).toFixed(n < 1e4 ? 1 : 0).replace(/\.0$/, '')}k` : `${(n / 1e6).toFixed(1).replace(/\.0$/, '')}M`),
    dur(ms) {
        const s = Math.max(0, Math.round(ms / 1000));
        return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`;
    },
};
```

Every running duration on the page is an element carrying `data-start`, and one 250ms ticker rewrites them all. A timer per row outlives the row it was started for, and one ticker keeps every counter in step.

```javascript
setInterval(() => {
    const now = Date.now();
    document.querySelectorAll('[data-start]').forEach((n) => { n.textContent = fmt.dur(now - Number(n.dataset.start)); });
}, 250);
```

A row that finishes replaces its `data-start` element with the final duration, which takes it off the ticker.
