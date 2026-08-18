<div align="center">
  <img src=".github/assets/logo.svg" alt="Claudex Logo" width="200">
  <h1>ClaudeX</h1>

  <a href="https://github.com/tanq16/claudex/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/claudex/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/claudex/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/claudex"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#install">Install</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#notes">Notes</a>
</div>

---

ClaudeX is a companion CLI for Claude Code that sets every account up identically, lays your instructions and skills into a project in the format each coding agent already reads, and starts a session with the right account in one command.

It exists because instructions installed globally reach every project at once, so an agent ends up reading rules written for a codebase it is not in. It is not a replacement for `claude`, which it execs straight into, and it is not a skills marketplace.

## Capabilities

What you get over plain `claude`:

| | plain `claude` | ClaudeX |
|---|---|---|
| Accounts | one `CLAUDE_CONFIG_DIR`, swapped by hand | all of them found on their own, picked at launch |
| Resuming | only the account you are already in | recent sessions from every account, in one list |
| Skills | marketplace installs, pinned versions, orphans left behind | `claudex apply` puts them in the project, read by Claude Code and by every agent following the Agent Skills layout |
| Instructions | one `CLAUDE.md` per tool, copied around | one `AGENTS.md`, symlinked to whatever each tool looks for |
| Usage limits | a dashboard in the browser | `claudex status`, all accounts at once |

It is built for juggling several Claude subscriptions, and everything except the account picker works the same on one.

| Category | Commands | What they do |
|---|---|---|
| Setup | `configure` | Every account and the shared defaults, once after installing |
| Project | `apply`, `apply-preset`, `create-preset` | The `AGENTS.md` and skills layout, and the preset bundles that stack on top of it |
| Sessions | `launch`, `switch` | Start a session with the right account, or move a project to another one |
| Accounts | `status`, `oauth-token` | Usage windows across every account, and a raw OAuth token |
| Removal | `clean-cwd` | Takes the layout back out of a project |

Run `configure` once, `apply` once per project, then `launch` every time you start work.

## Install

