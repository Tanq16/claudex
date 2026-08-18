<div align="center">
  <img src=".github/assets/logo.svg" alt="ClaudeX Logo" width="200">
  <h1>ClaudeX</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#install">Install</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#notes">Notes</a>
</div>

---

ClaudeX is a companion CLI for Claude Code that finds every Claude account on the machine and configures them identically, writes an `AGENTS.md` and skills layout into a project, and launches a session under the account you pick.

It exists for juggling several Claude subscriptions, and everything except the account picker works the same with one. It does not replace `claude`, which `launch` execs.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Accounts | `configure`, `status`, `switch`, `oauth-token` | Provision every account, read its usage, move a project between accounts |
| Sessions | `launch` | Pick the account and the session, then exec `claude` |
| Project layout | `apply`, `apply-preset`, `clean-cwd` | Write and remove the `AGENTS.md` and skills layout |
| Presets | `create-preset` | Scaffold your own bundle of skills and rules |

## Install

### Release binary

```bash
# Linux/macOS
curl -sL https://github.com/tanq16/claudex/releases/latest/download/claudex-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o claudex
chmod +x claudex
sudo mv claudex /usr/local/bin/
```

Each release carries `claudex-linux-amd64`, `claudex-linux-arm64`, `claudex-darwin-amd64`, and `claudex-darwin-arm64`.

### From source

Needs Go 1.26.

```bash
git clone https://github.com/tanq16/claudex.git
cd claudex
make build
```

`make build-all` produces all four platform binaries instead.

## Usage

`--debug` and `--for-ai` are on every command and are mutually exclusive: the first adds log lines, the second drops color and symbols. `-A/--account` takes an account config directory path, and on `launch` and `switch` it also matches on just the directory name. Global state lives under `~/.config/claudex/`, holding the plugin in `global/` and presets in `presets/`.

### configure

Provisions every discovered account, then lays down the global defaults once.

Per account it writes `statusline.sh` into the account directory and points `settings.json` at it. The statusline shows the account label, the model, the working directory, the git branch, context used, and the 5h and 7d rate-limit percentages. It also merges these keys into the existing `settings.json`, leaving every other key alone:

| Key | Value |
|-----|-------|
| `attribution.commit` | `""` |
| `effortLevel` | `xhigh` |
| `tui` | `fullscreen` |
| `autoMemoryEnabled` | `false` |
| `skipDangerousModePermissionPrompt` | `true` |
| `env.DISABLE_AUTOUPDATER` | `1` |
| `env.ENABLE_CLAUDEAI_MCP_SERVERS` | `false` |

An account whose `settings.json` is not valid JSON is skipped rather than overwritten.

The global defaults are a Claude Code plugin at `~/.config/claudex/global/`, carrying an `.lsp.json` that wires `gopls`, `pyright-langserver`, and `typescript-language-server`, plus the built-in presets extracted into `~/.config/claudex/presets/`.

`-A` configures one account instead of all of them. `-l/--label` overrides the account label in the statusline and requires `-A`; without it the label comes from the directory name, where `.claude` is `first`, `.claude2` is `second`, `.claude3` is `third`, and anything else uses the numeric suffix.

### apply

Writes the layout into the current directory:

```
AGENTS.md          base instruction block, between <!-- claudex:base --> markers
CLAUDE.md          -> AGENTS.md
.agents/skills/    session-summary, skill-creator, write-document
.claude/skills     -> ../.agents/skills
```

