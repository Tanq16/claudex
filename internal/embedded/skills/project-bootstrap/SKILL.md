---
name: project-bootstrap
description: Use to scaffold a new project from an implementation-plan.md, or to restructure an existing project to match the conventions defined in go-foundations. Orchestrates the other Go skills to create and reorganize files and config — not for ordinary in-file Go coding (use go-foundations for that).
user-invocable: true
---

# Project Bootstrap

**Creates project skeletons from plans or restructures existing projects to follow standard patterns.**

## When to Use

Use this skill when:
- Starting a new project from an `implementation-plan.md`
- Restructuring an existing project to match standard skill-defined patterns

**Related skills (loaded as needed):**
- `go-foundations` - Project layout, utils package, modern Go, dependency selection
- `go-cli` - Cobra command setup
- `go-frontend` - Frontend structure, HTML template
- `go-backend` - Internal package patterns
- `project-ci-cd` - Makefile, Dockerfile, workflows
- `project-readme` - README template
- `chrome-extension-basics` - Extension structure

Dependency auditing mechanics live in `./references/dependency-audit.md`; the dependency-selection
judgment it applies lives in `go-foundations` ("Dependency Selection").

## Workflow

### Step 1: Assess Current Directory

Before anything else, determine what state the current directory is in. Launch a subagent (or inspect directly) to check:

1. **Check for project indicators:**
   - Does `go.mod` exist?
   - Does `package.json` exist?
   - Does `manifest.json` exist (Chrome extension)?
   - Are there `.go` files, `cmd/` or `internal/` directories?
   - Are there any source code files at all?

2. **Check for `implementation-plan.md`** in the project root

3. **Classify the state into one of three scenarios:**

| Condition | Scenario | Action |
|-----------|----------|--------|
| No source code files, `implementation-plan.md` exists | Fresh Start | Go to Step 2 |
| No source code files, no `implementation-plan.md` | No Plan | Go to Step 2a |
| Source code files exist (go.mod, .go files, etc.) | Existing Project | Go to Step 3 |

**What counts as "source code files":** `go.mod`, `*.go` files, `package.json`, `manifest.json`, `cmd/`, `internal/`, `pkg/`, or similar project structure indicators. Documentation files (`.md`), notes, or loose scripts do NOT count.

---

### Step 2: Fresh Start (Plan Exists, No Project)

This is the straightforward path: create a skeleton from the plan.

1. Read `implementation-plan.md`
2. Extract: project name, project type, commands/features, internal packages
3. Detect project type (see detection rules below)
4. Load required skills based on type
5. Present a brief summary to the user:
   ```
   I'll create a [Project Type] skeleton for "[Project Name]" with:
   - [N] CLI commands: [list]
   - [N] internal packages: [list]
   - Config files: Makefile, README, go.mod, workflows
   
   Proceed?
   ```
6. On confirmation, create the skeleton (see "Creating the Skeleton" section below)
7. After creation, present a description of what was created (see "Post-Skeleton Summary" below)
8. Proceed to implementation based on `implementation-plan.md` priorities

---

### Step 2a: No Plan Exists

If there are no source code files AND no `implementation-plan.md`:

1. Tell the user:
   ```
   No implementation-plan.md found in the project root.
   Create an implementation-plan.md first, then come back to project-bootstrap.
   ```
2. **STOP.** Do not proceed further.

---

### Step 3: Existing Project (Restructure)

When source code files already exist, the goal is to audit the project and restructure it to follow skill-defined patterns.

#### Step 3a: Launch Dependency Audit Subagent

Launch a subagent to analyze the existing project. The subagent should:

1. **Scan the project structure** - Map out all directories and files
2. **Identify all dependencies** using the audit mechanics in `./references/dependency-audit.md`:
   - Go dependencies from `go.mod`
   - JS/CSS dependencies from HTML files and Makefile
   - Any other dependency sources
3. **Search for latest stable versions** of every dependency via web requests
4. **Evaluate alternatives** - For each dependency, apply the dependency-selection judgment from `go-foundations` ("Dependency Selection"). Consider if there's a better-maintained or more standard alternative:
   - Prefer standard library where possible
   - Prefer well-known packages from the `go-foundations` skill (e.g., `zerolog`, `cobra`, `charm.land/bubbletea/v2`)
   - Only suggest alternatives if they are clearly better (more maintained, more standard, better fit)
   - Stick with current dependencies if they are reasonable choices
