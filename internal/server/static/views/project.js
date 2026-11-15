(() => {
    let S = null;
    let markedReady = false;

    const tierOf = () => (ui.media.mobile() ? 'mobile' : ui.media.wide() ? 'wide' : 'laptop');
    const project = () => store.state?.projects.find((p) => p.name === S.name);
    const accountOf = (dir) => store.state?.accounts.find((a) => a.dir === dir);
    const dirname = (path) => path.split('/').slice(0, -1).join('/');
    const folderLabel = (folder) => (folder ? `${folder}/` : 'project root');
    const $ = (sel) => S.el.querySelector(sel);

    const fmtSize = (n) => {
        if (n < 1024) return `${n} B`;
        if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
        return `${(n / 1024 / 1024).toFixed(1)} MB`;
    };

    const fileIcon = (node) => {
        const ext = node.name.split('.').pop().toLowerCase();
        if (node.type === 'symlink') return ['link-2', 'text-teal'];
        if (node.kind === 'image' || ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext)) return ['file-image', 'text-pink'];
        if (node.kind === 'binary') return ['file-box', 'text-overlay1'];
        if (node.kind === 'markdown') return ['file-text', 'text-sky'];
        if (['json', 'yaml', 'yml', 'toml'].includes(ext)) return ['file-braces', 'text-yellow'];
        if (node.kind === 'code') return ['file-code', 'text-peach'];
        return ['file', 'text-overlay1'];
    };

    const slug = (text) => String(text).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)+/g, '');

    const initMarked = () => {
        if (markedReady) return;
        markedReady = true;
        marked.use({
            renderer: {
                code(token) {
                    const validLang = hljs.getLanguage(token.lang) ? token.lang : 'plaintext';
                    let highlighted = ui.esc(token.text);
                    try {
                        highlighted = hljs.highlight(token.text, { language: validLang }).value;
                    } catch {}
                    return `<pre><code class="hljs language-${validLang}">${highlighted}</code></pre>`;
                },
                heading(token) {
                    const text = this.parser.parseInline(token.tokens);
                    return `<h${token.depth} id="${slug(text.replace(/<[^>]*>/g, ''))}">${text}</h${token.depth}>`;
                },
                image(token) {
                    return `<img src="${token.href}" alt="${token.text || ''}" style="max-width:100%; border-radius:0.75rem;">`;
                },
                blockquote(token) {
                    const body = this.parser.parse(token.tokens);
                    const match = token.text.match(/^\[!(TIP|NOTE|INFO|WARNING|DANGER)\]/i);
                    if (!match) return `<blockquote>${body}</blockquote>`;
                    const type = match[1].toLowerCase();
                    const iconMap = { tip: 'lightbulb', info: 'info', danger: 'triangle-alert', warning: 'triangle-alert', note: 'sticky-note' };
                    const cleanBody = body.replace(/<p>\s*\[!(TIP|NOTE|INFO|WARNING|DANGER)\]\s*/i, '<p>');
                    return `<div class="callout ${type}"><div class="callout-icon"><i data-lucide="${iconMap[type] || 'info'}"></i></div><div class="callout-content">${cleanBody}</div></div>`;
                },
            },
        });
    };

    const addCopyButtons = (container) => {
        container.querySelectorAll('.markdown-body pre').forEach((block) => {
            if (block.querySelector('.copy-code-btn')) return;
            const codeEl = block.querySelector('code');
            if (!codeEl) return;
            const button = document.createElement('button');
            button.className = 'copy-code-btn';
            button.type = 'button';
            button.innerHTML = '<i data-lucide="copy" class="w-4 h-4"></i>';
            button.onclick = async (e) => {
                e.preventDefault();
                e.stopPropagation();
                await ui.copy(codeEl.textContent);
                button.innerHTML = '<i data-lucide="check" class="w-4 h-4"></i>';
                button.classList.add('copied');
                ui.icons(button);
                setTimeout(() => {
                    button.innerHTML = '<i data-lucide="copy" class="w-4 h-4"></i>';
                    button.classList.remove('copied');
                    ui.icons(button);
                }, 2000);
            };
            block.appendChild(button);
        });
    };

    const indexTree = (node, map = new Map()) => {
        map.set(node.path, node);
        node.children?.forEach((c) => indexTree(c, map));
        return map;
    };

    const defaultPath = () => {
        for (const p of ['README.md', 'AGENTS.md']) if (S.index.get(p)?.type === 'file') return p;
        const first = (n) => {
            if (n.type !== 'dir') return n.path;
            for (const c of n.children) {
                const f = first(c);
                if (f) return f;
            }
            return null;
        };
        return S.tree ? first(S.tree) : null;
    };

    const expandTo = (path) => {
        const parts = path.split('/');
        const before = S.expanded.size;
        for (let i = 1; i < parts.length; i++) S.expanded.add(parts.slice(0, i).join('/'));
        return S.expanded.size !== before;
    };

    const busyButton = (key, label, opts) => ui.button(label, {
        ...opts,
        icon: S.busy === key ? 'loader-circle' : opts.icon,
        spin: S.busy === key,
        disabled: !!S.busy,
        collapse: true,
    });

    const renderHeader = () => {
        const p = project();
        const header = $('[data-p-header]');
        if (!p || !header) return;
        const mobile = S.tier === 'mobile';
        const treeIcon = mobile ? 'folder-tree' : S.panels.tree?.collapsed ? 'panel-left-open' : 'panel-left-close';
        const detailsIcon = mobile ? 'info' : S.panels.details?.collapsed ? 'panel-right-open' : 'panel-right-close';
        const primary = p.active
            ? [
                busyButton('restart', 'Restart', { icon: 'rotate-cw', action: 'project-restart' }),
                busyButton('stop', 'Stop', { icon: 'square', variant: 'danger', action: 'project-stop' }),
            ]
            : [
                busyButton('continue', 'Continue', { icon: 'play', variant: 'primary', action: 'project-continue' }),
                busyButton('new', 'New', { icon: 'plus', action: 'project-new' }),
            ];
        header.innerHTML = `
            ${ui.iconButton('arrow-left', { title: 'Back to dashboard', action: 'project-back' })}
            ${ui.iconButton(treeIcon, { title: mobile ? 'Files' : 'Toggle file tree', action: 'project-toggle-tree' })}
            <div class="flex min-w-0 flex-1 items-center gap-2.5 px-1">
                <h1 class="truncate font-display text-xl text-text md:text-2xl">${ui.esc(p.name)}</h1>
                <span class="hidden shrink-0 sm:inline-flex">${ui.status(p.active)}</span>
                <span title="${p.active ? 'Active' : 'Inactive'}" class="size-2 shrink-0 rounded-full sm:hidden ${p.active ? 'bg-green' : 'bg-overlay0'}"></span>
            </div>
            ${primary.join('')}
            ${ui.iconButton(detailsIcon, { title: 'Session details', action: 'project-toggle-details' })}`;
        ui.icons(header);
    };

    const detailsHtml = (p) => {
        const acc = accountOf(p.account);
        const multi = store.state.accounts.length > 1;
        const busy = !!S.busy;
        const row = (label, value) => `<dt class="text-overlay1">${label}</dt><dd class="min-w-0 text-subtext1">${value}</dd>`;
        const sectionHead = (label, action = '') => `<header class="mb-2 flex h-8 items-center gap-2 px-2"><h3 class="${ui.cls.label} flex-1">${label}</h3>${action}</header>`;
        const chips = (list, icon, none) => `<p class="flex flex-wrap gap-1.5 px-2">${list.length ? list.map((n) => ui.chip(n, { icon })).join('') : `<span class="text-sm text-overlay1">${none}</span>`}</p>`;
        const actionRow = (icon, label, hint, action, { danger = false, disabled = false } = {}) => `
            <li><button type="button" data-action="${action}" ${disabled || busy ? 'disabled' : ''} class="flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors disabled:opacity-40 ${danger ? 'text-red hover:bg-red/10' : 'text-subtext1 hover:bg-surface0'}">
                <i data-lucide="${icon}" class="size-4 shrink-0"></i>
                <span class="min-w-0 flex-1"><span class="block text-sm">${label}</span><span class="block truncate text-xs ${danger ? 'text-red/70' : 'text-overlay1'}">${hint}</span></span>
            </button></li>`;
        return `
            <section>
                ${sectionHead('Session')}
                <dl class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-5 gap-y-3 px-2 text-sm">
                    ${row('Status', ui.status(p.active))}
                    ${row('Started', p.active && p.startedAt ? ui.timeAgo(p.startedAt) : 'Not running')}
                    ${row('Last active', ui.timeAgo(p.lastActive))}
                    ${row('Remote Control', `<span class="flex items-center gap-1.5"><i data-lucide="radio-tower" class="size-3.5 shrink-0 text-overlay1"></i><span class="min-w-0 wrap-anywhere">${ui.esc(p.rcName)}</span></span>`)}
                    ${row('Account', acc ? `<span class="block truncate">${ui.esc(acc.label)}</span><span class="block truncate font-mono text-xs text-overlay1">${ui.esc(acc.dir)}</span>` : ui.esc(p.account))}
                    ${row('Path', `<span class="block break-all font-mono text-xs">${ui.esc(p.dir)}</span>`)}
                    ${row('Connectors', p.active ? ui.chip(p.connectors ? 'On' : 'Off', { tone: p.connectors ? 'green' : 'neutral', icon: p.connectors ? 'cable' : 'unplug' }) : '<span class="text-overlay1">n/a</span>')}
                </dl>
            </section>
            ${p.stash ? `
            <section class="flex items-center gap-3 rounded-2xl bg-peach/10 p-3">
                <i data-lucide="archive-restore" class="size-5 shrink-0 text-peach"></i>
                <p class="min-w-0 flex-1 text-sm text-subtext1">A resume file is stashed from the last clean.</p>
                ${ui.button('Restore', { size: 'sm', action: 'project-restore', disabled: busy })}
            </section>` : ''}
            <section>
                ${sectionHead('MCP servers', ui.button('Edit', { icon: 'square-pen', size: 'sm', variant: 'ghost', action: 'project-mcp-edit', disabled: busy }))}
                ${chips(p.mcp, 'plug', 'No project MCP servers.')}
            </section>
            <section>
                ${sectionHead('Presets', ui.button('Re-apply', { icon: 'refresh-cw', size: 'sm', variant: 'ghost', action: 'project-presets', disabled: busy }))}
                ${chips(p.presets, 'layers', 'None applied.')}
            </section>
            <section>
                ${sectionHead('Actions')}
                <ul class="flex flex-col gap-0.5">
                    ${multi ? actionRow('arrow-left-right', 'Switch account', p.active ? 'Stop, move conversations, continue' : 'Move conversations to another account', 'project-switch') : ''}
                    ${actionRow('brush-cleaning', 'Run clean-cwd', 'Remove claudex files, stash the resume file', 'project-clean', { danger: true })}
                    ${actionRow('folder-x', 'Untrack', p.active ? 'Stop the session first' : 'Remove from the dashboard', 'project-untrack', { danger: true, disabled: p.active })}
                </ul>
            </section>`;
    };

    const renderDetails = () => {
        const p = project();
        if (!p) return;
        const body = $('[data-p-details-body]');
        if (body) {
            body.innerHTML = detailsHtml(p);
            ui.icons(body);
        }
        if (S.sheet?.el.isConnected) {
            const sheetBody = S.sheet.el.querySelector('[data-p-sheet]');
            sheetBody.innerHTML = detailsHtml(p);
            ui.icons(sheetBody);
        }
    };

    const treeRows = (node, depth) => node.children.map((c) => {
        const pad = `style="padding-left:${depth * 14 + 8}px"`;
        const selected = c.path === S.selected ? 'data-selected' : '';
        const base = 'flex h-8 w-full items-center gap-2 rounded-xl pr-2 text-left text-sm text-subtext0 transition-colors hover:bg-surface0/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-mauve/40 data-selected:bg-surface0 data-selected:text-text data-drop:bg-mauve/15 data-drop:text-mauve';
        if (c.type === 'dir') {
            const open = S.expanded.has(c.path);
            const dim = c.name.startsWith('.') ? 'opacity-60' : '';
            return `<li role="treeitem" aria-expanded="${open}">
                <button type="button" data-action="project-tree" data-path="${ui.esc(c.path)}" data-tree-dir="${ui.esc(c.path)}" tabindex="-1" class="${base} ${dim}" ${pad}>
                    <i data-lucide="chevron-right" data-chevron class="size-3.5 shrink-0 text-overlay0 transition-transform duration-150 ${open ? 'rotate-90' : ''}"></i>
                    <i data-lucide="folder" data-closed class="size-4 shrink-0 text-overlay1 ${open ? 'hidden' : ''}"></i>
                    <i data-lucide="folder-open" data-open class="size-4 shrink-0 text-overlay1 ${open ? '' : 'hidden'}"></i>
                    <span class="truncate">${ui.esc(c.name)}</span>
                </button>
                <ul role="group" ${open ? '' : 'hidden'}>${treeRows(c, depth + 1)}</ul>
            </li>`;
        }
        const [icon, tone] = fileIcon(c);
        return `<li role="treeitem">
            <button type="button" data-action="project-tree" data-path="${ui.esc(c.path)}" tabindex="${selected ? 0 : -1}" ${selected} class="${base}" ${pad}>
                <span class="size-3.5 shrink-0"></span>
                <i data-lucide="${icon}" class="size-4 shrink-0 ${tone}"></i>
                <span class="truncate">${ui.esc(c.name)}</span>
                ${c.type === 'symlink' ? `<span class="flex min-w-0 shrink items-center gap-1 text-xs text-overlay0"><i data-lucide="arrow-right" class="size-3 shrink-0"></i><span class="truncate font-mono">${ui.esc(c.target)}</span></span>` : ''}
            </button>
        </li>`;
    }).join('');

    const renderTree = () => {
        const ul = $('[data-p-tree-list]');
        if (!ul || !S.tree) return;
        const top = ul.scrollTop;
        ul.querySelectorAll(':scope > li').forEach((li) => li.remove());
        ul.insertAdjacentHTML('afterbegin', S.tree.children.length ? treeRows(S.tree, 0) : `<li class="px-3 py-6 text-center text-sm text-overlay1">No files yet</li>`);
        ui.icons(ul);
        ul.scrollTop = top;
        if (!ul.querySelector('[tabindex="0"]')) ul.querySelector('button')?.setAttribute('tabindex', '0');
    };

    const setExpanded = (path, open) => {
        if (open) S.expanded.add(path);
        else S.expanded.delete(path);
        const row = $(`[data-tree-dir="${CSS.escape(path)}"]`);
        if (!row) return;
        row.parentElement.setAttribute('aria-expanded', String(open));
        row.nextElementSibling.hidden = !open;
        row.querySelector('[data-chevron]').classList.toggle('rotate-90', open);
        row.querySelector('[data-closed]').classList.toggle('hidden', open);
        row.querySelector('[data-open]').classList.toggle('hidden', !open);
    };

    const markSelected = () => {
        S.el.querySelectorAll('[data-p-tree-list] [data-selected]').forEach((b) => {
            b.removeAttribute('data-selected');
            b.tabIndex = -1;
        });
        const row = $(`[data-p-tree-list] [data-path="${CSS.escape(S.selected || '')}"]`);
        if (!row) return;
        row.setAttribute('data-selected', '');
        row.tabIndex = 0;
        row.scrollIntoView({ block: 'nearest' });
    };

    const revealDir = (path) => {
        expandTo(`${path}/x`);
        S.folder = path;
        renderTree();
        if (S.tier === 'mobile') setDrawer(true);
        else if (S.panels.tree?.collapsed) {
            S.panels.tree.toggle(false);
            renderHeader();
        }
        const row = $(`[data-tree-dir="${CSS.escape(path)}"]`);
        row?.scrollIntoView({ block: 'nearest' });
        row?.focus();
    };

    const openFile = async (path) => {
        if (!path) {
            S.selected = null;
            S.file = null;
            renderViewer(true);
            return;
        }
        S.selected = path;
        S.folder = dirname(path);
        if (expandTo(path) || !$(`[data-p-tree-list] [data-path="${CSS.escape(path)}"]`)) renderTree();
        markSelected();
        if (S.tier === 'mobile') setDrawer(false);
        S.loading = true;
        renderViewerHeader();
        const cur = S;
        try {
            const f = await api.file(S.name, path);
            if (S !== cur || S.selected !== path) return;
            S.file = f;
        } catch (err) {
            if (S !== cur || S.selected !== path) return;
            S.file = { path, error: err.message };
        }
        S.loading = false;
        renderViewer(true);
    };

    const crumbs = (path) => {
        const parts = path ? path.split('/') : [];
        const pill = (label, target, last, icon) => `<button type="button" data-action="project-crumb" data-path="${ui.esc(target)}" ${last ? 'aria-current="page"' : ''} class="flex min-w-0 items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${last ? 'bg-surface0 text-text' : 'bg-surface0/40 text-overlay1 hover:bg-surface0 hover:text-text max-sm:hidden'}">${icon ? `<i data-lucide="${icon}" class="size-3.5 shrink-0"></i>` : ''}<span class="truncate">${ui.esc(label)}</span></button>`;
        const sep = '<i data-lucide="chevron-right" class="size-3 shrink-0 text-overlay0 max-sm:hidden"></i>';
        return [pill(S.name, '', !parts.length, 'folder-git-2'), ...parts.map((part, i) => pill(part, parts.slice(0, i + 1).join('/'), i === parts.length - 1))].join(sep);
    };

    const fileMeta = (f) => {
        if (!f || f.error) return '';
        if (f.kind === 'symlink') return 'symlink';
        if (f.kind === 'binary' || f.kind === 'image') return fmtSize(f.size || 0);
        const lines = f.content.replace(/\n$/, '').split('\n').length;
        return [f.kind === 'markdown' ? 'markdown' : f.lang, `${lines} line${lines === 1 ? '' : 's'}`, fmtSize(new Blob([f.content]).size)].filter(Boolean).join(' · ');
    };

    const renderViewerHeader = () => {
        const header = $('[data-p-viewer-header]');
        if (!header) return;
        const f = S.file;
        const textual = f && !f.error && (f.kind === 'markdown' || f.kind === 'code');
        header.innerHTML = `
            <nav aria-label="Path" class="flex min-w-0 flex-1 flex-wrap items-center gap-1">${crumbs(S.selected || '')}</nav>
            <p class="flex items-center gap-2 text-xs text-overlay1">${S.loading ? '<i data-lucide="loader-circle" class="size-3.5 animate-spin"></i>' : ''}<span class="hidden sm:inline">${ui.esc(fileMeta(f))}</span></p>
            ${f?.kind === 'markdown' && !f.error ? ui.segmented({ name: 'project-mode', value: S.mode, options: [{ value: 'rendered', label: 'Rendered', icon: 'eye', compact: true }, { value: 'source', label: 'Source', icon: 'code', compact: true }] }) : ''}
            ${textual ? ui.iconButton('copy', { title: 'Copy file contents', action: 'project-copy-file', size: 'sm' }) : ''}`;
        ui.icons(header);
    };

    const codeWell = (content, lang) => {
        const text = content.replace(/\n$/, '');
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        const html = hljs.highlight(text, { language }).value;
        const numbers = text.split('\n').map((_, i) => i + 1).join('\n');
        return `<pre class="m-3 flex overflow-x-auto rounded-2xl bg-base py-4 font-mono text-[13px] leading-6 md:m-4 3xl:text-sm 3xl:leading-7"><code aria-hidden="true" class="sticky left-0 shrink-0 select-none bg-base pl-4 pr-5 text-right text-overlay0">${numbers}</code><code class="hljs language-${language} bg-transparent! p-0! pr-6! text-subtext1">${html}</code></pre>`;
    };

    const notice = ({ icon, tone, title, detail, action = '' }) => `
        <div class="flex flex-1 items-center justify-center p-6">
            <section class="flex w-full max-w-md flex-col items-center gap-3 rounded-3xl bg-base px-6 py-10 text-center">
                <span class="flex size-14 items-center justify-center rounded-full bg-surface0 ${tone}"><i data-lucide="${icon}" class="size-6"></i></span>
                <h2 class="font-display text-lg text-text">${title}</h2>
                ${detail}
                ${action}
            </section>
        </div>`;

    const renderViewerBody = (resetScroll) => {
        const scroller = $('[data-p-viewer-body]');
        const body = $('[data-p-content]');
        if (!body) return;
        const top = scroller.scrollTop;
        const f = S.file;
        S.headings = [];
        if (!S.selected || !f) {
            body.innerHTML = ui.empty({ icon: 'file-search', title: S.tree ? 'Nothing selected' : 'Loading files', hint: S.tree ? 'Pick a file from the tree.' : '' });
        } else if (f.error) {
            body.innerHTML = notice({ icon: 'file-x-2', tone: 'text-red', title: 'Could not open this file', detail: `<p class="text-sm text-overlay1">${ui.esc(f.error)}</p>`, action: ui.button('Try again', { icon: 'rotate-cw', size: 'sm', action: 'project-reload-file' }) });
        } else if (f.kind === 'symlink') {
            const resolved = resolvePath(dirname(f.path), f.target);
            const node = S.index.get(resolved);
            const label = node?.type === 'dir' ? `Reveal ${resolved}/` : `Open ${resolved}`;
            body.innerHTML = notice({
                icon: 'link-2',
                tone: 'text-teal',
                title: `${ui.esc(f.path.split('/').pop())} is a symlink`,
                detail: `<p class="flex max-w-full items-center gap-2 rounded-full bg-surface0 px-4 py-1.5 font-mono text-sm text-subtext1"><i data-lucide="arrow-right" class="size-4 shrink-0 text-overlay1"></i><span class="truncate">${ui.esc(f.target)}</span></p>`,
                action: node ? ui.button(label, { icon: node.type === 'dir' ? 'folder-open' : 'file-text', variant: 'primary', action: 'project-open-target', attrs: `data-path="${ui.esc(resolved)}"` }) : '<p class="text-sm text-overlay1">The target is outside the project or missing.</p>',
            });
        } else if (f.kind === 'image') {
            body.innerHTML = `
                <figure class="flex flex-1 flex-col items-center justify-center gap-3 p-4 md:p-8">
                    <img src="${ui.esc(f.url)}" alt="${ui.esc(f.path)}" class="max-h-[calc(100dvh-20rem)] max-w-full rounded-2xl bg-base object-contain p-4">
                    <figcaption data-p-img-meta class="text-xs text-overlay1">${fmtSize(f.size || 0)}</figcaption>
                </figure>`;
            const img = body.querySelector('img');
            img.addEventListener('load', () => {
                const meta = body.querySelector('[data-p-img-meta]');
                if (meta && img.naturalWidth) meta.textContent = `${img.naturalWidth} × ${img.naturalHeight} · ${fmtSize(f.size || 0)}`;
            });
        } else if (f.kind === 'binary') {
            const node = S.index.get(f.path);
            const [icon] = node ? fileIcon(node) : ['file-box'];
            body.innerHTML = notice({
                icon,
                tone: 'text-overlay1',
                title: 'No preview',
                detail: `<p class="text-sm text-overlay1"><span class="font-mono text-subtext1">${ui.esc(f.path.split('/').pop())}</span> · ${fmtSize(f.size || 0)}</p>`,
            });
        } else if (f.kind === 'markdown' && S.mode === 'rendered') {
            initMarked();
            body.innerHTML = `
                <div class="flex justify-center gap-12 px-5 py-6 md:px-10 md:py-8 3xl:gap-16 3xl:px-16 3xl:py-12">
                    <article data-p-markdown class="markdown-body min-w-0 flex-1 3xl:text-[17px]!">${DOMPurify.sanitize(marked.parse(f.content))}</article>
                    <nav data-p-outline aria-label="Outline" class="sticky top-0 hidden w-56 shrink-0 self-start 3xl:w-72"></nav>
                </div>`;
            const article = body.querySelector('[data-p-markdown]');
            article.querySelectorAll('img[src]').forEach((img) => {
                const src = img.getAttribute('src');
                if (/^([a-z]+:|\/)/i.test(src)) return;
                img.src = api.rawUrl(S.name, resolvePath(dirname(f.path), src));
            });
            addCopyButtons(body);
            article.querySelectorAll('a[href^="http"]').forEach((a) => {
                a.target = '_blank';
                a.rel = 'noopener';
            });
            S.headings = [...article.querySelectorAll('h1, h2, h3')];
            renderOutline();
        } else {
            body.innerHTML = codeWell(f.content, f.kind === 'markdown' ? 'markdown' : f.lang);
        }
        ui.icons(body);
        scroller.scrollTop = resetScroll ? 0 : top;
        spy();
    };

    const renderOutline = () => {
        const nav = $('[data-p-outline]');
        if (!nav) return;
        const show = S.headings.length > 2 && $('[data-p-viewer-body]').clientWidth >= 1080;
        nav.classList.toggle('hidden', !show);
        nav.classList.toggle('block', show);
        if (!show) return;
        const minDepth = Math.min(...S.headings.map((h) => Number(h.tagName[1])));
        nav.innerHTML = `
            <h3 class="${ui.cls.label} mb-3 flex items-center gap-2 px-3"><i data-lucide="list-tree" class="size-3.5"></i>On this page</h3>
            <ul class="flex flex-col gap-0.5">${S.headings.map((h, i) => `
                <li><button type="button" data-action="project-outline" data-index="${i}" style="padding-left:${(Number(h.tagName[1]) - minDepth) * 12 + 12}px" class="block w-full truncate rounded-full py-1.5 pr-3 text-left text-sm text-overlay1 transition-colors hover:text-text data-active:bg-surface0 data-active:text-text">${ui.esc(h.textContent)}</button></li>`).join('')}
            </ul>`;
        ui.icons(nav);
    };

    const spy = () => {
        const body = $('[data-p-viewer-body]');
        const nav = $('[data-p-outline]');
        if (!body || !nav || !S.headings.length) return;
        const offset = body.getBoundingClientRect().top + 96;
        let active = 0;
        S.headings.forEach((h, i) => {
            if (h.getBoundingClientRect().top <= offset) active = i;
        });
        if (body.scrollHeight > body.clientHeight + 4 && body.scrollTop + body.clientHeight >= body.scrollHeight - 4) active = S.headings.length - 1;
        nav.querySelectorAll('[data-index]').forEach((b) => b.toggleAttribute('data-active', Number(b.dataset.index) === active));
    };

    const renderViewer = (resetScroll) => {
        renderViewerHeader();
        renderViewerBody(resetScroll);
    };

    const resolvePath = (base, rel) => {
        const out = base ? base.split('/') : [];
        for (const part of rel.split('/')) {
            if (part === '..') out.pop();
            else if (part && part !== '.') out.push(part);
        }
        return out.join('/');
    };

    const loadTree = async () => {
        try {
            const cur = S;
            const tree = await api.tree(S.name);
            if (S !== cur) return;
            S.tree = tree;
            S.index = indexTree(tree);
        } catch (err) {
            ui.toast(`Loading files failed: ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }
    };

    const setDrawer = (open) => {
        if (S.tier !== 'mobile') return;
        S.drawer = open;
        const aside = $('[data-p-tree]');
        const scrim = $('[data-p-scrim]');
        if (!aside) return;
        aside.style.transform = open ? '' : 'translateX(calc(-100% - 1.5rem))';
        scrim.hidden = !open;
    };

    const panelBtn = (icon, title, action, attrs = '') => ui.iconButton(icon, { title, action, size: 'sm', attrs });

    const layout = () => {
        const mobile = S.tier === 'mobile';
        const treeCls = mobile
            ? 'fixed inset-y-3 left-3 z-40 flex w-[min(21rem,calc(100vw-3rem))] flex-col rounded-3xl bg-mantle shadow-2xl shadow-crust transition-transform duration-200'
            : 'mr-3 flex flex-col rounded-3xl bg-mantle 3xl:mr-4';
        return `
            <section data-project class="relative flex h-full flex-col gap-3 3xl:gap-4">
                <header data-p-header class="flex shrink-0 items-center gap-1 md:gap-2"></header>
                <div class="flex min-h-0 flex-1">
                    ${mobile ? '<button type="button" data-p-scrim data-action="project-drawer-close" aria-label="Close files" hidden class="fixed inset-0 z-30 bg-crust/70 backdrop-blur-sm [animation:fade-in_.15s_ease-out]"></button>' : ''}
                    <aside data-p-tree aria-label="Files" class="${treeCls}" ${mobile ? 'style="transform:translateX(calc(-100% - 1.5rem))"' : ''}>
                        <header class="flex shrink-0 items-center gap-0.5 py-3 pl-5 pr-2 md:pt-4">
                            <h2 class="flex-1 truncate font-display text-sm font-medium text-text">Files</h2>
                            ${panelBtn('file-plus', 'New file', 'project-new-file')}
                            ${panelBtn('upload', 'Upload files', 'project-upload')}
                            ${panelBtn('chevrons-down-up', 'Collapse all', 'project-collapse')}
                            ${mobile ? panelBtn('x', 'Close', 'project-drawer-close') : ''}
                        </header>
                        <ul role="tree" data-p-tree-list data-scroll class="min-h-0 flex-1 overflow-y-auto px-2 pb-3"></ul>
                    </aside>
                    <article data-p-viewer class="flex min-w-0 flex-1 flex-col rounded-3xl bg-mantle">
                        <header data-p-viewer-header class="flex min-h-14 shrink-0 flex-wrap items-center gap-2 px-3 pt-3 pb-1 md:min-h-16 md:px-4 md:pt-4 3xl:px-6"></header>
                        <section data-p-viewer-body data-scroll class="flex min-h-0 flex-1 flex-col overflow-y-auto"><div data-p-content class="flex flex-1 shrink-0 flex-col"></div></section>
                    </article>
                    ${mobile ? '' : `
                    <aside data-p-details aria-label="Session details" class="ml-3 flex flex-col rounded-3xl bg-mantle 3xl:ml-4">
                        <header class="flex h-14 shrink-0 items-center pl-5 pr-2 md:h-16">
                            <h2 class="flex-1 truncate font-display text-sm font-medium text-text">Session details</h2>
                        </header>
                        <section data-scroll class="min-h-0 min-w-64 flex-1 overflow-y-auto px-3 pb-5"><div data-p-details-body class="flex flex-col gap-6"></div></section>
                    </aside>`}
                </div>
                <input type="file" multiple hidden data-p-file-input>
                <div data-p-drop aria-hidden="true" class="pointer-events-none absolute inset-0 z-50 hidden items-end justify-center rounded-3xl bg-mauve/15 pb-10 [animation:fade-in_.12s_ease-out]">
                    <p class="flex items-center gap-3 rounded-full bg-mauve py-3 pl-5 pr-6 text-sm font-medium text-crust shadow-2xl shadow-crust"><i data-lucide="cloud-upload" class="size-5"></i><span>Drop to upload into <span data-p-drop-target class="font-mono"></span></span></p>
                </div>
            </section>`;
    };

    const render = () => {
        S.tier = tierOf();
        S.panels = {};
        S.sheet?.close();
        const p = project();
        if (!p) {
            S.el.innerHTML = `<section class="h-full rounded-3xl bg-mantle">${ui.empty({
                icon: 'folder-x',
                title: `No tracked project named ${S.name}`,
                hint: 'It may have been untracked, or the link is out of date.',
                action: ui.button('Back to dashboard', { icon: 'arrow-left', action: 'project-back' }),
            })}</section>`;
            ui.icons(S.el);
            return;
        }
        S.el.innerHTML = layout();
        if (S.tier !== 'mobile') {
            const tree = $('[data-p-tree]');
            S.panels.tree = ui.sidebar(tree, { key: `project.tree.${S.tier}`, min: 200, max: 560, initial: S.tier === 'wide' ? 340 : 280 });
            const syncTree = (c) => { tree.style.marginRight = c ? '0px' : ''; };
            S.panels.tree.onToggle = syncTree;
            syncTree(S.panels.tree.collapsed);
            const details = $('[data-p-details]');
            S.panels.details = ui.sidebar(details, { key: `project.details.${S.tier}`, min: 260, max: 560, initial: S.tier === 'wide' ? 400 : 330, side: 'right', collapsed: S.tier === 'laptop' });
            const syncDetails = (c) => { details.style.marginLeft = c ? '0px' : ''; };
            S.panels.details.onToggle = syncDetails;
            syncDetails(S.panels.details.collapsed);
        }
        $('[data-p-file-input]').addEventListener('change', (e) => {
            const files = [...e.target.files];
            e.target.value = '';
            if (files.length) upload(S.folder, files);
        });
        $('[data-p-tree-list]').addEventListener('keydown', treeKeys);
        const body = $('[data-p-viewer-body]');
        body.addEventListener('scroll', spy, { passive: true });
        body.addEventListener('click', markdownLinks);
        new ResizeObserver(() => {
            if (!S || !body.isConnected) return;
            renderOutline();
            spy();
        }).observe(body);
        S.el.addEventListener('ui-change', (e) => {
            if (e.detail.name !== 'project-mode') return;
            S.mode = e.detail.value;
            renderViewerBody(true);
        });
        renderHeader();
        renderDetails();
        ui.icons(S.el);
        if (S.tree) {
            renderTree();
            renderViewer(false);
        } else {
            renderViewer(true);
            loadTree().then(() => {
                renderTree();
                openFile(defaultPath());
            });
        }
    };

    const treeKeys = (e) => {
        const ul = e.currentTarget;
        const rows = [...ul.querySelectorAll('button[data-path]')].filter((b) => b.offsetParent !== null);
        const i = rows.indexOf(document.activeElement);
        if (i < 0) return;
        const row = rows[i];
        const dir = row.dataset.treeDir !== undefined ? row.dataset.treeDir : null;
        const open = dir !== null && S.expanded.has(dir);
        const focus = (b) => {
            if (!b) return;
            rows.forEach((r) => { r.tabIndex = -1; });
            b.tabIndex = 0;
            b.focus();
        };
        if (e.key === 'ArrowDown') focus(rows[i + 1]);
        else if (e.key === 'ArrowUp') focus(rows[i - 1]);
        else if (e.key === 'ArrowRight' && dir !== null) open ? focus(rows[i + 1]) : setExpanded(dir, true);
        else if (e.key === 'ArrowLeft') {
            if (open) setExpanded(dir, false);
            else focus(row.parentElement.parentElement.closest('li')?.firstElementChild);
        } else return;
        e.preventDefault();
    };

    const markdownLinks = (e) => {
        const a = e.target.closest('.markdown-body a[href]');
        if (!a) return;
        const href = a.getAttribute('href');
        if (/^[a-z]+:/i.test(href)) return;
        e.preventDefault();
        if (href.startsWith('#')) {
            document.getElementById(href.slice(1))?.scrollIntoView({ behavior: 'smooth', block: 'start' });
            return;
        }
        const target = resolvePath(dirname(S.selected), href.split('#')[0]);
        const node = S.index.get(target);
        if (!node) return ui.toast(`${target} is not in this project`, { icon: 'file-x-2' });
        if (node.type === 'dir') revealDir(target);
        else openFile(target);
    };

    const showDrop = (folder) => {
        const overlay = $('[data-p-drop]');
        if (!overlay) return;
        overlay.classList.toggle('hidden', folder === null);
        overlay.classList.toggle('flex', folder !== null);
        if (folder !== null) overlay.querySelector('[data-p-drop-target]').textContent = folderLabel(folder);
        S.el.querySelectorAll('[data-drop]').forEach((r) => r.removeAttribute('data-drop'));
        if (folder) $(`[data-tree-dir="${CSS.escape(folder)}"]`)?.setAttribute('data-drop', '');
    };

    const dropFolder = (e) => {
        const row = e.target.closest?.('[data-tree-dir]');
        return row ? row.dataset.treeDir : S.folder;
    };

    const hasFiles = (e) => [...(e.dataTransfer?.types || [])].includes('Files');

    const onDragEnter = (e) => {
        if (!hasFiles(e) || !project()) return;
        e.preventDefault();
        S.dragDepth++;
        showDrop(dropFolder(e));
    };
    const onDragOver = (e) => {
        if (!hasFiles(e) || !project()) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'copy';
        showDrop(dropFolder(e));
    };
    const onDragLeave = (e) => {
        if (!hasFiles(e)) return;
        S.dragDepth = Math.max(0, S.dragDepth - 1);
        if (!S.dragDepth) showDrop(null);
    };
    const onDrop = (e) => {
        if (!hasFiles(e) || !project()) return;
        e.preventDefault();
        S.dragDepth = 0;
        const folder = dropFolder(e);
        showDrop(null);
        const files = [...e.dataTransfer.files];
        if (files.length) upload(folder, files);
    };

    const onKey = (e) => {
        if (e.key === 'Escape' && S.drawer && !ui.modal.stack.length) setDrawer(false);
    };

    const onMedia = () => {
        if (tierOf() !== S.tier) {
            S.drawer = false;
            render();
        }
    };

    const upload = async (folder, files) => {
        const n = files.length;
        const r = await app.act('Upload', () => api.upload(S.name, folder, files), { success: `Uploaded ${n} file${n === 1 ? '' : 's'} to ${folderLabel(folder)}` });
        if (r.ok) openFile(folder ? `${folder}/${files[0].name}` : files[0].name);
    };

    const runBusy = async (key, label, fn, opts) => {
        S.busy = key;
        renderHeader();
        renderDetails();
        const cur = S;
        const r = await app.act(label, fn, opts);
        if (S !== cur) return r;
        S.busy = null;
        renderHeader();
        renderDetails();
        return r;
    };

    const restart = () => {
        const p = project();
        return runBusy('restart', 'Restart', () => api.restart(p.name), { success: `Restarted ${p.rcName} with --continue` });
    };

    const offerRestart = async (what) => {
        const p = project();
        if (!p?.active) return;
        const yes = await ui.confirm({ title: 'Restart to apply?', icon: 'rotate-cw', message: `${what} The running session only picks this up after a restart, which continues the same conversation.`, confirmLabel: 'Restart now' });
        if (yes) restart();
    };

    const field = (label, control, hint = '') => `<label class="block"><span class="${ui.cls.label} mb-1.5 block px-1">${label}</span>${control}${hint ? `<span class="mt-1 block px-1 text-xs text-overlay1">${hint}</span>` : ''}</label>`;

    const errorLine = '<p data-form-error class="hidden rounded-2xl bg-red/10 px-4 py-2.5 text-sm text-red"></p>';
    const showError = (root, msg) => {
        const el = root.querySelector('[data-form-error]');
        el.textContent = msg;
        el.classList.remove('hidden');
    };

    const openNewFile = () => {
        const dirs = [...S.index.values()].filter((n) => n.type === 'dir').map((n) => ({ value: n.path, label: n.path ? `${n.path}/` : '/ (project root)', icon: n.path ? 'folder' : 'folder-git-2' }));
        const m = ui.modal.open({
            title: 'New file',
            icon: 'file-plus',
            subtitle: ui.esc(S.name),
            body: `<div class="flex flex-col gap-4 pt-1">
                ${field('Folder', ui.select({ name: 'nf-folder', value: S.folder, options: dirs, searchable: dirs.length > 8, icon: 'folder' }))}
                ${field('File name', `<input data-name="nf-name" class="${ui.cls.input} font-mono" placeholder="notes.md" autocomplete="off" spellcheck="false">`)}
                ${field('Content', `<textarea data-name="nf-content" rows="12" spellcheck="false" class="${ui.cls.input} min-h-48 resize-y font-mono leading-relaxed" placeholder="Paste the file contents"></textarea>`)}
                ${errorLine}
            </div>`,
            footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-modal-cancel' })}${ui.button('Create file', { icon: 'check', variant: 'primary', attrs: 'data-submit' })}`,
        });
        m.el.querySelector('[data-name="nf-name"]').focus();
        m.el.querySelector('[data-modal-cancel]').addEventListener('click', () => m.close());
        m.el.querySelector('[data-submit]').addEventListener('click', async () => {
            const folder = ui.value(m.el, 'nf-folder');
            const name = ui.value(m.el, 'nf-name').trim();
            const content = ui.value(m.el, 'nf-content');
            if (!name) return showError(m.el, 'Enter a file name.');
            if (name.includes('/')) return showError(m.el, 'Pick the folder above; the name cannot contain a slash.');
            const path = folder ? `${folder}/${name}` : name;
            if (S.index.has(path)) return showError(m.el, `${path} already exists.`);
            m.close();
            const r = await app.act('Create file', () => api.createFile(S.name, path, content), { success: `Created ${path}` });
            if (r.ok) openFile(path);
        });
    };

    const kvRow = (key = '', value = '', kp = 'Key') => `<li class="flex items-center gap-2"><input data-kv-key class="${ui.cls.input} font-mono" placeholder="${kp}" value="${ui.esc(key)}" spellcheck="false"><input data-kv-value class="${ui.cls.input} font-mono" placeholder="Value" value="${ui.esc(value)}" spellcheck="false">${ui.iconButton('x', { title: 'Remove row', size: 'sm', attrs: 'data-kv-remove' })}</li>`;

    const kvList = (name, label, kp) => `
        <fieldset>
            <legend class="${ui.cls.label} mb-1.5 flex w-full items-center px-1"><span class="flex-1">${label}</span></legend>
            <ul data-kv="${name}" data-kp="${kp}" class="flex flex-col gap-2">${kvRow('', '', kp)}</ul>
            ${ui.button(`Add ${kp.toLowerCase()}`, { icon: 'plus', size: 'sm', variant: 'ghost', attrs: `data-kv-add="${name}"` })}
        </fieldset>`;

    const readKv = (root, name) => Object.fromEntries([...root.querySelectorAll(`[data-kv="${name}"] li`)]
        .map((li) => [li.querySelector('[data-kv-key]').value.trim(), li.querySelector('[data-kv-value]').value])
        .filter(([k]) => k));

    const openEditMcp = () => {
        const p = project();
        const lib = store.state.mcpLibrary;
        const libNames = new Set(lib.map((m) => m.name));
        const added = [];
        const listHtml = () => {
            const rows = [
                ...lib.map((m) => ({ name: m.name, hint: `${m.transport} · library` })),
                ...p.mcp.filter((n) => !libNames.has(n)).map((n) => ({ name: n, hint: 'custom entry' })),
                ...added.map((e) => ({ name: e.name, hint: `${e.transport} · new custom entry` })),
            ];
            if (!rows.length) return ui.empty({ icon: 'library', title: 'No servers yet', hint: 'Add entries to the library in Settings, or add a custom entry.' });
            return rows.map((r) => ui.checkbox({ name: `mcp:${r.name}`, label: r.name, hint: r.hint, checked: p.mcp.includes(r.name) || added.some((e) => e.name === r.name) })).join('');
        };
        const m = ui.modal.open({
            title: 'MCP servers',
            icon: 'plug',
            subtitle: `Checked servers are written to ${ui.esc(p.name)}/.mcp.json`,
            body: `<div class="flex flex-col gap-4 pt-1">
                ${ui.segmented({ name: 'mcp-source', value: 'list', full: true, options: [{ value: 'list', label: 'Servers', icon: 'list-checks' }, { value: 'custom', label: 'Custom entry', icon: 'square-pen' }] })}
                <section data-pane="list" data-mcp-list>${listHtml()}</section>
                <section data-pane="custom" hidden class="flex flex-col gap-4">
                    ${field('Name', `<input data-name="mcp-name" class="${ui.cls.input} font-mono" placeholder="sentry" autocomplete="off" spellcheck="false">`)}
                    <span class="flex flex-col items-start gap-1.5"><span class="${ui.cls.label} px-1">Transport</span>${ui.segmented({ name: 'mcp-transport', value: 'http', options: [{ value: 'http', label: 'HTTP', icon: 'globe' }, { value: 'stdio', label: 'stdio', icon: 'square-terminal' }] })}</span>
                    <section data-transport="http" class="flex flex-col gap-4">
                        ${field('URL', `<input data-name="mcp-url" class="${ui.cls.input} font-mono" placeholder="https://mcp.example.com/mcp" spellcheck="false">`)}
                        ${kvList('headers', 'Headers', 'Header')}
                    </section>
                    <section data-transport="stdio" hidden class="flex flex-col gap-4">
                        ${field('Command', `<input data-name="mcp-command" class="${ui.cls.input} font-mono" placeholder="npx" spellcheck="false">`)}
                        ${field('Arguments', `<input data-name="mcp-args" class="${ui.cls.input} font-mono" placeholder="-y @scope/server --flag value" spellcheck="false">`, 'Separated by spaces.')}
                        ${kvList('env', 'Environment', 'Variable')}
                    </section>
                    <span>${ui.button('Add to servers', { icon: 'plus', attrs: 'data-custom-add' })}</span>
                </section>
                ${errorLine}
            </div>`,
            footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-modal-cancel' })}${ui.button('Apply', { icon: 'check', variant: 'primary', attrs: 'data-submit' })}`,
        });
        const showPane = (pane) => {
            m.el.querySelectorAll('[data-pane]').forEach((s) => { s.hidden = s.dataset.pane !== pane; });
            const group = m.el.querySelector('[data-name="mcp-source"]');
            group.dataset.value = pane;
            group.querySelectorAll('[role="radio"]').forEach((r) => r.setAttribute('aria-checked', String(r.dataset.value === pane)));
        };
        m.el.addEventListener('ui-change', (e) => {
            if (e.detail.name === 'mcp-source') showPane(e.detail.value);
            if (e.detail.name === 'mcp-transport') m.el.querySelectorAll('[data-transport]').forEach((s) => { s.hidden = s.dataset.transport !== e.detail.value; });
        });
        m.el.addEventListener('click', async (e) => {
            const add = e.target.closest('[data-kv-add]');
            if (add) {
                const list = m.el.querySelector(`[data-kv="${add.dataset.kvAdd}"]`);
                list.insertAdjacentHTML('beforeend', kvRow('', '', list.dataset.kp));
                ui.icons(list);
                list.lastElementChild.querySelector('input').focus();
                return;
            }
            const remove = e.target.closest('[data-kv-remove]');
            if (remove) return remove.closest('li').remove();
            if (e.target.closest('[data-modal-cancel]')) return m.close();
            if (e.target.closest('[data-custom-add]')) {
                const name = ui.value(m.el, 'mcp-name').trim();
                const transport = ui.value(m.el, 'mcp-transport');
                if (!/^[\w.-]+$/.test(name)) return showError(m.el, 'Enter a name using letters, digits, dots, dashes, or underscores.');
                if (libNames.has(name) || p.mcp.includes(name) || added.some((x) => x.name === name)) return showError(m.el, `${name} is already in the list.`);
                const entry = { name, transport };
                if (transport === 'http') {
                    entry.url = ui.value(m.el, 'mcp-url').trim();
                    if (!/^https?:\/\//.test(entry.url)) return showError(m.el, 'Enter an http or https URL.');
                    entry.headers = readKv(m.el, 'headers');
                } else {
                    entry.command = ui.value(m.el, 'mcp-command').trim();
                    if (!entry.command) return showError(m.el, 'Enter the command that starts the server.');
                    entry.args = ui.value(m.el, 'mcp-args').trim().split(/\s+/).filter(Boolean);
                    entry.env = readKv(m.el, 'env');
                }
                added.push(entry);
                m.el.querySelector('[data-form-error]').classList.add('hidden');
                const list = m.el.querySelector('[data-mcp-list]');
                list.innerHTML = listHtml();
                ui.icons(list);
                showPane('list');
                return;
            }
            if (!e.target.closest('[data-submit]')) return;
            const checked = (n) => ui.value(m.el, `mcp:${n}`);
            const names = [...lib.map((x) => x.name), ...p.mcp.filter((n) => !libNames.has(n))].filter(checked);
            const servers = [...names, ...added.filter((x) => checked(x.name))];
            const next = servers.map((x) => (typeof x === 'string' ? x : x.name));
            m.close();
            if (next.length === p.mcp.length && next.every((n) => p.mcp.includes(n))) return;
            const r = await runBusy('mcp', 'Update MCP servers', () => api.setMcp(S.name, servers), { success: next.length ? `.mcp.json now has ${next.join(', ')}` : '.mcp.json has no servers' });
            if (r.ok) offerRestart('The MCP servers changed.');
        });
    };

    const openPresets = () => {
        const p = project();
        const { builtin, remote } = store.state.presets;
        const group = (label, list) => (list.length ? `<h3 class="${ui.cls.label} mt-2 mb-1 px-3">${label}</h3>${list.map((x) => ui.checkbox({ name: `preset:${x.name}`, label: x.name, hint: x.description, checked: p.presets.includes(x.name) })).join('')}` : '');
        const m = ui.modal.open({
            title: 'Presets',
            icon: 'layers',
            subtitle: 'Re-apply, add, or remove presets. Remote presets pull their latest version.',
            body: `<div class="flex flex-col pt-1">${group('Built-in', builtin)}${group('Tracked remote', remote)}<div class="mt-3">${errorLine}</div></div>`,
            footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-modal-cancel' })}${ui.button('Apply presets', { icon: 'check', variant: 'primary', attrs: 'data-submit' })}`,
        });
        m.el.querySelector('[data-modal-cancel]').addEventListener('click', () => m.close());
        m.el.querySelector('[data-submit]').addEventListener('click', async () => {
            const list = [...builtin, ...remote].map((x) => x.name).filter((n) => ui.value(m.el, `preset:${n}`));
            if (!list.length) return showError(m.el, 'Pick at least one preset.');
            m.close();
            const r = await runBusy('presets', 'Apply presets', () => api.applyPresets(S.name, list), { success: `Applied ${list.join(', ')}` });
            if (r.ok) offerRestart('The instruction files and skills changed.');
        });
    };

    const openSwitch = () => {
        const p = project();
        const others = store.state.accounts.filter((a) => a.dir !== p.account);
        const m = ui.modal.open({
            title: 'Switch account',
            icon: 'arrow-left-right',
            subtitle: ui.esc(p.name),
            body: `<div class="flex flex-col gap-4 pt-1">
                <p class="px-1 text-sm leading-relaxed text-subtext0">${p.active ? 'Stops the session, moves' : 'Moves'} this project's conversations to the chosen account, then continues under it as <span class="font-mono text-text">${ui.esc(p.rcName)}</span>.</p>
                ${ui.choices({ name: 'switch-account', value: others[0]?.dir, options: store.state.accounts.map((a) => ({ value: a.dir, label: a.label, hint: a.dir === p.account ? `${a.dir} · current` : a.dir, icon: 'user-round', disabled: a.dir === p.account })) })}
            </div>`,
            footer: `${ui.button('Cancel', { variant: 'ghost', attrs: 'data-modal-cancel' })}${ui.button('Switch and continue', { icon: 'arrow-left-right', variant: 'primary', attrs: 'data-submit' })}`,
        });
        m.el.querySelector('[data-modal-cancel]').addEventListener('click', () => m.close());
        m.el.querySelector('[data-submit]').addEventListener('click', () => {
            const dir = ui.value(m.el, 'switch-account');
            m.close();
            runBusy('switch', 'Switch account', () => api.switchAccount(S.name, dir), { success: `Continuing under ${accountOf(dir)?.label || dir}` });
        });
    };

    const cleanCwd = async () => {
        const yes = await ui.confirm({
            title: 'Run clean-cwd?',
            icon: 'brush-cleaning',
            danger: true,
            confirmLabel: 'Clean',
            message: `Deletes <span class="font-mono text-text">.agents/</span>, <span class="font-mono text-text">AGENTS.md</span>, <span class="font-mono text-text">CLAUDE.md</span>, and <span class="font-mono text-text">.claude/skills</span> from ${ui.esc(S.name)}. The resume file is stashed under <span class="font-mono text-text">~/.config/claudex</span> first, so you can restore it.`,
        });
        if (yes) runBusy('clean', 'clean-cwd', () => api.cleanCwd(S.name), { success: 'Cleaned. The resume file is stashed.' });
    };

    const restoreResume = async () => {
        const r = await runBusy('restore', 'Restore resume file', () => api.restoreResume(S.name), { success: 'Restored .agents/session-resume/resume.md' });
        if (r.ok) openFile('.agents/session-resume/resume.md');
    };

    const untrack = async () => {
        const yes = await ui.confirm({ title: `Untrack ${S.name}?`, icon: 'folder-x', danger: true, confirmLabel: 'Untrack', message: 'Removes it from the dashboard. Files on disk stay as they are.' });
        if (!yes) return;
        const r = await app.act('Untrack', () => api.untrack(S.name), { success: `Untracked ${S.name}`, refresh: false });
        if (!r.ok) return;
        app.go('#/');
        app.refresh();
    };

    ui.on('project-back', () => app.go('#/'));
    ui.on('project-continue', () => views.newSession.open({ name: S.name, mode: 'continue' }));
    ui.on('project-new', () => views.newSession.open({ name: S.name, mode: 'new' }));
    ui.on('project-restart', restart);
    ui.on('project-stop', async () => {
        const p = project();
        const yes = await ui.confirm({ title: `Stop ${p.name}?`, icon: 'square', danger: true, confirmLabel: 'Stop session', message: `Closes the <span class="font-mono text-text">claudex:${ui.esc(p.rcName)}</span> tmux window. The conversation stays on disk, so Continue picks it up later.` });
        if (!yes) return;
        const r = await runBusy('stop', 'Stop', () => api.stop(p.name), { success: `Stopped ${p.rcName}` });
        if (r.ok) app.go('#/');
    });
    ui.on('project-mcp-edit', openEditMcp);
    ui.on('project-presets', openPresets);
    ui.on('project-restore', restoreResume);
    ui.on('project-switch', openSwitch);
    ui.on('project-clean', cleanCwd);
    ui.on('project-untrack', untrack);
    ui.on('project-toggle-tree', () => {
        if (S.tier === 'mobile') return setDrawer(!S.drawer);
        S.panels.tree.toggle();
        renderHeader();
    });
    ui.on('project-drawer-close', () => setDrawer(false));
    ui.on('project-toggle-details', () => {
        if (S.tier === 'mobile') {
            const p = project();
            S.sheet = ui.modal.open({ title: p.name, icon: 'info', subtitle: 'Session details', body: '<div data-p-sheet class="flex flex-col gap-6 pb-2"></div>', onClose: () => { if (S) S.sheet = null; } });
            renderDetails();
            return;
        }
        S.panels.details.toggle();
        renderHeader();
    });
    ui.on('project-tree', (el) => {
        const path = el.dataset.path;
        const node = S.index.get(path);
        if (!node) return;
        if (node.type === 'dir') {
            S.folder = path;
            setExpanded(path, !S.expanded.has(path));
        } else openFile(path);
    });
    ui.on('project-collapse', () => {
        S.expanded.clear();
        renderTree();
        markSelected();
    });
    ui.on('project-crumb', (el) => {
        const path = el.dataset.path;
        if (path === S.selected) return;
        revealDir(path);
    });
    ui.on('project-open-target', (el) => {
        const node = S.index.get(el.dataset.path);
        if (node?.type === 'dir') revealDir(node.path);
        else openFile(el.dataset.path);
    });
    ui.on('project-outline', (el) => {
        const h = S.headings[Number(el.dataset.index)];
        const body = $('[data-p-viewer-body]');
        body.scrollTo({ top: body.scrollTop + h.getBoundingClientRect().top - body.getBoundingClientRect().top - 24, behavior: 'smooth' });
    });
    ui.on('project-copy-file', () => S.file?.content !== undefined && ui.copy(S.file.content));
    ui.on('project-reload-file', () => openFile(S.selected));
    ui.on('project-new-file', () => S.index && openNewFile());
    ui.on('project-upload', () => $('[data-p-file-input]').click());

    const mqs = [matchMedia('(max-width: 767px)'), matchMedia('(min-width: 1920px)')];

    views.project = {
        mount(el, params) {
            S = {
                el,
                name: params[0] || '',
                tier: null,
                tree: null,
                index: new Map(),
                expanded: new Set(),
                selected: null,
                folder: '',
                file: null,
                mode: 'rendered',
                loading: false,
                busy: null,
                panels: {},
                drawer: false,
                dragDepth: 0,
                headings: [],
                sheet: null,
            };
            document.addEventListener('dragenter', onDragEnter);
            document.addEventListener('dragover', onDragOver);
            document.addEventListener('dragleave', onDragLeave);
            document.addEventListener('drop', onDrop);
            document.addEventListener('keydown', onKey);
            mqs.forEach((q) => q.addEventListener('change', onMedia));
            render();
        },

        async update() {
            if (!S) return;
            const p = project();
            const showing = !!$('[data-project]');
            if (!p || !showing) {
                if (!p && !showing) return;
                render();
                return;
            }
            renderHeader();
            renderDetails();
            const cur = S;
            await loadTree();
            if (S !== cur || !$('[data-project]')) return;
            renderTree();
            const node = S.index.get(S.selected);
            if (!node || node.type === 'dir') return openFile(defaultPath());
            markSelected();
            try {
                const f = await api.file(S.name, S.selected);
                if (S !== cur || S.selected !== f.path) return;
                if (f.content !== S.file?.content || f.target !== S.file?.target || f.size !== S.file?.size || S.file?.error) {
                    S.file = f;
                    renderViewer(false);
                }
            } catch {}
        },

        unmount() {
            document.removeEventListener('dragenter', onDragEnter);
            document.removeEventListener('dragover', onDragOver);
            document.removeEventListener('dragleave', onDragLeave);
            document.removeEventListener('drop', onDrop);
            document.removeEventListener('keydown', onKey);
            mqs.forEach((q) => q.removeEventListener('change', onMedia));
            S.sheet?.close();
            S = null;
        },
    };
})();