`AGENTS.md` and `.agents/skills/` are the real files, following the [Agent Skills](https://agentskills.io) layout that Cursor and Codex read on their own, and the two symlinks exist because Claude Code looks for the other names.

An existing `AGENTS.md` keeps everything outside the markers: the base block is inserted or replaced in place. The same four paths also go into `.git/info/exclude`, which is local to your clone, so the layout never appears in `git status` and never gets pushed.

Nothing is written when any of those paths already holds something ClaudeX did not put there. Every conflict is reported at once so you can clear them in one pass rather than one run per path.

### apply-preset

A preset is a directory under `~/.config/claudex/presets/` holding a `preset.yaml`, an optional `AGENTS.partial.md`, and a `skills/` directory. Applying one symlinks its skills into `.agents/skills/` and writes its partial as its own marked section of `AGENTS.md`, keyed by preset name, so re-applying replaces that section instead of appending a second copy. It needs `claudex apply` to have run first.

```bash
claudex apply-preset            # multi-select picker
claudex apply-preset private    # by name; several names apply in order
```

`-s/--skills` links only the skills and leaves `AGENTS.md` alone. `-a/--agents` writes only the section and links no skills. Neither flag applies the whole preset; passing one narrows the run to that half.

One preset ships in the binary. `private` carries 30 skills covering Go and Node conventions, containers, release workflows, and testing, plus the author's development, pull request, and operating rules as an `AGENTS.md` section.

The manifest keys:

| Key | Default | Description |
|-----|---------|-------------|
| `name` | the directory name | Shown in the picker and used as the `AGENTS.md` section key |
| `description` | empty | One line shown beside the name in the picker |
| `skills` | every directory under `skills/` holding a `SKILL.md` | Which skills to link |

### create-preset

Scaffolds `~/.config/claudex/presets/<name>/` with a `preset.yaml`, an `AGENTS.partial.md`, and an empty `skills/`. Names take lowercase letters, digits, and single hyphens.

### clean-cwd

Removes what `apply` and `apply-preset` wrote: `.agents/`, both symlinks, the ClaudeX sections of `AGENTS.md`, and the `.git/info/exclude` block.

### launch

Prompts for new or resume, for the account, and for MCP mode, then execs `claude` with `--dangerously-skip-permissions`, a `--plugin-dir` pointing at the global plugin, and `CLAUDE_CONFIG_DIR` set to the account you picked. Any inherited `CLAUDE_CONFIG_DIR` is stripped first so it cannot override that choice. The plugin is rebuilt on every launch, so language servers work without running `configure`.

The new-or-resume prompt only appears when this project has sessions, and the account prompt only when there is more than one account. Resume lists this project's 10 most recent sessions across every account, and picking one launches under the account that holds it.

| Flag | Effect |
|------|--------|
| `-A/--account` | Skip the account picker |
| `--new` | Start a new session |
| `--resume` | Resume: the latest session, or a list when there is more than one |
| `--session <id>` | Resume that session by id |
| `--mcp mcps\|connectors\|none` | Skip the MCP picker |

`--new`, `--resume`, and `--session` are mutually exclusive. The MCP modes are `mcps` for MCP servers on, `connectors` to add Claude.ai connectors on top, and `none` for `--strict-mcp-config`. Launch needs an interactive terminal, so `--for-ai` errors.

### status

Per account, the 5h session window and the 7d windows as bars with their reset times. It reads the account's OAuth token from the macOS Keychain or from `.credentials.json` and queries Anthropic's usage endpoint, so an expired token shows as a prompt to open Claude Code on that account. `-j/--json` prints the raw numbers, `-A` limits it to one account.

### switch

Moves the current project's session files and history entries out of the account holding them and into another. It needs at least two accounts, and `-A/--account` is required under `--for-ai`.

The picker lists this project's sessions from every account, under a row that takes all the ones in the account holding the newest. Picking a single session moves only that session, out of whichever account holds it. `--session <id>` names one directly and skips the picker, and `--for-ai` has no picker, so it moves the whole account unless `--session` narrows it.

### oauth-token

Runs the OAuth PKCE flow in a browser and prints an access token to stdout. `-p/--port` sets the local callback port and `-e/--expires-in` the requested expiry in seconds, which the server may override.

## Notes

- **Account discovery.** `~/.claude` and every `~/.claudeN` directory whose suffix is digits, such as `~/.claude2`. Nothing else in your home directory counts as an account.
- **Preset skills are symlinks.** They point back into `~/.config/claudex/presets/`, so editing a preset changes every project that applied it. The three base skills from `apply` are real copies extracted from the binary, so re-running `apply` is what updates them.
- **Built-in presets are refreshed from the binary.** `~/.config/claudex/presets/private/` is rewritten whenever a preset command runs, so edits to it do not survive. Presets you create yourself are never touched.
- **Language server binaries are yours to install.** ClaudeX writes the `.lsp.json`, and a server whose binary is missing is skipped while the rest still start. Install commands and the `typescript@5` pin are in [docs/language-servers.md](docs/language-servers.md).
- **Another repo's agent files.** `.git/info/exclude` only reaches untracked files, so a `GEMINI.md` or `.cursor/` that the repository itself tracks stays in your working tree. [docs/foreign-agent-files.md](docs/foreign-agent-files.md) covers the sparse-checkout that removes them.
