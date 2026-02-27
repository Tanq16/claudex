<div align="center">
  <img src=".github/assets/logo.svg" alt="Claude Usage Logo" width="200">
  <h1>Claude Usage</h1>

  <a href="https://github.com/tanq16/claude-usage/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claude-usage/actions/workflows/release.yaml/badge.svg"></a><br>
  <a href="https://github.com/tanq16/claude-usage/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claude-usage"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Monitor Claude Code usage across multiple accounts. Fetches real-time utilization from Anthropic's OAuth API to show exact 5-hour session, 7-day overall, and 7-day Sonnet-specific limits, reset times, and capacity planning for pending tasks.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h session, 7d overall, and 7d Sonnet utilization with reset countdowns |
| History | `history` | Daily breakdown of messages, sessions, tool calls, and token usage |
| Conversations | `conversations` / `convos` | List recent conversations with session IDs, message counts, and projects |
| Tasks | `task add`, `task list`, `task done`, `task remove` | Track pending work with size estimates |
| Planning | `plan` | Fit pending tasks into available capacity across accounts |

## Installation

### Binary

Download from [releases](https://github.com/tanq16/claude-usage/releases):

```bash
# Linux/macOS
curl -sL https://github.com/tanq16/claude-usage/releases/latest/download/claude-usage-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o claude-usage
chmod +x claude-usage
sudo mv claude-usage /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/tanq16/claude-usage
cd claude-usage
make build
```

## Usage

### Global Flags

- `--debug` - Enable debug logging (zerolog)

### `status`

Show live usage for all monitored accounts. Displays 5-hour session, 7-day overall, and 7-day Sonnet-specific utilization with reset countdowns. Fetches directly from Anthropic's usage API using OAuth tokens stored in macOS Keychain.

```bash
claude-usage status
claude-usage status -j
claude-usage status -a ~/.claude2
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor (default: `~/.claude` only)
- `-j, --json` - Output as JSON

### `history`

Show daily usage history from the local stats-cache, including messages, sessions, tool calls, and token usage by model.

```bash
claude-usage history
claude-usage history -d 14
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-d, --days` - Number of days to show (default: `7`)
- `-j, --json` - Output as JSON

### `conversations` / `convos`

List recent conversations across all monitored accounts. Shows session ID, message count, project, first message, and last activity time.

```bash
claude-usage convos
claude-usage convos -n 5
claude-usage conversations -j
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-n, --limit` - Number of conversations to show (default: `10`)
- `-j, --json` - Output as JSON (includes full session UUIDs)

### `task`

Manage tasks for capacity planning. Each task has a size estimate (S/M/L/XL) that maps to approximate percentage of a 5-hour session window.

```bash
claude-usage task add "Implement auth module" --size L
claude-usage task add "Fix CSS bug" -s S
claude-usage task list           # or: task ls
claude-usage task done <id>      # or: task complete <id>
claude-usage task remove <id>    # or: task rm <id>
```

**Aliases:** `list` → `ls`, `done` → `complete`, `remove` → `rm`

**Size Estimates:**

| Size | Approx. Turns | % of 5h Window |
|------|---------------|----------------|
| S | ~10 | ~5% |
| M | ~30 | ~15% |
| L | ~75 | ~35% |
| XL | ~150 | ~70% |

### `plan`

Assign pending tasks to accounts based on available capacity using greedy bin-packing.

```bash
claude-usage plan
claude-usage plan -j
claude-usage plan -a ~/.claude2
```

**Flags:**
- `-a, --accounts` - Additional Claude config directories to monitor
- `-j, --json` - Output as JSON

### `plugin instate`

Instantiate plugins for a local project with version reconciliation.

```bash
claude-usage plugin instate
claude-usage plugin instate -c ~/.claude2 -p ./myproject
claude-usage plugin instate -P core@ai-brain -u
claude-usage plugin instate -A
```

**Flags:**
- `-c, --config-dir` - Claude config directory (default `~/.claude`)
- `-p, --project` - Project directory (default cwd)
- `-P, --plugins` - Comma-separated plugin keys
- `-A, --all` - Instate all available plugins
- `-u, --update` - Git pull marketplace repos before reconciling

### `plugin cleanup`

Clean up orphaned or stale plugin cache entries.

```bash
claude-usage plugin cleanup
claude-usage plugin cleanup -o -A
claude-usage plugin cleanup -c ~/.claude2 -P core@ai-brain
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
- Task data persists in `~/.config/claude-usage/tasks.json`
