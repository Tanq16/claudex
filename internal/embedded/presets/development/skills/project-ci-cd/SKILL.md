---
name: project-ci-cd
description: Use when setting up CI/CD for projects - covers Makefile, GitHub Actions, semantic versioning, Docker images, multi-platform binaries, Node runtime-bundled tarballs and self-contained binaries, and extension packaging
user-invocable: false
---

# Project CI/CD

**Standardized CI/CD setup for projects with Makefile-driven builds and GitHub Actions.**

## When to Use

Use this skill when:
- Setting up CI/CD for a Go project
- Setting up CI/CD for a Node Web Only project
- Setting up CI/CD for a Chrome extension
- Adding release automation to an existing project
- Configuring Docker builds and multi-platform binaries

**Related skills:**
- `go-foundations` - Go project layout
- `node-foundations` - Node Web Only project layout
- `chrome-extension-basics` - Chrome extension structure

---

## Start here — required reading

This skill branches by stack. Read the **Always** files for a Go/extension project before touching its CI/CD; for a Node Web Only project, read the Node files instead. A subagent may read any of these if you delegate that work.

**Always (Go / Chrome extension):**
- `./references/makefile-template.md` — the Makefile with asset management and build targets
- `./references/release-workflow.md` — the GitHub Actions release automation

**When adding a Dockerfile (Go):**
- `./references/dockerfile-template.md` — two-stage Docker build

**When the project is Node Web Only (read these instead of the Go files):**
- `./references/node-makefile.md` — Node Makefile: vendor assets, build native addon, verify, assemble
- `./references/node-release.md` — Node release: binary vs tarball, Bun/SEA, Debian-slim Docker, release matrix

## CI/CD Files Checklist

### Go CLI Only Projects

| File | Purpose |
|------|---------|
| `Makefile` | Build logic (build, build-all, clean, version — NO docker targets) |
| `.github/workflows/release.yaml` | Automated releases: binaries only, NO docker job |

**No Dockerfile, no docker-compose, no docker targets in Makefile.**

### Go Web Only Projects

> **CLI + Web hybrids** use this same Web Only CI/CD (Docker + multi-platform binaries). A **Headless API Service** uses it too but drops the frontend-asset targets (`make assets` / `make verify-assets`).

| File | Purpose |
|------|---------|
| `Makefile` | All build logic (assets, build, docker, version) |
| `Dockerfile` | Two-stage build for containerized deployment |
| `.github/workflows/release.yaml` | Automated releases: docker + binaries |
| `docker-compose.yaml` | Simple compose for running the container (optional) |

### Node Web Only Projects

> The only Node type for now — an HTTP + WebSocket server that also serves a vendored SPA frontend. Node CLI Only, CLI + Web, and Library are future/out-of-scope. The release artifact is a **runtime-bundled tarball** (native-addon apps, the default) or a **single self-contained binary** (pure-JS apps) — see `./references/node-release.md` for the decision rule.

| File | Purpose |
|------|---------|
| `Makefile` | Vendor assets, build native addon from source, verify, assemble artifact (see `node-makefile.md`) |
| `scripts/bundle.sh` | Assemble the per-platform runtime-bundled tarball (tarball path) |
| `Dockerfile` | Two-stage `debian:*-slim` (glibc) build — NOT Alpine (musl breaks glibc addons) |
| `.github/workflows/release.yaml` | Tag-triggered matrix release (Linux x64/arm64, macOS x64/arm64; no Windows) |

### Chrome Extensions

| File | Purpose |
|------|---------|
| `Makefile` | Build zip for distribution |
| `.github/workflows/release.yaml` | Automated releases on push to main |

**Do NOT create:**
- Separate build scripts (exception: a Node Web Only tarball release uses a single `scripts/bundle.sh`)
- Multiple Dockerfiles
- Additional workflow files beyond release

---

## Secrets Required

Configure these secrets in GitHub repository settings:

| Secret | Purpose | Project Type |
|--------|---------|--------------|
| `DOCKER_ACCESS_TOKEN` | Push images to Docker Hub | **Web Only / CLI + Web** |

The `GITHUB_TOKEN` is automatically available - no configuration needed. CLI Only projects need no additional secrets. A Node Web Only release that only ships tarballs/binaries needs no extra secrets either; add `DOCKER_ACCESS_TOKEN` only if it also pushes a Docker image.

---

## Workflow

### Step 1: Create Makefile

**Go / Chrome extensions:** Use `./references/makefile-template.md`.

**Node Web Only:** Use `./references/node-makefile.md` instead (vendors assets from `node_modules`, builds the native addon from source, verifies it, assembles the artifact) — the Go build/ldflags material does not apply.

**CLI Only projects (Go):** Use the CLI Only Template (no assets, no docker targets).

**Web Only projects with frontend assets (Go):** Use the Web Only Template with full `assets`, `verify-assets`, `docker-build`, and `docker-push` targets.

**Web Only projects without frontend assets (Go):** Use the Web Only Template but remove `assets` and `verify-assets` targets.