5. **Return a report** with:
   - Current project structure (directory tree)
   - Dependency inventory with current vs latest versions
   - Any recommended dependency changes with rationale

#### Step 3b: Analyze Against Skill Patterns

Once the dependency audit is complete, compare the existing project **strictly** against skill-defined patterns. Do NOT invent requirements that aren't in a skill.

1. **Read the relevant skills** (`go-foundations`, `go-cli`, etc.) based on what the project contains
2. **Identify gaps ONLY where a skill explicitly defines the expected pattern.** Every gap must cite its source:
   - Missing standard directories → cite `go-foundations` project layout
   - Non-standard file organization → cite `go-foundations` project layout
   - Missing config files (Makefile, workflows, README) → cite `project-ci-cd` or `project-readme`
   - Missing patterns (no printer abstractions, no logging setup) → cite `go-foundations` utils or logging sections
   - Missing Cobra patterns → cite `go-cli`
3. **If no skill defines it, it is not a gap.** Do not flag missing linting, formatting, pre-commit hooks, or any other practice not covered by a loaded skill.
4. **Identify existing code that maps to patterns** - Don't throw away working code, identify what already follows conventions

#### Step 3c: Present Restructuring Plan

Present the analysis to the user. **Every restructuring item must cite the skill that defines it.** If you cannot cite a skill, remove the item.

```
Current project analysis:
- Type: [CLI Only | CLI + Web | Chrome Extension]
- [N] Go dependencies ([M] need updates)
- [N] JS/CSS dependencies ([M] need updates)

Restructuring needed:
- [Change description] (per [skill-name]: [section])
- [Change description] (per [skill-name]: [section])
- [Change description] (per [skill-name]: [section])

Existing code preserved:
- [What maps to standard patterns already]

Proceed with restructuring?
```

**Do NOT include restructuring items for things not defined in any loaded skill.** Common examples of things to omit: linting, formatting, and pre-commit hooks.

#### Step 3d: Execute Restructuring

On confirmation:

1. **Update dependencies** to latest stable versions:
   - Go: run `go get -u ./...` followed by `go mod tidy`
   - JS/CSS CDN imports: update version numbers in HTML files and Makefile download URLs to the latest versions identified in the dependency audit
2. **Reorganize files** to match standard project layout (per `go-foundations` project layout)
3. **Add missing scaffolding** using the corresponding skills:
   - Utils package (`utils/`) → **CLI Only only** — use `go-foundations` utils section and `../go-foundations/references/utils-templates.md`
   - Makefile → use `../project-ci-cd/references/makefile-template.md` (CLI Only template for CLI Only, Full template for CLI + Web)
   - Dockerfile → **CLI + Web only** — use `../project-ci-cd/references/dockerfile-template.md`
   - GitHub Actions workflow → use `../project-ci-cd/references/release-workflow.md` (CLI Only workflow for CLI Only, CLI + Web workflow for CLI + Web)
   - README → use `../project-readme/references/readme-templates.md`
   - HTML template (if CLI + Web) → use `../go-frontend/references/html-template.md`
4. **Preserve existing business logic** - Move files, don't rewrite them
5. **Create missing config files** from skill templates listed above

After restructuring, present the post-skeleton summary and offer to continue with implementation.

---

## Detect Project Type

Project types match the canonical taxonomy in `go-foundations` (Project Taxonomy).

| If Plan/Project Contains... | Project Type |
|-----------------------------|--------------|
| Web UI, dashboard, browser, serve, frontend | CLI + Web |
| REST/gRPC API or service, no frontend | Headless API Service |
| Importable library/package, no `main`/entry point | Library / Module |
| Chrome, extension, browser extension, popup | Chrome Extension |
| CLI commands only, no UI | CLI Only |

**Default:** CLI Only (if unclear)

## Load Required Skills

| Project Type | Skills to Load |
|--------------|----------------|
| CLI Only | `go-foundations`, `go-cli`, `project-ci-cd`, `project-readme` |
| CLI + Web | All above + `go-frontend`, `go-backend` |
| Headless API Service | `go-foundations`, `go-backend`, `project-ci-cd`, `project-readme` (+ `go-cli` only if it has subcommands beyond `serve`) |
| Library / Module | `go-foundations`, `project-readme` |
| Chrome Extension | `chrome-extension-basics`, `project-ci-cd`, `project-readme` |

