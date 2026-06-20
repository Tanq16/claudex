# Makefile Template

Complete Makefile for Go projects with CI/CD automation.

## CLI + Web Template (With Frontend Assets and Docker)

For CLI + Web projects with embedded frontend (web servers, dashboards):

```makefile
.PHONY: help assets verify-assets clean build build-for build-all docker-build docker-push version

# =============================================================================
# Variables
# =============================================================================
APP_NAME := [APP_NAME]
DOCKER_USER := [GITHUB_USER]

# Build variables (set by CI or use defaults)
VERSION ?= dev-build
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Asset versions - update as needed
TAILWIND_VERSION := latest
LUCIDE_VERSION := 0.468.0
MARKEDJS_VERSION := 15.0.6
HIGHLIGHTJS_VERSION := 11.11.1
MERMAIDJS_VERSION := 11.4.1

# Directories
STATIC_DIR := internal/server/static
JS_DIR := $(STATIC_DIR)/js
CSS_DIR := $(STATIC_DIR)/css
FONTS_DIR := $(STATIC_DIR)/fonts

# Console colors
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
# Assets
# =============================================================================
assets: ## Download static assets
	@echo "$(CYAN)Downloading assets...$(NC)"
	@mkdir -p $(JS_DIR) $(CSS_DIR) $(FONTS_DIR)
	@curl -sL "https://cdn.tailwindcss.com" -o "$(JS_DIR)/tailwindcss.js"
	@curl -sL "https://unpkg.com/lucide@$(LUCIDE_VERSION)/dist/umd/lucide.min.js" -o "$(JS_DIR)/lucide.min.js"
	@curl -sL "https://cdn.jsdelivr.net/npm/marked@$(MARKEDJS_VERSION)/marked.min.js" -o "$(JS_DIR)/marked.min.js"
	@curl -sL "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/highlight.min.js" -o "$(JS_DIR)/highlight.min.js"
	@curl -sL "https://cdn.jsdelivr.net/npm/mermaid@$(MERMAIDJS_VERSION)/dist/mermaid.min.js" -o "$(JS_DIR)/mermaid.min.js"
	@curl -sL "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/$(HIGHLIGHTJS_VERSION)/styles/github-dark.min.css" -o "$(CSS_DIR)/github-dark.min.css"
	@curl -sL "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap" -H "User-Agent: Mozilla/5.0" -o "$(CSS_DIR)/inter.css"
	@grep -o "https://fonts.gstatic.com/[^)']*" "$(CSS_DIR)/inter.css" | sort -u | while read url; do \
		filename=$$(basename "$$url" | sed 's/?.*//'); \
		curl -sL "$$url" -o "$(FONTS_DIR)/$$filename"; \
	done
	@sed -i.bak -E 's|https://fonts.gstatic.com/s/inter/[^/]+/||g' "$(CSS_DIR)/inter.css" && rm -f "$(CSS_DIR)/inter.css.bak"
	@sed -i.bak 's|src: url(|src: url(/static/fonts/|g' "$(CSS_DIR)/inter.css" && rm -f "$(CSS_DIR)/inter.css.bak"
	@echo "$(GREEN)Assets downloaded$(NC)"

verify-assets: ## Verify required assets exist
	@test -f $(JS_DIR)/tailwindcss.js || (echo "$(YELLOW)tailwindcss.js missing. Run 'make assets'$(NC)" && exit 1)
	@echo "$(GREEN)Assets verified$(NC)"

clean: ## Remove built artifacts and downloaded assets
	@rm -f $(APP_NAME) $(APP_NAME)-*
	@rm -rf $(JS_DIR)/*.js $(CSS_DIR)/*.css $(FONTS_DIR)/*.woff2
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Build
# =============================================================================
build: assets verify-assets ## Build binary for current platform
	@go build -ldflags="-s -w -X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME) .
	@echo "$(GREEN)Built: ./$(APP_NAME)$(NC)"

build-for: verify-assets ## Build binary for specified GOOS/GOARCH
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME)-$(GOOS)-$(GOARCH) .
	@echo "$(GREEN)Built: ./$(APP_NAME)-$(GOOS)-$(GOARCH)$(NC)"

build-all: assets verify-assets ## Build all platform binaries
	@$(MAKE) build-for GOOS=linux GOARCH=amd64
	@$(MAKE) build-for GOOS=linux GOARCH=arm64
	@$(MAKE) build-for GOOS=darwin GOARCH=amd64
	@$(MAKE) build-for GOOS=darwin GOARCH=arm64

# =============================================================================
# Docker
# =============================================================================
docker-build: ## Build Docker image
	@docker build -t $(DOCKER_USER)/$(APP_NAME):$(VERSION) .
	@docker tag $(DOCKER_USER)/$(APP_NAME):$(VERSION) $(DOCKER_USER)/$(APP_NAME):latest
	@echo "$(GREEN)Docker image built$(NC)"

docker-push: docker-build ## Push Docker image to Docker Hub
	@docker push $(DOCKER_USER)/$(APP_NAME):$(VERSION)
	@docker push $(DOCKER_USER)/$(APP_NAME):latest
	@echo "$(GREEN)Docker image pushed$(NC)"

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
```

