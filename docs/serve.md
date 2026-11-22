# serve

A local web UI that manages the projects under one root directory and the Claude sessions running in them.

```bash
claudex serve                                   # http://127.0.0.1:8080, projects under ~/.config/claudex/projects
claudex serve --root ~/code -p 9000
```

`serve` also builds the global plugin and lays down the built-in presets and base skills under `~/.config/claudex/` at every start.

## Session modes

A session runs detached or attached, chosen when it starts. A detached session is a window in a tmux session named `claudex`, running `claude` with no shell in between, so `tmux attach -t claudex` reaches the terminal at any time. Detached sessions outlive `serve`, since tmux is its own process. Remote Control is an optional launch toggle for them, and its name defaults to the project name.

An attached session is a `claude -p` stream-json process that `serve` hosts itself and shows as a chat on the project page. Attached sessions stop with `serve` and come back with `claude --continue` at its next start. Ctrl+C on `serve` exits at once when every attached session is idle. Otherwise it lists what is still running, and a second Ctrl+C stops those sessions and exits. `SIGTERM`, or a `serve` without a terminal, stops them without asking.

A project shows as active while its session runs, and a restart continues the conversation with `claude --continue`.

## Projects

The new-session dialog starts from an existing directory under the root, a new empty one, or a clone from `gh`. It picks the session mode, Remote Control, and whether to continue the last conversation, then applies presets, picks MCP servers from the library, and launches. A project page browses and renders the project's files, takes new files and uploads, runs shell commands in the project directory, and copies the conversation as Markdown. It also holds restart, stop, account switch, `clean-cwd`, preset, and MCP controls.

## Chat

An attached session's page centers on the chat, which shows replies, tool calls, subagents, background commands, workflows, and to-dos as they happen. Approval cards answer tool permissions, questions, plans, and MCP prompts. The composer switches the model, effort, and permission mode, suggests slash commands, and Esc stops the turn. A pending approval in a project that is not on screen raises a toast and marks the Dashboard link. Settings covers each account's compliance with the defaults, the defaults themselves, tracked remote presets, and the MCP library, whose entries carry their credentials inline in `~/.config/claudex/mcp-library.json`.

## Defaults editor

The defaults editor shows your active keys grouped by section, each with an inline control, or the whole set as one editable `settings.json` document. Add key searches Claude Code's published settings schema (https://json.schemastore.org/claude-code-settings.json), cached at `~/.config/claudex/settings-schema.json` and refreshed from the page, and also takes keys and environment variables the schema does not list yet. Each account card lists the keys in its `settings.json` that the defaults do not cover. Adopt copies one into the defaults, and Remove deletes it from that account.
