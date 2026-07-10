<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>Claudex</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

A multi-account companion for Claude Code: monitor usage across accounts, browse and move conversations, launch configured sessions, configure account settings and the statusline, and apply a bundled skill set to projects. Usage data comes in real time from Anthropic's OAuth API — exact 5-hour session, 7-day overall, and 7-day Sonnet limits with reset times.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h session, 7d overall, and 7d Sonnet utilization with reset countdowns |
| Conversations | `list` | List recent conversations with session IDs, message counts, and projects |
| | `switch` | Move a conversation between accounts, interactively or by session ID |
| Launcher | `launch` | Interactive TUI to configure and launch a Claude Code session, optionally loading plugins |
| Configuration | `configure` | Apply claudex's preferred Claude Code settings and statusline to an account |
| Skills | `apply-skills` | Install the embedded development skill set into the current project |
| | `apply-output-styles` | Install the embedded output style(s) into the current project |
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

Show live usage for all monitored accounts. Displays 5-hour session, 7-day overall, and 7-day Sonnet-specific utilization with reset countdowns per account. Fetches directly from Anthropic's usage API using OAuth tokens stored in macOS Keychain.

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

Interactively configure and launch a Claude Code session. Presents a TUI to select session mode, account, and MCP/connector behavior, then execs directly into `claude` with the assembled flags and environment.

```bash
claudex launch
```

The TUI starts with a mode selection (shown only when resumable sessions exist):
- **New session** — walks through account and MCP/connector selection
- **Resume** — pick from the most recent sessions for the current directory (across all accounts)

For new sessions, the remaining steps are:
- **Account** — select which `~/.claude*` directory to use (skipped if only one exists)
- **MCP + Connectors** — choose one of:
  - **MCPs only** — load your configured MCP servers, no claude.ai connectors
  - **MCPs + Connectors** — load MCP servers and enable claude.ai connectors (Gmail, Slack, etc.) via the `ENABLE_CLAUDEAI_MCP_SERVERS` setting
  - **None** — pass `--strict-mcp-config` to block all MCP servers and connectors

Resume sessions skip the prompts and automatically target the correct account via `CLAUDE_CONFIG_DIR`.

`launch` can also load Claude Code plugins alongside the session. A machine-local default plugin under `~/.config/claudex/default-plugin` is always loaded for every account (created empty on first use, for you to fill with your own always-on skills and output styles). Pass `--plugins` with one or more local directories or git repo URLs to load additional plugins; named git repos are cloned (or shallow-updated) under `~/.config/claudex/plugins` and applied on both new and resumed sessions.

```bash
claudex launch
claudex launch --plugins ~/my-plugin
claudex launch --plugins https://github.com/user/some-plugin
```

### `configure`

Apply claudex's preferred Claude Code configuration to an account. Writes `statusline.sh` into the account directory and merges a set of opinionated defaults into its `settings.json` — the statusline block, effort level, fullscreen TUI, auto-updater and connector env vars, and similar quality-of-life settings. Existing unrelated settings are preserved, and any env vars you already set survive the merge. The statusline label shown is derived from the directory name (`~/.claude` → `first`, `~/.claude2` → `second`).

```bash
claudex configure
claudex configure -A ~/.claude2
claudex configure -A ~/.claude2 --label prod
```

### `apply-skills`

Install a skill set into the current project, under `.claude/skills/`. By default it installs claudex's embedded development skills; point `--dir` at a directory to install your own skills instead. Matching is by skill name: each source skill **replaces** any same-named skill directory wholesale (so renamed or removed files never linger), while any existing skill that doesn't match a source name is left untouched. Use `--preserve-local` to keep every existing project skill and only add ones that aren't already present, or `--full-wipe` to clear the project's `.claude` skills and settings (`settings.json`/`settings.local.json`) first for a clean slate. Run it from the project root.

```bash
claudex apply-skills
claudex apply-skills --dir ~/my-skills
claudex apply-skills --preserve-local
```

### `apply-output-styles`

Install claudex's embedded output style(s) into the current project, under `.claude/output-styles/` (overwriting only same-named files, leaving your other styles untouched). Once installed, enable one ad hoc via `/config`. Run it from the project root.

```bash
claudex apply-output-styles
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

- Accounts are auto-discovered (`~/.claude`, `~/.claude2`, …)
- Usage data comes directly from Anthropic's OAuth API - same source as the official dashboard
- OAuth tokens are read from macOS Keychain; they refresh automatically when Claude Code is running
- If a token is expired, launch Claude Code on that account to refresh it
