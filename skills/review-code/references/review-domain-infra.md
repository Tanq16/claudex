# Review Domain: Infrastructure

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web, Chrome Extension **Skills to load** (paths relative to the plugin root provided in the sub-agent context):
- `../../project-ci-cd/SKILL.md`
- `../../chrome-extension-basics/SKILL.md` (Chrome Extension projects only)
- `../../project-readme/SKILL.md`
- `../../go-foundations/SKILL.md` (Dependency Selection — for Category 15)

---

## Category 12: CI/CD Configuration (project-ci-cd)

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web, Chrome Extension

**Common checks (all Go project types):**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Makefile exists | `Makefile` in project root | Glob for Makefile |
| Makefile targets | `build`, `build-all`, `clean`, `version`, `help` targets present | Read Makefile, check target list |
| No lint/format targets | Makefile does NOT have `lint`, `fmt`, `vet` targets | Read Makefile, flag if these exist |
| Release workflow | `.github/workflows/release.yaml` exists, triggers on push to main | Read workflow file |
| Semantic versioning | Release workflow uses commit message parsing for version bumps (`[minor-release]`, `[major-release]`) | Read release workflow version logic |
| ldflags version injection | Build commands use `-ldflags "-X cmd.AppVersion=..."` or equivalent | Grep for ldflags in Makefile |
| APP_NAME variable | Makefile defines `APP_NAME` variable | Read top of Makefile |

**CLI Only additional checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| No Dockerfile | `Dockerfile` does NOT exist in project root | Glob for Dockerfile, flag if present |
| No docker Makefile targets | Makefile has NO `docker-build` or `docker-push` targets | Read Makefile, flag if docker targets exist |
| No docker job in workflow | Release workflow has NO `docker` job | Read release workflow, flag if docker job exists |

**Web Only / CLI + Web additional checks:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Dockerfile exists | Two-stage build: Go builder stage + minimal runtime stage (e.g., `alpine`) | Read Dockerfile |
| Docker Makefile targets | Makefile has `docker-build` and `docker-push` targets | Read Makefile, check for docker targets |
| Docker job in workflow | Release workflow has `docker` job for building and pushing images | Read release workflow |
| Assets targets (if frontend) | `assets` and `verify-assets` targets for web projects with frontend | Read Makefile, check target list |

---

## Category 13: Chrome Extension Structure (chrome-extension-basics)

**Applies to:** Chrome Extension only

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| Manifest V3 | `manifest_version: 3` in `manifest.json` (NOT V2) | Read `manifest.json` |
| Required manifest fields | `name`, `version`, `description`, `icons`, `action`, `permissions` present | Read `manifest.json` |
| Icon sizes | `icons/` directory with 16x16, 32x32, 48x48, 128x128 PNG files | Glob for icon files |
| Popup directory | `popup/popup.html`, `popup/popup.css`, `popup/popup.js` | Glob for popup files |
| Catppuccin popup theme | `popup.css` uses Catppuccin Mocha CSS variables (--base, --text, --surface0, etc.) | Read `popup.css` |
| Content scripts | If used, in `content/content.js` with IIFE wrapper and message listener | Read content script |
| Background service worker | If used, in `background/service-worker.js` with `onInstalled` and `onMessage` handlers | Read service worker |
| Minimum permissions | Only necessary permissions declared; `activeTab` preferred over broad `tabs` | Read `manifest.json` permissions |

---

## Category 14: README Structure (project-readme)

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web, Chrome Extension

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| README exists | `README.md` in project root | Glob for README.md |
| Centered header | `<div align="center">` with logo image and project name | Read top of README.md |
| Badges | Build status badge and GitHub Release badge present | Grep for badge image URLs |
| Navigation links | Anchor links to main sections | Read header section |
| Correct template type | CLI Only, Web Only, CLI + Web, or Chrome Extension template used based on project type | Compare README structure to project-readme templates |
| Security disclaimer (extensions) | If extension handles sensitive data, security note is present | Read README for security disclaimer |

---

## Category 15: Dependencies (go-foundations: Dependency Selection)

**Applies to:** Go CLI Only, Go Web Only, Go CLI + Web, Chrome Extension

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| go.mod dependencies current | All direct Go dependencies at latest stable versions | Read `go.mod`, search for latest versions of each dependency |
| JS/CSS dependencies current | CDN imports and downloaded assets at latest stable versions | Read HTML files and Makefile for version strings |
| No unnecessary dependencies | Every dependency in `go.mod` is actually imported somewhere | Cross-reference `go.mod` requires with actual imports |
| Preferred packages used | Using recommended packages from go-foundations (zerolog not logrus, cobra not urfave/cli, charm.land/bubbletea/v2 not termui). Non-pre-approved packages are fine when they fill a genuine need with no reasonable stdlib alternative; only flag when a better or stdlib-based option exists. | Read `go.mod` for dependency choices; for non-pre-approved packages, evaluate whether they are justified before flagging |

**CLI Only additional check:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| CLI packages present | `zerolog`, `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2` expected in `go.mod` | Read `go.mod`, verify these are present |

**Web Only / CLI + Web additional check:**

| Check | Expected Pattern | How to Verify |
|-------|-----------------|---------------|
| No CLI-only packages | `zerolog`, `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2` should NOT be in `go.mod` | Read `go.mod`, flag if these are present |

---

## Output Format

Report findings in this exact format:

```
## Domain: Infrastructure

### [PASS] Category Name (source-skill)

All checks passed.

### [ISSUES] Category Name (source-skill)

1. **[Issue title]** (source-skill: section)
   - **Current:** [what the code does now]
   - **Expected:** [what the skill says it should do]
   - **Fix:** [specific action to take]

### [SKIP] Category Name (source-skill)

Not applicable to this project type.
```

End your response with exactly:
```
SUMMARY_LINE: categories_checked=N pass=N issues=N skipped=N total_issues=N
```

---

## Out of Scope (Hard Boundary)

Do NOT flag any of the following — they are not defined in any loaded skill:

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

**Rule:** If you cannot cite a specific section in a loaded skill for a finding, do not report it.