**Customize these values (Go):**
- `APP_NAME` - your project name
- Asset versions - update to latest as needed (**Web Only / CLI + Web**)
- `STATIC_DIR` paths - match your project structure (**Web Only / CLI + Web**)
- Version variable path in `-ldflags` - match your cmd package path

### Step 2: Create Dockerfile (Web Only)

**CLI Only projects skip this step entirely — they do not have a Dockerfile.**

**Node Web Only:** Use the two-stage `debian:*-slim` (glibc) Dockerfile in `./references/node-release.md` — **not** the Go Alpine template. Alpine's musl libc will not load the official glibc Node build or glibc-compiled native addons. The rest of this step (Go Alpine template) does not apply to Node.

Use `./references/dockerfile-template.md` to create the Dockerfile (Go).

**For Web Only projects with frontend assets:**
- Include `make assets` in the build stage

**For Web Only projects without frontend assets:**
- Use the Minimal Template (no `make assets` step)

**Customize:**
- Go version (currently 1.26)
- Exposed port if different from 8080
- Default CMD arguments for your app

### Step 3: Create Release Workflow

**Go / Chrome extensions:** Use `./references/release-workflow.md` to create `.github/workflows/release.yaml`.

**Customize (Go):**
- Go version in the setup step
- Docker Hub username (if different)

**Node Web Only:** Use the tag-triggered matrix workflow sketch in `./references/node-release.md`. It compiles the native addon from source per platform (`npm_config_build_from_source=true`) on arch-native runners, runs `scripts/bundle.sh`, smoke-tests the tarball with a scrubbed PATH, and attaches per-platform tarballs to the release. For a pure-JS app, swap the bundle step for `bun build --compile` or `node --build-sea` and upload the binary.

### Step 4: Create docker-compose.yaml (Web Only, Optional)

**CLI Only projects skip this step — they do not use Docker.**

For easy local deployment of Web Only projects:

```yaml
services:
  [app-name]:
    image: [GITHUB_USER]/[app-name]:latest
    container_name: [app-name]
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

---

## Semantic Versioning

Version bumps are automatic based on commit message:

| Commit Message Contains | Version Bump | Example |
|------------------------|--------------|---------|
| (nothing special) | Patch | `v1.0.0` → `v1.0.1` |
| `[minor-release]` | Minor | `v1.0.1` → `v1.1.0` |
| `[major-release]` | Major | `v1.1.0` → `v2.0.0` |

**Usage:**
```bash
git commit -m "Add new feature [minor-release]"
git commit -m "Breaking API change [major-release]"
git commit -m "Fix bug in handler"  # patch release
```

---

## Build Targets Reference

| Target | Description | Project Type |
|--------|-------------|--------------|
| `make help` | Show all available targets | All |
| `make assets` | Download frontend assets | **Web Only / CLI + Web** |
| `make verify-assets` | Verify required assets exist | **Web Only / CLI + Web** |
| `make clean` | Remove built binaries and assets | All |
| `make build` | Build for current platform | All |
| `make build-for GOOS=linux GOARCH=amd64` | Build for specific platform | All |
| `make build-all` | Build all platform binaries | All |
| `make docker-build` | Build Docker image | **Web Only / CLI + Web** |
| `make docker-push` | Build and push Docker image | **Web Only / CLI + Web** |
| `make setup` | Install deps, vendor assets, verify addon | **Node Web Only** |
| `make vendor` | Vendor JS libs + woff2 fonts into `public/` | **Node Web Only** |
| `make verify` | Prove the native addon loads and runs | **Node Web Only** |
| `make bundle` | Assemble runtime-bundled tarball | **Node Web Only** |
| `make binary` | Compile single self-contained binary (pure-JS) | **Node Web Only** |
| `make version` | Calculate next version from commits | All |

---

## References

This skill uses the following reference files:

| File | Type | Purpose |
|------|------|---------|
| `./references/makefile-template.md` | Template | Complete Makefile with asset management and build targets (Go + Extensions) |
| `./references/dockerfile-template.md` | Template | Two-stage Docker build with instructions (Go) |
| `./references/release-workflow.md` | Workflow | GitHub Actions release automation (Go) |
| `./references/node-makefile.md` | Template | Node Web Only Makefile: vendor assets, build native addon, verify, assemble artifact |
| `./references/node-release.md` | Workflow | Node Web Only release: binary vs tarball decision, Bun/SEA, Debian-slim Docker, release matrix |

### Using References

- **makefile-template**: Copy and customize for your project. Remove assets section for CLI-only projects.
- **dockerfile-template**: Copy and adjust Go version, ports, and CMD as needed.
- **release-workflow**: Copy directly, update Go version if needed.
- **node-makefile**: For Node Web Only. Copy, swap `[APP_NAME]`, drop the native-addon/`verify` targets if pure-JS.
- **node-release**: For Node Web Only. Pick the binary or tarball path, then copy the matching Docker + workflow.
