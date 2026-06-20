# Dependency Audit (mechanics)

The mechanical workflow for discovering a project's dependencies and comparing them against the
latest stable versions. Used during restructuring (and on demand when auditing or planning
upgrades).

The *judgment* for which dependencies to choose — prefer stdlib, pre-approved packages, evaluate
whether a third-party dep is justified, keep versions current — lives in `go-foundations`
("Dependency Selection"). This file is only the how-to-scan-and-compare procedure.

## Dependency sources

| Source | What it contains | Where to find |
|--------|------------------|---------------|
| `go.mod` | Go module dependencies | Root of Go projects |
| `*.html` | JS/CSS CDN imports via `<script>` / `<link>` | `internal/server/static/` |
| `Makefile` | Asset download URLs with versions | Root of project |

## Step 1: Identify current dependencies

### Go dependencies

1. Read `go.mod`.
2. Extract all `require` statements; note direct vs indirect.
3. Record the current version for each.

```
require github.com/spf13/cobra v1.8.0
→ Package: github.com/spf13/cobra   Current: v1.8.0
```

### JS/CSS dependencies (CDN imports)

1. Search HTML files for `<script src="...">` and `<link href="...">`.
2. Search the Makefile for CDN download URLs.
3. Extract package name + version from each URL.

```
Common CDN patterns:
- unpkg.com/[package]@[version]/...
- cdn.jsdelivr.net/npm/[package]@[version]/...
- cdnjs.cloudflare.com/ajax/libs/[package]/[version]/...
- fonts.googleapis.com/css2?family=[font]...
```

## Step 2: Search for latest versions

**Always search online for accurate latest versions. Do not guess.**

### Go packages

Search `pkg.go.dev/[package]` (official registry) or the package's GitHub releases page.

### JS/CSS packages

| Package type | Where to check |
|--------------|----------------|
| npm packages | `npmjs.com/package/[name]` |
| Lucide | `lucide.dev` or `unpkg.com/lucide@latest` |
| Font Awesome | `fontawesome.com/changelog` |
| Tailwind CSS | `tailwindcss.com/docs` or `cdn.tailwindcss.com` |
| Dev Icons | `devicon.dev` |
| Google Fonts | `fonts.google.com` |
| Chart.js | `chartjs.org` |
| Marked.js | `marked.js.org` |
| Mermaid | `mermaid.js.org` |

Common registries: `pkg.go.dev/[import-path]`, `npmjs.com/package/[name]`,
`unpkg.com/[package]@[version]`, `cdn.jsdelivr.net/npm/[package]@[version]`,
`cdnjs.cloudflare.com/ajax/libs/[name]/[version]`.

## Step 3: Compare and report

```markdown
## Dependency Report: [Project Name]

### Go Dependencies (go.mod)

| Package | Current | Latest | Status |
|---------|---------|--------|--------|
| github.com/spf13/cobra | v1.8.0 | v1.8.1 | Update available |
| charm.land/bubbletea/v2 | v2.0.2 | v2.0.2 | Up to date |

### JS/CSS Dependencies (CDN)

| Package | Current | Latest | Source | Status |
|---------|---------|--------|--------|--------|
| Lucide | 0.300.0 | 0.469.0 | unpkg | Update available |
| Tailwind CSS | 3.4.0 | 3.4.17 | cdn.tailwindcss.com | Update available |

### Summary
- Total dependencies: X   Up to date: Y   Updates available: Z
```

## Step 4: Provide upgrade commands

### Go

```bash
go get github.com/spf13/cobra@v1.8.1   # specific package
go get -u ./...                         # update all
go mod tidy
```

### JS/CSS

For CDN imports and Makefile downloads, provide the updated URLs:

```
Current: https://unpkg.com/lucide@0.300.0/dist/umd/lucide.min.js
Updated: https://unpkg.com/lucide@0.469.0/dist/umd/lucide.min.js
```

## Notes

- Always search online for latest versions — never assume.
- Check release notes for breaking changes on major version bumps.
- Prefer stable releases over pre-release/beta.
- Go indirect dependencies update automatically with their direct parents.
- CDN versions may lag npm — verify CDN availability for the latest version.
