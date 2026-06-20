# GitHub Actions Release Workflow

Automated release workflow triggered on push to main. Creates versioned releases with Docker images and multi-platform binaries.

## CLI + Web Workflow (Docker + Binaries)

```yaml
name: Release

on:
  push:
    branches: [main]

permissions:
  contents: write
  packages: write

jobs:
  # ===========================================================================
  # Step 0: Run unit tests (gates the release)
  # ===========================================================================
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Download Assets
        run: make assets  # embed.FS needs static/ populated to compile

      - name: Run Tests
        run: go test ./...

  # ===========================================================================
  # Step 1: Calculate version and create draft release
  # ===========================================================================
  create-release:
    needs: test
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
      release_created: ${{ steps.create_release.outputs.release_created }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Calculate Version
        id: version
        run: |
          NEW_VERSION=$(make -s version)
          echo "New version: $NEW_VERSION"
          echo "version=$NEW_VERSION" >> "$GITHUB_OUTPUT"

      - name: Create Draft Release
        id: create_release
        run: |
          gh release create "${{ steps.version.outputs.version }}" \
            --title "Release ${{ steps.version.outputs.version }}" \
            --draft \
            --notes "[APP_NAME] ${{ steps.version.outputs.version }}" \
            --target ${{ github.sha }}
          echo "release_created=true" >> "$GITHUB_OUTPUT"
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Step 2a: Build and push Docker image (parallel)
  # ===========================================================================
  docker:
    needs: create-release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Login to Docker Hub
        uses: docker/login-action@v3
        with:
          username: [GITHUB_USER]
          password: ${{ secrets.DOCKER_ACCESS_TOKEN }}

      - name: Build and Push
        run: make docker-push VERSION=${{ needs.create-release.outputs.version }}

  # ===========================================================================
  # Step 2b: Build binaries (parallel matrix)
  # ===========================================================================
  binaries:
    needs: create-release
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [linux, darwin]
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Download Assets
        run: make assets

      - name: Build Binary
        run: make build-for GOOS=${{ matrix.os }} GOARCH=${{ matrix.arch }} VERSION=${{ needs.create-release.outputs.version }}

      - name: Upload Release Asset
        run: |
          BINARY=$(ls *-${{ matrix.os }}-${{ matrix.arch }} 2>/dev/null | head -1)
          gh release upload "${{ needs.create-release.outputs.version }}" \
            "$BINARY" \
            --clobber
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Step 3: Publish release
  # ===========================================================================
  publish:
    needs: [create-release, docker, binaries]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Publish Release
        run: gh release edit "${{ needs.create-release.outputs.version }}" --draft=false
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Cleanup on failure
  # ===========================================================================
  cleanup-on-failure:
    needs: [create-release, docker, binaries, publish]
    if: always() && (needs.docker.result == 'failure' || needs.binaries.result == 'failure' || needs.publish.result == 'failure') && needs.create-release.outputs.release_created == 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Delete Draft Release
        run: |
          echo "Cleaning up draft release due to workflow failure"
          gh release delete "${{ needs.create-release.outputs.version }}" --yes
        env:
          GH_TOKEN: ${{ github.token }}
```

---

## CLI Only Workflow (Binaries Only, No Docker)

Complete standalone workflow for CLI Only projects. No docker job, no "Download Assets" step.

```yaml
name: Release

on:
  push:
    branches: [main]

permissions:
  contents: write
  packages: write

jobs:
  # ===========================================================================
  # Step 0: Run unit tests (gates the release)
  # ===========================================================================
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Run Tests
        run: go test ./...

  # ===========================================================================
  # Step 1: Calculate version and create draft release
  # ===========================================================================
  create-release:
    needs: test
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
      release_created: ${{ steps.create_release.outputs.release_created }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Calculate Version
        id: version
        run: |
          NEW_VERSION=$(make -s version)
          echo "New version: $NEW_VERSION"
          echo "version=$NEW_VERSION" >> "$GITHUB_OUTPUT"

      - name: Create Draft Release
        id: create_release
        run: |
          gh release create "${{ steps.version.outputs.version }}" \
            --title "Release ${{ steps.version.outputs.version }}" \
            --draft \
            --notes "[APP_NAME] ${{ steps.version.outputs.version }}" \
            --target ${{ github.sha }}
          echo "release_created=true" >> "$GITHUB_OUTPUT"
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Step 2: Build binaries (matrix)
  # ===========================================================================
  binaries:
    needs: create-release
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [linux, darwin]
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Build Binary
        run: make build-for GOOS=${{ matrix.os }} GOARCH=${{ matrix.arch }} VERSION=${{ needs.create-release.outputs.version }}

      - name: Upload Release Asset
        run: |
          BINARY=$(ls *-${{ matrix.os }}-${{ matrix.arch }} 2>/dev/null | head -1)
          gh release upload "${{ needs.create-release.outputs.version }}" \
            "$BINARY" \
            --clobber
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Step 3: Publish release
  # ===========================================================================
  publish:
    needs: [create-release, binaries]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Publish Release
        run: gh release edit "${{ needs.create-release.outputs.version }}" --draft=false
        env:
          GH_TOKEN: ${{ github.token }}

  # ===========================================================================
  # Cleanup on failure
  # ===========================================================================
  cleanup-on-failure:
    needs: [create-release, binaries, publish]
    if: always() && (needs.binaries.result == 'failure' || needs.publish.result == 'failure') && needs.create-release.outputs.release_created == 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Delete Draft Release
        run: |
          echo "Cleaning up draft release due to workflow failure"
          gh release delete "${{ needs.create-release.outputs.version }}" --yes
        env:
          GH_TOKEN: ${{ github.token }}
```

---

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your application name (for release notes) |

### Workflow Structure

The workflow has 5 phases:

1. **test** - Run `go test ./...`; gates everything else (no release if tests fail)
2. **create-release** - Calculate version from commits, create draft release
3. **docker** + **binaries** - Run in parallel after release is created
4. **publish** - Make release public after all artifacts uploaded
5. **cleanup-on-failure** - Delete draft release if any job fails

Unit testing is first-class (see `go-foundations`), so the release is gated on `go test ./...`.
The CLI + Web test job runs `make assets` first because `//go:embed static` needs the `static/`
tree populated to compile.

### Version Calculation

Uses `make -s version` which reads the latest git tag and commit message:
- Default: Patch bump (`v1.0.0` → `v1.0.1`)
- `[minor-release]` in commit: Minor bump (`v1.0.1` → `v1.1.0`)
- `[major-release]` in commit: Major bump (`v1.1.0` → `v2.0.0`)

### Secrets Required

| Secret | Purpose | Project Type |
|--------|---------|--------------|
| `DOCKER_ACCESS_TOKEN` | Docker Hub push access | **CLI + Web only** |

`GITHUB_TOKEN` is automatically available via `github.token`. CLI Only projects need no additional secrets.

### Build Matrix

Default platforms:
- `linux/amd64` - Standard Linux servers
- `linux/arm64` - ARM Linux (Raspberry Pi, AWS Graviton)
- `darwin/amd64` - Intel Macs
- `darwin/arm64` - Apple Silicon Macs

To add Windows:
```yaml
matrix:
  include:
    - os: linux
      arch: amd64
    - os: linux
      arch: arm64
    - os: darwin
      arch: amd64
    - os: darwin
      arch: arm64
    - os: windows
      arch: amd64
```

### Draft Release Strategy

The workflow creates a **draft release** first, then publishes only after all artifacts succeed. This ensures:
- No partial releases with missing artifacts
- Clean rollback on failure (draft is deleted)
- Atomic release publishing
