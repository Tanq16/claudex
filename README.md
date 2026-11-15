<div align="center">
  <img src=".github/assets/logo.svg" alt="ClaudeX Logo" width="200">
  <h1>ClaudeX</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#install">Install</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#notes">Notes</a>
</div>

---

ClaudeX is a companion CLI and local web UI for Claude Code. It configures every Claude account on the machine identically, writes an `AGENTS.md` and skills layout into a project, and runs Claude sessions under the account you pick, either in your terminal or in tmux behind Remote Control.

It exists for juggling several Claude subscriptions, and everything except account selection works the same with one. It does not replace `claude`, which it execs or runs in tmux.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Accounts | `configure`, `status`, `switch` | Provision every account, read its usage, move a project between accounts |
| Sessions | `launch`, `serve` | Exec `claude` under one account, or manage tmux sessions from a web UI |
| Project layout | `apply`, `apply-preset`, `clean-cwd` | Write and remove the `AGENTS.md` and skills layout |
| Presets | `create-preset`, `pull-preset` | Scaffold your own bundle of skills and rules, or pull one from a repository |

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

Needs Go 1.27.

```bash
git clone https://github.com/tanq16/claudex.git
cd claudex
make build
```

`make build-all` produces all four platform binaries instead.

## Usage

`--debug` is on every command and swaps the styled output for zerolog: a timestamped console line in a terminal, and JSON when stdout is not one. Color and cursor redraws drop automatically off a terminal too, so piping any command gives plain text with no flag. Every input is a flag or an argument, and nothing prompts. `-A/--account` takes an account config directory path, and on `launch` and `switch` it also matches on just the directory name. Global state lives under `~/.config/claudex/`, holding the plugin in `global/`, presets in `presets/`, and the web UI's state in `state.json`.

### configure

Provisions every discovered account, then lays down the global defaults once.

Per account it writes `statusline.sh` into the account directory and points `settings.json` at it. The statusline shows the account label, the model, the working directory, the git branch, context used, and the 5h and 7d rate-limit percentages. It also merges the default keys from `~/.config/claudex/defaults.json` into the existing `settings.json`, leaving every other key alone. The file starts with these keys:

| Key | Value |
|-----|-------|
| `attribution.commit` | `""` |
| `attribution.pr` | `""` |
| `effortLevel` | `xhigh` |
| `tui` | `fullscreen` |
| `outputStyle` | `Concise` |
| `autoMemoryEnabled` | `false` |
| `autoCompactEnabled` | `false` |
| `idleCompaction` | `false` |
| `skipDangerousModePermissionPrompt` | `true` |
| `enableAllProjectMcpServers` | `true` |
| `enableArtifact` | `false` |
| `feedbackDrafts` | `off` |
| `feedbackSurveyRate` | `0` |
| `disableBundledSkills` | `true` |
| `syncClaudeAiSkills` | `false` |
| `syncClaudeAiPlugins` | `false` |
| `includeGitInstructions` | `false` |
| `spinnerTipsEnabled` | `false` |
| `promptSuggestionEnabled` | `false` |
| `awaySummaryEnabled` | `false` |
| `env.DISABLE_AUTOUPDATER` | `1` |
| `env.ENABLE_CLAUDEAI_MCP_SERVERS` | `false` |

A key taken out of the defaults is deleted from each account the next time `configure` or the web UI applies them, but only from accounts ClaudeX wrote it to. The `statusLine` keys are always set by ClaudeX. An account whose `settings.json` is not valid JSON is skipped rather than overwritten.

The global defaults are a Claude Code plugin at `~/.config/claudex/global/`, carrying an `.lsp.json` that wires `gopls`, `pyright-langserver`, and `typescript-language-server`, plus the built-in presets extracted into `~/.config/claudex/presets/`.

`-A` configures one account instead of all of them. `-l/--label` overrides the account label in the statusline and requires `-A`; without it the label comes from the directory name, where `.claude` is `first`, `.claude2` is `second`, `.claude3` is `third`, and anything else uses the numeric suffix.

### apply

Writes the layout into the current directory:

