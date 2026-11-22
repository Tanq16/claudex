.PHONY: help assets verify-assets font fontawesome pdfjs clean build build-for build-all version

# =============================================================================
# Variables
# =============================================================================
APP_NAME := claudex
MODULE := github.com/tanq16/claudex

# Build variables (set by CI or use defaults)
VERSION ?= dev-build
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Pinned asset versions. Bump deliberately, never float.
TAILWIND_VERSION    := 4.3.5
LUCIDE_VERSION      := 1.73.0
FONTAWESOME_VERSION := 7.3.1
MARKED_VERSION      := 18.2.0
HIGHLIGHTJS_VERSION := 11.12.0
DOMPURIFY_VERSION   := 3.4.18
MERMAID_VERSION     := 11.17.2
PDFJS_VERSION       := 6.4.299

STATIC_DIR := internal/server/static
JS_DIR     := $(STATIC_DIR)/js
CSS_DIR    := $(STATIC_DIR)/css
FONTS_DIR  := $(STATIC_DIR)/fonts
FA_DIR     := $(STATIC_DIR)/fontawesome
PDFJS_DIR  := $(JS_DIR)/pdfjs
STAMP      := $(STATIC_DIR)/.assets-stamp

# Google Fonts serves woff2 only to a browser-shaped User-Agent; an unrecognized
# one gets ttf, which is roughly twice the bytes.
UA := Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36

