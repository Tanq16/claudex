---
name: project-ci-cd
description: Use when setting up CI/CD for projects - covers Makefile, GitHub Actions, semantic versioning, Docker images, multi-platform binaries, and extension packaging
user-invocable: false
---

# Project CI/CD

**Standardized CI/CD setup for projects with Makefile-driven builds and GitHub Actions.**

## When to Use

Use this skill when:
- Setting up CI/CD for a Go project
- Setting up CI/CD for a Chrome extension
- Adding release automation to an existing project
- Configuring Docker builds and multi-platform binaries

**Related skills:**
- `go-foundations` - Go project layout
- `chrome-extension-basics` - Chrome extension structure

---

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

### Chrome Extensions

| File | Purpose |
|------|---------|
| `Makefile` | Build zip for distribution |
| `.github/workflows/release.yaml` | Automated releases on push to main |

**Do NOT create:**
- Separate build scripts
- Multiple Dockerfiles
- Additional workflow files beyond release

---

## Secrets Required

Configure these secrets in GitHub repository settings:

| Secret | Purpose | Project Type |
|--------|---------|--------------|
| `DOCKER_ACCESS_TOKEN` | Push images to Docker Hub | **Web Only / CLI + Web** |

The `GITHUB_TOKEN` is automatically available - no configuration needed. CLI Only projects need no additional secrets.

---

## Workflow

### Step 1: Create Makefile

Use `./references/makefile-template.md` to create the Makefile.

**CLI Only projects:** Use the CLI Only Template (no assets, no docker targets).

**Web Only projects with frontend assets:** Use the Web Only Template with full `assets`, `verify-assets`, `docker-build`, and `docker-push` targets.

**Web Only projects without frontend assets:** Use the Web Only Template but remove `assets` and `verify-assets` targets.

**Customize these values:**
- `APP_NAME` - your project name
- Asset versions - update to latest as needed (**Web Only / CLI + Web**)
- `STATIC_DIR` paths - match your project structure (**Web Only / CLI + Web**)
- Version variable path in `-ldflags` - match your cmd package path

### Step 2: Create Dockerfile (Web Only)

**CLI Only projects skip this step entirely — they do not have a Dockerfile.**

Use `./references/dockerfile-template.md` to create the Dockerfile.

**For Web Only projects with frontend assets:**
- Include `make assets` in the build stage

**For Web Only projects without frontend assets:**
- Use the Minimal Template (no `make assets` step)

**Customize:**
- Go version (currently 1.26)
- Exposed port if different from 8080
- Default CMD arguments for your app

### Step 3: Create Release Workflow

Use `./references/release-workflow.md` to create `.github/workflows/release.yaml`.

**Customize:**
- Go version in the setup step
- Docker Hub username (if different)

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
| `make version` | Calculate next version from commits | All |

---

## References

This skill uses the following reference files:

| File | Type | Purpose |
|------|------|---------|
| `./references/makefile-template.md` | Template | Complete Makefile with asset management and build targets (Go + Extensions) |
| `./references/dockerfile-template.md` | Template | Two-stage Docker build with instructions |
| `./references/release-workflow.md` | Workflow | GitHub Actions release automation |

### Using References

- **makefile-template**: Copy and customize for your project. Remove assets section for CLI-only projects.
- **dockerfile-template**: Copy and adjust Go version, ports, and CMD as needed.
- **release-workflow**: Copy directly, update Go version if needed.
