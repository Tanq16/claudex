# Node Web Only Release

**Applies to: Node Web Only projects** (an HTTP + WebSocket server that also serves a vendored single-page frontend). There is no Node CLI Only, CLI + Web, or Library type yet — those are out of scope.

Node has two release shapes. Pick one with the decision rule below; do not ship both.

## Decision Rule

1. **Pure-JS app (no native addons) that wants a Go-like single self-contained binary** → **Path 1: compiled binary** (`bun build --compile` or Node SEA).
2. **Any native addon present (e.g. `node-pty`), or neither binary path fits cleanly** → **Path 2: runtime-bundled tarball**. This always works and is the default fallback.

Native `.node` addons are the deciding factor: neither `bun build --compile` nor Node SEA embeds a compiled native addon reliably, so an app with one takes Path 2.

---

## Path 1: Single Self-Contained Binary

Two toolchains produce a single executable that bundles a JS runtime + your server + the embedded frontend. Use whichever fits.

### Option A — `bun build --compile`

Bundles the Bun runtime, your JS, and embedded frontend assets into one cross-compilable executable. `Bun.serve` handles both HTTP and WebSocket in-process.

```bash
bun build ./bin/[APP_NAME].js --compile --minify \
  --target=bun-linux-x64-musl \
  --outfile dist/[APP_NAME]-linux-x64
```

Cross-compile every target from a single host by changing `--target`:

| Platform | `--target` |
|----------|-----------|
| Linux x64 (musl, static) | `bun-linux-x64-musl` |
| Linux arm64 (musl, static) | `bun-linux-arm64-musl` |
| macOS x64 | `bun-darwin-x64` |
| macOS arm64 | `bun-darwin-arm64` |

Embed frontend assets by importing them so Bun folds them into the binary:

```js
import index from '../public/index.html' with { type: 'file' };
```

**Honest caveat:** the resulting binary runs on Bun's engine (JavaScriptCore), **not** V8. Behaviour can differ from `node`. Develop and unit-test on real Node, but smoke-test the *compiled* binary on Bun before release.

### Option B — Node SEA (Single Executable Application)

Stays on V8. Node 25.5+ compiles in one step:

```bash
node --build-sea build/[APP_NAME].cjs --output dist/[APP_NAME]
```

SEA needs a single **CommonJS** entry, so bundle ESM → CJS with esbuild first:

```bash
esbuild bin/[APP_NAME].js --bundle --platform=node --format=cjs \
  --outfile build/[APP_NAME].cjs
```

Embed frontend assets via the `assets` map in `sea-config.json` and read them at runtime with `sea.getAsset()`:

```js
import { getAsset } from 'node:sea';
const html = getAsset('index.html', 'utf8');
```

Node SEA does **not** cross-compile — build each target on a matching runner (or arch-native runner). Older Node versions require the manual blob + `postject` inject flow instead of `--build-sea`.

**Trade-off:** Bun `--compile` cross-compiles from one host but runs on JSC; Node SEA keeps you on V8 but is clunkier and per-platform.

---

## Path 2: Runtime-Bundled Tarball (default fallback)