Download from [releases](https://github.com/tanq16/claudex/releases):

```bash
# Linux/macOS
curl -sL https://github.com/tanq16/claudex/releases/latest/download/claudex-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o claudex
chmod +x claudex
sudo mv claudex /usr/local/bin/
```

Building from source needs Go:

```bash
git clone https://github.com/tanq16/claudex
cd claudex
make build
```

## Usage

No command needs a flag. Every prompt has a flag that skips it, so supplying all of them gives a run that never prompts. `--debug` turns on structured zerolog output and `--for-ai` switches to plain prefixed text that reads prompt answers from stdin, for driving ClaudeX from a script or an agent. The two are mutually exclusive, and `launch` refuses `--for-ai` since it execs into an interactive `claude`.

### `configure`

Run this once, right after installing. With no arguments it provisions every account it discovers in a single pass:

- **Per account**: a statusline and a set of opinionated `settings.json` defaults. Existing settings and env vars are preserved and only ClaudeX's keys are merged in.
- **The global plugin**: built at `~/.config/claudex/global` and shared by every account. It carries a `.lsp.json` wiring up the Go, Python, and TypeScript language servers, and nothing else, because everything in that directory loads into every session.
- **Presets**: `~/.config/claudex/presets/`, where the built-in `private` preset is laid down from the binary and refreshed on every run. Presets of your own live here too and are never touched.

The base skills need no directory here. They come straight out of the binary when you run `apply`.

`-A <path>` targets a single account. `--label` names that account's statusline and only applies with `-A`.

```bash
claudex configure
claudex configure -A ~/.claude2 --label prod
```

Language servers ship on by default and you install the binaries yourself, which [docs/language-servers.md](docs/language-servers.md) covers.

### `apply`

Run this once in a project. It writes the layout every coding agent reads, and commits nothing to the repo:

```
AGENTS.md                   # the real file, your instructions
CLAUDE.md      -> AGENTS.md
.agents/skills/<skill>      # the base skills, straight from the binary
.claude/skills -> ../.agents/skills
```

The base set is three skills. `skill-creator` writes a new skill in the Agent Skills format and sized to the context budget. `session-summary` pauses a session into `.agents/session-resume/resume.md` and resumes from it later. `write-document` settles who a document is for and what shape that consumer needs before it is written. Everything beyond those comes from a preset.

It checks before it writes. A path already holding something else, a real `CLAUDE.md` or a `.claude/skills` directory of your own, stops the run with the full list of what to move aside, and nothing at all is written. `apply-preset` checks the skills it would link the same way.

The four paths go into `.git/info/exclude`, which is local to your clone and never pushed, so nothing shows up as a diff in someone else's repo. Outside a git repository that step is skipped silently and everything else still happens, so run `apply` again after a later `git init`. Keeping a repo's own `GEMINI.md` or `.cursor/rules/` out of your clone needs more than that, which [docs/foreign-agent-files.md](docs/foreign-agent-files.md) covers.

ClaudeX's part of `AGENTS.md` sits between `<!-- claudex:base -->` markers, so a re-apply refreshes it and leaves anything you wrote around it alone. Re-running `apply` also refreshes the base skills from the binary, which is how an upgrade reaches a project.

```bash
claudex apply
```

### `apply-preset`

A preset is a bundle of skills plus a section it adds to `AGENTS.md`. Run it after `apply`. Presets stack, so you can apply several.

```bash
claudex apply-preset                 # pick from the list, space to toggle
claudex apply-preset private         # or name them directly
claudex apply-preset private -s      # --skills: link the skills, leave AGENTS.md alone
claudex apply-preset private -a      # --agents: write the AGENTS.md section, link no skills
```

Neither flag applies both halves, which is what you want almost every time. Either one narrows the run to that half and leaves the other exactly as it was, down to the preflight check: `--agents` never reports a skill link it was not going to write.

**`private`** ships in the binary and is the author's own working set rather than a neutral default: 30 skills covering Go and Node project layout, idioms, CLI surface, servers, auth, frontends, concurrency, Makefiles, containers, releases, README, and unit testing, plus the `develop` entry point and the `review-code` audit. Its `AGENTS.md` section adds development, pull request, and operating principles, and the operating half names one particular machine's paths. Everything portable is in the base `apply` instead, so build your own preset rather than adopting this one.

Preset skills are symlinked from `~/.config/claudex/presets/` into `.agents/skills/`, one link per skill, so editing a preset reaches every project using it. Each preset's contribution to `AGENTS.md` is wrapped in `<!-- claudex:preset:<name> -->` markers and replaced in place, so applying twice changes nothing the second time.

### `create-preset`

Scaffolds a preset of your own at `~/.config/claudex/presets/<name>/`. Yours and the built-in one are discovered the same way, with no registration step:

```
~/.config/claudex/presets/<name>/
├── preset.yaml             # name and the description shown in the picker
├── AGENTS.partial.md       # the section this preset adds to AGENTS.md
└── skills/<skill>/         # the skills it brings
```

`preset.yaml` may list `skills:` explicitly. Left out, every skill under `skills/` is included. Tracking the directory in git is up to you.

```bash
claudex create-preset go-web
```

### `clean-cwd`

Removes what `apply` wrote here: `.agents/`, the `CLAUDE.md` and `.claude/skills` symlinks, ClaudeX's marked sections of `AGENTS.md`, and the `.git/info/exclude` block. Prose you wrote around those markers stays, `AGENTS.md` is deleted only when nothing of yours is left in it, and your own `.claude/settings.json` is untouched.

This is the only command that removes anything, which is why `apply` has no wipe-and-rebuild flag.

```bash
claudex clean-cwd
```

### `launch`

The one command you run to start working. Run it in your project and it asks only what it needs to, then execs straight into `claude`:

```
  MCP + Connectors
  › MCPs only
    MCPs + Connectors
    None

  enter select · esc cancel
```

Arrow keys and enter are the whole interface, and a prompt appears only when there is a real choice to make. In order, launch asks:

- **New session, or resume?** Only when this project already has sessions. Resuming lists recent ones across every account, and with a single session it skips the list and resumes it directly.
- **Which account?** Only when you have more than one.
- **MCP + connectors.** MCPs only, MCPs plus claude.ai connectors such as Gmail and Slack, or none.

The flags that skip them are `-A/--account`, `--mcp mcps|connectors|none`, and `--new`, `--resume`, or `--session <id>`. `--resume` takes the latest when there is one session and lists them otherwise.

```bash
claudex launch
claudex launch -A ~/.claude2 --mcp none --new   # never prompts
```

### `status`

Usage for every account at once: the 5-hour session window, the weekly overall window, and the weekly per-model windows (currently Fable), each with a reset countdown. One glance tells you which account has room and which is about to hit a limit.

Numbers come from Anthropic's OAuth usage API, the same source as the official dashboard. Tokens are read from the macOS Keychain, or from each account's `.credentials.json` on Linux and Windows, and refresh on their own while Claude Code is running. If one shows as expired, launch Claude Code on that account to refresh it.

```bash
claudex status
claudex status -A ~/.claude2
claudex status -j          # json
```

### `switch`

Moves the current directory's project to another account so you can pick the thread back up there. It works out which account the project lives in from its most-recent session, then moves that account's sessions for it, files and history entries both. Run bare, it switches silently when there is only one other account and asks which when there are several, and `-A/--account` names the target directly.

```bash
claudex switch
claudex switch -A ~/.claude2
```

### `oauth-token`

A Claude OAuth access token from the browser-based PKCE flow, valid one hour by default. It opens your browser to authenticate and prints the token, and only the token, to stdout, so `TOKEN=$(claudex oauth-token)` works. `--expires-in` and `--port` are there if you need them.

```bash
claudex oauth-token
```

## Notes

- **Accounts are found, never created.** ClaudeX picks up `~/.claude` and any numbered sibling (`~/.claude2`, `~/.claude3`, and so on). To add one, point Claude Code at a fresh directory and log in there with `CLAUDE_CONFIG_DIR=~/.claude2 claude`, then run `configure` again.
- **The file names are what each tool already looks for.** `AGENTS.md` sits where every other tool expects it and `CLAUDE.md` symlinks to it because Claude Code reads that name instead. Skills work the same way round: `.agents/skills/` is the [Agent Skills](https://agentskills.io) convention Cursor and Codex scan on their own, and `.claude/skills` points at it for Claude Code.
- **Language servers are not MCP.** `--mcp none` suppresses every MCP server for the session and leaves the language servers running.
- **The global plugin loads every launch.** Launching before you have run `configure` works too and lays down anything missing without touching what you have customized. Skills are not part of it, and come from `claudex apply` in the project you launch from.
