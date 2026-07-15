---
name: review-code
description: Use when reviewing existing code against development skill best practices — orchestrates multi-agent domain reviews for thoroughness
user-invocable: true
---

# Review Code

**Review code against all development skill best practices — with sub-agent orchestration for thoroughness.**

## When to Use

Use this skill when:
- Reviewing an existing project for compliance with skill-defined patterns
- Checking a specific package or domain against best practices
- Running a full multi-agent code review

## Review Domains

Reviews are organized into 6 domains, each with its own check tables:

| Domain | Reference File | Categories | Skills |
|--------|---------------|------------|--------|
| Go Foundations | `./references/review-domain-go-foundations.md` | 1-4 (Layout, Principles, Modern Go, Logging, Utils) | go-foundations |
| Go CLI | `./references/review-domain-go-cli.md` | 5-7 (Cobra, Commands, Output Lifecycle, TUI) | go-cli |
| Go Backend & Frontend | `./references/review-domain-go-backend-frontend.md` | 8-9 (Architecture, Assets) | go-backend, go-frontend |
| Go Concurrency | `./references/review-domain-go-concurrency.md` | 10-11 (Concurrency, Pipeline) | go-concurrency |
| Infrastructure | `./references/review-domain-infra.md` | 12-15 (CI/CD, Chrome, README, Deps) | project-ci-cd, chrome-extension-basics, project-readme, go-foundations |
| Node | `./references/review-domain-node.md` | 16-18 (Foundations, Backend, Frontend) | node-foundations, node-backend, node-frontend |

> Reference paths in this skill are relative to this `SKILL.md`. Sibling skills (e.g. `go-cli`, `go-foundations`) live alongside this skill, so their files are at `../<skill-name>/SKILL.md` and `../<skill-name>/references/<file>.md`.

---

## Workflow

### Step 1: Detect Project Type

Inspect the current directory to determine what kind of project this is:

| Indicator | Project Type |
|-----------|--------------|
| `go.mod` + `cmd/`, `utils/`, no `internal/server/` | Go CLI Only |
| `go.mod` + `internal/server/static/`, no `utils/` | Go Web Only |
| `go.mod` + `internal/server/static/` **and** `utils/` | Go CLI + Web (hybrid) |
| `manifest.json` with `manifest_version` field | Chrome Extension |
| `package.json` with `"type":"module"` + `public/` + `src/`, no `go.mod` | Node Web Only |
| `go.mod` only (minimal) | Go Project (treat as CLI Only) |

If no recognizable project structure is found, report that and stop.

### Step 2: Parse Target Argument

If no target argument was provided, go to **Step 3a** (Full Review).

If a target argument was provided:

**2a. Check keyword table:**

| Keyword(s) | Domain | Subset |
|------------|--------|--------|
| `foundations`, `core`, `layout`, `logging`, `utils` | Go Foundations | All categories |
| `cli`, `cobra`, `commands`, `tui` | Go CLI | All categories |
| `backend`, `frontend`, `server`, `web` | Go Backend & Frontend | All categories |
| `concurrency`, `pipeline`, `highway`, `goroutines` | Go Concurrency | All categories |
| `cicd`, `ci`, `cd`, `docker`, `makefile` | Infrastructure | CI/CD category only |
| `chrome`, `extension` | Infrastructure | Chrome category only |
| `readme` | Infrastructure | README category only |
| `deps`, `dependencies` | Infrastructure | Deps category only |
| `infra`, `infrastructure` | Infrastructure | All categories |
| `node`, `nodejs`, `esm`, `node-backend`, `node-frontend` | Node | All categories |

If a keyword matches, go to **Step 3b** (Targeted Review) with the mapped domain and subset.

**2b. Check if package path:**

If the target contains `/`, treat it as a package path. Map to domain(s):

| Path Pattern | Domain(s) |
|-------------|-----------|
| `cmd/**` | Go CLI |
| `internal/server/static/**` | Go Backend & Frontend |
| `internal/server/**` | Go Backend & Frontend |
| `internal/highway/**` or `internal/display/**` | Go Concurrency |
| `utils/**` | Go Foundations |
| `internal/**` (anything else) | Go Foundations + Go Backend & Frontend |
| `src/**`, `public/**`, `test/**`, `package.json` | Node |
| `.github/**`, `Makefile`, `Dockerfile`, `README.md` | Infrastructure |

Go to **Step 3b** (Targeted Review) with the mapped domain(s).

**2c. Validate applicability:**

If the target maps to a domain that does not apply to the detected project type (e.g., `backend` for a Go CLI Only project), report:

```
The [domain] domain does not apply to this project type ([project type]).
```

And stop.

---

### Step 3a: Full Review (Multi-Agent Orchestration)

Determine which domains apply based on project type:

| Project Type | Applicable Domains |
|--------------|-------------------|
| Go CLI Only | Go Foundations, Go CLI, Go Concurrency (conditional), Infrastructure |
| Go Web Only | Go Foundations, Go CLI, Go Backend & Frontend, Infrastructure |
| Go CLI + Web (hybrid) | Go Foundations, Go CLI, Go Backend & Frontend, Go Concurrency (conditional), Infrastructure |
| Chrome Extension | Infrastructure |
| Node Web Only | Node, Infrastructure |

**If only one domain applies** (e.g., Chrome Extension), skip sub-agents and handle it directly — read the domain reference file, load its skills, run checks, and generate the report inline. Go to **Step 4**.

