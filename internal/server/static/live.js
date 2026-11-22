const live = (() => {
    const s = {
        es: null,
        session: '',
        handler: null,
        lastId: '',
        heard: 0,
        retry: 0,
        retryTimer: 0,
        refreshTimer: 0,
        summary: {},
        started: false,
    };

    const STATES = {
        starting: ['Starting', 'neutral', true],
        idle: ['Idle', 'neutral', false],
        running: ['Working', 'green', true],
        requires_action: ['Needs approval', 'yellow', true],
        exited: ['Exited', 'red', false],
    };

    const stateChip = (state) => {
        const [label, tone, pulse] = STATES[state] || STATES.idle;
        return ui.status(label, tone, { pulse });
    };

    const onScreen = () => (app.current === 'project' ? app.params[0] || '' : '');

    const shape = (m) => Object.entries(m).map(([n, v]) => `${n}:${v.epoch}:${v.state === 'exited'}`).sort().join(',');

    const applySummary = (next) => {
        const prev = s.summary;
        s.summary = next;
        const here = onScreen();
        for (const [name, v] of Object.entries(next)) {
            if (v.pending > (prev[name]?.pending || 0) && name !== here) {
                ui.toast(`${name} is waiting for approval`, {
                    tone: 'yellow',
                    icon: 'shield-question',
                    timeout: 8000,
                    action: { label: 'Open', onClick: () => app.go(`#/project/${encodeURIComponent(name)}`) },
                });
            }
        }
        if (s.started && shape(prev) !== shape(next)) {
            clearTimeout(s.refreshTimer);
            s.refreshTimer = setTimeout(() => app.refresh(), 300);
        }
        s.started = true;
        app.renderNav();
        document.dispatchEvent(new CustomEvent('live-summary'));
    };

    const open = (cursor) => {
        s.es?.close();
        clearTimeout(s.retryTimer);
        const q = new URLSearchParams();
        if (s.session) q.set('session', s.session);
        if (cursor) q.set('after', cursor);
        const qs = q.toString();
        const es = new EventSource(`/api/events${qs ? `?${qs}` : ''}`);
        s.es = es;
        s.heard = Date.now();
        const listen = (name, fn) => es.addEventListener(name, (e) => {
            if (s.es !== es) return;
            s.heard = Date.now();
            s.retry = 0;
            let data;
            try {
                data = JSON.parse(e.data);
            } catch {
                return;
            }
            fn(data, e);
        });
        listen('summary', (data) => applySummary(data));
        listen('reset', (data, e) => {
            s.lastId = data.session ? e.lastEventId : '';
            s.handler?.('reset', data, s.lastId);
        });
        listen('session', (data) => s.handler?.('session', data, ''));
        for (const name of ['message', 'approval']) {
            listen(name, (data, e) => {
                s.lastId = e.lastEventId;
                s.handler?.(name, data, e.lastEventId);
            });
        }
        es.onerror = () => {
            if (s.es !== es || es.readyState !== EventSource.CLOSED) return;
            s.retryTimer = setTimeout(() => open(s.lastId), Math.min(15000, 1000 * 2 ** s.retry++));
        };
    };

    return {
        start() {
            open('');
            setInterval(() => {
                if (document.visibilityState === 'visible' && Date.now() - s.heard > 45000) open(s.lastId);
            }, 5000);
            document.addEventListener('visibilitychange', () => {
                if (document.visibilityState === 'visible') open(s.lastId);
            });
        },

        watch(name, handler) {
            s.session = name;
            s.handler = handler;
            s.lastId = '';
            open('');
        },

        unwatch() {
            if (!s.session) return;
            s.session = '';
            s.handler = null;
            s.lastId = '';
            open('');
        },

        reconnect(fresh = false) {
            open(fresh ? '' : s.lastId);
        },

        get summary() {
            return s.summary;
        },

        awayPending() {
            const here = onScreen();
            return Object.entries(s.summary).filter(([n, v]) => v.pending > 0 && n !== here).length;
        },

        stateChip,
    };
})();
