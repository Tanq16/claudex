# Reader

**PDF.js renders the document in a `base` well through its own viewer, loaded on first use and torn down whenever the view leaves.**

## Assets

PDF.js is `pdfjs-dist@6.5.438`, vendored under `js/pdfjs/` by the Makefile's `pdfjs` target. The files are `build/pdf.min.mjs`, `build/pdf.worker.min.mjs`, `web/pdf_viewer.mjs`, `web/pdf_viewer.css`, `web/images/loading-icon.gif`, and `wasm/openjpeg.wasm`, `wasm/jbig2.wasm`, and `wasm/qcms_bg.wasm`.

```javascript
const PDFJS = '/static/js/pdfjs';
```

## The View

```javascript
reader() {
    return `<section class="flex min-h-0 flex-1 flex-col rounded-3xl bg-mantle">
        <header class="flex shrink-0 items-center gap-2 p-2 sm:gap-3 sm:p-3">
            <span class="max-sm:hidden">${ui.tile('book-open', 'mauve', 'sm')}</span>
            <h2 class="min-w-0 flex-1 truncate pl-2 font-display text-[16px] font-medium text-text sm:pl-0">${ui.esc(MEDIA.book.title)}</h2>
            <span class="flex shrink-0 items-center gap-1.5 text-xs tabular-nums text-overlay1"><input data-reader-page inputmode="numeric" autocomplete="off" aria-label="Page number" class="h-8 w-12 rounded-full bg-surface0 text-center text-sm text-text tabular-nums outline-none transition-shadow focus:ring-2 focus:ring-mauve/40 pointer-coarse:text-[16px]"><span data-reader-count>/ 0</span></span>
            <span class="flex shrink-0 items-center gap-0.5 rounded-full bg-surface0 p-1">
                ${ui.iconButton('zoom-out', { tip: 'Zoom out', size: 'sm', variant: 'inset', attrs: 'data-action="reader-zoom-out"' })}
                ${ui.iconButton('move-horizontal', { tip: 'Fit to width', size: 'sm', variant: 'inset', attrs: 'data-action="reader-fit"' })}
                ${ui.iconButton('zoom-in', { tip: 'Zoom in', size: 'sm', variant: 'inset', attrs: 'data-action="reader-zoom-in"' })}
            </span>
            <span class="max-md:hidden">${ui.iconButton('external-link', { tip: "Open in the browser's viewer", attrs: 'data-action="reader-open"' })}</span>
        </header>
        <div class="relative mx-2 mb-2 min-h-0 flex-1 overflow-hidden rounded-2xl bg-base">
            <div data-reader-container data-scroll tabindex="-1" class="absolute inset-0 overflow-auto overscroll-contain outline-none"><div class="pdfViewer py-3 md:py-5"></div></div>
            <div data-reader-loading class="pointer-events-none absolute inset-0 grid place-items-center text-overlay1">${ui.spinner('size-6')}</div>
        </div>
    </section>`;
},
```

The page sits in a `base` well, since white pages vanish against a near-white ground. The header holds a page input pill, the zoom and fit buttons as `inset` icon buttons in a `surface0` track, and an open-in-browser button from `md` up.

## The Controller

```javascript
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

    async open(root) {
        reader.close();
        const seq = reader.seq;
        const container = root.querySelector('[data-reader-container]');
        const loading = root.querySelector('[data-reader-loading]');
        const page = root.querySelector('[data-reader-page]');
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
            eventBus.on('pagechanging', ({ pageNumber }) => { if (document.activeElement !== page) page.value = String(pageNumber); });
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
            reader.task = lib.getDocument({ url: MEDIA.book.src, wasmUrl: `${PDFJS}/wasm/` });
            const doc = await reader.task.promise;
            if (seq !== reader.seq) return;
            pdfViewer.setDocument(doc);
            linkService.setDocument(doc);
            root.querySelector('[data-reader-count]').textContent = `/ ${doc.numPages}`;
            page.value = '1';
        } catch (err) {
            if (seq !== reader.seq) return;
            loading.hidden = true;
            ui.toast(`Failed to open "${MEDIA.book.title}": ${err.message}`, { tone: 'red', icon: 'circle-alert' });
        }

        page.addEventListener('focus', () => page.select());
        page.addEventListener('keydown', (e) => { if (e.key === 'Enter') page.blur(); });
        page.addEventListener('change', () => {
            const v = reader.view;
            if (!v?.pdfDocument) return;
            const n = Math.round(Number(page.value));
            if (n >= 1 && n <= v.pagesCount) v.currentPageNumber = n;
            page.value = String(v.currentPageNumber);
        });
    },

    renderFit(root, preset) {
        const b = root.querySelector('[data-action="reader-fit"]');
        const width = preset === 'page-width';
        b.innerHTML = `<i data-lucide="${width ? 'move-vertical' : 'move-horizontal'}" class="size-4"></i>`;
        b.dataset.tip = width ? 'Fit page' : 'Fit to width';
        b.setAttribute('aria-label', b.dataset.tip);
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
```

The modules load once, on the first open, and a failed load clears the cached promise so the next open retries. `seq` rises on every close, so an open that finishes after the view has left drops its result instead of drawing into a dead container.

The document opens at `page-width` on a phone and `auto` elsewhere. The fit button flips between `page-width` and `page-fit`, and a pinch zooms through PDF.js's `TouchManager`.

A `ResizeObserver` on the container reassigns the current preset whenever the container changes size, because the viewer keeps its old scale otherwise. A window `resize` misses a collapsed sidebar or a narrowed column, which resize the container and leave the window alone.

Closing aborts the pinch listeners, disconnects the observer, detaches the document, and destroys the loading task.

```javascript
ui.on('reader-zoom-in', () => reader.view?.increaseScale());
ui.on('reader-zoom-out', () => reader.view?.decreaseScale());
ui.on('reader-fit', () => {
    if (reader.view) reader.view.currentScaleValue = reader.view.currentScaleValue === 'page-width' ? 'page-fit' : 'page-width';
});
ui.on('reader-open', () => window.open(MEDIA.book.src, '_blank', 'noopener'));
```
