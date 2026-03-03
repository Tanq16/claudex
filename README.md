<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>Claudex</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a><br>
  <a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Monitor Claude Code usage across multiple accounts. Fetches real-time utilization from Anthropic's OAuth API to show exact 5-hour session, 7-day overall, and 7-day Sonnet-specific limits and reset times.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h session, 7d overall, and 7d Sonnet utilization with reset countdowns |
| History | `history` | Daily breakdown of messages, sessions, tool calls, and token usage |
| Conversations | `conversations` / `convos` | List recent conversations with session IDs, message counts, and projects |

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

### Global Flags

- `--debug` - Enable debug logging (zerolog)

### `status`

Show live usage for all monitored accounts. Displays 5-hour session, 7-day overall, and 7-day Sonnet-specific utilization with reset countdowns. Fetches directly from Anthropic's usage API using OAuth tokens stored in macOS Keychain. By default, multiple accounts are shown side by side with individual bars and reset times per row; use `-s` to see each account as a fully separate block with recommendations.

```bash
claudex status
claudex status -a ~/.claude2
claudex status -s
claudex status -j
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor (default: `~/.claude` only)
- `-s, --separate` - Show each account as a separate block with individual recommendations
- `-j, --json` - Output as JSON

### `history`

Show daily usage history from the local stats-cache, including messages, sessions, tool calls, and token usage by model.

```bash
claudex history
claudex history -d 14
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-d, --days` - Number of days to show (default: `7`)
- `-j, --json` - Output as JSON

### `conversations` / `convos`

List recent conversations across all monitored accounts. Shows session ID, message count, project, first message, and last activity time.

```bash
claudex convos
claudex convos -n 5
claudex conversations -j
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-n, --limit` - Number of conversations to show (default: `10`)
- `-j, --json` - Output as JSON (includes full session UUIDs)

### `plugin instate`

Instantiate plugins for a local project with version reconciliation.

```bash
claudex plugin instate
claudex plugin instate -c ~/.claude2
claudex plugin instate -P core@ai-brain -u
claudex plugin instate -A
```

**Flags:**
- `-c, --config-dir` - Claude config directory (default `~/.claude`)
- `-P, --plugins` - Comma-separated plugin keys
- `-A, --all` - Instate all available plugins
- `-u, --update` - Git pull marketplace repos before reconciling

### `plugin cleanup`

Clean up orphaned or stale plugin cache entries.

```bash
claudex plugin cleanup
claudex plugin cleanup -o -A
claudex plugin cleanup -c ~/.claude2 -P core@ai-brain
```

**Flags:**
- `-c, --config-dir` - Claude config directory (default `~/.claude`)
- `-P, --plugins` - Comma-separated plugin keys
- `-o, --orphans-only` - Only remove orphaned version dirs
- `-A, --all` - Target all plugins

## Tips and Notes

- Default monitoring is `~/.claude` only; use `-a/--accounts` on each monitoring command to add more (e.g., `-a ~/.claude2,~/.claude3`)
- Usage data comes directly from Anthropic's OAuth API - same source as the official dashboard
- OAuth tokens are read from macOS Keychain; they refresh automatically when Claude Code is running
- If a token is expired, launch Claude Code on that account to refresh it
