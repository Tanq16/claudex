# WebSocket Client

A small reconnecting WebSocket client as a standalone ES module (`public/js/ws.js`). It opens a socket to the backend, reconnects with exponential backoff and jitter on close, dispatches incoming JSON messages by their `type` field, and queues outgoing messages until the socket is open. The `node-backend` skill owns the matching server end.

Messages are JSON objects with a `type` string; everything else in the object is the payload for that type.

## public/js/ws.js

```javascript
export function connect(path, options = {}) {
    const { onStatus = () => {}, baseDelay = 500, maxDelay = 15000 } = options;
    const handlers = new Map();
    const queue = [];

    let socket = null;
    let attempts = 0;
    let closed = false;

    function url() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${proto}//${location.host}${path}`;
    }

    function backoff() {
        const capped = Math.min(baseDelay * 2 ** attempts, maxDelay);
        return capped / 2 + Math.random() * (capped / 2);
    }

    function open() {
        socket = new WebSocket(url());

        socket.addEventListener('open', () => {
            attempts = 0;
            onStatus(true);
            while (queue.length) socket.send(queue.shift());
        });

        socket.addEventListener('message', (event) => {
            let msg;
            try {
                msg = JSON.parse(event.data);
            } catch {
                return;
            }
            const fns = handlers.get(msg.type);
            if (fns) for (const fn of fns) fn(msg);
        });

        socket.addEventListener('close', () => {
            onStatus(false);
            if (closed) return;
            const delay = backoff();
            attempts += 1;
            setTimeout(open, delay);
        });

        socket.addEventListener('error', () => socket.close());
    }

    open();

    return {
        on(type, handler) {
            const fns = handlers.get(type) ?? [];
            fns.push(handler);
            handlers.set(type, fns);
            return this;
        },
        send(type, payload = {}) {
            const data = JSON.stringify({ type, ...payload });
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(data);
            } else {
                queue.push(data);
            }
        },
        close() {
            closed = true;
            if (socket) socket.close();
        },
    };
}
```

## Usage

```javascript
import { connect } from '/js/ws.js';

const socket = connect('/ws', {
    onStatus(online) {
        document.getElementById('status').textContent = online ? 'online' : 'reconnecting…';
    },
});

socket
    .on('update', (msg) => render(msg.payload))
    .on('error', (msg) => console.error('ERROR', msg.message));

socket.send('subscribe', { channel: 'events' });
```
