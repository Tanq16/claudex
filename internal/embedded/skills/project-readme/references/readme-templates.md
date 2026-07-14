# README Templates

Complete templates for each project type.

---

## Header Pattern (All Projects)

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://hub.docker.com/r/[GITHUB_USER]/[REPO_NAME]"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/[GITHUB_USER]/[REPO_NAME]"></a><br>
  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#screenshots">Screenshots</a> &bull; <a href="#installation-and-usage">Install & Use</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---
```

**Notes:**
- Logo at `.github/assets/logo.png` or `.github/assets/logo.svg`
- If logo embedded in frontend, reference that path instead
- Badges: CI status, Docker pulls (if applicable), GitHub release
- Navigation links to main sections

---

## Web Only Project (Web Apps, Dashboards)

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://hub.docker.com/r/[GITHUB_USER]/[REPO_NAME]"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/[GITHUB_USER]/[REPO_NAME]"></a><br>
  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#screenshots">Screenshots</a> &bull; <a href="#installation-and-usage">Install & Use</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Brief 2-3 sentence description of what the project does and who it's for.

## Features

- Feature one with brief explanation
- Feature two with brief explanation
- Feature three with brief explanation

## Screenshots

<details>
<summary>Click to expand screenshots</summary>

![Screenshot 1](path/to/screenshot1.png)
*Caption for screenshot 1*

![Screenshot 2](path/to/screenshot2.png)
*Caption for screenshot 2*

</details>

## Installation and Usage

### Docker (Recommended)

\```bash
docker run -d -p 8080:8080 [GITHUB_USER]/[REPO_NAME]
\```

### Binary

Download from [releases](https://github.com/[GITHUB_USER]/[REPO_NAME]/releases) and run:

\```bash
./[REPO_NAME] serve --port 8080
\```

### Build from Source

\```bash
git clone https://github.com/[GITHUB_USER]/[REPO_NAME]
cd [REPO_NAME]
make build
./[REPO_NAME] serve
\```

## Tips and Notes

- Tip one about usage
- Tip two about configuration
- Note about edge cases
```

---

## Node Web Only Project (Node server that serves a frontend)

A single Node process you call and it serves — an HTTP + WebSocket server that also serves an embedded/vendored SPA. Ships as a runtime-bundled tarball or a self-contained binary, not as a bare `node` invocation the user has to wire up.

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://hub.docker.com/r/[GITHUB_USER]/[REPO_NAME]"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/[GITHUB_USER]/[REPO_NAME]"></a><br>
  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#screenshots">Screenshots</a> &bull; <a href="#installation-and-usage">Install & Use</a> &bull; <a href="#configuration">Configuration</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Brief 2-3 sentence description of what the project does and who it's for.

## Features

- Feature one with brief explanation
- Feature two with brief explanation
- Feature three with brief explanation

## Screenshots

<details>
<summary>Click to expand screenshots</summary>

![Screenshot 1](path/to/screenshot1.png)
*Caption for screenshot 1*

</details>

## Installation and Usage

[PROJECT_NAME] is a single process you launch and it serves the web UI over HTTP and WebSocket. Point a browser at the address it prints.

### Self-Contained Binary