**If multiple domains apply**, launch parallel sub-agents. For each applicable domain, launch a Task with these parameters:

- `subagent_type: "general-purpose"`
- `model: "sonnet"`

Before building the prompt, resolve `[SKILLS_DIR]` to an **absolute path**: it is the directory that contains the sibling skills — i.e. the parent of this `review-code` skill directory (the directory holding this `SKILL.md`, then one level up). Sub-agents run in the target project's working directory, so they need absolute paths to locate skill files.

Use this prompt template for each sub-agent. Substitute `[SKILLS_DIR]` with its resolved absolute value, and fill in all other bracketed values:

```
You are a focused code review agent for the [DOMAIN_NAME] domain.

## Context
- Project type: [PROJECT_TYPE]
- Working directory: [CWD]
- Skills directory (absolute path, for locating skill/reference files): [SKILLS_DIR]

## Instructions

1. Read your domain check tables:
   Read the file at [SKILLS_DIR]/review-code/references/review-domain-[DOMAIN_FILE].md

2. Read the skill files listed in your domain reference header. The skill files are at:
   [SKILLS_DIR]/[SKILL_NAME]/SKILL.md
   Also read any reference files mentioned within the skills (reference files within a skill are at [SKILLS_DIR]/[SKILL_NAME]/references/).

3. Thoroughly inspect the project in [CWD] against EVERY check in your domain reference:
   - Use Read, Glob, Grep, and Bash (read-only) to verify each check
   - Only check categories marked as applicable to [PROJECT_TYPE]
   - For conditional categories, first detect if the pattern exists before deep-diving
   - Read the ACTUAL source files — do not guess or assume

4. Report findings using the output format defined in your domain reference file.

5. HARD BOUNDARY: Do not flag anything not defined in your check tables or loaded skill files.
   If you cannot cite a specific skill section, it is NOT a finding.

6. End your response with exactly:
   SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

**Launch ALL domain sub-agents in a single message** (parallel tool calls) for maximum throughput.

Go to **Step 4**.

---

### Step 3b: Targeted Review (Single Agent, Inline)

For targeted reviews, do NOT launch sub-agents. Handle the review directly:

1. Read the domain reference file: `./references/review-domain-[domain].md`
2. Load the skill files listed in the reference header (sibling skills at `../[SKILL_NAME]/SKILL.md`)
3. If a **subset** was specified (e.g., keyword `cicd` maps to only the CI/CD category within Infrastructure), check only that category — skip the rest in the reference file
4. If a **package path** was specified, narrow checks to patterns relevant within that package scope (skip file-existence checks outside the package, but still check cross-cutting concerns like error handling and logging patterns)
5. Run all applicable checks and generate the report

Go to **Step 4**.

---

### Step 4: Generate Review Report

**For full reviews with sub-agents:** After all Task calls complete, collect their outputs and combine into a unified report.

**For targeted reviews:** You already have the findings from Step 3b.

Present the final report in this format:

```
## Code Review Report: [Project Name]

**Project type:** [CLI Only | Web Only | CLI + Web | Chrome Extension | Node Web Only]
**Review scope:** [Full | Targeted: domain-name | Targeted: package-path]
**Skills checked against:** [list of loaded skills]

---

[Domain sections — each sub-agent's output, or inline findings for targeted reviews]

---

### Summary

| Category | Status | Issues |
|----------|--------|--------|
| Project Layout | PASS | 0 |
| Core Principles | ISSUES | 2 |
| Logging | PASS | 0 |
| ... | ... | ... |

**Total issues found:** N
```

### Step 5: Offer to Fix

After presenting the report, ask:

```
Would you like me to fix these issues? I can address them one category at a time.
```

If the user agrees, work through each category's issues in order, making the changes described in the "Fix" suggestions.

---

## Key Principles

- **Skills are the single source of truth** — Every finding MUST cite the specific skill and section it comes from. If no skill defines a pattern, it is NOT a finding.
- **Only check relevant categories** — Skip categories that don't apply to the detected project type.
- **Be specific** — "Missing `PrintFatal` function in printer.go" not "Utils package is incomplete."
- **Reference fixes to skill patterns** — Don't invent best practices. Every "Expected" and "Fix" must come from a loaded skill.
- **Don't flag what skills don't cover** — No findings for: missing linting, missing formatting, missing pre-commit hooks, missing documentation beyond README.
- **Detect before deep-diving** — Check if a pattern is even used before reviewing it in depth (e.g., don't review concurrency if no goroutines exist).
- **Sub-agents read actual code** — Each domain agent must Read the actual source files, not guess based on file names.

## Out of Scope (Hard Boundary)

The following are NOT defined in any loaded skill and MUST NOT be flagged as issues:

| Category | Specific Examples |
|----------|-------------------|
| Linting & Formatting | No golangci-lint, no gofmt, inconsistent formatting |
| Pre-commit | No pre-commit hooks, no husky |
| Code Quality CI | No lint/format CI steps |
| Documentation beyond README | No godoc, no changelogs, no contributing guide |
| Docker Compose | No docker-compose for development |
| Database | No migrations, no schema files |
| Dependency tooling | No dependabot, no renovate |
| Security scanning | No SAST, no container scanning |
| Code style opinions | Naming conventions not in skills, personal preferences |

**Rule of thumb:** If you are about to flag something and cannot point to a specific section in a loaded skill that defines the expected pattern, do not flag it.