---

## CLI Only Template (No Frontend Assets, No Docker)

For CLI Only projects — terminal tools with multi-platform binaries. No Docker targets, no asset targets.

```makefile
.PHONY: help clean build build-for build-all version

# =============================================================================
# Variables
# =============================================================================
APP_NAME := [APP_NAME]

# Build variables (set by CI or use defaults)
VERSION ?= dev-build
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

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

clean: ## Remove built binaries
	@rm -f $(APP_NAME) $(APP_NAME)-*
	@echo "$(GREEN)Cleaned$(NC)"

# =============================================================================
# Build
# =============================================================================
build: ## Build binary for current platform
	@go build -ldflags="-s -w -X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME) .
	@echo "$(GREEN)Built: ./$(APP_NAME)$(NC)"

build-for: ## Build binary for specified GOOS/GOARCH
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=$(VERSION)'" -o $(APP_NAME)-$(GOOS)-$(GOARCH) .
	@echo "$(GREEN)Built: ./$(APP_NAME)-$(GOOS)-$(GOARCH)$(NC)"

build-all: ## Build all platform binaries
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
```

---

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your application name (e.g., `kairo`, `backsync`) |

### Version Injection

The `-ldflags` path must match your cmd package structure:

```makefile
# If your version variable is in cmd/root.go as AppVersion:
-X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=$(VERSION)'

# If your version variable is in main.go as Version:
-X 'main.Version=$(VERSION)'
```

### Common Asset Libraries

Add/remove from assets target as needed:

| Library | CDN URL |
|---------|---------|
| Tailwind CSS | `https://cdn.tailwindcss.com` |
| Lucide Icons | `https://unpkg.com/lucide@VERSION/dist/umd/lucide.min.js` |
| Marked.js | `https://cdn.jsdelivr.net/npm/marked@VERSION/marked.min.js` |
| Highlight.js | `https://cdnjs.cloudflare.com/ajax/libs/highlight.js/VERSION/highlight.min.js` |
| Mermaid.js | `https://cdn.jsdelivr.net/npm/mermaid@VERSION/dist/mermaid.min.js` |
| Chart.js | `https://cdn.jsdelivr.net/npm/chart.js@VERSION/dist/chart.umd.js` |
| Font Awesome | `https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@VERSION/css/all.min.css` |
| Inter Font | `https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap` |
| JetBrains Mono | `https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&display=swap` |

---

## Chrome Extension Template

For Chrome extensions (builds distributable zip):

```makefile
.PHONY: help clean build version

# =============================================================================
# Variables
# =============================================================================
EXT_NAME := [EXTENSION_NAME]
VERSION ?= $(shell cat manifest.json | grep '"version"' | head -1 | sed 's/.*"version": "\(.*\)".*/\1/')

# Directories and files to include in zip
DIST_DIR := dist
SRC_FILES := manifest.json popup/ content/ background/ icons/ lib/

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

# =============================================================================
# Build
# =============================================================================
clean: ## Remove build artifacts
	@rm -rf $(DIST_DIR)
	@echo "$(GREEN)Cleaned$(NC)"

build: clean ## Build extension zip for distribution
	@mkdir -p $(DIST_DIR)
	@# Create zip with only the files that exist
	@zip -r $(DIST_DIR)/$(EXT_NAME)-$(VERSION).zip $(shell for f in $(SRC_FILES); do [ -e "$$f" ] && echo "$$f"; done) -x "*.DS_Store" -x "*/.git/*"
	@echo "$(GREEN)Built: $(DIST_DIR)/$(EXT_NAME)-$(VERSION).zip$(NC)"

# =============================================================================
# Version
# =============================================================================
version: ## Show current version from manifest.json
	@echo "v$(VERSION)"

# =============================================================================
# Development
# =============================================================================
dev: ## Instructions for loading unpacked extension
	@echo "$(CYAN)To load extension in Chrome:$(NC)"
	@echo "  1. Open chrome://extensions/"
	@echo "  2. Enable 'Developer mode' (top right)"
	@echo "  3. Click 'Load unpacked'"
	@echo "  4. Select this directory: $(PWD)"
```

### Chrome Extension Customization

| Placeholder | Replace With |
|-------------|--------------|
| `[EXTENSION_NAME]` | Extension name, lowercase with dashes (e.g., `cookie-extractor`) |

### Extension Build Notes

- Version is read from `manifest.json` automatically
- Only existing directories are included in the zip
- `.DS_Store` and `.git` files are excluded
- The zip can be extracted and loaded as unpacked in Chrome
