const api = (() => {
    const enc = encodeURIComponent;
    const project = (name) => `/api/projects/${enc(name)}`;

    const request = async (method, url, body) => {
        const init = { method };
        if (body instanceof FormData) init.body = body;
        else if (body !== undefined) {
            init.headers = { 'Content-Type': 'application/json' };
            init.body = JSON.stringify(body);
        }
        let res;
        try {
            res = await fetch(url, init);
        } catch {
            throw new Error('claudex serve is not reachable');
        }
        const text = await res.text();
        let data = null;
        try {
            data = text ? JSON.parse(text) : null;
        } catch {}
        if (!res.ok) throw new Error(data?.error || text.trim() || `${res.status} ${res.statusText}`);
        return data;
    };

    const get = (url) => request('GET', url);
    const post = (url, body) => request('POST', url, body);
    const put = (url, body) => request('PUT', url, body);
    const del = (url) => request('DELETE', url);

    return {
        state: () => get('/api/state'),
        directories: () => get('/api/directories'),
        repos: () => get('/api/repos'),
        validatePreset: (name) => get(`/api/presets/validate?name=${enc(name)}`),

        start: (req) => post('/api/sessions', req),
        stop: (name) => post(`${project(name)}/stop`),
        restart: (name) => post(`${project(name)}/restart`),
        switchAccount: (name, account) => post(`${project(name)}/switch`, { account }),
        untrack: (name) => del(project(name)),
        track: (name, foreignChoice) => post(`${project(name)}/track`, { foreignChoice }),
        setConnectors: (name, connectors) => put(`${project(name)}/connectors`, { connectors }),

        cleanCwd: (name) => post(`${project(name)}/clean`),
        restoreResume: (name) => post(`${project(name)}/restore-resume`),
        applyPresets: (name, presets) => put(`${project(name)}/presets`, { presets }),
        setMcp: (name, servers) => put(`${project(name)}/mcp`, { servers }),

        tree: (name) => get(`${project(name)}/tree`),
        file: (name, path) => get(`${project(name)}/file?path=${enc(path)}`),
        rawUrl: (name, path) => `${project(name)}/raw?path=${enc(path)}`,
        createFile: (name, path, content) => post(`${project(name)}/files`, { path, content }),
        upload: (name, folder, files) => {
            const form = new FormData();
            form.append('folder', folder || '');
            for (const f of files) form.append('files', f);
            return post(`${project(name)}/upload`, form);
        },

        settings: () => get('/api/settings'),
        applyDefaults: (dir) => post('/api/accounts/apply-defaults', { dir }),
        defaults: () => get('/api/defaults'),
        setDefault: (key, value) => put('/api/defaults', { key, value }),
        resetDefault: (key) => del(`/api/defaults?key=${enc(key)}`),
        refreshSchema: () => post('/api/defaults/refresh'),
        addRemotePreset: (repo, path) => post('/api/presets/remote', { repo, path }),
        updateRemotePreset: (name) => post('/api/presets/remote/update', { name }),
        removeRemotePreset: (name) => post('/api/presets/remote/remove', { name }),
        saveMcp: (entry, previousName) => put('/api/mcp', { entry, previousName }),
        deleteMcp: (name) => del(`/api/mcp/${enc(name)}`),
    };
})();
