# Node Makefile Template

Makefile for **Node Web Only** projects: vendor the frontend from `node_modules`, compile any native addon from source, verify it, and assemble the release artifact (runtime-bundled tarball or compiled binary). Pairs with `node-release.md`.

The `verify` target and the native-addon build are only needed when the app has a native addon (e.g. `node-pty`). A pure-JS app drops both and uses a plain `npm ci`.

## Node Web Only Template

```makefile
.PHONY: help setup vendor verify clean bundle binary version

# =============================================================================
# Variables
# =============================================================================
APP_NAME := [APP_NAME]
VERSION ?= dev-build
NODE_VERSION := 24.17.0

PUBLIC_DIR := public
VENDOR := $(PUBLIC_DIR)/vendor
FONTS_DIR := $(PUBLIC_DIR)/fonts

# The Nerd Font variant ships the glyphs; "Mono" keeps single-cell icon advances.
NERDFONT_URL := https://github.com/ryanoasis/nerd-fonts/releases/download/v3.2.1/JetBrainsMono.zip
# `uv tool run` is the exact equivalent of uvx, which is not always installed alongside uv.
UVX := uv tool run

CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m

# =============================================================================
# Help
# =============================================================================
help: ## Show this help
	@echo "$(CYAN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

# =============================================================================
# Setup + native addon build
# =============================================================================
setup: node_modules vendor verify ## Install deps, vendor assets, verify native addon

# Build the native addon from source: required on Linux (no prebuilt) and avoids the
# broken macOS spawn-helper. PYTHON points node-gyp at uv's managed interpreter.
# Run make from a shell where fnm has activated .node-version.
node_modules: package-lock.json
	@uv python install
	npm_config_build_from_source=true PYTHON="$$(uv python find)" npm ci
	@touch node_modules

# =============================================================================
# Vendor frontend assets from node_modules into public/
# =============================================================================
vendor: $(VENDOR)/xterm.js $(VENDOR)/xterm.css $(FONTS_DIR)/inter-400.woff2 $(FONTS_DIR)/inter-600.woff2 $(FONTS_DIR)/JetBrainsMonoNerdFontMono-Regular.woff2 ## Vendor JS libs + woff2 fonts

$(VENDOR)/xterm.js: node_modules
	@mkdir -p $(VENDOR)
	cp node_modules/@xterm/xterm/lib/xterm.js $@

$(VENDOR)/xterm.css: node_modules
	@mkdir -p $(VENDOR)
	cp node_modules/@xterm/xterm/css/xterm.css $@

$(FONTS_DIR)/inter-400.woff2: node_modules
	@mkdir -p $(FONTS_DIR)
	cp node_modules/@fontsource/inter/files/inter-latin-400-normal.woff2 $@

$(FONTS_DIR)/inter-600.woff2: node_modules
	@mkdir -p $(FONTS_DIR)
	cp node_modules/@fontsource/inter/files/inter-latin-600-normal.woff2 $@

# The Nerd Font variant is not on npm; download the release and compress ttf -> woff2.
# Multi-target pattern rule ('%' matches the literal '.') so make 3.81 runs the recipe once for both weights.
$(FONTS_DIR)/JetBrainsMonoNerdFontMono-Regular%woff2 $(FONTS_DIR)/JetBrainsMonoNerdFontMono-Bold%woff2:
	@mkdir -p $(FONTS_DIR)
	@set -e; tmp="$$(mktemp -d)"; trap 'rm -rf "$$tmp"' EXIT; \
	curl -fL -o "$$tmp/JetBrainsMono.zip" "$(NERDFONT_URL)"; \
	unzip -q -j "$$tmp/JetBrainsMono.zip" \
	  JetBrainsMonoNerdFontMono-Regular.ttf JetBrainsMonoNerdFontMono-Bold.ttf -d "$$tmp"; \
	$(UVX) --from "fonttools[woff]" fonttools ttLib.woff2 compress \
	  -o $(FONTS_DIR)/JetBrainsMonoNerdFontMono-Regular.woff2 "$$tmp/JetBrainsMonoNerdFontMono-Regular.ttf"; \
	$(UVX) --from "fonttools[woff]" fonttools ttLib.woff2 compress \
	  -o $(FONTS_DIR)/JetBrainsMonoNerdFontMono-Bold.woff2 "$$tmp/JetBrainsMonoNerdFontMono-Bold.ttf"

# =============================================================================
# Verify native addon (spawn it and assert a known output)
# =============================================================================
verify: ## Prove the native addon loads and runs
	@node --input-type=module -e "import pty from 'node-pty'; const t=pty.spawn('/usr/bin/env',['sh','-c','echo pty-ok'],{name:'xterm-256color',cols:120,rows:36,env:process.env}); let o=''; t.onData(d=>{o+=d;}); t.onExit(e=>{const ok=o.includes('pty-ok')&&e.exitCode===0; console.log(ok?'verify: node-pty OK':'verify: node-pty FAILED'); process.exit(ok?0:1);});"
	@echo "$(GREEN)Native addon verified$(NC)"

# =============================================================================
# Release artifacts
# =============================================================================
bundle: vendor ## Assemble runtime-bundled tarball for the host platform (see node-release.md)
	@bash scripts/bundle.sh "$$(node -p 'process.platform')" "$$(node -p 'process.arch')"
	@echo "$(GREEN)Bundle assembled in dist/$(NC)"

# Pure-JS apps only (no native addon): one binary via Bun. Cross-compile by changing --target.
binary: vendor ## Compile a single self-contained binary (pure-JS apps)
	@bun build ./bin/$(APP_NAME).js --compile --minify --outfile dist/$(APP_NAME)
	@echo "$(GREEN)Built: dist/$(APP_NAME)$(NC)"

clean: ## Remove node_modules and vendored assets
	@rm -rf node_modules $(VENDOR) $(FONTS_DIR)/*.woff2 dist
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Version (shared semver convention — identical to the Go makefile-template.md
# target; kept here only so this Makefile runs standalone)
# =============================================================================
version: ## Calculate next version from commit message
	@LATEST_TAG=$$(git tag --sort=-v:refname | head -n1 || echo "0.0.0"); \
	LATEST_TAG=$${LATEST_TAG#v}; \
	MAJOR=$$(echo "$$LATEST_TAG" | cut -d. -f1); \
	MINOR=$$(echo "$$LATEST_TAG" | cut -d. -f2); \
	PATCH=$$(echo "$$LATEST_TAG" | cut -d. -f3); \
	MAJOR=$${MAJOR:-0}; MINOR=$${MINOR:-0}; PATCH=$${PATCH:-0}; \
	COMMIT_MSG="$$(git log -1 --pretty=%B)"; \
	if echo "$$COMMIT_MSG" | grep -q "\[major-release\]"; then \
		MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0; \
	elif echo "$$COMMIT_MSG" | grep -q "\[minor-release\]"; then \
		MINOR=$$((MINOR + 1)); PATCH=0; \
	else \
		PATCH=$$((PATCH + 1)); \
	fi; \
	echo "v$${MAJOR}.$${MINOR}.$${PATCH}"
```

