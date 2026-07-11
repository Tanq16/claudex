<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>Claudex</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

A multi-account companion and single point of control for Claude Code: monitor usage across accounts, browse and move conversations, launch configured sessions, and provision every account at once. The model has two axes — **global** defaults you set once with `configure` (a curated always-on plugin, launch-time flavors, and per-account statusline/settings written across *every* discovered account) and **per-session** choices you make at `launch` (account, MCP mode, flavor, and any extra plugins). Usage data comes in real time from Anthropic's OAuth API — exact 5-hour session, 7-day overall, and 7-day Sonnet limits with reset times.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h session, 7d overall, and 7d Sonnet utilization with reset countdowns |
| Conversations | `list` | List recent conversations with session IDs, message counts, and projects |
| | `switch` | Move a conversation between accounts, interactively or by session ID |
| Launcher | `launch` | Interactive TUI to launch a Claude Code session — pick account, MCP mode, and flavor; always loads the global default plugin |
| Configuration | `configure` | Provision all accounts at once: preferred settings and statusline per account, plus the global default plugin and flavors directory |
| Skills | `apply-skills` | Install the embedded personal development skill set into the current project |
| Authentication | `oauth-token` | Obtain a Claude OAuth access token via browser-based PKCE flow |

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

## Usage

### `status`

Show live usage for all monitored accounts. Displays 5-hour session, 7-day overall, and 7-day Sonnet-specific utilization with reset countdowns per account. Fetches directly from Anthropic's usage API using each account's OAuth token — read from the macOS Keychain on macOS, and from `.credentials.json` in the account's config dir on Linux/Windows.

```bash
claudex status
claudex status -A ~/.claude2
claudex status -j
```

### `list`

List recent conversations across all discovered accounts. Shows session ID, message count, project, first message, and last activity time.

```bash
claudex list
claudex list -n 5
claudex list -j
```

### `switch`

Move a conversation from one account to another. Transfers session files and migrates history entries between config directories. Run with no flags for the interactive selector: it lists recent sessions for the current project across all accounts, then prompts for the target account (skipped when only one other account exists). Pass `--id`/`--from`/`--to` to skip the prompts and move a specific session non-interactively (handy for scripting).

```bash
claudex switch
claudex switch --id <session-uuid> --to ~/.claude2
claudex switch --id <session-uuid> --from ~/.claude2 --to ~/.claude3
```

### `launch`

Interactively launch a Claude Code session — this is where you make your per-session choices. Presents a TUI to select session mode, account, MCP/connector behavior, and flavor, then execs directly into `claude` with the assembled flags and environment. The global default plugin is always loaded (see below).

```bash
claudex launch
```

The TUI starts with a mode selection (shown only when resumable sessions exist):
- **New session** — walks through account, MCP/connector, and flavor selection
- **Resume** — pick from the most recent sessions for the current directory (across all accounts)

For new sessions, the remaining steps are:
- **Account** — select which `~/.claude*` directory to use (skipped if only one exists)
- **MCP + Connectors** — choose one of:
  - **MCPs only** — load your configured MCP servers, no claude.ai connectors
  - **MCPs + Connectors** — load MCP servers and enable claude.ai connectors (Gmail, Slack, etc.) via the `ENABLE_CLAUDEAI_MCP_SERVERS` setting
  - **None** — pass `--strict-mcp-config` to block all MCP servers and connectors
- **Flavor** — pick a system-prompt posture from `~/.config/claudex/flavors/` (see below); shown whenever at least one non-default flavor exists (a lone `default.md` is applied silently with no prompt)

Resume sessions skip the account/MCP prompts and automatically target the correct account via `CLAUDE_CONFIG_DIR`, but still apply the global plugin and flavor.

**Global default plugin.** A plugin under `~/.config/claudex/global` (manifest name `claudex`) is always loaded for every account, on every launch. It ships curated, broadly-useful content: the `cross-ai` and `ai-docs` skills (invoked as `/claudex:cross-ai` and `/claudex:ai-docs`) and the `caveman` output style. `launch` performs a gentle write-if-missing build — it lays down any missing curated items so a launch-before-configure still gets the defaults, but never clobbers items you or `configure` already refreshed. You can drop your own always-on skills and output styles into the conventional plugin folders and they load on every launch, across every account:

```
~/.config/claudex/global/
├── .claude-plugin/plugin.json    # manifest named "claudex"
├── skills/<skill-name>/SKILL.md  # your always-on skills, invoked as /claudex:<skill-name>
└── output-styles/<name>.md       # your always-on output styles
```

Because `caveman` ships inside this plugin, it is available in every claudex-launched session — no separate install step. Select it once with `/output-style` and the choice persists.

**Flavors.** Files in `~/.config/claudex/flavors/*.md` are launch-time system-prompt append postures. Each file's whole content is the append text (no frontmatter), applied via `--append-system-prompt`, and the filename stem is its TUI label. The `default.md` file is not a master switch — it is the auto-apply-when-alone, pre-selected-when-many entry:

| `flavors/` contains | Behavior at launch |
|---|---|
| nothing | nothing applied, no prompt |
| only `default.md` | apply it silently — no TUI, no None |
| `default.md` + others | pick one (`default` pre-selected) or None |
| others, no `default.md` | pick one or None |

Running `configure` creates this directory for you; add your own flavor `.md` files to it.

**Extra plugins.** Pass `--plugins` with one or more local directories or git repo URLs to load additional plugins alongside the global one; named git repos are cloned (or shallow-updated) under `~/.config/claudex/plugins` and applied on both new and resumed sessions.

```bash
claudex launch
claudex launch --plugins ~/my-plugin
claudex launch --plugins https://github.com/user/some-plugin
```

### `configure`

The single control point for your global defaults. By default it provisions **every discovered account at once** and lays down all account-independent defaults in one run:

- **Per account** — writes `statusline.sh` into each account directory and merges a set of opinionated defaults into its `settings.json`: the statusline block, effort level, fullscreen TUI, auto-updater and connector env vars, and similar quality-of-life settings. Existing unrelated settings are preserved, and any env vars you already set survive the merge. The statusline label is derived from the directory name (`~/.claude` → `first`, `~/.claude2` → `second`, `~/.claude-alice` → `alice`).
- **Global default plugin** — authoritatively refreshes the curated items in `~/.config/claudex/global` to the latest embedded versions (replace-by-name), while preserving any skills or output styles you added yourself. (`launch` only writes missing items; `configure` brings them fully up to date.)
- **Flavors** — scaffolds `~/.config/claudex/flavors/` so you can drop in your launch-time postures.

Use `-A <path>` to target a single account instead of all of them. `--label` is a per-account statusline override and is only meaningful with `-A` (it errors if given without it).

```bash
claudex configure
claudex configure -A ~/.claude2
claudex configure -A ~/.claude2 --label prod
```

### `apply-skills`

Install the personal development skill set into the current project, under `.claude/skills/` — the per-project helper (the broadly-useful `cross-ai` and `ai-docs` skills now ride the global plugin instead). By default it installs claudex's embedded development skills; point `--dir` at a directory to install your own skills instead. Matching is by skill name: each source skill **replaces** any same-named skill directory wholesale (so renamed or removed files never linger), while any existing skill that doesn't match a source name is left untouched. Use `--preserve-local` to keep every existing project skill and only add ones that aren't already present, or `--full-wipe` to clear the project's `.claude` skills and settings (`settings.json`/`settings.local.json`) first for a clean slate. Run it from the project root.

```bash
claudex apply-skills
claudex apply-skills --dir ~/my-skills
claudex apply-skills --preserve-local
```

### `oauth-token`

Obtain a Claude OAuth access token via the browser-based PKCE flow. Opens your browser for authentication and prints the access token to stdout.

```bash
claudex oauth-token
TOKEN=$(claudex oauth-token)
claudex oauth-token --expires-in 3600
claudex oauth-token --port 8080
```

## Tips and Notes

- Accounts are auto-discovered permissively — any `~/.claude*` directory counts, so `~/.claude`, `~/.claude2`, and arbitrary names like `~/.claude-alice` or `~/.claude-bob` are all picked up automatically
- Run `configure` once to provision every account and lay down the global defaults (curated plugin + flavors directory); make per-session choices at `launch`
- Usage data comes directly from Anthropic's OAuth API - same source as the official dashboard
- OAuth tokens are read from the macOS Keychain (macOS) or `.credentials.json` in the account's config dir (Linux/Windows); they refresh automatically when Claude Code is running
- If a token is expired, launch Claude Code on that account to refresh it
