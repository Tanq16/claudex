const mock = (() => {
    const now = Date.now();
    const min = 60_000;
    const hour = 60 * min;
    const day = 24 * hour;

    const accountsAll = [
        { dir: '~/.claude', label: 'personal@example.com', isDefault: true },
        { dir: '~/.claude2', label: 'work@example.com', isDefault: false },
        { dir: '~/.claude3', label: 'research@example.com', isDefault: false },
    ];

    const project = (name, extra = {}) => ({
        dir: `~/Documents/${name}`,
        name,
        rcName: name,
        account: '~/.claude',
        active: false,
        startedAt: null,
        lastActive: now - 3 * day,
        presets: ['base'],
        mcp: [],
        connectors: true,
        stash: false,
        ...extra,
    });

    const baseProjects = () => [
        project('claudex', { active: true, startedAt: now - 2 * hour - 14 * min, lastActive: now - 4 * min, presets: ['base', 'private'], mcp: ['github'] }),
        project('anbu', { active: true, account: '~/.claude2', startedAt: now - 26 * hour, lastActive: now - 38 * min, presets: ['base', 'private'], connectors: false }),
        project('telly', { active: true, rcName: 'telly-ui', startedAt: now - 47 * min, lastActive: now - 1 * min, mcp: ['context7', 'linear'] }),
        project('senkaimon', { lastActive: now - 3 * day, presets: ['base', 'private'], stash: true }),
        project('homelab-infra', { account: '~/.claude2', lastActive: now - 5 * day, presets: ['base', 'acme-team-presets/infra'], mcp: ['postgres-local'] }),
        project('notes-vault', { lastActive: now - 8 * day, connectors: false }),
        project('blog', { lastActive: now - 12 * day }),
        project('dotfiles', { lastActive: now - 15 * day, presets: ['base', 'private'] }),
        project('kube-lab', { account: '~/.claude2', lastActive: now - 18 * day, presets: ['base', 'acme-team-presets/infra'] }),
        project('recipes-app', { lastActive: now - 21 * day, presets: ['base', 'acme-team-presets/frontend'] }),
        project('pdf-tools', { lastActive: now - 25 * day }),
        project('ml-notebooks', { account: '~/.claude3', lastActive: now - 30 * day }),
        project('home-dashboard', { lastActive: now - 34 * day, presets: ['base', 'acme-team-presets/frontend'] }),
        project('scratch-api', { lastActive: now - 41 * day }),
    ];

    const manyProjects = () => {
        const extra = ['atlas', 'beacon', 'cinder', 'drift', 'ember', 'fjord', 'glyph', 'harbor', 'ion', 'jetty', 'kelp', 'lumen', 'mosaic', 'nimbus', 'orbit', 'prism', 'quill', 'relay', 'sable', 'tundra', 'umbra', 'vertex', 'willow', 'xenon', 'yarrow', 'zephyr'];
        return [
            ...baseProjects(),
            ...extra.map((n, i) => project(n, { active: i % 7 === 0, startedAt: i % 7 === 0 ? now - (i + 1) * 9 * min : null, lastActive: now - (i + 2) * 19 * hour, account: accountsAll[i % 3].dir })),
        ];
    };

    const untracked = [
        { name: 'old-site', foreign: [] },
        { name: 'experiments', foreign: [] },
        { name: 'vendor-sdk', foreign: ['CLAUDE.md', '.mcp.json'] },
        { name: 'interview-prep', foreign: [] },
        { name: 'photos-sorter', foreign: ['.claude/', 'AGENTS.md'] },
        { name: 'go-playground', foreign: [] },
    ];

    const repoNames = [
        'Tanq16/claudex', 'Tanq16/anbu', 'Tanq16/telly', 'Tanq16/senkaimon', 'Tanq16/dotfiles', 'Tanq16/blog', 'Tanq16/ai-context',
        'Tanq16/danzo', 'Tanq16/nottif', 'Tanq16/expenseowl', 'Tanq16/backhub', 'Tanq16/cybernest', 'Tanq16/local-content-share',
        'Tanq16/gh-actions-lab', 'Tanq16/linkding-sync', 'Tanq16/flow-notes', 'Tanq16/vault-bridge', 'Tanq16/cloudlens', 'Tanq16/kube-lab',
        'Tanq16/rss-reader', 'Tanq16/terminal-themes', 'Tanq16/home-dashboard', 'Tanq16/pdf-tools', 'Tanq16/recipes-app',
        'acme-labs/platform-api', 'acme-labs/web-console', 'acme-labs/design-system', 'acme-labs/infra-modules', 'acme-labs/billing-service',
        'acme-labs/auth-gateway', 'acme-labs/data-pipeline', 'acme-labs/mobile-app', 'acme-labs/docs-site', 'acme-labs/status-page',
        'acme-team-presets/presets', 'oss-friends/fast-yaml', 'oss-friends/tiny-router', 'oss-friends/log-lens', 'oss-friends/htmx-kit',
        'research-group/paper-notes', 'research-group/eval-harness', 'research-group/datasets', 'research-group/plots',
    ];
    const repos = repoNames.map((full, i) => ({
        full,
        description: ['CLI and web control plane', 'Security toolkit', 'Media server', 'Self-hosted notes', 'Shell and editor config', 'Static blog', 'Context tooling', 'Download manager', 'Notifications relay', 'Expense tracker'][i % 10],
        private: i % 4 === 1,
        updatedAt: now - (i * 37 + 3) * hour,
    }));

    const presets = {
        builtin: [
            { name: 'base', description: 'Default rules, session-summary, skill-creator, agents-md, write-document', defaultOn: true },
            { name: 'private', description: 'Go and Node conventions, develop, review-code', defaultOn: false },
        ],
        remote: [
            { name: 'acme-team-presets/infra', repo: 'acme-team-presets/presets', description: 'Terraform and Kubernetes rules', updatedAt: now - 2 * day },
            { name: 'acme-team-presets/frontend', repo: 'acme-team-presets/presets', description: 'React and design-system rules', updatedAt: now - 2 * day },
            { name: 'research-group/notebooks', repo: 'research-group/agent-presets', description: 'Notebook hygiene and plotting', updatedAt: now - 9 * day },
        ],
    };

    const mcpLibrary = [
        { name: 'github', transport: 'http', url: 'https://api.githubcopilot.com/mcp/', headers: { Authorization: 'Bearer ghp_7f3Kx9QmZ2vB8nR4tW1yL6pD0sH5jA' } },
        { name: 'context7', transport: 'http', url: 'https://mcp.context7.com/mcp', headers: { CONTEXT7_API_KEY: 'ctx7sk-a91b2c3d4e5f60718293' } },
        { name: 'linear', transport: 'http', url: 'https://mcp.linear.app/mcp', headers: {} },
        { name: 'postgres-local', transport: 'stdio', command: 'npx', args: ['-y', '@modelcontextprotocol/server-postgres', 'postgresql://localhost:5432/homelab'], env: { PGPASSWORD: 'hunter2-local-only' } },
    ];

    const configureDefaults = {
        enableAllProjectMcpServers: true,
        'statusLine.command': '~/.claude/statusline.sh',
        'permissions.defaultMode': 'acceptEdits',
        includeCoAuthoredBy: false,
        'env.CLAUDE_CODE_DISABLE_TERMINAL_TITLE': '1',
    };

    const compliance = {
        '~/.claude': { settingsFound: true, drift: [] },
        '~/.claude2': {
            settingsFound: true,
            drift: [
                { key: 'enableAllProjectMcpServers', expected: 'true', actual: 'missing' },
                { key: 'statusLine.command', expected: '~/.claude/statusline.sh', actual: '~/bin/status.sh' },
            ],
        },
        '~/.claude3': { settingsFound: false, drift: [] },
    };

    const readme = (name) => `# ${name}

<img src="assets/logo.svg" alt="${name} logo" width="96">

A small, focused tool that lives under \`~/Documents/${name}\`. This README is mock content rendered by the claudex project viewer, so it exercises **every Markdown feature** the viewer styles.

> [!TIP]
> Attach to the running session from a terminal with \`tmux attach -t claudex:${name}\`.

## Install

\`\`\`bash
go install github.com/example/${name}@latest
${name} --help
\`\`\`

## Usage

Run the server and open the page:

\`\`\`go
func main() {
    srv := server.New(server.Config{Host: "127.0.0.1", Port: 8080})
    if err := srv.ListenAndServe(); err != nil {
        log.Fatal().Err(err).Msg("server stopped")
    }
}
\`\`\`

| Flag | Default | Meaning |
|---|---|---|
| \`--host\` | \`127.0.0.1\` | Address to bind |
| \`--port\` | \`8080\` | Port to listen on |
| \`--debug\` | \`false\` | Verbose logging |

### Notes

- Sessions run as windows in the \`claudex\` tmux session.
- A stopped session keeps its transcript, so **Continue** picks it up.
- Nested lists work too:
  1. First step
  2. Second step

> [!WARNING]
> Restarting a session drops any prompt that was typed but not sent.

> [!NOTE]
> This file is read-only in the viewer. Edit it in your editor or ask Claude.

---

See the [Claude Code docs](https://code.claude.com/docs) for more.
`;

    const agentsMd = `# Project rules

## Prose

- Lead with the answer.
- One idea per bullet.
- Never hard-wrap prose.

## Development

- The expected default is 0 code comments.
- Follow YAGNI and KISS.

<!-- claudex:preset:private -->
## Private preset

- Load the develop skill before writing code.
<!-- /claudex:preset -->
`;

    const resumeMd = `# Session resume: serve mock UI

## State
in progress - the mock is built and waiting for review.

## Open
- Untracking a directory from the dashboard.
`;

    const goFile = `package cmd

import (
	"github.com/spf13/cobra"
)

var serveFlags struct {
	host string
	port int
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local web control plane",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runServe(serveFlags.host, serveFlags.port)
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveFlags.host, "host", "127.0.0.1", "Address to bind")
	serveCmd.Flags().IntVar(&serveFlags.port, "port", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
`;

    const mcpJson = (p) => JSON.stringify({ mcpServers: Object.fromEntries(p.mcp.map((n) => [n, { type: 'http', url: `https://mcp.example.com/${n}` }])) }, null, 2);

    const makeFiles = (p) => ({
        'README.md': { kind: 'markdown', content: readme(p.name) },
        'AGENTS.md': { kind: 'markdown', content: agentsMd },
        'CLAUDE.md': { kind: 'symlink', target: 'AGENTS.md' },
        '.mcp.json': { kind: 'code', lang: 'json', content: mcpJson(p) },
        '.agents/session-resume/resume.md': { kind: 'markdown', content: resumeMd },
        '.agents/skills/develop/SKILL.md': { kind: 'markdown', content: '# Develop\n\n**Work out what the task is, load the skills that govern it.**\n' },
        '.claude/skills': { kind: 'symlink', target: '../.agents/skills' },
        'cmd/root.go': { kind: 'code', lang: 'go', content: goFile.replace('serve', 'root') },
        'cmd/serve.go': { kind: 'code', lang: 'go', content: goFile },
        'internal/server/server.go': { kind: 'code', lang: 'go', content: 'package server\n\nimport "net/http"\n\nfunc New() *http.ServeMux {\n\treturn http.NewServeMux()\n}\n' },
        'internal/server/static/index.html': { kind: 'code', lang: 'html', content: '<!DOCTYPE html>\n<html lang="en">\n<head><title>app</title></head>\n<body></body>\n</html>\n' },
        'docs/architecture.md': { kind: 'markdown', content: `# Architecture\n\n## Layers\n\n| Layer | Package |\n|---|---|\n| CLI | \`cmd/\` |\n| Server | \`internal/server\` |\n\n> [!INFO]\n> The server never prompts.\n` },
        'assets/logo.svg': { kind: 'image', url: '/static/icons/logo.svg', size: 73407 },
        'docs/old-ai-files.zip': { kind: 'binary', size: 284113 },
        '.github/workflows/release.yaml': { kind: 'code', lang: 'yaml', content: 'name: release\non:\n  push:\n    branches: [main]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n' },
        'go.mod': { kind: 'code', lang: 'go', content: `module github.com/example/${p.name}\n\ngo 1.27\n\nrequire github.com/spf13/cobra v1.10.2\n` },
        'Makefile': { kind: 'code', lang: 'makefile', content: 'build:\n\t@go build -o app .\n' },
        'main.go': { kind: 'code', lang: 'go', content: 'package main\n\nfunc main() {\n\tcmd.Execute()\n}\n' },
    });

    let db;
    const settings = { latency: 'normal', failNext: false, accounts: 3, projects: 'normal' };

    const reset = () => {
        const projects = settings.projects === 'empty' ? [] : settings.projects === 'many' ? manyProjects() : baseProjects();
        db = {
            root: '~/Documents',
            accounts: accountsAll.slice(0, settings.accounts),
            projects,
            untracked: structuredClone(untracked),
            presets: structuredClone(presets),
            mcpLibrary: structuredClone(mcpLibrary),
            configureDefaults: structuredClone(configureDefaults),
            compliance: structuredClone(compliance),
            files: Object.fromEntries(projects.map((p) => [p.name, makeFiles(p)])),
        };
    };
    reset();

    const delay = (ms) => new Promise((r) => setTimeout(r, ms * { fast: 0.2, normal: 1, slow: 4 }[settings.latency]));

    const call = async (ms, fn) => {
        await delay(ms);
        if (settings.failNext) {
            settings.failNext = false;
            renderPanel();
            throw new Error('simulated server error (tmux exited with status 1)');
        }
        return structuredClone(fn());
    };

    const find = (name) => {
        const p = db.projects.find((x) => x.name === name);
        if (!p) throw new Error(`${name} is not tracked`);
        return p;
    };

    const tree = (name) => {
        const files = db.files[name] || {};
        const root = { name, path: '', type: 'dir', children: [] };
        for (const [path, f] of Object.entries(files)) {
            const parts = path.split('/');
            let node = root;
            parts.forEach((part, i) => {
                const sub = parts.slice(0, i + 1).join('/');
                const last = i === parts.length - 1;
                let child = node.children.find((c) => c.name === part);
                if (!child) {
                    child = last
                        ? { name: part, path: sub, type: f.kind === 'symlink' ? 'symlink' : 'file', kind: f.kind, target: f.target, size: f.size ?? (f.content || '').length }
                        : { name: part, path: sub, type: 'dir', children: [] };
                    node.children.push(child);
                }
                node = child;
            });
        }
        const sort = (n) => {
            if (!n.children) return;
            n.children.sort((a, b) => (a.type === 'dir') === (b.type === 'dir') ? a.name.localeCompare(b.name) : a.type === 'dir' ? -1 : 1);
            n.children.forEach(sort);
        };
        sort(root);
        return root;
    };

    const api = {
        state: () => call(250, () => ({
            root: db.root,
            accounts: db.accounts,
            projects: db.projects,
            presets: db.presets,
            mcpLibrary: db.mcpLibrary.map((m) => ({ name: m.name, transport: m.transport })),
        })),

        directories: () => call(200, () => db.untracked),

        repos: () => call(1100, () => repos),

        validatePreset: (name) => call(600, () => {
            const known = [...db.presets.builtin.map((p) => p.name), ...db.presets.remote.map((p) => p.name)];
            if (known.includes(name)) return { ok: true, name };
            if (/^[a-z0-9-]+\/[a-z0-9-]+$/.test(name) && !name.includes('missing')) return { ok: true, name, note: 'Found, preset.yaml parsed' };
            return { ok: false, name, error: name.includes('/') ? 'No preset.yaml at that path' : 'Use <owner-repo>/<preset>, or a preset already tracked' };
        }),

        start: (body) => call(1400, () => {
            let p = db.projects.find((x) => x.name === body.name);
            if (!p) {
                if (body.source === 'clone' && db.projects.some((x) => x.name === body.name)) throw new Error(`${body.name} already exists`);
                p = project(body.name);
                db.projects.unshift(p);
                db.files[p.name] = makeFiles(p);
                db.untracked = db.untracked.filter((u) => u.name !== body.name);
            }
            Object.assign(p, {
                rcName: body.rcName || p.name,
                account: body.account || p.account,
                presets: body.presets || p.presets,
                mcp: body.mcp || p.mcp,
                connectors: body.connectors ?? p.connectors,
                active: true,
                startedAt: Date.now(),
                lastActive: Date.now(),
            });
            return { name: p.name, mode: body.mode };
        }),

        stop: (name) => call(500, () => {
            const p = find(name);
            Object.assign(p, { active: false, startedAt: null, lastActive: Date.now() });
            return { name };
        }),

        restart: (name) => call(1200, () => {
            const p = find(name);
            Object.assign(p, { active: true, startedAt: Date.now() });
            return { name };
        }),

        switchAccount: (name, account) => call(1500, () => {
            const p = find(name);
            Object.assign(p, { account, active: true, startedAt: Date.now() });
            return { name, account };
        }),

        untrack: (name) => call(300, () => {
            const p = find(name);
            if (p.active) throw new Error('stop the session before untracking');
            db.projects = db.projects.filter((x) => x.name !== name);
            db.untracked.unshift({ name, foreign: [] });
            return { name };
        }),

        cleanCwd: (name) => call(700, () => {
            const p = find(name);
            p.stash = true;
            const files = db.files[name];
            for (const k of Object.keys(files)) if (k.startsWith('.agents/') || k === '.claude/skills' || k === 'AGENTS.md' || k === 'CLAUDE.md') delete files[k];
            return { name };
        }),

        restoreResume: (name) => call(400, () => {
            const p = find(name);
            if (!p.stash) throw new Error('no stashed resume file');
            p.stash = false;
            db.files[name]['.agents/session-resume/resume.md'] = { kind: 'markdown', content: resumeMd };
            return { name };
        }),

        applyPresets: (name, list) => call(1000, () => {
            const p = find(name);
            p.presets = list;
            Object.assign(db.files[name], { 'AGENTS.md': { kind: 'markdown', content: agentsMd }, 'CLAUDE.md': { kind: 'symlink', target: 'AGENTS.md' }, '.claude/skills': { kind: 'symlink', target: '../.agents/skills' } });
            return { name, presets: list };
        }),

        setMcp: (name, servers) => call(600, () => {
            const p = find(name);
            p.mcp = servers.map((x) => (typeof x === 'string' ? x : x.name));
            db.files[name]['.mcp.json'] = { kind: 'code', lang: 'json', content: mcpJson(p) };
            return { name, mcp: p.mcp };
        }),

        rawUrl: (name, path) => db.files[name]?.[path]?.url || '',

        tree: (name) => call(200, () => tree(name)),

        file: (name, path) => call(180, () => {
            const f = db.files[name]?.[path];
            if (!f) throw new Error(`${path} not found`);
            return { path, ...f };
        }),

        createFile: (name, path, content) => call(400, () => {
            if (db.files[name][path]) throw new Error(`${path} already exists`);
            const ext = path.split('.').pop();
            db.files[name][path] = ext === 'md' ? { kind: 'markdown', content } : { kind: 'code', lang: ext, content };
            return { path };
        }),

        upload: (name, folder, files) => call(900, () => {
            for (const f of files) {
                const path = folder ? `${folder}/${f.name}` : f.name;
                db.files[name][path] = f.type.startsWith('image/') ? { kind: 'image', url: URL.createObjectURL(f), size: f.size } : { kind: 'binary', size: f.size };
            }
            return { count: files.length };
        }),

        settings: () => call(300, () => ({
            root: db.root,
            accounts: db.accounts.map((a) => ({ ...a, ...db.compliance[a.dir], projects: db.projects.filter((p) => p.account === a.dir).length })),
            configureDefaults: db.configureDefaults,
            presets: db.presets,
            mcpLibrary: db.mcpLibrary,
        })),

        applyDefaults: (dir) => call(800, () => {
            db.compliance[dir] = { settingsFound: true, drift: [] };
            return { dir };
        }),

        addRemotePreset: (repo, path) => call(1600, () => {
            if (!/^[\w.-]+\/[\w.-]+$/.test(repo)) throw new Error('repository must look like owner/repo');
            const slug = repo.toLowerCase().replace('/', '-');
            const name = `${slug}/${path || 'default'}`;
            if (db.presets.remote.some((p) => p.name === name)) throw new Error(`${name} is already tracked`);
            db.presets.remote.push({ name, repo, description: 'Pulled from GitHub', updatedAt: Date.now() });
            return { name };
        }),

        updateRemotePreset: (name) => call(1200, () => {
            const p = db.presets.remote.find((x) => x.name === name);
            p.updatedAt = Date.now();
            return { name };
        }),

        removeRemotePreset: (name) => call(300, () => {
            db.presets.remote = db.presets.remote.filter((x) => x.name !== name);
            return { name };
        }),

        saveMcp: (entry, previousName) => call(400, () => {
            const i = db.mcpLibrary.findIndex((m) => m.name === (previousName || entry.name));
            if (i >= 0) db.mcpLibrary[i] = entry;
            else {
                if (db.mcpLibrary.some((m) => m.name === entry.name)) throw new Error(`${entry.name} already exists`);
                db.mcpLibrary.push(entry);
            }
            return { name: entry.name };
        }),

        deleteMcp: (name) => call(300, () => {
            db.mcpLibrary = db.mcpLibrary.filter((m) => m.name !== name);
            return { name };
        }),
    };

    const panelId = 'mock-panel';

    const renderPanel = () => {
        let el = document.getElementById(panelId);
        if (!el) {
            el = document.createElement('div');
            el.id = panelId;
            el.className = 'fixed bottom-3 left-3 z-40 md:bottom-5 md:left-5';
            document.getElementById('layer-root').appendChild(el);
            el.addEventListener('ui-change', (e) => {
                const { name, value } = e.detail;
                if (name === 'mock-fail') settings.failNext = value;
                if (name === 'mock-latency') settings.latency = value;
                if (name === 'mock-accounts') { settings.accounts = Number(value); reset(); app.refresh(); app.route(); }
                if (name === 'mock-projects') { settings.projects = value; reset(); app.refresh(); app.route(); }
            });
            ui.on('mock-toggle', () => { el.dataset.open = el.dataset.open === 'true' ? 'false' : 'true'; renderPanel(); });
            ui.on('mock-exit', () => {
                const active = db.projects.filter((p) => p.active);
                if (!active.length) return ui.toast('No active session to end', { icon: 'info' });
                const p = active[Math.floor(Math.random() * active.length)];
                Object.assign(p, { active: false, startedAt: null, lastActive: Date.now() });
                ui.toast(`${p.name} ran /exit elsewhere. Refresh or refocus the tab to see it.`, { icon: 'terminal', timeout: 5000 });
            });
            ui.on('mock-reset', () => { reset(); app.refresh(); app.route(); });
        }
        const open = el.dataset.open === 'true';
        el.innerHTML = open ? `
            <section class="w-80 max-w-[calc(100vw-1.5rem)] rounded-3xl bg-mantle p-4 shadow-2xl shadow-crust ring-1 ring-surface0 [animation:pop-in_.12s_ease-out]">
                <header class="mb-3 flex items-center gap-2">
                    <i data-lucide="flask-conical" class="size-4 text-peach"></i>
                    <h3 class="flex-1 font-display text-sm font-medium text-text">Mock scenarios</h3>
                    ${ui.iconButton('x', { title: 'Close', action: 'mock-toggle', size: 'sm' })}
                </header>
                <p class="${ui.cls.label} mb-1.5 px-1">Accounts discovered</p>
                ${ui.segmented({ name: 'mock-accounts', value: String(settings.accounts), full: true, options: [{ value: '1', label: 'One' }, { value: '2', label: 'Two' }, { value: '3', label: 'Three' }] })}
                <p class="${ui.cls.label} mt-3 mb-1.5 px-1">Tracked projects</p>
                ${ui.segmented({ name: 'mock-projects', value: settings.projects, full: true, options: [{ value: 'empty', label: 'None' }, { value: 'normal', label: 'Some' }, { value: 'many', label: 'Many' }] })}
                <p class="${ui.cls.label} mt-3 mb-1.5 px-1">Server latency</p>
                ${ui.segmented({ name: 'mock-latency', value: settings.latency, full: true, options: [{ value: 'fast', label: 'Fast' }, { value: 'normal', label: 'Normal' }, { value: 'slow', label: 'Slow' }] })}
                <div class="mt-3 -mx-1">${ui.switch({ name: 'mock-fail', label: 'Fail the next action', hint: 'The next API call returns an error', checked: settings.failNext })}</div>
                <div class="mt-2 flex flex-wrap gap-2">
                    ${ui.button('/exit a session elsewhere', { icon: 'terminal', size: 'sm', action: 'mock-exit' })}
                    ${ui.button('Reset data', { icon: 'rotate-ccw', size: 'sm', variant: 'ghost', action: 'mock-reset' })}
                </div>
            </section>` : `
            <button type="button" data-action="mock-toggle" class="flex items-center gap-2 rounded-full bg-peach/15 px-3.5 py-2 text-xs font-medium text-peach shadow-lg shadow-crust/60 backdrop-blur transition-colors hover:bg-peach/25">
                <i data-lucide="flask-conical" class="size-3.5"></i><span class="max-md:hidden">Mock</span>${settings.failNext ? '<span class="size-1.5 rounded-full bg-red"></span>' : ''}
            </button>`;
        ui.icons(el);
    };

    document.addEventListener('DOMContentLoaded', renderPanel);

    return { api };
})();

const api = mock.api;