```
AGENTS.md          base instruction block, between <!-- claudex:base --> markers
CLAUDE.md          -> AGENTS.md
.agents/skills/    agents-md, session-summary, skill-creator, write-document
.claude/skills     -> ../.agents/skills
```

`AGENTS.md` and `.agents/skills/` are the real files, following the [Agent Skills](https://agentskills.io) layout that Cursor and Codex read on their own, and the two symlinks exist because Claude Code looks for the other names.

An existing `AGENTS.md` keeps everything outside the markers: the base block is inserted or replaced in place. The same four paths and `.mcp.json` also go into `.git/info/exclude`, which is local to your clone, so the layout never appears in `git status` and never gets pushed.

A directory holding AI files ClaudeX did not create (`.mcp.json`, `.claude/`, `.agents/`, `CLAUDE.md`, or `AGENTS.md`) is refused until `--foreign` says what to do with them. `--foreign preserve` zips them into `.agents/old-ai-files.zip` and `--foreign clean` deletes them, and both then apply fresh. A directory ClaudeX already set up is re-applied without asking.

`--restore-resume` puts back the `.agents/session-resume/resume.md` that `clean-cwd` stashed for this directory.

### apply-preset

A preset is a directory holding a `preset.yaml`, an optional `AGENTS.partial.md`, an optional `skills/` directory, and an optional `mcp.json` in the `.mcp.json` shape (`{"mcpServers": {...}}`). Local presets sit under `~/.config/claudex/presets/`, and presets pulled from a repository sit under `~/.config/claudex/remote-presets/<owner>-<repo>/`. Applying one symlinks its skills into `.agents/skills/`, writes its partial as its own marked section of `AGENTS.md`, and merges its MCP servers into the project's `.mcp.json`. Re-applying replaces that section instead of appending a second copy. It also removes any skill link whose target no longer exists, which is what a skill dropped from a preset leaves behind. It needs `claudex apply` to have run first.

A preset is addressed by its directory name, and a pulled one is qualified with the repository slug so two repositories can ship the same name.

```bash
claudex apply-preset                             # lists the available presets
claudex apply-preset private                     # local, by name; several apply in order
claudex apply-preset tanq16-presets/go-strict    # pulled, by <slug>/<name>
```

`--skills` links only the skills and leaves `AGENTS.md` alone. `--agents` writes only the section and links no skills. Neither flag applies the whole preset, MCP servers included; passing one narrows the run to that half.

One preset ships in the binary. `private` carries 30 skills covering Go and Node conventions, containers, release workflows, and testing, plus the author's development, pull request, and operating rules as an `AGENTS.md` section.

The manifest keys:

| Key | Default | Description |
|-----|---------|-------------|
| `name` | the directory name | The directory a pulled preset installs into |
| `description` | empty | One line shown beside the name in the picker |
| `skills` | every directory under `skills/` holding a `SKILL.md` | Which skills to link |

### create-preset

Scaffolds `~/.config/claudex/presets/<name>/` with a `preset.yaml`, an empty `AGENTS.partial.md`, and an empty `skills/`. Names take lowercase letters, digits, and single hyphens.

### pull-preset

Clones a repository at depth 1 into a temporary directory and installs the presets it holds under `~/.config/claudex/remote-presets/<owner>-<repo>/`. Nothing tracks the clone afterwards, so pulling again is how you update.

```bash
claudex pull-preset tanq16/presets                          # every preset in the repo
claudex pull-preset https://github.com/tanq16/presets.git   # a full URL works too
claudex pull-preset tanq16/monorepo --path tools/go-strict  # one, by repo-relative path
```

Without `--path`, a preset is the repository root when it holds a `preset.yaml`, and otherwise every directory one level under the root that holds one. The whole slug directory is replaced, so a preset deleted upstream disappears on the next pull. With `--path`, only that one preset is replaced and the rest of the slug is left alone.

Cloning runs `git`, so a private repository works with whatever credentials you already have. A path on this machine is rejected rather than cloned, because a preset already here belongs in the presets directory. Removing a pulled repository means deleting its slug directory.

### clean-cwd

Removes what `apply` and `apply-preset` wrote: `.agents/`, both symlinks, the ClaudeX sections of `AGENTS.md`, and the `.git/info/exclude` block. A `.mcp.json` stays, and so does its exclude entry. The session resume file is stashed under `~/.config/claudex/resume-stash/` first, for `apply --restore-resume` to bring back.

### launch

Execs `claude` in the current directory with `--dangerously-skip-permissions`, a `--plugin-dir` pointing at the global plugin, and `CLAUDE_CONFIG_DIR` set to the account. Any inherited `CLAUDE_CONFIG_DIR` is stripped first so it cannot override that choice. The plugin is rebuilt on every launch, so language servers work without running `configure`. The directory is marked trusted in the account's `.claude.json`, so the trust dialog never appears.

| Flag | Effect |
|------|--------|
| `-A/--account` | The account to run under, `~/.claude` by default |
| `--continue` | Continue the most recent conversation in this directory |
| `-r/--remote-control <name>` | Enable Remote Control under that session name |
| `--connectors` | Load the claude.ai connectors as MCP servers |

An older conversation is picked with `claude --resume` inside the session.

### serve

A local web UI that manages the projects under one root directory and the Claude sessions running in them.

```bash
claudex serve                                   # http://127.0.0.1:8080, projects under ~/.config/claudex/projects
claudex serve --root ~/code --host 0.0.0.0 -p 9000
```

Each session is a window in a tmux session named `claudex`, running `claude --remote-control <name>` with no shell in between, so `tmux attach -t claudex` reaches the terminal at any time. Sessions outlive `serve`, since tmux is its own process. A project shows as active while its window runs and inactive otherwise, and Continue restarts it with `claude --continue`.

The new-session dialog starts from an existing directory under the root, a new empty one, or a clone from `gh`, then applies presets, picks MCP servers from the library, and launches. A project page browses and renders the project's files, takes new files and uploads, and holds restart, stop, account switch, `clean-cwd`, preset, and MCP controls. Settings covers each account's compliance with the defaults, the defaults themselves, tracked remote presets, and the MCP library, whose entries carry their credentials inline in `~/.config/claudex/mcp-library.json`.

The defaults editor lists every key in Claude Code's published settings schema (https://json.schemastore.org/claude-code-settings.json), cached at `~/.config/claudex/settings-schema.json` and refreshed from the page. A key is either active, meaning it is in your defaults, or default, meaning Anthropic's own value applies. Accept copies Anthropic's default into your defaults, and Edit sets your own value. Add key covers settings and environment variables the schema does not list yet.

`serve` has no authentication and binds to `127.0.0.1` by default. Exposing it beyond the machine is a reverse proxy's job.

### status

Per account, the 5h session window and the 7d windows as bars with their reset times. It reads the account's OAuth token from the macOS Keychain or from `.credentials.json` and queries Anthropic's usage endpoint, so an expired token shows as a prompt to open Claude Code on that account. `--json` prints the raw numbers, `-A` limits it to one account.

### switch

Moves every session of the current directory, with its history entries, out of the other accounts and into `-A/--account`, which is required.

## Notes

- **tmux is required for `serve`.** `gh` is needed only to list and clone repositories in the new-session dialog.
- **Account discovery.** `~/.claude` and every `~/.claudeN` directory whose suffix is digits, such as `~/.claude2`. Nothing else in your home directory counts as an account.
- **Preset skills are symlinks.** They point back into `~/.config/claudex/presets/`, so editing a preset changes every project that applied it. The four base skills from `apply` are real copies extracted from the binary, so re-running `apply` is what updates them.
- **Built-in presets are refreshed from the binary.** `~/.config/claudex/presets/private/` is rewritten whenever a preset command runs, so edits to it do not survive. Presets you create yourself are never touched.
- **Language server binaries are yours to install.** ClaudeX writes the `.lsp.json`, and a server whose binary is missing is skipped while the rest still start. Install commands and the `typescript@5` pin are in [docs/language-servers.md](docs/language-servers.md).
- **Another repo's agent files.** `.git/info/exclude` only reaches untracked files, so a `GEMINI.md` or `.cursor/` that the repository itself tracks stays in your working tree. [docs/foreign-agent-files.md](docs/foreign-agent-files.md) covers the sparse-checkout that removes them.
