---
name: develop
description: The entry point for any coding work in a project that has these skills installed - implementing a feature, changing or refactoring code, fixing a bug, writing tests, scaffolding something new, or touching build and CI. Selects and loads the skills that govern the task before any code is written, holds the work to them while coding, and ends with a self-review of the diff against them. Use this first whenever you are about to develop anything. Not for a pure question with no code change, and not for a full audit of an existing codebase.
user-invocable: true
---

# Develop

**Work out what the task is, load the skills that govern it, follow them while coding, then check that you did.**

The other skills carry the conventions and only help when the right ones are in context before the first line is written and still honored after the last. This makes that deterministic rather than incidental.

## When to Use

Run this as the first step of any coding task: a feature, a refactor, a bug fix, new or changed tests, a new project, a build or CI change, a README.

Skip it for a question with no code change. A deliberate re-audit of an existing codebase is a different job with its own multi-agent orchestration, and this only reviews the diff you just wrote.

## Step 1: Frame the task

State in one line what is being built or changed, then classify it.

**Project type**, read off the tree: CLI Only, Web Only, CLI + Web, Headless API Service, Library, Node Web Only, or Chrome Extension. `go.mod` with `cmd/` and `utils/` and no `internal/server/` is CLI Only; `internal/server/static/` without `utils/` is Web Only; both together is the hybrid; `package.json` with `"type":"module"` plus `public/` and `src/` is Node Web Only; `manifest.json` with `manifest_version` is an extension.

**Work type**: new project, feature, refactor, bug fix, tests, infrastructure, docs.

## Step 2: Select and read the governing skills

Pick from the Skill Map below, then read each selected `SKILL.md` in full. Naming a skill is not reading it, and a convention you half-remember is the one that produces a plausible file nobody wants.

Load the skills the task actually touches rather than the whole set. When in doubt, take the two always-in-scope skills for the language plus the one that matches the files being edited.

A delegated sub-task is briefed with the same skills, so a subagent inherits the constraints rather than reinventing them.

## Step 3: State the rules in effect

Before writing code, emit a short checklist of the specific rules from the loaded skills that apply to this task. Concrete bullets, not whole skills restated.

```
Rules in effect (CLI Only, new command):
- Comments: default none; keep one only where its why is load-bearing
- Output through the utils printers, never fmt.Println
- zerolog only behind --debug; utils printers otherwise
- Tables via utils.PrintTable, honoring the --for-ai markdown path
- New command file under cmd/, registered in root.go init()
- Flags grouped in a per-command struct, registered in init()
```

The cross-cutting rules go on the list every time, comment discipline first among them. Being cross-cutting rather than task-specific, they are the first to fall off the list and the first to decay mid-session, and the written checklist is the defense against that.

## Step 4: Do the work

Implement, holding to the Step 3 checklist. When a task spans several skills, keep each one's rules in view for the part it governs: the concurrency skill for the worker pool, the command skill for the wiring around it.

## Step 5: Self-review the diff

Pass over what you changed this session and check it against the Step 3 checklist. Fix small deviations directly and call out anything larger that needs a decision.

Scope it to the files you touched, check the checklist rather than every rule in every skill, and keep it quick. This catches the drift that creeps in mid-session, which is most of what a review of fresh work finds.

## Skill Map

| Task touches | Load |
|---|---|
| Any Go code | `go-project-layout`, `go-idioms` |
| Tests, in any language | `unit-testing` |
| Cobra root, commands, subcommands, flags | `go-cli-commands` |
| Printing, tables, `--debug` and `--for-ai`, terminal colors | `go-cli-output` |
| Interactive prompts, passwords, selection lists | `go-cli-prompts` |
| Running/done progress, phases, progress bars | `go-cli-progress` |
| `internal/` package structure, error boundaries, storage | `go-package-architecture` |
| `net/http` server, embedded static serving, middleware | `go-http-server` |
| OAuth login for a CLI client | `go-oauth-cli` |
| Goroutines, errgroup, semaphores, fan-out/fan-in | `go-concurrency` |
| A multi-job pipeline with progress and resume | `go-job-pipeline` |
| The embedded SPA under `internal/server/static/` | `go-embedded-frontend` |
| Rendering Markdown in a browser page | `web-markdown-rendering` |
| Mermaid diagrams in a browser page | `web-mermaid-diagrams` |
| Any Node code | `node-project-layout`, `node-idioms` |
| `config.json`, deep merge, `state.json`, the session secret | `node-config-state` |
| The `node:http` and `ws` server, routing, static serving | `node-http-ws-server` |
| Password and session-cookie auth in Node | `node-auth` |
| The vanilla-JS SPA under `public/` | `node-frontend` |
| A Go or Chrome extension Makefile | `go-makefile` |
| A Node Makefile, vendoring, the native addon build | `node-makefile` |
| What a Node project ships: binary or tarball | `node-release-artifacts` |
| A Dockerfile or docker-compose, in any language | `dockerize` |
| `.github/workflows/release.yaml`, version bumps | `github-release-workflow` |
| README | `project-readme` |
| Chrome extension | `chrome-extension` |

A whole project type pulls in a predictable set. A CLI Only tool takes the two Go skills plus `go-cli-commands` and `go-cli-output`, and adds the others as the surface grows. A Web Only service takes the two Go skills plus `go-http-server`, `go-package-architecture`, and `go-embedded-frontend`.

## Principles

Skills are the source of truth for what a rule is. Follow, and self-flag against, only what a loaded skill defines; anything no skill covers is not a rule and does not belong on the checklist.

Load and read before writing, not after. A convention applied retroactively is a second diff on top of the first.
