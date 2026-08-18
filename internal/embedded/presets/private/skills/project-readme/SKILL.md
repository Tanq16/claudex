---
name: project-readme
description: README structure for every project type - the centered header, badges, and the section order each kind of project uses. Use when creating a README, restructuring one, adding badges or navigation links, or documenting installation, configuration, and container behavior. Triggers on README.md, shields.io badges, .github/assets/logo.png, a capabilities table, an install section, and documenting the user a container runs as.
user-invocable: false
---

# Project README

**A centered header with a logo, badges, and jump links, then the section order for whichever kind of project this is.**

Match a project's existing README style when it already has one. Restructuring a working README to match a template is churn the reader has to re-read to discover nothing changed.

## Header

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="[PROJECT_NAME] Logo" width="200">
  <h1>[PROJECT_NAME]</h1>

  <a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/[REPO_NAME]/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/[GITHUB_USER]/[REPO_NAME]/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/[REPO_NAME]"></a><br><br>
  <a href="#section1">Section1</a> &bull; <a href="#section2">Section2</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---
```

Placeholders are replaced with the project's real name, repository, and owner. A hardcoded account in a template is a broken badge in every project that copies it.

The logo lives at `.github/assets/logo.png`, or at the frontend's own static path when the app already embeds one, so the README does not carry a second copy of the same image.

| Badge | Include when |
|---|---|
| Build status | the project has a release workflow |
| GitHub release | the project publishes versioned releases |
| Docker pulls | the project publishes a container image |

Navigation links point at sections that exist. A jump link to a heading that was renamed scrolls nowhere and is invisible in review.

## Section Order

| Project type | Sections after the header |
|---|---|
| CLI Only | Capabilities table, Installation, Usage by command, Tips and Notes |
| Web Only | Intro, Features, Screenshots, Installation and Usage, Tips and Notes |
| Node Web Only | Intro, Features, Screenshots, Installation and Usage, Configuration, Tips and Notes |
| CLI + Web | the CLI Only shape, plus a Web UI section with screenshots |
| Headless API Service | the CLI Only shape without screenshots |
| Library / Module | Intro, Installation via `go get`, API and usage, Tips and Notes |
| Chrome Extension | Intro, Features, Screenshots, Installation, Permissions, Tips and Notes |

A CLI tool leads with a capabilities table because a reader is deciding whether the tool does the thing they need. A web app leads with screenshots because a reader is deciding whether they want to look at it.

## Capabilities Table

The CLI Only opener, grouping commands so the surface is readable before any of it is explained.

```markdown
## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Files | `rename`, `bulk-rename`, `duplicates` | File management utilities |
| Network | `tunnel`, `http-server` | Network tools |
| Crypto | `encrypt`, `decrypt`, `keygen` | Cryptographic operations |
```

Each command then gets its own subsection under Usage: what it does, its invocation, its flags with defaults, and at least one worked example.

## Installation

Every install path the project actually publishes is listed, and none that it does not. Ordering is by what most readers will use.

A binary release names the platform matrix. A container run shows the port and volume mapping. Building from source states the toolchain version it needs.

## Configuration

A project that reads a config file documents the keys as a table with defaults, and says that omitted keys fall back to those defaults.

```markdown
| Key | Description | Default |
|-----|-------------|---------|
| `port` | Port the server listens on | `8080` |
| `host` | Bind address | `127.0.0.1` |
```

## Containers

A project that ships a container image documents the user and group it runs as, and what that means for a mounted volume. A reader who mounts a host directory and gets permission errors has no way to find the UID otherwise, since it is a number inside a Dockerfile they never opened.

```markdown
The container runs as UID/GID `10001:10001`, not root. A mounted volume needs to be
writable by that user:

    mkdir -p ./data && sudo chown 10001:10001 ./data
```

## Screenshots

Screenshots go inside a `<details>` block, so a long README stays scannable and a reader on a slow connection is not paying for images they did not open.

```markdown
<details>
<summary>Click to expand screenshots</summary>

![Screenshot 1](.github/assets/screenshot1.png)
*Caption for screenshot 1*

</details>
```

## Extension Security Disclaimer

A Chrome extension that handles sensitive data carries this note immediately after the introduction, because a reader deciding whether to install it needs the scope before the feature list.

```markdown
> **Note:** This extension is intended for developers and security professionals.
> Misuse for unauthorized access or data collection is not intended.
```

It applies to an extension that extracts or modifies cookies, monitors network traffic, reads authentication tokens, or captures form data. An extension that does none of those omits it, since a disclaimer on everything is a disclaimer on nothing.
