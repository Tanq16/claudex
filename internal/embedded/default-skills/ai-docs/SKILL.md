---
name: ai-docs
description: Use when the user says "AI-docs" / "put this in AI-docs", or explicitly asks to capture a structured deliverable — architecture, design, analysis, research, planning, or comparison results — as a persisted document. Writes a curated HTML doc into a docs directory (default AI-docs/, or any directory the user names) served by a local viewer. NOT for ordinary scratch/markdown file writes or quick notes unless the user asks for an AI-doc.
user-invocable: true
---

# AI-docs

**Capture durable artifacts as curated HTML in a docs directory, viewed through a local server. Governs how output is represented, not what the task produces.**

## When to Use

Activate this skill when the user wants something *captured* (not just discussed):

- The user says **"AI-docs"**, "put this in AI-docs", "write a doc", "save/store this", "document this".
- A plan, architecture decision, analysis, research finding, or comparison is *ready* to be recorded.
- Any artifact that the user will re-read later and benefits from structure + a readable viewer.

**`AI-docs` is the default trigger word and the default directory name.** It is not a fixed location: if the user names another directory (e.g. "put the security review in `security-docs/`"), write there instead. The same methodology serves any number of directories at once — different folders for different kinds of work, each viewable by pointing the engine at it. Treat the directory as a parameter, not a constant; the only hardcoded default (`AI-docs/`) lives in the engine, not in your behavior.

### When NOT to use

- Exploration, questions, half-formed thoughts, quick clarifications → stay in chat.
- The agent's reasoning, decisions, and back-and-forth → stay in chat. The user reads those there.
- Don't pre-emptively dump intermediate notes into the docs directory. Capture when something is *ready*.

## Start here — required reading

Read the **Always** files now, in full, before writing any doc — they define the representation rules you'll be held to. Read the **When** file before the sub-task it names; a subagent may read it if you delegate that work.

**Always:**
- `./references/writing-style.md` — prose + structure rules (representation, not substance)
- `./references/document-template.md` — the default doc spine + worked example
- `./references/component-vocabulary.md` — the only HTML classes/components the viewer supports

**When the doc has any diagram, flow, or chart:**
- `./references/diagrams.md` — the house SVG style + copyable worked example

## Core principle

This skill changes **representation, not substance.** The information the agent would produce for a task is unchanged — it is reorganized into a light structure, phrased tighter, and rendered as HTML so it is easy to consume long-term. Nothing is dropped: depth and exhaust move into a collapsible `Raw material` section, they are not deleted. See `./references/writing-style.md`.

## Workflow

### Step 1: Write the doc

Pick the target directory (default `AI-docs/`, or whatever the user named) and create a `.html` file under it — `mkdir -p` the directory if it does not exist yet. There is nothing to scaffold: the engine lives in this skill and is pointed at the directory at run time. Create subfolders freely (the server walks the tree every request, so new files/folders appear on browser refresh). There is **no required directory layout**; organize by what makes sense for the work.

Author a **body fragment only** (no `<html>`/`<head>`/`<body>`/`<style>`/`<script>`). The first line MUST be `<!-- curated -->` so the converter never overwrites it.

Follow the default spine — **Header + lede → Details → Process → optional Raw material** — from `./references/document-template.md`. It's a starting shape, not a cage. Use the component classes from `./references/component-vocabulary.md` and the prose rules from `./references/writing-style.md`.

### Step 2: Report

Drop the absolute file path into chat in one line. No summary, no preamble — the user reads it in the viewer. Don't restate a doc you just wrote.

**The user starts and runs the server themselves — never do it for them.** Don't start, stop, restart, or check whether it's running, and don't worry about ports. If this is the first doc in a directory, give the one-line start command so the user can open the viewer. Substitute the absolute path to *this skill's* `assets/server.mjs`, and `--docs` only when the directory is not the default `AI-docs/`:

```bash
node <skill>/assets/server.mjs                          # serves ./AI-docs on http://127.0.0.1:4321
node <skill>/assets/server.mjs --docs security-docs     # serve a different directory
node <skill>/assets/server.mjs --docs security-docs --port 4322   # second dir, second viewer
```

Run from the project root; `--docs` resolves relative to the current directory. Two directories means two `node ... --docs <dir> --port <n>` processes — one viewer each.

## Authoring rules (quick reference)

- First line of every doc: `<!-- curated -->`.
- Body fragments only — the viewer provides all chrome (palette, headings, TOC, PDF).
- Use only the classes in `./references/component-vocabulary.md`. No new colors/callouts/emoji.
- For any topology, flow, sequence, or quantitative picture, hand-author an SVG in the house style from `./references/diagrams.md` — **pick the representation first** (topology vs swimlane vs bar chart vs just a table), then follow the canvas + `<g><title>` tooltips + semantic colors. No Mermaid/ASCII/images.
- A right-hand TOC auto-builds from `<h2>`/`<h3>`. Lean on headings for long docs.
- **Write curated HTML — the hub serves `.html` only.** A loose `.md` will not appear in the viewer. Markdown is an on-ramp, not a doc format: if the user asks to convert/move existing markdown into the hub, run the converter (see "How it works") to turn it into `.html`; otherwise author `.html` directly.
- Don't write README / INDEX / summary docs unless asked. Don't restructure the tree without an ask.

## How it works

The viewer machinery (`server.mjs`, `convert.mjs`, `index.html`, `print.html`, `package.json`) lives in this skill's `assets/` and is never copied into the project. The user runs the server and points it at a docs directory: `node <skill>/assets/server.mjs [--docs <dir>] [--port <n>]`. With no `--docs` it serves `./AI-docs` (created if missing) on port 4321. It serves the `.html` docs in that directory; the browser **live-reloads automatically** when files change — no manual refresh. The "PDF" button renders via headless Chrome. The viewer loads Tailwind / fonts / highlight.js from CDNs, so the browser needs internet. **The server itself has no dependencies** and runs with just `node`, nothing else. The viewer also injects copy controls on load: every code block copies its text, and every diagram copies as SVG source or as a rendered PNG (dark background baked in) — authors write plain `<pre>`/`<svg>`. The file-tree sidebar and the right-hand TOC are both collapsible from the top bar, and the choice persists.

**Markdown is handled out-of-band, on request only — the server never renders `.md`.** When the user asks to convert or move existing markdown into the hub, run `node <skill>/assets/convert.mjs --docs <dir>` (default `AI-docs/`). It bulk-converts untouched `.md` → sibling `.html` and leaves curated files (those with the `<!-- curated -->` marker) untouched. `convert.mjs` self-installs its one dependency (`marked`) into `assets/` on first run, so no setup is needed.

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/document-template.md` | Template | The default doc spine + worked example |
| `./references/component-vocabulary.md` | Patterns | The exact HTML components/classes the viewer supports |
| `./references/diagrams.md` | Patterns | The house SVG diagram style + copyable worked example |
| `./references/writing-style.md` | Patterns | Prose + structure rules (representation, not substance) |

### Using references

- **document-template**: read before writing a doc; copy the skeleton and fill it in.
- **component-vocabulary**: the source of truth for every allowed class. Don't invent components.
- **diagrams**: read before authoring any architecture/flow diagram; copy the skeleton and recolor it.
- **writing-style**: how to phrase and what to push into `Raw material`.

## Assets (the fixed machinery — do not modify, do not copy)

`./assets/` ships the viewer and server. Point the engine at a docs directory with `--docs`; do not copy it into the project. The color scheme, layout, TOC, and PDF export are fixed; do not change them.
