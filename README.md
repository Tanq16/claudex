<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>ClaudeX</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a>
</div>

---

ClaudeX is a companion CLI for running Claude Code across more than one account. If you keep separate Claude subscriptions — a personal one, a work one, a spare for when the first hits its limit — ClaudeX is the single place to see where each one stands, jump into the right one, move a conversation between them, and set them all up identically in one shot.

It finds your accounts on its own: `~/.claude` and its numbered siblings (`~/.claude2`, `~/.claude3`, …). Everything ClaudeX sets up for itself — a shared global plugin, your launch flavors, and any plugins it fetches — lives under `~/.config/claudex`. The whole workflow is two steps: run `configure` once to provision every account, then use `launch` every time you start working.

## Capabilities

| Command | What it gives you |
|---------|-------------------|
| `configure` | One-shot setup of every account plus the shared global plugin — run it once after installing |
| `status` | Live usage across all accounts: 5h session, weekly overall, and weekly per-model windows, each with a reset countdown |
| `launch` | Guided start of a Claude Code session — right account, MCP mode, and flavor, with the global plugin always loaded |
| `switch` | Move a conversation from one account to another and continue it there |
| `oauth-token` | A Claude OAuth access token via the browser PKCE flow |
| `apply-skills` | Drop ClaudeX's opinionated development skills into the current project |
| `ai-docs` | Serve the ai-docs viewer for capturing durable HTML docs in the current project |

## Installation

### Binary

Download from [releases](https://github.com/tanq16/claudex/releases):

```bash
# Linux/macOS
curl -sL https://github.com/tanq16/claudex/releases/latest/download/claudex-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o claudex
chmod +x claudex
sudo mv claudex /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/tanq16/claudex
cd claudex
make build
```

Once it's on your PATH, run `claudex configure` — that one command sets up every account and lays down the global defaults. After that, `claudex launch` is all you need day to day.

## Usage

### `configure`

Run this once, right after installing. With no arguments it provisions **every account it discovers** in a single pass:

- **Per account** — writes a statusline and a set of opinionated `settings.json` defaults into each account directory. Your existing settings and env vars are preserved; only ClaudeX's keys are merged in.
- **The global plugin** — builds a single always-on plugin at `~/.config/claudex/global` that every account shares. This is ClaudeX's single point of control for global content: it ships one output style (`caveman`) and two skills, `cross-ai` and `ai-docs`, so those are present in every session on every account with no per-account setup. Anything you drop into its `skills/` or `output-styles/` folders rides along the same way.
- **Flavors** — creates `~/.config/claudex/flavors/` for your launch-time system-prompt postures (see [`launch`](#launch)).

Target a single account with `-A <path>`; `--label` names that account's statusline and only applies with `-A`. After this, day-to-day use is just `launch`.

```bash
claudex configure
claudex configure -A ~/.claude2 --label prod
```

### `status`

See live usage for every account at once — the 5-hour session window, the weekly overall window, and the weekly per-model windows (currently Fable), each with a reset countdown. This is the multi-account payoff: one glance tells you which account has room and which is about to hit a limit, so you know where to launch next.

Numbers come straight from Anthropic's OAuth usage API — the same source as the official dashboard. Tokens are read from the macOS Keychain, or from each account's `.credentials.json` on Linux/Windows, and refresh on their own while Claude Code is running; if one shows as expired, launch Claude Code on that account to refresh it.

```bash
claudex status
claudex status -A ~/.claude2
claudex status -j
```

### `launch`

The one command you run to start working. Rather than remembering which account is which, which env vars turn on connectors, and which system prompt you wanted, `launch` walks you through it and execs straight into `claude` with everything wired up:

- **Account** — pick which one to use (skipped if you only have one).
- **MCP + connectors** — MCP servers only, MCP servers plus claude.ai connectors (Gmail, Slack, …), or none at all.
- **Flavor** — a system-prompt posture (see below).
- **Resume** — when the current directory has recent sessions, jump back into one instead; it targets the right account automatically.

Every launch loads the global plugin that `configure` built, so your global skills and output style are always there. (Launching before you've run `configure` still works — it lays down anything missing without touching what you've customized.)

**Flavors** are reusable launch-time postures — one `.md` file per posture in `~/.config/claudex/flavors/`, where the whole file becomes the appended system prompt and the filename is its label. Keep a few modes around (terse, planning-first, review-only, whatever fits) and pick one at launch. `default.md` is not a master switch, just a convenient default:

| `flavors/` contains | Behavior at launch |
|---|---|
| nothing | nothing applied, no prompt |
| only `default.md` | applied silently — no prompt |
| `default.md` + others | pick one (`default` pre-selected) or None |
| others, no `default.md` | pick one or None |

**Extra plugins for a single session** go through `--plugins`, taking a local directory or a git URL. Git repos are cloned (or shallow-updated if already fetched) under `~/.config/claudex/plugins` and loaded alongside the global one. This is deliberately simpler than Claude Code's built-in plugin manager, which tends to leave orphaned versions behind — claudex just clones the repo to a fixed spot, pulls latest when it's already there, and loads it inline.

```bash
claudex launch
claudex launch --plugins ~/my-plugin
claudex launch --plugins https://github.com/user/some-plugin
```

### `switch`

Started a conversation on the wrong account, or want to keep going on one with more capacity? `switch` moves a conversation from one account to another — it relocates the session's project directory and its history entries into the target account so you can pick the thread right back up there. Run it bare for an interactive picker, or pass `--id`/`--from`/`--to` to move a specific session non-interactively.

```bash
claudex switch
claudex switch --id <session-uuid> --to ~/.claude2
```

### `oauth-token`

Obtain a Claude OAuth access token via the browser-based PKCE flow (valid one hour by default). It opens your browser to authenticate and prints the token to stdout, so `TOKEN=$(claudex oauth-token)` just works. `--expires-in` and `--port` are there if you need them.

```bash
claudex oauth-token
TOKEN=$(claudex oauth-token)
```

### `apply-skills`

An author-opinionated set of development skills you can drop into a project's `.claude/skills/` — run it from a project root and it installs ClaudeX's embedded dev skills there, and only there. Point `--dir` at your own skill set instead, `--preserve-local` to add only what's missing, or `--full-wipe` to clear the project's ClaudeX skills and settings first for a clean slate. (These are per-project skills; the always-on `cross-ai` and `ai-docs` skills ride the global plugin instead.)

```bash
claudex apply-skills
claudex apply-skills --dir ~/my-skills
```

### `ai-docs`

`ai-docs` is one of the global-plugin skills — it captures durable deliverables (architecture, design, research, analysis) as curated HTML you read through a small local viewer. This command is a thin launcher for that viewer: it execs the skill's Node server and serves `./AI-docs` in the current project (created on first run) at `http://127.0.0.1:4321`. It depends on Node.js being on your PATH and the global plugin being built (`claudex configure`). Use `--docs` to serve a different directory and `--port` to run more than one viewer at once.

```bash
claudex ai-docs
claudex ai-docs --docs security-docs --port 4322
```