---

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your application name |
| `@xterm/xterm` copies | Whatever JS libs your SPA vendors from `node_modules` |
| `node-pty` in `verify` | Your actual native addon (drop the whole target if pure-JS) |
| `NODE_VERSION` | Pinned Node version, matching `.node-version` |

### Managed Toolchain

- **fnm** pins Node to `.node-version` — run `make` from a shell where fnm has activated it.
- **uv** provides both the Python that node-gyp needs (`PYTHON="$(uv python find)"`) and the `fonttools` used for woff2 compression (`uv tool run`). No global Python or pip install required.
- `npm_config_build_from_source=true` forces node-gyp to compile the addon rather than pulling a prebuilt binary.

### Fonts (shared vendored set)

Both fonts are woff2-only and self-hosted — **no Google Fonts CDN at runtime**:

- **Inter** (body/UI) — copied from `node_modules/@fontsource/inter`.
- **JetBrains Mono Nerd Font** (mono/code, the glyph-bearing Nerd Font variant) — downloaded from the nerd-fonts release and compressed ttf → woff2.

The matching `@font-face` declarations live in `public/css/inter.css` and `public/css/jetbrains-mono.css` (`font-family: 'Inter'` and `font-family: 'JetBrains Mono'`), authored per the `node-frontend` skill. This is the same font set and `@font-face` pattern the Go frontend vendors; only the static-root path differs (`public/` vs `internal/server/static/`).

### Release Targets

- `make bundle` → runtime-bundled tarball (native-addon apps; the always-works path).
- `make binary` → single self-contained binary via Bun (pure-JS apps only).

See `node-release.md` for the full two-path decision rule, the `scripts/bundle.sh` assembly, Node SEA as a V8 alternative to Bun, Docker base-image guidance, and the release workflow.