# Console colors
CYAN := \033[0;36m
GREEN := \033[0;32m
NC := \033[0m

# =============================================================================
# Help
# =============================================================================
help: ## Show this help
	@echo "$(CYAN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

clean: ## Remove built binaries and downloaded assets
	@rm -f $(APP_NAME) $(APP_NAME)-* $(STAMP)
	@rm -rf $(JS_DIR) $(CSS_DIR) $(FONTS_DIR) $(FA_DIR)
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Assets
# =============================================================================
assets: $(STAMP) ## Download pinned frontend assets (never committed)
	@:

# Every pin lives in this file, so bumping one invalidates the stamp and re-downloads.
$(STAMP): $(MAKEFILE_LIST)
	@mkdir -p $(JS_DIR) $(CSS_DIR) $(FONTS_DIR) $(FA_DIR)/css $(FA_DIR)/webfonts
	@curl -sfL "https://cdn.jsdelivr.net/npm/@tailwindcss/browser@$(TAILWIND_VERSION)" -o "$(JS_DIR)/tailwind.js"
	@curl -sfL "https://cdn.jsdelivr.net/npm/lucide@$(LUCIDE_VERSION)/dist/umd/lucide.min.js" -o "$(JS_DIR)/lucide.min.js"
	@curl -sfL "https://cdn.jsdelivr.net/npm/marked@$(MARKED_VERSION)/lib/marked.umd.js" -o "$(JS_DIR)/marked.umd.js"
	@curl -sfL "https://cdn.jsdelivr.net/npm/dompurify@$(DOMPURIFY_VERSION)/dist/purify.min.js" -o "$(JS_DIR)/purify.min.js"
	@curl -sfL "https://cdn.jsdelivr.net/npm/@highlightjs/cdn-assets@$(HIGHLIGHTJS_VERSION)/highlight.min.js" -o "$(JS_DIR)/highlight.min.js"
	@curl -sfL "https://cdn.jsdelivr.net/npm/mermaid@$(MERMAID_VERSION)/dist/mermaid.min.js" -o "$(JS_DIR)/mermaid.min.js"
	@$(MAKE) --no-print-directory fontawesome
	@$(MAKE) --no-print-directory pdfjs
	@$(MAKE) --no-print-directory font FAMILY="Inter" SLUG=inter WEIGHTS="400;500;600;700"
	@$(MAKE) --no-print-directory font FAMILY="Google+Sans" SLUG=google-sans WEIGHTS="400;500;700"
	@$(MAKE) --no-print-directory font FAMILY="JetBrains+Mono" SLUG=jetbrains-mono WEIGHTS="400;700"
	@touch $(STAMP)
	@echo "$(GREEN)Assets downloaded$(NC)"

# URLs are repointed locally so no font is fetched at run time, and only the latin
# blocks are kept because `go:embed` compiles every other subset into the binary too.
font:
	@curl -sfL -H "User-Agent: $(UA)" \
	  "https://fonts.googleapis.com/css2?family=$(FAMILY):wght@$(WEIGHTS)&display=swap" \
	  -o "$(CSS_DIR)/$(SLUG).raw"
	@awk -v slug="$(SLUG)" -v css="$(CSS_DIR)/$(SLUG).css" ' \
	  /^\/\* /{keep = ($$0 ~ /^\/\* latin(-ext)? \*\/$$/)} \
	  !keep {next} \
	  match($$0, /https:\/\/fonts\.gstatic\.com\/[^)]*/) { \
	    url = substr($$0, RSTART, RLENGTH); \
	    if (!(url in name)) { name[url] = slug "-" (++n) ".woff2"; print url, name[url] } \
	    $$0 = substr($$0, 1, RSTART - 1) "/static/fonts/" name[url] substr($$0, RSTART + RLENGTH) \
	  } \
	  {print > css}' "$(CSS_DIR)/$(SLUG).raw" \
	  | xargs -P 8 -n 2 sh -c 'curl -sfL "$$1" -o "$(FONTS_DIR)/$$2"' _
	@rm -f "$(CSS_DIR)/$(SLUG).raw"

# Font Awesome's stylesheet points at ../webfonts, which does not resolve once the
# file is served from /static/fontawesome/css/.
fontawesome:
	@curl -sfL "https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@$(FONTAWESOME_VERSION)/css/all.min.css" -o "$(FA_DIR)/css/all.min.css"
	@for f in fa-brands-400 fa-regular-400 fa-solid-900; do \
	  curl -sfL "https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@$(FONTAWESOME_VERSION)/webfonts/$$f.woff2" -o "$(FA_DIR)/webfonts/$$f.woff2"; \
	done
	@sed -i.bak 's|../webfonts/|/static/fontawesome/webfonts/|g' "$(FA_DIR)/css/all.min.css"
	@rm -f "$(FA_DIR)/css/all.min.css.bak"

pdfjs:
	@mkdir -p $(PDFJS_DIR)/build $(PDFJS_DIR)/web/images $(PDFJS_DIR)/wasm
	@set -e; for f in build/pdf.min.mjs build/pdf.worker.min.mjs web/pdf_viewer.mjs web/pdf_viewer.css web/images/loading-icon.gif wasm/openjpeg.wasm wasm/jbig2.wasm wasm/qcms_bg.wasm; do \
	  curl -sfL "https://cdn.jsdelivr.net/npm/pdfjs-dist@$(PDFJS_VERSION)/$$f" -o "$(PDFJS_DIR)/$$f"; \
	done

verify-assets: ## Fail early if the embedded tree is missing an asset
	@test -s $(JS_DIR)/tailwind.js || (echo "tailwind.js missing, run 'make assets'" && exit 1)
	@test -s $(JS_DIR)/lucide.min.js || (echo "lucide.min.js missing, run 'make assets'" && exit 1)
	@test -s $(CSS_DIR)/inter.css || (echo "inter.css missing, run 'make assets'" && exit 1)
	@test -s $(CSS_DIR)/google-sans.css || (echo "google-sans.css missing, run 'make assets'" && exit 1)
	@test -s $(CSS_DIR)/jetbrains-mono.css || (echo "jetbrains-mono.css missing, run 'make assets'" && exit 1)

# =============================================================================
# Build
# =============================================================================
build: assets verify-assets ## Build binary for current platform
	@go build -ldflags="-s -w -X '$(MODULE)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME) .
	@echo "$(GREEN)Built: ./$(APP_NAME)$(NC)"

build-for: verify-assets ## Build binary for specified GOOS/GOARCH
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X '$(MODULE)/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME)-$(GOOS)-$(GOARCH) .
	@echo "$(GREEN)Built: ./$(APP_NAME)-$(GOOS)-$(GOARCH)$(NC)"

build-all: assets verify-assets ## Build all platform binaries
	@$(MAKE) build-for GOOS=linux GOARCH=amd64
	@$(MAKE) build-for GOOS=linux GOARCH=arm64
	@$(MAKE) build-for GOOS=darwin GOARCH=amd64
	@$(MAKE) build-for GOOS=darwin GOARCH=arm64


# =============================================================================
# Version
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
