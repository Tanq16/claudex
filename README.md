<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>Claudex</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a> &bull; <a href="#oauth-token">OAuth</a>
</div>

---

Monitor Claude Code usage across multiple accounts. Fetches real-time utilization from Anthropic's OAuth API to show exact 5-hour session, 7-day overall, and 7-day Sonnet-specific limits and reset times.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h session, 7d overall, and 7d Sonnet utilization with reset countdowns |
| History | `history` | Daily breakdown of messages, sessions, tool calls, and token usage |
| Conversations | `convos list` | List recent conversations with session IDs, message counts, and projects |
| | `convos switch` | Move a conversation from one account to another |
| | `convos reproject` | Change which project a conversation is associated with |
| | `convos find` | Find conversations by ID or keyword search |
| Plugins | `plugin instate` | Instantiate plugins for a local project with version reconciliation |
| | `plugin cleanup` | Clean up orphaned or stale plugin cache entries |
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

### Global Flags

- `--debug` - Enable debug logging (zerolog with timestamps and full error details)
- `--for-ai` - AI-friendly output (plain text prefixes like `[OK]`, `[INFO]`, `[ERROR]`, `[WARN]`; markdown tables; piped stdin for prompts)

### `status`

Show live usage for all monitored accounts. Displays 5-hour session, 7-day overall, and 7-day Sonnet-specific utilization with reset countdowns per account. Fetches directly from Anthropic's usage API using OAuth tokens stored in macOS Keychain.

```bash
claudex status
claudex status -a ~/.claude2
claudex status -j
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor (default: `~/.claude` only)
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

### `convos list`

List recent conversations across all monitored accounts. Shows session ID, message count, project, first message, and last activity time.

```bash
claudex convos list
claudex convos list -n 5
claudex convos list -j
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-n, --limit` - Number of conversations to show (default: `10`)
- `-j, --json` - Output as JSON (includes full session UUIDs)

### `convos switch`

Move a conversation from one account to another. Transfers session files and migrates history entries between config directories.

```bash
claudex convos switch --id <session-uuid> --to ~/.claude2
claudex convos switch --id <session-uuid> --from ~/.claude2 --to ~/.claude3
```

**Flags:**
- `--id` - Session UUID to switch (required)
- `--from` - Source config directory (default: `~/.claude`)
- `--to` - Target config directory (required)

### `convos reproject`

Change which project a conversation is associated with. Moves session files to the target project's directory and updates history entries.

```bash
claudex convos reproject --id <session-uuid>
claudex convos reproject --id <session-uuid> --project /path/to/project
claudex convos reproject --id <session-uuid> -c ~/.claude2
```

**Flags:**
- `--id` - Session UUID to reproject (required)
- `-c, --config-dir` - Config directory (default: `~/.claude`)
- `--project` - Target project path (default: current directory)

### `convos find`

Find conversations by session ID or keyword search across all monitored accounts.

```bash
claudex convos find --id <session-uuid>
claudex convos find -k "search term"
claudex convos find -k "regex pattern" -a ~/.claude2 -j
```

**Flags:**
- `--id` - Session UUID to find
- `-k, --keyword` - Regex keyword to search
- `-a, --accounts` - Additional Claude config directories to search
- `-j, --json` - Output as JSON

### `oauth-token`

Obtain a Claude OAuth access token via the browser-based PKCE flow. Opens your browser for authentication and prints the access token to stdout.

```bash
claudex oauth-token
TOKEN=$(claudex oauth-token)
claudex oauth-token --expires-in 3600
claudex oauth-token --port 8080
```

**Flags:**
- `-p, --port` - Local port for OAuth callback server (default: `54545`)
- `-e, --expires-in` - Requested token expiry in seconds (default: `3600`; server may override)

### `plugin instate`

Instantiate plugins for a local project with version reconciliation. Supports both local marketplace plugins (resolved from the marketplace directory) and external GitHub-hosted plugins (shallow-cloned automatically).

```bash
claudex plugin instate
claudex plugin instate -c ~/.claude2
claudex plugin instate -P core@ai-brain -u
claudex plugin instate -P sales@praetorian-ai-marketplace
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