Headless API Service and Library / Module have no dedicated skeleton templates beyond what
`go-foundations` (Project Taxonomy + layout) describes: an API service is a CLI + Web tree without
the `static/` frontend (reuse the `go-backend` server template, drop the `embed.FS`/`handleIndex`
parts); a library has no `cmd/`, `utils/`, or entry point.

For existing project restructuring, also use the dependency audit mechanics in
`./references/dependency-audit.md` (judgment from `go-foundations`).

---

## Creating the Skeleton

### Project Structures

#### CLI Only Projects

Includes `utils/` with globals.go and printer.go. NO Dockerfile, NO Docker in CI/CD.

```
project-root/
├── .github/
│   ├── assets/
│   │   └── .keep              # Placeholder for logo
│   └── workflows/
│       ├── release.yaml       # CLI Only workflow (binaries only, no docker)
├── cmd/
│   ├── root.go                # CLI Only root command (zerolog, --debug, --for-ai, utils)
│   └── [command].go           # Skeleton for each command in plan (uses utils)
├── internal/
│   └── [package]/             # Directories for each domain in plan
│       └── [package].go       # Skeleton with types/stubs
├── utils/                     # Utils package (from go-foundations, top-level)
│   ├── globals.go
│   └── printer.go
├── .gitignore
├── go.mod
├── main.go
├── Makefile                   # CLI Only version (no assets, no docker)
└── README.md                  # CLI Only template
```

#### CLI + Web Projects

Standalone tree — NO `utils/`, includes `Dockerfile`, simplified root command. Uses `log.Printf` with manual prefixes.

```
project-root/
├── .github/
│   ├── assets/
│   │   └── .keep              # Placeholder for logo
│   └── workflows/
│       ├── release.yaml       # CLI + Web workflow (docker + binaries)
├── cmd/
│   ├── root.go                # CLI + Web root command (no debug/for-ai, no zerolog, no utils)
│   └── serve.go               # Serve command using log.Printf/log.Fatalf
├── internal/
│   ├── [package]/             # Directories for each domain in plan
│   │   └── [package].go       # Skeleton with types/stubs
│   └── server/
│       ├── server.go          # HTTP server skeleton
│       └── static/
│           ├── index.html     # HTML template (from go-frontend)
│           ├── css/
│           │   └── .keep
│           ├── fonts/
│           │   └── .keep
│           ├── fontawesome/
│           │   └── .keep
│           ├── icons/
│           │   └── .keep
│           └── js/
│               └── .keep
├── Dockerfile                 # From project-ci-cd
├── .gitignore
├── go.mod
├── main.go
├── Makefile                   # Full version (with assets and docker)
└── README.md                  # CLI + Web template
```

#### Chrome Extensions

```
extension-root/
├── .github/
│   ├── assets/
│   │   └── .keep
│   └── workflows/
│       └── release.yaml
├── background/
│   └── service-worker.js      # Skeleton
├── content/
│   └── content.js             # Skeleton
├── icons/
│   └── .keep
├── popup/
│   ├── popup.html             # Template from chrome-extension-basics
│   ├── popup.css              # Catppuccin theme
│   └── popup.js               # Skeleton
├── .gitignore
├── manifest.json              # V3 template
├── Makefile                   # Extension version
└── README.md                  # Extension template
```

### Skeleton Code Templates

#### main.go (Go projects)

```go
package main

import "github.com/[GITHUB_USER]/[PROJECT]/cmd"

func main() {
    cmd.Execute()
}
```

#### cmd/root.go

Use the CLI Only or CLI + Web template from `go-cli` based on project type. Customize with project name and brief description from plan.

#### cmd/[command].go — CLI Only

```go
package cmd

import (
    "github.com/spf13/cobra"
    u "github.com/[GITHUB_USER]/[PROJECT]/utils"
)

var [command]Cmd = &cobra.Command{
    Use:   "[command]",
    Short: "[Brief description from plan]",
    Run: func(cmd *cobra.Command, args []string) {
        // TODO: Implement
        u.PrintInfo("[command] not yet implemented")
    },
}

func init() {
    rootCmd.AddCommand([command]Cmd)
}
```

