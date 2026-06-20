import { createServer } from 'node:http';
import { readFile, readdir, mkdtemp, unlink, access, rmdir, mkdir } from 'node:fs/promises';
import { watch } from 'node:fs';
import { extname, join, isAbsolute, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';
import { tmpdir } from 'node:os';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

// Docs root is decoupled from the engine: --docs <dir> > DOCS_DIR env > ./AI-docs (cwd).
// One engine can serve any directory, and any number of them via separate processes.
function argValue(name) {
  const a = process.argv.slice(2);
  const i = a.indexOf(name);
  if (i !== -1 && a[i + 1]) return a[i + 1];
  const eq = a.find(s => s.startsWith(`${name}=`));
  return eq ? eq.slice(name.length + 1) : null;
}
const docsArg = argValue('--docs') ?? process.env.DOCS_DIR ?? 'AI-docs';
const DOCS_ROOT = isAbsolute(docsArg) ? docsArg : resolve(process.cwd(), docsArg);
await mkdir(DOCS_ROOT, { recursive: true });

// Port precedence: --port flag > PORT env > default 4321.
const PORT = Number(argValue('--port') ?? process.env.PORT ?? 4321);
const HOST = process.env.HOST ?? '127.0.0.1';

const EXCLUDE_NAMES = new Set([
  'server.mjs',
  'convert.mjs',
  'index.html',
  'print.html',
  'package.json',
  'package-lock.json',
  'node_modules',
  'CLAUDE.md',
  '.DS_Store',
]);

// The hub serves curated HTML only. Markdown is handled out-of-band by convert.mjs.
const TYPE_BY_EXT = {
  '.html': 'html',
};

const CHROME_PATHS = [
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
  '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
  '/usr/bin/google-chrome',
  '/usr/bin/chromium',
  '/usr/bin/chromium-browser',
];

let printTemplate = await readFile(join(__dirname, 'print.html'), 'utf8');

// Live reload: SSE clients + a debounced recursive watch over the docs tree.
const sseClients = new Set();
function notifyReload() {
  for (const res of sseClients) {
    try { res.write('event: change\ndata: 1\n\n'); } catch {}
  }
}

async function walk(dir, base = '') {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = new Map();
  const dirs = [];
  for (const e of entries) {
    if (e.name.startsWith('.')) continue;
    if (e.name.startsWith('_')) continue;
    if (EXCLUDE_NAMES.has(e.name)) continue;
    const rel = base ? `${base}/${e.name}` : e.name;
    const full = join(dir, e.name);
    if (e.isDirectory()) {
      const children = await walk(full, rel);
      if (children.length) dirs.push({ name: e.name, path: rel, kind: 'dir', children });
      continue;
    }
    const ext = extname(e.name).toLowerCase();
    if (!TYPE_BY_EXT[ext]) continue;
    const basename = e.name.replace(/\.html$/i, '');
    files.set(basename, { name: basename, file: e.name, path: rel, type: TYPE_BY_EXT[ext] });
  }
  const items = [];
  for (const [, file] of files) {
    items.push({ ...file, kind: 'file' });
  }
  items.push(...dirs);
  return items.sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === 'file' ? -1 : 1;
    return a.path.localeCompare(b.path, undefined, { numeric: true });
  });
}

function send(res, status, body, type = 'text/plain; charset=utf-8', extra = {}) {
  res.writeHead(status, { 'Content-Type': type, 'Cache-Control': 'no-store', ...extra });
  res.end(body);
}

function safeResolve(rel) {
  if (!rel) return null;
  if (rel.includes('..') || rel.startsWith('/')) return null;
  return join(DOCS_ROOT, rel);
}

async function exists(p) {
  try { await access(p); return true; } catch { return false; }
}

async function findChrome() {
  for (const p of CHROME_PATHS) if (await exists(p)) return p;
  return null;
}

