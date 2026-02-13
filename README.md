<div align="center">
  <h1>Claude Usage</h1>

  <a href="https://github.com/tanq16/claude-usage/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claude-usage/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claude-usage/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claude-usage"></a><br><br>

  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Monitor Claude Code usage across multiple accounts. Fetches real-time utilization from Anthropic's OAuth API to show exact 5-hour session and 7-day rolling limits, reset times, and capacity planning for pending tasks.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Monitoring | `status` | Live 5h/7d utilization from Anthropic API with reset countdowns |
| History | `history` | Daily breakdown of messages, sessions, and token usage |
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
make build-local
```

## Usage

### `status`

Show live usage for all monitored accounts. Fetches directly from Anthropic's usage API using OAuth tokens stored in macOS Keychain.

```bash
claude-usage status
claude-usage status --json
claude-usage status --accounts ~/.claude2
```

**Flags:**
- `--json` - Output as JSON
- `--accounts` - Additional Claude config directories to monitor (default: `~/.claude` only)

### `history`

Show daily usage history from the local stats-cache.

```bash
claude-usage history
claude-usage history --days 14
```

**Flags:**
- `--days, -d` - Number of days to show (default: `7`)
- `--json` - Output as JSON

### `task`

Manage tasks for capacity planning. Each task has a size estimate (S/M/L/XL) that maps to approximate percentage of a 5-hour session window.

```bash
claude-usage task add "Implement auth module" --size L
claude-usage task add "Fix CSS bug" --size S
claude-usage task list
claude-usage task done <id>
claude-usage task remove <id>
```

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
claude-usage plan --accounts ~/.claude2
```

## Tips and Notes

- Default monitoring is `~/.claude` only; use `--accounts` to add more (e.g., `--accounts ~/.claude2,~/.claude3`)
- Usage data comes directly from Anthropic's OAuth API - same source as the official dashboard
- OAuth tokens are read from macOS Keychain; they refresh automatically when Claude Code is running
- If a token is expired, launch Claude Code on that account to refresh it
- Task data persists in `~/.claude-usage/data.db` (BoltDB)
- Use `--debug` on any command for detailed zerolog output