#### cmd/[command].go — CLI + Web

```go
package cmd

import (
    "log"

    "github.com/spf13/cobra"
)

var [command]Cmd = &cobra.Command{
    Use:   "[command]",
    Short: "[Brief description from plan]",
    Run: func(cmd *cobra.Command, args []string) {
        // TODO: Implement
        log.Printf("INFO [command] not yet implemented")
    },
}

func init() {
    rootCmd.AddCommand([command]Cmd)
}
```

#### internal/[package]/[package].go

```go
package [package]

// TODO: Implement
```

#### internal/server/server.go (CLI + Web)

Use the **skeleton variant** in `../go-backend/references/http-server-template.md` — the same
`embed.FS` server with concrete API handlers dropped and a `// TODO: Add API routes` marker. Do
not redefine the `embed.FS`/`fs.Sub`/`StripPrefix`/`handleIndex` boilerplate here; `go-backend`
owns it.

### Config File Templates

#### .gitignore

```
# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Build artifacts
dist/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Test
*.test
coverage.out

# Environment
.env
.env.local
```

#### go.mod

```
module github.com/[GITHUB_USER]/[PROJECT]

go 1.26
```

---

## Post-Skeleton Summary

After creating or restructuring, output varies by project type:

**CLI Only:**
```
Project [NAME] bootstrapped successfully!

Created structure:
├── .github/workflows/ (release — binaries only, no docker)
├── cmd/ (root + [N] commands: [list])
├── internal/ ([packages])
├── utils/ (globals.go, printer.go)
├── Makefile (no docker targets), README.md, go.mod, main.go

Project type: CLI Only
```

**CLI + Web:**
```
Project [NAME] bootstrapped successfully!

Created structure:
├── .github/workflows/ (release — docker + binaries)
├── cmd/ (root + serve)
├── internal/ ([packages] + server/)
├── Dockerfile
├── Makefile (with docker + assets targets), README.md, go.mod, main.go

Project type: CLI + Web (no utils/ — uses log.Printf)
```

Then **proceed to implementation** based on `implementation-plan.md` priorities. Work through the plan's components in priority order, implementing each one fully before moving to the next.

---

## Key Principles

- **Skills are the single source of truth** - ONLY suggest, create, or verify things that are explicitly defined in a loaded skill. If no skill mentions it, it does not belong. Every suggestion must trace back to a specific skill and section.
- **Assess before acting** - Always check directory state first
- **Skeleton only (fresh start)** - No business logic, just structure and stubs
- **Preserve existing code (restructure)** - Move files, don't rewrite working logic
- **Minimal TODOs** - Simple one-line markers; the plan document has the details
- **Use skill templates** - Pull from go-cli, go-frontend, project-ci-cd for actual content
- **Match plan vocabulary** - Use names from the plan for packages and commands
- **Don't over-structure** - Create only what's mentioned in the plan, not speculative packages
- **Latest stable deps** - When restructuring, always update to latest stable versions

## References

| File | Type | Purpose |
|------|------|---------|
| `./references/dependency-audit.md` | Workflow | Mechanics for discovering project dependencies and comparing them against latest stable versions (selection judgment lives in `go-foundations`) |

## Out of Scope (Hard Boundary)

**The following are NOT defined in any skill and MUST NOT be created, suggested, or flagged as missing — ever.** This applies to both fresh starts and restructuring:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | `make lint`, `make fmt`, golangci-lint, gofmt configs, `.golangci.yaml` |
| Pre-commit | Pre-commit hooks, husky, lint-staged, `.pre-commit-config.yaml` |
| Code Quality CI | CI steps for lint, format, vet, staticcheck |
| Documentation beyond README | Godoc, changelogs, contributing guides, architecture docs, wiki |
| Docker Compose | `docker-compose.yaml` for development (only optional for deployment per `project-ci-cd`) |
| Database | Migrations, schemas, ORM configs |
| Dependency tooling | Dependabot, renovate configs |
| Security scanning | SAST, container scanning, trivy |

**Rule of thumb:** If you are about to suggest something and cannot point to a specific section in a loaded skill that defines it, do not suggest it.
