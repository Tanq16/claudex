---
name: develop
description: Entry point for ANY coding work in a project that has these skills installed — implementing a feature, changing or refactoring code, fixing a bug, writing tests, scaffolding something new, or touching build/CI. Selects and loads the development skills that govern the task up front, holds the work to them while coding, and ends with a quick self-review that the skills were actually followed. Use this whenever you are about to develop anything.
user-invocable: true
---

# Develop

**The front door for development work: figure out the task, load the skills that govern it, follow them while coding, then verify you did.**

The other skills carry the conventions; they only help if the right ones are in context *before* you write code and are still honored *after*. This skill makes that deterministic — it is a pre-step you run first, and a quick self-check you run last.

## When to Use

Use this skill **as the first step** of any coding task in a project that has these skills installed:

- A new feature, a refactor, a bug fix, new or changed tests
- A new project or a new component scaffolded from scratch
- A build, CI/CD, Makefile, README, or config change

Run it *before* writing code — its whole job is to make sure the governing skills are loaded and applied, not to fix things afterward.

**Not for:**
- A pure question with no code change.
- A deliberate, thorough re-audit of an existing codebase — use `review-code` (heavyweight, multi-agent). `develop` only self-reviews the diff you just wrote; see "Relationship to review-code" below.

## Workflow

### Step 1: Frame the task

State in one line what is being built or changed. Then classify:

- **Project type** — per the `go-foundations` taxonomy: CLI Only / Web Only / CLI + Web / Headless API Service / Library / Chrome Extension, or **Node Web Only** (a Node server process that serves a frontend). Infer it from the tree (`go.mod`, `cmd/`, `internal/server/static/`, `manifest.json`, `package.json` with `"type":"module"` + `public/`, presence/absence of `utils/`, …).
- **Work type** — new project, feature, refactor, bug fix, tests, infra/CI, docs.

### Step 2: Select and READ the governing skills

Use the **Skill Map** below to pick the skills this task touches. Then actually **read** each selected `SKILL.md` (and note which reference files will matter for this task) — do not just name them. `go-foundations` is always in scope for any Go work.

Pull in the skills the task *actually* touches — not all of them. When in doubt, load `go-foundations` plus the one domain skill that matches the files you are editing.

### Step 3: State the rules in effect

Before writing code, emit a short checklist — the **specific** rules from the loaded skills that apply to *this* task, not a restatement of whole skills. A handful of concrete bullets, e.g.:

```
Rules in effect (CLI Only, new command):
- Comments: default none; keep one only if its *why* is load-bearing, judged on its own merit — why-not-what, one line, never restate code or embed scaffolding
- Output via utils printer (PrintInfo/PrintSuccess/...), never fmt.Println
- zerolog only behind --debug; human output through utils otherwise
- Tables via utils.PrintTable; honor --for-ai plain-text path
- New command file under cmd/, registered in root.go init()
```

A few rules are **always in effect** for any Go work regardless of task — first among them comment discipline (`go-foundations` → *Comments and Code Style*). Put these on the checklist every time: being cross-cutting rather than task-specific, they are the first to drop off the list and the first to decay mid-session. This keeps the rules in front of you while you code — it is the main defense against rules decaying out of attention mid-session.

### Step 4: Do the work

Implement, holding to the Step 3 checklist. When the task spans several skills, keep each skill's rules in view for the part it governs (e.g. `go-concurrency` for the worker pool, `go-cli` for the command wiring).

For a large task, brief any sub-agents you spawn with the relevant skill(s) and their Step 3 rules so they inherit the same constraints.

### Step 5: Quick self-review (inline)

After the change is done, do a **fast** pass over what you changed *this session* and check it against the rules you listed in Step 3. Fix gaps inline. This is a lightweight self-check, not a full audit:

- **Scope it to the files you touched this session** — not the whole project.
- **Check the Step 3 checklist**, not every rule in every skill.
- **Fix small deviations directly**; call out anything larger that needs a decision.
- No sub-agents, no domain orchestration — keep it quick.

Escalate to `review-code` instead when: you are re-engaging a project you have not touched in a while, the skills may have changed since the code was written, or you want a thorough multi-agent domain audit.

## Skill Map

| Task touches… | Load |
|---|---|
| Any Go code | `go-foundations` (always) |
| Cobra commands, subcommands, flags, CLI output lifecycle, TUI | `go-cli` |
| HTTP servers, internal package architecture, storage, OAuth/auth | `go-backend` |
| Embedded SPA frontend (`embed.FS`, Tailwind, Catppuccin, PWA) | `go-frontend` |
| Goroutines, concurrent pipelines, fan-out/fan-in, progress/resume | `go-concurrency` |
| Any Node code (Web Only: a server process that serves a frontend) | `node-foundations` (always, for Node) |
| Node HTTP/WebSocket server, routing, auth, JSON state | `node-backend` |
| Vanilla-JS SPA frontend (Catppuccin, vendored assets, WebSocket client) | `node-frontend` |
| Makefile, GitHub Actions, Docker, releases, versioning | `project-ci-cd` |
| README | `project-readme` |
| Chrome extension (manifest, popup, content/background scripts) | `chrome-extension-basics` |
| Persisted HTML deliverable doc (architecture/design/research/analysis) | `ai-docs` |

For which skills a whole project type typically pulls in, defer to the `go-foundations` taxonomy and layout — it is the source of truth for project structure.

## Relationship to review-code

These are deliberately different tools:

- **`develop`** — lightweight, inline, *during active coding*. Confirms the change you just made follows the skills you loaded for it. Fast, scoped to your diff, no sub-agents.
- **`review-code`** — heavyweight, separate, multi-agent domain audit of an *existing codebase*. For re-engaging a project, or a deliberate compliance pass when skills may have changed.

Use `develop` every time you code. Reach for `review-code` when you specifically want the thorough audit.

## Key Principles

- **Skills first, code second** — load and read the governing skills in Steps 1–2 before writing, not after.
- **Skills are the single source of truth** — only follow (and only self-flag) what a loaded skill defines. If no skill covers it, it is not a rule.
- **Keep the checklist concrete** — Step 3 lists the specific rules for *this* task, not whole skills restated.
- **Always close with the self-review** — Step 5 is cheap and catches the drift that creeps in mid-session.
- **Don't over-load** — pull in the skills the task actually touches, not the whole set.
