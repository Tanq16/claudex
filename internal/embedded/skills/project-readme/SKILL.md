---
name: project-readme
description: Use when creating README files for any project type - covers header patterns, badges, structure templates for CLI tools, web apps, and Chrome extensions
user-invocable: false
---

# Project README

**Standardized README patterns for all project types.**

## When to Use

Use this skill when:
- Creating a README for a new project
- Updating an existing README to follow conventions
- Adding badges, headers, or navigation links
- Structuring documentation for CLI tools, web apps, or extensions

## README Types

| Project Type | Key Sections |
|--------------|--------------|
| CLI + Web (Web Apps, Dashboards) | Header → Intro → Features → Screenshots → Install/Usage → Tips |
| CLI Only (Command-Line Tools) | Header → Capabilities Table → Installation → Usage (by command) → Tips |
| Chrome Extension | Header → Intro → Features → Screenshots → Install → Permissions → Tips |

Project type names match the canonical taxonomy in `go-foundations` (Project Taxonomy). A Headless
API Service uses the CLI Only README shape (no screenshots); a Library / Module uses an API/usage
README focused on `go get` and exported functions.

## Common Header

Default to a centered header pattern with logo and badges (match the project's existing README style if it already has one):

```markdown
<div align="center">
  <img src=".github/assets/logo.png" alt="PROJECT_NAME Logo" width="200">
  <h1>PROJECT_NAME</h1>

  <a href="https://github.com/[GITHUB_USER]/REPO_NAME/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/[GITHUB_USER]/REPO_NAME/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/[GITHUB_USER]/REPO_NAME/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/[GITHUB_USER]/REPO_NAME"></a><br><br>
  <a href="#section1">Section1</a> &bull; <a href="#section2">Section2</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---
```

Replace the bracketed placeholders (`PROJECT_NAME`, `REPO_NAME`, `[GITHUB_USER]`) with the
project's actual name, repo, and GitHub org/user — never hardcode a specific account.

### Badge Options

| Badge | When to Include |
|-------|-----------------|
| Build Status | Recommended (if using GitHub Actions) |
| Docker Pulls | If publishing Docker images |
| GitHub Release | Recommended (for versioned releases) |

### Logo Location

- Primary: `.github/assets/logo.png` or `.github/assets/logo.svg`
- Alternative: Reference frontend static path if logo is embedded in app

## Workflow

### Step 1: Identify Project Type

| If Project Has... | Type |
|-------------------|------|
| Web UI, server, dashboard | CLI + Web |
| CLI commands, terminal tool | CLI Only |
| manifest.json, browser extension | Chrome Extension |

### Step 2: Generate README

Use `./references/readme-templates.md` to copy the appropriate template.

**Customize:**
- Replace `[PROJECT_NAME]` and `[REPO_NAME]` placeholders
- Adjust navigation links to match actual sections
- Add/remove badges based on what's applicable

### Step 3: Add Security Disclaimer (Extensions Only)

For Chrome extensions that handle sensitive data (cookies, traffic, credentials), add this note immediately after the introduction:

```markdown
> **Note:** This extension is intended for developers and security professionals. 
> Misuse for unauthorized access or data collection is not intended.
```

Only include this for extensions that:
- Extract or modify cookies
- Monitor network traffic
- Access authentication tokens
- Capture sensitive form data

## References

| File | Purpose |
|------|---------|
| `./references/readme-templates.md` | Complete templates for all project types |

See `./references/readme-templates.md` for full copy-paste templates.
