(() => {
    let S = null;

    const PDFJS = '/static/js/pdfjs';

    const tierOf = () => (ui.media.mobile() ? 'mobile' : ui.media.wide() ? 'wide' : 'laptop');
    const project = () => store.state?.projects.find((p) => p.name === S.name);
    const isAttached = (p) => p?.mode === 'attached';
    const running = (p) => p.active || (isAttached(p) && views.chat.state(p.name) === 'exited');
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
        if (node.kind === 'pdf') return ['file-text', 'text-red'];
        if (node.kind === 'binary') return ['file-box', 'text-overlay1'];
        if (node.kind === 'markdown') return ['file-text', 'text-sky'];
        if (['json', 'yaml', 'yml', 'toml'].includes(ext)) return ['file-braces', 'text-yellow'];
        if (node.kind === 'code') return ['file-code', 'text-peach'];
        return ['file', 'text-overlay1'];
    };

    const wrapLines = {
        on: localStorage.getItem('claudex.wrap') === 'on',
        toggle(b) {
            wrapLines.on = !wrapLines.on;
            try { localStorage.setItem('claudex.wrap', wrapLines.on ? 'on' : 'off'); } catch {}
            b.setAttribute('aria-pressed', String(wrapLines.on));
            S.el.querySelectorAll('[data-code]').forEach((n) => n.toggleAttribute('data-wrap', wrapLines.on));
        },
    };

    const reader = {
        modules: null,
        view: null,
        task: null,
        seq: 0,
        pinch: 1,
        abort: null,
        observer: null,

        load() {
            reader.modules ||= (async () => {
                const link = Object.assign(document.createElement('link'), { rel: 'stylesheet', href: `${PDFJS}/web/pdf_viewer.css` });
                document.head.append(link);
                const lib = await import(`${PDFJS}/build/pdf.min.mjs`);
                lib.GlobalWorkerOptions.workerSrc = `${PDFJS}/build/pdf.worker.min.mjs`;
                const viewer = await import(`${PDFJS}/web/pdf_viewer.mjs`);
                return { lib, viewer };
            })().catch((err) => {
                reader.modules = null;
                throw err;
            });
            return reader.modules;
        },

        async open(root, src, title) {
            reader.close();
            const seq = reader.seq;
            const container = root.querySelector('[data-reader-container]');
            const loading = root.querySelector('[data-reader-loading]');
            const page = () => root.querySelector('[data-reader-page]');
            try {
                const { lib, viewer } = await reader.load();
                if (seq !== reader.seq) return;
                const eventBus = new viewer.EventBus();
                const linkService = new viewer.PDFLinkService({ eventBus, externalLinkTarget: viewer.LinkTarget.BLANK, externalLinkRel: 'noopener noreferrer' });
                const pdfViewer = new viewer.PDFViewer({ container, viewer: container.firstElementChild, eventBus, linkService, removePageBorders: true, annotationEditorMode: lib.AnnotationEditorType.DISABLE });
                linkService.setViewer(pdfViewer);
                eventBus.on('pagesinit', () => {
                    pdfViewer.currentScaleValue = ui.media.mobile() ? 'page-width' : 'auto';
                    loading.hidden = true;
                });
                eventBus.on('pagechanging', ({ pageNumber }) => {
                    const input = page();
                    if (input && document.activeElement !== input) input.value = String(pageNumber);
                });
                eventBus.on('scalechanging', ({ presetValue }) => reader.renderFit(root, presetValue));
                reader.abort = new AbortController();
                new lib.TouchManager({
                    container,
                    isPinchingDisabled: () => !pdfViewer.pdfDocument,
                    onPinching: (origin, prev, dist) => {
                        reader.pinch *= dist / prev;
                        const scale = pdfViewer.currentScale;
                        if (Math.round(scale * reader.pinch * 100) === Math.round(scale * 100)) return;
                        pdfViewer.updateScale({ scaleFactor: reader.pinch, origin, drawingDelay: 400 });
                        reader.pinch = 1;
                    },
                    onPinchEnd: () => { reader.pinch = 1; },
                    signal: reader.abort.signal,
                });
                reader.observer = new ResizeObserver(() => {
                    const v = reader.view;
                    if (v === pdfViewer && v.pdfDocument && ['auto', 'page-width', 'page-fit'].includes(v.currentScaleValue)) {
                        // PDF.js does not refit on resize; assigning the same preset recomputes it for the new width.
                        v.currentScaleValue = v.currentScaleValue;
                    }
                });
                reader.observer.observe(container);
                reader.view = pdfViewer;
                reader.task = lib.getDocument({ url: src, wasmUrl: `${PDFJS}/wasm/` });
                const doc = await reader.task.promise;
                if (seq !== reader.seq) return;
                pdfViewer.setDocument(doc);
                linkService.setDocument(doc);
                const count = root.querySelector('[data-reader-count]');
                if (count) count.textContent = `/ ${doc.numPages}`;
                if (page()) page().value = '1';
            } catch (err) {
                if (seq !== reader.seq) return;
                loading.hidden = true;
                ui.toast(`Failed to open "${title}": ${err.message}`, { tone: 'red', icon: 'circle-alert' });
            }
        },

        renderFit(root, preset) {
            const b = root.querySelector('[data-action="reader-fit"]');
            if (!b) return;
            const width = preset === 'page-width';
            b.innerHTML = `<i data-lucide="${width ? 'move-vertical' : 'move-horizontal'}" class="size-4"></i>`;
            b.title = width ? 'Fit page' : 'Fit to width';
            b.setAttribute('aria-label', b.title);
            ui.icons(b);
        },

        close() {
            reader.seq++;
            reader.abort?.abort();
            reader.abort = null;
            reader.observer?.disconnect();
            reader.observer = null;
            reader.view?.setDocument(null);
            reader.view = null;
            reader.task?.destroy();
            reader.task = null;
        },
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

    const primary = (p, size) => (running(p)
        ? [
            busyButton('restart', 'Restart', { icon: 'rotate-cw', action: 'project-restart', size }),
            busyButton('stop', 'Stop', { icon: 'square', variant: 'danger', action: 'project-stop', size }),
        ]
        : [
            busyButton('continue', 'Continue', { icon: 'play', variant: 'primary', action: 'project-continue', size }),
            busyButton('new', 'New', { icon: 'plus', action: 'project-new', size }),
        ]).join('');

    const railActions = (p) => {
        const b = (key, icon, title, action, variant = 'ghost') => ui.iconButton(S.busy === key ? 'loader-circle' : icon, { title, action, variant, spin: S.busy === key, disabled: !!S.busy });
        return running(p)
            ? b('restart', 'rotate-cw', 'Restart', 'project-restart') + b('stop', 'square', 'Stop', 'project-stop', 'danger')
            : b('continue', 'play', 'Continue', 'project-continue') + b('new', 'plus', 'New session', 'project-new');
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
        const remote = p.rcName
            ? `<span class="flex items-center gap-1.5"><i data-lucide="radio-tower" class="size-3.5 shrink-0 text-overlay1"></i><span class="min-w-0 wrap-anywhere">${ui.esc(p.rcName)}</span></span>`
            : '<span class="text-overlay1">Off</span>';
        const state = isAttached(p) ? views.chat.state(p.name) : null;
        const snap = isAttached(p) ? views.chat.snapshot(p.name) : null;
        return `
            <section>
                <dl class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-5 gap-y-3 px-2 pt-1 text-sm">
                    ${row('Status', state ? live.stateChip(state) : ui.status(p.active))}
                    ${snap ? row('Model', views.chat.modelLabel(snap)) : ''}
                    ${snap ? row('Permissions', ui.esc(views.chat.modeLabel(snap.permissionMode || 'default'))) : ''}
                    ${row('Started', p.active && p.startedAt ? ui.timeAgo(p.startedAt) : 'Not running')}
                    ${row('Last active', ui.timeAgo(p.lastActive))}
                    ${p.mode === 'attached' ? '' : row('Remote Control', remote)}
                    ${row('Account', acc ? `<span class="block truncate">${ui.esc(acc.label)}</span><span class="block truncate font-mono text-xs text-overlay1">${ui.esc(acc.dir)}</span>` : ui.esc(p.account))}
                    ${row('Path', `<span class="block break-all font-mono text-xs">${ui.esc(p.dir)}</span>`)}
                    ${row('Connectors', p.active ? ui.toggle({ name: 'project-connectors', checked: p.connectors, label: 'Connectors', disabled: busy }) : '<span class="text-overlay1">n/a</span>')}
                </dl>
            </section>
            ${isAttached(p) ? '<div data-p-live class="flex flex-col gap-6 empty:hidden"></div>' : ''}
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
                ${sectionHead('Presets', ui.button('Edit', { icon: 'square-pen', size: 'sm', variant: 'ghost', action: 'project-presets', disabled: busy }))}
                ${chips(p.presets, 'layers', 'None applied.')}
            </section>
            <section>
                ${sectionHead('Actions')}
                <ul class="flex flex-col gap-0.5">
                    ${actionRow('clipboard-copy', 'Copy session', 'The conversation as Markdown', 'project-copy-session')}
                    ${actionRow('square-terminal', 'Run a command', 'A shell in the project directory', 'project-shell')}
                    ${multi ? actionRow('arrow-left-right', 'Switch account', p.active ? 'Stop, move conversations, continue' : 'Move conversations to another account', 'project-switch') : ''}
                    ${actionRow('brush-cleaning', 'Run clean-cwd', 'Remove claudex files, stash the resume file', 'project-clean', { danger: true })}
                    ${actionRow('folder-x', 'Untrack', p.active ? 'Stop the session first' : 'Remove from the dashboard', 'project-untrack', { danger: true, disabled: p.active })}
                </ul>
            </section>`;
    };

    const renderSession = () => {
        const p = project();
        if (!p) return;
        const paint = (el, html) => {
            if (!el) return;
            el.innerHTML = html;
            ui.icons(el);
        };
        paint($('[data-p-session-head]'), `
            <h2 class="min-w-0 flex-1 truncate font-display text-sm font-medium text-text">Session</h2>
            ${primary(p, 'sm')}
            ${ui.iconButton('panel-right-close', { title: 'Hide session details', action: 'project-toggle-details', size: 'sm' })}`);
        paint($('[data-p-rail="right"]'), `${ui.iconButton('panel-right-open', { title: 'Show session details', action: 'project-toggle-details' })}${railActions(p)}`);
        paint($('[data-p-bar]'), `
            ${ui.iconButton('folder-tree', { title: 'Files', action: 'project-toggle-tree' })}
            <div class="flex min-w-0 flex-1 items-center gap-2 px-1">
                <h1 class="truncate font-display text-xl text-text">${ui.esc(p.name)}</h1>
                <span title="${p.active ? 'Active' : 'Inactive'}" class="size-2 shrink-0 rounded-full ${p.active ? 'bg-green' : 'bg-overlay0'}"></span>
            </div>
            ${primary(p, 'md')}
            ${ui.iconButton('info', { title: 'Session details', action: 'project-toggle-details' })}`);
        paint($('[data-p-details-body]'), detailsHtml(p));
        if (S.sheet?.el.isConnected) paint(S.sheet.el.querySelector('[data-p-sheet]'), detailsHtml(p));
        if (isAttached(p)) views.chat.paintLive();
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
        else if (S.panels.tree?.collapsed) S.panels.tree.toggle(false);
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
        showPane('file');
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
        if (f.kind === 'binary' || f.kind === 'image' || f.kind === 'pdf') return fmtSize(f.size || 0);
        const lines = f.content.replace(/\n$/, '').split('\n').length;
        return [f.kind === 'markdown' ? 'markdown' : f.lang, `${lines} line${lines === 1 ? '' : 's'}`, fmtSize(new Blob([f.content]).size)].filter(Boolean).join(' · ');
    };

    const readerControls = () => {
        const width = reader.view?.currentScaleValue === 'page-width';
        return `
            <span class="flex shrink-0 items-center gap-1.5 text-xs tabular-nums text-overlay1"><input data-reader-page inputmode="numeric" autocomplete="off" aria-label="Page number" value="${reader.view?.currentPageNumber ?? ''}" class="h-8 w-12 rounded-full bg-surface0 text-center text-sm text-text tabular-nums outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px]"><span data-reader-count>/ ${reader.view?.pagesCount ?? 0}</span></span>
            <span class="flex shrink-0 items-center gap-0.5 rounded-full bg-surface0 p-1">
                ${ui.iconButton('zoom-out', { title: 'Zoom out', size: 'sm', variant: 'inset', action: 'reader-zoom-out' })}
                ${ui.iconButton(width ? 'move-vertical' : 'move-horizontal', { title: width ? 'Fit page' : 'Fit to width', size: 'sm', variant: 'inset', action: 'reader-fit' })}
                ${ui.iconButton('zoom-in', { title: 'Zoom in', size: 'sm', variant: 'inset', action: 'reader-zoom-in' })}
            </span>
            <span class="max-md:hidden">${ui.iconButton('external-link', { title: "Open in the browser's viewer", size: 'sm', action: 'reader-open' })}</span>`;
    };

    const renderViewerHeader = () => {
        const header = $('[data-p-viewer-header]');
        if (!header) return;
        const f = S.file;
        const ok = f && !f.error;
        const textual = ok && (f.kind === 'markdown' || f.kind === 'code');
        const rendered = ok && f.kind === 'markdown' && S.mode === 'rendered';
        header.innerHTML = `
            <nav aria-label="Path" class="flex min-w-0 flex-1 flex-wrap items-center gap-1">${crumbs(S.selected || '')}</nav>
            <p class="flex items-center gap-2 text-xs text-overlay1">${S.loading ? '<i data-lucide="loader-circle" class="size-3.5 animate-spin"></i>' : ''}<span class="hidden sm:inline">${ui.esc(fileMeta(f))}</span></p>
            ${ok && f.kind === 'pdf' ? readerControls() : ''}
            ${ok && f.kind === 'markdown' ? ui.segmented({ name: 'project-mode', value: S.mode, options: [{ value: 'rendered', label: 'Rendered', icon: 'eye', compact: true }, { value: 'source', label: 'Source', icon: 'code', compact: true }] }) : ''}
            ${textual && !rendered ? ui.iconButton('wrap-text', { title: 'Wrap lines', size: 'sm', pressed: wrapLines.on, action: 'project-wrap' }) : ''}
            ${rendered ? `<span data-p-contents hidden>${ui.iconButton('list', { title: 'Contents', size: 'sm', action: 'project-contents' })}</span><span class="max-md:hidden">${ui.iconButton('fold-horizontal', { title: 'Narrow column', size: 'sm', pressed: column.confined, action: 'column-toggle' })}</span>` : ''}
            ${textual ? ui.iconButton('copy', { title: 'Copy file contents', action: 'project-copy-file', size: 'sm' }) : ''}`;
        ui.icons(header);
    };

    const splitLines = (html) => {
        const lines = [];
        const open = [];
        let line = '';
        for (const part of html.split(/(<span[^>]*>|<\/span>|\n)/)) {
            if (part === '\n') {
                lines.push(line + '</span>'.repeat(open.length));
                line = open.join('');
            } else {
                if (part.startsWith('<span')) open.push(part);
                else if (part === '</span>') open.pop();
                line += part;
            }
        }
        lines.push(line + '</span>'.repeat(open.length));
        return lines;
    };

    const codeWell = (content, lang) => {
        const text = content.replace(/\n$/, '');
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        const lines = splitLines(hljs.highlight(text, { language }).value);
        return `<pre data-p-scroller data-scroll data-code ${wrapLines.on ? 'data-wrap' : ''} style="--gutter:${String(lines.length).length}ch" class="code-lines m-3 min-h-0 overflow-auto rounded-2xl bg-base py-4 font-mono text-[13px] leading-6 md:m-4 3xl:text-sm 3xl:leading-7"><code class="hljs language-${language} block overflow-visible! bg-transparent! p-0! text-subtext1">${lines.map((l) => `<span><span>${l}</span></span>`).join('')}</code></pre>`;
    };

    const readerWell = () => `
        <div class="relative mx-3 mb-3 min-h-0 flex-1 overflow-hidden rounded-2xl bg-base md:mx-4 md:mb-4">
            <div data-reader-container data-scroll tabindex="-1" class="absolute inset-0 overflow-auto overscroll-contain outline-none"><div class="pdfViewer py-3 md:py-5"></div></div>
            <div data-reader-loading class="pointer-events-none absolute inset-0 grid place-items-center text-overlay1"><i data-lucide="loader-circle" class="size-6 animate-spin"></i></div>
        </div>`;

    const notice = ({ icon, tone, title, detail, action = '' }) => `
        <div class="flex flex-1 items-center justify-center p-6">
            <section class="flex w-full max-w-md flex-col items-center gap-3 rounded-3xl bg-base px-6 py-10 text-center">
                <span class="flex size-14 items-center justify-center rounded-full bg-surface0 ${tone}"><i data-lucide="${icon}" class="size-6"></i></span>
                <h2 class="font-display text-lg text-text">${title}</h2>
                ${detail}
                ${action}
            </section>
        </div>`;

    const scroller = (inner) => `<div data-p-scroller data-scroll class="flex min-h-0 flex-1 flex-col overflow-y-auto rounded-b-3xl">${inner}</div>`;

    const renderMarkdown = (content, f) => {
        content.innerHTML = `
            <div class="flex min-h-0 flex-1">
                <div data-p-scroller data-p-doc data-scroll class="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto rounded-bl-3xl px-5 py-6 md:px-10 md:py-8 3xl:px-16 3xl:py-12">
                    <article data-p-markdown class="markdown-body w-full 3xl:text-[17px]!"></article>
                </div>
                <div data-p-outline-wrap class="flex min-h-0 max-w-0 shrink-0 overflow-hidden pl-3 opacity-0 transition-[max-width,opacity] duration-300 ease-out">
                    <nav data-p-outline aria-label="Outline" class="flex min-h-0 flex-col pt-6 md:pt-8 3xl:pt-12"></nav>
                </div>
            </div>`;
        const article = content.querySelector('[data-p-markdown]');
        md.render(article, f.content);
        article.querySelectorAll('img[src]').forEach((img) => {
            const src = img.getAttribute('src');
            if (/^([a-z]+:|\/)/i.test(src)) return;
            img.src = api.rawUrl(S.name, resolvePath(dirname(f.path), src));
        });
        article.querySelectorAll('table').forEach((table) => {
            const wrap = document.createElement('div');
            wrap.className = 'mb-6 overflow-x-auto rounded-xl border border-surface1/50';
            wrap.dataset.scroll = '';
            wrap.dataset.table = '';
            table.replaceWith(wrap);
            wrap.appendChild(table);
        });
        article.querySelectorAll('a[href^="http"]').forEach((a) => {
            a.target = '_blank';
            a.rel = 'noopener';
        });
        article.querySelectorAll(':scope > *').forEach((n) => {
            if (n.matches('.group, .mermaid, [data-table]')) return;
            n.dataset.column = '';
            n.classList.add('mx-auto', 'transition-[max-width]', 'duration-300', 'ease-out', column.cls());
        });
        S.headings = [...article.querySelectorAll('h1, h2, h3')];
        buildOutline(content);
        content.querySelector('[data-p-doc]').addEventListener('scroll', spy, { passive: true });
    };

    const buildOutline = (content) => {
        const nav = content.querySelector('[data-p-outline]');
        if (S.headings.length < 3) return;
        const depth = (h) => Number(h.tagName[1]);
        const minDepth = Math.min(...S.headings.map(depth));
        const indent = (h) => (depth(h) - minDepth) * 12 + 12;
        nav.innerHTML = `
            <h3 class="${ui.cls.label} mb-2 flex shrink-0 items-center gap-2 px-3"><i data-lucide="list-tree" class="size-3.5"></i>On this page</h3>
            <ul data-scroll class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto pr-3 pb-3">${S.headings.map((h, i) => `
                <li><button type="button" data-action="project-outline" data-index="${i}" title="${ui.esc(h.textContent)}" style="padding-left:${indent(h)}px" class="flex min-h-8 w-full items-center rounded-xl py-1.5 pr-3 text-left text-sm text-overlay1 transition-colors hover:bg-surface0/50 hover:text-text data-active:bg-surface0 data-active:text-text data-active:hover:bg-surface0"><span class="truncate">${ui.esc(h.textContent)}</span></button></li>`).join('')}
            </ul>`;
        const ctx = document.createElement('canvas').getContext('2d');
        ctx.font = '14px Inter';
        const widest = Math.max(...S.headings.map((h) => indent(h) + ctx.measureText(h.textContent).width));
        ui.sidebar(nav, { key: 'project.outline', min: 160, max: 320, initial: Math.round(Math.min(320, Math.max(160, widest + 32))), side: 'right', divider: true });
        renderOutline(true);
    };

    const renderOutline = (instant = false) => {
        const wrap = $('[data-p-outline-wrap]');
        if (!wrap) return;
        const nav = wrap.firstElementChild;
        const show = S.headings.length > 2 && wrap.parentElement.clientWidth - nav.offsetWidth - 12 >= 680;
        if (instant) wrap.style.transition = 'none';
        wrap.classList.toggle('max-w-0', !show);
        wrap.classList.toggle('opacity-0', !show);
        wrap.classList.toggle('max-w-96', show);
        wrap.inert = !show;
        const contents = $('[data-p-contents]');
        if (contents) contents.hidden = show || S.headings.length < 3;
        if (instant) {
            void wrap.offsetWidth;
            wrap.style.transition = '';
        }
    };

    const spy = () => {
        const doc = $('[data-p-doc]');
        const nav = $('[data-p-outline]');
        if (!doc || !nav || !S.headings.length) return;
        const offset = doc.getBoundingClientRect().top + 96;
        let active = 0;
        S.headings.forEach((h, i) => {
            if (h.getBoundingClientRect().top <= offset) active = i;
        });
        if (doc.scrollHeight > doc.clientHeight + 4 && doc.scrollTop + doc.clientHeight >= doc.scrollHeight - 4) active = S.headings.length - 1;
        nav.querySelectorAll('[data-index]').forEach((b) => b.toggleAttribute('data-active', Number(b.dataset.index) === active));
    };

    const renderViewerBody = (resetScroll) => {
        const content = $('[data-p-content]');
        if (!content) return;
        const prev = content.querySelector('[data-p-scroller]');
        const top = prev?.scrollTop ?? 0;
        const left = prev?.scrollLeft ?? 0;
        const f = S.file;
        S.headings = [];
        reader.close();
        if (!S.selected || !f) {
            content.innerHTML = scroller(ui.empty({ icon: 'file-search', title: S.tree ? 'Nothing selected' : 'Loading files', hint: S.tree ? 'Pick a file from the tree.' : '' }));
        } else if (f.error) {
            content.innerHTML = scroller(notice({ icon: 'file-x-2', tone: 'text-red', title: 'Could not open this file', detail: `<p class="text-sm text-overlay1">${ui.esc(f.error)}</p>`, action: ui.button('Try again', { icon: 'rotate-cw', size: 'sm', action: 'project-reload-file' }) }));
        } else if (f.kind === 'symlink') {
            const resolved = resolvePath(dirname(f.path), f.target);
            const node = S.index.get(resolved);
            const label = node?.type === 'dir' ? `Reveal ${resolved}/` : `Open ${resolved}`;
            content.innerHTML = scroller(notice({
                icon: 'link-2',
                tone: 'text-teal',
                title: `${ui.esc(f.path.split('/').pop())} is a symlink`,
                detail: `<p class="flex max-w-full items-center gap-2 rounded-full bg-surface0 px-4 py-1.5 font-mono text-sm text-subtext1"><i data-lucide="arrow-right" class="size-4 shrink-0 text-overlay1"></i><span class="truncate">${ui.esc(f.target)}</span></p>`,
                action: node ? ui.button(label, { icon: node.type === 'dir' ? 'folder-open' : 'file-text', variant: 'primary', action: 'project-open-target', attrs: `data-path="${ui.esc(resolved)}"` }) : '<p class="text-sm text-overlay1">The target is outside the project or missing.</p>',
            }));
        } else if (f.kind === 'image') {
            content.innerHTML = scroller(`
                <figure class="flex flex-1 flex-col items-center justify-center gap-3 p-4 md:p-8">
                    <img src="${ui.esc(f.url)}" alt="${ui.esc(f.path)}" class="max-h-[calc(100dvh-20rem)] max-w-full rounded-2xl bg-base object-contain p-4">
                    <figcaption data-p-img-meta class="text-xs text-overlay1">${fmtSize(f.size || 0)}</figcaption>
                </figure>`);
            const img = content.querySelector('img');
            img.addEventListener('load', () => {
                const meta = content.querySelector('[data-p-img-meta]');
                if (meta && img.naturalWidth) meta.textContent = `${img.naturalWidth} × ${img.naturalHeight} · ${fmtSize(f.size || 0)}`;
            });
        } else if (f.kind === 'pdf') {
            content.innerHTML = readerWell();
            reader.open(S.el, f.url, f.path);
        } else if (f.kind === 'binary') {
            const node = S.index.get(f.path);
            const [icon] = node ? fileIcon(node) : ['file-box'];
            content.innerHTML = scroller(notice({
                icon,
                tone: 'text-overlay1',
                title: 'No preview',
                detail: `<p class="text-sm text-overlay1"><span class="font-mono text-subtext1">${ui.esc(f.path.split('/').pop())}</span> · ${fmtSize(f.size || 0)}</p>`,
            }));
        } else if (f.kind === 'markdown' && S.mode === 'rendered') {
            renderMarkdown(content, f);
        } else {
            content.innerHTML = codeWell(f.content, f.kind === 'markdown' ? 'markdown' : f.lang);
        }
        ui.icons(content);
        const next = content.querySelector('[data-p-scroller]');
        if (next && !resetScroll) {
            next.scrollTop = top;
            next.scrollLeft = left;
        }
        spy();
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

    const viewerHtml = (head) => `
        <header data-p-viewer-header class="flex ${head} shrink-0 flex-wrap items-center gap-2 px-3 md:px-4 3xl:px-6"></header>
        <div data-p-content class="relative flex min-h-0 flex-1 flex-col"></div>`;

    const centerHtml = () => (S.centerMode === 'attached'
        ? `<header data-p-center-tabs class="flex min-h-14 shrink-0 items-center gap-2 px-3 py-2 md:min-h-16 md:px-4 3xl:px-6"></header>
            <div data-p-pane="chat" class="flex min-h-0 flex-1 flex-col">${views.chat.html()}</div>
            <div data-p-pane="file" hidden class="flex min-h-0 flex-1 flex-col">${viewerHtml('pb-2')}</div>`
        : viewerHtml('min-h-14 py-2 md:min-h-16'));

    const renderTabs = () => {
        const host = $('[data-p-center-tabs]');
        if (!host) return;
        const options = [{ value: 'chat', label: 'Chat', icon: 'message-square' }];
        if (S.selected) options.push({ value: 'file', label: S.selected.split('/').pop(), icon: fileIcon(S.index.get(S.selected) || { name: S.selected })[0] });
        host.innerHTML = `<span class="min-w-0 overflow-x-auto">${ui.segmented({ name: 'project-pane', value: S.pane, options })}</span>
            <span class="ml-auto ${S.pane === 'chat' ? '' : 'hidden'}">${ui.iconButton('fold-horizontal', { title: 'Narrow column', size: 'sm', pressed: column.confined, action: 'column-toggle' })}</span>`;
        ui.icons(host);
    };

    const showPane = (which) => {
        if (S.centerMode !== 'attached') return;
        if (S.pane === 'chat' && which !== 'chat') views.chat.saveScroll();
        S.pane = which;
        S.el.querySelectorAll('[data-p-pane]').forEach((el) => { el.hidden = el.dataset.pPane !== which; });
        if (which === 'chat') views.chat.restoreScroll();
        renderTabs();
    };

    const mountCenter = () => {
        if (S.centerMode === 'attached') {
            views.chat.mount($('[data-p-pane="chat"]'), S.name, { onSession: renderSession });
            renderTabs();
        }
        const content = $('[data-p-content]');
        content.addEventListener('click', markdownLinks);
        new ResizeObserver(() => {
            if (!S || !content.isConnected) return;
            renderOutline();
            spy();
        }).observe(content);
    };

    const layout = () => {
        const p = project();
        const mobile = S.tier === 'mobile';
        const treeCls = mobile
            ? 'fixed inset-y-3 left-3 z-40 flex w-[min(21rem,calc(100vw-3rem))] flex-col rounded-3xl bg-mantle shadow-2xl shadow-crust transition-transform duration-200'
            : 'mr-3 flex flex-col rounded-3xl bg-mantle 3xl:mr-4';
        const rail = (side, cls, inner = '') => `<nav data-p-rail="${side}" aria-label="${side === 'left' ? 'Files' : 'Session'}" hidden class="${cls} flex shrink-0 flex-col items-center gap-1 pt-2 md:pt-3 [animation:fade-in_.2s_ease-out]">${inner}</nav>`;
        return `
            <section data-project class="relative flex h-full flex-col gap-3">
                ${mobile ? '<header data-p-bar class="flex shrink-0 items-center gap-1"></header>' : ''}
                <div class="flex min-h-0 flex-1">
                    ${mobile
                        ? '<button type="button" data-p-scrim data-action="project-drawer-close" aria-label="Close files" hidden class="fixed inset-0 z-30 bg-crust/70 backdrop-blur-sm [animation:fade-in_.15s_ease-out]"></button>'
                        : rail('left', 'mr-3 3xl:mr-4', ui.iconButton('panel-left-open', { title: 'Show files', action: 'project-toggle-tree' }))}
                    <aside data-p-tree aria-label="Files" class="${treeCls}" ${mobile ? 'style="transform:translateX(calc(-100% - 1.5rem))"' : ''}>
                        <header class="flex h-14 shrink-0 items-center gap-0.5 pr-2 md:h-16 ${mobile ? 'pl-5' : 'pl-2'}">
                            ${mobile ? '' : panelBtn('panel-left-close', 'Hide files', 'project-toggle-tree')}
                            <h2 title="${ui.esc(p.dir)}" class="min-w-0 flex-1 truncate px-1 font-display text-sm font-medium text-text">${ui.esc(p.name)}</h2>
                            ${panelBtn('file-plus', 'New file', 'project-new-file')}
                            ${panelBtn('upload', 'Upload files', 'project-upload')}
                            ${panelBtn('chevrons-down-up', 'Collapse all', 'project-collapse')}
                            ${mobile ? panelBtn('x', 'Close', 'project-drawer-close') : ''}
                        </header>
                        <ul role="tree" data-p-tree-list data-scroll class="min-h-0 flex-1 overflow-y-auto px-2 pb-3"></ul>
                    </aside>
                    <section data-p-center class="flex min-w-0 flex-1 flex-col rounded-3xl bg-mantle">${centerHtml()}</section>
                    ${mobile ? '' : `
                    <aside data-p-details aria-label="Session details" class="ml-3 flex flex-col rounded-3xl bg-mantle 3xl:ml-4">
                        <header data-p-session-head class="flex h-14 shrink-0 items-center gap-1.5 pr-2 pl-5 md:h-16"></header>
                        <section data-scroll="lane" class="min-h-0 min-w-64 flex-1 overflow-y-auto px-3 pb-5"><div data-p-details-body class="flex flex-col gap-6"></div></section>
                    </aside>
                    ${rail('right', 'ml-3 3xl:ml-4')}`}
                </div>
                <input type="file" multiple hidden data-p-file-input>
                <div data-p-drop aria-hidden="true" class="pointer-events-none absolute inset-0 z-50 hidden items-end justify-center rounded-3xl bg-mauve/15 pb-10 [animation:fade-in_.12s_ease-out]">
                    <p class="flex items-center gap-3 rounded-full bg-mauve py-3 pl-5 pr-6 text-sm font-medium text-crust shadow-2xl shadow-crust"><i data-lucide="cloud-upload" class="size-5"></i><span>Drop to upload into <span data-p-drop-target class="font-mono"></span></span></p>
                </div>
            </section>`;
    };

    const readerPage = (root) => {
        root.addEventListener('focusin', (e) => { if (e.target.matches('[data-reader-page]')) e.target.select(); });
        root.addEventListener('keydown', (e) => { if (e.key === 'Enter' && e.target.matches('[data-reader-page]')) e.target.blur(); });
        root.addEventListener('change', (e) => {
            if (!e.target.matches('[data-reader-page]')) return;
            const v = reader.view;
            if (!v?.pdfDocument) return;
            const n = Math.round(Number(e.target.value));
            if (n >= 1 && n <= v.pagesCount) v.currentPageNumber = n;
            e.target.value = String(v.currentPageNumber);
        });
    };

    const render = () => {
        views.chat.unmount();
        S.tier = tierOf();
        S.panels = {};
        S.sheet?.close();
        reader.close();
        const p = project();
        S.centerMode = p?.mode === 'attached' ? 'attached' : 'detached';
        if (S.centerMode !== 'attached') S.pane = 'chat';
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
        const root = $('[data-project]');
        if (S.tier !== 'mobile') {
            const rail = (side, collapsed) => { $(`[data-p-rail="${side}"]`).hidden = !collapsed; };
            S.panels.tree = ui.sidebar($('[data-p-tree]'), { key: `project.tree.${S.tier}`, min: 200, max: 560, initial: S.tier === 'wide' ? 340 : 280 });
            S.panels.tree.onToggle = (c) => rail('left', c);
            rail('left', S.panels.tree.collapsed);
            S.panels.details = ui.sidebar($('[data-p-details]'), { key: `project.details.${S.tier}`, min: 280, max: 560, initial: S.tier === 'wide' ? 400 : 330, side: 'right', collapsed: S.tier === 'laptop' });
            S.panels.details.onToggle = (c) => rail('right', c);
            rail('right', S.panels.details.collapsed);
        }
        $('[data-p-file-input]').addEventListener('change', (e) => {
            const files = [...e.target.files];
            e.target.value = '';
            if (files.length) upload(S.folder, files);
        });
        $('[data-p-tree-list]').addEventListener('keydown', treeKeys);
        root.addEventListener('ui-change', (e) => {
            if (e.detail.name === 'project-pane') showPane(e.detail.value);
            if (e.detail.name !== 'project-mode') return;
            S.mode = e.detail.value;
            renderViewer(true);
        });
        readerPage(root);
        mountCenter();
        renderSession();
        ui.icons(S.el);
        if (S.tree) {
            renderTree();
            renderViewer(false);
            showPane(S.pane);
        } else {
            renderViewer(true);
            loadTree().then(() => {
                renderTree();
                if (S.centerMode !== 'attached') openFile(defaultPath());
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
        renderSession();
        const cur = S;
        const r = await app.act(label, fn, opts);
        if (S !== cur) return r;
        S.busy = null;
        renderSession();
        return r;
    };

    const restart = () => {
        const p = project();
        return runBusy('restart', 'Restart', () => api.restart(p.name), { success: `Restarted ${p.name} with --continue` });
    };

    const setConnectors = async (on) => {
        const p = project();
        const yes = await ui.confirm({ title: `Turn Connectors ${on ? 'on' : 'off'}?`, icon: on ? 'cable' : 'unplug', message: 'The session restarts with --continue to pick up the change.', confirmLabel: 'Restart now' });
        if (!yes) return renderSession();
        runBusy('connectors', 'Connectors', () => api.setConnectors(p.name, on), { success: `Connectors ${on ? 'on' : 'off'}, restarted ${p.name} with --continue` });
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
            subtitle: 'Add, remove, or re-apply presets. Remote presets pull their latest version.',
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
                <p class="px-1 text-sm leading-relaxed text-subtext0">${p.active ? 'Stops the session, moves' : 'Moves'} this project's conversations to the chosen account, then continues <span class="font-mono text-text">${ui.esc(p.name)}</span> under it.</p>
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
    ui.on('project-continue', () => views.newSession.open({ name: S.name, continue: true }));
    ui.on('project-new', () => views.newSession.open({ name: S.name }));
    ui.on('project-copy-session', (b) => ui.menu(b, [
        { icon: 'tags', label: 'Copy with tags', hint: 'Each message starts with [user] or [agent]', onSelect: () => views.chat.copySession(S.name, true) },
        { icon: 'clipboard-copy', label: 'Copy without tags', hint: 'Messages separated by blank lines', onSelect: () => views.chat.copySession(S.name, false) },
    ]));
    ui.on('project-shell', () => views.shell.open(S.name));
    ui.on('project-restart', restart);
    ui.on('project-stop', async () => {
        const p = project();
        const message = isAttached(p)
            ? 'Stops the claude process, its turn, and its background tasks. The conversation stays on disk, so Continue picks it up later.'
            : `Closes the <span class="font-mono text-text">claudex:${ui.esc(p.name)}</span> tmux window. The conversation stays on disk, so Continue picks it up later.`;
        const yes = await ui.confirm({ title: `Stop ${p.name}?`, icon: 'square', danger: true, confirmLabel: 'Stop session', message });
        if (!yes) return;
        const r = await runBusy('stop', 'Stop', () => api.stop(p.name), { success: `Stopped ${p.name}` });
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
        $(S.panels.tree.collapsed ? '[data-p-rail="left"] button' : '[data-p-tree] header button')?.focus();
    });
    ui.on('project-drawer-close', () => setDrawer(false));
    ui.on('project-toggle-details', () => {
        if (S.tier === 'mobile') {
            const p = project();
            S.sheet = ui.modal.open({ title: p.name, icon: 'info', subtitle: 'Session details', body: '<div data-p-sheet class="flex flex-col gap-6 pb-2"></div>', onClose: () => { if (S) S.sheet = null; } });
            renderSession();
            return;
        }
        S.panels.details.toggle();
        $(S.panels.details.collapsed ? '[data-p-rail="right"] button' : '[data-p-session-head] [data-action="project-toggle-details"]')?.focus();
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
    const goHeading = (h) => {
        const doc = $('[data-p-doc]');
        doc.scrollTo({ top: doc.scrollTop + h.getBoundingClientRect().top - doc.getBoundingClientRect().top - 24, behavior: 'smooth' });
    };
    ui.on('project-outline', (el) => goHeading(S.headings[Number(el.dataset.index)]));
    ui.on('project-contents', (b) => ui.menu(b, S.headings.map((h) => ({ icon: h.tagName === 'H3' ? 'corner-down-right' : 'hash', label: h.textContent, onSelect: () => goHeading(h) }))));
    ui.on('project-wrap', (b) => wrapLines.toggle(b));
    ui.on('project-copy-file', (b) => S.file?.content !== undefined && ui.copy(S.file.content, b));
    ui.on('project-reload-file', () => openFile(S.selected));
    ui.on('project-new-file', () => S.index && openNewFile());
    ui.on('project-upload', () => $('[data-p-file-input]').click());
    ui.on('reader-zoom-in', () => reader.view?.increaseScale());
    ui.on('reader-zoom-out', () => reader.view?.decreaseScale());
    ui.on('reader-fit', () => {
        if (reader.view) reader.view.currentScaleValue = reader.view.currentScaleValue === 'page-width' ? 'page-fit' : 'page-width';
    });
    ui.on('reader-open', () => S?.file?.url && window.open(S.file.url, '_blank', 'noopener'));
    document.addEventListener('ui-change', (e) => {
        if (S && e.detail.name === 'project-connectors') setConnectors(e.detail.value);
    });

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
                centerMode: 'detached',
                pane: 'chat',
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
            if ((p.mode === 'attached' ? 'attached' : 'detached') !== S.centerMode) {
                render();
                return;
            }
            renderSession();
            const cur = S;
            await loadTree();
            if (S !== cur || !$('[data-project]')) return;
            renderTree();
            if (S.centerMode === 'attached' && !S.selected) return;
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
            reader.close();
            views.chat.unmount();
            S.sheet?.close();
            S = null;
        },
    };
})();