async function renderPrintHtml(rel) {
  const full = safeResolve(rel);
  if (!full) throw new Error('bad path');
  const body = await readFile(full, 'utf8');
  const title = rel.replace(/\.html$/i, '').replace(/^.*\//, '');
  return printTemplate.replace('__TITLE__', escapeHtml(title)).replace('__CONTENT__', body);
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

async function generatePdf(rel) {
  const chrome = await findChrome();
  if (!chrome) throw new Error('chrome not found (looked in /Applications and /usr/bin)');
  const dir = await mkdtemp(join(tmpdir(), 'docs-pdf-'));
  const out = join(dir, 'out.pdf');
  const url = `http://${HOST}:${PORT}/print/${encodeURI(rel)}`;
  await new Promise((resolve, reject) => {
    const proc = spawn(chrome, [
      '--headless=new',
      '--disable-gpu',
      '--no-pdf-header-footer',
      '--hide-scrollbars',
      '--run-all-compositor-stages-before-draw',
      '--virtual-time-budget=4000',
      `--print-to-pdf=${out}`,
      url,
    ], { stdio: ['ignore', 'pipe', 'pipe'] });
    let stderr = '';
    proc.stderr.on('data', d => { stderr += d.toString(); });
    proc.on('error', reject);
    proc.on('exit', code => code === 0 ? resolve() : reject(new Error(`chrome exit ${code}: ${stderr.slice(0, 400)}`)));
  });
  const pdf = await readFile(out);
  await unlink(out).catch(() => {});
  await rmdir(dir).catch(() => {});
  return pdf;
}

const server = createServer(async (req, res) => {
  try {
    const url = new URL(req.url, `http://${req.headers.host}`);
    if (req.method !== 'GET') return send(res, 405, 'method not allowed');

    if (url.pathname === '/' || url.pathname === '/index.html') {
      const html = await readFile(join(__dirname, 'index.html'), 'utf8');
      return send(res, 200, html, 'text/html; charset=utf-8');
    }

    if (url.pathname === '/api/events') {
      res.writeHead(200, {
        'Content-Type': 'text/event-stream; charset=utf-8',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      });
      res.write('retry: 2000\n\n');
      sseClients.add(res);
      req.on('close', () => sseClients.delete(res));
      return;
    }

    if (url.pathname === '/api/docs') {
      const tree = await walk(DOCS_ROOT);
      return send(res, 200, JSON.stringify(tree), 'application/json');
    }

    if (url.pathname.startsWith('/api/doc/')) {
      const rel = decodeURIComponent(url.pathname.slice('/api/doc/'.length));
      const full = safeResolve(rel);
      if (!full) return send(res, 400, 'bad path');
      const content = await readFile(full, 'utf8');
      return send(res, 200, content, 'text/plain; charset=utf-8');
    }

    if (url.pathname.startsWith('/print/')) {
      const rel = decodeURIComponent(url.pathname.slice('/print/'.length));
      if (process.env.PRINT_TEMPLATE_RELOAD === '1') {
        printTemplate = await readFile(join(__dirname, 'print.html'), 'utf8');
      }
      const html = await renderPrintHtml(rel);
      return send(res, 200, html, 'text/html; charset=utf-8');
    }

    if (url.pathname === '/api/pdf') {
      const rel = url.searchParams.get('path');
      const safe = safeResolve(rel ?? '');
      if (!safe) return send(res, 400, 'bad path');
      try {
        const pdf = await generatePdf(rel);
        const name = rel.replace(/\.html$/i, '').replace(/^.*\//, '');
        return send(res, 200, pdf, 'application/pdf', {
          'Content-Disposition': `inline; filename="${name}.pdf"`,
        });
      } catch (err) {
        return send(res, 500, `pdf failed: ${err.message}`);
      }
    }

    return send(res, 404, 'not found');
  } catch (err) {
    return send(res, 500, String(err?.message ?? err));
  }
});

server.listen(PORT, HOST, () => {
  console.log(`docs server: http://${HOST}:${PORT}  ->  ${DOCS_ROOT}`);
});

// Watch the docs tree and push a reload event to connected browsers on any change.
let reloadDebounce = null;
try {
  watch(DOCS_ROOT, { recursive: true }, (_event, filename) => {
    const name = filename ? String(filename) : '';
    if (name.includes('node_modules') || name.includes('.git')) return;
    clearTimeout(reloadDebounce);
    reloadDebounce = setTimeout(notifyReload, 120);
  });
} catch (err) {
  console.warn(`live-reload watch unavailable: ${err?.message ?? err}`);
}