Download the binary for your platform from [releases](https://github.com/[GITHUB_USER]/[REPO_NAME]/releases) and run it:

\```bash
chmod +x [REPO_NAME]
./[REPO_NAME]
\```

### Runtime-Bundled Tarball

Download the `.tar.gz` for your platform, extract, and run the launcher — it bundles the Node runtime, native addons, and assets, and injects the bundled config:

\```bash
tar -xzf [REPO_NAME]-linux-x64.tar.gz
cd [REPO_NAME]
./[REPO_NAME]
\```

### Docker

\```bash
docker run -d -p 8080:8080 -v $(pwd)/config.json:/app/config.json [GITHUB_USER]/[REPO_NAME]
\```

### From Source

Requires Node >=24 (see `.node-version`).

\```bash
git clone https://github.com/[GITHUB_USER]/[REPO_NAME]
cd [REPO_NAME]
npm install
node bin/app.js
\```

## Configuration

[PROJECT_NAME] reads an optional `config.json` deep-merged over built-in defaults; anything you omit falls back to the default. Copy `config.example.json` and edit what you need, then pass its path:

\```bash
./[REPO_NAME] --config ./config.json
\```

| Key | Description | Default |
|-----|-------------|---------|
| `port` | Port the server listens on | `8080` |
| `host` | Bind address | `127.0.0.1` |

## Tips and Notes

- Ships as a self-contained binary or a runtime-bundled tarball — no separate Node install required for release artifacts
- Session secrets are ephemeral (regenerated on boot); only durable state is persisted to `state.json`
- Note about reverse-proxy / TLS termination if applicable
```

---

## CLI Only Project (Command-Line Tools)

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Brief description of the CLI tool.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Files | `rename`, `bulk-rename`, `duplicates` | File management utilities |
| Network | `tunnel`, `http-server` | Network tools |
| Crypto | `encrypt`, `decrypt`, `keygen` | Cryptographic operations |

## Installation

### Binary

Download from [releases](https://github.com/[GITHUB_USER]/[REPO_NAME]/releases):

\```bash
# Linux/macOS
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64; [ "$ARCH" = "aarch64" ] && ARCH=arm64
curl -sL https://github.com/[GITHUB_USER]/[REPO_NAME]/releases/latest/download/[REPO_NAME]-$(uname -s | tr '[:upper:]' '[:lower:]')-$ARCH -o [REPO_NAME]
chmod +x [REPO_NAME]
sudo mv [REPO_NAME] /usr/local/bin/
\```

### Build from Source

\```bash
git clone https://github.com/[GITHUB_USER]/[REPO_NAME]
cd [REPO_NAME]
make build
\```

## Usage

### Command Category 1

#### `command-name`

Brief description of what this command does.

\```bash
[REPO_NAME] command-name --flag value
\```

**Flags:**
- `--flag, -f` - Description of flag (default: `value`)
- `--other, -o` - Description of other flag

**Examples:**

\```bash
# Example 1
[REPO_NAME] command-name --flag value

# Example 2 with different options
[REPO_NAME] command-name --flag other-value --other something
\```

### Command Category 2

<!-- Repeat pattern for additional commands -->

## Tips and Notes

- Tip about common workflows
- Note about environment variables
- Tip about combining commands
```

---

## Chrome Extension

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#features">Features</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#permissions">Permissions</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

Brief 2-3 sentence description of what the extension does.

<!-- SECURITY DISCLAIMER - Include ONLY for sensitive extensions -->
<!-- Uncomment if extension handles cookies, traffic, credentials, etc. -->
<!--
> **Note:** This extension is intended for developers and security professionals.
> Misuse for unauthorized access or data collection is not intended.
-->

## Features

- Feature one with brief explanation
- Feature two with brief explanation
- Feature three with brief explanation

## Screenshots

<details>
<summary>Click to expand screenshots</summary>

![Screenshot 1](.github/assets/screenshot1.png)
*Caption for screenshot 1*

![Screenshot 2](.github/assets/screenshot2.png)
*Caption for screenshot 2*

</details>

## Installation

### From Release (Recommended)

1. Download the latest `.zip` from [releases](https://github.com/[GITHUB_USER]/[REPO_NAME]/releases)
2. Extract the zip file
3. Open Chrome and go to `chrome://extensions/`
4. Enable **Developer mode** (toggle in top right)
5. Click **Load unpacked** and select the extracted folder

### Build from Source

\```bash
git clone https://github.com/[GITHUB_USER]/[REPO_NAME]
cd [REPO_NAME]
make build
# Load the generated zip as unpacked extension
\```

## Permissions

This extension requires the following permissions:

| Permission | Purpose |
|------------|---------|
| `activeTab` | Access current tab when extension is clicked |
| `storage` | Save extension settings locally |

## Tips and Notes

- Tip one about usage
- Tip two about keyboard shortcuts
- Note about browser compatibility
```

---

## Placeholders Reference

| Placeholder | Replace With |
|-------------|--------------|
| `[PROJECT_NAME]` | Display name (e.g., "Kairo", "Cookie Extractor") |
| `[REPO_NAME]` | Repository name, lowercase (e.g., `kairo`, `cookie-extractor`) |