For native-addon apps (or when a binary won't fit): a per-platform `.tar.gz` bundling the Node runtime, the compiled native `.node` addon, the vendored frontend, and a launcher that injects `--config`. This is the always-works path.

Bundle layout produced per platform:

```
[APP_NAME]-<os>-<arch>/
├── bin/[APP_NAME]          # launcher shell script (injects --config)
├── runtime/bin/node        # downloaded Node runtime for this os/arch
└── lib/
    ├── bin/                # bin/[APP_NAME].js entry
    ├── src/                # backend
    ├── public/             # vendored frontend
    └── node_modules/       # deps incl. compiled native .node addon
```

`scripts/bundle.sh` downloads the matching Node runtime and assembles the tree. Node ships `.tar.xz` for Linux and `.tar.gz` for macOS:

```bash
#!/usr/bin/env bash
set -euo pipefail
OS="$1"; ARCH="$2"
NODE_VERSION="24.17.0"
[ "$OS" = "darwin" ] && NODE_EXT="tar.gz" || NODE_EXT="tar.xz"
NODE_PKG="node-v${NODE_VERSION}-${OS}-${ARCH}"
BUNDLE="dist/[APP_NAME]-${OS}-${ARCH}"

curl -fL "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_PKG}.${NODE_EXT}" -o node.${NODE_EXT}
tar -xf node.${NODE_EXT}
mkdir -p "$BUNDLE/runtime/bin" "$BUNDLE/lib"
cp "${NODE_PKG}/bin/node" "$BUNDLE/runtime/bin/node"

cp -R bin/. src/. public/. node_modules/. "$BUNDLE/lib/"   # .node addon + spawn-helper ride along in node_modules
cp launcher/[APP_NAME] "$BUNDLE/bin/[APP_NAME]"
chmod +x "$BUNDLE/bin/[APP_NAME]"

tar -czf "${BUNDLE}.tar.gz" -C dist "$(basename "$BUNDLE")"
```

The launcher runs the bundled Node against the bundled entry and injects the config path so the extracted tarball is self-contained (no system `node` on PATH required):

```sh
#!/bin/sh
HERE="$(cd "$(dirname "$0")/.." && pwd)"
exec "$HERE/runtime/bin/node" "$HERE/lib/bin/[APP_NAME].js" --config "$HERE/lib/config.json" "$@"
```

**Prove self-containment** in CI: extract the tarball, scrub system `node` from `PATH`, start the launcher, and curl an endpoint. If it serves with only the bundled runtime, the bundle is correct.

---

## Docker (bundled-runtime path)

Use **`debian:<codename>-slim`** (latest slim, glibc) for the runtime image — **not Alpine**. The official Node build and glibc-compiled native addons are glibc-linked; they will not load on Alpine's musl libc. Two-stage build:

```dockerfile
FROM debian:trixie-slim AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates build-essential python3 unzip && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY . .
RUN npm_config_build_from_source=true npm ci && make vendor

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app .
EXPOSE 8080
ENTRYPOINT ["node", "bin/[APP_NAME].js"]
CMD ["--config", "/data/config.json"]
```

**Alpine is the documented exception, not the default.** It is only viable with a deliberate musl matrix: a musl Node build plus every native addon recompiled against musl inside an Alpine builder. Never copy glibc artifacts (the official Node binary or glibc-compiled `.node` files) into an Alpine image — they will fail to load at runtime.

---

## GitHub Actions Release Workflow

Tag-triggered matrix over Linux x64/arm64 + macOS x64/arm64. **No Windows.** Native addons compile from source per platform (`npm_config_build_from_source=true`), and arm/Intel builds run on arch-native runners to avoid cross-compilation.

```yaml
name: release

on:
  push:
    tags: ['v*']
  workflow_dispatch:

permissions:
  contents: write

jobs:
  build:
    name: build ${{ matrix.os }}-${{ matrix.arch }}
    runs-on: ${{ matrix.runner }}
    strategy:
      fail-fast: false
      matrix:
        include:
          - { os: linux,  arch: x64,   runner: ubuntu-24.04 }
          - { os: linux,  arch: arm64, runner: ubuntu-24.04-arm }
          - { os: darwin, arch: arm64, runner: macos-14 }
          - { os: darwin, arch: x64,   runner: macos-15-intel }
    steps:
      - uses: actions/checkout@v7

      - uses: actions/setup-node@v6
        with:
          node-version: 24.17.0

      - name: Install deps (compile native addon from source)
        run: npm_config_build_from_source=true npm ci

      - name: Build bundle
        run: bash scripts/bundle.sh ${{ matrix.os }} ${{ matrix.arch }}

      - name: Smoke test the bundled tarball
        run: bash scripts/smoke-test.sh ${{ matrix.os }} ${{ matrix.arch }}

      - uses: actions/upload-artifact@v7
        with:
          name: [APP_NAME]-${{ matrix.os }}-${{ matrix.arch }}
          path: dist/[APP_NAME]-${{ matrix.os }}-${{ matrix.arch }}.tar.gz
          if-no-files-found: error

  release:
    name: release
    needs: build
    runs-on: ubuntu-24.04
    if: startsWith(github.ref, 'refs/tags/')
    steps:
      - uses: actions/download-artifact@v8
        with:
          path: dist

      - name: Create GitHub release
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh release create "$GITHUB_REF_NAME" dist/**/*.tar.gz \
            --repo "$GITHUB_REPOSITORY" \
            --title "$GITHUB_REF_NAME" \
            --notes "Self-contained [APP_NAME] release ${GITHUB_REF_NAME}"
```

**For Path 1 (compiled binary)** swap the `Build bundle` step for a `bun build --compile` (one step, all targets from one host) or `node --build-sea` (per-runner) step, and upload the binary instead of the tarball.

---

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your application name |
| `NODE_VERSION` / `.node-version` | Pinned Node version (target `>=24`) |
| `trixie` | Current Debian slim codename |

### Versioning

Semver, release markers, and version calculation are **shared with the rest of this skill** (see the Semantic Versioning section in `SKILL.md`). The Node material above is only the artifact/toolchain layer — it does not redefine those conventions. The tag-triggered variant shown here (`push: tags: ['v*']`) is an alternative to the push-to-main draft-release flow used by the Go workflows; use whichever the project already follows.

### Which Path, Restated

- No native addon + want one file → `bun build --compile` (cross-compiles, JSC) or Node SEA (per-platform, V8).
- Native addon (`node-pty`, etc.) → runtime-bundled tarball. Always works.
