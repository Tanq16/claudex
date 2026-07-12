import { readdir, readFile, writeFile, stat } from 'node:fs/promises';
import { join, extname, isAbsolute, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

// Docs root is decoupled from the engine: --docs <dir> > DOCS_DIR env > ./AI-docs (cwd).
function argValue(name) {
  const a = process.argv.slice(2);
  const i = a.indexOf(name);
  if (i !== -1 && a[i + 1]) return a[i + 1];
  const eq = a.find(s => s.startsWith(`${name}=`));
  return eq ? eq.slice(name.length + 1) : null;
}
const docsArg = argValue('--docs') ?? process.env.DOCS_DIR ?? 'AI-docs';
const DOCS_ROOT = isAbsolute(docsArg) ? docsArg : resolve(process.cwd(), docsArg);

const EXCLUDE_DIRS = new Set(['node_modules', '.git']);
const EXCLUDE_FILES = new Set(['CLAUDE.md']);

// `marked` self-installs on first use, so `node convert.mjs` works with no prior setup.
function installDeps() {
  return new Promise((resolve, reject) => {
    const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
    const proc = spawn(npm, ['install', '--silent', '--no-audit', '--no-fund'], {
      cwd: __dirname,
      stdio: 'inherit',
    });
    proc.on('error', reject);
    proc.on('exit', code => (code === 0 ? resolve() : reject(new Error(`npm install failed (${code})`))));
  });
}
let mod;
try {
  mod = await import('marked');
} catch {
  console.log('ai-docs: installing marked (one-time) ...');
  await installDeps();
  mod = await import('marked');
}
const { marked } = mod;
marked.setOptions({ gfm: true, breaks: false, headerIds: false, mangle: false });

async function walk(dir) {
  const out = [];
  const entries = await readdir(dir, { withFileTypes: true });
  for (const e of entries) {
    if (e.name.startsWith('.')) continue;
    const full = join(dir, e.name);
    if (e.isDirectory()) {
      if (EXCLUDE_DIRS.has(e.name)) continue;
      out.push(...await walk(full));
    } else if (extname(e.name).toLowerCase() === '.md' && !EXCLUDE_FILES.has(e.name)) {
      out.push(full);
    }
  }
  return out;
}

const force = process.argv.includes('--force');
const CURATED_MARKER = '<!-- curated -->';

async function isCurated(path) {
  try {
    const head = (await readFile(path, 'utf8')).slice(0, 200);
    return head.trimStart().startsWith(CURATED_MARKER);
  } catch { return false; }
}

const files = await walk(DOCS_ROOT);
let converted = 0, skipped = 0, curated = 0;
for (const md of files) {
  const html = md.replace(/\.md$/i, '.html');
  if (!force && await isCurated(html)) { curated++; continue; }
  if (!force) {
    try {
      const [a, b] = await Promise.all([stat(md), stat(html)]);
      if (b.mtimeMs >= a.mtimeMs) { skipped++; continue; }
    } catch {}
  }
  const src = await readFile(md, 'utf8');
  const out = marked.parse(src);
  await writeFile(html, out);
  console.log(`  ${md.replace(DOCS_ROOT, '.')}  ->  ${html.replace(DOCS_ROOT, '.')}`);
  converted++;
}
console.log(`\nconverted: ${converted}, up-to-date: ${skipped}, curated: ${curated}`);
