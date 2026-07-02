---
name: cross-ai
description: User-invoked dispatcher that hands a task to one or more external headless CLI coding agents (agy, gemini, codex, cursor) or Claude Code headless and relays their output. Invoked explicitly as `/cross-ai <agent> [task]`. Do NOT auto-trigger this from incidental mentions of these tools, and never use it for native Claude Code sub-agents (the Task tool).
user-invocable: true
---

# cross-ai

**Delegate the current task to an external CLI coding agent over bash, then relay its answer. You are the dispatcher: pick the command from the table, feed it the context, run it, report back. Nothing else.**

This skill is explicit-invoke only (`/cross-ai`). It is NOT triggered by phrases like "use agy" on their own, and it is NOT the native Task tool — "launch a sub-agent" means the Task tool, not this.

## Invocation

```
/cross-ai <agent> [task]
```

- `<agent>` is one of: `agy`, `gemini`, `codex`, `cursor`, `claude` (Claude Code headless).
- `[task]` is the work. If omitted, use the surrounding request as the task.
- Multiple agents in one request → run **each** on the same task, in parallel, and present results side by side labeled by agent. Two separate `/cross-ai agy` and `/cross-ai codex` invocations against the same ask are the same thing: launch both.

## Dispatch table (read-only defaults)

Run these verbatim, substituting `<TASK>` and `<FILE>`. Each is read-only — no edits, no auto-approve of writes.

| Agent | Command |
|-------|---------|
| `agy` | `agy -p "<TASK> @<FILE>" --model gemini-3.1-pro --dangerously-skip-permissions < /dev/null 2>&1` |
| `gemini` | `gemini -m gemini-3.1-pro-preview -p "<TASK> @<FILE>" --output-format json \| jq -r '.response'` |
| `codex` | `codex exec -m gpt-5.5 --skip-git-repo-check "<TASK>" 2>/dev/null` |
| `cursor` | `cursor-agent -p "<TASK>" --model gpt-5.5 --output-format json \| jq -r '.result'` |
| `claude` | `claude -p "<TASK>" --model claude-opus-4-8 --output-format json --allowedTools "Read,Grep,Glob,Bash" \| jq -r '.result'` |

`@<FILE>` injection is for `agy`/`gemini`. `codex`/`cursor`/`claude` read the current working directory themselves — reference paths in `<TASK>` and they will open them.

`agy` truncation guard: if its output looks cut off or garbled (it misbehaves on a non-TTY pipe), rerun under a pty and strip ANSI:
```
script -qec 'agy -p "<TASK> @<FILE>" --model gemini-3.1-pro --dangerously-skip-permissions' /dev/null < /dev/null | sed -r 's/\x1B\[[0-9;]*[A-Za-z]//g'
```

## Execution recipe

1. **Stage context the agent can't see on its own.** It starts blank — it does not inherit this conversation. If the task needs our analysis, a specific diff, or files outside its cwd, write that to a temp file first: `CTX=$(mktemp)`; e.g. `git diff > "$CTX"` or `gh pr diff > "$CTX"` for "review current PR". If the agent can gather it itself in the cwd (codex/cursor/claude can run `git diff`), skip this.
2. **Build the command** from the table. Inject the staged file via `@$CTX` (agy/gemini) or name `$CTX`/paths in `<TASK>` (codex/cursor/claude).
3. **Run it** with Bash. For multiple agents, launch them concurrently and wait for all.
4. **Relay** each agent's raw output, labeled by agent. Don't compress or editorialize unless asked — the point is what the other model said.

## Model override

Default to the flagship in the table. If the user names a model after the agent (`/cross-ai cursor opus`, `/cross-ai cursor with grok`, `/cross-ai codex gpt-5.4`), map it and pass it to that agent's model flag:

| Agent | Flag | Flagship (default) | Other selectable |
|-------|------|--------------------|------------------|
| `agy` | `--model` | `gemini-3.1-pro` | `gemini-3.5-flash`, `claude-opus`, `claude-sonnet`, `gpt-oss-120b` |
| `gemini` | `-m` | `gemini-3.1-pro-preview` | `gemini-3-pro-preview`, `gemini-2.5-pro`, `gemini-3-flash` |
| `codex` | `-m` | `gpt-5.5` | `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` |
| `cursor` | `--model` | `gpt-5.5` | `gpt-5.5-fast`, `opus-4.8`, `sonnet`, `gemini-3.1-pro`, `grok-4.3`, `composer-2.5` |
| `claude` | `--model` | `claude-opus-4-8` | `claude-sonnet-4-6`, `claude-opus-4-8[1m]` |

## Rules of engagement

- **Just run it.** Assume the binary is installed and authenticated. Do NOT `which`, version-check, `--help`, or test-run first. Build the command and execute in one step.
- **Read-only by default.** Never pass write/auto-approve flags (`--force`, `--yolo`, `--sandbox workspace-write`, `--permission-mode acceptEdits`) unless the user explicitly asks the external agent to modify files.
- **Stale model id is the only thing to check.** If a run fails because a model id was rejected, that is the one time to look: run the tool's list (`agy models`, `gemini --list-models`, `cursor-agent --list-models`, or check `--help`), pick the current flagship, and rerun. Model ids drift; the table is a snapshot.
- **Auth failure → report and stop.** If a run fails with an auth/login error, surface it verbatim and stop. Don't try to fix the other tool's credentials.
- **Not for Claude sub-agents.** `claude` here is the headless CLI process. The native Task tool ("launch a sub-agent") is unrelated and not dispatched through this skill.
